package agents

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/activity"
	"github.com/vibeasik/vibeasik/pkg/models"
)

type Registry struct {
	mu       sync.RWMutex
	adapters map[string]AgentAdapter
}

var globalRegistry *Registry
var once sync.Once

// GetRegistry returns the singleton plug-and-play agent registry
func GetRegistry() *Registry {
	once.Do(func() {
		globalRegistry = &Registry{
			adapters: make(map[string]AgentAdapter),
		}
		// Register default built-in adapters
		globalRegistry.Register(NewAgyAdapter())
		globalRegistry.Register(NewClaudeAdapter())
		globalRegistry.Register(NewAiderAdapter())
		globalRegistry.Register(NewReaAdapter())
	})
	return globalRegistry
}

func (r *Registry) Register(adapter AgentAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.Key()] = adapter
}

func (r *Registry) Get(key string) (AgentAdapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[key]
	return a, ok
}

// DiscoverAll returns live statuses for all built-in and user-custom registered agents
func (r *Registry) DiscoverAll(db *gorm.DB) []models.AgentStatusDTO {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []models.AgentStatusDTO

	// 1. Built-in adapters
	for _, a := range r.adapters {
		detected, path, version := a.Detect()
		list = append(list, models.AgentStatusDTO{
			Key:          a.Key(),
			Name:         a.Name(),
			Command:      a.Command(),
			Detected:     detected,
			Path:         path,
			Version:      version,
			Description:  a.Description(),
			Capabilities: a.Capabilities(),
			IsBuiltin:    true,
		})
	}

	// 2. Custom database agents
	if db != nil {
		var customs []models.LocalAgentConfig
		if err := db.Where("enabled = ?", true).Find(&customs).Error; err == nil {
			for _, c := range customs {
				path, err := exec.LookPath(c.Command)
				detected := err == nil
				list = append(list, models.AgentStatusDTO{
					Key:          c.Key,
					Name:         c.Name,
					Command:      c.Command,
					Detected:     detected,
					Path:         path,
					Version:      "Custom Plug-in",
					Description:  c.Description,
					Capabilities: []string{"custom-cli", "terminal-execution"},
					IsBuiltin:    false,
				})
			}
		}
	}

	return list
}

// Execute runs the requested local agent on the target project directory
func (r *Registry) Execute(ctx context.Context, req models.AgentExecutionRequest, db *gorm.DB) (*models.AgentExecutionResult, error) {
	startTime := time.Now()

	// Default timeout
	timeout := 120 * time.Second
	if req.TimeoutSec > 0 {
		timeout = time.Duration(req.TimeoutSec) * time.Second
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var execPath string
	var args []string

	// Check if it's built-in
	if adapter, ok := r.Get(req.AgentKey); ok {
		detected, path, _ := adapter.Detect()
		if !detected {
			return nil, fmt.Errorf("agent '%s' tidak terdeteksi pada sistem (perintah '%s' tidak ditemukan di PATH)", adapter.Name(), adapter.Command())
		}
		execPath = path
		args = adapter.BuildArgs(req.Prompt, req.ProjectPath)
	} else if db != nil {
		// Check custom agent from DB
		var custom models.LocalAgentConfig
		if err := db.Where("key = ?", req.AgentKey).First(&custom).Error; err != nil {
			return nil, fmt.Errorf("agent dengan key '%s' tidak ditemukan", req.AgentKey)
		}
		path, err := exec.LookPath(custom.Command)
		if err != nil {
			return nil, fmt.Errorf("perintah '%s' tidak ditemukan di PATH sistem", custom.Command)
		}
		execPath = path

		// Build args from template with robust quote-aware prompt substitution
		args = splitArgs(custom.ArgsTemplate, req.Prompt)
	} else {
		return nil, fmt.Errorf("agent '%s' tidak dikenali", req.AgentKey)
	}

	// Validate project path
	workDir := req.ProjectPath
	if workDir == "" {
		workDir = "."
	}
	if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
		// Fallback to current working directory
		workDir = "."
	}

	done := activity.BeginLocalAgent(req.AgentKey, req.Prompt, req.ProjectPath)

	cmd := exec.CommandContext(execCtx, execPath, args...)
	cmd.Dir = workDir

	// Inherit environment plus standard PATH
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	duration := time.Since(startTime).Round(time.Millisecond).String()

	exitCode := 0
	success := true
	if err != nil {
		success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	outputText := string(out)
	if outputText == "" && !success {
		outputText = err.Error()
	}
	errMsg := ""
	if !success {
		errMsg = fmt.Sprintf("exit code %d", exitCode)
	}
	done(success, errMsg)

	return &models.AgentExecutionResult{
		AgentKey: req.AgentKey,
		Success:  success,
		Output:   outputText,
		ExitCode: exitCode,
		Duration: duration,
		Message:  fmt.Sprintf("Eksekusi selesai dalam %s dengan status code %d", duration, exitCode),
	}, nil
}

func splitArgs(template, prompt string) []string {
	template = strings.TrimSpace(template)
	if template == "" || template == "{{prompt}}" {
		return []string{prompt}
	}

	var rawTokens []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(template); i++ {
		ch := template[i]
		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
		} else if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
		} else if ch == ' ' && !inSingleQuote && !inDoubleQuote {
			if current.Len() > 0 {
				rawTokens = append(rawTokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		rawTokens = append(rawTokens, current.String())
	}

	var finalArgs []string
	hasPrompt := false
	for _, t := range rawTokens {
		if strings.Contains(t, "{{prompt}}") {
			replaced := strings.ReplaceAll(t, "{{prompt}}", prompt)
			finalArgs = append(finalArgs, replaced)
			hasPrompt = true
		} else {
			finalArgs = append(finalArgs, t)
		}
	}
	if !hasPrompt {
		finalArgs = append(finalArgs, prompt)
	}

	return finalArgs
}

// SplitArgs memecah args_template agent custom (quote-aware) dan menyisipkan prompt di {{prompt}}.
func SplitArgs(template, prompt string) []string {
	return splitArgs(template, prompt)
}
