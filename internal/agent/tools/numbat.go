package tools

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"charm.land/fantasy"
)

//go:embed numbat.md
var numbatDescription string

const NumbatToolName = "numbat"

// numbatTimeout is the maximum time a numbat evaluation can run.
const numbatTimeout = 30 * time.Second

// NumbatParams are the parameters for the numbat tool.
type NumbatParams struct {
	Code string `json:"code" description:"Numbat code to evaluate — supports units, dimensional analysis, and scientific computation"`
}

// NewNumbatTool creates a tool that evaluates expressions using the numbat CLI.
// Numbat is a statically-typed language for scientific computation with
// first-class physical dimensions and units. Returns an error response if
// numbat is not installed.
func NewNumbatTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		NumbatToolName,
		numbatDescription,
		func(ctx context.Context, params NumbatParams, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Code == "" {
				return fantasy.NewTextErrorResponse("code is required"), nil
			}

			result, err := evalNumbat(ctx, params.Code)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			return fantasy.NewTextResponse(result), nil
		},
	)
}

// evalNumbat executes numbat code via the CLI and returns the output.
func evalNumbat(ctx context.Context, code string) (string, error) {
	numbatPath, err := exec.LookPath("numbat")
	if err != nil {
		return "", fmt.Errorf("numbat is not installed or not on $PATH: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, numbatTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, numbatPath,
		"--no-config",
		"--no-init",
		"--color", "never",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdin = strings.NewReader(code)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If the context timed out, return a clear message.
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("numbat evaluation timed out after %s", numbatTimeout)
		}

		// Numbat writes errors to stderr.
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("numbat error:\n%s", errMsg)
		}

		return "", fmt.Errorf("numbat error: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "(no output)", nil
	}

	return output, nil
}
