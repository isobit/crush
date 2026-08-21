package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSandbox(t *testing.T) {
	t.Parallel()

	t.Run("allows configured writable path", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Options: &Options{
			Sandbox: &SandboxOptions{WritablePaths: []string{"/tmp/crush-cache"}},
		}}
		require.NoError(t, cfg.ValidateSandbox())
	})

	t.Run("expands home writable path", func(t *testing.T) {
		t.Parallel()
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		cfg := &Config{Options: &Options{
			Sandbox: &SandboxOptions{WritablePaths: []string{"~/.cache/crush"}},
		}}
		require.NoError(t, cfg.ValidateSandbox())
		require.Equal(t, []string{home + "/.cache/crush"}, cfg.Options.Sandbox.WritablePaths)
	})

	t.Run("rejects protected writable path", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Options: &Options{
			Sandbox: &SandboxOptions{WritablePaths: []string{"/etc"}},
		}}
		require.ErrorContains(t, cfg.ValidateSandbox(), "protected system path")
	})

	t.Run("rejects relative hidden path", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Options: &Options{
			Sandbox: &SandboxOptions{HiddenPaths: []string{".aws"}},
		}}
		require.ErrorContains(t, cfg.ValidateSandbox(), "must be an absolute path")
	})
}
