package config

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetDefaultsRecordsGitInfo(t *testing.T) {
	t.Parallel()

	main := t.TempDir()
	runGitForGitInfoTest(t, main, "init", "-q")
	runGitForGitInfoTest(t, main, "-c", "user.name=test", "-c", "user.email=test", "commit", "-q", "--allow-empty", "-m", "init")
	worktree := filepath.Join(t.TempDir(), "linked")
	runGitForGitInfoTest(t, main, "worktree", "add", "-q", worktree)

	main, err := filepath.EvalSymlinks(main)
	require.NoError(t, err)
	worktree, err = filepath.EvalSymlinks(worktree)
	require.NoError(t, err)

	t.Run("linked worktree", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{}
		cfg.setDefaults(worktree, "")
		require.Equal(t, GitInfo{
			WorktreeRoot:            worktree,
			MainWorktreeRoot:        main,
			LinkedWorktreeCommonDir: filepath.Join(main, ".git"),
		}, cfg.Git)
	})

	t.Run("main checkout", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{}
		cfg.setDefaults(main, "")
		require.Equal(t, GitInfo{WorktreeRoot: main, MainWorktreeRoot: main}, cfg.Git)
	})

	t.Run("not a repository", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{}
		cfg.setDefaults(t.TempDir(), "")
		require.Equal(t, GitInfo{}, cfg.Git)
	})
}

func runGitForGitInfoTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
