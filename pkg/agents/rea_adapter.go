package agents

import (
	"os/exec"
	"strings"
)

type ReaAdapter struct{}

func NewReaAdapter() *ReaAdapter {
	return &ReaAdapter{}
}

func (a *ReaAdapter) Key() string {
	return "rea"
}

func (a *ReaAdapter) Name() string {
	return "REA Reverse Engineering (rea)"
}

func (a *ReaAdapter) Command() string {
	if _, err := exec.LookPath("rea"); err == nil {
		return "rea"
	}
	if _, err := exec.LookPath("npx"); err == nil {
		return "npx rea-agents"
	}
	return "rea"
}

func (a *ReaAdapter) Description() string {
	return "Reverse engineer anything — inspect binaries, Electron apps, .NET assemblies, Android APKs, websites, and databases without source code using AI agents and MCP tools."
}

func (a *ReaAdapter) Capabilities() []string {
	return []string{
		"reverse-engineering",
		"binary-analysis",
		"javascript-inspection",
		"electron-analysis",
		"android-apk",
		"website-inspection",
		"sqlite-analysis",
		"mcp-tools",
	}
}

func (a *ReaAdapter) Detect() (bool, string, string) {
	path, err := exec.LookPath("rea")
	if err == nil {
		cmd := exec.Command(path, "--version")
		out, err := cmd.Output()
		version := ""
		if err == nil {
			version = strings.TrimSpace(string(out))
		}
		return true, path, version
	}

	path, err = exec.LookPath("npx")
	if err == nil {
		cmd := exec.Command(path, "rea-agents", "--version")
		out, err := cmd.Output()
		version := ""
		if err == nil {
			version = strings.TrimSpace(string(out))
		}
		return true, path, version
	}

	return false, "", ""
}

func (a *ReaAdapter) BuildArgs(prompt string, projectPath string) []string {
	target := projectPath
	if target == "" {
		target = "."
	}
	if _, err := exec.LookPath("rea"); err != nil {
		if _, err := exec.LookPath("npx"); err == nil {
			return []string{"-y", "rea-agents@latest", "analyze-javascript-application", target, "--json"}
		}
	}
	return []string{"analyze-javascript-application", target, "--json"}
}
