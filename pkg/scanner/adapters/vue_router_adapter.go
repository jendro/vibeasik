package adapters

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

type VueRouterAdapter struct{}

func NewVueRouterAdapter() *VueRouterAdapter {
	return &VueRouterAdapter{}
}

func (a *VueRouterAdapter) Name() string {
	return "Vue (Vue Router / Vite SPA)"
}

func (a *VueRouterAdapter) Type() string {
	return "frontend"
}

func (a *VueRouterAdapter) Detect(dir string) bool {
	// If it's Nuxt, let NuxtAdapter handle it
	if fileExists(filepath.Join(dir, "nuxt.config.ts")) || fileExists(filepath.Join(dir, "nuxt.config.js")) {
		return false
	}
	pkgPath := filepath.Join(dir, "package.json")
	if fileExists(pkgPath) {
		content := readFileString(pkgPath)
		if strings.Contains(content, `"vue-router"`) || (strings.Contains(content, `"vue"`) && !strings.Contains(content, `"nuxt"`)) {
			return true
		}
	}
	for _, rf := range []string{"src/router/index.ts", "src/router/index.js", "src/router.ts", "src/router.js"} {
		if fileExists(filepath.Join(dir, rf)) {
			return true
		}
	}
	return false
}

func (a *VueRouterAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	var routes []models.RouteInfo
	seen := make(map[string]bool)

	apiFuncMap := BuildFrontendApiFunctionMap(dir)

	// 1. Scan Vue Router definition files
	routeObjRegex := regexp.MustCompile(`path:\s*['"]([^'"]+)['"](?:\s*,\s*name:\s*['"]([^'"]+)['"])?(?:\s*,\s*component:\s*([^,}]+))?`)

	routerFiles := []string{
		"src/router/index.ts", "src/router/index.js",
		"src/router.ts", "src/router.js", "router/index.js",
	}

	for _, rf := range routerFiles {
		fullPath := filepath.Join(dir, rf)
		if fileExists(fullPath) {
			contentBytes, _ := os.ReadFile(fullPath)
			contentStr := string(contentBytes)

			for _, m := range routeObjRegex.FindAllStringSubmatch(contentStr, -1) {
				path := m[1]
				name := m[2]
				handler := name
				if handler == "" && len(m) >= 4 {
					handler = strings.TrimSpace(m[3])
				}
				if handler == "" {
					handler = "VueComponent"
				}

				routeKey := "PAGE " + path
				if !seen[routeKey] {
					seen[routeKey] = true
					routes = append(routes, models.RouteInfo{
						Method:      "PAGE (GET)",
						Path:        path,
						Handler:     handler + " (" + rf + ")",
						Description: inferFrontendPageDescription(path),
						Middleware:  "Public",
					})
				}
			}
		}
	}

	// 2. Scan views/ and pages/
	for _, viewDirRel := range []string{"src/views", "src/pages", "views", "pages"} {
		fullViewDir := filepath.Join(dir, viewDirRel)
		if dirExists(fullViewDir) {
			_ = filepath.WalkDir(fullViewDir, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if filepath.Ext(path) != ".vue" {
					return nil
				}

				relToDir, _ := filepath.Rel(fullViewDir, path)
				relNoExt := strings.TrimSuffix(relToDir, ".vue")
				routePath := "/" + filepath.ToSlash(relNoExt)
				if strings.HasSuffix(routePath, "/index") {
					routePath = strings.TrimSuffix(routePath, "/index")
					if routePath == "" {
						routePath = "/"
					}
				}

				contentBytes, _ := os.ReadFile(path)
				contentStr := string(contentBytes)
				targetRequests := ExtractPageTargetRequests(path, contentStr, dir, apiFuncMap)

				routeKey := "PAGE " + routePath
				if !seen[routeKey] {
					seen[routeKey] = true
					relHandler, _ := filepath.Rel(dir, path)
					routes = append(routes, models.RouteInfo{
						Method:         "PAGE (GET)",
						Path:           routePath,
						Handler:        filepath.ToSlash(relHandler),
						Description:    inferFrontendPageDescription(routePath),
						Middleware:     "Public",
						TargetRequests: targetRequests,
					})
				} else {
					// Enrich target requests if already added via router file
					for idx, existing := range routes {
						if existing.Path == routePath && len(existing.TargetRequests) == 0 {
							routes[idx].TargetRequests = targetRequests
							break
						}
					}
				}
				return nil
			})
		}
	}

	return routes, nil
}
