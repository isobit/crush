package globmatch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAny(t *testing.T) {
	require.True(t, Any([]string{"lsp_*", "view"}, "lsp_definition"))
	require.True(t, Any([]string{"*"}, "anything"))
	require.False(t, Any([]string{"lsp_*"}, "grep"))
}

func TestValidate(t *testing.T) {
	require.NoError(t, Validate("tools", []string{"lsp_*", "view"}))
	require.Error(t, Validate("tools", []string{"["}))
}
