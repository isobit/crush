//go:build linux

package shell

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildBwrapArgs(t *testing.T) {
	t.Parallel()

	t.Run("basic sandbox no network", func(t *testing.T) {
		t.Parallel()
		cfg := &SandboxConfig{Enabled: true}
		got := buildBwrapArgs("/home/user/project", cfg)

		// Root is mounted read-only; CWD and /tmp are writable binds.
		require.Contains(t, got, "--ro-bind")
		require.Contains(t, got, "--bind")
		require.Contains(t, got, "--unshare-net")
		require.Contains(t, got, "--unshare-pid")
		require.Contains(t, got, "--die-with-parent")
	})

	t.Run("network enabled omits unshare-net", func(t *testing.T) {
		t.Parallel()
		cfg := &SandboxConfig{Enabled: true, Network: true}
		got := buildBwrapArgs("/home/user/project", cfg)
		require.NotContains(t, got, "--unshare-net")
	})

	t.Run("writable paths appear as bind mounts", func(t *testing.T) {
		t.Parallel()
		cfg := &SandboxConfig{
			Enabled:       true,
			WritablePaths: []string{"/home/user/go/pkg/mod"},
		}
		got := buildBwrapArgs("/home/user/project", cfg)
		// Find the writable path bind.
		found := false
		for i, arg := range got {
			if arg == "--bind" && i+2 < len(got) && got[i+1] == "/home/user/go/pkg/mod" {
				found = true
				break
			}
		}
		require.True(t, found, "expected --bind /home/user/go/pkg/mod in args: %v", got)
	})

	t.Run("home-relative paths are expanded before binding", func(t *testing.T) {
		t.Parallel()
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		cfg := &SandboxConfig{
			Enabled:       true,
			WritablePaths: []string{"~/.cache/crush"},
			HiddenPaths:   []string{"~/.passage/identities"},
		}
		got := buildBwrapArgs("/home/user/project", cfg)

		require.Contains(t, got, filepath.Join(home, ".cache/crush"))
		require.Contains(t, got, filepath.Join(home, ".passage/identities"))
		require.NotContains(t, got, "~/.cache/crush")
		require.NotContains(t, got, "~/.passage/identities")
	})

	t.Run("hidden paths mask with a ro-bind placeholder", func(t *testing.T) {
		t.Parallel()
		// Use a real file and a real dir so the placeholder type matches.
		dir := t.TempDir()
		hiddenFile := filepath.Join(dir, "secret.txt")
		require.NoError(t, os.WriteFile(hiddenFile, []byte("x"), 0o644))
		hiddenDir := filepath.Join(dir, "secretdir")
		require.NoError(t, os.Mkdir(hiddenDir, 0o755))

		cfg := &SandboxConfig{
			Enabled:     true,
			HiddenPaths: []string{hiddenFile, hiddenDir},
		}
		got := buildBwrapArgs("/home/user/project", cfg)

		file, placeholderDir, err := hiddenPathPlaceholders()
		require.NoError(t, err)

		requireBind := func(src, dst string) {
			found := false
			for i, arg := range got {
				if arg == "--ro-bind" && i+2 < len(got) && got[i+1] == src && got[i+2] == dst {
					found = true
					break
				}
			}
			require.True(t, found, "expected --ro-bind %s %s in args: %v", src, dst, got)
		}
		requireBind(file, hiddenFile)
		requireBind(placeholderDir, hiddenDir)
	})
}

func TestSandboxHandler_HiddenPathMasked(t *testing.T) {
	t.Parallel()

	if !BwrapAvailable() {
		t.Skip("bwrap not available")
	}

	dir := t.TempDir()
	secretDir := t.TempDir()
	secret := filepath.Join(secretDir, "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("top secret"), 0o644))

	sh := NewShell(&Options{
		WorkingDir: dir,
		Sandbox: &SandboxConfig{
			Enabled:     true,
			HiddenPaths: []string{secret},
		},
	})

	// An external command reading the hidden file sees the placeholder
	// notice, never the real contents.
	stdout, _, err := sh.Exec(t.Context(), "cat "+secret)
	require.NoError(t, err)
	require.NotContains(t, stdout, "top secret")
	require.Contains(t, stdout, "hidden from the sandboxed shell")
}

func TestSandboxHandler_Nil(t *testing.T) {
	t.Parallel()

	// When sandbox is nil, commands pass through unchanged.
	sh := NewShell(&Options{
		WorkingDir: t.TempDir(),
		Sandbox:    nil,
	})

	stdout, _, err := sh.Exec(t.Context(), "echo hello")
	require.NoError(t, err)
	require.Equal(t, "hello\n", stdout)
}

func TestSandboxHandler_EnabledFalse(t *testing.T) {
	t.Parallel()

	// A non-nil config with Enabled=false should pass through.
	sh := NewShell(&Options{
		WorkingDir: t.TempDir(),
		Sandbox: &SandboxConfig{
			Enabled: false,
			Network: true,
		},
	})

	stdout, _, err := sh.Exec(t.Context(), "echo passthrough")
	require.NoError(t, err)
	require.Equal(t, "passthrough\n", stdout)
}

func TestSandboxHandler_WrapsExternalCommands(t *testing.T) {
	t.Parallel()

	// An external command in sandbox mode: the handler rewrites it to
	// "bwrap ... -- ls /". If bwrap isn't available this will fail with
	// exit code 127 which proves the wrapping happened.
	sh := NewShell(&Options{
		WorkingDir: t.TempDir(),
		Sandbox: &SandboxConfig{
			Enabled: true,
		},
	})

	stdout, _, err := sh.Exec(t.Context(), "ls /")
	if !BwrapAvailable() {
		// bwrap not available — should error trying to exec bwrap.
		require.Error(t, err)
	} else {
		require.NoError(t, err)
		require.NotEmpty(t, stdout)
	}
}

func TestSandboxHandler_EnabledWithBwrap(t *testing.T) {
	t.Parallel()

	if !BwrapAvailable() {
		t.Skip("bwrap not available")
	}

	dir := t.TempDir()
	sh := NewShell(&Options{
		WorkingDir: dir,
		Sandbox: &SandboxConfig{
			Enabled: true,
		},
	})

	// Basic command should work inside the sandbox.
	stdout, _, err := sh.Exec(t.Context(), "echo hello")
	require.NoError(t, err)
	require.Equal(t, "hello\n", stdout)

	// Writing to the working dir should succeed.
	stdout, _, err = sh.Exec(t.Context(), "touch "+dir+"/test.txt && echo ok")
	require.NoError(t, err)
	require.Equal(t, "ok\n", stdout)

	// Writing to /tmp should succeed.
	stdout, _, err = sh.Exec(t.Context(), "touch /tmp/sandbox-test-$$ && echo tmp-ok")
	require.NoError(t, err)
	require.Contains(t, stdout, "tmp-ok")

	// Writing outside CWD and /tmp hits the read-only root and fails.
	_, _, err = sh.Exec(t.Context(), "touch /opt/sandbox-test-file && echo ro-ok")
	require.Error(t, err)
}

func TestSandboxHandler_NetworkBlocked(t *testing.T) {
	t.Parallel()

	if !BwrapAvailable() {
		t.Skip("bwrap not available")
	}

	dir := t.TempDir()
	sh := NewShell(&Options{
		WorkingDir: dir,
		Sandbox: &SandboxConfig{
			Enabled: true,
			Network: false,
		},
	})

	// In a network namespace with --unshare-net, connecting to an
	// external address should fail.
	_, _, err := sh.Exec(t.Context(), "cat < /dev/tcp/1.1.1.1/80")
	require.Error(t, err)
}

func TestSandboxHandler_NetworkAllowed(t *testing.T) {
	t.Parallel()

	if !BwrapAvailable() {
		t.Skip("bwrap not available")
	}

	dir := t.TempDir()
	sh := NewShell(&Options{
		WorkingDir: dir,
		Sandbox: &SandboxConfig{
			Enabled: true,
			Network: true,
		},
	})

	// With network allowed, network interfaces should be present.
	stdout, _, err := sh.Exec(t.Context(), "ls /sys/class/net/ 2>/dev/null")
	require.NoError(t, err)
	// Should have at least "lo" visible.
	require.Contains(t, stdout, "lo")
}
