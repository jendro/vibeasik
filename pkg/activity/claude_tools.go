package activity

import (
	"path/filepath"
	"strings"
)

// Helper bersama untuk event Claude Code: dipakai `vibeasik hook claude-code` dan watcher transkrip.

// ClaudeToolAction membuat kalimat singkat tentang tool yang sedang dipakai agent.
func ClaudeToolAction(tool string, in map[string]interface{}) string {
	str := func(k string) string { v, _ := in[k].(string); return strings.TrimSpace(v) }
	base := func(k string) string {
		if v := str(k); v != "" {
			return filepath.Base(v)
		}
		return ""
	}
	switch tool {
	case "Edit", "MultiEdit", "Write", "NotebookEdit":
		return "Mengedit " + firstNonEmpty(base("file_path"), base("notebook_path"), "file")
	case "Read":
		return "Membaca " + firstNonEmpty(base("file_path"), "file")
	case "Bash":
		return "Menjalankan: " + firstNonEmpty(str("description"), str("command"))
	case "Grep", "Glob":
		return "Mencari " + firstNonEmpty(str("pattern"), "di kode")
	case "WebFetch", "WebSearch":
		return "Browsing " + firstNonEmpty(str("url"), str("query"), "web")
	case "Task", "Agent":
		return "Mendelegasikan ke subagent: " + firstNonEmpty(str("description"), "tugas")
	case "TodoWrite", "TaskCreate", "TaskUpdate":
		return "Memperbarui daftar tugas"
	}
	if strings.HasPrefix(tool, "mcp__") {
		parts := strings.Split(tool, "__")
		return "MCP: " + parts[len(parts)-1]
	}
	return "Memakai " + firstNonEmpty(tool, "tool")
}

// ClaudeActiveTask mengambil item yang sedang dikerjakan dari daftar tugas internal agent.
func ClaudeActiveTask(tool string, in map[string]interface{}) string {
	switch tool {
	case "TodoWrite":
		todos, _ := in["todos"].([]interface{})
		for _, t := range todos {
			item, _ := t.(map[string]interface{})
			if s, _ := item["status"].(string); s == "in_progress" {
				form, _ := item["activeForm"].(string)
				content, _ := item["content"].(string)
				return firstNonEmpty(form, content)
			}
		}
	case "TaskUpdate":
		if s, _ := in["status"].(string); s == "in_progress" {
			form, _ := in["activeForm"].(string)
			subject, _ := in["subject"].(string)
			return firstNonEmpty(form, subject)
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
