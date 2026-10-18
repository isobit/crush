package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/markdiff"
	"github.com/stretchr/testify/require"
)

// globalConfigJSON is the global crush.json the test writes. It is shown in
// the markdiff document so reviewers can see which settings produced the table.
const globalConfigJSON = `{
  "options": {
    "sandbox": {
      "mode": "auto",
      "writable_paths": ["/srv/shared-cache"]
    }
  }
}
`

// TestSandboxEnvironments records the git-derived config and the bash sandbox
// options that Crush produces for several working directories.
func TestSandboxEnvironments(t *testing.T) {
	// Point global config at a temp dir so the developer's own crush.json
	// does not change the recorded output.
	globalDir := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", globalDir)
	t.Setenv("CRUSH_GLOBAL_DATA", t.TempDir())
	require.NoError(t, os.WriteFile(
		filepath.Join(globalDir, "crush.json"),
		[]byte(globalConfigJSON),
		0o644,
	))

	base, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	plainDir := filepath.Join(base, "plain")
	mainDir := filepath.Join(base, "main")
	linkedDir := filepath.Join(base, "linked")
	require.NoError(t, os.Mkdir(plainDir, 0o755))
	require.NoError(t, os.Mkdir(mainDir, 0o755))
	runGitForEnvironmentTest(t, mainDir, "init", "-q")
	runGitForEnvironmentTest(t, mainDir, "-c", "user.name=test", "-c", "user.email=test", "commit", "-q", "--allow-empty", "-m", "init")
	runGitForEnvironmentTest(t, mainDir, "worktree", "add", "-q", linkedDir)

	environments := []struct {
		name string
		dir  string
	}{
		{"Plain directory", plainDir},
		{"Main checkout", mainDir},
		{"Linked worktree", linkedDir},
	}

	// show renders empty values as "-" and hides the temp root so the
	// document is identical across runs.
	show := func(s string) string {
		if s == "" {
			return "-"
		}
		return strings.ReplaceAll(s, base, "<tmp>")
	}

	doc := markdiff.New(t, "Sandbox environments",
		"Crush derives repository layout from the working directory and uses it "+
			"for the data directory and for bash sandbox writable paths. This "+
			"document shows those values for a plain directory, a main checkout, "+
			"and a linked worktree. Paths under the temporary test root appear as <tmp>.")
	configSec := doc.Section("Global config",
		"Every environment below loads this global crush.json, which lives in the "+
			"test's global config directory. Its sandbox settings apply to all three "+
			"working directories.")
	configSec.Code("json", globalConfigJSON)

	sec := doc.Section("Config and bash options",
		"Each row is one working directory. The git columns come from the loaded "+
			"config, the data directory is where session state lives, and the "+
			"writable paths are what sandboxed bash commands may write to. In the "+
			"linked worktree, the main checkout's git directory is added so git "+
			"commands can update refs and worktree metadata.")

	var rows [][]string
	for _, env := range environments {
		store, err := config.Load(env.dir, "", false)
		require.NoError(t, err)
		cfg := store.Config()
		opts := buildBashSandboxOptions(cfg)

		writable := make([]string, 0, len(opts.WritablePaths))
		for _, p := range opts.WritablePaths {
			writable = append(writable, show(p))
		}
		rows = append(rows, []string{
			env.name,
			show(cfg.Git.WorktreeRoot),
			show(cfg.Git.MainWorktreeRoot),
			show(cfg.Git.LinkedWorktreeCommonDir),
			show(cfg.Options.DataDirectory),
			string(opts.Mode),
			show(strings.Join(writable, ", ")),
		})
	}
	sec.Table([]string{
		"Environment",
		"Worktree root",
		"Main checkout",
		"Linked common dir",
		"Data directory",
		"Sandbox mode",
		"Writable paths",
	}, rows...)
}

func runGitForEnvironmentTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
