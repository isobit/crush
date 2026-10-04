package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/agent/prompt"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/markdiff"
	"github.com/stretchr/testify/require"
)

func TestCoderPromptMarkdiff(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workingDir := filepath.Join(root, "workspace")
	dataDir := filepath.Join(root, "data")
	require.NoError(t, os.MkdirAll(workingDir, 0o755))
	require.NoError(t, os.MkdirAll(dataDir, 0o755))

	configPath := filepath.Join(root, "crush.json")
	configJSON := `{
  "options": {
    "disable_default_providers": true,
    "disable_provider_auto_update": true,
    "global_context_paths": ["` + filepath.ToSlash(filepath.Join(root, "missing-global-context.md")) + `"]
  },
  "providers": {
    "mock": {
      "id": "mock",
      "name": "Mock",
      "type": "openai",
      "base_url": "http://127.0.0.1:9/v1",
      "api_key": "test-key",
      "models": [{"id": "mock-model", "name": "Mock Model", "context_window": 8192, "default_max_tokens": 128}]
    }
  },
  "models": {
    "large": {"provider": "mock", "model": "mock-model"},
    "small": {"provider": "mock", "model": "mock-model"}
  }
}`
	require.NoError(t, os.WriteFile(configPath, []byte(configJSON), 0o600))

	cfg, err := config.Init(workingDir, dataDir, false, config.WithConfigFiles([]string{configPath}))
	require.NoError(t, err)
	cfg.Config().Options.SkillsPaths = nil
	cfg.Config().Agents["reviewer"] = config.Agent{
		Description: "Reviews code changes for correctness and regression risks.",
	}
	cfg.SetupAgents()

	fixedTime := func() time.Time {
		return time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	systemPromptTemplate, err := coderPrompt(
		prompt.WithTimeFunc(fixedTime),
		prompt.WithPlatform("linux"),
		prompt.WithWorkingDir("/workspace"),
		prompt.WithContextPaths([]string{}),
	)
	require.NoError(t, err)
	systemPrompt, err := systemPromptTemplate.Build(context.Background(), "mock", "mock-model", cfg)
	require.NoError(t, err)

	agentDescription := availableAgentDescription(cfg.Config().Agents)
	require.Contains(t, agentDescription, "- reviewer: Reviews code changes for correctness and regression risks.")

	doc := markdiff.New(t, "Coder prompt and configured agent profiles",
		"Captures the system prompt and delegation profiles exposed to the coder agent, including a user-configured profile.")

	promptSection := doc.Section("Coder system prompt",
		"This is the rendered system prompt sent to the coder agent with a fixed environment and no user-specific context.")
	promptSection.Code("markdown", systemPrompt)

	profilesSection := doc.Section("Available agent profiles",
		"The agent tool description lists built-in profiles and configured custom profiles in stable order.")
	profilesSection.Code("text", agentDescription)
}
