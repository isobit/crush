package tools

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kagi "github.com/kagisearch/kagi-openapi-golang"
	"github.com/stretchr/testify/require"
)

func TestSearchKagiIncludesHTTPErrorResponse(t *testing.T) {
	t.Parallel()

	responseBody := `{"error":"quota exceeded"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Kagi-Trace", "trace-123")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(server.Close)

	_, err := searchKagi(context.Background(), newTestKagiClient(server), "test query", 10)

	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP status 429 Too Many Requests; trace: trace-123")
	require.Contains(t, err.Error(), "response body: "+responseBody)
}

func TestSearchKagiIncludesDecodeErrorResponse(t *testing.T) {
	t.Parallel()

	responseBody := `{"meta":{},"data":{"search":[{"url":"https://example.com"}]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(server.Close)

	_, err := searchKagi(context.Background(), newTestKagiClient(server), "test query", 10)

	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP status 200 OK")
	require.Contains(t, err.Error(), "no value given for required property title")
	require.Contains(t, err.Error(), "response body: "+responseBody)
}

func TestSearchKagiRejectsNonOKResponse(t *testing.T) {
	t.Parallel()

	responseBody := `{"meta":{},"data":{}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(server.Close)

	_, err := searchKagi(context.Background(), newTestKagiClient(server), "test query", 10)

	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP status 201 Created")
	require.Contains(t, err.Error(), "unexpected HTTP status")
	require.Contains(t, err.Error(), "response body: "+responseBody)
}

func TestFormatKagiSearchErrorWrapsUnderlyingError(t *testing.T) {
	t.Parallel()

	underlyingErr := errors.New("request failed")
	err := formatKagiSearchError(nil, underlyingErr)

	require.ErrorIs(t, err, underlyingErr)
}

func TestFormatKagiSearchErrorTruncatesResponseBody(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("x", maxKagiErrorResponseBody+1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	response, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	err = formatKagiSearchError(response, errors.New("server error"))

	require.Error(t, err)
	require.Contains(t, err.Error(), strings.Repeat("x", maxKagiErrorResponseBody)+"...")
	require.NotContains(t, err.Error(), strings.Repeat("x", maxKagiErrorResponseBody+1))
}

func newTestKagiClient(server *httptest.Server) *kagi.APIClient {
	cfg := kagi.NewConfiguration()
	cfg.Servers[0].URL = server.URL
	cfg.HTTPClient = server.Client()
	return kagi.NewAPIClient(cfg)
}
