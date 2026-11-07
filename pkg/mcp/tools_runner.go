package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/runner"
)

// Runner tools let AI agents allocate stable ports and start/stop dev servers through vibeasik.
// In stdio mode the calls are forwarded to the running daemon (SetDaemonURL) so the processes are
// owned by the engine the studio UI sees, not by the short-lived MCP process.

// SetDaemonURL forwards runner tools to a running local engine (e.g. http://127.0.0.1:8765).
func (s *Server) SetDaemonURL(url string) {
	s.daemonURL = strings.TrimRight(url, "/")
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

// callRunnerTool returns handled=false when name is not a runner tool.
func (s *Server) callRunnerTool(name string, args map[string]interface{}) (*ToolCallResult, bool, error) {
	cid := argUint(args, "component_id")
	pid := argUint(args, "project_id")
	reassign, _ := args["reassign_if_busy"].(bool)

	switch name {
	case "vibeasik_port_registry":
		if s.daemonURL != "" {
			res, err := s.daemonCall(http.MethodGet, "/api/ports/registry", nil)
			return res, true, err
		}
		entries, err := s.runner.Registry()
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(map[string]interface{}{"entries": entries, "range": []int{runner.RangeStart, runner.RangeEnd}})
		return res, true, err

	case "vibeasik_assign_port":
		if cid == 0 {
			return nil, true, fmt.Errorf("valid component_id is required")
		}
		port := int(argUint(args, "port"))
		force, _ := args["reassign"].(bool)
		if s.daemonURL != "" {
			res, err := s.daemonCall(http.MethodPost, fmt.Sprintf("/api/components/%d/port", cid), map[string]interface{}{"port": port, "reassign": force})
			return res, true, err
		}
		comp, err := s.runner.AssignPort(cid, port, force)
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(map[string]interface{}{"component_id": comp.ID, "run_port": comp.RunPort})
		return res, true, err

	case "vibeasik_run_component":
		if cid == 0 && pid == 0 {
			return nil, true, fmt.Errorf("component_id or project_id is required")
		}
		body := map[string]interface{}{"reassign": reassign}
		if s.daemonURL != "" {
			path := fmt.Sprintf("/api/components/%d/run", cid)
			if cid == 0 {
				path = fmt.Sprintf("/api/projects/%d/run", pid)
			}
			res, err := s.daemonCall(http.MethodPost, path, body)
			return res, true, err
		}
		var out interface{}
		var err error
		if cid > 0 {
			out, err = s.runner.Run(cid, reassign)
		} else {
			out, err = s.runner.RunProject(pid, reassign)
		}
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(out)
		return res, true, err

	case "vibeasik_stop_component":
		if cid == 0 && pid == 0 {
			return nil, true, fmt.Errorf("component_id or project_id is required")
		}
		if s.daemonURL != "" {
			path := fmt.Sprintf("/api/components/%d/stop", cid)
			if cid == 0 {
				path = fmt.Sprintf("/api/projects/%d/stop", pid)
			}
			res, err := s.daemonCall(http.MethodPost, path, nil)
			return res, true, err
		}
		var out interface{}
		var err error
		if cid > 0 {
			out, err = s.runner.Stop(cid)
		} else {
			out, err = s.runner.StopProject(pid)
		}
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(out)
		return res, true, err

	case "vibeasik_run_status":
		lines := int(argUint(args, "log_lines"))
		if lines <= 0 {
			lines = 60
		}
		if cid == 0 {
			if s.daemonURL != "" {
				res, err := s.daemonCall(http.MethodGet, "/api/runs", nil)
				return res, true, err
			}
			res, err := jsonResult(map[string]interface{}{"runs": s.runner.List()})
			return res, true, err
		}
		var st runner.Status
		if s.daemonURL != "" {
			raw, err := s.daemonRequest(http.MethodGet, fmt.Sprintf("/api/components/%d/run", cid), nil)
			if err != nil {
				return nil, true, err
			}
			if err := json.Unmarshal(raw, &st); err != nil {
				return nil, true, err
			}
		} else {
			var err error
			if st, err = s.runner.Status(cid, true); err != nil {
				return nil, true, err
			}
		}
		if len(st.Logs) > lines {
			st.Logs = st.Logs[len(st.Logs)-lines:]
		}
		res, err := jsonResult(st)
		return res, true, err

	case "vibeasik_ensure_unique_ports":
		if s.daemonURL != "" {
			path := "/api/ports/ensure-unique"
			if pid > 0 {
				path = fmt.Sprintf("/api/projects/%d/ensure-unique-ports", pid)
			}
			res, err := s.daemonCall(http.MethodPost, path, nil)
			return res, true, err
		}
		updated, err := s.runner.EnsureUniquePorts(pid)
		if err != nil {
			return nil, true, err
		}
		res, err := jsonResult(map[string]interface{}{
			"message":       "Port unik lokal berhasil dialokasikan",
			"updated_count": len(updated),
			"components":    updated,
		})
		return res, true, err
	}
	return nil, false, nil
}

func (s *Server) daemonCall(method, path string, body interface{}) (*ToolCallResult, error) {
	raw, err := s.daemonRequest(method, path, body)
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
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, s.daemonURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
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
