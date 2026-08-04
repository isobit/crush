package config

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestValidateProfileName(t *testing.T) {
	t.Parallel()

	valid := []string{"", "psqr", "work-2", "team.staging", "a_b", "A1"}
	for _, name := range valid {
		require.NoError(t, validateProfileName(name), "expected %q to be valid", name)
	}

	invalid := []string{"..", "../evil", "a/b", "a\\b", ".hidden", "-dash", "with space", "foo/../bar"}
	for _, name := range invalid {
		require.Error(t, validateProfileName(name), "expected %q to be invalid", name)
	}
}

func TestProfileConfigPaths_RespectEnvOverrides(t *testing.T) {
	cfgDir := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", cfgDir)
	t.Setenv("CRUSH_GLOBAL_DATA", dataDir)

	require.Equal(t, filepath.Join(cfgDir, "crush.psqr.json"), GlobalConfigProfile("psqr"))
	require.Equal(t, filepath.Join(dataDir, "crush.psqr.json"), GlobalConfigDataProfile("psqr"))
}

func TestLookupConfigs_ProfileLayering(t *testing.T) {
	cfgDir := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", cfgDir)
	t.Setenv("CRUSH_GLOBAL_DATA", dataDir)

	got := lookupConfigs(t.TempDir(), "psqr")

	baseCfg := slices.Index(got, GlobalConfig())
	profCfg := slices.Index(got, GlobalConfigProfile("psqr"))
	baseData := slices.Index(got, GlobalConfigData())
	profData := slices.Index(got, GlobalConfigDataProfile("psqr"))

	require.NotEqual(t, -1, baseCfg, "base config missing: %v", got)
	require.NotEqual(t, -1, profCfg, "profile config missing: %v", got)
	require.NotEqual(t, -1, baseData, "base data config missing: %v", got)
	require.NotEqual(t, -1, profData, "profile data config missing: %v", got)

	// Profile variants must sit directly on top of their base so they
	// override it while the base stays in the chain (shared hooks etc.).
	require.Greater(t, profCfg, baseCfg, "profile config must layer after base config")
	require.Greater(t, profData, baseData, "profile data config must layer after base data config")
}

func TestLookupConfigs_NoProfileOmitsVariants(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", cfgDir)
	t.Setenv("CRUSH_GLOBAL_DATA", cfgDir)

	got := lookupConfigs(t.TempDir(), "")
	require.NotContains(t, got, GlobalConfigProfile("psqr"))
	require.NotContains(t, got, GlobalConfigDataProfile("psqr"))
}

func TestLoad_ProfileLayersOverBaseConfig(t *testing.T) {
	cfgDir := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", cfgDir)
	t.Setenv("CRUSH_GLOBAL_DATA", dataDir)

	// Base config carries a shared setting; the profile only overrides one
	// field, so the base value must be preserved (reused).
	require.NoError(t, os.WriteFile(
		filepath.Join(cfgDir, "crush.json"),
		[]byte(`{"options":{"debug":true,"tui":{"compact_mode":false}}}`),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(cfgDir, "crush.psqr.json"),
		[]byte(`{"options":{"tui":{"compact_mode":true}}}`),
		0o644,
	))

	project := t.TempDir()
	store, err := Load(project, filepath.Join(project, ".crush"), false, WithProfile("psqr"))
	require.NoError(t, err)

	cfg := store.Config()
	require.True(t, cfg.Options.Debug, "base config value should be reused under a profile")
	require.True(t, cfg.Options.TUI.CompactMode, "profile config should override the base")
}

func TestLoad_ProfileIsolatesGlobalDataWrites(t *testing.T) {
	cfgDir := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", cfgDir)
	t.Setenv("CRUSH_GLOBAL_DATA", dataDir)

	project := t.TempDir()
	store, err := Load(project, filepath.Join(project, ".crush"), false, WithProfile("psqr"))
	require.NoError(t, err)

	// ScopeGlobal writes (OAuth tokens, preferred models, etc.) must land in
	// the profile data file, not the shared base data file.
	require.Equal(t, GlobalConfigDataProfile("psqr"), store.globalDataPath)

	require.NoError(t, store.SetConfigField(ScopeGlobal, "mcp.demo.oauth_token", map[string]any{
		"access_token": "secret",
	}))

	profileData, err := os.ReadFile(GlobalConfigDataProfile("psqr"))
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(profileData, "mcp.demo.oauth_token.access_token").Exists(),
		"token must be written to the profile data file")

	// The shared base data file must not receive the profile's token.
	if baseData, err := os.ReadFile(GlobalConfigData()); err == nil {
		require.False(t, gjson.GetBytes(baseData, "mcp.demo.oauth_token").Exists(),
			"token must not leak into the shared base data file")
	}
}

func TestLoad_InvalidProfileNameFails(t *testing.T) {
	project := t.TempDir()
	_, err := Load(project, filepath.Join(project, ".crush"), false, WithProfile("../evil"))
	require.Error(t, err)
}

func TestReloadFromDisk_PreservesProfileChain(t *testing.T) {
	cfgDir := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", cfgDir)
	t.Setenv("CRUSH_GLOBAL_DATA", dataDir)

	require.NoError(t, os.WriteFile(
		filepath.Join(cfgDir, "crush.psqr.json"),
		[]byte(`{"options":{"tui":{"compact_mode":true}}}`),
		0o644,
	))

	project := t.TempDir()
	store, err := Load(project, filepath.Join(project, ".crush"), false, WithProfile("psqr"))
	require.NoError(t, err)
	require.True(t, store.Config().Options.TUI.CompactMode)

	require.NoError(t, store.ReloadFromDisk(context.Background()))
	require.True(t, store.Config().Options.TUI.CompactMode,
		"reload must keep the profile layer in the config chain")
	require.Equal(t, "psqr", store.profile)
}
