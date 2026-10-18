package agent

import (
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/stretchr/testify/require"
)

func TestFilterDisabledTools(t *testing.T) {
	t.Parallel()

	newTools := func() []fantasy.AgentTool {
		return []fantasy.AgentTool{
			tools.NewWebFetchTool(t.TempDir(), nil),
			tools.NewSourcegraphTool(nil),
		}
	}

	names := func(ts []fantasy.AgentTool) []string {
		out := make([]string, 0, len(ts))
		for _, tool := range ts {
			out = append(out, tool.Info().Name)
		}
		return out
	}

	t.Run("no disabled tools keeps all", func(t *testing.T) {
		t.Parallel()
		got := filterDisabledTools(newTools(), nil)
		require.Equal(t, []string{"web_fetch", "sourcegraph"}, names(got))
	})

	t.Run("exact name removes tool", func(t *testing.T) {
		t.Parallel()
		got := filterDisabledTools(newTools(), []string{"sourcegraph"})
		require.Equal(t, []string{"web_fetch"}, names(got))
	})

	t.Run("glob pattern removes matching tools", func(t *testing.T) {
		t.Parallel()
		got := filterDisabledTools(newTools(), []string{"web_*"})
		require.Equal(t, []string{"sourcegraph"}, names(got))
	})
}
