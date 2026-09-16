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
	})

	require.Contains(t, description, agentToolDescription)
	require.Less(t, strings.Index(description, "- alpha: Runs first."), strings.Index(description, "- zeta: Runs last."))
	require.Contains(t, description, "- empty: No description provided.")
}
