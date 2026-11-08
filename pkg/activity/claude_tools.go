package activity

import (
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"
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
		return "Mengedit " + util.FirstNonEmpty(base("file_path"), base("notebook_path"), "file")
	case "Read":
		return "Membaca " + util.FirstNonEmpty(base("file_path"), "file")
	case "Bash":
		return "Menjalankan: " + util.FirstNonEmpty(str("description"), str("command"))
	case "Grep", "Glob":
		return "Mencari " + util.FirstNonEmpty(str("pattern"), "di kode")
	case "WebFetch", "WebSearch":
		return "Browsing " + util.FirstNonEmpty(str("url"), str("query"), "web")
	case "Task", "Agent":
		return "Mendelegasikan ke subagent: " + util.FirstNonEmpty(str("description"), "tugas")
	case "TodoWrite", "TaskCreate", "TaskUpdate":
		return "Memperbarui daftar tugas"
	}
	if strings.HasPrefix(tool, "mcp__") {
		parts := strings.Split(tool, "__")
		return "MCP: " + parts[len(parts)-1]
	}
	return "Memakai " + util.FirstNonEmpty(tool, "tool")
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
				return util.FirstNonEmpty(form, content)
			}
		}
	case "TaskUpdate":
		if s, _ := in["status"].(string); s == "in_progress" {
			form, _ := in["activeForm"].(string)
			subject, _ := in["subject"].(string)
			return util.FirstNonEmpty(form, subject)
		}
	}
	return ""
}
