package agents

import (
	"os/exec"
	"strings"
)

type AiderAdapter struct{}

func NewAiderAdapter() *AiderAdapter {
	return &AiderAdapter{}
}

func (a *AiderAdapter) Key() string {
	return "aider"
}

func (a *AiderAdapter) Name() string {
	return "Aider AI (aider)"
}

func (a *AiderAdapter) Command() string {
	return "aider"
}

func (a *AiderAdapter) Description() string {
	return "AI pair programming tool in your terminal with fast edits and multi-file context."
}

func (a *AiderAdapter) Capabilities() []string {
	return []string{"pair-programming", "auto-commits", "diff-editing"}
}

func (a *AiderAdapter) Detect() (bool, string, string) {
	path, err := exec.LookPath("aider")
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

func (a *AiderAdapter) BuildArgs(prompt string, projectPath string) []string {
	return []string{"--message", prompt, "--no-auto-commits", "--yes-always"}
}
