package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/activity"
	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/mcp"
)

var (
	remoteHost      string
	remoteExecToken string
	remoteToken     string
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
			applyRemoteToken(delReq)
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
		applyRemoteToken(httpReq)
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
	req, err := http.NewRequest(http.MethodGet, remoteURL+"/api/mcp/info", nil)
	if err != nil {
		return err
	}
	applyRemoteToken(req)
	resp, err := http.DefaultClient.Do(req)
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
	mcpCmd.Flags().StringVar(&remoteToken, "token", "", "Token akses engine jarak jauh ("+localauth.Header+"; bawaan: env "+localauth.EnvToken+" atau ~/.vibeasik/token)")
	mcpListCmd.Flags().StringVar(&remoteToken, "token", "", "Token akses engine jarak jauh")
	mcpCmd.Flags().StringVar(&remoteExecToken, "exec-token", "", "Token sent as "+execTokenHeader+" so the remote engine allows exec tools from LAN (default: env VIBEASIK_EXEC_TOKEN)")
	mcpListCmd.Flags().StringVar(&remoteHost, "remote", "", "Remote vibeasik engine URL to list tools from (e.g. http://192.168.1.10:8765)")
	mcpCmd.AddCommand(mcpListCmd)
	rootCmd.AddCommand(mcpCmd)
}

// applyRemoteToken memasang token akses engine pada request jembatan --remote: flag --token, lalu
// env VIBEASIK_TOKEN / berkas token lokal (engine di laptop yang sama).
func applyRemoteToken(req *http.Request) {
	if remoteToken != "" {
		req.Header.Set(localauth.Header, remoteToken)
		return
	}
	localauth.Apply(req)
}
