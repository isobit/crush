// Package git is the single place Crush asks git about repository layout.
// Other packages should use these helpers instead of invoking git directly,
// so worktree handling stays consistent across config loading and sandboxing.
package git

import (
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// commandTimeout bounds each git invocation so a hung repository cannot
// stall startup.
const commandTimeout = 5 * time.Second

// worktreeRootCache memoizes WorktreeRoot per directory. The root is stable
// for the life of the process, so config reloads should not re-run git.
// Keys are directories; values are the resolved root ("" when not in a
// worktree).
var worktreeRootCache sync.Map // map[string]string

// WorktreeRoot returns the absolute path of the working tree root containing
// dir, or "" when dir is not inside a working tree (bare repositories, plain
// directories, or a missing git binary). Linked worktrees report their own
// root, not the main checkout's.
func WorktreeRoot(dir string) string {
	if cached, ok := worktreeRootCache.Load(dir); ok {
		return cached.(string)
	}
	root := ""
	if out, ok := run(dir, "rev-parse", "--show-toplevel"); ok && out != "" {
		root = absPath(out)
	}
	worktreeRootCache.Store(dir, root)
	return root
}

// MainWorktreeRoot returns the primary working tree path for the repository
// containing dir, or "" when it cannot be determined.
func MainWorktreeRoot(dir string) string {
	out, ok := run(dir, "worktree", "list", "--porcelain")
	if !ok {
		return ""
	}

	// Git lists the main worktree first. A bare entry clears the candidate,
	// so the first non-bare entry after it is used instead.
	var root string
	for line := range strings.SplitSeq(out, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			root = strings.TrimPrefix(line, "worktree ")
		case line == "bare":
			root = ""
		case line == "" && root != "":
			return absPath(root)
		}
	}
	if root == "" {
		return ""
	}
	return absPath(root)
}

// LinkedWorktreeCommonDir returns the repository's shared git directory when
// dir is a linked worktree, and "" for main checkouts and non-repositories.
// Git keeps refs and worktree metadata there, so commands run from a linked
// worktree must be able to write to it.
func LinkedWorktreeCommonDir(dir string) string {
	out, ok := run(dir, "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir")
	if !ok {
		return ""
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 2 || lines[0] == lines[1] {
		return ""
	}
	return filepath.Clean(lines[1])
}

// run executes git with args inside dir and returns its trimmed stdout. The
// bool reports whether git exited successfully; callers decide whether empty
// output is meaningful.
func run(dir string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	trimmed := strings.TrimSpace(string(out))
	return trimmed, true
}

// absPath returns the absolute form of p, or "" if it cannot be resolved.
func absPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return ""
	}
	return abs
}

// CurrentBranch returns the branch checked out in dir. It returns "" on a
// detached HEAD or outside a repository.
func CurrentBranch(dir string) string {
	out, _ := run(dir, "branch", "--show-current")
	return out
}

// StatusShort returns the first limit lines of `git status --short` for dir.
// The bool is false when git fails, including when dir is not a repository.
// An empty string with true means the working tree is clean.
func StatusShort(dir string, limit int) (string, bool) {
	out, ok := run(dir, "status", "--short")
	if !ok {
		return "", false
	}
	lines := strings.Split(out, "\n")
	if len(lines) > limit {
		lines = lines[:limit]
	}
	return strings.Join(lines, "\n"), true
}

// RecentCommits returns the n most recent commits in dir as one-line
// summaries. The bool is false when git fails.
func RecentCommits(dir string, n int) (string, bool) {
	return run(dir, "log", "--oneline", "-n", strconv.Itoa(n))
}
