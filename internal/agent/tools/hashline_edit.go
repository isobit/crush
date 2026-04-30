package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/diff"
	"github.com/charmbracelet/crush/internal/filepathext"
	"github.com/charmbracelet/crush/internal/filetracker"
	"github.com/charmbracelet/crush/internal/fsext"
	"github.com/charmbracelet/crush/internal/hashline"
	"github.com/charmbracelet/crush/internal/history"
	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/permission"
)

//go:embed hashline_edit.md
var hashlineEditDescription []byte

const HashlineEditToolName = "hashline_edit"

type HashlineEditParams struct {
	Path   string       `json:"path" description:"The file path to edit"`
	Edits  []HashlineOp `json:"edits,omitempty" description:"Array of edit operations"`
	Delete bool         `json:"delete,omitempty" description:"Delete the file"`
	Move   string       `json:"move,omitempty" description:"Move/rename the file to this path"`
}

type HashlineOp struct {
	Op    string          `json:"op" description:"Operation: replace, append, or prepend"`
	Pos   string          `json:"pos,omitempty" description:"Start line reference (LINE#HASH)"`
	End   string          `json:"end,omitempty" description:"End line reference for range replace (LINE#HASH)"`
	Lines json.RawMessage `json:"lines" description:"New content lines: string[], string, or null (delete)"`
}

type HashlineEditPermissionsParams struct {
	FilePath   string `json:"file_path"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

func NewHashlineEditTool(
	lspManager *lsp.Manager,
	permissions permission.Service,
	files history.Service,
	filetracker filetracker.Service,
	workingDir string,
) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		HashlineEditToolName,
		string(hashlineEditDescription),
		func(ctx context.Context, params HashlineEditParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Path == "" {
				return fantasy.NewTextErrorResponse("path is required"), nil
			}

			filePath := filepathext.SmartJoin(workingDir, params.Path)

			if params.Delete {
				return hashlineDeleteFile(ctx, permissions, files, filetracker, filePath, workingDir, call)
			}

			if params.Move != "" {
				return hashlineMoveFile(ctx, permissions, files, filetracker, filePath, params.Move, workingDir, call)
			}

			if len(params.Edits) == 0 {
				return fantasy.NewTextErrorResponse("edits array is required when not using delete or move"), nil
			}

			isNewFile := false
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				isNewFile = true
			}

			if isNewFile {
				return hashlineCreateFile(ctx, permissions, files, filetracker, lspManager, filePath, workingDir, params.Edits, call)
			}

			return hashlineEditFile(ctx, permissions, files, filetracker, lspManager, filePath, workingDir, params.Edits, call)
		})
}

func hashlineDeleteFile(
	ctx context.Context,
	permissions permission.Service,
	files history.Service,
	filetracker filetracker.Service,
	filePath, workingDir string,
	call fantasy.ToolCall,
) (fantasy.ToolResponse, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("file not found: %s", filePath)), nil
	}

	sessionID := GetSessionFromContext(ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, fmt.Errorf("session ID is required")
	}

	granted, err := permissions.Request(ctx, permission.CreatePermissionRequest{
		SessionID:   sessionID,
		Path:        fsext.PathOrPrefix(filePath, workingDir),
		ToolCallID:  call.ID,
		ToolName:    HashlineEditToolName,
		Action:      "write",
		Description: fmt.Sprintf("Delete file %s", filePath),
		Params:      HashlineEditPermissionsParams{FilePath: filePath},
	})
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	if !granted {
		return NewPermissionDeniedResponse(), nil
	}

	oldContent, err := os.ReadFile(filePath)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
	}

	if err := os.Remove(filePath); err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to delete file: %w", err)
	}

	_, _ = files.Create(ctx, sessionID, filePath, string(oldContent))
	_, _ = files.CreateVersion(ctx, sessionID, filePath, "")

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse("File deleted: "+filePath),
		EditResponseMetadata{OldContent: string(oldContent), NewContent: ""},
	), nil
}

func hashlineMoveFile(
	ctx context.Context,
	permissions permission.Service,
	files history.Service,
	filetracker filetracker.Service,
	oldPath, newRelPath, workingDir string,
	call fantasy.ToolCall,
) (fantasy.ToolResponse, error) {
	newPath := filepathext.SmartJoin(workingDir, newRelPath)

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("file not found: %s", oldPath)), nil
	}

	sessionID := GetSessionFromContext(ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, fmt.Errorf("session ID is required")
	}

	granted, err := permissions.Request(ctx, permission.CreatePermissionRequest{
		SessionID:   sessionID,
		Path:        fsext.PathOrPrefix(oldPath, workingDir),
		ToolCallID:  call.ID,
		ToolName:    HashlineEditToolName,
		Action:      "write",
		Description: fmt.Sprintf("Move file %s to %s", oldPath, newPath),
		Params:      HashlineEditPermissionsParams{FilePath: oldPath},
	})
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	if !granted {
		return NewPermissionDeniedResponse(), nil
	}

	dir := filepath.Dir(newPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to create parent directories: %w", err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to move file: %w", err)
	}

	filetracker.RecordRead(ctx, sessionID, newPath)

	return fantasy.NewTextResponse(fmt.Sprintf("File moved: %s → %s", oldPath, newPath)), nil
}

func hashlineCreateFile(
	ctx context.Context,
	permissions permission.Service,
	files history.Service,
	filetracker filetracker.Service,
	lspManager *lsp.Manager,
	filePath, workingDir string,
	edits []HashlineOp,
	call fantasy.ToolCall,
) (fantasy.ToolResponse, error) {
	for _, edit := range edits {
		if edit.Pos != "" || edit.End != "" {
			return fantasy.NewTextErrorResponse("cannot use pos/end references when creating a new file; use anchorless append/prepend only"), nil
		}
		if edit.Op != "append" && edit.Op != "prepend" {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("only append/prepend allowed for file creation, got %q", edit.Op)), nil
		}
	}

	sessionID := GetSessionFromContext(ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, fmt.Errorf("session ID is required")
	}

	var allLines []string
	for _, edit := range edits {
		lines, err := parseLines(edit.Lines)
		if err != nil {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("invalid lines: %v", err)), nil
		}
		if lines == nil {
			continue
		}
		allLines = append(allLines, *lines...)
	}

	newContent := strings.Join(allLines, "\n")
	if len(allLines) > 0 {
		newContent += "\n"
	}

	_, additions, removals := diff.GenerateDiff("", newContent, strings.TrimPrefix(filePath, workingDir))

	granted, err := permissions.Request(ctx, permission.CreatePermissionRequest{
		SessionID:   sessionID,
		Path:        fsext.PathOrPrefix(filePath, workingDir),
		ToolCallID:  call.ID,
		ToolName:    HashlineEditToolName,
		Action:      "write",
		Description: fmt.Sprintf("Create file %s", filePath),
		Params: HashlineEditPermissionsParams{
			FilePath:   filePath,
			NewContent: newContent,
		},
	})
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	if !granted {
		return NewPermissionDeniedResponse(), nil
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to create parent directories: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	_, _ = files.Create(ctx, sessionID, filePath, "")
	_, _ = files.CreateVersion(ctx, sessionID, filePath, newContent)
	filetracker.RecordRead(ctx, sessionID, filePath)

	notifyLSPs(ctx, lspManager, filePath)

	text := fmt.Sprintf("<result>\nFile created: %s\n</result>\n", filePath)
	text += getDiagnostics(filePath, lspManager)

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse(text),
		EditResponseMetadata{
			NewContent: newContent,
			Additions:  additions,
			Removals:   removals,
		},
	), nil
}

func hashlineEditFile(
	ctx context.Context,
	permissions permission.Service,
	files history.Service,
	filetracker filetracker.Service,
	lspManager *lsp.Manager,
	filePath, workingDir string,
	edits []HashlineOp,
	call fantasy.ToolCall,
) (fantasy.ToolResponse, error) {
	sessionID := GetSessionFromContext(ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, fmt.Errorf("session ID is required")
	}

	lastRead := filetracker.LastReadTime(ctx, sessionID, filePath)
	if lastRead.IsZero() {
		return fantasy.NewTextErrorResponse("you must read the file before editing it. Use the view tool first"), nil
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to stat file: %w", err)
	}

	modTime := fileInfo.ModTime().Truncate(time.Second)
	if modTime.After(lastRead) {
		return fantasy.NewTextErrorResponse(
			fmt.Sprintf("file %s has been modified since it was last read (mod time: %s, last read: %s)",
				filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339),
			)), nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
	}

	oldContent, isCrlf := fsext.ToUnixLineEndings(string(content))

	lines := strings.Split(oldContent, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Validation phase: parse all refs, check bounds, verify hashes.
	var resolved []resolvedEdit
	var allRefs []string
	var validationErrors []string

	for i, edit := range edits {
		re := resolvedEdit{op: edit.Op, origIdx: i}

		switch edit.Op {
		case "replace", "append", "prepend":
		default:
			validationErrors = append(validationErrors, fmt.Sprintf("edit %d: unknown op %q", i+1, edit.Op))
			continue
		}

		newLines, err := parseLines(edit.Lines)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("edit %d: invalid lines: %v", i+1, err))
			continue
		}
		re.newLines = newLines

		if edit.Pos != "" {
			lineNum, hash, err := hashline.ParseRef(edit.Pos)
			if err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("edit %d: invalid pos: %v", i+1, err))
				continue
			}
			if lineNum < 1 || lineNum > len(lines) {
				validationErrors = append(validationErrors, fmt.Sprintf("edit %d: pos line %d out of bounds (file has %d lines)", i+1, lineNum, len(lines)))
				continue
			}
			re.startLine = lineNum
			re.hasAnchor = true
			allRefs = append(allRefs, fmt.Sprintf("%d#%s", lineNum, hash))
		}

		if edit.End != "" {
			if edit.Op != "replace" {
				validationErrors = append(validationErrors, fmt.Sprintf("edit %d: end is only valid for replace operations", i+1))
				continue
			}
			lineNum, hash, err := hashline.ParseRef(edit.End)
			if err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("edit %d: invalid end: %v", i+1, err))
				continue
			}
			if lineNum < 1 || lineNum > len(lines) {
				validationErrors = append(validationErrors, fmt.Sprintf("edit %d: end line %d out of bounds (file has %d lines)", i+1, lineNum, len(lines)))
				continue
			}
			re.endLine = lineNum
			allRefs = append(allRefs, fmt.Sprintf("%d#%s", lineNum, hash))
		} else if edit.Op == "replace" && re.hasAnchor {
			re.endLine = re.startLine
		}

		if edit.Op == "replace" && !re.hasAnchor {
			validationErrors = append(validationErrors, fmt.Sprintf("edit %d: replace requires pos", i+1))
			continue
		}

		if re.endLine > 0 && re.startLine > 0 && re.endLine < re.startLine {
			validationErrors = append(validationErrors, fmt.Sprintf("edit %d: end line %d is before start line %d", i+1, re.endLine, re.startLine))
			continue
		}

		resolved = append(resolved, re)
	}

	if len(validationErrors) > 0 {
		return fantasy.NewTextErrorResponse("Validation errors:\n" + strings.Join(validationErrors, "\n")), nil
	}

	// Hash verification.
	if len(allRefs) > 0 {
		mismatches := hashline.ValidateRefs(lines, allRefs...)
		if len(mismatches) > 0 {
			var errParts []string
			errParts = append(errParts, "Hash mismatch — file has changed. Re-read with view tool:")
			for _, m := range mismatches {
				errParts = append(errParts, fmt.Sprintf(
					"  line %d: expected hash %s, actual %s, content: %q",
					m.Line, m.ExpectedHash, m.ActualHash, m.CurrentContent,
				))
			}
			return fantasy.NewTextErrorResponse(strings.Join(errParts, "\n")), nil
		}
	}

	// Check for overlapping ranges.
	if err := checkOverlaps(resolved); err != nil {
		return fantasy.NewTextErrorResponse(err.Error()), nil
	}

	// Sort edits descending by anchor line (bottom-up application).
	sort.SliceStable(resolved, func(i, j int) bool {
		ai := anchorLine(resolved[i])
		aj := anchorLine(resolved[j])
		return ai > aj
	})

	// Apply edits.
	for _, re := range resolved {
		newLines := re.newLines
		stripped := stripHashlinePrefixes(newLines)

		switch re.op {
		case "replace":
			start := re.startLine - 1
			end := re.endLine
			if stripped == nil {
				lines = append(lines[:start], lines[end:]...)
			} else {
				replacement := make([]string, 0, len(*stripped)+len(lines))
				replacement = append(replacement, lines[:start]...)
				replacement = append(replacement, *stripped...)
				replacement = append(replacement, lines[end:]...)
				lines = replacement
			}

		case "append":
			if !re.hasAnchor {
				if stripped != nil {
					lines = append(lines, *stripped...)
				}
			} else {
				insertAt := re.startLine
				if stripped != nil {
					newSlice := make([]string, 0, len(lines)+len(*stripped))
					newSlice = append(newSlice, lines[:insertAt]...)
					newSlice = append(newSlice, *stripped...)
					newSlice = append(newSlice, lines[insertAt:]...)
					lines = newSlice
				}
			}

		case "prepend":
			if !re.hasAnchor {
				if stripped != nil {
					lines = append(*stripped, lines...)
				}
			} else {
				insertAt := re.startLine - 1
				if stripped != nil {
					newSlice := make([]string, 0, len(lines)+len(*stripped))
					newSlice = append(newSlice, lines[:insertAt]...)
					newSlice = append(newSlice, *stripped...)
					newSlice = append(newSlice, lines[insertAt:]...)
					lines = newSlice
				}
			}
		}
	}

	newContent := strings.Join(lines, "\n")
	if len(lines) > 0 {
		newContent += "\n"
	}

	if oldContent == newContent {
		return fantasy.NewTextErrorResponse("no changes detected. Re-read the file to verify current content"), nil
	}

	_, additions, removals := diff.GenerateDiff(oldContent, newContent, strings.TrimPrefix(filePath, workingDir))

	granted, err := permissions.Request(ctx, permission.CreatePermissionRequest{
		SessionID:   sessionID,
		Path:        fsext.PathOrPrefix(filePath, workingDir),
		ToolCallID:  call.ID,
		ToolName:    HashlineEditToolName,
		Action:      "write",
		Description: fmt.Sprintf("Edit file %s", filePath),
		Params: HashlineEditPermissionsParams{
			FilePath:   filePath,
			OldContent: oldContent,
			NewContent: newContent,
		},
	})
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	if !granted {
		return NewPermissionDeniedResponse(), nil
	}

	writeContent := newContent
	if isCrlf {
		writeContent, _ = fsext.ToWindowsLineEndings(newContent)
	}

	if err := os.WriteFile(filePath, []byte(writeContent), 0o644); err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	file, err := files.GetByPathAndSession(ctx, filePath, sessionID)
	if err != nil {
		_, err = files.Create(ctx, sessionID, filePath, oldContent)
		if err != nil {
			return fantasy.ToolResponse{}, fmt.Errorf("error creating file history: %w", err)
		}
	}
	if file.Content != oldContent {
		_, err = files.CreateVersion(ctx, sessionID, filePath, oldContent)
		if err != nil {
			slog.Error("Error creating file history version", "error", err)
		}
	}
	_, err = files.CreateVersion(ctx, sessionID, filePath, newContent)
	if err != nil {
		slog.Error("Error creating file history version", "error", err)
	}

	filetracker.RecordRead(ctx, sessionID, filePath)

	notifyLSPs(ctx, lspManager, filePath)

	text := fmt.Sprintf("<result>\nFile edited: %s (%d additions, %d removals)\n</result>\n", filePath, additions, removals)
	text += getDiagnostics(filePath, lspManager)

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse(text),
		EditResponseMetadata{
			OldContent: oldContent,
			NewContent: newContent,
			Additions:  additions,
			Removals:   removals,
		},
	), nil
}

// parseLines normalizes the json.RawMessage Lines field into *[]string.
// Returns nil for JSON null (meaning delete), a slice for array or
// single string.
func parseLines(raw json.RawMessage) (*[]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return &arr, nil
	}

	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		lines := strings.Split(single, "\n")
		return &lines, nil
	}

	return nil, fmt.Errorf("lines must be a string array, a string, or null")
}

// stripHashlinePrefixes strips `LINE#HASH| ` prefixes from content
// lines only when a majority (≥50%) of non-empty lines carry the
// prefix. This avoids corrupting legitimate content like markdown
// anchors or comments that happen to match the pattern.
func stripHashlinePrefixes(lines *[]string) *[]string {
	if lines == nil {
		return nil
	}

	nonEmpty := 0
	prefixCount := 0
	for _, line := range *lines {
		if line == "" {
			continue
		}
		nonEmpty++
		if hasHashlinePrefix(line) {
			prefixCount++
		}
	}

	if nonEmpty == 0 || prefixCount == 0 || prefixCount < (nonEmpty+1)/2 {
		result := make([]string, len(*lines))
		copy(result, *lines)
		return &result
	}

	result := make([]string, len(*lines))
	for i, line := range *lines {
		result[i] = stripSingleHashlinePrefix(line)
	}
	return &result
}

// hasHashlinePrefix reports whether a line starts with a `LINE#HASH| `
// pattern matching the hashline view output format.
func hasHashlinePrefix(s string) bool {
	return stripSingleHashlinePrefix(s) != s
}

// stripSingleHashlinePrefix strips a leading `LINE#HASH| ` pattern
// from a single line. Returns the line unchanged if no prefix is found.
func stripSingleHashlinePrefix(s string) string {
	for i, ch := range s {
		if i == 0 && (ch < '0' || ch > '9') {
			return s
		}
		if ch == '#' {
			rest := s[i+1:]
			pipeIdx := strings.Index(rest, "| ")
			if pipeIdx >= 1 && pipeIdx <= 4 {
				hashPart := rest[:pipeIdx]
				allHex := true
				for _, hc := range hashPart {
					if !((hc >= '0' && hc <= '9') || (hc >= 'a' && hc <= 'f') || (hc >= 'A' && hc <= 'F')) {
						allHex = false
						break
					}
				}
				if allHex {
					return rest[pipeIdx+2:]
				}
			}
			return s
		}
		if ch < '0' || ch > '9' {
			return s
		}
	}
	return s
}

func checkOverlaps(edits []resolvedEdit) error {
	type lineRange struct {
		start, end, idx int
	}
	var ranges []lineRange
	for _, re := range edits {
		if re.op == "replace" && re.startLine > 0 {
			ranges = append(ranges, lineRange{re.startLine, re.endLine, re.origIdx})
		}
	}
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].start < ranges[j].start
	})
	for i := 1; i < len(ranges); i++ {
		if ranges[i].start <= ranges[i-1].end {
			return fmt.Errorf("overlapping edits: edit %d (lines %d-%d) overlaps with edit %d (lines %d-%d)",
				ranges[i-1].idx+1, ranges[i-1].start, ranges[i-1].end,
				ranges[i].idx+1, ranges[i].start, ranges[i].end,
			)
		}
	}
	return nil
}

func anchorLine(re resolvedEdit) int {
	if re.startLine > 0 {
		return re.startLine
	}
	if re.op == "append" {
		return 1<<31 - 1
	}
	return 0
}

// resolvedEdit is used internally during edit processing.
type resolvedEdit struct {
	op        string
	startLine int
	endLine   int
	newLines  *[]string
	hasAnchor bool
	origIdx   int
}
