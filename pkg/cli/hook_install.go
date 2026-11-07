package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var hookSettingsPath string

// hookEvents adalah event hook Claude Code yang diteruskan ke Kantor Agent ("" = tanpa matcher).
var hookEvents = []struct{ event, matcher string }{
	{"SessionStart", ""},
	{"UserPromptSubmit", ""},
	{"PreToolUse", "*"},
	{"Notification", ""},
	{"Stop", ""},
	{"SessionEnd", ""},
}

var hookInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Register `vibeasik hook claude-code` in Claude Code settings (~/.claude/settings.json)",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := hookSettingsPath
		if path == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			path = filepath.Join(home, ".claude", "settings.json")
		}
		exe, err := os.Executable()
		if err != nil {
			exe = "vibeasik"
		}
		added, err := installClaudeHooks(path, fmt.Sprintf("%q hook claude-code", exe))
		if err != nil {
			return err
		}
		if added == 0 {
			fmt.Printf("✅ Hook Kantor Agent sudah terpasang di %s\n", path)
			return nil
		}
		fmt.Printf("✅ %d hook Kantor Agent ditambahkan ke %s\n", added, path)
		fmt.Println("   Sesi Claude Code baru akan tampil di Studio → Kantor Agent (/agent-office).")
		return nil
	},
}

// installClaudeHooks menggabungkan hook ke file settings tanpa mengubah konfigurasi lain (idempoten).
func installClaudeHooks(path, command string) (int, error) {
	settings := map[string]interface{}{}
	original, err := os.ReadFile(path)
	if err == nil && len(strings.TrimSpace(string(original))) > 0 {
		if err := json.Unmarshal(original, &settings); err != nil {
			return 0, fmt.Errorf("gagal membaca %s (JSON tidak valid): %w", path, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return 0, err
	}

	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
	}
	added := 0
	for _, h := range hookEvents {
		groups, _ := hooks[h.event].([]interface{})
		if hasHookCommand(groups, "hook claude-code") {
			continue
		}
		group := map[string]interface{}{
			"hooks": []interface{}{map[string]interface{}{"type": "command", "command": command, "timeout": 5}},
		}
		if h.matcher != "" {
			group["matcher"] = h.matcher
		}
		hooks[h.event] = append(groups, group)
		added++
	}
	if added == 0 {
		return 0, nil
	}
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}
	if len(original) > 0 {
		_ = os.WriteFile(path+".bak", original, 0o644)
	}
	return added, os.WriteFile(path, append(out, '\n'), 0o644)
}

func hasHookCommand(groups []interface{}, needle string) bool {
	for _, g := range groups {
		group, _ := g.(map[string]interface{})
		list, _ := group["hooks"].([]interface{})
		for _, h := range list {
			item, _ := h.(map[string]interface{})
			if c, _ := item["command"].(string); strings.Contains(c, needle) {
				return true
			}
		}
	}
	return false
}

func init() {
	hookInstallCmd.Flags().StringVar(&hookSettingsPath, "settings", "", "Claude Code settings file (default ~/.claude/settings.json)")
}
