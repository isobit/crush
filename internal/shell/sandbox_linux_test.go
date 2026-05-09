//go:build linux

package shell

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildBwrapArgs(t *testing.T) {
	t.Parallel()

	overlayAvailable := BwrapOverlayAvailable()

	t.Run("basic sandbox no network", func(t *testing.T) {
		t.Parallel()
		cfg := &SandboxConfig{Enabled: true}
		got := buildBwrapArgs("/home/user/project", cfg)

		if overlayAvailable {
			require.Contains(t, got, "--overlay-src")
			require.Contains(t, got, "--tmp-overlay")
		} else {
			require.Contains(t, got, "--ro-bind")
		}
		// Common args regardless of overlay support.
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

	t.Run("persistent overlay uses upper/work dirs", func(t *testing.T) {
		t.Parallel()
		if !overlayAvailable {
			t.Skip("overlay not available")
		}
		dir := t.TempDir()
		cfg := &SandboxConfig{Enabled: true, OverlayDir: dir}
		got := buildBwrapArgs("/home/user/project", cfg)
		require.Contains(t, got, "--overlay")
		require.NotContains(t, got, "--tmp-overlay")
	})
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

	// Writing outside CWD and /tmp: with overlay this succeeds silently,
	// without overlay (ro-bind fallback) it fails.
	_, _, err = sh.Exec(t.Context(), "touch /opt/sandbox-test-file && echo overlay-ok")
	if BwrapOverlayAvailable() {
		require.NoError(t, err)
	} else {
		require.Error(t, err)
	}
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
