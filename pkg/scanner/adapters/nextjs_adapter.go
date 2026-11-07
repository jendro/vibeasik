package adapters

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

type NextJsAdapter struct{}

func NewNextJsAdapter() *NextJsAdapter {
	return &NextJsAdapter{}
}

func (a *NextJsAdapter) Name() string {
	return "Next.js (App Router / Pages Router)"
}

func (a *NextJsAdapter) Type() string {
	return "frontend"
}

func (a *NextJsAdapter) Detect(dir string) bool {
	for _, f := range []string{"next.config.js", "next.config.mjs", "next.config.ts"} {
		if fileExists(filepath.Join(dir, f)) {
			return true
		}
	}
	pkgPath := filepath.Join(dir, "package.json")
	if fileExists(pkgPath) {
		content := readFileString(pkgPath)
		if strings.Contains(content, `"next"`) {
			return true
		}
	}
	return false
}

func (a *NextJsAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	var routes []models.RouteInfo
	seen := make(map[string]bool)

	apiFuncMap := BuildFrontendApiFunctionMap(dir)

	// 1. Scan Next.js App Router (app/ or src/app/)
	appDirs := []string{"app", "src/app"}
	for _, appRel := range appDirs {
		appFull := filepath.Join(dir, appRel)
		if dirExists(appFull) {
			_ = filepath.WalkDir(appFull, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					if shouldIgnoreName(d.Name()) {
						return filepath.SkipDir
					}
					return nil
				}

				baseName := d.Name()
				isPage := strings.HasPrefix(baseName, "page.") && (strings.HasSuffix(baseName, ".tsx") || strings.HasSuffix(baseName, ".jsx") || strings.HasSuffix(baseName, ".js"))
				isRoute := strings.HasPrefix(baseName, "route.") && (strings.HasSuffix(baseName, ".ts") || strings.HasSuffix(baseName, ".js"))

				if !isPage && !isRoute {
					return nil
				}

				relToApp, _ := filepath.Rel(appFull, path)
				routeDir := filepath.Dir(relToApp)
				routePath := "/"
				if routeDir != "." {
					routePath = "/" + filepath.ToSlash(routeDir)
				}

				// Remove Route Groups: (group) -> empty
				groupRegex := regexp.MustCompile(`/\([^)]+\)`)
				routePath = groupRegex.ReplaceAllString(routePath, "")
				if routePath == "" {
					routePath = "/"
				}

				// Transform parameters: [id] -> :id, [...slug] -> *
				routePath = strings.ReplaceAll(routePath, "[[...", "*")
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
						middleware := "Public"
						if strings.Contains(contentStr, "getServerSession") || strings.Contains(contentStr, "auth()") {
							middleware = "Protected (NextAuth)"
						}
						routes = append(routes, models.RouteInfo{
							Method:         "PAGE (GET)",
							Path:           routePath,
							Handler:        filepath.ToSlash(relHandler),
							Description:    inferFrontendPageDescription(routePath),
							Middleware:     middleware,
							TargetRequests: targetRequests,
						})
					}
				} else if isRoute {
					// Next.js Route Handler: export async function GET/POST/etc
					methodRegex := regexp.MustCompile(`export\s+(?:async\s+)?function\s+(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\b`)
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
								Middleware:  "Route Handler",
							})
						}
					}
				}

				return nil
			})
		}
	}

	// 2. Scan Next.js Pages Router (pages/ or src/pages/)
	pageDirs := []string{"pages", "src/pages"}
	for _, pageRel := range pageDirs {
		pageFull := filepath.Join(dir, pageRel)
		if dirExists(pageFull) {
			_ = filepath.WalkDir(pageFull, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				ext := filepath.Ext(path)
				if ext != ".tsx" && ext != ".jsx" && ext != ".js" {
					return nil
				}

				relToPages, _ := filepath.Rel(pageFull, path)
				relNoExt := strings.TrimSuffix(relToPages, ext)
				if strings.HasPrefix(relNoExt, "_") {
					// Skip _app, _document
					return nil
				}

				// Check if it's api route in pages/api
				isApi := strings.HasPrefix(relNoExt, "api/") || strings.HasPrefix(relNoExt, "api\\")

				routePath := "/" + filepath.ToSlash(relNoExt)
				if strings.HasSuffix(routePath, "/index") {
					routePath = strings.TrimSuffix(routePath, "/index")
					if routePath == "" {
						routePath = "/"
					}
				}

				routePath = strings.ReplaceAll(routePath, "[...", "*")
				routePath = strings.ReplaceAll(routePath, "[", ":")
				routePath = strings.ReplaceAll(routePath, "]", "")

				fileData, _ := os.ReadFile(path)
				contentStr := string(fileData)
				relHandler, _ := filepath.Rel(dir, path)

				if isApi {
					routeKey := "ALL " + routePath
					if !seen[routeKey] {
						seen[routeKey] = true
						routes = append(routes, models.RouteInfo{
							Method:      "ALL",
							Path:        routePath,
							Handler:     filepath.ToSlash(relHandler),
							Description: inferEndpointDescription("ALL", routePath),
							Middleware:  "API Handler",
						})
					}
				} else {
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
				}

				return nil
			})
		}
	}

	return routes, nil
}
