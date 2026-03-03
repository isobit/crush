package hashline

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"spaces only", "   ", ""},
		{"simple", "Hello World", "hello world"},
		{"tabs", "Hello\tWorld", "hello world"},
		{"multiple spaces", "Hello    World", "hello world"},
		{"leading trailing", "  Hello World  ", "hello world"},
		{"mixed whitespace", " \t Hello  \t World \t ", "hello world"},
		{"unicode", "Héllo Wörld", "héllo wörld"},
		{"CRLF", "Hello\r\n", "hello"},
		{"BOM", "\uFEFFHello", "hello"},
		{"tabs and spaces", "\t  func main() {  \t", "func main() {"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, NormalizeLine(tt.input))
		})
	}
}

func TestComputeHash(t *testing.T) {
	t.Parallel()

	h1 := ComputeHash("func main() {")
	require.Len(t, h1, 3, "hash should be 3 hex chars")

	require.Equal(t, h1, ComputeHash("func main() {"), "deterministic")
	require.Equal(t, h1, ComputeHash("  func main() {  "), "whitespace insensitive")
	require.Equal(t, h1, ComputeHash("FUNC MAIN() {"), "case insensitive")
	require.Equal(t, h1, ComputeHash("\tfunc  main()  {"), "tab/multi-space insensitive")

	h2 := ComputeHash("func foo() {")
	require.NotEqual(t, h1, h2, "different content should (usually) differ")

	require.Len(t, ComputeHash(""), 3, "empty line should still produce a 3-char hash")
}

func TestFormatLine(t *testing.T) {
	t.Parallel()
	result := FormatLine(42, "a4f", "func main() {")
	require.Equal(t, "42#a4f| func main() {", result)
}

func TestFormatLines(t *testing.T) {
	t.Parallel()

	lines := []string{
		"package main",
		"",
		"func main() {",
		"}",
	}

	result := FormatLines(lines, 1)
	outputLines := strings.Split(result, "\n")
	require.Len(t, outputLines, 4)

	for _, line := range outputLines {
		require.Contains(t, line, "#")
		require.Contains(t, line, "| ")
	}

	require.True(t, strings.HasPrefix(outputLines[0], "1#"))
	require.True(t, strings.HasPrefix(outputLines[3], "4#"))

	require.Empty(t, FormatLines(nil, 1))
	require.Empty(t, FormatLines([]string{}, 1))
}

func TestFormatLinesWithOffset(t *testing.T) {
	t.Parallel()
	lines := []string{"line a", "line b"}
	result := FormatLines(lines, 10)
	outputLines := strings.Split(result, "\n")
	require.True(t, strings.HasPrefix(outputLines[0], "10#"))
	require.True(t, strings.HasPrefix(outputLines[1], "11#"))
}

func TestFormatLinesCRLF(t *testing.T) {
	t.Parallel()
	lines := []string{"hello\r", "world\r"}
	result := FormatLines(lines, 1)
	require.NotContains(t, result, "\r")
}

func TestParseRef(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ref     string
		line    int
		hash    string
		wantErr bool
	}{
		{"valid", "23#a4f", 23, "a4f", false},
		{"line 1", "1#000", 1, "000", false},
		{"large line", "99999#fff", 99999, "fff", false},
		{"no hash separator", "23", 0, "", true},
		{"empty hash", "23#", 0, "", true},
		{"non-numeric line", "abc#a4f", 0, "", true},
		{"empty string", "", 0, "", true},
		{"negative line", "-1#a4f", -1, "a4f", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			line, hash, err := ParseRef(tt.ref)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.line, line)
			require.Equal(t, tt.hash, hash)
		})
	}
}

func TestHashFile(t *testing.T) {
	t.Parallel()

	lines := []string{"package main", "", "func main() {", "}"}
	refs := HashFile(lines)
	require.Len(t, refs, 4)

	for i, ref := range refs {
		require.Equal(t, i+1, ref.Num)
		require.Len(t, ref.Hash, 3)
		require.Equal(t, lines[i], ref.Content)
	}

	require.Empty(t, HashFile(nil))
}

func TestValidateRefs(t *testing.T) {
	t.Parallel()

	lines := []string{"package main", "", "func main() {", "}"}
	refs := HashFile(lines)

	t.Run("all valid", func(t *testing.T) {
		t.Parallel()
		validRefs := make([]string, len(refs))
		for i, ref := range refs {
			validRefs[i] = fmt.Sprintf("%d#%s", ref.Num, ref.Hash)
		}
		mismatches := ValidateRefs(lines, validRefs...)
		require.Empty(t, mismatches)
	})

	t.Run("hash mismatch", func(t *testing.T) {
		t.Parallel()
		mismatches := ValidateRefs(lines, "1#fff")
		require.Len(t, mismatches, 1)
		require.Equal(t, 1, mismatches[0].Line)
		require.Equal(t, "fff", mismatches[0].ExpectedHash)
		require.Equal(t, refs[0].Hash, mismatches[0].ActualHash)
		require.Equal(t, "package main", mismatches[0].CurrentContent)
	})

	t.Run("out of bounds", func(t *testing.T) {
		t.Parallel()
		mismatches := ValidateRefs(lines, "99#abc")
		require.Len(t, mismatches, 1)
		require.Equal(t, 99, mismatches[0].Line)
		require.Empty(t, mismatches[0].ActualHash)
	})

	t.Run("line zero", func(t *testing.T) {
		t.Parallel()
		mismatches := ValidateRefs(lines, "0#abc")
		require.Len(t, mismatches, 1)
	})

	t.Run("malformed ref", func(t *testing.T) {
		t.Parallel()
		mismatches := ValidateRefs(lines, "badref")
		require.Len(t, mismatches, 1)
		require.Equal(t, "badref", mismatches[0].Reference)
	})

	t.Run("empty file", func(t *testing.T) {
		t.Parallel()
		mismatches := ValidateRefs([]string{}, "1#abc")
		require.Len(t, mismatches, 1)
	})

	t.Run("multiple mismatches", func(t *testing.T) {
		t.Parallel()
		mismatches := ValidateRefs(lines, "1#fff", "2#fff", "3#fff")
		require.Len(t, mismatches, 3)
	})

	t.Run("no refs", func(t *testing.T) {
		t.Parallel()
		mismatches := ValidateRefs(lines)
		require.Empty(t, mismatches)
	})
}

func TestCollisionRate(t *testing.T) {
	t.Parallel()

	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = fmt.Sprintf("func example%d(ctx context.Context) error { return nil }", i)
	}

	seen := make(map[string]int)
	for _, line := range lines {
		h := ComputeHash(line)
		seen[h]++
	}

	uniqueHashes := len(seen)
	collisionRate := 1.0 - float64(uniqueHashes)/float64(len(lines))
	require.Less(t, collisionRate, 0.5, "collision rate should be reasonable for 1000 distinct lines in 4096 buckets")
}

func TestSingleLine(t *testing.T) {
	t.Parallel()

	lines := []string{"only line"}
	refs := HashFile(lines)
	require.Len(t, refs, 1)
	require.Equal(t, 1, refs[0].Num)

	result := FormatLines(lines, 1)
	require.Contains(t, result, "1#")
	require.Contains(t, result, "| only line")
}

func TestVeryLongLine(t *testing.T) {
	t.Parallel()

	longLine := strings.Repeat("x", 10000)
	h := ComputeHash(longLine)
	require.Len(t, h, 3)

	lines := []string{longLine}
	result := FormatLines(lines, 1)
	require.True(t, strings.HasPrefix(result, "1#"))
}
