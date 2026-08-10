//go:build linux

package shell

import (
	"context"

	"mvdan.cc/sh/v3/interp"
)

// sandboxHandler returns an ExecHandler that wraps external command
// execution inside bubblewrap (bwrap) for filesystem and network
// isolation. The root filesystem is mounted read-only; only the working
// directory, /tmp, and configured writable paths can be written.
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
// configuration. The root filesystem is bound read-only; the working
// directory, /tmp, and any configured writable paths are bound
// read-write to the real filesystem.
func buildBwrapArgs(cwd string, cfg *SandboxConfig) []string {
	var args []string

	// Root is read-only; CWD and /tmp binds below grant real-disk writes.
	args = append(args,
		"--die-with-parent",
		"--unshare-pid",
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
		"--bind", "/tmp", "/tmp",
		"--bind", cwd, cwd,
	)

	// Additional writable paths bind read-write to the real filesystem.
	for _, path := range cfg.WritablePaths {
		args = append(args, "--bind", path, path)
	}

	if !cfg.Network {
		args = append(args, "--unshare-net")
	}

	return args
}
