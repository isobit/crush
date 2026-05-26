//go:build linux

package shell

import (
	"context"
	"os"
	"path/filepath"

	"mvdan.cc/sh/v3/interp"
)

// sandboxHandler returns an ExecHandler that wraps external command
// execution inside bubblewrap (bwrap) for filesystem and network
// isolation. Uses overlayfs to provide a writable filesystem view
// without modifying the real filesystem.
func sandboxHandler(cwd string, cfg *SandboxConfig) func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if cfg == nil || !cfg.Enabled {
				return next(ctx, args)
			}
			if len(args) == 0 {
				return next(ctx, args)
			}

			// Don't sandbox bwrap itself (avoid double-wrapping).
			if args[0] == "bwrap" {
				return next(ctx, args)
			}

			bwrapArgs := buildBwrapArgs(cwd, cfg)
			bwrapArgs = append(bwrapArgs, "--")
			bwrapArgs = append(bwrapArgs, args...)

			return next(ctx, append([]string{"bwrap"}, bwrapArgs...))
		}
	}
}

// buildBwrapArgs constructs the bubblewrap argument list for the given
// configuration. Prefers overlayfs when available (persistent or tmpfs);
// falls back to --ro-bind when the kernel doesn't support unprivileged
// overlayfs.
func buildBwrapArgs(cwd string, cfg *SandboxConfig) []string {
	var args []string

	if BwrapOverlayAvailable() {
		if cfg.OverlayDir != "" {
			// Persistent overlay: writes accumulate in upper dir across
			// commands within the session.
			upper := filepath.Join(cfg.OverlayDir, "upper")
			work := filepath.Join(cfg.OverlayDir, "work")
			_ = os.MkdirAll(upper, 0o755)
			_ = os.MkdirAll(work, 0o755)
			args = []string{
				"--overlay-src", "/",
				"--overlay", upper, work, "/",
			}
		} else {
			// Temporary overlay: writes are discarded when command exits.
			args = []string{
				"--overlay-src", "/",
				"--tmp-overlay", "/",
			}
		}
	} else {
		// Fallback: read-only bind mount (no overlay support).
		args = []string{
			"--ro-bind", "/", "/",
		}
	}

	// CWD bind takes precedence over the overlay — writes persist to
	// real disk.
	args = append(args,
		"--bind", cwd, cwd,
		"--bind", "/tmp", "/tmp",
		"--dev", "/dev",
		"--proc", "/proc",
		"--die-with-parent",
		"--unshare-pid",
	)

	// Additional writable paths punch through the overlay to real disk.
	for _, path := range cfg.WritablePaths {
		args = append(args, "--bind", path, path)
	}

	if !cfg.Network {
		args = append(args, "--unshare-net")
	}

	return args
}
