package chat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStreamingMarkdownRendersMermaidDiagram(t *testing.T) {
	t.Parallel()

	const width = 79
	renderer := newTestRenderer(t, width)
	var markdown streamingMarkdown

	out := markdown.Render("```mermaid\ngraph TD\nA --> B\n```", width, renderer)

	require.Contains(t, out, "A")
	require.Contains(t, out, "B")
	require.NotContains(t, out, "graph TD")
	require.NotContains(t, out, "```mermaid")
	require.Empty(t, markdown.stablePrefix)
}
