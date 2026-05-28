package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/filetracker"
	"github.com/charmbracelet/crush/internal/hashline"
	"github.com/stretchr/testify/require"
)

// mockFiletracker implements filetracker.Service for tests.
type mockFiletracker struct {
	reads map[string]time.Time
}

var _ filetracker.Service = (*mockFiletracker)(nil)

func newMockFiletracker() *mockFiletracker {
	return &mockFiletracker{reads: make(map[string]time.Time)}
}

func (m *mockFiletracker) RecordRead(_ context.Context, _, path string) {
	m.reads[path] = time.Now()
}

func (m *mockFiletracker) LastReadTime(_ context.Context, _, path string) time.Time {
	return m.reads[path]
}

func (m *mockFiletracker) ListReadFiles(_ context.Context, _ string) ([]string, error) {
	var paths []string
	for p := range m.reads {
		paths = append(paths, p)
	}
	return paths, nil
}

// ref builds a "LINE#HASH" reference for a given line in the file.
func ref(lines []string, lineNum int) string {
	return fmt.Sprintf("%d#%s", lineNum, hashline.ComputeHash(lines[lineNum-1]))
}

// testCtx returns a context with a session ID set.
func testCtx() context.Context {
	return context.WithValue(context.Background(), SessionIDContextKey, "test-session")
}

// setupTestFile writes content, records a read in the tracker, and
// returns the absolute path.
func setupTestFile(t *testing.T, dir, content string, ft *mockFiletracker) string {
	t.Helper()
	filePath := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
	ft.RecordRead(testCtx(), "test-session", filePath)
	time.Sleep(10 * time.Millisecond)
	return filePath
}

// callEdit is a helper that invokes hashlineEditFile through the real
// codepath with mock deps.
func callEdit(t *testing.T, filePath, workingDir string, ft *mockFiletracker, edits []HashlineOp) (fantasy.ToolResponse, error) {
	t.Helper()
	perms := &mockPermissionService{}
	hist := &mockHistoryService{}
	call := fantasy.ToolCall{ID: "test-call"}
	return hashlineEditFile(testCtx(), perms, hist, ft, nil, filePath, workingDir, edits, call)
}

// callCreate is a helper that invokes hashlineCreateFile.
func callCreate(t *testing.T, filePath, workingDir string, ft *mockFiletracker, edits []HashlineOp) (fantasy.ToolResponse, error) {
	t.Helper()
	perms := &mockPermissionService{}
	hist := &mockHistoryService{}
	call := fantasy.ToolCall{ID: "test-call"}
	return hashlineCreateFile(testCtx(), perms, hist, ft, nil, filePath, workingDir, edits, call)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

// fileLines splits a file's content into lines (trimming trailing
// newline) for building refs.
func fileLines(content string) []string {
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
}

// --- Unit tests for helpers ------------------------------------------------

func TestParseLines(t *testing.T) {
	t.Parallel()

	t.Run("null", func(t *testing.T) {
		t.Parallel()
		result, err := parseLines(json.RawMessage("null"))
		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("empty raw", func(t *testing.T) {
		t.Parallel()
		result, err := parseLines(nil)
		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("string array", func(t *testing.T) {
		t.Parallel()
		result, err := parseLines(json.RawMessage(`["line1", "line2"]`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, []string{"line1", "line2"}, *result)
	})

	t.Run("single string", func(t *testing.T) {
		t.Parallel()
		result, err := parseLines(json.RawMessage(`"hello\nworld"`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, []string{"hello", "world"}, *result)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseLines(json.RawMessage(`42`))
		require.Error(t, err)
	})
}

func TestStripSingleHashlinePrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no prefix", "func main() {", "func main() {"},
		{"with prefix", "10#a4f| func main() {", "func main() {"},
		{"just number", "42", "42"},
		{"empty", "", ""},
		{"no hash", "10| func main() {", "10| func main() {"},
		{"hash too long", "10#abcde| func main() {", "10#abcde| func main() {"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, stripSingleHashlinePrefix(tt.input))
		})
	}
}

func TestStripHashlinePrefixes(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, stripHashlinePrefixes(nil))
	})

	t.Run("majority prefixed strips all", func(t *testing.T) {
		t.Parallel()
		lines := []string{"10#a4f| hello", "", "20#bcd| world"}
		result := stripHashlinePrefixes(&lines)
		require.NotNil(t, result)
		require.Equal(t, []string{"hello", "", "world"}, *result)
	})

	t.Run("minority prefixed preserves all", func(t *testing.T) {
		t.Parallel()
		lines := []string{"10#a4f| hello", "plain line", "another plain line"}
		result := stripHashlinePrefixes(&lines)
		require.NotNil(t, result)
		require.Equal(t, []string{"10#a4f| hello", "plain line", "another plain line"}, *result)
	})

	t.Run("no prefixes preserves all", func(t *testing.T) {
		t.Parallel()
		lines := []string{"42#abc| some markdown anchor", "normal text", "more normal text"}
		result := stripHashlinePrefixes(&lines)
		require.NotNil(t, result)
		require.Equal(t, []string{"42#abc| some markdown anchor", "normal text", "more normal text"}, *result)
	})

	t.Run("all empty lines preserves", func(t *testing.T) {
		t.Parallel()
		lines := []string{"", "", ""}
		result := stripHashlinePrefixes(&lines)
		require.NotNil(t, result)
		require.Equal(t, []string{"", "", ""}, *result)
	})

	t.Run("single prefixed line strips", func(t *testing.T) {
		t.Parallel()
		lines := []string{"5#abc| content"}
		result := stripHashlinePrefixes(&lines)
		require.NotNil(t, result)
		require.Equal(t, []string{"content"}, *result)
	})

	t.Run("exactly half prefixed strips", func(t *testing.T) {
		t.Parallel()
		lines := []string{"10#a4f| hello", "plain line"}
		result := stripHashlinePrefixes(&lines)
		require.NotNil(t, result)
		require.Equal(t, []string{"hello", "plain line"}, *result)
	})
}

func TestCheckOverlaps(t *testing.T) {
	t.Parallel()

	t.Run("no overlap", func(t *testing.T) {
		t.Parallel()
		edits := []resolvedEdit{
			{op: "replace", startLine: 1, endLine: 3, origIdx: 0},
			{op: "replace", startLine: 5, endLine: 7, origIdx: 1},
		}
		require.NoError(t, checkOverlaps(edits))
	})

	t.Run("overlap", func(t *testing.T) {
		t.Parallel()
		edits := []resolvedEdit{
			{op: "replace", startLine: 1, endLine: 5, origIdx: 0},
			{op: "replace", startLine: 3, endLine: 7, origIdx: 1},
		}
		err := checkOverlaps(edits)
		require.Error(t, err)
		require.Contains(t, err.Error(), "overlapping")
	})

	t.Run("adjacent is ok", func(t *testing.T) {
		t.Parallel()
		edits := []resolvedEdit{
			{op: "replace", startLine: 1, endLine: 3, origIdx: 0},
			{op: "replace", startLine: 4, endLine: 6, origIdx: 1},
		}
		require.NoError(t, checkOverlaps(edits))
	})

	t.Run("append ignored", func(t *testing.T) {
		t.Parallel()
		edits := []resolvedEdit{
			{op: "append", startLine: 5, origIdx: 0},
			{op: "replace", startLine: 5, endLine: 5, origIdx: 1},
		}
		require.NoError(t, checkOverlaps(edits))
	})
}

// --- Integration tests calling hashlineEditFile directly -------------------

func TestHashlineEditIntegration(t *testing.T) {
	t.Parallel()

	t.Run("single line replace", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\nline3\n"
		filePath := setupTestFile(t, dir, content, ft)
		lines := fileLines(content)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: ref(lines, 2), Lines: json.RawMessage(`["REPLACED"]`)},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)

		got := readFile(t, filePath)
		require.Equal(t, "line1\nREPLACED\nline3\n", got)
	})

	t.Run("range replace", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\nline3\nline4\nline5\n"
		filePath := setupTestFile(t, dir, content, ft)
		lines := fileLines(content)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: ref(lines, 2), End: ref(lines, 4), Lines: json.RawMessage(`["REPLACED_RANGE"]`)},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)

		got := readFile(t, filePath)
		require.Equal(t, "line1\nREPLACED_RANGE\nline5\n", got)
	})

	t.Run("delete lines via null", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\nline3\n"
		filePath := setupTestFile(t, dir, content, ft)
		lines := fileLines(content)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: ref(lines, 2), Lines: json.RawMessage(`null`)},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)

		got := readFile(t, filePath)
		require.Equal(t, "line1\nline3\n", got)
	})

	t.Run("append with anchor", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\nline3\n"
		filePath := setupTestFile(t, dir, content, ft)
		lines := fileLines(content)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "append", Pos: ref(lines, 2), Lines: json.RawMessage(`["INSERTED"]`)},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)

		got := readFile(t, filePath)
		require.Equal(t, "line1\nline2\nINSERTED\nline3\n", got)
	})

	t.Run("append without anchor", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\n"
		filePath := setupTestFile(t, dir, content, ft)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "append", Lines: json.RawMessage(`["APPENDED"]`)},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)

		got := readFile(t, filePath)
		require.Equal(t, "line1\nline2\nAPPENDED\n", got)
	})

	t.Run("prepend with anchor", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\nline3\n"
		filePath := setupTestFile(t, dir, content, ft)
		lines := fileLines(content)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "prepend", Pos: ref(lines, 2), Lines: json.RawMessage(`["PREPENDED"]`)},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)

		got := readFile(t, filePath)
		require.Equal(t, "line1\nPREPENDED\nline2\nline3\n", got)
	})

	t.Run("prepend without anchor", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\n"
		filePath := setupTestFile(t, dir, content, ft)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "prepend", Lines: json.RawMessage(`["FIRST"]`)},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)

		got := readFile(t, filePath)
		require.Equal(t, "FIRST\nline1\nline2\n", got)
	})

	t.Run("file creation via hashlineCreateFile", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		filePath := filepath.Join(dir, "new.go")

		resp, err := callCreate(t, filePath, dir, ft, []HashlineOp{
			{Op: "append", Lines: json.RawMessage(`["package main", "", "func main() {}"]`)},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)

		got := readFile(t, filePath)
		require.Equal(t, "package main\n\nfunc main() {}\n", got)
	})

	t.Run("file creation rejects pos refs", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		filePath := filepath.Join(dir, "new.go")

		resp, err := callCreate(t, filePath, dir, ft, []HashlineOp{
			{Op: "append", Pos: "1#abc", Lines: json.RawMessage(`["hello"]`)},
		})
		require.NoError(t, err)
		require.True(t, resp.IsError)
		require.Contains(t, resp.Content, "cannot use pos/end")
	})
}

// --- Error path tests calling hashlineEditFile directly --------------------

func TestHashlineEditErrors(t *testing.T) {
	t.Parallel()

	t.Run("hash mismatch", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\nline3\n"
		filePath := setupTestFile(t, dir, content, ft)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: "2#fff", Lines: json.RawMessage(`["REPLACED"]`)},
		})
		require.NoError(t, err)
		require.True(t, resp.IsError)
		require.Contains(t, resp.Content, "Hash mismatch")
	})

	t.Run("no-op returns error", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\n"
		filePath := setupTestFile(t, dir, content, ft)
		lines := fileLines(content)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: ref(lines, 1), Lines: json.RawMessage(`["line1"]`)},
		})
		require.NoError(t, err)
		require.True(t, resp.IsError)
		require.Contains(t, resp.Content, "no changes")
	})

	t.Run("overlapping ranges rejected", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "a\nb\nc\nd\ne\n"
		filePath := setupTestFile(t, dir, content, ft)
		lines := fileLines(content)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: ref(lines, 1), End: ref(lines, 3), Lines: json.RawMessage(`["X"]`)},
			{Op: "replace", Pos: ref(lines, 2), End: ref(lines, 4), Lines: json.RawMessage(`["Y"]`)},
		})
		require.NoError(t, err)
		require.True(t, resp.IsError)
		require.Contains(t, resp.Content, "overlapping")
	})

	t.Run("out of bounds", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\n"
		filePath := setupTestFile(t, dir, content, ft)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: "99#abc", Lines: json.RawMessage(`["X"]`)},
		})
		require.NoError(t, err)
		require.True(t, resp.IsError)
		require.Contains(t, resp.Content, "out of bounds")
	})

	t.Run("unknown op", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\n"
		filePath := setupTestFile(t, dir, content, ft)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "bogus", Lines: json.RawMessage(`["X"]`)},
		})
		require.NoError(t, err)
		require.True(t, resp.IsError)
		require.Contains(t, resp.Content, "unknown op")
	})

	t.Run("must read before edit", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		filePath := filepath.Join(dir, "test.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("hello\n"), 0o644))

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: "1#abc", Lines: json.RawMessage(`["X"]`)},
		})
		require.NoError(t, err)
		require.True(t, resp.IsError)
		require.Contains(t, resp.Content, "must read the file")
	})

	t.Run("malformed ref", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\n"
		filePath := setupTestFile(t, dir, content, ft)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: "badref", Lines: json.RawMessage(`["X"]`)},
		})
		require.NoError(t, err)
		require.True(t, resp.IsError)
		require.Contains(t, resp.Content, "invalid pos")
	})

	t.Run("end before start", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "a\nb\nc\nd\n"
		filePath := setupTestFile(t, dir, content, ft)
		lines := fileLines(content)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: ref(lines, 3), End: ref(lines, 1), Lines: json.RawMessage(`["X"]`)},
		})
		require.NoError(t, err)
		require.True(t, resp.IsError)
		require.Contains(t, resp.Content, "before start")
	})
}

// --- Bottom-up ordering and multi-edit tests -------------------------------

func TestHashlineEditBottomUpOrdering(t *testing.T) {
	t.Parallel()

	ft := newMockFiletracker()
	dir := t.TempDir()
	content := "a\nb\nc\nd\ne\n"
	filePath := setupTestFile(t, dir, content, ft)
	lines := fileLines(content)

	resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
		{Op: "replace", Pos: ref(lines, 2), Lines: json.RawMessage(`["B"]`)},
		{Op: "replace", Pos: ref(lines, 4), Lines: json.RawMessage(`["D"]`)},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, resp.Content)

	got := readFile(t, filePath)
	require.Equal(t, "a\nB\nc\nD\ne\n", got)
}

func TestHashlineEditAdjacentLines(t *testing.T) {
	t.Parallel()

	ft := newMockFiletracker()
	dir := t.TempDir()
	content := "line1\nline2\nline3\nline4\n"
	filePath := setupTestFile(t, dir, content, ft)
	lines := fileLines(content)

	resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
		{Op: "replace", Pos: ref(lines, 2), Lines: json.RawMessage(`["REPLACED2"]`)},
		{Op: "replace", Pos: ref(lines, 3), Lines: json.RawMessage(`["REPLACED3"]`)},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, resp.Content)

	got := readFile(t, filePath)
	require.Equal(t, "line1\nREPLACED2\nREPLACED3\nline4\n", got)
}

func TestHashlineEditCRLF(t *testing.T) {
	t.Parallel()

	ft := newMockFiletracker()
	dir := t.TempDir()
	content := "line1\r\nline2\r\nline3\r\n"
	filePath := setupTestFile(t, dir, content, ft)

	unixLines := fileLines(strings.ReplaceAll(content, "\r\n", "\n"))

	resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
		{Op: "replace", Pos: ref(unixLines, 2), Lines: json.RawMessage(`["REPLACED"]`)},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, resp.Content)

	got := readFile(t, filePath)
	require.Equal(t, "line1\r\nREPLACED\r\nline3\r\n", got)
}

func TestHashlineEditEmptyFile(t *testing.T) {
	t.Parallel()

	ft := newMockFiletracker()
	dir := t.TempDir()
	filePath := setupTestFile(t, dir, "", ft)

	resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
		{Op: "append", Lines: json.RawMessage(`["first line"]`)},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, resp.Content)

	got := readFile(t, filePath)
	require.Equal(t, "first line\n", got)
}

func TestHashlineEditPrefixStripping(t *testing.T) {
	t.Parallel()

	t.Run("strips when all lines prefixed", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\n"
		filePath := setupTestFile(t, dir, content, ft)
		lines := fileLines(content)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: ref(lines, 1), Lines: json.RawMessage(`["1#abc| REPLACED"]`)},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)

		got := readFile(t, filePath)
		require.Equal(t, "REPLACED\nline2\n", got)
	})

	t.Run("preserves legitimate content when minority prefixed", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		content := "line1\nline2\nline3\n"
		filePath := setupTestFile(t, dir, content, ft)
		lines := fileLines(content)

		resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
			{Op: "replace", Pos: ref(lines, 1), End: ref(lines, 3),
				Lines: json.RawMessage(`["42#abc| markdown anchor", "normal text", "more normal text"]`)},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)

		got := readFile(t, filePath)
		require.Equal(t, "42#abc| markdown anchor\nnormal text\nmore normal text\n", got)
	})
}

func TestHashlineDeleteFile(t *testing.T) {
	t.Parallel()

	t.Run("deletes existing file", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		filePath := filepath.Join(dir, "to_delete.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("content\n"), 0o644))

		perms := &mockPermissionService{}
		hist := &mockHistoryService{}
		call := fantasy.ToolCall{ID: "test-call"}
		resp, err := hashlineDeleteFile(testCtx(), perms, hist, ft, filePath, dir, call)
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)
		require.Contains(t, resp.Content, "File deleted")

		_, statErr := os.Stat(filePath)
		require.True(t, os.IsNotExist(statErr))
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		filePath := filepath.Join(dir, "ghost.txt")

		perms := &mockPermissionService{}
		hist := &mockHistoryService{}
		call := fantasy.ToolCall{ID: "test-call"}
		resp, err := hashlineDeleteFile(testCtx(), perms, hist, ft, filePath, dir, call)
		require.NoError(t, err)
		require.True(t, resp.IsError)
		require.Contains(t, resp.Content, "file not found")
	})
}

func TestHashlineMoveFile(t *testing.T) {
	t.Parallel()

	t.Run("moves file and creates parent dirs", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		oldPath := filepath.Join(dir, "original.txt")
		require.NoError(t, os.WriteFile(oldPath, []byte("hello\n"), 0o644))

		newRelPath := "sub/dir/moved.txt"
		perms := &mockPermissionService{}
		hist := &mockHistoryService{}
		call := fantasy.ToolCall{ID: "test-call"}
		resp, err := hashlineMoveFile(testCtx(), perms, hist, ft, oldPath, newRelPath, dir, call)
		require.NoError(t, err)
		require.False(t, resp.IsError, resp.Content)
		require.Contains(t, resp.Content, "File moved")

		_, statErr := os.Stat(oldPath)
		require.True(t, os.IsNotExist(statErr))

		newPath := filepath.Join(dir, newRelPath)
		got := readFile(t, newPath)
		require.Equal(t, "hello\n", got)
	})

	t.Run("non-existent source returns error", func(t *testing.T) {
		t.Parallel()
		ft := newMockFiletracker()
		dir := t.TempDir()
		oldPath := filepath.Join(dir, "ghost.txt")

		perms := &mockPermissionService{}
		hist := &mockHistoryService{}
		call := fantasy.ToolCall{ID: "test-call"}
		resp, err := hashlineMoveFile(testCtx(), perms, hist, ft, oldPath, "new.txt", dir, call)
		require.NoError(t, err)
		require.True(t, resp.IsError)
		require.Contains(t, resp.Content, "file not found")
	})
}

func TestHashlineEditStaleFile(t *testing.T) {
	t.Parallel()

	ft := newMockFiletracker()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("original\n"), 0o644))

	// Record read, then modify file after the read.
	ft.RecordRead(testCtx(), "test-session", filePath)
	time.Sleep(50 * time.Millisecond)

	// Touch the file with a new mod time.
	now := time.Now().Add(2 * time.Second)
	require.NoError(t, os.Chtimes(filePath, now, now))

	resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
		{Op: "replace", Pos: "1#abc", Lines: json.RawMessage(`["changed"]`)},
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	require.Contains(t, resp.Content, "has been modified since")
}

func TestHashlineEditEndOnNonReplace(t *testing.T) {
	t.Parallel()

	ft := newMockFiletracker()
	dir := t.TempDir()
	content := "line1\nline2\nline3\n"
	filePath := setupTestFile(t, dir, content, ft)
	lines := fileLines(content)

	resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
		{Op: "append", Pos: ref(lines, 1), End: ref(lines, 2), Lines: json.RawMessage(`["X"]`)},
	})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	require.Contains(t, resp.Content, "end is only valid for replace")
}

func TestHashlineEditMixedOps(t *testing.T) {
	t.Parallel()

	ft := newMockFiletracker()
	dir := t.TempDir()
	content := "a\nb\nc\nd\ne\n"
	filePath := setupTestFile(t, dir, content, ft)
	lines := fileLines(content)

	resp, err := callEdit(t, filePath, dir, ft, []HashlineOp{
		{Op: "replace", Pos: ref(lines, 3), Lines: json.RawMessage(`["C"]`)},
		{Op: "append", Pos: ref(lines, 5), Lines: json.RawMessage(`["AFTER_E"]`)},
		{Op: "prepend", Pos: ref(lines, 1), Lines: json.RawMessage(`["BEFORE_A"]`)},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, resp.Content)

	got := readFile(t, filePath)
	require.Equal(t, "BEFORE_A\na\nb\nC\nd\ne\nAFTER_E\n", got)
}

func TestStripSingleHashlinePrefixBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hash len 1", "5#a| content", "content"},
		{"hash len 2", "5#ab| content", "content"},
		{"hash len 3", "5#abc| content", "content"},
		{"hash len 4", "5#abcd| content", "content"},
		{"hash len 5 rejected", "5#abcde| content", "5#abcde| content"},
		{"hash len 0 rejected", "5#| content", "5#| content"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, stripSingleHashlinePrefix(tt.input))
		})
	}
}
