package chat

import (
	"encoding/json"
	"strings"

	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/ui/styles"
)

// -----------------------------------------------------------------------------
// Numbat Tool
// -----------------------------------------------------------------------------

// NumbatToolMessageItem is a message item that represents a numbat tool call.
type NumbatToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*NumbatToolMessageItem)(nil)

// NewNumbatToolMessageItem creates a new [NumbatToolMessageItem].
func NewNumbatToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &NumbatToolRenderContext{}, canceled)
}

// NumbatToolRenderContext renders numbat tool messages, showing the code
// prominently for human review.
type NumbatToolRenderContext struct{}

// RenderTool implements the [ToolRenderer] interface.
func (n *NumbatToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	cappedWidth := cappedMessageWidth(width)
	if opts.IsPending() {
		return pendingTool(sty, "Numbat", opts.Anim, opts.Compact)
	}

	var params tools.NumbatParams
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "Invalid parameters"}, cappedWidth)
	}

	// Show a one-line summary of the code in the header.
	codeSummary := strings.ReplaceAll(params.Code, "\n", " ")
	codeSummary = strings.ReplaceAll(codeSummary, "\t", " ")
	header := toolHeader(sty, opts.Status, "Numbat", cappedWidth, opts.Compact, codeSummary)
	if opts.Compact {
		return header
	}

	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	if !opts.HasResult() {
		return header
	}

	bodyWidth := cappedWidth - toolBodyLeftPaddingTotal

	// Build body: show full code block when multi-line, then the result.
	var bodyParts []string
	if strings.Contains(params.Code, "\n") {
		codeBlock := sty.Tool.Body.Render(
			toolOutputPlainContent(sty, params.Code, bodyWidth, opts.ExpandedContent),
		)
		bodyParts = append(bodyParts, codeBlock)
	}

	if opts.Result.Content != "" {
		// Add a separator when both code and result are shown.
		if len(bodyParts) > 0 {
			sep := sty.Tool.Body.Render(
				sty.Tool.ParamMain.Render(strings.Repeat("─", min(bodyWidth, 40))),
			)
			bodyParts = append(bodyParts, sep)
		}
		resultBlock := sty.Tool.Body.Render(
			toolOutputPlainContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent),
		)
		bodyParts = append(bodyParts, resultBlock)
	}

	if len(bodyParts) == 0 {
		return header
	}

	body := strings.Join(bodyParts, "\n")
	return joinToolParts(header, body)
}
