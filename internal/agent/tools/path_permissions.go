package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/permission"
)

func resolveToolPath(workingDir, path string) (string, bool, error) {
	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", false, fmt.Errorf("error resolving working directory: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false, fmt.Errorf("error resolving path: %w", err)
	}

	relPath, err := filepath.Rel(absWorkingDir, absPath)
	return absPath, err != nil || strings.HasPrefix(relPath, ".."), err
}

func requestToolPathPermission(
	ctx context.Context,
	permissions permission.Service,
	workingDir, path string,
	call fantasy.ToolCall,
	toolName, action, description, sessionError string,
	params any,
) (string, bool, error) {
	absPath, outside, err := resolveToolPath(workingDir, path)
	if err != nil {
		return "", false, err
	}
	if !outside {
		return absPath, true, nil
	}

	sessionID := GetSessionFromContext(ctx)
	if sessionID == "" {
		return "", false, fmt.Errorf("%s", sessionError)
	}

	granted, err := permissions.Request(
		ctx,
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        absPath,
			ToolCallID:  call.ID,
			ToolName:    toolName,
			Action:      action,
			Description: fmt.Sprintf(description, absPath),
			Params:      params,
		},
	)
	if err != nil {
		return "", false, err
	}
	return absPath, granted, nil
}
