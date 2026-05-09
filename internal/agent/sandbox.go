package agent

import (
	"path/filepath"

	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/shell"
)

// sandboxModeFromConfig extracts the sandbox mode from config options,
// defaulting to "auto" when unset.
func sandboxModeFromConfig(opts *config.Options) shell.SandboxMode {
	if opts == nil || opts.Sandbox == nil || opts.Sandbox.Mode == nil {
		return shell.SandboxModeAuto
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

// sandboxPersistFromConfig extracts whether overlay persistence is
// enabled, defaulting to true.
func sandboxPersistFromConfig(opts *config.Options) bool {
	if opts == nil || opts.Sandbox == nil || opts.Sandbox.Persist == nil {
		return true
	}
	return *opts.Sandbox.Persist
}

// buildBashSandboxOptions constructs sandbox options from config for the
// bash tool. When persist is enabled, the overlay directory is placed
// inside the data directory.
func buildBashSandboxOptions(opts *config.Options) tools.BashSandboxOptions {
	mode := sandboxModeFromConfig(opts)
	network := sandboxNetworkFromConfig(opts)
	persist := sandboxPersistFromConfig(opts)

	var overlayDir string
	if persist && shell.ShouldSandbox(mode) {
		dataDir := opts.DataDirectory
		if dataDir == "" {
			dataDir = ".crush"
		}
		overlayDir = filepath.Join(dataDir, "sandbox")
	}

	return tools.BashSandboxOptions{
		Mode:           mode,
		NetworkDefault: network,
		OverlayDir:     overlayDir,
	}
}
