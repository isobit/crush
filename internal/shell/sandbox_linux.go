//go:build linux

package shell

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

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
// read-write to the real filesystem. Configured hidden paths are masked
// with a placeholder so their real contents are not visible.
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
		path = expandSandboxHome(path)
		args = append(args, "--bind", path, path)
	}

	// Hidden paths are masked with a placeholder so their real contents are
	// not visible inside the sandbox. Binds are appended after the writable
	// binds above so they take precedence when a hidden path nests inside a
	// writable root (bwrap applies later binds last). A directory must be
	// masked by a directory and a file by a file, since a bind mount's
	// source and destination types must match.
	if len(cfg.HiddenPaths) > 0 {
		if file, dir, err := hiddenPathPlaceholders(); err != nil {
			slog.Error("Failed to create sandbox hidden-path placeholder", "error", err)
		} else {
			for _, path := range cfg.HiddenPaths {
				path = expandSandboxHome(path)
				src := file
				if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
					src = dir
				}
				args = append(args, "--ro-bind", src, path)
			}
		}
	}

	if !cfg.Network {
		args = append(args, "--unshare-net")
	}

	return args
}

// hiddenPathNotice is written into the placeholder that masks hidden paths
// so anyone (human or model) inspecting a hidden path learns why it is
// empty rather than assuming the file is missing.
const hiddenPathNotice = `This path is hidden from the sandboxed shell by Crush configuration ` +
	`(options.sandbox.hidden_paths). Its real contents are not accessible here.
`

var (
	hiddenPlaceholderOnce sync.Once
	hiddenPlaceholderFile string
	hiddenPlaceholderDir  string
	hiddenPlaceholderErr  error
)

// hiddenPathPlaceholders lazily creates and returns two placeholder paths
// used to mask hidden files and directories: a notice file and a directory
// containing that notice. They are created once per process and reused for
// every bind, since their contents are static.
func hiddenPathPlaceholders() (file, dir string, err error) {
	hiddenPlaceholderOnce.Do(func() {
		base, e := os.MkdirTemp("", "crush-sandbox-hidden-")
		if e != nil {
			hiddenPlaceholderErr = e
			return
		}
		f := filepath.Join(base, "HIDDEN")
		if e := os.WriteFile(f, []byte(hiddenPathNotice), 0o644); e != nil {
			hiddenPlaceholderErr = e
			return
		}
		d := filepath.Join(base, "hidden")
		if e := os.Mkdir(d, 0o755); e != nil {
			hiddenPlaceholderErr = e
			return
		}
		if e := os.WriteFile(filepath.Join(d, "HIDDEN"), []byte(hiddenPathNotice), 0o644); e != nil {
			hiddenPlaceholderErr = e
			return
		}
		hiddenPlaceholderFile, hiddenPlaceholderDir = f, d
	})
	return hiddenPlaceholderFile, hiddenPlaceholderDir, hiddenPlaceholderErr
}
