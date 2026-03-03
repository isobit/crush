package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyEnvOverrides(t *testing.T) {
	t.Run("bool field", func(t *testing.T) {
		opts := &Options{}
		t.Setenv("CRUSH_DEBUG", "true")
		applyEnvOverrides(opts)
		require.True(t, opts.Debug)
	})

	t.Run("bool pointer field", func(t *testing.T) {
		opts := &Options{}
		t.Setenv("CRUSH_HASHLINE_EDIT", "true")
		applyEnvOverrides(opts)
		require.NotNil(t, opts.HashlineEdit)
		require.True(t, *opts.HashlineEdit)
	})

	t.Run("bool pointer field false", func(t *testing.T) {
		b := true
		opts := &Options{HashlineEdit: &b}
		t.Setenv("CRUSH_HASHLINE_EDIT", "false")
		applyEnvOverrides(opts)
		require.NotNil(t, opts.HashlineEdit)
		require.False(t, *opts.HashlineEdit)
	})

	t.Run("string field", func(t *testing.T) {
		opts := &Options{}
		t.Setenv("CRUSH_DATA_DIRECTORY", "/tmp/test")
		applyEnvOverrides(opts)
		require.Equal(t, "/tmp/test", opts.DataDirectory)
	})

	t.Run("string slice field", func(t *testing.T) {
		opts := &Options{}
		t.Setenv("CRUSH_DISABLED_TOOLS", "bash, sourcegraph")
		applyEnvOverrides(opts)
		require.Equal(t, []string{"bash", "sourcegraph"}, opts.DisabledTools)
	})

	t.Run("unset env does not override", func(t *testing.T) {
		opts := &Options{Debug: true}
		applyEnvOverrides(opts)
		require.True(t, opts.Debug)
	})

	t.Run("invalid bool logs warning", func(t *testing.T) {
		opts := &Options{}
		t.Setenv("CRUSH_DEBUG", "notabool")
		applyEnvOverrides(opts)
		require.False(t, opts.Debug)
	})

	t.Run("nil options", func(t *testing.T) {
		applyEnvOverrides(nil)
	})

	t.Run("existing env vars still work", func(t *testing.T) {
		t.Setenv("CRUSH_DISABLE_PROVIDER_AUTO_UPDATE", "true")
		t.Setenv("CRUSH_DISABLE_DEFAULT_PROVIDERS", "1")
		opts := &Options{}
		applyEnvOverrides(opts)
		require.True(t, opts.DisableProviderAutoUpdate)
		require.True(t, opts.DisableDefaultProviders)
	})
}
