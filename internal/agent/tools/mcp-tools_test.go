package tools

import (
	"sync"
	"testing"

	"github.com/charmbracelet/crush/internal/agent/tools/mcp"
	"github.com/stretchr/testify/require"
)

func TestMCPToolInfoDoesNotMutateInputSchema(t *testing.T) {
	t.Parallel()

	properties := map[string]any{
		"query": map[string]any{"type": "string"},
	}
	tool := &Tool{
		mcpName: "test",
		tool: &mcp.Tool{
			Name: "query_metrics",
			InputSchema: map[string]any{
				"properties": properties,
				"required":   []any{"query"},
			},
		},
	}

	info := tool.Info()

	require.Contains(t, info.Parameters, mcpOutputFileParam)
	require.NotContains(t, properties, mcpOutputFileParam)
	require.Equal(t, []string{"query"}, info.Required)
}

func TestMCPToolInfoSupportsConcurrentCalls(t *testing.T) {
	t.Parallel()

	tool := &Tool{
		mcpName: "test",
		tool: &mcp.Tool{
			Name: "query_metrics",
			InputSchema: map[string]any{
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
				},
			},
		},
	}

	const workers = 16
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			info := tool.Info()
			require.Contains(t, info.Parameters, mcpOutputFileParam)
		})
	}
	wg.Wait()
}
