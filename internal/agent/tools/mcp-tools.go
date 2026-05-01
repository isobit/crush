package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/tools/mcp"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/permission"
)

// whitelistDockerTools contains Docker MCP tools that don't require permission.
var whitelistDockerTools = []string{
	"mcp_docker_mcp-find",
	"mcp_docker_mcp-add",
	"mcp_docker_mcp-remove",
	"mcp_docker_mcp-config-set",
	"mcp_docker_code-mode",
}

// GetMCPTools gets all the currently available MCP tools.
func GetMCPTools(permissions permission.Service, cfg *config.ConfigStore, wd string) []*Tool {
	var result []*Tool
	for mcpName, tools := range mcp.Tools() {
		for _, tool := range tools {
			result = append(result, &Tool{
				mcpName:     mcpName,
				tool:        tool,
				permissions: permissions,
				workingDir:  wd,
				cfg:         cfg,
			})
		}
	}
	return result
}

// Tool is a tool from a MCP.
type Tool struct {
	mcpName         string
	tool            *mcp.Tool
	cfg             *config.ConfigStore
	permissions     permission.Service
	workingDir      string
	providerOptions fantasy.ProviderOptions
}

func (m *Tool) SetProviderOptions(opts fantasy.ProviderOptions) {
	m.providerOptions = opts
}

func (m *Tool) ProviderOptions() fantasy.ProviderOptions {
	return m.providerOptions
}

func (m *Tool) Name() string {
	return fmt.Sprintf("mcp_%s_%s", m.mcpName, m.tool.Name)
}

func (m *Tool) MCP() string {
	return m.mcpName
}

func (m *Tool) MCPToolName() string {
	return m.tool.Name
}

func (m *Tool) Info() fantasy.ToolInfo {
	parameters := make(map[string]any)
	required := make([]string, 0)

	if input, ok := m.tool.InputSchema.(map[string]any); ok {
		if props, ok := input["properties"].(map[string]any); ok {
			parameters = props
		}
		if req, ok := input["required"].([]any); ok {
			// Convert []any -> []string when elements are strings
			for _, v := range req {
				if s, ok := v.(string); ok {
					required = append(required, s)
				}
			}
		} else if reqStr, ok := input["required"].([]string); ok {
			// Handle case where it's already []string
			required = reqStr
		}
	}

	parameters[mcpOutputFileParam] = map[string]any{
		"type":        "boolean",
		"description": "Pseudo-parameter handled by the agent, NOT forwarded to the MCP server. If true, write the tool output to a temporary file and return the file path instead of inline content. Useful when you expect large output and want to process it with view or grep.",
	}

	return fantasy.ToolInfo{
		Name:        m.Name(),
		Description: m.tool.Description,
		Parameters:  parameters,
		Required:    required,
	}
}

// mcpOutputFileParam is the injected parameter name that agents can set to
// request output be written to a temporary file instead of returned inline.
const mcpOutputFileParam = "__output_file"

func (m *Tool) Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
	sessionID := GetSessionFromContext(ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, fmt.Errorf("session ID is required for creating a new file")
	}

	// Extract the __output_file flag before forwarding to the MCP server.
	outputFile, mcpInput := extractOutputFileParam(params.Input)

	// Skip permission for whitelisted Docker MCP tools.
	if !slices.Contains(whitelistDockerTools, params.Name) {
		permissionDescription := fmt.Sprintf("execute %s with the following parameters:", m.Info().Name)
		p, err := m.permissions.Request(ctx,
			permission.CreatePermissionRequest{
				SessionID:   sessionID,
				ToolCallID:  params.ID,
				Path:        m.workingDir,
				ToolName:    m.Info().Name,
				Action:      "execute",
				Description: permissionDescription,
				Params:      mcpInput,
			},
		)
		if err != nil {
			return fantasy.ToolResponse{}, err
		}
		if !p {
			return NewPermissionDeniedResponse(), nil
		}
	}

	result, err := mcp.RunTool(ctx, m.cfg, m.mcpName, m.tool.Name, mcpInput)
	if err != nil {
		return fantasy.NewTextErrorResponse(err.Error()), nil
	}

	switch result.Type {
	case "image", "media":
		if !GetSupportsImagesFromContext(ctx) {
			modelName := GetModelNameFromContext(ctx)
			return fantasy.NewTextErrorResponse(fmt.Sprintf("This model (%s) does not support image data.", modelName)), nil
		}

		var response fantasy.ToolResponse
		if result.Type == "image" {
			response = fantasy.NewImageResponse(result.Data, result.MediaType)
		} else {
			response = fantasy.NewMediaResponse(result.Data, result.MediaType)
		}
		response.Content = result.Content
		return response, nil
	default:
		if outputFile || len(result.Content) > LargeContentThreshold {
			return m.spillToFile(result.Content)
		}
		return fantasy.NewTextResponse(result.Content), nil
	}
}

// spillToFile writes content to a temporary file in the data directory and
// returns a response pointing the agent at the file path.
func (m *Tool) spillToFile(content string) (fantasy.ToolResponse, error) {
	dataDir := m.cfg.Config().Options.DataDirectory
	tempFile, createErr := os.CreateTemp(dataDir, "mcp-*")
	if createErr != nil {
		slog.Warn("Failed to create temp file for MCP output, returning inline", "error", createErr)
		return fantasy.NewTextResponse(content), nil
	}
	tempFilePath := tempFile.Name()

	if _, writeErr := tempFile.WriteString(content); writeErr != nil {
		slog.Warn("Failed to write MCP output to temp file, returning inline", "path", tempFilePath, "error", writeErr)
		_ = tempFile.Close()
		return fantasy.NewTextResponse(content), nil
	}
	if closeErr := tempFile.Close(); closeErr != nil {
		slog.Warn("Failed to close MCP temp file, returning inline", "path", tempFilePath, "error", closeErr)
		return fantasy.NewTextResponse(content), nil
	}

	var msg strings.Builder
	fmt.Fprintf(&msg, "MCP tool %s returned output (%d bytes)\n\n", m.tool.Name, len(content))
	fmt.Fprintf(&msg, "Content saved to: %s\n\n", tempFilePath)
	msg.WriteString("Use the view and grep tools to analyze this file.")
	return fantasy.NewTextResponse(msg.String()), nil
}

// extractOutputFileParam removes the __output_file key from the JSON input
// and returns whether it was set to true, along with the cleaned input.
func extractOutputFileParam(input string) (bool, string) {
	var args map[string]any
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return false, input
	}

	val, ok := args[mcpOutputFileParam]
	if !ok {
		return false, input
	}

	outputFile, _ := val.(bool)
	delete(args, mcpOutputFileParam)

	cleaned, err := json.Marshal(args)
	if err != nil {
		return outputFile, input
	}
	return outputFile, string(cleaned)
}
