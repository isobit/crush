package common

import (
	"sort"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

type bashDisplayEdit struct {
	start int
	end   int
	text  string
}

func FormatBashDisplay(source string) string {
	file, err := syntax.NewParser().Parse(strings.NewReader(source), "")
	if err != nil {
		return source
	}

	formatter := bashDisplayFormatter{source: source}
	formatter.walkStmts(file.Stmts, 0, syntax.Pos{})
	return formatter.apply()
}

type bashDisplayFormatter struct {
	source string
	edits  []bashDisplayEdit
}

func (f *bashDisplayFormatter) walkStmts(stmts []*syntax.Stmt, depth int, nextKeyword syntax.Pos) {
	for index, stmt := range stmts {
		if stmt == nil {
			continue
		}
		if stmt.Semicolon.IsValid() && (index != len(stmts)-1 || !f.before(stmt.Semicolon, nextKeyword)) {
			f.after(stmt.Semicolon.Offset()+1, depth)
		}
		f.walkStmt(stmt, depth)
	}
}

func (f *bashDisplayFormatter) walkStmt(stmt *syntax.Stmt, depth int) {
	switch command := stmt.Cmd.(type) {
	case *syntax.ForClause:
		f.afterKeyword(command.DoPos, "do", depth+1)
		f.walkStmts(command.Do, depth+1, command.DonePos)
		f.beforeKeyword(command.DonePos, depth)
	case *syntax.WhileClause:
		f.afterKeyword(command.DoPos, "do", depth+1)
		f.walkStmts(command.Cond, depth, command.DoPos)
		f.walkStmts(command.Do, depth+1, command.DonePos)
		f.beforeKeyword(command.DonePos, depth)
	case *syntax.IfClause:
		f.walkIfClause(command, depth)
	case *syntax.Block:
		f.walkStmts(command.Stmts, depth+1, syntax.Pos{})
	case *syntax.Subshell:
		f.walkStmts(command.Stmts, depth+1, syntax.Pos{})
	case *syntax.BinaryCmd:
		f.walkStmt(command.X, depth)
		f.walkStmt(command.Y, depth)
	case *syntax.FuncDecl:
		f.walkStmt(command.Body, depth+1)
	case *syntax.TimeClause:
		f.walkStmt(command.Stmt, depth)
	}
}

func (f *bashDisplayFormatter) walkIfClause(clause *syntax.IfClause, depth int) {
	f.walkStmts(clause.Cond, depth, clause.ThenPos)
	f.afterKeyword(clause.ThenPos, "then", depth+1)
	nextKeyword := clause.FiPos
	if clause.Else != nil {
		nextKeyword = clause.Else.Position
	}
	f.walkStmts(clause.Then, depth+1, nextKeyword)
	if clause.Else != nil {
		f.beforeEdit(clause.Else.Position.Offset(), depth)
		f.walkIfClause(clause.Else, depth)
		if !clause.Else.ThenPos.IsValid() {
			f.afterKeyword(clause.Else.Position, "else", depth+1)
		}
	}
	f.beforeKeyword(clause.FiPos, depth)
}

func (f *bashDisplayFormatter) before(pos, next syntax.Pos) bool {
	if !next.IsValid() || pos.Offset() >= next.Offset() {
		return false
	}
	for offset := int(pos.Offset()) + 1; offset < int(next.Offset()); offset++ {
		if f.source[offset] != ' ' && f.source[offset] != '\t' {
			return false
		}
	}
	return true
}

func (f *bashDisplayFormatter) afterKeyword(pos syntax.Pos, keyword string, depth int) {
	if pos.IsValid() {
		f.after(pos.Offset()+uint(len(keyword)), depth)
	}
}

func (f *bashDisplayFormatter) beforeKeyword(pos syntax.Pos, depth int) {
	if pos.IsValid() {
		f.beforeEdit(pos.Offset(), depth)
	}
}

func (f *bashDisplayFormatter) after(offset uint, depth int) {
	start := int(offset)
	end := start
	for end < len(f.source) && (f.source[end] == ' ' || f.source[end] == '\t') {
		end++
	}
	f.edits = append(f.edits, bashDisplayEdit{start: start, end: end, text: "\n" + strings.Repeat("  ", depth)})
}

func (f *bashDisplayFormatter) beforeEdit(offset uint, depth int) {
	start := int(offset)
	for start > 0 && (f.source[start-1] == ' ' || f.source[start-1] == '\t') {
		start--
	}
	f.edits = append(f.edits, bashDisplayEdit{start: start, end: int(offset), text: "\n" + strings.Repeat("  ", depth)})
}

func (f *bashDisplayFormatter) apply() string {
	if len(f.edits) == 0 {
		return f.source
	}
	sort.SliceStable(f.edits, func(left, right int) bool {
		return f.edits[left].start < f.edits[right].start
	})

	formatted := f.source
	for index := len(f.edits) - 1; index >= 0; index-- {
		edit := f.edits[index]
		formatted = formatted[:edit.start] + edit.text + formatted[edit.end:]
	}
	return formatted
}
