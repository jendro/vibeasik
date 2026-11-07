package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/activity"
	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/mcp"
)

var (
	remoteHost      string
	remoteExecToken string
)

// execTokenHeader must match api.headerExecToken: the engine's exec guard accepts LAN callers that send
// the VIBEASIK_EXEC_TOKEN configured on the engine (e.g. to run vibeasik_runtime_simulation remotely).
const execTokenHeader = "X-Vibeasik-Exec-Token"

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run the Model Context Protocol (MCP) server over stdio for AI agents",
	Long: `Starts the stdio JSON-RPC loop adhering to the Model Context Protocol (MCP).
Allows autonomous AI coding agents (Antigravity, Cursor, Claude Code, Cline) to inspect
local project bundles, query component topologies, run diagnostics, and plan cross-stack features.
Can also act as a remote stdio proxy bridge when --remote <url> is passed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetRemote := remoteHost
		if targetRemote == "" {
			targetRemote = os.Getenv("VIBEASIK_REMOTE_URL")
		}

		if targetRemote != "" {
			token := remoteExecToken
			if token == "" {
				token = os.Getenv("VIBEASIK_EXEC_TOKEN")
			}
			return runRemoteStdioBridge(targetRemote, token, os.Stdin, os.Stdout)
		}

		db, err := database.InitDB(dbPath)
		if err != nil {
			return err
		}

		activity.Init(db)
		server := mcp.NewServer(db)
		// Bila daemon lokal hidup, proses dev server dijalankan olehnya (terlihat di studio & tidak mati bersama sesi MCP).
		daemonURL := fmt.Sprintf("http://127.0.0.1:%d", config.GetConfig().Port)
		if resp, err := (&http.Client{Timeout: 500 * time.Millisecond}).Get(daemonURL + "/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				server.SetDaemonURL(daemonURL)
			}
		}
		return server.RunStdioLoop(os.Stdin, os.Stdout)
	},
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tools provided by the vibeasik MCP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		targetRemote := remoteHost
		if targetRemote == "" {
			targetRemote = os.Getenv("VIBEASIK_REMOTE_URL")
		}

		if targetRemote != "" {
			return listRemoteTools(targetRemote)
		}

		db, err := database.InitDB(dbPath)
		if err != nil {
			return err
		}

		server := mcp.NewServer(db)
		tools := server.GetToolsList()

		fmt.Println("==================================================================")
		fmt.Printf("🛠️  vibeasik.com Model Context Protocol (MCP) Tools (%d Total)\n", len(tools))
		fmt.Println("==================================================================")
		for i, t := range tools {
			fmt.Printf("%2d. %-30s : %s\n", i+1, t.Name, t.Description)
		}
		fmt.Println("==================================================================")
		return nil
	},
}

var mcpInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Auto-register vibeasik MCP into Claude Desktop, Cursor, and Antigravity",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPAutoInstall()
	},
}

func runMCPAutoInstall() error {
	fmt.Println("==================================================================")
	fmt.Println("🚀 Installing & Registering vibeasik MCP Server for AI Agents...")
	fmt.Println("==================================================================")

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}

	cmdStr := "vibeasik"
	inPath := false
	if p, err := exec.LookPath("vibeasik"); err == nil {
		inPath = true
		cmdStr = p
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home directory: %w", err)
	}

	// 1. Install binary to ~/.local/bin if not in PATH
	if !inPath && runtime.GOOS != "windows" {
		localBin := filepath.Join(home, ".local", "bin")
		_ = os.MkdirAll(localBin, 0755)
		targetBin := filepath.Join(localBin, "vibeasik")
		if execPath != targetBin {
			_ = os.Remove(targetBin)
			if err := os.Symlink(execPath, targetBin); err == nil {
				fmt.Printf("✅ Symlinked executable to %s\n", targetBin)
				cmdStr = targetBin
			} else {
				// Fallback copy
				if copyErr := copyFile(execPath, targetBin); copyErr == nil {
					_ = os.Chmod(targetBin, 0755)
					fmt.Printf("✅ Copied executable to %s\n", targetBin)
					cmdStr = targetBin
				} else {
					cmdStr = execPath
				}
			}
		} else {
			cmdStr = targetBin
		}
	} else if !inPath {
		cmdStr = execPath
	}

	// 2. Register into Claude Desktop
	claudeConfigPath := getClaudeDesktopConfigPath(home)
	if claudeConfigPath != "" {
		if err := registerMCPConfig(claudeConfigPath, "vibeasik", cmdStr, []string{"mcp"}); err == nil {
			fmt.Printf("✅ Registered in Claude Desktop: %s\n", claudeConfigPath)
		} else {
			fmt.Printf("⚠️ Claude Desktop config update skipped: %v\n", err)
		}
	}

	// 3. Register into Cursor
	cursorConfigPath := filepath.Join(home, ".cursor", "mcp.json")
	if err := registerMCPConfig(cursorConfigPath, "vibeasik", cmdStr, []string{"mcp"}); err == nil {
		fmt.Printf("✅ Registered in Cursor: %s\n", cursorConfigPath)
	} else {
		fmt.Printf("⚠️ Cursor config update skipped: %v\n", err)
	}

	// 4. Register into Windsurf if available
	windsurfDir := filepath.Join(home, ".codeium", "windsurf")
	if _, err := os.Stat(windsurfDir); err == nil {
		windsurfConfigPath := filepath.Join(windsurfDir, "mcp_config.json")
		if err := registerMCPConfig(windsurfConfigPath, "vibeasik", cmdStr, []string{"mcp"}); err == nil {
			fmt.Printf("✅ Registered in Windsurf: %s\n", windsurfConfigPath)
		}
	}

	// 5. Register into Antigravity (agy CLI) if available
	if agyPath, err := exec.LookPath("agy"); err == nil {
		out, err := exec.Command(agyPath, "mcp", "add", "vibeasik", "--", cmdStr, "mcp").CombinedOutput()
		if err == nil {
			fmt.Println("✅ Registered in Google Antigravity (agy)")
		} else {
			fmt.Printf("ℹ️ Antigravity (agy) note: %s\n", string(out))
		}
	}

	// 6. Register into VS Code (Cline extension) if available
	clineConfigPath := filepath.Join(home, ".config", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json")
	clineDir := filepath.Dir(clineConfigPath)
	if _, err := os.Stat(filepath.Dir(clineDir)); err == nil {
		if err := registerMCPConfig(clineConfigPath, "vibeasik", cmdStr, []string{"mcp"}); err == nil {
			fmt.Printf("✅ Registered in VS Code (Cline): %s\n", clineConfigPath)
		}
	}

	// 7. Register into VS Code (Roo Code extension) if available
	rooConfigPath := filepath.Join(home, ".config", "Code", "User", "globalStorage", "rooveterinaryinc.roo-cline", "settings", "cline_mcp_settings.json")
	rooDir := filepath.Dir(rooConfigPath)
	if _, err := os.Stat(filepath.Dir(rooDir)); err == nil {
		if err := registerMCPConfig(rooConfigPath, "vibeasik", cmdStr, []string{"mcp"}); err == nil {
			fmt.Printf("✅ Registered in VS Code (Roo Code): %s\n", rooConfigPath)
		}
	}

	fmt.Println("==================================================================")
	fmt.Printf("🎉 vibeasik MCP is ready!\n")
	fmt.Printf("   Command: %s mcp\n", cmdStr)
	fmt.Printf("   AI agents can now access project catalogs, scan repos, and run diagnostics.\n")
	fmt.Println("==================================================================")
	return nil
}

func getClaudeDesktopConfigPath(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Claude", "claude_desktop_config.json")
	default: // linux and others
		return filepath.Join(home, ".config", "claude", "claude_desktop_config.json")
	}
}

func registerMCPConfig(configPath, serverName, command string, args []string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	dataMap := make(map[string]interface{})
	if b, err := os.ReadFile(configPath); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &dataMap)
	}

	mcpServers, ok := dataMap["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
		dataMap["mcpServers"] = mcpServers
	}

	mcpServers[serverName] = map[string]interface{}{
		"command": command,
		"args":    args,
	}

	outBytes, err := json.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, outBytes, 0644)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func runRemoteStdioBridge(remoteURL, execToken string, in io.Reader, out io.Writer) error {
	remoteURL = strings.TrimRight(remoteURL, "/")
	endpoint := remoteURL + "/mcp"

	client := &http.Client{Timeout: 60 * time.Second}
	reader := bufio.NewReader(in)

	// Sesi MCP dari engine (header Mcp-Session-Id) diteruskan agar agent ini tampil sebagai satu orang di Kantor Agent.
	sessionID := ""
	defer func() {
		if sessionID == "" {
			return
		}
		if delReq, err := http.NewRequest(http.MethodDelete, endpoint, nil); err == nil {
			delReq.Header.Set("Mcp-Session-Id", sessionID)
			if resp, err := client.Do(delReq); err == nil {
				resp.Body.Close()
			}
		}
	}()

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		line = []byte(strings.TrimSpace(string(line)))
		if len(line) == 0 {
			continue
		}

		var req map[string]interface{}
		_ = json.Unmarshal(line, &req)

		isNotification := false
		if _, hasID := req["id"]; !hasID {
			isNotification = true
		} else if method, ok := req["method"].(string); ok && strings.HasPrefix(method, "notifications/") {
			isNotification = true
		}

		httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(line))
		if err != nil {
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if execToken != "" {
			httpReq.Header.Set(execTokenHeader, execToken)
		}
		if sessionID != "" {
			httpReq.Header.Set("Mcp-Session-Id", sessionID)
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			if !isNotification {
				errResp := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      req["id"],
					"error": map[string]interface{}{
						"code":    -32603,
						"message": fmt.Sprintf("Failed to connect to remote vibeasik engine at %s: %v", remoteURL, err),
					},
				}
				b, _ := json.Marshal(errResp)
				out.Write(b)
				out.Write([]byte("\n"))
			}
			continue
		}

		if id := resp.Header.Get("Mcp-Session-Id"); id != "" {
			sessionID = id
		}
		if isNotification || resp.StatusCode == http.StatusNoContent {
			resp.Body.Close()
			continue
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err == nil && len(bodyBytes) > 0 {
			out.Write(bodyBytes)
			out.Write([]byte("\n"))
		}
	}
}

func listRemoteTools(remoteURL string) error {
	remoteURL = strings.TrimRight(remoteURL, "/")
	resp, err := http.Get(remoteURL + "/api/mcp/info")
	if err != nil {
		return fmt.Errorf("could not connect to remote vibeasik at %s: %w", remoteURL, err)
	}
	defer resp.Body.Close()

	var data struct {
		ToolsCount int        `json:"tools_count"`
		Tools      []mcp.Tool `json:"tools"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("failed to parse remote tools: %w", err)
	}

	fmt.Println("==================================================================")
	fmt.Printf("🌐 Remote vibeasik MCP Tools at %s (%d Total)\n", remoteURL, len(data.Tools))
	fmt.Println("==================================================================")
	for i, t := range data.Tools {
		fmt.Printf("%2d. %-30s : %s\n", i+1, t.Name, t.Description)
	}
	fmt.Println("==================================================================")
	return nil
}

func init() {
	mcpCmd.Flags().StringVar(&remoteHost, "remote", "", "Remote vibeasik engine URL to proxy MCP (e.g. http://192.168.1.10:8765)")
	mcpCmd.Flags().StringVar(&remoteExecToken, "exec-token", "", "Token sent as "+execTokenHeader+" so the remote engine allows exec tools from LAN (default: env VIBEASIK_EXEC_TOKEN)")
	mcpListCmd.Flags().StringVar(&remoteHost, "remote", "", "Remote vibeasik engine URL to list tools from (e.g. http://192.168.1.10:8765)")
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpInstallCmd)
	rootCmd.AddCommand(mcpCmd)
}
