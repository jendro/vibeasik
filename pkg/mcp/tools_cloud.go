package mcp

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/agents"
	"github.com/vibeasik/vibeasik/pkg/cloud"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func (s *Server) toolTutorials(args map[string]interface{}) (*ToolCallResult, error) {
	category, _ := args["category"].(string)
	tuts, err := cloud.NewClient().GetTutorials(category)
	if err != nil {
		return nil, err
	}
	return prettyJSONText(tuts), nil
}

func (s *Server) toolSearchDomain(args map[string]interface{}) (*ToolCallResult, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}
	domains, err := cloud.NewClient().SearchDomains(query)
	if err != nil {
		return nil, err
	}
	return prettyJSONText(domains), nil
}

func (s *Server) toolDeployPlan(args map[string]interface{}) (*ToolCallResult, error) {
	plan, err := cloud.NewClient().GenerateDeployPlan(args)
	if err != nil {
		return nil, err
	}
	return prettyJSONText(plan), nil
}

func (s *Server) toolRunLocalAgent(args map[string]interface{}) (*ToolCallResult, error) {
	agentKey, _ := args["agent_key"].(string)
	prompt, _ := args["prompt"].(string)
	projectPath, _ := args["project_path"].(string)
	req := models.AgentExecutionRequest{
		AgentKey:    agentKey,
		Prompt:      prompt,
		ProjectPath: projectPath,
		TimeoutSec:  120,
	}
	res, err := agents.GetRegistry().Execute(context.Background(), req, s.db)
	if err != nil {
		return nil, err
	}
	return prettyJSONText(res), nil
}

func (s *Server) toolAITemplates(args map[string]interface{}) (*ToolCallResult, error) {
	tmpls, err := cloud.NewClient().GetAITemplates()
	if err != nil {
		return nil, err
	}
	return prettyJSONText(tmpls), nil
}

func (s *Server) toolInjectAITemplate(args map[string]interface{}) (*ToolCallResult, error) {
	tmplID, _ := args["template_id"].(string)
	targetPath, _ := args["target_path"].(string)
	if tmplID == "" || targetPath == "" {
		return nil, fmt.Errorf("template_id and target_path are required")
	}
	tmpls, err := cloud.NewClient().GetAITemplates()
	if err != nil {
		return nil, err
	}
	matched := findAITemplate(tmpls, tmplID)
	if matched == nil {
		return nil, fmt.Errorf("template %s not found", tmplID)
	}
	createdFiles, err := writeAITemplateFiles(matched, targetPath)
	if err != nil {
		return nil, err
	}
	return textResult(fmt.Sprintf("Template %s injected successfully into %s. Created files: %s", matched.Name, targetPath, strings.Join(createdFiles, ", "))), nil
}

func findAITemplate(tmpls []cloud.AITemplateDTO, id string) *cloud.AITemplateDTO {
	for i := range tmpls {
		if tmpls[i].ID == id {
			return &tmpls[i]
		}
	}
	return nil
}

// writeAITemplateFiles menulis berkas template ke targetPath dan mengembalikan nama berkas yang dibuat.
func writeAITemplateFiles(tmpl *cloud.AITemplateDTO, targetPath string) ([]string, error) {
	var created []string
	for _, f := range tmpl.Files {
		fullPath := filepath.Join(targetPath, filepath.FromSlash(f.Filename))
		_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err := os.WriteFile(fullPath, []byte(f.Content), 0644); err != nil {
			return nil, err
		}
		created = append(created, f.Filename)
	}
	return created, nil
}

func (s *Server) toolCheckPorts(args map[string]interface{}) (*ToolCallResult, error) {
	portsRaw, ok := args["ports"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("ports array required")
	}
	results := make(map[string]bool)
	for _, p := range portsRaw {
		if port := portNumber(p); port > 0 {
			results[strconv.Itoa(port)] = portInUse(port)
		}
	}
	return prettyJSONText(results), nil
}

func portNumber(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

func portInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 150*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (s *Server) toolREAStatus(args map[string]interface{}) (*ToolCallResult, error) {
	adapter, ok := agents.GetRegistry().Get("rea")
	if !ok {
		return textResult(`{"installed": false, "message": "REA adapter not registered"}`), nil
	}
	detected, path, version := adapter.Detect()
	status := map[string]interface{}{
		"installed":    detected,
		"path":         path,
		"version":      version,
		"capabilities": adapter.Capabilities(),
		"install_cmd":  "npm install --global rea-agents",
		"setup_cmd":    "npx rea-agents setup",
		"description":  adapter.Description(),
	}
	if !detected {
		status["message"] = "REA belum terpasang. Jalankan: npm install --global rea-agents && npx rea-agents setup"
	}
	return prettyJSONText(status), nil
}

// reaSubcommand memetakan jenis analisis REA ke subperintah CLI-nya (default: aplikasi JavaScript).
var reaSubcommand = map[string]string{
	"native": "analyze", "binary": "analyze",
	"android": "analyze", "apk": "analyze",
	"sqlite": "analyze", "database": "analyze",
}

// reaCommand menyusun perintah REA: biner `rea` langsung, jika tidak ada lewat npx.
func reaCommand(analysisType, targetPath string) (string, []string, error) {
	var cmdArgs []string
	cmdName, err := exec.LookPath("rea")
	if err != nil {
		cmdName, err = exec.LookPath("npx")
		if err != nil {
			return "", nil, fmt.Errorf("REA belum terpasang. Jalankan: npm install --global rea-agents")
		}
		cmdArgs = append(cmdArgs, "-y", "rea-agents@latest")
	}
	sub, ok := reaSubcommand[analysisType]
	if !ok {
		sub = "analyze-javascript-application"
	}
	return cmdName, append(cmdArgs, sub, targetPath, "--json"), nil
}

func (s *Server) toolREAAnalyze(args map[string]interface{}) (*ToolCallResult, error) {
	targetPath, _ := args["target_path"].(string)
	if targetPath == "" {
		return nil, fmt.Errorf("target_path is required")
	}
	analysisType, _ := args["analysis_type"].(string)
	if analysisType == "" {
		analysisType = "javascript"
	}
	cmdName, cmdArgs, err := reaCommand(analysisType, targetPath)
	if err != nil {
		return nil, err
	}

	execCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(execCtx, cmdName, cmdArgs...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	outputText := string(out)
	if err != nil && outputText == "" {
		outputText = fmt.Sprintf("REA analysis error: %s", err.Error())
	}
	return textResult(outputText), nil
}
