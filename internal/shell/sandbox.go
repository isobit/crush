package shell

import (
	"context"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"mvdan.cc/sh/v3/interp"
)

// SandboxMode controls whether sandbox isolation is enabled.
type SandboxMode string

const (
	// SandboxModeAuto enables sandbox when bwrap is available on Linux.
	SandboxModeAuto SandboxMode = "auto"
	// SandboxModeOn enables sandbox unconditionally (fails if unavailable).
	SandboxModeOn SandboxMode = "on"
	// SandboxModeOff disables sandbox entirely.
	SandboxModeOff SandboxMode = "off"
)

// SandboxConfig controls OS-level command isolation via bubblewrap.
type SandboxConfig struct {
	// Enabled indicates whether sandbox is active for this shell.
	Enabled bool
	// WritablePaths are additional paths (files or directories) to bind
	// read-write inside the sandbox (beyond the working directory and /tmp).
	// These paths write directly to the real filesystem.
	WritablePaths []string
	// HiddenPaths are files or directories that must not be visible inside
	// the sandbox. Each is masked by binding a placeholder over it (a notice
	// file or, for directories, a notice-containing directory) and reads of
	// them via the shell's own in-process I/O are rejected.
	HiddenPaths []string
	// Network allows network access inside the sandbox when true.
	Network bool
}

// protectedPaths are directories that cannot be requested as writable.
var protectedPaths = []string{
	"/",
	"/bin",
	"/boot",
	"/etc",
	"/lib",
	"/lib64",
	"/sbin",
	"/usr",
	"/var",
	"/proc",
	"/sys",
	"/dev",
}

// protectedHomeDirs are subdirectories of $HOME that cannot be requested
// as writable.
var protectedHomeDirs = []string{
	".ssh",
	".gnupg",
	".config/crush",
}

// resolveSandboxPaths expands a leading home-directory marker and rejects
// relative paths. It returns cleaned absolute paths for sandbox use.
func resolveSandboxPaths(paths []string, home string, invalidPath func(string) error) ([]string, error) {
	resolvedPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		resolved := path
		if path == "~" || strings.HasPrefix(path, "~/") {
			resolved = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
		if !filepath.IsAbs(resolved) {
			return nil, invalidPath(path)
		}
		resolvedPaths = append(resolvedPaths, filepath.Clean(resolved))
	}
	return resolvedPaths, nil
}

// ResolveWritablePaths expands a leading home-directory marker and validates
// requested writable paths. Relative paths remain invalid. It returns the
// cleaned absolute paths that must be passed to the sandbox.
func ResolveWritablePaths(dirs []string, home string) ([]string, error) {
	paths, err := resolveSandboxPaths(dirs, home, func(path string) error {
		return &InvalidSandboxPathError{Path: path, Reason: "must be an absolute path"}
	})
	if err != nil {
		return nil, err
	}
	for i, path := range paths {
		if slices.Contains(protectedPaths, path) {
			return nil, &InvalidSandboxPathError{Path: dirs[i], Reason: "protected system path"}
		}
		if home != "" {
			for _, sub := range protectedHomeDirs {
				protected := filepath.Join(home, sub)
				if path == protected || strings.HasPrefix(path, protected+"/") {
					return nil, &InvalidSandboxPathError{Path: dirs[i], Reason: "protected home directory"}
				}
			}
		}
	}
	return paths, nil
}

// ResolveHiddenPaths expands a leading home-directory marker and rejects
// relative paths. It returns the cleaned absolute paths to hide.
func ResolveHiddenPaths(paths []string, home string) ([]string, error) {
	return resolveSandboxPaths(paths, home, func(path string) error {
		return &InvalidSandboxHiddenPathError{Path: path}
	})
}

// ValidateWritablePaths checks that requested writable paths are safe.
// Returns an error describing the first invalid path found.
func ValidateWritablePaths(dirs []string, home string) error {
	_, err := ResolveWritablePaths(dirs, home)
	return err
}

func ValidateHiddenPaths(paths []string, home string) error {
	_, err := ResolveHiddenPaths(paths, home)
	return err
}

// InvalidSandboxPathError is returned when a requested writable path is
// not allowed.
type InvalidSandboxPathError struct {
	Path   string
	Reason string
}

func (e *InvalidSandboxPathError) Error() string {
	return "invalid sandbox writable path " + e.Path + ": " + e.Reason
}

type InvalidSandboxHiddenPathError struct {
	Path string
}

func (e *InvalidSandboxHiddenPathError) Error() string {
	return "invalid sandbox hidden path " + e.Path + ": must be an absolute path"
}

var (
	bwrapAvailable     bool
	bwrapAvailableOnce sync.Once
)

// BwrapAvailable returns whether bubblewrap (bwrap) is installed and the
// platform is Linux.
func BwrapAvailable() bool {
	bwrapAvailableOnce.Do(func() {
		if runtime.GOOS != "linux" {
			bwrapAvailable = false
			return
		}
		_, err := exec.LookPath("bwrap")
		bwrapAvailable = err == nil
	})
	return bwrapAvailable
}

// sandboxOpenHandler returns an [interp.OpenHandlerFunc] that applies the
// sandbox's writable-path policy to files the shell opens itself:
// redirections (>, >>, <), heredoc/herestring targets, and any other I/O
// mvdan/sh performs in-process. External commands are already contained by
// bwrap (see sandboxHandler); those in-process opens are not, because they
// run in the Crush process and would otherwise reach the real filesystem
// regardless of the sandbox.
//
// Reads are allowed from anywhere — the sandbox mounts the root filesystem
// readable — except configured HiddenPaths, which are rejected so their real
// contents stay hidden (bwrap masks them for external commands with a
// placeholder). Writes are permitted only inside the working directory, /tmp,
// /dev, /proc, and any configured WritablePaths; a write anywhere else is
// rejected with a permission error so it cannot modify the real filesystem.
func sandboxOpenHandler(cwd string, cfg *SandboxConfig) interp.OpenHandlerFunc {
	def := interp.DefaultOpenHandler()
	roots := sandboxWritableRoots(cwd, cfg)
	hidden := sandboxHiddenRoots(cfg)
	return func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		if path != "" && !(runtime.GOOS == "windows" && path == "/dev/null") {
			abs := path
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(interp.HandlerCtx(ctx).Dir, abs)
			}
			abs = resolveSandboxPath(abs)
			// Hidden paths are neither readable nor writable in-process:
			// bwrap masks them for external commands, but redirections and
			// heredocs the shell opens itself would otherwise reach the real
			// file, so reject them here too.
			if pathWithinRoots(abs, hidden) {
				return nil, &os.PathError{
					Op:   "open",
					Path: path,
					Err:  fs.ErrPermission,
				}
			}
			if isWriteFlag(flag) && !pathWithinRoots(abs, roots) {
				return nil, &os.PathError{
					Op:   "open",
					Path: path,
					Err:  fs.ErrPermission,
				}
			}
		}
		return def(ctx, path, flag, perm)
	}
}

// sandboxHiddenRoots returns the symlink-resolved roots that must be
// hidden from the sandboxed shell's own in-process I/O. It mirrors the
// placeholder binds buildBwrapArgs applies to external commands.
func sandboxHiddenRoots(cfg *SandboxConfig) []string {
	roots := make([]string, 0, len(cfg.HiddenPaths))
	for _, p := range cfg.HiddenPaths {
		roots = append(roots, resolveSandboxPath(p))
	}
	return roots
}

// sandboxWritableRoots returns the filesystem roots a sandboxed shell may
// write to via its own redirections. It mirrors the writable mounts that
// buildBwrapArgs grants external commands: the working directory, /tmp,
// /dev, /proc, and any explicitly configured WritablePaths. Roots are
// symlink-resolved so the comparison matches the resolved target path.
func sandboxWritableRoots(cwd string, cfg *SandboxConfig) []string {
	raw := make([]string, 0, len(cfg.WritablePaths)+4)
	if cwd != "" {
		raw = append(raw, cwd)
	}
	raw = append(raw, "/tmp", "/dev", "/proc")
	raw = append(raw, cfg.WritablePaths...)

	roots := make([]string, 0, len(raw))
	for _, r := range raw {
		roots = append(roots, resolveSandboxPath(r))
	}
	return roots
}

// pathWithinRoots reports whether path is equal to, or nested under, any of
// the given roots. All inputs are expected to be cleaned absolute paths.
func pathWithinRoots(path string, roots []string) bool {
	for _, root := range roots {
		if path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// isWriteFlag reports whether an open flag would allow modifying the file.
// O_RDONLY is 0, so any of the write-intent bits means a write.
func isWriteFlag(flag int) bool {
	return flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_TRUNC) != 0
}

func expandSandboxHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

// resolveSandboxPath cleans path and resolves symlinks so a symlinked
// parent directory cannot be used to escape the writable roots. The final
// component may not exist yet (file creation), so it falls back to
// resolving the deepest existing ancestor and re-attaching the remainder.
func resolveSandboxPath(path string) string {
	cleaned := filepath.Clean(expandSandboxHome(path))
	if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
		return resolved
	}
	dir, base := filepath.Split(cleaned)
	if dir == "" {
		return cleaned
	}
	if resolvedDir, err := filepath.EvalSymlinks(filepath.Clean(dir)); err == nil {
		return filepath.Join(resolvedDir, base)
	}
	return cleaned
}

// ShouldSandbox determines whether sandboxing should be enabled given the
// configured mode.
func ShouldSandbox(mode SandboxMode) bool {
	switch mode {
	case SandboxModeOn:
		return true
	case SandboxModeOff:
		return false
	case SandboxModeAuto, "":
		return BwrapAvailable()
	default:
		return BwrapAvailable()
	}
}
