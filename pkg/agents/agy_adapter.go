package agents

import (
	"os/exec"
	"strings"
)

type AgyAdapter struct{}

func NewAgyAdapter() *AgyAdapter {
	return &AgyAdapter{}
}

func (a *AgyAdapter) Key() string {
	return "agy"
}

func (a *AgyAdapter) Name() string {
	return "Google Antigravity CLI (agy)"
}

func (a *AgyAdapter) Command() string {
	return "agy"
}

func (a *AgyAdapter) Description() string {
	return "Autonomous AI coding agent with subagents, deep reasoning, and MCP tool execution."
}

func (a *AgyAdapter) Capabilities() []string {
	return []string{"autonomous-coding", "subagent-delegation", "mcp-tools", "gemini-models"}
}

func (a *AgyAdapter) Detect() (bool, string, string) {
	path, err := exec.LookPath("agy")
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

func (a *AgyAdapter) BuildArgs(prompt string, projectPath string) []string {
	// agy non-interactive print mode requires -p / --print
	return []string{"-p", prompt}
}
