package hashline

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"unicode"
)

// LineRef holds a line number, its content hash, and the original content.
type LineRef struct {
	Num     int
	Hash    string
	Content string
}

// Mismatch describes a hash verification failure for a single line
// reference.
type Mismatch struct {
	Line           int
	ExpectedHash   string
	ActualHash     string
	CurrentContent string
	Reference      string
}

// NormalizeLine trims whitespace, collapses internal runs of whitespace
// to a single space, and lowercases the result. This provides
// hash stability across minor formatting changes.
func NormalizeLine(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\uFEFF")
	var b strings.Builder
	b.Grow(len(s))
	inSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !inSpace {
				b.WriteRune(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(unicode.ToLower(r))
			inSpace = false
		}
	}
	return b.String()
}

// ComputeHash normalizes the line then computes an FNV-1a hash
// truncated to 3 hex characters (12 bits, 4096 buckets).
func ComputeHash(line string) string {
	norm := NormalizeLine(line)
	h := fnv.New32a()
	h.Write([]byte(norm))
	return fmt.Sprintf("%03x", h.Sum32()&0xFFF)
}

// FormatLine produces a single hashline-formatted line:
// "LINE#HASH| CONTENT".
func FormatLine(lineNum int, hash, content string) string {
	return fmt.Sprintf("%d#%s| %s", lineNum, hash, content)
}

// FormatLines formats a slice of content lines starting from
// startLine (1-based) using hashline format.
func FormatLines(lines []string, startLine int) string {
	if len(lines) == 0 {
		return ""
	}
	var result []string
	for i, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		lineNum := i + startLine
		hash := ComputeHash(line)
		result = append(result, FormatLine(lineNum, hash, line))
	}
	return strings.Join(result, "\n")
}

// ParseRef parses a "LINE#HASH" reference string into its
// components. Returns an error for malformed input.
func ParseRef(ref string) (int, string, error) {
	parts := strings.SplitN(ref, "#", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("malformed ref %q: expected LINE#HASH", ref)
	}
	lineNum, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("malformed ref %q: invalid line number: %w", ref, err)
	}
	hash := parts[1]
	if hash == "" {
		return 0, "", fmt.Errorf("malformed ref %q: empty hash", ref)
	}
	return lineNum, hash, nil
}

// HashFile computes a LineRef for every line in the given slice.
// Line numbers are 1-based.
func HashFile(lines []string) []LineRef {
	refs := make([]LineRef, len(lines))
	for i, line := range lines {
		refs[i] = LineRef{
			Num:     i + 1,
			Hash:    ComputeHash(line),
			Content: line,
		}
	}
	return refs
}

// ValidateRefs batch-validates line references against the file
// contents. Returns all mismatches found. An empty return means all
// refs are valid.
func ValidateRefs(lines []string, refs ...string) []Mismatch {
	var mismatches []Mismatch
	for _, ref := range refs {
		lineNum, expectedHash, err := ParseRef(ref)
		if err != nil {
			mismatches = append(mismatches, Mismatch{
				Line:         0,
				ExpectedHash: "",
				ActualHash:   "",
				Reference:    ref,
			})
			continue
		}
		if lineNum < 1 || lineNum > len(lines) {
			mismatches = append(mismatches, Mismatch{
				Line:         lineNum,
				ExpectedHash: expectedHash,
				ActualHash:   "",
				Reference:    ref,
			})
			continue
		}
		actualHash := ComputeHash(lines[lineNum-1])
		if actualHash != expectedHash {
			mismatches = append(mismatches, Mismatch{
				Line:           lineNum,
				ExpectedHash:   expectedHash,
				ActualHash:     actualHash,
				CurrentContent: lines[lineNum-1],
				Reference:      ref,
			})
		}
	}
	return mismatches
}
