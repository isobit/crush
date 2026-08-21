package model

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/ui/chat"
	"github.com/charmbracelet/crush/internal/workspace"
	"github.com/stretchr/testify/require"
)

type retryWorkspace struct {
	workspace.Workspace

	sessionID string
	messageID string
}

func (w *retryWorkspace) AgentRetry(_ context.Context, sessionID, messageID string) error {
	w.sessionID = sessionID
	w.messageID = messageID
	return nil
}

func TestRetrySelectedErrorMessage(t *testing.T) {
	t.Parallel()

	failedMessage := message.Message{
		ID:   "assistant",
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.Finish{Reason: message.FinishReasonError},
		},
	}
	ws := &retryWorkspace{}
	m := newBusyUI(ws)
	m.focus = uiFocusMain
	warmCaches(m, false)
	m.chat.SetMessages(chat.NewAssistantMessageItem(m.com.Styles, &failedMessage))
	m.chat.SetSelected(0)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r'})
	result, ok := cmd().(retryMessageMsg)
	require.True(t, ok)
	require.NoError(t, result.err)
	require.Equal(t, "s1", ws.sessionID)
	require.Equal(t, failedMessage.ID, ws.messageID)
}
