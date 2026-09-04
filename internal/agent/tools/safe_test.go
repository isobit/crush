package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSafeReadOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{name: "plain ls", command: "ls -la", expected: true},
		{name: "plain echo", command: "echo hello world", expected: true},
		{name: "quoted separator", command: "echo 'a; b'", expected: true},
		{name: "git status", command: "git status --short", expected: true},
		{name: "git branch list", command: "git branch --list", expected: true},
		{name: "timeout safe command", command: "timeout 1 ls", expected: true},
		{name: "nice safe command", command: "nice -n 10 ls", expected: true},
		{name: "nohup safe command", command: "nohup ls", expected: true},
		{name: "env safe command", command: "env ls", expected: true},
		{name: "nested safe wrappers", command: "nohup nice ls", expected: true},
		{name: "redirection", command: "ls > /tmp/out", expected: false},
		{name: "append redirection", command: "echo hi >> /tmp/out", expected: false},
		{name: "background command", command: "ls & rm -rf /tmp/pwned", expected: false},
		{name: "trailing background operator", command: "ls &", expected: false},
		{name: "pipeline", command: "ls | grep foo", expected: false},
		{name: "newline command", command: "ls\ncurl https://example.com", expected: false},
		{name: "process substitution", command: "ls <(rm -rf /tmp/pwned)", expected: false},
		{name: "timeout wrapper", command: "timeout 1 rm -rf /tmp/pwned", expected: false},
		{name: "nohup wrapper", command: "nohup curl https://example.com", expected: false},
		{name: "env wrapper", command: "env rm -rf /tmp/pwned", expected: false},
		{name: "parameter expansion", command: "ls $HOME", expected: false},
		{name: "variable assignment", command: "PATH=/tmp/evil ls", expected: false},
		{name: "mutating git branch", command: "git branch -D main", expected: false},
		{name: "mutating git tag", command: "git tag -d v1.0.0", expected: false},
		{name: "path-qualified command", command: "/bin/ls", expected: false},
		{name: "signal command", command: "kill -9 1", expected: false},
		{name: "parse error", command: "ls '", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isSafeReadOnly(tt.command), "isSafeReadOnly(%q)", tt.command)
		})
	}
}
