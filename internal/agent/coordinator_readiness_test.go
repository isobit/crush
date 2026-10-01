package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/agent/prompt"
	"github.com/charmbracelet/crush/internal/agent/tools/mcp"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/stretchr/testify/require"
)

// TestBuildAgentReadinessSurvivesCallerCancellation is a regression test for
// sub-agents running before their asynchronous prompt and tool setup finishes.
//
// buildAgent builds the system prompt and tool list asynchronously. The
// coordinator's readiness wait covered only its current agent, not sub-agents
// created later by the agent tool, so a sub-agent could send a request before
// its own setup completed. SessionAgent.Run now waits for that agent's
// readiness. The setup context is detached from the caller so a short-lived
// HTTP request context cannot cancel the MCP initialization wait.
func TestBuildAgentReadinessSurvivesCallerCancellation(t *testing.T) {
	env := testEnv(t)

	// Minimal hermetic config: one openai-typed provider with selected large
	// and small models so buildAgentModels and the system-prompt build both
	// succeed. No MCP servers are configured, so initialization would complete
	// instantly if we let it — we deliberately do not, so WaitForInit stays
	// blocked for the duration of the assertion.
	crushJSON := `{
  "options": {"disable_default_providers": true, "disable_provider_auto_update": true},
  "providers": {"mock": {"id": "mock", "name": "Mock", "type": "openai",
    "base_url": "http://127.0.0.1:9/v1", "api_key": "test-key",
    "models": [{"id": "mock-model", "name": "Mock", "context_window": 8192, "default_max_tokens": 128}]}},
  "models": {"large": {"provider": "mock", "model": "mock-model"},
             "small": {"provider": "mock", "model": "mock-model"}}
}`
	require.NoError(t, os.WriteFile(filepath.Join(env.workingDir, "crush.json"), []byte(crushJSON), 0o644))

	cfg, err := config.Init(env.workingDir, "", false, testConfigFile(env.workingDir))
	require.NoError(t, err)
	cfg.SetupAgents()

	coord := &coordinator{
		cfg:         cfg,
		sessions:    env.sessions,
		messages:    env.messages,
		permissions: env.permissions,
		history:     env.history,
		filetracker: *env.filetracker,
	}

	// Arm the MCP init gate so buildAgent's tool-readiness task blocks in
	// WaitForInit. It remains parked during the assertion, preventing Run from
	// reaching message/session access before setup completes.
	mcp.ArmInit()

	p, err := coderPrompt(prompt.WithWorkingDir(env.workingDir))
	require.NoError(t, err)
	agentCfg := cfg.Config().Agents[config.AgentCoder]

	ctx, cancel := context.WithCancel(context.Background())
	agent, err := coord.buildAgent(ctx, p, agentCfg, false)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		_, err := agent.Run(ctx, SessionAgentCall{SessionID: "session", Prompt: "prompt"})
		done <- err
	}()

	// The caller goes away, mirroring an HTTP handler returning and canceling
	// its request context while MCP init is still in flight. Run must continue
	// waiting for readiness instead of racing ahead with an empty prompt/tools.
	cancel()

	select {
	case err := <-done:
		t.Fatalf("Run returned before agent readiness completed: %v", err)
	case <-time.After(250 * time.Millisecond):
	}
}
