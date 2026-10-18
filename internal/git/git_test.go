package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// newLinkedWorktree creates a repository with one commit and a linked
// worktree of it. It returns the main checkout and linked worktree paths,
// with symlinks resolved so comparisons match git's output.
func newLinkedWorktree(t *testing.T) (string, string) {
	t.Helper()
	main := t.TempDir()
	runGit(t, main, "init", "-q")
	runGit(t, main, "-c", "user.name=test", "-c", "user.email=test", "commit", "-q", "--allow-empty", "-m", "init")

	worktree := filepath.Join(t.TempDir(), "linked")
	runGit(t, main, "worktree", "add", "-q", worktree)

	main, err := filepath.EvalSymlinks(main)
	require.NoError(t, err)
	worktree, err = filepath.EvalSymlinks(worktree)
	require.NoError(t, err)
	return main, worktree
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func TestWorktreeRoot(t *testing.T) {
	t.Parallel()

	main, worktree := newLinkedWorktree(t)
	require.Equal(t, main, WorktreeRoot(main))
	require.Equal(t, worktree, WorktreeRoot(worktree))
	require.Empty(t, WorktreeRoot(t.TempDir()))
}

func TestMainWorktreeRoot(t *testing.T) {
	t.Parallel()

	main, worktree := newLinkedWorktree(t)
	require.Equal(t, main, MainWorktreeRoot(worktree))
	require.Equal(t, main, MainWorktreeRoot(main))
	require.Empty(t, MainWorktreeRoot(t.TempDir()))
}

func TestLinkedWorktreeCommonDir(t *testing.T) {
	t.Parallel()

	main, worktree := newLinkedWorktree(t)
	mainGitDir := filepath.Join(main, ".git")

	require.Equal(t, mainGitDir, LinkedWorktreeCommonDir(worktree))
	require.Empty(t, LinkedWorktreeCommonDir(main))
	require.Empty(t, LinkedWorktreeCommonDir(t.TempDir()))
}

func TestRepositoryStateQueries(t *testing.T) {
	t.Parallel()

	main, _ := newLinkedWorktree(t)
	require.NotEmpty(t, CurrentBranch(main))
	require.Empty(t, CurrentBranch(t.TempDir()))

	status, ok := StatusShort(main, 20)
	require.True(t, ok)
	require.Empty(t, status)

	require.NoError(t, os.WriteFile(filepath.Join(main, "a.txt"), []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(main, "b.txt"), []byte("b"), 0o644))
	status, ok = StatusShort(main, 1)
	require.True(t, ok)
	require.Equal(t, "?? a.txt", status)

	_, ok = StatusShort(t.TempDir(), 20)
	require.False(t, ok)

	commits, ok := RecentCommits(main, 3)
	require.True(t, ok)
	require.Contains(t, commits, "init")
	_, ok = RecentCommits(t.TempDir(), 3)
	require.False(t, ok)
}
