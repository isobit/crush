package model

import (
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

// viMode represents the current vi editing mode.
type viMode uint8

const (
	viInsert viMode = iota
	viNormal
)

// viState holds the state for vi-style keybindings.
type viState struct {
	enabled bool
	mode    viMode

	// pending stores a partial normal-mode command (e.g., "d" waiting for
	// a motion, or "g" waiting for another "g").
	pending string

	// baseCursorShape is the cursor shape configured by the theme, used in
	// insert mode. Normal mode always uses CursorBlock.
	baseCursorShape tea.CursorShape
}

// viHandleNormalKey processes a keypress in vi normal mode. Returns true if
// the key was consumed.
func (m *UI) viHandleNormalKey(msg tea.KeyPressMsg) (consumed bool, cmd tea.Cmd) {
	k := msg.String()

	// Handle pending commands first.
	if m.vi.pending != "" {
		return m.viHandlePending(k)
	}

	switch {
	// Mode switching.
	case k == "i":
		m.viEnterInsert()
	case k == "I":
		m.textarea.CursorStart()
		m.viEnterInsert()
	case k == "a":
		m.viCursorRight()
		m.viEnterInsert()
	case k == "A":
		m.textarea.CursorEnd()
		m.viEnterInsert()
	case k == "o":
		m.textarea.CursorEnd()
		m.textarea.InsertRune('\n')
		m.viEnterInsert()
	case k == "O":
		m.textarea.CursorStart()
		m.textarea.InsertRune('\n')
		m.textarea.CursorUp()
		m.viEnterInsert()

	// Movement.
	case k == "h" || k == "left":
		m.viCursorLeft()
	case k == "l" || k == "right":
		m.viCursorRight()
	case k == "j" || k == "down":
		m.textarea.CursorDown()
	case k == "k" || k == "up":
		m.textarea.CursorUp()
	case k == "w":
		m.viWordForward()
	case k == "b":
		m.viWordBackward()
	case k == "e":
		m.viWordEnd()
	case k == "0" || k == "home":
		m.textarea.CursorStart()
	case k == "$" || k == "end":
		m.textarea.CursorEnd()

	// Document movement.
	case k == "G":
		m.textarea.MoveToEnd()
	case k == "g":
		m.vi.pending = "g"

	// Editing.
	case k == "x" || k == "delete":
		m.viDeleteCharForward()
	case k == "d":
		m.vi.pending = "d"
	case k == "c":
		m.vi.pending = "c"
	case k == "C":
		m.viDeleteToEnd()
		m.viEnterInsert()
	case k == "D":
		m.viDeleteToEnd()
	case k == "s":
		m.viDeleteCharForward()
		m.viEnterInsert()
	case k == "S":
		m.viDeleteLine()
		m.viEnterInsert()

	default:
		return true, nil
	}

	return true, nil
}

// viHandlePending handles the second key of a two-key command.
func (m *UI) viHandlePending(k string) (bool, tea.Cmd) {
	pending := m.vi.pending
	m.vi.pending = ""

	switch pending {
	case "d":
		switch k {
		case "d":
			m.viDeleteLine()
		case "w":
			m.viDeleteWord()
		case "$":
			m.viDeleteToEnd()
		case "0":
			m.viDeleteToStart()
		default:
			return true, nil
		}
	case "c":
		switch k {
		case "c":
			m.viChangeLine()
		case "w":
			m.viChangeWord(false)
		case "W":
			m.viChangeWord(true)
		case "$":
			m.viDeleteToEnd()
			m.viEnterInsert()
		case "0":
			m.viDeleteToStart()
			m.viEnterInsert()
		case "i", "a":
			m.vi.pending = "c" + k
			return true, nil
		}
	case "ci":
		if k == "w" {
			m.viChangeInnerWord()
		}
	case "ca":
		if k == "w" {
			m.viChangeAroundWord()
		}
	case "g":
		switch k {
		case "g":
			m.textarea.MoveToBegin()
		default:
			return true, nil
		}
	}

	return true, nil
}

// viEnterInsert switches to insert mode and updates the cursor shape.
func (m *UI) viEnterInsert() {
	m.vi.mode = viInsert
	m.viUpdateCursor()
}

// viEnterNormal switches to normal mode and updates the cursor shape.
func (m *UI) viEnterNormal() {
	m.vi.mode = viNormal
	m.vi.pending = ""
	m.viUpdateCursor()
}

// viUpdateCursor sets the cursor shape based on the current mode.
func (m *UI) viUpdateCursor() {
	s := m.textarea.Styles()
	if m.vi.mode == viNormal {
		s.Cursor.Shape = tea.CursorBlock
	} else {
		s.Cursor.Shape = m.vi.baseCursorShape
	}
	m.textarea.SetStyles(s)
}

func viKeyMsg(code rune, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Mod: mod}
}

// viCursorLeft moves the cursor one character left.
func (m *UI) viCursorLeft() {
	ta, _ := m.textarea.Update(viKeyMsg(tea.KeyLeft, 0))
	m.textarea = ta
}

// viCursorRight moves the cursor one character right.
func (m *UI) viCursorRight() {
	ta, _ := m.textarea.Update(viKeyMsg(tea.KeyRight, 0))
	m.textarea = ta
}

// viWordForward moves the cursor forward to the start of the next word.
func (m *UI) viWordForward() {
	ta, _ := m.textarea.Update(viKeyMsg(tea.KeyRight, tea.ModAlt))
	m.textarea = ta
}

// viWordBackward moves the cursor backward to the start of the previous word.
func (m *UI) viWordBackward() {
	ta, _ := m.textarea.Update(viKeyMsg(tea.KeyLeft, tea.ModAlt))
	m.textarea = ta
}

// viWordEnd moves cursor to end of current/next word.
func (m *UI) viWordEnd() {
	m.viWordForward()
}

func (m *UI) viReplaceRange(start, end int) {
	lines := strings.Split(m.textarea.Value(), "\n")
	row := m.textarea.Line()
	if row >= len(lines) {
		return
	}

	runes := []rune(lines[row])
	start = min(max(start, 0), len(runes))
	end = min(max(end, start), len(runes))
	lines[row] = string(append(runes[:start], runes[end:]...))
	m.textarea.SetValue(strings.Join(lines, "\n"))
	m.textarea.MoveToBegin()
	for range row {
		m.textarea.CursorDown()
	}
	m.textarea.SetCursorColumn(start)
}

func (m *UI) viChangeWord(blankDelimited bool) {
	lines := strings.Split(m.textarea.Value(), "\n")
	row := m.textarea.Line()
	if row >= len(lines) {
		return
	}

	runes := []rune(lines[row])
	start := min(m.textarea.Column(), len(runes))
	end := start
	for end < len(runes) && (isViWordRune(runes[end]) || (blankDelimited && !unicode.IsSpace(runes[end]))) {
		end++
	}
	if end == start {
		for end < len(runes) && unicode.IsSpace(runes[end]) {
			end++
		}
		start = end
		for end < len(runes) && (isViWordRune(runes[end]) || (blankDelimited && !unicode.IsSpace(runes[end]))) {
			end++
		}
	}
	m.viReplaceRange(start, end)
	m.viEnterInsert()
}

func (m *UI) viChangeInnerWord() {
	lines := strings.Split(m.textarea.Value(), "\n")
	row := m.textarea.Line()
	if row >= len(lines) {
		return
	}

	runes := []rune(lines[row])
	cursor := min(m.textarea.Column(), len(runes))
	for cursor < len(runes) && !isViWordRune(runes[cursor]) {
		cursor++
	}
	start := cursor
	for start > 0 && isViWordRune(runes[start-1]) {
		start--
	}
	end := cursor
	for end < len(runes) && isViWordRune(runes[end]) {
		end++
	}
	m.viReplaceRange(start, end)
	m.viEnterInsert()
}

func (m *UI) viChangeAroundWord() {
	lines := strings.Split(m.textarea.Value(), "\n")
	row := m.textarea.Line()
	if row >= len(lines) {
		return
	}

	runes := []rune(lines[row])
	cursor := min(m.textarea.Column(), len(runes))
	for cursor < len(runes) && !isViWordRune(runes[cursor]) {
		cursor++
	}
	start := cursor
	for start > 0 && isViWordRune(runes[start-1]) {
		start--
	}
	end := cursor
	for end < len(runes) && isViWordRune(runes[end]) {
		end++
	}
	for end < len(runes) && unicode.IsSpace(runes[end]) {
		end++
	}
	if end == cursor {
		return
	}
	m.viReplaceRange(start, end)
	m.viEnterInsert()
}

func (m *UI) viChangeLine() {
	lines := strings.Split(m.textarea.Value(), "\n")
	row := m.textarea.Line()
	if row >= len(lines) {
		return
	}

	m.viReplaceRange(0, len([]rune(lines[row])))
	m.viEnterInsert()
}

func isViWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// viDeleteCharForward deletes the character under the cursor.
func (m *UI) viDeleteCharForward() {
	ta, _ := m.textarea.Update(viKeyMsg(tea.KeyDelete, 0))
	m.textarea = ta
}

// viDeleteWord deletes from cursor to start of next word.
func (m *UI) viDeleteWord() {
	ta, _ := m.textarea.Update(viKeyMsg(tea.KeyDelete, tea.ModAlt))
	m.textarea = ta
}

// viDeleteToEnd deletes from cursor to end of line (ctrl+k).
func (m *UI) viDeleteToEnd() {
	ta, _ := m.textarea.Update(viKeyMsg('k', tea.ModCtrl))
	m.textarea = ta
}

// viDeleteToStart deletes from cursor to start of line (ctrl+u).
func (m *UI) viDeleteToStart() {
	ta, _ := m.textarea.Update(viKeyMsg('u', tea.ModCtrl))
	m.textarea = ta
}

// viDeleteLine deletes the entire current line.
func (m *UI) viDeleteLine() {
	value := m.textarea.Value()
	lines := strings.Split(value, "\n")

	row := m.textarea.Line()
	if row >= len(lines) {
		return
	}

	lines = append(lines[:row], lines[row+1:]...)
	newValue := strings.Join(lines, "\n")
	m.textarea.SetValue(newValue)

	for range min(row, len(lines)-1) {
		m.textarea.CursorDown()
	}
	m.textarea.CursorStart()
}

// viEnabled returns whether vi mode is active.
func (m *UI) viEnabled() bool {
	return m.vi.enabled
}

// viIsNormal returns whether we're in normal mode.
func (m *UI) viIsNormal() bool {
	return m.vi.enabled && m.vi.mode == viNormal
}

// viModeIndicator returns a short string for the status bar.
func (m *UI) viModeIndicator() string {
	if !m.vi.enabled {
		return ""
	}
	if m.vi.mode == viNormal {
		if m.vi.pending != "" {
			return m.vi.pending
		}
		return "NORMAL"
	}
	return "INSERT"
}
