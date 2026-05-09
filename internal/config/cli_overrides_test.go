package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCLIOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    map[string]string
		wantErr string
	}{
		{
			name: "single key=value",
			args: []string{"debug=true"},
			want: map[string]string{"debug": "true"},
		},
		{
			name: "multiple pairs",
			args: []string{"debug=true", "tui.compact_mode=true"},
			want: map[string]string{"debug": "true", "tui.compact_mode": "true"},
		},
		{
			name: "value with equals sign",
			args: []string{"data_directory=/path/to=dir"},
			want: map[string]string{"data_directory": "/path/to=dir"},
		},
		{
			name:    "missing equals",
			args:    []string{"debug"},
			wantErr: `invalid --set value "debug": must be key=value`,
		},
		{
			name:    "empty key",
			args:    []string{"=value"},
			wantErr: `invalid --set value "=value": key cannot be empty`,
		},
		{
			name: "empty value is valid",
			args: []string{"data_directory="},
			want: map[string]string{"data_directory": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseCLIOverrides(tt.args)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSetFieldByPath(t *testing.T) {
	t.Parallel()

	t.Run("flat string field", func(t *testing.T) {
		t.Parallel()
		opts := &Options{}
		err := SetFieldByPath(opts, "data_directory", "/tmp/test")
		require.NoError(t, err)
		require.Equal(t, "/tmp/test", opts.DataDirectory)
	})

	t.Run("flat bool field", func(t *testing.T) {
		t.Parallel()
		opts := &Options{}
		err := SetFieldByPath(opts, "debug", "true")
		require.NoError(t, err)
		require.True(t, opts.Debug)
	})

	t.Run("pointer bool field", func(t *testing.T) {
		t.Parallel()
		opts := &Options{}
		err := SetFieldByPath(opts, "auto_lsp", "false")
		require.NoError(t, err)
		require.NotNil(t, opts.AutoLSP)
		require.False(t, *opts.AutoLSP)
	})

	t.Run("nested struct field", func(t *testing.T) {
		t.Parallel()
		opts := &Options{TUI: &TUIOptions{}}
		err := SetFieldByPath(opts, "tui.compact_mode", "true")
		require.NoError(t, err)
		require.True(t, opts.TUI.CompactMode)
	})

	t.Run("nested struct allocates nil pointer", func(t *testing.T) {
		t.Parallel()
		opts := &Options{}
		err := SetFieldByPath(opts, "tui.compact_mode", "true")
		require.NoError(t, err)
		require.NotNil(t, opts.TUI)
		require.True(t, opts.TUI.CompactMode)
	})

	t.Run("nested pointer int", func(t *testing.T) {
		t.Parallel()
		opts := &Options{}
		err := SetFieldByPath(opts, "tui.sidebar_width", "40")
		require.NoError(t, err)
		require.NotNil(t, opts.TUI)
		require.NotNil(t, opts.TUI.SidebarWidth)
		require.Equal(t, 40, *opts.TUI.SidebarWidth)
	})

	t.Run("string slice field", func(t *testing.T) {
		t.Parallel()
		opts := &Options{}
		err := SetFieldByPath(opts, "disabled_tools", "bash, sourcegraph")
		require.NoError(t, err)
		require.Equal(t, []string{"bash", "sourcegraph"}, opts.DisabledTools)
	})

	t.Run("unknown field returns error", func(t *testing.T) {
		t.Parallel()
		opts := &Options{}
		err := SetFieldByPath(opts, "nonexistent", "value")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown field")
	})

	t.Run("invalid bool returns error", func(t *testing.T) {
		t.Parallel()
		opts := &Options{}
		err := SetFieldByPath(opts, "debug", "notabool")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid bool")
	})
}

func TestApplyCLIOverrides(t *testing.T) {
	t.Parallel()

	t.Run("sets string option", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Options: &Options{}}
		cfg.setDefaults(t.TempDir(), "")
		store := &ConfigStore{
			config:     cfg,
			workingDir: t.TempDir(),
		}

		err := store.ApplyCLIOverrides(map[string]string{
			"initialize_as": "CRUSH.md",
		})
		require.NoError(t, err)
		require.Equal(t, "CRUSH.md", store.Config().Options.InitializeAs)
	})

	t.Run("sets bool option", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Options: &Options{}}
		cfg.setDefaults(t.TempDir(), "")
		store := &ConfigStore{
			config:     cfg,
			workingDir: t.TempDir(),
		}

		err := store.ApplyCLIOverrides(map[string]string{
			"debug": "true",
		})
		require.NoError(t, err)
		require.True(t, store.Config().Options.Debug)
	})

	t.Run("sets nested option", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Options: &Options{}}
		cfg.setDefaults(t.TempDir(), "")
		store := &ConfigStore{
			config:     cfg,
			workingDir: t.TempDir(),
		}

		err := store.ApplyCLIOverrides(map[string]string{
			"tui.compact_mode": "true",
		})
		require.NoError(t, err)
		require.True(t, store.Config().Options.TUI.CompactMode)
	})

	t.Run("empty overrides is no-op", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{Options: &Options{}}
		cfg.setDefaults(t.TempDir(), "")
		store := &ConfigStore{
			config:     cfg,
			workingDir: t.TempDir(),
		}

		err := store.ApplyCLIOverrides(nil)
		require.NoError(t, err)
	})
}
