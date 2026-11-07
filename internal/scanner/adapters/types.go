package adapters

import (
	"os"
	"strings"

	"github.com/vibeasik/vibeasik/internal/models"
)

// RouterScanner is the common interface for all framework-specific scanners
type RouterScanner interface {
	Name() string
	Type() string // "backend" or "frontend"
	Detect(dir string) bool
	Scan(dir string) ([]models.RouteInfo, error)
}

// BackendAdapter specializes in backend REST API endpoint detection (Go, Laravel, Express, NestJS, FastAPI, etc.)
type BackendAdapter interface {
	RouterScanner
}

// FrontendAdapter specializes in frontend page routing and target API request detection (Nuxt, Next.js, React Router, Vue, SvelteKit, etc.)
type FrontendAdapter interface {
	RouterScanner
}

// Helper utilities for all adapters

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func shouldIgnoreName(name string) bool {
	ignored := map[string]bool{
		"node_modules": true,
		".git":         true,
		".nuxt":        true,
		".output":      true,
		".next":        true,
		"dist":         true,
		"build":        true,
		".cache":       true,
		"vendor":       true,
		".turbo":       true,
		".idea":        true,
		".vscode":      true,
		"tmp":          true,
		"coverage":     true,
	}
	return ignored[name] || strings.HasPrefix(name, ".")
}

func readFileString(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return string(b)
}

// CleanURLPath normalizes URL paths: trims redundant slashes, ensures leading slash, removes trailing slash (except root "/")
func CleanURLPath(p string) string {
	parts := strings.Split(p, "/")
	var cleanParts []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			cleanParts = append(cleanParts, trimmed)
		}
	}
	if len(cleanParts) == 0 {
		return "/"
	}
	return "/" + strings.Join(cleanParts, "/")
}
