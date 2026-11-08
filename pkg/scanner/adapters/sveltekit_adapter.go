package adapters

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
)

type SvelteKitAdapter struct{}

func NewSvelteKitAdapter() *SvelteKitAdapter {
	return &SvelteKitAdapter{}
}

func (a *SvelteKitAdapter) Name() string {
	return "SvelteKit"
}

func (a *SvelteKitAdapter) Type() string {
	return "frontend"
}

func (a *SvelteKitAdapter) Detect(dir string) bool {
	if util.FileExists(filepath.Join(dir, "svelte.config.js")) || util.FileExists(filepath.Join(dir, "svelte.config.ts")) {
		return true
	}
	pkgPath := filepath.Join(dir, "package.json")
	if util.FileExists(pkgPath) {
		content := util.ReadFileString(pkgPath)
		if strings.Contains(content, `"@sveltejs/kit"`) || strings.Contains(content, `"svelte"`) {
			return true
		}
	}
	return false
}

func (a *SvelteKitAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	var routes []models.RouteInfo
	seen := make(map[string]bool)

	apiFuncMap := BuildFrontendApiFunctionMap(dir)

	routesDir := filepath.Join(dir, "src/routes")
	if !dirExists(routesDir) {
		routesDir = filepath.Join(dir, "routes")
	}

	if dirExists(routesDir) {
		_ = filepath.WalkDir(routesDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			baseName := d.Name()
			isPage := strings.HasPrefix(baseName, "+page.") && strings.HasSuffix(baseName, ".svelte")
			isServer := strings.HasPrefix(baseName, "+server.") && (strings.HasSuffix(baseName, ".js") || strings.HasSuffix(baseName, ".ts"))

			if !isPage && !isServer {
				return nil
			}

			relToRoutes, _ := filepath.Rel(routesDir, path)
			routeDir := filepath.Dir(relToRoutes)
			routePath := "/"
			if routeDir != "." {
				routePath = "/" + filepath.ToSlash(routeDir)
			}

			// Param transforms: [id] -> :id
			routePath = strings.ReplaceAll(routePath, "[...", "*")
			routePath = strings.ReplaceAll(routePath, "[", ":")
			routePath = strings.ReplaceAll(routePath, "]", "")

			fileData, _ := os.ReadFile(path)
			contentStr := string(fileData)
			relHandler, _ := filepath.Rel(dir, path)

			if isPage {
				routeKey := "PAGE " + routePath
				if !seen[routeKey] {
					seen[routeKey] = true
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
			} else if isServer {
				// SvelteKit +server: export function GET/POST
				methodRegex := regexp.MustCompile(`export\s+(?:async\s+)?function\s+(GET|POST|PUT|DELETE|PATCH)\b`)
				for _, mm := range methodRegex.FindAllStringSubmatch(contentStr, -1) {
					m := strings.ToUpper(mm[1])
					routeKey := m + " " + routePath
					if !seen[routeKey] {
						seen[routeKey] = true
						routes = append(routes, models.RouteInfo{
							Method:      m,
							Path:        routePath,
							Handler:     filepath.ToSlash(relHandler),
							Description: inferEndpointDescription(m, routePath),
							Middleware:  "SvelteKit Endpoint",
						})
					}
				}
			}

			return nil
		})
	}

	return routes, nil
}
