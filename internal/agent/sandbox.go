package agent

import (
	"slices"

	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/shell"
)

// sandboxModeFromConfig extracts the sandbox mode from config options,
// defaulting to "off" when unset.
func sandboxModeFromConfig(opts *config.Options) shell.SandboxMode {
	if opts == nil || opts.Sandbox == nil || opts.Sandbox.Mode == nil {
		return shell.SandboxModeOff
	}
	return shell.SandboxMode(*opts.Sandbox.Mode)
}

// sandboxHiddenPathsFromConfig extracts the paths to hide inside the
// sandbox from config options.
func sandboxHiddenPathsFromConfig(opts *config.Options) []string {
	if opts == nil || opts.Sandbox == nil {
		return nil
	}
	return opts.Sandbox.HiddenPaths
}

func sandboxWritablePathsFromConfig(opts *config.Options) []string {
	if opts == nil || opts.Sandbox == nil {
		return nil
	}
	return opts.Sandbox.WritablePaths
}

// buildBashSandboxOptions constructs sandbox options from config for the
// bash tool.
func buildBashSandboxOptions(cfg *config.Config) tools.BashSandboxOptions {
	opts := cfg.Options
	writablePaths := sandboxWritablePathsFromConfig(opts)
	// Git needs write access to the shared metadata directory from linked
	// worktrees, so it is granted by default rather than per command.
	if gitDir := cfg.Git.LinkedWorktreeCommonDir; gitDir != "" && !slices.Contains(writablePaths, gitDir) {
		writablePaths = append(slices.Clone(writablePaths), gitDir)
	}

	return tools.BashSandboxOptions{
		Mode:             sandboxModeFromConfig(opts),
		WritablePaths:    writablePaths,
		BlockedCommands:  cfg.Tools.Bash.BlockedCommands,
		BlockedArguments: cfg.Tools.Bash.BlockedArguments,
		HiddenPaths:      sandboxHiddenPathsFromConfig(opts),
	}
}
