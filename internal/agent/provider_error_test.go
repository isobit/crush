package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstreamProviderDetail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "string raw payload with provider name",
			body: `HTTP/2.0 400 Bad Request
Content-Type: application/json

{"error":{"message":"Provider returned error","code":400,"metadata":{"provider_name":"Anthropic","raw":"{\"type\":\"error\",\"error\":{\"message\":\"prompt is too long\"}}"}}}`,
			want: `Anthropic: {"type":"error","error":{"message":"prompt is too long"}}`,
		},
		{
			name: "object raw payload",
			body: `{"error":{"message":"Provider returned error","metadata":{"raw":{"type":"invalid_request_error"}}}}`,
			want: `{"type":"invalid_request_error"}`,
		},
		{
			name: "falls back to error message",
			body: `{"error":{"message":"Provider returned error"}}`,
			want: "Provider returned error",
		},
		{
			name: "no json",
			body: "plain text",
			want: "",
		},
		{
			name: "empty body",
			body: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, upstreamProviderDetail([]byte(tt.body)))
		})
	}
}
