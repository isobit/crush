package agent

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"strings"
)

// upstreamProviderDetail extracts the underlying provider's error text from a
// gateway error body. OpenRouter wraps upstream failures in a generic
// "Provider returned error" message and puts the real reason under
// error.metadata.raw. It returns "" when the body has no such detail.
func upstreamProviderDetail(body []byte) string {
	start := bytes.IndexByte(body, '{')
	if start < 0 {
		return ""
	}

	var envelope struct {
		Error struct {
			Message  string `json:"message"`
			Metadata struct {
				ProviderName string          `json:"provider_name"`
				Raw          json.RawMessage `json:"raw"`
			} `json:"metadata"`
		} `json:"error"`
	}
	// Decode only the first JSON value so trailing dump data is ignored.
	if err := json.NewDecoder(bytes.NewReader(body[start:])).Decode(&envelope); err != nil {
		return ""
	}

	raw := strings.TrimSpace(string(envelope.Error.Metadata.Raw))
	// The raw field is often a JSON string holding the upstream payload.
	var rawString string
	if json.Unmarshal(envelope.Error.Metadata.Raw, &rawString) == nil {
		raw = rawString
	}
	detail := cmp.Or(raw, envelope.Error.Message)
	if detail == "" || detail == "null" {
		return ""
	}
	if name := envelope.Error.Metadata.ProviderName; name != "" {
		return fmt.Sprintf("%s: %s", name, detail)
	}
	return detail
}

// truncateForLog caps s at max bytes so large response bodies stay readable
// in the logs.
func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
