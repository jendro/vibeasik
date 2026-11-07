package agents

import (
	"os/exec"
	"strings"
)

type ClaudeAdapter struct{}

func NewClaudeAdapter() *ClaudeAdapter {
	return &ClaudeAdapter{}
}

func (a *ClaudeAdapter) Key() string {
	return "claude"
}

func (a *ClaudeAdapter) Name() string {
	return "Claude Code CLI (claude)"
}

func (a *ClaudeAdapter) Command() string {
	return "claude"
}

func (a *ClaudeAdapter) Description() string {
	return "Anthropic CLI agent specialized in codebase exploration, refactoring, and terminal automation."
}

func (a *ClaudeAdapter) Capabilities() []string {
	return []string{"cli-interactive", "code-editing", "git-automation", "claude-3-7-sonnet"}
}

func (a *ClaudeAdapter) Detect() (bool, string, string) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return false, "", ""
	}

	cmd := exec.Command(path, "--version")
	out, err := cmd.Output()
	version := ""
	if err == nil {
		version = strings.TrimSpace(string(out))
	}

	return true, path, version
}

func (a *ClaudeAdapter) BuildArgs(prompt string, projectPath string) []string {
	// Claude code non-interactive prompt mode: claude -p "<prompt>"
	return []string{"-p", prompt}
}
