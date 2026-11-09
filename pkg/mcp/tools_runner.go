package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/runner"
)

// Runner tools let AI agents allocate stable ports and start/stop dev servers through vibeasik.
// In stdio mode the calls are forwarded to the running daemon (SetDaemonURL) so the processes are
// owned by the engine the studio UI sees, not by the short-lived MCP process.

// SetDaemonURL forwards runner tools to a running local engine (e.g. http://127.0.0.1:8765).
// Alamatnya juga disimpan di tingkat paket agar tool modul (registry.ToolProvider) bisa meneruskan ke daemon
// lewat DaemonURL/DaemonRequest.
func (s *Server) SetDaemonURL(url string) {
	s.daemonURL = strings.TrimRight(url, "/")
	daemonBase.Store(s.daemonURL)
}

var daemonBase atomic.Value // string

// DaemonURL: alamat engine lokal yang sedang dipakai proses MCP stdio ini, "" bila tidak ada.
func DaemonURL() string {
	v, _ := daemonBase.Load().(string)
	return v
}

func (s *Server) runnerToolsList() []Tool {
	return []Tool{
		{
			Name:        "vibeasik_port_registry",
			Description: "Lists the local port registry: every component's declared port vs the unique run port vibeasik allocated, with live listening/run state. Use it before choosing a port to avoid collisions.",
			InputSchema: ToolSchema{Type: "object", Properties: map[string]interface{}{}},
		},
		{
			Name:        "vibeasik_assign_port",
			Description: "Allocates a stable, collision-free run port for a component (keeps the project's own port when unique, otherwise picks from 4100-4999). Pass port to set it manually or reassign=true to move to a new free port.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"component_id": prop("integer", "ID of the component"),
					"port":         prop("integer", "Optional manual port (1024-65535)"),
					"reassign":     prop("boolean", "Pick a different free port than the current one"),
				},
				Required: []string{"component_id"},
			},
		},
		{
			Name:        "vibeasik_run_component",
			Description: "Starts a component's dev command on its allocated port (PORT env + framework port flag injected), or every runnable component of a project when project_id is given. Returns URL and status; check vibeasik_run_status for logs.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"component_id":     prop("integer", "ID of the component to run"),
					"project_id":       prop("integer", "ID of the project (runs all runnable components, backends first)"),
					"reassign_if_busy": prop("boolean", "If the allocated port is held by a foreign process, move to a free port instead of failing"),
				},
			},
		},
		{
			Name:        "vibeasik_stop_component",
			Description: "Stops a dev server started by vibeasik (whole process tree), or all of a project's running components when project_id is given.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"component_id": prop("integer", "ID of the component to stop"),
					"project_id":   prop("integer", "ID of the project (stops all its running components)"),
				},
			},
		},
		{
			Name:        "vibeasik_run_status",
			Description: "Shows run state, URL, warnings (e.g. app ignores PORT) and recent log lines of a component started by vibeasik; without component_id lists every launched process.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"component_id": prop("integer", "ID of the component"),
					"log_lines":    prop("integer", "Number of trailing log lines to include (default 60)"),
				},
			},
		},
		{
			Name:        "vibeasik_ensure_unique_ports",
			Description: "Ensures every component in a project (or all projects) has a unique, collision-free local run port allocated (from 4100-4999 if colliding with another component or in-use port), so executive board and dev team can test features smoothly.",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": prop("integer", "Optional project ID; if omitted or 0, ensures unique ports across all projects"),
				},
			},
		},
	}
}

// runnerToolHandlers memetakan tool runner (port & proses komponen) ke penanganannya.
var runnerToolHandlers = map[string]toolHandler{
	"vibeasik_port_registry":       (*Server).toolPortRegistry,
	"vibeasik_assign_port":         (*Server).toolAssignPort,
	"vibeasik_run_component":       (*Server).toolRunComponent,
	"vibeasik_stop_component":      (*Server).toolStopComponent,
	"vibeasik_run_status":          (*Server).toolRunStatus,
	"vibeasik_ensure_unique_ports": (*Server).toolEnsureUniquePorts,
}

// callRunnerTool returns handled=false when name is not a runner tool.
func (s *Server) callRunnerTool(name string, args map[string]interface{}) (*ToolCallResult, bool, error) {
	h, ok := runnerToolHandlers[name]
	if !ok {
		return nil, false, nil
	}
	res, err := h(s, args)
	return res, true, err
}

func (s *Server) toolPortRegistry(args map[string]interface{}) (*ToolCallResult, error) {
	if s.daemonURL != "" {
		return s.daemonCall(http.MethodGet, "/api/ports/registry", nil)
	}
	entries, err := s.runner.Registry()
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"entries": entries, "range": []int{runner.RangeStart, runner.RangeEnd}})
}

func (s *Server) toolAssignPort(args map[string]interface{}) (*ToolCallResult, error) {
	cid := argUint(args, "component_id")
	if cid == 0 {
		return nil, fmt.Errorf("valid component_id is required")
	}
	port := int(argUint(args, "port"))
	force, _ := args["reassign"].(bool)
	if s.daemonURL != "" {
		return s.daemonCall(http.MethodPost, fmt.Sprintf("/api/components/%d/port", cid), map[string]interface{}{"port": port, "reassign": force})
	}
	comp, err := s.runner.AssignPort(cid, port, force)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"component_id": comp.ID, "run_port": comp.RunPort})
}

// componentOrProjectPath memilih path daemon: komponen bila cid ada, jika tidak proyek.
func componentOrProjectPath(cid, pid uint, action string) string {
	if cid == 0 {
		return fmt.Sprintf("/api/projects/%d/%s", pid, action)
	}
	return fmt.Sprintf("/api/components/%d/%s", cid, action)
}

func (s *Server) toolRunComponent(args map[string]interface{}) (*ToolCallResult, error) {
	cid := argUint(args, "component_id")
	pid := argUint(args, "project_id")
	reassign, _ := args["reassign_if_busy"].(bool)
	if cid == 0 && pid == 0 {
		return nil, fmt.Errorf("component_id or project_id is required")
	}
	if s.daemonURL != "" {
		return s.daemonCall(http.MethodPost, componentOrProjectPath(cid, pid, "run"), map[string]interface{}{"reassign": reassign})
	}
	var out interface{}
	var err error
	if cid > 0 {
		out, err = s.runner.Run(cid, reassign)
	} else {
		out, err = s.runner.RunProject(pid, reassign)
	}
	if err != nil {
		return nil, err
	}
	return jsonResult(out)
}

func (s *Server) toolStopComponent(args map[string]interface{}) (*ToolCallResult, error) {
	cid := argUint(args, "component_id")
	pid := argUint(args, "project_id")
	if cid == 0 && pid == 0 {
		return nil, fmt.Errorf("component_id or project_id is required")
	}
	if s.daemonURL != "" {
		return s.daemonCall(http.MethodPost, componentOrProjectPath(cid, pid, "stop"), nil)
	}
	var out interface{}
	var err error
	if cid > 0 {
		out, err = s.runner.Stop(cid)
	} else {
		out, err = s.runner.StopProject(pid)
	}
	if err != nil {
		return nil, err
	}
	return jsonResult(out)
}

func (s *Server) toolRunStatus(args map[string]interface{}) (*ToolCallResult, error) {
	cid := argUint(args, "component_id")
	lines := int(argUint(args, "log_lines"))
	if lines <= 0 {
		lines = 60
	}
	if cid == 0 {
		if s.daemonURL != "" {
			return s.daemonCall(http.MethodGet, "/api/runs", nil)
		}
		return jsonResult(map[string]interface{}{"runs": s.runner.List()})
	}
	st, err := s.componentRunStatus(cid)
	if err != nil {
		return nil, err
	}
	if len(st.Logs) > lines {
		st.Logs = st.Logs[len(st.Logs)-lines:]
	}
	return jsonResult(st)
}

// componentRunStatus membaca status proses komponen dari daemon lokal atau runner in-process.
func (s *Server) componentRunStatus(cid uint) (runner.Status, error) {
	var st runner.Status
	if s.daemonURL == "" {
		return s.runner.Status(cid, true)
	}
	raw, err := s.daemonRequest(http.MethodGet, fmt.Sprintf("/api/components/%d/run", cid), nil)
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(raw, &st); err != nil {
		return st, err
	}
	return st, nil
}

func (s *Server) toolEnsureUniquePorts(args map[string]interface{}) (*ToolCallResult, error) {
	pid := argUint(args, "project_id")
	if s.daemonURL != "" {
		path := "/api/ports/ensure-unique"
		if pid > 0 {
			path = fmt.Sprintf("/api/projects/%d/ensure-unique-ports", pid)
		}
		return s.daemonCall(http.MethodPost, path, nil)
	}
	updated, err := s.runner.EnsureUniquePorts(pid)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{
		"message":       "Port unik lokal berhasil dialokasikan",
		"updated_count": len(updated),
		"components":    updated,
	})
}

func (s *Server) daemonCall(method, path string, body interface{}) (*ToolCallResult, error) {
	return DaemonCall(s.daemonURL, method, path, body)
}

// DaemonCall meneruskan satu panggilan ke engine lokal di base dan membungkus jawabannya sebagai hasil tool.
func DaemonCall(base, method, path string, body interface{}) (*ToolCallResult, error) {
	raw, err := DaemonRequest(base, method, path, body)
	if err != nil {
		return nil, err
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, raw, "", "  ") != nil {
		pretty.Reset()
		pretty.Write(raw)
	}
	return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: pretty.String()}}}, nil
}

func (s *Server) daemonRequest(method, path string, body interface{}) ([]byte, error) {
	return DaemonRequest(s.daemonURL, method, path, body)
}

// DaemonRequest mengirim request bertoken lokal ke engine di base dan mengembalikan body mentahnya.
func DaemonRequest(base, method, path string, body interface{}) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, base+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	localauth.Apply(req)
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("local engine tidak bisa dihubungi: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(raw, &e) == nil && e.Error != "" {
			return nil, errors.New(e.Error)
		}
		return nil, fmt.Errorf("local engine HTTP %d", resp.StatusCode)
	}
	return raw, nil
}
