package markdiff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/aymanbagabas/go-udiff"
)

// maxHeadingLevel is the deepest heading Markdown supports.
const maxHeadingLevel = 6

// Body is a sequence of Markdown blocks. It is shared by [Doc], [Section],
// and the contents of [Body.Details]. Blocks are separated by blank lines
// and appear in the order they were added.
//
// Pair captured blocks (code, lists, tables, diffs) with a [Body.Text]
// paragraph that tells the reader what they are looking at.
type Body struct {
	tb     testing.TB
	level  int
	blocks []block
}

type block interface {
	markdown() string
}

type rawBlock string

func (r rawBlock) markdown() string { return string(r) }

func (b *Body) add(s string) {
	b.blocks = append(b.blocks, rawBlock(s))
}

func (b *Body) render() string {
	parts := make([]string, 0, len(b.blocks))
	for _, blk := range b.blocks {
		if s := strings.TrimRight(blk.markdown(), "\n"); s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n") + "\n"
}

// Section is a headed part of a document describing one behavior. Create
// one with [Body.Section].
type Section struct {
	Body

	title   string
	summary string
}

func (s *Section) markdown() string {
	level := min(s.level, maxHeadingLevel)
	out := strings.Repeat("#", level) + " " + s.title + "\n\n" + s.summary + "\n"
	if body := s.render(); body != "" {
		out += "\n" + body
	}
	return out
}

// Section adds a subsection one heading level deeper than b. The summary
// must state, in plain words, the behavior the section captures; the test
// fails if it is empty.
func (b *Body) Section(title, summary string) *Section {
	b.tb.Helper()
	s := &Section{
		Body:    Body{tb: b.tb, level: b.level + 1},
		title:   strings.TrimSpace(title),
		summary: strings.TrimSpace(summary),
	}
	if s.title == "" {
		b.tb.Fatalf("markdiff: section needs a title")
	}
	if s.summary == "" {
		b.tb.Fatalf("markdiff: section %q needs a summary describing the behavior it captures", s.title)
	}
	b.blocks = append(b.blocks, s)
	return s
}

// Text adds a paragraph of prose. Use it to explain the next captured
// block, or to point out what a reviewer should notice in the previous one.
func (b *Body) Text(paragraph string) {
	b.add(strings.TrimSpace(paragraph))
}

// Textf adds a paragraph formatted with [fmt.Sprintf].
func (b *Body) Textf(format string, args ...any) {
	b.Text(fmt.Sprintf(format, args...))
}

// Markdown adds raw Markdown verbatim, for structures the builder does not
// cover.
func (b *Body) Markdown(markdown string) {
	b.add(markdown)
}

// Code adds a fenced code block. Lang is the info string used for syntax
// highlighting and may be empty. The fence grows as needed so content
// containing backticks is preserved.
func (b *Body) Code(lang, content string) {
	b.add(codeBlock(lang, content))
}

// JSON adds v as an indented JSON code block. It is a convenient way to
// capture structured values; prefer it over %v formatting, whose output is
// harder to read and less stable.
func (b *Body) JSON(v any) {
	b.tb.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		b.tb.Fatalf("markdiff: encoding JSON: %v", err)
		return
	}
	b.Code("json", buf.String())
}

// Diff adds a unified diff between before and after as a diff code block.
// It is useful for showing how an operation transforms its input. When the
// two are equal, a note saying so is added instead.
func (b *Body) Diff(before, after string) {
	d := udiff.Unified("before", "after", ensureNewline(before), ensureNewline(after))
	if d == "" {
		b.add("_(no differences)_")
		return
	}
	b.Code("diff", d)
}

// List adds a bulleted list. Multi-line items are indented so they stay
// within their bullet. An empty list is rendered as an explicit note so the
// reader can tell it apart from missing output.
func (b *Body) List(items ...string) {
	b.add(list(items, func(int) string { return "- " }))
}

// NumberedList adds an ordered list, for captured sequences where order
// matters, such as events or steps.
func (b *Body) NumberedList(items ...string) {
	b.add(list(items, func(i int) string { return strconv.Itoa(i+1) + ". " }))
}

// Table adds a table. Each row must have at most len(header) cells; short
// rows are padded. Pipes and newlines in cells are escaped so they do not
// break the table.
func (b *Body) Table(header []string, rows ...[]string) {
	b.tb.Helper()
	if len(header) == 0 {
		b.tb.Fatalf("markdiff: table needs a header")
		return
	}
	var sb strings.Builder
	writeRow := func(cells []string) {
		sb.WriteString("|")
		for i := range header {
			cell := ""
			if i < len(cells) {
				cell = tableCell(cells[i])
			}
			sb.WriteString(" " + cell + " |")
		}
		sb.WriteString("\n")
	}
	writeRow(header)
	sb.WriteString("|")
	for range header {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")
	for i, row := range rows {
		if len(row) > len(header) {
			b.tb.Fatalf("markdiff: table row %d has %d cells but the header has %d", i, len(row), len(header))
			return
		}
		writeRow(row)
	}
	b.add(sb.String())
}

// Quote adds a block quote, for example to capture a message shown to a
// user.
func (b *Body) Quote(text string) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, l := range lines {
		if l == "" {
			lines[i] = ">"
		} else {
			lines[i] = "> " + l
		}
	}
	b.add(strings.Join(lines, "\n"))
}

// Details adds a collapsible block whose contents are built by fill. Use it
// for long captured output that supports the document but would distract
// from the main narrative.
func (b *Body) Details(summary string, fill func(*Body)) {
	inner := &Body{tb: b.tb, level: b.level}
	fill(inner)
	b.add("<details>\n<summary>" + strings.TrimSpace(summary) + "</summary>\n\n" + inner.render() + "\n</details>")
}

func codeBlock(lang, content string) string {
	fence := strings.Repeat("`", max(3, longestRun(content, '`')+1))
	return fence + lang + "\n" + ensureNewline(content) + fence
}

func ensureNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

func longestRun(s string, c byte) int {
	longest, cur := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			cur++
			longest = max(longest, cur)
		} else {
			cur = 0
		}
	}
	return longest
}

func list(items []string, marker func(int) string) string {
	if len(items) == 0 {
		return "_(none)_"
	}
	var sb strings.Builder
	for i, item := range items {
		m := marker(i)
		indent := strings.Repeat(" ", len(m))
		lines := strings.Split(strings.TrimRight(item, "\n"), "\n")
		sb.WriteString(m + lines[0] + "\n")
		for _, l := range lines[1:] {
			if l == "" {
				sb.WriteString("\n")
			} else {
				sb.WriteString(indent + l + "\n")
			}
		}
	}
	return sb.String()
}

func tableCell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "<br>")
	return s
}
