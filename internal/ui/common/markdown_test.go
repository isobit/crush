package common

import (
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestMarkdownCodeBlockIndentRepro(t *testing.T) {
	t.Parallel()

	sty := styles.IsobitStyles()
	markdown := "```rust\n" + strings.Join([]string{
		"enum JobMessage {",
		"    Stop,",
		"}",
		"",
		"struct Job {",
		"    label: String, // This worker's private state",
		"}",
		"",
		"fn start_job_worker(label: String) -> Result<WorkerHandle, StartError> {",
		"    // Starts one independent actor/process with its own state.",
		"    spawn_actor(Job { label }, handle_job_message)",
		"}",
		"",
		"fn start_application() -> Result<(), StartError> {",
		"    // The factory knows how to create a worker, but doesn't create one yet.",
		"    let job_factory = FactorySupervisor::new(start_job_worker)",
		"        .restart_workers(Transient); // Restart only if a worker crashes.",
		"}",
	}, "\n") + "\n```"

	rendered, err := MarkdownRenderer(&sty, 80).Render(markdown)
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(rendered)
	privateStateLine := lineContaining(t, plain, "private state")
	require.NotContains(t, privateStateLine, "}", "the following source line was appended to the comment line")

	letLine := lineContaining(t, plain, "let job_factory")
	require.Equal(t, 6, len(letLine)-len(strings.TrimLeft(letLine, " ")), "the source indentation changed: %q", letLine)
	spawnLine := lineContaining(t, plain, "spawn_actor")
	require.Equal(t, 6, len(spawnLine)-len(strings.TrimLeft(spawnLine, " ")), "the source indentation changed: %q", spawnLine)
}

func lineContaining(t *testing.T, content, fragment string) string {
	t.Helper()
	for line := range strings.SplitSeq(content, "\n") {
		if strings.Contains(line, fragment) {
			return line
		}
	}
	t.Fatalf("could not find %q in rendered output", fragment)
	return ""
}
