package common

import (
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestRenderMermaidDiagrams(t *testing.T) {
	t.Parallel()

	markdown := "Before\n\n```mermaid\ngraph TD\n    A[Start] --> B[Finish]\n```\n\nAfter"
	rendered, changed := RenderMermaidDiagrams(markdown, 80)

	require.True(t, changed)
	require.Contains(t, rendered, "Before")
	require.Contains(t, rendered, "After")
	require.Contains(t, rendered, "Start")
	require.Contains(t, rendered, "Finish")
	require.NotContains(t, rendered, "graph TD")
}

func TestRenderMermaidDiagramsLeavesNonMermaidFencesUnchanged(t *testing.T) {
	t.Parallel()

	for _, markdown := range []string{
		"```go\nfmt.Println(1)\n```",
		"```\nplain code\n```",
		"```mermaid\nnot a diagram\n```",
		"```mermaid\ngraph TD\nA --> B",
	} {
		rendered, changed := RenderMermaidDiagrams(markdown, 80)
		require.False(t, changed)
		require.Equal(t, markdown, rendered)
	}
}

func TestRenderMarkdownRendersMermaidDiagram(t *testing.T) {
	t.Parallel()

	sty := styles.IsobitStyles()
	renderer := MarkdownRenderer(&sty, 83)
	rendered, err := RenderMarkdown(renderer, "```mermaid\ngraph TD\nA --> B\n```", 83)

	require.NoError(t, err)
	plain := ansi.Strip(rendered)
	require.Contains(t, plain, "A")
	require.Contains(t, plain, "B")
	require.NotContains(t, plain, "graph TD")
	require.NotContains(t, plain, "```mermaid")
}

func TestRenderMermaidDiagramsSupportsTildeFencesAndCRLF(t *testing.T) {
	t.Parallel()

	markdown := strings.Join([]string{"~~~MERMAID", "graph LR", "A --> B", "~~~"}, "\r\n")
	rendered, changed := RenderMermaidDiagrams(markdown, 80)

	require.True(t, changed)
	require.Contains(t, rendered, "A")
	require.Contains(t, rendered, "B")
	require.NotContains(t, rendered, "graph LR")
}
