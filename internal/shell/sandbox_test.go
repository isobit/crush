package shell

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateWritablePaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dirs    []string
		home    string
		wantErr string
	}{
		{
			name: "valid absolute paths",
			dirs: []string{"/home/user/go/pkg/mod", "/home/user/.cache/pip"},
			home: "/home/user",
		},
		{
			name:    "relative path rejected",
			dirs:    []string{"./relative/path"},
			home:    "/home/user",
			wantErr: "must be an absolute path",
		},
		{
			name:    "root rejected",
			dirs:    []string{"/"},
			home:    "/home/user",
			wantErr: "protected system path",
		},
		{
			name:    "/etc rejected",
			dirs:    []string{"/etc"},
			home:    "/home/user",
			wantErr: "protected system path",
		},
		{
			name:    "/usr rejected",
			dirs:    []string{"/usr"},
			home:    "/home/user",
			wantErr: "protected system path",
		},
		{
			name:    "/var rejected",
			dirs:    []string{"/var"},
			home:    "/home/user",
			wantErr: "protected system path",
		},
		{
			name:    "/boot rejected",
			dirs:    []string{"/boot"},
			home:    "/home/user",
			wantErr: "protected system path",
		},
		{
			name:    "/sbin rejected",
			dirs:    []string{"/sbin"},
			home:    "/home/user",
			wantErr: "protected system path",
		},
		{
			name:    "path traversal into /etc via ..",
			dirs:    []string{"/home/../etc"},
			home:    "/home/user",
			wantErr: "protected system path",
		},
		{
			name:    "path traversal into ~/.ssh via ..",
			dirs:    []string{"/home/user/foo/../.ssh"},
			home:    "/home/user",
			wantErr: "protected home directory",
		},
		{
			name:    "~/.ssh rejected",
			dirs:    []string{"/home/user/.ssh"},
			home:    "/home/user",
			wantErr: "protected home directory",
		},
		{
			name:    "~/.ssh/subdir rejected",
			dirs:    []string{"/home/user/.ssh/keys"},
			home:    "/home/user",
			wantErr: "protected home directory",
		},
		{
			name:    "~/.gnupg rejected",
			dirs:    []string{"/home/user/.gnupg"},
			home:    "/home/user",
			wantErr: "protected home directory",
		},
		{
			name:    "~/.config/crush rejected",
			dirs:    []string{"/home/user/.config/crush"},
			home:    "/home/user",
			wantErr: "protected home directory",
		},
		{
			name:    "~/.config/crush/subdir rejected",
			dirs:    []string{"/home/user/.config/crush/sessions"},
			home:    "/home/user",
			wantErr: "protected home directory",
		},
		{
			name: "~/.config (not crush) allowed",
			dirs: []string{"/home/user/.config/other"},
			home: "/home/user",
		},
		{
			name: "empty dirs valid",
			dirs: []string{},
			home: "/home/user",
		},
		{
			name:    "no home still validates system paths",
			dirs:    []string{"/etc"},
			home:    "",
			wantErr: "protected system path",
		},
		{
			name: "no home allows home-relative paths",
			dirs: []string{"/home/user/.ssh"},
			home: "",
		},
		{
			name:    "second dir in list is invalid",
			dirs:    []string{"/home/user/valid", "/etc"},
			home:    "/home/user",
			wantErr: "protected system path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateWritablePaths(tt.dirs, tt.home)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateHiddenPaths(t *testing.T) {
	t.Parallel()

	require.NoError(t, ValidateHiddenPaths([]string{"/home/user/.aws"}))
	require.ErrorContains(t, ValidateHiddenPaths([]string{".aws"}), "must be an absolute path")
}

func TestSandboxWritablePolicy(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	writable := t.TempDir()
	cfg := &SandboxConfig{Enabled: true, WritablePaths: []string{writable}}
	roots := sandboxWritableRoots(cwd, cfg)

	allowed := []string{
		filepath.Join(cwd, "out.txt"),
		filepath.Join(writable, "out.txt"),
		"/tmp/crush-sandbox-test.txt",
		"/dev/null",
	}
	for _, p := range allowed {
		require.True(t, pathWithinRoots(resolveSandboxPath(p), roots),
			"write to %s should be allowed", p)
	}

	denied := []string{
		"/etc/crush-sandbox-should-not-write",
		"/home/other/file",
		"/var/lib/crush-escape",
		"/usr/local/crush-escape",
	}
	for _, p := range denied {
		require.False(t, pathWithinRoots(resolveSandboxPath(p), roots),
			"write to %s should be denied", p)
	}
}

func TestSandboxOpenHandlerDeniesWrite(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	cfg := &SandboxConfig{Enabled: true}
	h := sandboxOpenHandler(cwd, cfg)

	// The denial path returns before reaching the underlying open, so a
	// bare context (no HandlerContext) is sufficient here.
	_, err := h(context.Background(), "/etc/crush-sandbox-should-not-write",
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	require.ErrorIs(t, err, os.ErrPermission)
}

func TestSandboxRedirectionIsContained(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	// A path outside every writable root (cwd, /tmp, /dev, /proc). t.TempDir
	// lives under /tmp, so it cannot be used as an escape target here.
	escape := "/etc/crush-sandbox-escaped.txt"
	cfg := &SandboxConfig{Enabled: true}

	sh := NewShell(&Options{WorkingDir: cwd, Sandbox: cfg})

	// A redirection targeting a path outside the writable roots must fail
	// and must not create the file on the real filesystem.
	_, _, err := sh.Exec(context.Background(), "echo pwned > "+escape)
	require.Error(t, err)
	_, statErr := os.Stat(escape)
	require.True(t, os.IsNotExist(statErr), "file outside sandbox must not be created")

	// A redirection inside the working directory succeeds and persists.
	inside := filepath.Join(cwd, "ok.txt")
	_, _, err = sh.Exec(context.Background(), "echo ok > "+inside)
	require.NoError(t, err)
	data, err := os.ReadFile(inside)
	require.NoError(t, err)
	require.Equal(t, "ok\n", string(data))
}

func TestSandboxOpenHandlerDeniesHiddenRead(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	secret := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("top secret"), 0o644))

	cfg := &SandboxConfig{Enabled: true, HiddenPaths: []string{secret}}
	h := sandboxOpenHandler(cwd, cfg)

	// Reads of a hidden path are rejected even though reads are otherwise
	// allowed anywhere in the sandbox. The denial returns before reaching
	// the underlying open, so a bare context is sufficient here.
	_, err := h(context.Background(), secret, os.O_RDONLY, 0)
	require.ErrorIs(t, err, os.ErrPermission)
}

func TestSandboxHiddenReadIsContained(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	secret := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("top secret"), 0o644))

	cfg := &SandboxConfig{Enabled: true, HiddenPaths: []string{secret}}
	sh := NewShell(&Options{WorkingDir: cwd, Sandbox: cfg})

	// An in-process input redirection from a hidden path must fail rather
	// than exposing the real file contents.
	_, _, err := sh.Exec(context.Background(), "cat < "+secret)
	require.Error(t, err)
}

func TestShouldSandbox(t *testing.T) {
	t.Parallel()

	require.False(t, ShouldSandbox(SandboxModeOff))
	require.True(t, ShouldSandbox(SandboxModeOn))

	// Auto and empty string both delegate to BwrapAvailable — verify
	// consistent behavior (both should return the same value).
	auto := ShouldSandbox(SandboxModeAuto)
	empty := ShouldSandbox("")
	require.Equal(t, auto, empty)

	// An invalid mode should fall back to auto behavior.
	invalid := ShouldSandbox("bogus")
	require.Equal(t, auto, invalid)
}

func TestInvalidSandboxPathError(t *testing.T) {
	t.Parallel()

	err := &InvalidSandboxPathError{Path: "/etc", Reason: "protected system path"}
	require.Equal(t, "invalid sandbox writable path /etc: protected system path", err.Error())
}
