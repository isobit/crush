package tools

import (
	"context"
	"path/filepath"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

type recordingPathPermissionService struct {
	*mockViewPermissionService
	request permission.CreatePermissionRequest
}

func (s *recordingPathPermissionService) Request(_ context.Context, req permission.CreatePermissionRequest) (bool, error) {
	s.request = req
	return true, nil
}

func TestRequestToolPathPermission(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	outsideDir := t.TempDir()
	permissions := &recordingPathPermissionService{
		mockViewPermissionService: &mockViewPermissionService{
			Broker: pubsub.NewBroker[permission.PermissionRequest](),
		},
	}
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	call := fantasy.ToolCall{ID: "test-call"}

	path, granted, err := requestToolPathPermission(
		ctx,
		permissions,
		workingDir,
		outsideDir,
		call,
		GrepToolName,
		"read",
		"Search files outside working directory: %s",
		"session ID is required",
		GrepParams{Path: outsideDir},
	)
	require.NoError(t, err)
	require.True(t, granted)
	require.Equal(t, outsideDir, path)
	require.Equal(t, "test-session", permissions.request.SessionID)
	require.Equal(t, outsideDir, permissions.request.Path)
	require.Equal(t, GrepToolName, permissions.request.ToolName)
	require.Equal(t, "read", permissions.request.Action)
	require.Equal(t, "Search files outside working directory: "+outsideDir, permissions.request.Description)

	insidePath := filepath.Join(workingDir, "nested")
	path, granted, err = requestToolPathPermission(
		ctx,
		permissions,
		workingDir,
		insidePath,
		call,
		GlobToolName,
		"read",
		"Search directory outside working directory: %s",
		"session ID is required",
		GlobParams{Path: insidePath},
	)
	require.NoError(t, err)
	require.True(t, granted)
	require.Equal(t, insidePath, path)
	require.Equal(t, GrepToolName, permissions.request.ToolName)
}
