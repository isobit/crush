package tools

import (
	"runtime"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

type safeCommand struct {
	argv          []string
	restrictFlags bool
	allowFlags    []string
	denyFlags     []string
	requireFlag   bool
	allowOperands bool
}

var gitCodeExecFlags = []string{
	"--ext-diff",
	"--open-files-in-pager",
	"-O",
	"--output",
	"--output-indicator-new",
	"--upload-pack",
	"--receive-pack",
	"--exec",
}

var safeCommands = []safeCommand{
	// Bash builtins and core utils
	{argv: []string{"cal"}, allowOperands: true},
	{argv: []string{"date"}, denyFlags: []string{"-s", "--set"}, allowOperands: true},
	{argv: []string{"df"}, allowOperands: true},
	{argv: []string{"du"}, allowOperands: true},
	{argv: []string{"echo"}, allowOperands: true},
	{argv: []string{"env"}, allowOperands: false},
	{argv: []string{"free"}, allowOperands: true},
	{argv: []string{"groups"}, allowOperands: true},
	{argv: []string{"hostname"}, restrictFlags: true, allowFlags: []string{"-f", "--fqdn", "-s", "--short", "-d", "--domain", "-i", "--ip-address", "-I", "--all-ip-addresses", "-A"}},
	{argv: []string{"id"}, allowOperands: true},
	{argv: []string{"ls"}, allowOperands: true},
	{argv: []string{"printenv"}, allowOperands: true},
	{argv: []string{"ps"}, allowOperands: true},
	{argv: []string{"pwd"}},
	{argv: []string{"top"}, allowOperands: true},
	{argv: []string{"type"}, allowOperands: true},
	{argv: []string{"uname"}, allowOperands: true},
	{argv: []string{"uptime"}, allowOperands: true},
	{argv: []string{"whatis"}, allowOperands: true},
	{argv: []string{"whereis"}, allowOperands: true},
	{argv: []string{"which"}, allowOperands: true},
	{argv: []string{"whoami"}},

	// Git
	{argv: []string{"git", "blame"}, denyFlags: gitCodeExecFlags, allowOperands: true},
	{argv: []string{"git", "describe"}, denyFlags: gitCodeExecFlags, allowOperands: true},
	{argv: []string{"git", "diff"}, denyFlags: gitCodeExecFlags, allowOperands: true},
	{argv: []string{"git", "grep"}, denyFlags: gitCodeExecFlags, allowOperands: true},
	{argv: []string{"git", "log"}, denyFlags: gitCodeExecFlags, allowOperands: true},
	{argv: []string{"git", "ls-files"}, denyFlags: gitCodeExecFlags, allowOperands: true},
	{argv: []string{"git", "rev-parse"}, denyFlags: gitCodeExecFlags, allowOperands: true},
	{argv: []string{"git", "shortlog"}, denyFlags: gitCodeExecFlags, allowOperands: true},
	{argv: []string{"git", "show"}, denyFlags: gitCodeExecFlags, allowOperands: true},
	{argv: []string{"git", "status"}, denyFlags: gitCodeExecFlags, allowOperands: true},
	{argv: []string{"git", "branch"}, restrictFlags: true, allowFlags: []string{"-l", "--list", "-a", "--all", "-r", "--remotes", "-v", "-vv", "--verbose", "--show-current", "--format", "--sort"}},
	{argv: []string{"git", "tag"}, restrictFlags: true, allowFlags: []string{"-l", "--list", "-n", "--contains", "--no-contains", "--merged", "--no-merged", "--points-at", "--format", "--sort"}},
	{argv: []string{"git", "remote"}, restrictFlags: true, allowFlags: []string{"-v", "--verbose"}},
	{argv: []string{"git", "config"}, restrictFlags: true, allowFlags: []string{"--get", "--get-all", "--list", "-l"}, requireFlag: true, allowOperands: true},
}

func init() {
	if runtime.GOOS == "windows" {
		safeCommands = append(
			safeCommands,
			// Windows-specific commands
			safeCommand{argv: []string{"ipconfig"}, allowOperands: true},
			safeCommand{argv: []string{"systeminfo"}, allowOperands: true},
			safeCommand{argv: []string{"tasklist"}, allowOperands: true},
			safeCommand{argv: []string{"where"}, allowOperands: true},
		)
	}
}

func isSafeReadOnly(command string) bool {
	file, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil || len(file.Stmts) != 1 {
		return false
	}
	stmt := file.Stmts[0]
	if stmt == nil || stmt.Cmd == nil || len(stmt.Redirs) > 0 || stmt.Background || stmt.Coprocess || stmt.Negated || stmt.Disown {
		return false
	}
	call, ok := stmt.Cmd.(*syntax.CallExpr)
	if !ok || len(call.Assigns) > 0 || len(call.Args) == 0 {
		return false
	}
	argv, ok := literalArgs(call.Args)
	return ok && safeArgv(argv, 0)
}

func safeArgv(argv []string, depth int) bool {
	if len(argv) == 0 {
		return false
	}
	if inner, ok := peelWrapper(argv); ok {
		if depth >= maxWrapperDepth {
			return false
		}
		return safeArgv(inner, depth+1)
	}
	return slices.ContainsFunc(safeCommands, func(command safeCommand) bool {
		return command.matches(argv)
	})
}

type commandWrapper struct {
	name         string
	valueFlags   []string
	skipOperands int
}

var commandWrappers = []commandWrapper{
	{name: "env", valueFlags: []string{"-u", "--unset", "-C", "--chdir", "-S", "--split-string"}},
	{name: "nohup"},
	{name: "nice", valueFlags: []string{"-n", "--adjustment"}},
	{name: "timeout", skipOperands: 1, valueFlags: []string{"-s", "--signal", "-k", "--kill-after"}},
}

const maxWrapperDepth = 4

func peelWrapper(argv []string) ([]string, bool) {
	idx := slices.IndexFunc(commandWrappers, func(wrapper commandWrapper) bool {
		return wrapper.name == argv[0]
	})
	if idx < 0 {
		return nil, false
	}
	wrapper := commandWrappers[idx]
	rest := argv[1:]
	operandsSkipped := 0
	for len(rest) > 0 {
		token := rest[0]
		switch {
		case token == "--":
			rest = rest[1:]
			if len(rest) == 0 {
				return nil, false
			}
			return rest, true
		case isFlag(token):
			consumesValue := slices.Contains(wrapper.valueFlags, flagName(token)) && !strings.Contains(token, "=")
			rest = rest[1:]
			if consumesValue {
				if len(rest) == 0 {
					return nil, false
				}
				rest = rest[1:]
			}
		case strings.Contains(token, "=") && wrapper.name == "env":
			return nil, false
		case operandsSkipped < wrapper.skipOperands:
			operandsSkipped++
			rest = rest[1:]
		default:
			return rest, true
		}
	}
	return nil, false
}

func (command safeCommand) matches(argv []string) bool {
	if len(argv) < len(command.argv) || !slices.Equal(argv[:len(command.argv)], command.argv) {
		return false
	}
	rest := argv[len(command.argv):]
	operandsOnly := false
	sawFlag := false
	for _, arg := range rest {
		if arg == "--" {
			operandsOnly = true
			continue
		}
		if !operandsOnly && isFlag(arg) {
			for _, name := range flagNames(arg) {
				if command.restrictFlags {
					if !slices.Contains(command.allowFlags, name) {
						return false
					}
				} else if slices.Contains(command.denyFlags, name) {
					return false
				}
			}
			sawFlag = true
			continue
		}
		if !command.allowOperands {
			return false
		}
	}
	return sawFlag || !command.requireFlag
}

func isFlag(token string) bool {
	return len(token) > 1 && strings.HasPrefix(token, "-") && token != "--"
}

func flagName(token string) string {
	name, _, _ := strings.Cut(token, "=")
	return name
}

func flagNames(token string) []string {
	name := flagName(token)
	if strings.HasPrefix(name, "--") || len(name) <= 2 {
		return []string{name}
	}
	names := []string{name}
	for _, char := range name[1:] {
		names = append(names, "-"+string(char))
	}
	return names
}

func literalArgs(words []*syntax.Word) ([]string, bool) {
	args := make([]string, 0, len(words))
	for _, word := range words {
		value, ok := literalWord(word)
		if !ok {
			return nil, false
		}
		args = append(args, value)
	}
	return args, true
}

func literalWord(word *syntax.Word) (string, bool) {
	var value strings.Builder
	for _, part := range word.Parts {
		switch part := part.(type) {
		case *syntax.Lit:
			value.WriteString(part.Value)
		case *syntax.SglQuoted:
			if part.Dollar {
				return "", false
			}
			value.WriteString(part.Value)
		case *syntax.DblQuoted:
			if part.Dollar {
				return "", false
			}
			for _, nested := range part.Parts {
				literal, ok := nested.(*syntax.Lit)
				if !ok {
					return "", false
				}
				value.WriteString(literal.Value)
			}
		default:
			return "", false
		}
	}
	return value.String(), true
}
