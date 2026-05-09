package shell

import (
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
