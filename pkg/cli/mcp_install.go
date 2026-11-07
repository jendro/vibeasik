package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// Agent coding yang bisa didaftari vibeasik MCP. Konfigurasi hanya ditulis bila aplikasinya terdeteksi
// (foldernya sudah ada) — vibeasik tidak membuat folder aplikasi yang tidak Anda pasang.
type mcpAgent struct {
	key    string
	name   string
	detect func(home string) (path string, ok bool) // lokasi berkas konfigurasi JSON "mcpServers"
	cli    string                                   // bila diisi: daftarkan lewat CLI agent ini (bukan berkas)
}

func jsonAgentAt(rel ...string) func(string) (string, bool) {
	return func(home string) (string, bool) {
		p := filepath.Join(append([]string{home}, rel...)...)
		_, err := os.Stat(filepath.Dir(p))
		return p, err == nil
	}
}

func claudeDesktopPath(home string) (string, bool) {
	var p string
	switch runtime.GOOS {
	case "darwin":
		p = filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		app := os.Getenv("APPDATA")
		if app == "" {
			app = filepath.Join(home, "AppData", "Roaming")
		}
		p = filepath.Join(app, "Claude", "claude_desktop_config.json")
	default:
		p = filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	}
	_, err := os.Stat(filepath.Dir(p))
	return p, err == nil
}

func mcpAgents() []mcpAgent {
	vs := func(ext string) func(string) (string, bool) {
		return jsonAgentAt(".config", "Code", "User", "globalStorage", ext, "settings", "cline_mcp_settings.json")
	}
	return []mcpAgent{
		{key: "claude-code", name: "Claude Code (CLI)", cli: "claude"},
		{key: "claude-desktop", name: "Claude Desktop", detect: claudeDesktopPath},
		{key: "cursor", name: "Cursor", detect: jsonAgentAt(".cursor", "mcp.json")},
		{key: "windsurf", name: "Windsurf", detect: jsonAgentAt(".codeium", "windsurf", "mcp_config.json")},
		{key: "antigravity", name: "Google Antigravity (agy)", cli: "agy"},
		{key: "cline", name: "VS Code · Cline", detect: func(h string) (string, bool) {
			p, _ := vs("saoudrizwan.claude-dev")(h)
			_, err := os.Stat(filepath.Dir(filepath.Dir(p)))
			return p, err == nil
		}},
		{key: "roo", name: "VS Code · Roo Code", detect: func(h string) (string, bool) {
			p, _ := vs("rooveterinaryinc.roo-cline")(h)
			_, err := os.Stat(filepath.Dir(filepath.Dir(p)))
			return p, err == nil
		}},
	}
}

var (
	mcpDryRun    bool
	mcpOnly      string
	mcpUninstall bool
)

// vibeasikCommand path perintah yang ditulis ke konfigurasi agent: `vibeasik` di PATH bila ada (tetap berlaku
// setelah pembaruan), selain itu binary yang sedang berjalan.
func vibeasikCommand() (string, error) {
	if p, err := exec.LookPath("vibeasik"); err == nil {
		if abs, err := filepath.Abs(p); err == nil {
			return abs, nil
		}
	}
	return os.Executable()
}

var mcpInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Daftarkan vibeasik MCP ke agent coding yang terpasang (Claude Code, Cursor, Windsurf, Antigravity, Cline, …)",
	Long: `Mendaftarkan server MCP vibeasik (stdio: "vibeasik mcp") ke agent coding yang terdeteksi di komputer ini.

  • Hanya agent yang aplikasinya sudah ada yang disentuh; folder baru tidak dibuat.
  • Berkas konfigurasi dicadangkan ke <berkas>.vibeasik.bak sebelum diubah (sekali).
  • Aman dijalankan berulang. Pakai --dry-run untuk melihat apa yang akan dilakukan.
  • Hapus pendaftaran dengan "vibeasik mcp install --uninstall".

MCP stdio tidak butuh token: dibaca dari ~/.vibeasik/token.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		command, err := vibeasikCommand()
		if err != nil {
			return err
		}
		only := map[string]bool{}
		for _, k := range strings.Split(mcpOnly, ",") {
			if k = strings.TrimSpace(k); k != "" {
				only[k] = true
			}
		}
		verb := "Mendaftarkan"
		if mcpUninstall {
			verb = "Menghapus pendaftaran"
		}
		fmt.Printf("%s vibeasik MCP (%s mcp)%s\n\n", verb, command, map[bool]string{true: " — DRY RUN", false: ""}[mcpDryRun])

		done := 0
		for _, a := range mcpAgents() {
			if len(only) > 0 && !only[a.key] {
				continue
			}
			msg, ok := applyAgent(a, home, command)
			icon := "·"
			if ok {
				icon, done = "✓", done+1
			}
			fmt.Printf("  %s %-26s %s\n", icon, a.name, msg)
		}
		if done == 0 {
			fmt.Println("\nTidak ada agent yang terdeteksi. Daftarkan manual: vibeasik mcp  (command stdio) di konfigurasi agent Anda.")
			return nil
		}
		if !mcpDryRun && !mcpUninstall {
			fmt.Println("\nSelesai. Restart agent Anda agar tool vibeasik muncul.")
		}
		return nil
	},
}

func applyAgent(a mcpAgent, home, command string) (string, bool) {
	if a.cli != "" {
		bin, err := exec.LookPath(a.cli)
		if err != nil {
			return "tidak terpasang", false
		}
		var args []string
		switch {
		case a.key == "claude-code" && mcpUninstall:
			args = []string{"mcp", "remove", "--scope", "user", "vibeasik"}
		case a.key == "claude-code":
			args = []string{"mcp", "add", "--scope", "user", "vibeasik", "--", command, "mcp"}
		case mcpUninstall:
			args = []string{"mcp", "remove", "vibeasik"}
		default:
			args = []string{"mcp", "add", "vibeasik", "--", command, "mcp"}
		}
		if mcpDryRun {
			return bin + " " + strings.Join(args, " "), true
		}
		out, err := exec.Command(bin, args...).CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err != nil {
			if strings.Contains(strings.ToLower(text), "already exists") {
				return "sudah terdaftar", true
			}
			return "gagal: " + firstLine(text), false
		}
		return "terdaftar lewat CLI", true
	}
	path, ok := a.detect(home)
	if !ok {
		return "tidak terpasang", false
	}
	changed, err := editMCPConfig(path, command, mcpUninstall, mcpDryRun)
	switch {
	case err != nil:
		return "dilewati: " + err.Error(), false
	case !changed && mcpUninstall:
		return "tidak terdaftar", false
	case !changed:
		return "sudah terdaftar (" + path + ")", true
	case mcpDryRun:
		return "akan diubah: " + path, true
	}
	return path, true
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// editMCPConfig menambah/menghapus entri "vibeasik" di mcpServers tanpa menyentuh entri lain.
// Mengembalikan changed=false bila tidak ada yang perlu diubah.
func editMCPConfig(path, command string, remove, dry bool) (bool, error) {
	data := map[string]any{}
	existing, readErr := os.ReadFile(path)
	if readErr == nil && len(strings.TrimSpace(string(existing))) > 0 {
		if err := json.Unmarshal(existing, &data); err != nil {
			return false, fmt.Errorf("%s bukan JSON valid (tidak diubah)", filepath.Base(path))
		}
	}
	servers, _ := data["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	want := map[string]any{"command": command, "args": []any{"mcp"}}
	cur, has := servers["vibeasik"]
	if remove {
		if !has {
			return false, nil
		}
		delete(servers, "vibeasik")
	} else {
		if has && sameEntry(cur, want) {
			return false, nil
		}
		servers["vibeasik"] = want
	}
	if dry {
		return true, nil
	}
	data["mcpServers"] = servers
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return false, err
	}
	if readErr == nil {
		bak := path + ".vibeasik.bak"
		if _, err := os.Stat(bak); os.IsNotExist(err) {
			if err := os.WriteFile(bak, existing, 0o600); err != nil {
				return false, err
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	return true, os.WriteFile(path, append(out, '\n'), mode)
}

func sameEntry(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func init() {
	mcpInstallCmd.Flags().BoolVar(&mcpDryRun, "dry-run", false, "Tampilkan rencana tanpa mengubah apa pun")
	mcpInstallCmd.Flags().BoolVar(&mcpUninstall, "uninstall", false, "Hapus pendaftaran vibeasik dari agent")
	keys := []string{}
	for _, a := range mcpAgents() {
		keys = append(keys, a.key)
	}
	sort.Strings(keys)
	mcpInstallCmd.Flags().StringVar(&mcpOnly, "agent", "", "Batasi ke agent tertentu: "+strings.Join(keys, ", "))
	mcpCmd.AddCommand(mcpInstallCmd)
}
