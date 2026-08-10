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

// sandboxNetworkFromConfig extracts the default sandbox network setting
// from config options.
func sandboxNetworkFromConfig(opts *config.Options) bool {
	if opts == nil || opts.Sandbox == nil || opts.Sandbox.Network == nil {
		return false
	}
	return *opts.Sandbox.Network
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
		Mode:           sandboxModeFromConfig(opts),
		NetworkDefault: sandboxNetworkFromConfig(opts),
		HiddenPaths:    sandboxHiddenPathsFromConfig(opts),
	}
}
