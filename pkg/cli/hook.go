package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/activity"
	"github.com/vibeasik/vibeasik/pkg/config"
)

var hookAgentName string

// hookCmd meneruskan event hook agent coding (format hook Claude Code) ke Kantor Agent.
// Selalu keluar dengan kode 0 dan tanpa output: output stdout hook bisa masuk ke konteks agent,
// dan kegagalan menghubungi engine tidak boleh menghambat kerja agent.
var hookCmd = &cobra.Command{
	Use:   "hook [agent]",
	Short: "Report coding-agent hook events (Claude Code format) to the vibeasik Agent Office",
	Long: `Reads one hook event JSON from stdin (Claude Code hooks: SessionStart, UserPromptSubmit,
PreToolUse, Notification, Stop, SessionEnd) and reports it to the local engine so the agent shows up
live as a character in Studio → Kantor Agent (/agent-office).

Register it with: vibeasik hook install`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := hookAgentName
		if len(args) == 1 && name == "" {
			name = args[0]
		}
		if name == "" {
			name = "claude-code"
		}
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
		if err != nil {
			return nil
		}
		report, ok := hookEventToReport(name, raw)
		if !ok {
			return nil
		}
		body, _ := json.Marshal(report)
		url := fmt.Sprintf("http://127.0.0.1:%d/api/agents/activity", config.GetConfig().Port)
		if remote := os.Getenv("VIBEASIK_REMOTE_URL"); remote != "" {
			url = strings.TrimRight(remote, "/") + "/api/agents/activity"
		}
		client := &http.Client{Timeout: 1500 * time.Millisecond}
		if resp, err := client.Post(url, "application/json", bytes.NewReader(body)); err == nil {
			resp.Body.Close()
		}
		return nil
	},
}

type hookEvent struct {
	SessionID     string                 `json:"session_id"`
	Cwd           string                 `json:"cwd"`
	HookEventName string                 `json:"hook_event_name"`
	Prompt        string                 `json:"prompt"`
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	Message       string                 `json:"message"`
	Model         interface{}            `json:"model"`
}

// hookEventToReport memetakan satu event hook ke payload POST /api/agents/activity.
func hookEventToReport(agent string, raw []byte) (map[string]interface{}, bool) {
	var ev hookEvent
	if err := json.Unmarshal(raw, &ev); err != nil || ev.HookEventName == "" {
		return nil, false
	}
	r := map[string]interface{}{
		"agent_name":   agent,
		"session_id":   ev.SessionID,
		"project_path": ev.Cwd,
		"transport":    "hook",
	}
	if ev.Cwd != "" {
		r["project_name"] = filepath.Base(ev.Cwd)
	}
	switch m := ev.Model.(type) {
	case string:
		r["model"] = m
	case map[string]interface{}:
		if id, ok := m["id"].(string); ok {
			r["model"] = id
		}
	}

	switch ev.HookEventName {
	case "SessionStart":
		r["status"], r["action"] = "idle", "Sesi dimulai"
	case "UserPromptSubmit":
		r["status"], r["action"], r["task"] = "working", "Membaca instruksi", ev.Prompt
	case "PreToolUse":
		r["status"], r["action"] = "working", activity.ClaudeToolAction(ev.ToolName, ev.ToolInput)
		if task := activity.ClaudeActiveTask(ev.ToolName, ev.ToolInput); task != "" {
			r["task"] = task
		}
	case "Notification":
		r["status"], r["action"] = "idle", firstNonEmpty(ev.Message, "Menunggu respons pengguna")
	case "Stop":
		// "idle", bukan "done": jumlah task selesai dihitung watcher transkrip agar tidak terhitung ganda.
		r["status"], r["action"] = "idle", "Menunggu instruksi berikutnya"
	case "SessionEnd":
		r["status"], r["action"] = "left", "Sesi berakhir"
	default:
		return nil, false
	}
	return r, true
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func init() {
	hookCmd.Flags().StringVar(&hookAgentName, "agent", "", "Agent name shown in the office (default: claude-code)")
	hookCmd.AddCommand(hookInstallCmd)
	rootCmd.AddCommand(hookCmd)
}
