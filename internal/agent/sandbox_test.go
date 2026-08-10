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

	t.Run("nil opts defaults to off", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, shell.SandboxModeOff, sandboxModeFromConfig(nil))
	})

	t.Run("nil sandbox field defaults to off", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, shell.SandboxModeOff, sandboxModeFromConfig(&config.Options{}))
	})

	t.Run("nil mode defaults to off", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, shell.SandboxModeOff, sandboxModeFromConfig(&config.Options{
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

func TestSandboxHiddenPathsFromConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil opts", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, sandboxHiddenPathsFromConfig(nil))
	})

	t.Run("nil sandbox", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, sandboxHiddenPathsFromConfig(&config.Options{}))
	})

	t.Run("paths", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, []string{"/home/user/.aws"}, sandboxHiddenPathsFromConfig(&config.Options{
			Sandbox: &config.SandboxOptions{HiddenPaths: []string{"/home/user/.aws"}},
		}))
	})
}
