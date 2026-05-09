package shell

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
	// These paths bypass the overlay and write directly to disk.
	WritablePaths []string
	// Network allows network access inside the sandbox when true.
	Network bool
	// OverlayDir, when set, enables persistent overlay mode. Writes
	// outside CWD and WritablePaths go to OverlayDir/upper and persist
	// across commands. OverlayDir/work is used as the overlayfs workdir.
	// When empty, uses --tmp-overlay (writes discarded each command).
	OverlayDir string
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

// ValidateWritablePaths checks that requested writable paths are safe.
// Returns an error describing the first invalid path found.
func ValidateWritablePaths(dirs []string, home string) error {
	for _, dir := range dirs {
		if !filepath.IsAbs(dir) {
			return &InvalidSandboxPathError{Path: dir, Reason: "must be an absolute path"}
		}
		cleaned := filepath.Clean(dir)
		for _, p := range protectedPaths {
			if cleaned == p {
				return &InvalidSandboxPathError{Path: dir, Reason: "protected system path"}
			}
		}
		if home != "" {
			for _, sub := range protectedHomeDirs {
				protected := filepath.Join(home, sub)
				if cleaned == protected || strings.HasPrefix(cleaned, protected+"/") {
					return &InvalidSandboxPathError{Path: dir, Reason: "protected home directory"}
				}
			}
		}
	}
	return nil
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

var (
	bwrapAvailable     bool
	bwrapAvailableOnce sync.Once

	bwrapOverlayAvailable     bool
	bwrapOverlayAvailableOnce sync.Once
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

// BwrapOverlayAvailable returns whether bwrap overlay support works on
// this system (requires bwrap ≥ 0.11.0 and kernel support for
// unprivileged overlayfs with userxattr).
func BwrapOverlayAvailable() bool {
	bwrapOverlayAvailableOnce.Do(func() {
		if !BwrapAvailable() {
			bwrapOverlayAvailable = false
			return
		}
		// Probe: try a tmp-overlay mount. If it fails, overlay isn't
		// usable on this system.
		cmd := exec.Command("bwrap", "--overlay-src", "/", "--tmp-overlay", "/", "--", "true")
		bwrapOverlayAvailable = cmd.Run() == nil
	})
	return bwrapOverlayAvailable
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
