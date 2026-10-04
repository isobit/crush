package agent

import (
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/stretchr/testify/require"
)

func TestAvailableAgentDescription(t *testing.T) {
	description := availableAgentDescription(map[string]config.Agent{
		"zeta":  {Description: "Runs last."},
		"alpha": {Description: "Runs first."},
		"empty": {},
		"custom": {
			Description:  "Audits code.",
			AllowedTools: []string{"agent", "view", "question", "grep"},
			AllowedMCP:   map[string][]string{"docs": {"read"}},
		},
	})

	require.Contains(t, description, agentToolDescription)
	require.Less(t, strings.Index(description, "- alpha: Runs first."), strings.Index(description, "- zeta: Runs last."))
	require.Contains(t, description, "- empty: No description provided.\n  Tools: none")
	require.Contains(t, description, "- custom: Audits code.\n  Tools: grep, view\n  MCP access: docs (tools: read)")
	require.Contains(t, description, "- alpha: Runs first.\n  Tools: none\n  MCP access: unrestricted (if configured)")
	require.NotContains(t, description, "Tools: agent")
	require.NotContains(t, description, "Tools: question")
}
