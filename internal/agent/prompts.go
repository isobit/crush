package agent

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/charmbracelet/crush/internal/agent/prompt"
	"github.com/charmbracelet/crush/internal/config"

	"github.com/charmbracelet/crush/internal/filepathext"
	"github.com/charmbracelet/crush/internal/home"
)

//go:embed templates/coder.md.tpl
var coderPromptTmpl []byte

//go:embed templates/task.md.tpl
var taskPromptTmpl []byte

//go:embed templates/initialize.md.tpl
var initializePromptTmpl []byte

func coderPrompt(opts ...prompt.Option) (*prompt.Prompt, error) {
	systemPrompt, err := prompt.NewPrompt("coder", string(coderPromptTmpl), opts...)
	if err != nil {
		return nil, err
	}
	return systemPrompt, nil
}

func taskPrompt(opts ...prompt.Option) (*prompt.Prompt, error) {
	systemPrompt, err := prompt.NewPrompt("task", string(taskPromptTmpl), opts...)
	if err != nil {
		return nil, err
	}
	return systemPrompt, nil
}

func InitializePrompt(cfg *config.ConfigStore) (string, error) {
	systemPrompt, err := prompt.NewPrompt("initialize", string(initializePromptTmpl))
	if err != nil {
		return "", err
	}
	return systemPrompt.Build(context.Background(), "", "", cfg)
}

func agentPrompt(agent config.Agent, workingDir string) (*prompt.Prompt, error) {
	options := []prompt.Option{
		prompt.WithWorkingDir(workingDir),
		prompt.WithContextPaths(agent.ContextPaths),
	}

	if agent.Prompt == "" {
		if agent.ID == config.AgentTask {
			return taskPrompt(options...)
		}
		return coderPrompt(options...)
	}

	path := home.Long(filepathext.SmartJoin(workingDir, agent.Prompt))
	template, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agent prompt %q: %w", path, err)
	}
	return prompt.NewPrompt(agent.ID, string(template), options...)
}
