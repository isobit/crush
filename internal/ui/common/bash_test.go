package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatBashDisplay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "separates commands and formats loop body",
			source: "thing-a; thing-b; for z in quux; do echo a; done",
			want:   "thing-a;\nthing-b;\nfor z in quux; do\n  echo a;\ndone",
		},
		{
			name:   "preserves semicolons in quotes",
			source: "printf '%s' 'a; b'; echo done",
			want:   "printf '%s' 'a; b';\necho done",
		},
		{
			name:   "returns invalid shell unchanged",
			source: "for z in; do",
			want:   "for z in; do",
		},
		{
			name:   "formats conditional body",
			source: "if test -f file; then echo yes; else echo no; fi",
			want:   "if test -f file; then\n  echo yes;\nelse\n  echo no;\nfi",
		},
		{
			name:   "formats while body",
			source: "while test -f file; do echo ok; done",
			want:   "while test -f file; do\n  echo ok;\ndone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, FormatBashDisplay(tt.source))
		})
	}
}
