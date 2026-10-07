package common

import (
	"strings"

	"github.com/AlexanderGrooff/mermaid-ascii/pkg/diagram"
	mermaidrender "github.com/AlexanderGrooff/mermaid-ascii/pkg/render"
)

// RenderMermaidDiagrams replaces complete Mermaid fenced code blocks with
// rendered terminal diagrams. Unsupported or invalid diagrams remain unchanged.
func RenderMermaidDiagrams(markdown string, width int) (string, bool) {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	changed := false

	for i := 0; i < len(lines); {
		marker, markerLen, info, ok := parseCodeFence(lines[i])
		if !ok {
			out = append(out, lines[i])
			i++
			continue
		}

		closing := i + 1
		for closing < len(lines) && !isClosingFence(lines[closing], marker, markerLen) {
			closing++
		}
		if closing == len(lines) {
			out = append(out, lines[i:]...)
			break
		}

		fields := strings.Fields(info)
		if len(fields) == 0 || !strings.EqualFold(fields[0], "mermaid") {
			out = append(out, lines[i:closing+1]...)
			i = closing + 1
			continue
		}

		source := strings.Join(lines[i+1:closing], "\n")
		config := diagram.DefaultConfig()
		if width > 0 {
			config.MaxWidth = width
		}
		rendered, err := mermaidrender.RenderDiagram(source, config)
		if err != nil {
			out = append(out, lines[i:closing+1]...)
			i = closing + 1
			continue
		}

		out = append(out, "```text")
		out = append(out, strings.Split(strings.TrimRight(rendered, "\n"), "\n")...)
		out = append(out, "```")
		changed = true
		i = closing + 1
	}

	if !changed {
		return markdown, false
	}
	return strings.Join(out, "\n"), true
}

func parseCodeFence(line string) (byte, int, string, bool) {
	line = strings.TrimSuffix(line, "\r")
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if indent > 3 || indent == len(line) {
		return 0, 0, "", false
	}
	line = line[indent:]
	marker := line[0]
	if marker != '`' && marker != '~' {
		return 0, 0, "", false
	}
	markerLen := 0
	for markerLen < len(line) && line[markerLen] == marker {
		markerLen++
	}
	if markerLen < 3 {
		return 0, 0, "", false
	}
	info := strings.TrimSpace(line[markerLen:])
	if marker == '`' && strings.Contains(info, "`") {
		return 0, 0, "", false
	}
	return marker, markerLen, info, true
}

func isClosingFence(line string, marker byte, markerLen int) bool {
	line = strings.TrimSuffix(line, "\r")
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if indent > 3 || indent == len(line) {
		return false
	}
	line = line[indent:]
	if line[0] != marker {
		return false
	}
	count := 0
	for count < len(line) && line[count] == marker {
		count++
	}
	return count >= markerLen && strings.TrimSpace(line[count:]) == ""
}
