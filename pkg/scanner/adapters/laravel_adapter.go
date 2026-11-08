package adapters

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
)

type LaravelAdapter struct{}

func NewLaravelAdapter() *LaravelAdapter {
	return &LaravelAdapter{}
}

func (a *LaravelAdapter) Name() string {
	return "PHP / Laravel"
}

func (a *LaravelAdapter) Type() string {
	return "backend"
}

func (a *LaravelAdapter) Detect(dir string) bool {
	if util.FileExists(filepath.Join(dir, "artisan")) {
		return true
	}
	composerPath := filepath.Join(dir, "composer.json")
	if util.FileExists(composerPath) {
		content := util.ReadFileString(composerPath)
		if strings.Contains(content, "laravel/framework") || strings.Contains(content, "illuminate/") {
			return true
		}
	}
	if util.FileExists(filepath.Join(dir, "routes/web.php")) || util.FileExists(filepath.Join(dir, "routes/api.php")) {
		return true
	}
	return false
}

func (a *LaravelAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	var routes []models.RouteInfo
	scannedFiles := make(map[string]bool)

	routesDir := filepath.Join(dir, "routes")
	if dirExists(routesDir) {
		entries, _ := os.ReadDir(routesDir)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".php") {
				p := filepath.Join(routesDir, e.Name())
				scannedFiles[p] = true
				defaultPrefix := ""
				if e.Name() == "api.php" {
					defaultPrefix = "/api"
				}
				r := ParseLaravelRoutesFromFile(p, defaultPrefix)
				routes = append(routes, r...)
			}
		}
	}

	return routes, nil
}

// ParseLaravelRoutesFromFile extracts Route:: declarations from a Laravel route file
func ParseLaravelRoutesFromFile(filePath string, basePrefix string) []models.RouteInfo {
	var routes []models.RouteInfo

	file, err := os.Open(filePath)
	if err != nil {
		return routes
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	routeMethodRegex := regexp.MustCompile(`(?i)Route::(get|post|put|delete|patch|options|any)\(\s*['"]([^'"]+)['"]\s*,\s*([^)]+)\)`)
	apiResourceRegex := regexp.MustCompile(`(?i)Route::(apiResource|resource)\(\s*['"]([^'"]+)['"]\s*,\s*([^)]+)\)`)
	prefixGroupRegex := regexp.MustCompile(`(?i)Route::prefix\(\s*['"]([^'"]+)['"]\s*\)`)

	currentPrefix := basePrefix

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}

		// Detect prefix group
		if pm := prefixGroupRegex.FindStringSubmatch(line); len(pm) >= 2 {
			p := strings.Trim(pm[1], "/")
			if basePrefix != "" {
				currentPrefix = strings.TrimRight(basePrefix, "/") + "/" + p
			} else {
				currentPrefix = "/" + p
			}
		}

		// Standard Route::get/post/etc
		if rm := routeMethodRegex.FindStringSubmatch(line); len(rm) >= 4 {
			method := strings.ToUpper(rm[1])
			path := rm[2]
			handlerPart := rm[3]

			fullPath := combinePrefixAndPath(currentPrefix, path)
			handler := cleanLaravelHandler(handlerPart)
			middleware := "Public"
			if strings.Contains(line, "auth") || strings.Contains(line, "sanctum") || strings.Contains(line, "verified") {
				middleware = "Protected (Auth Guard)"
			}

			if method == "ANY" {
				method = "ALL"
			}

			routes = append(routes, models.RouteInfo{
				Method:      method,
				Path:        fullPath,
				Handler:     handler,
				Description: inferEndpointDescription(method, fullPath),
				Middleware:  middleware,
			})
			continue
		}

		// Route::apiResource('users', UserController::class)
		if arm := apiResourceRegex.FindStringSubmatch(line); len(arm) >= 4 {
			resType := strings.ToLower(arm[1])
			resName := strings.Trim(arm[2], "/")
			ctrl := cleanLaravelHandler(arm[3])
			baseResPath := combinePrefixAndPath(currentPrefix, resName)

			// Standard REST API routes for resources
			routes = append(routes,
				models.RouteInfo{
					Method:      "GET",
					Path:        baseResPath,
					Handler:     ctrl + "@index",
					Description: fmt.Sprintf("Daftar sumber daya %s (index)", resName),
					Middleware:  "Public",
				},
				models.RouteInfo{
					Method:      "POST",
					Path:        baseResPath,
					Handler:     ctrl + "@store",
					Description: fmt.Sprintf("Pembuatan sumber daya %s baru (store)", resName),
					Middleware:  "Public",
				},
				models.RouteInfo{
					Method:      "GET",
					Path:        baseResPath + "/{id}",
					Handler:     ctrl + "@show",
					Description: fmt.Sprintf("Detail sumber daya %s berdasarkan ID (show)", resName),
					Middleware:  "Public",
				},
				models.RouteInfo{
					Method:      "PUT",
					Path:        baseResPath + "/{id}",
					Handler:     ctrl + "@update",
					Description: fmt.Sprintf("Pembaruan sumber daya %s (update)", resName),
					Middleware:  "Public",
				},
				models.RouteInfo{
					Method:      "DELETE",
					Path:        baseResPath + "/{id}",
					Handler:     ctrl + "@destroy",
					Description: fmt.Sprintf("Penghapusan sumber daya %s (destroy)", resName),
					Middleware:  "Public",
				},
			)

			if resType == "resource" {
				// Additional web resource routes: create and edit
				routes = append(routes,
					models.RouteInfo{
						Method:      "GET",
						Path:        baseResPath + "/create",
						Handler:     ctrl + "@create",
						Description: fmt.Sprintf("Form pembuatan %s (create)", resName),
						Middleware:  "Public",
					},
					models.RouteInfo{
						Method:      "GET",
						Path:        baseResPath + "/{id}/edit",
						Handler:     ctrl + "@edit",
						Description: fmt.Sprintf("Form pengeditan %s (edit)", resName),
						Middleware:  "Public",
					},
				)
			}
		}
	}

	return routes
}

func combinePrefixAndPath(prefix, path string) string {
	cleanP := strings.Trim(path, "/")
	if prefix == "" {
		if cleanP == "" {
			return "/"
		}
		return "/" + cleanP
	}
	cleanPref := strings.TrimRight(prefix, "/")
	if cleanP == "" {
		return cleanPref
	}
	return cleanPref + "/" + cleanP
}

func cleanLaravelHandler(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ")")
	raw = strings.TrimSuffix(raw, ";")
	raw = strings.TrimSpace(raw)

	// Pattern [UserController::class, 'index']
	arrayRegex := regexp.MustCompile(`\[\s*([a-zA-Z0-9_\\]+)::class\s*,\s*['"]([a-zA-Z0-9_]+)['"]\s*\]`)
	if m := arrayRegex.FindStringSubmatch(raw); len(m) >= 3 {
		className := m[1]
		if lastSlash := strings.LastIndex(className, "\\"); lastSlash != -1 {
			className = className[lastSlash+1:]
		}
		return className + "@" + m[2]
	}

	// Pattern UserController::class
	classRegex := regexp.MustCompile(`^([a-zA-Z0-9_\\]+)::class$`)
	if m := classRegex.FindStringSubmatch(raw); len(m) >= 2 {
		className := m[1]
		if lastSlash := strings.LastIndex(className, "\\"); lastSlash != -1 {
			className = className[lastSlash+1:]
		}
		return className
	}

	// Pattern 'UserController@index'
	stringRegex := regexp.MustCompile(`['"]([a-zA-Z0-9_\\]+@[a-zA-Z0-9_]+)['"]`)
	if m := stringRegex.FindStringSubmatch(raw); len(m) >= 2 {
		return m[1]
	}

	// Closure or inline function
	if strings.Contains(raw, "function") || strings.Contains(raw, "fn(") {
		return "Closure / Inline Action"
	}

	return raw
}
