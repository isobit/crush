package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeResolver struct {
	calls int
	err   error
}

func (r *fakeResolver) ResolveValue(value string) (string, error) {
	r.calls++
	if r.err != nil {
		return "", r.err
	}
	return "resolved-" + value, nil
}

func TestResolveWebSearchKagiKey(t *testing.T) {
	t.Parallel()

	t.Run("returns resolved key without mutating config", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Tools: Tools{WebSearch: ToolWebSearch{KagiAPIKey: "$(secret)"}}}
		got, err := resolveWebSearchKagiKey(cfg, &fakeResolver{})
		require.NoError(t, err)
		require.Equal(t, "resolved-$(secret)", got)
		require.Equal(t, "$(secret)", cfg.Tools.WebSearch.KagiAPIKey)
	})

	t.Run("noop without key and non-kagi provider", func(t *testing.T) {
		t.Parallel()
		r := &fakeResolver{}
		got, err := resolveWebSearchKagiKey(&Config{}, r)
		require.NoError(t, err)
		require.Empty(t, got)
		require.Equal(t, 0, r.calls)
	})

	t.Run("errors when kagi selected without key", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Tools: Tools{WebSearch: ToolWebSearch{Provider: "kagi"}}}
		_, err := resolveWebSearchKagiKey(cfg, &fakeResolver{})
		require.Error(t, err)
	})

	t.Run("propagates resolver errors", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Tools: Tools{WebSearch: ToolWebSearch{KagiAPIKey: "$(secret)"}}}
		_, err := resolveWebSearchKagiKey(cfg, &fakeResolver{err: errors.New("boom")})
		require.Error(t, err)
	})
}

func TestConfigStoreResolveWebSearchKagiKeyCaches(t *testing.T) {
	t.Parallel()

	cfg := &Config{Tools: Tools{WebSearch: ToolWebSearch{KagiAPIKey: "$(secret)"}}}
	store := NewTestStore(cfg)
	r := &fakeResolver{}
	store.resolver = r

	for range 3 {
		got, err := store.ResolveWebSearchKagiKey()
		require.NoError(t, err)
		require.Equal(t, "resolved-$(secret)", got)
	}
	require.Equal(t, 1, r.calls, "expansion should run once while the template is unchanged")

	// Changing the raw template must trigger re-resolution.
	changed := &Config{Tools: Tools{WebSearch: ToolWebSearch{KagiAPIKey: "$(other)"}}}
	store.setConfig(changed)
	got, err := store.ResolveWebSearchKagiKey()
	require.NoError(t, err)
	require.Equal(t, "resolved-$(other)", got)
	require.Equal(t, 2, r.calls)

	// Invalidation (used on reload) must force a fresh expansion.
	store.invalidateWebSearchKagiKey()
	_, err = store.ResolveWebSearchKagiKey()
	require.NoError(t, err)
	require.Equal(t, 3, r.calls)
}

func TestConfigStoreSeedWebSearchKagiKeySkipsExpansion(t *testing.T) {
	t.Parallel()

	cfg := &Config{Tools: Tools{WebSearch: ToolWebSearch{KagiAPIKey: "$(secret)"}}}
	store := NewTestStore(cfg)
	r := &fakeResolver{}
	store.resolver = r

	store.seedWebSearchKagiKey("$(secret)", "seeded")
	got, err := store.ResolveWebSearchKagiKey()
	require.NoError(t, err)
	require.Equal(t, "seeded", got)
	require.Equal(t, 0, r.calls)
}
