package agent

import (
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/shell"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func TestSandboxModeFromConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil opts defaults to auto", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, shell.SandboxModeAuto, sandboxModeFromConfig(nil))
	})

	t.Run("nil sandbox field defaults to auto", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, shell.SandboxModeAuto, sandboxModeFromConfig(&config.Options{}))
	})

	t.Run("nil mode defaults to auto", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, shell.SandboxModeAuto, sandboxModeFromConfig(&config.Options{
			Sandbox: &config.SandboxOptions{},
		}))
	})

	t.Run("on mode", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, shell.SandboxModeOn, sandboxModeFromConfig(&config.Options{
			Sandbox: &config.SandboxOptions{Mode: ptr("on")},
		}))
	})

	t.Run("off mode", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, shell.SandboxModeOff, sandboxModeFromConfig(&config.Options{
			Sandbox: &config.SandboxOptions{Mode: ptr("off")},
		}))
	})

	t.Run("auto mode", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, shell.SandboxModeAuto, sandboxModeFromConfig(&config.Options{
			Sandbox: &config.SandboxOptions{Mode: ptr("auto")},
		}))
	})
}

func TestSandboxNetworkFromConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil opts defaults to false", func(t *testing.T) {
		t.Parallel()
		require.False(t, sandboxNetworkFromConfig(nil))
	})

	t.Run("nil sandbox defaults to false", func(t *testing.T) {
		t.Parallel()
		require.False(t, sandboxNetworkFromConfig(&config.Options{}))
	})

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		require.True(t, sandboxNetworkFromConfig(&config.Options{
			Sandbox: &config.SandboxOptions{Network: ptr(true)},
		}))
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		require.False(t, sandboxNetworkFromConfig(&config.Options{
			Sandbox: &config.SandboxOptions{Network: ptr(false)},
		}))
	})
}

func TestSandboxPersistFromConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil opts defaults to true", func(t *testing.T) {
		t.Parallel()
		require.True(t, sandboxPersistFromConfig(nil))
	})

	t.Run("nil sandbox defaults to true", func(t *testing.T) {
		t.Parallel()
		require.True(t, sandboxPersistFromConfig(&config.Options{}))
	})

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		require.True(t, sandboxPersistFromConfig(&config.Options{
			Sandbox: &config.SandboxOptions{Persist: ptr(true)},
		}))
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		require.False(t, sandboxPersistFromConfig(&config.Options{
			Sandbox: &config.SandboxOptions{Persist: ptr(false)},
		}))
	})
}
