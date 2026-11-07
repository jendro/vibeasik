package adapters

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

type ReactRouterAdapter struct{}

func NewReactRouterAdapter() *ReactRouterAdapter {
	return &ReactRouterAdapter{}
}

func (a *ReactRouterAdapter) Name() string {
	return "React (React Router / TanStack Router / SPA)"
}

func (a *ReactRouterAdapter) Type() string {
	return "frontend"
}

func (a *ReactRouterAdapter) Detect(dir string) bool {
	pkgPath := filepath.Join(dir, "package.json")
	if fileExists(pkgPath) {
		content := readFileString(pkgPath)
		if strings.Contains(content, `"react-router"`) ||
			strings.Contains(content, `"react-router-dom"`) ||
			strings.Contains(content, `"@tanstack/react-router"`) ||
			(strings.Contains(content, `"react"`) && !strings.Contains(content, `"next"`)) {
			return true
		}
	}
	return false
}

func (a *ReactRouterAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	var routes []models.RouteInfo
	seen := make(map[string]bool)

	apiFuncMap := BuildFrontendApiFunctionMap(dir)

	// 1. Scan router files (App.tsx, routes.tsx, router.tsx, main.tsx) for <Route path=...> and createBrowserRouter
	routeTagRegex := regexp.MustCompile(`<Route\s+[^>]*path=["']([^"']+)["'][^>]*element=\{<([a-zA-Z0-9_]+)`)
	routerObjectRegex := regexp.MustCompile(`path:\s*['"]([^'"]+)['"]\s*,\s*element:\s*<([a-zA-Z0-9_]+)`)

	candidateRouterFiles := []string{
		"src/App.tsx", "src/App.jsx", "src/routes.tsx", "src/routes.jsx",
		"src/router.tsx", "src/router.jsx", "src/router/index.tsx", "src/router/index.ts",
		"src/main.tsx", "src/main.jsx", "src/index.tsx", "src/index.jsx",
	}

	for _, rf := range candidateRouterFiles {
		fullPath := filepath.Join(dir, rf)
		if fileExists(fullPath) {
			contentBytes, _ := os.ReadFile(fullPath)
			contentStr := string(contentBytes)

			// Match <Route path=... element=<Comp />
			for _, m := range routeTagRegex.FindAllStringSubmatch(contentStr, -1) {
				path := m[1]
				handler := m[2]
				routeKey := "PAGE " + path
				if !seen[routeKey] {
					seen[routeKey] = true
					targetRequests := ExtractPageTargetRequests(fullPath, contentStr, dir, apiFuncMap)
					routes = append(routes, models.RouteInfo{
						Method:         "PAGE (GET)",
						Path:           path,
						Handler:        handler + " (" + rf + ")",
						Description:    inferFrontendPageDescription(path),
						Middleware:     "Public",
						TargetRequests: targetRequests,
					})
				}
			}

			// Match createBrowserRouter([{ path: ..., element: ... }])
			for _, m := range routerObjectRegex.FindAllStringSubmatch(contentStr, -1) {
				path := m[1]
				handler := m[2]
				routeKey := "PAGE " + path
				if !seen[routeKey] {
					seen[routeKey] = true
					targetRequests := ExtractPageTargetRequests(fullPath, contentStr, dir, apiFuncMap)
					routes = append(routes, models.RouteInfo{
						Method:         "PAGE (GET)",
						Path:           path,
						Handler:        handler + " (" + rf + ")",
						Description:    inferFrontendPageDescription(path),
						Middleware:     "Public",
						TargetRequests: targetRequests,
					})
				}
			}
		}
	}

	// 2. Scan file-based page directories: src/pages, src/views, src/screens
	for _, pageDirRel := range []string{"src/pages", "pages", "src/views", "src/screens"} {
		fullPageDir := filepath.Join(dir, pageDirRel)
		if dirExists(fullPageDir) {
			_ = filepath.WalkDir(fullPageDir, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				ext := filepath.Ext(path)
				if ext != ".tsx" && ext != ".jsx" && ext != ".js" {
					return nil
				}

				relToDir, _ := filepath.Rel(fullPageDir, path)
				relNoExt := strings.TrimSuffix(relToDir, ext)
				routePath := "/" + filepath.ToSlash(relNoExt)
				if strings.HasSuffix(routePath, "/index") {
					routePath = strings.TrimSuffix(routePath, "/index")
					if routePath == "" {
						routePath = "/"
					}
				}

				routeKey := "PAGE " + routePath
				if !seen[routeKey] {
					seen[routeKey] = true
					contentBytes, _ := os.ReadFile(path)
					contentStr := string(contentBytes)
					relHandler, _ := filepath.Rel(dir, path)
					targetRequests := ExtractPageTargetRequests(path, contentStr, dir, apiFuncMap)

					routes = append(routes, models.RouteInfo{
						Method:         "PAGE (GET)",
						Path:           routePath,
						Handler:        filepath.ToSlash(relHandler),
						Description:    inferFrontendPageDescription(routePath),
						Middleware:     "Public",
						TargetRequests: targetRequests,
					})
				}
				return nil
			})
		}
	}

	return routes, nil
}
