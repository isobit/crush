package workspace

import (
	"context"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/stretchr/testify/require"
)

type recordingRetryCoordinator struct {
	agent.Coordinator
	prompt string
}

func (c *recordingRetryCoordinator) Run(_ context.Context, _ string, prompt string, _ ...message.Attachment) (*fantasy.AgentResult, error) {
	c.prompt = prompt
	return nil, nil
}

func TestAppWorkspaceAgentRetryUsesContinuationPrompt(t *testing.T) {
	t.Parallel()

	conn, err := db.Connect(t.Context(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	q := db.New(conn)
	sessions := session.NewService(q, conn)
	sess, err := sessions.Create(t.Context(), "test")
	require.NoError(t, err)
	messages := message.NewService(q)

	_, err = messages.Create(t.Context(), sess.ID, message.CreateMessageParams{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Finish the implementation"},
		},
	})
	require.NoError(t, err)
	failed, err := messages.Create(t.Context(), sess.ID, message.CreateMessageParams{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.Finish{Reason: message.FinishReasonError, Message: "Provider Error", Details: "connection reset"},
		},
	})
	require.NoError(t, err)

	coordinator := &recordingRetryCoordinator{}
	workspace := NewAppWorkspace(&app.App{
		Messages:         messages,
		AgentCoordinator: coordinator,
	}, nil)

	require.NoError(t, workspace.AgentRetry(t.Context(), sess.ID, failed.ID))
	require.Contains(t, coordinator.prompt, "Provider Error")
	require.Contains(t, coordinator.prompt, "connection reset")
	require.NotContains(t, coordinator.prompt, "Finish the implementation")
	require.Contains(t, coordinator.prompt, "existing partial workspace")
	require.Contains(t, coordinator.prompt, "Do not repeat work that is already complete")
}
