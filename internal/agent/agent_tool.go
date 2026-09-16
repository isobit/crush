package agent

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"sort"
	"strings"

	"charm.land/fantasy"

	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/config"
)

//go:embed templates/agent_tool.md
var agentToolDescription string

type AgentParams struct {
	Agent  string `json:"agent,omitempty" description:"The configured agent profile to run"`
	Prompt string `json:"prompt" description:"The task for the agent to perform"`
}

const (
	AgentToolName = "agent"
)

func availableAgentDescription(agents map[string]config.Agent) string {
	ids := make([]string, 0, len(agents))
	for id := range agents {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	profiles := make([]string, 0, len(ids))
	for _, id := range ids {
		agent := agents[id]
		description := agent.Description
		if description == "" {
			description = "No description provided."
		}
		profiles = append(profiles, fmt.Sprintf("- %s: %s", id, description))
	}

	return strings.Join(append([]string{agentToolDescription, "", "Available agent profiles:"}, profiles...), "\n")
}

func (c *coordinator) agentTool() (fantasy.AgentTool, error) {
	description := availableAgentDescription(c.cfg.Config().Agents)
	return fantasy.NewParallelAgentTool(
		AgentToolName,
		description,
		func(ctx context.Context, params AgentParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Prompt == "" {
				return fantasy.NewTextErrorResponse("prompt is required"), nil
			}

			agentID := params.Agent
			if agentID == "" {
				agentID = config.AgentTask
			}
			agentCfg, ok := c.cfg.Config().Agents[agentID]
			if !ok || agentCfg.Disabled {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("agent profile %q is not configured", agentID)), nil
			}

			prompt, err := agentPrompt(agentCfg, c.cfg.WorkingDir())
			if err != nil {
				return fantasy.ToolResponse{}, err
			}
			agent, err := c.buildAgent(ctx, prompt, agentCfg, true)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}

			sessionID := tools.GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, errors.New("session id missing from context")
			}

			agentMessageID := tools.GetMessageFromContext(ctx)
			if agentMessageID == "" {
				return fantasy.ToolResponse{}, errors.New("agent message id missing from context")
			}

			return c.runSubAgent(ctx, subAgentParams{
				Agent:          agent,
				SessionID:      sessionID,
				AgentMessageID: agentMessageID,
				ToolCallID:     call.ID,
				Prompt:         params.Prompt,
				SessionTitle:   agentCfg.Name + " Session",
			})
		},
	), nil
}
