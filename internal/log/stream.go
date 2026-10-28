package log

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// maxStreamLogBytes caps how much of an offending SSE chunk gets logged.
const maxStreamLogBytes = 2000

// knownFinishReasons are the normalized finish reasons that map to a
// meaningful turn end. Anything else is surfaced in the logs.
var knownFinishReasons = map[string]bool{
	"stop":           true,
	"length":         true,
	"tool_calls":     true,
	"function_call":  true,
	"content_filter": true,
}

// NewStreamWatchHTTPClient returns an HTTP client that logs upstream stream
// errors and unrecognized finish reasons in server-sent event responses.
// When debug is true, full request and response logging is layered on top.
func NewStreamWatchHTTPClient(debug bool) *http.Client {
	var rt http.RoundTripper = http.DefaultTransport
	if debug {
		rt = &HTTPRoundTripLogger{Transport: rt}
	}
	return &http.Client{
		Transport: &streamWatchRoundTripper{Transport: rt},
	}
}

type streamWatchRoundTripper struct {
	Transport http.RoundTripper
}

// RoundTrip implements http.RoundTripper, wrapping event-stream bodies so
// their SSE lines can be inspected as they are read.
func (s *streamWatchRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := s.Transport.RoundTrip(req)
	if err != nil || resp == nil {
		return resp, err
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return resp, nil
	}
	resp.Body = &sseWatcher{body: resp.Body}
	return resp, nil
}

// sseWatcher passes reads through unchanged while inspecting each complete
// SSE line. It never buffers the whole body, so streaming is unaffected.
type sseWatcher struct {
	body    io.ReadCloser
	pending []byte
}

func (w *sseWatcher) Read(p []byte) (int, error) {
	n, err := w.body.Read(p)
	w.pending = append(w.pending, p[:n]...)
	for {
		i := bytes.IndexByte(w.pending, '\n')
		if i < 0 {
			break
		}
		inspectSSELine(w.pending[:i])
		w.pending = w.pending[i+1:]
	}
	if err != nil && len(w.pending) > 0 {
		inspectSSELine(w.pending)
		w.pending = nil
	}
	return n, err
}

func (w *sseWatcher) Close() error {
	return w.body.Close()
}

// inspectSSELine logs data chunks that carry a top-level error object or a
// finish reason we do not recognize. Providers such as OpenRouter report
// mid-stream upstream failures this way while keeping the HTTP status 200.
func inspectSSELine(line []byte) {
	data, ok := bytes.CutPrefix(bytes.TrimSpace(line), []byte("data:"))
	if !ok {
		return
	}
	data = bytes.TrimSpace(data)
	if !bytes.Contains(data, []byte("finish_reason")) && !bytes.Contains(data, []byte(`"error"`)) {
		return
	}

	var chunk struct {
		Error   json.RawMessage `json:"error"`
		Choices []struct {
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &chunk); err != nil {
		return
	}

	var reason string
	for _, c := range chunk.Choices {
		if c.FinishReason != nil && *c.FinishReason != "" {
			reason = *c.FinishReason
		}
	}
	hasError := len(chunk.Error) > 0 && string(chunk.Error) != "null"
	if !hasError && (reason == "" || knownFinishReasons[reason]) {
		return
	}

	slog.Warn(
		"Upstream stream reported an error or unrecognized finish reason",
		"finish_reason", reason,
		"error", truncateForLog(string(chunk.Error)),
		"chunk", truncateForLog(string(data)),
	)
}

func truncateForLog(s string) string {
	if len(s) <= maxStreamLogBytes {
		return s
	}
	return s[:maxStreamLogBytes] + "...(truncated)"
}
