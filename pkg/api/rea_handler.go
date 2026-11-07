package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/agents"
)

// getReaStatus returns the installation status and capabilities of REA
func (s *Server) getReaStatus(c *fiber.Ctx) error {
	reg := agents.GetRegistry()
	adapter, ok := reg.Get("rea")
	if !ok {
		return c.JSON(fiber.Map{
			"installed":   false,
			"message":     "REA adapter not registered",
			"install_cmd": "npm install --global rea-agents",
			"setup_cmd":   "npx rea-agents setup",
		})
	}

	detected, path, version := adapter.Detect()
	result := fiber.Map{
		"installed":    detected,
		"path":         path,
		"version":      version,
		"capabilities": adapter.Capabilities(),
		"description":  adapter.Description(),
		"install_cmd":  "npm install --global rea-agents",
		"setup_cmd":    "npx rea-agents setup",
	}
	if !detected {
		result["message"] = "REA belum terpasang. Jalankan: npm install --global rea-agents && npx rea-agents setup"
	}
	return c.JSON(result)
}

// runReaAnalysis runs a REA reverse engineering analysis on a target path
func (s *Server) runReaAnalysis(c *fiber.Ctx) error {
	var req struct {
		TargetPath   string `json:"target_path"`
		AnalysisType string `json:"analysis_type"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body: " + err.Error()})
	}
	if req.TargetPath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "target_path is required"})
	}
	if req.AnalysisType == "" {
		req.AnalysisType = "javascript"
	}

	// Build REA command
	var cmdName string
	var cmdArgs []string

	reaPath, err := exec.LookPath("rea")
	if err != nil {
		reaPath, err = exec.LookPath("npx")
		if err != nil {
			return c.Status(422).JSON(fiber.Map{
				"error":       "REA belum terpasang di sistem ini",
				"install_cmd": "npm install --global rea-agents",
			})
		}
		cmdName = reaPath
		cmdArgs = append(cmdArgs, "-y", "rea-agents@latest")
	} else {
		cmdName = reaPath
	}

	switch req.AnalysisType {
	case "javascript", "electron":
		cmdArgs = append(cmdArgs, "analyze-javascript-application", req.TargetPath, "--json")
	default:
		cmdArgs = append(cmdArgs, "analyze-javascript-application", req.TargetPath, "--json")
	}

	ctx, cancel := context.WithTimeout(c.Context(), 120*time.Second)
	defer cancel()

	startTime := time.Now()
	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	out, execErr := cmd.CombinedOutput()
	duration := time.Since(startTime).Round(time.Millisecond).String()

	outputText := string(out)
	if execErr != nil && outputText == "" {
		outputText = fmt.Sprintf("REA analysis error: %s", execErr.Error())
	}

	// Try to parse as JSON for structured response
	var parsed interface{}
	if json.Valid(out) {
		_ = json.Unmarshal(out, &parsed)
		return c.JSON(fiber.Map{
			"success":       execErr == nil,
			"analysis_type": req.AnalysisType,
			"target_path":   req.TargetPath,
			"duration":      duration,
			"result":        parsed,
		})
	}

	return c.JSON(fiber.Map{
		"success":       execErr == nil,
		"analysis_type": req.AnalysisType,
		"target_path":   req.TargetPath,
		"duration":      duration,
		"output":        outputText,
	})
}
