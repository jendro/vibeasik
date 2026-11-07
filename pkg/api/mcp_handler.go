package api

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/activity"
	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/mcp"
)

const headerMCPSession = "Mcp-Session-Id"

func (s *Server) handleMCPPost(c *fiber.Ctx) error {
	var req mcp.JSONRPCRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   &mcp.RPCError{Code: -32700, Message: "Parse error: " + err.Error()},
		})
	}

	// Must run before the notification branch: a tools/call without an ID is still executed.
	if req.Method == "tools/call" {
		var params mcp.ToolCallParams
		if json.Unmarshal(req.Params, &params) == nil && mcp.ToolRunsCommands(params.Name, params.Arguments) {
			if d := execDecisionFor(c); !d.Allowed {
				return c.Status(fiber.StatusForbidden).JSON(mcp.JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &mcp.RPCError{Code: -32001, Message: fmt.Sprintf("Tool '%s' ditolak. %s", params.Name, d.Reason), Data: fiber.Map{"code": d.Code}},
				})
			}
		}
	}

	caller := mcpCaller(c, req.Method)

	// Notifications have no ID and must not receive a response
	if req.ID == nil || strings.HasPrefix(req.Method, "notifications/") {
		s.mcpServer.HandleRPCFrom(req, caller)
		return c.SendStatus(fiber.StatusNoContent)
	}

	resp := s.mcpServer.HandleRPCFrom(req, caller)

	// If this request belongs to an SSE session (SSE transport)
	sessionID := c.Query("sessionId")
	if sessionID != "" {
		sm := mcp.GetSessionManager()
		if session, ok := sm.GetSession(sessionID); ok {
			data, err := json.Marshal(resp)
			if err == nil {
				select {
				case session.Messages <- data:
					return c.Status(fiber.StatusAccepted).SendString("Accepted")
				default:
				}
			}
		}
	}

	// Direct HTTP JSON-RPC response
	return c.JSON(resp)
}

// mcpCaller mengidentifikasi agent HTTP untuk Kantor Agent: sesi SSE, header Mcp-Session-Id
// (diterbitkan saat initialize, lalu dikirim ulang klien sesuai spesifikasi MCP), atau IP sebagai cadangan.
func mcpCaller(c *fiber.Ctx, method string) activity.Caller {
	if id := c.Query("sessionId"); id != "" {
		return activity.Caller{Key: "mcp-sse:" + id, Transport: "sse"}
	}
	id := c.Get(headerMCPSession)
	if id == "" && method == "initialize" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		id = hex.EncodeToString(b)
		c.Set(headerMCPSession, id)
	}
	if id != "" {
		return activity.Caller{Key: "mcp-http:" + id, Transport: "http"}
	}
	return activity.Caller{Key: "mcp-http-ip:" + c.IP(), Transport: "http"}
}

// handleMCPDelete menutup sesi MCP HTTP (spesifikasi Streamable HTTP): agent pulang dari kantor.
func (s *Server) handleMCPDelete(c *fiber.Ctx) error {
	if id := c.Get(headerMCPSession); id != "" {
		activity.ClientDisconnected(activity.Caller{Key: "mcp-http:" + id})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) handleMCPSSE(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	sm := mcp.GetSessionManager()
	session := sm.CreateSession()

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer sm.RemoveSession(session.ID)
		defer activity.ClientDisconnected(activity.Caller{Key: "mcp-sse:" + session.ID})

		// Send endpoint event with session ID
		endpointEvent := fmt.Sprintf("event: endpoint\ndata: /mcp?sessionId=%s\n\n", session.ID)
		if _, err := w.WriteString(endpointEvent); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-session.Messages:
				if !ok {
					return
				}
				eventData := fmt.Sprintf("event: message\ndata: %s\n\n", string(msg))
				if _, err := w.WriteString(eventData); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				if _, err := w.WriteString(": keepalive\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-session.Done:
				return
			}
		}
	})
	return nil
}

func (s *Server) getMCPInfo(c *fiber.Ctx) error {
	execPath, _ := os.Executable()
	binaryName := filepath.Base(execPath)
	inPath := false
	if _, err := exec.LookPath("vibeasik"); err == nil {
		inPath = true
	}

	host := c.Hostname()
	if strings.Contains(host, ":") {
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
	}
	if host == "" {
		host = "localhost"
	}
	cfg := config.GetConfig()
	httpURL := fmt.Sprintf("http://%s:%d/mcp", host, cfg.Port)
	sseURL := fmt.Sprintf("http://%s:%d/mcp/sse", host, cfg.Port)

	cmdStr := "vibeasik"
	if !inPath && execPath != "" {
		cmdStr = execPath
	}

	claudeConfig := fiber.Map{
		"mcpServers": fiber.Map{
			"vibeasik": fiber.Map{
				"command": cmdStr,
				"args":    []string{"mcp"},
			},
		},
	}

	cursorConfig := fiber.Map{
		"mcpServers": fiber.Map{
			"vibeasik": fiber.Map{
				"command": cmdStr,
				"args":    []string{"mcp"},
			},
		},
	}

	httpServer := fiber.Map{"url": httpURL}
	sseServer := fiber.Map{"url": sseURL}
	remoteArgs := []string{"mcp", "--remote", fmt.Sprintf("http://%s:%d", host, cfg.Port)}
	execTokenSet := os.Getenv(envExecToken) != ""
	if execTokenSet {
		// Placeholder saja: endpoint ini terbuka untuk LAN, jadi nilai token tidak boleh ikut terkirim.
		headers := fiber.Map{headerExecToken: "<isi VIBEASIK_EXEC_TOKEN>"}
		httpServer["headers"] = headers
		sseServer["headers"] = headers
	}
	httpConfig := fiber.Map{"mcpServers": fiber.Map{"vibeasik": httpServer}}
	sseConfig := fiber.Map{"mcpServers": fiber.Map{"vibeasik": sseServer}}
	remoteConfig := fiber.Map{"mcpServers": fiber.Map{"vibeasik": fiber.Map{
		"command": "vibeasik",
		"args":    remoteArgs,
		"env":     fiber.Map{envExecToken: "<isi VIBEASIK_EXEC_TOKEN>"},
	}}}

	tools := s.mcpServer.GetToolsList()

	return c.JSON(fiber.Map{
		"status":                      "active",
		"binary_path":                 execPath,
		"binary_name":                 binaryName,
		"in_path":                     inPath,
		"os":                          runtime.GOOS,
		"arch":                        runtime.GOARCH,
		"http_url":                    httpURL,
		"sse_url":                     sseURL,
		"tools_count":                 len(tools),
		"exec_token_required_for_lan": execTokenSet,
		"tools":                       tools,
		"snippets": fiber.Map{
			"claude":      claudeConfig,
			"cursor":      cursorConfig,
			"http":        httpConfig,
			"sse":         sseConfig,
			"remote":      remoteConfig,
			"agy_command": fmt.Sprintf("agy mcp add vibeasik -- %s mcp", cmdStr),
			"claude_cli":  fmt.Sprintf("claude mcp add vibeasik -- %s mcp", cmdStr),
		},
	})
}
