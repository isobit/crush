package agent

import (
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

// buildBashSandboxOptions constructs sandbox options from config for the
// bash tool.
func buildBashSandboxOptions(opts *config.Options) tools.BashSandboxOptions {
	return tools.BashSandboxOptions{
		Mode:        sandboxModeFromConfig(opts),
		HiddenPaths: sandboxHiddenPathsFromConfig(opts),
	}
}
