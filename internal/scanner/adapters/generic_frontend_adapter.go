package adapters

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/internal/models"
)

type GenericFrontendAdapter struct{}

func NewGenericFrontendAdapter() *GenericFrontendAdapter {
	return &GenericFrontendAdapter{}
}

func (a *GenericFrontendAdapter) Name() string {
	return "Generic Frontend (SPA / MPA / Custom UI)"
}

func (a *GenericFrontendAdapter) Type() string {
	return "frontend"
}

func (a *GenericFrontendAdapter) Detect(dir string) bool {
	// Matches if there are frontend files
	for _, cand := range []string{"pages", "src/pages", "src/views", "src/screens", "app", "src/app"} {
		if dirExists(filepath.Join(dir, cand)) {
			return true
		}
	}
	return fileExists(filepath.Join(dir, "package.json")) || fileExists(filepath.Join(dir, "index.html"))
}

func (a *GenericFrontendAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	var routes []models.RouteInfo
	seen := make(map[string]bool)

	apiFuncMap := BuildFrontendApiFunctionMap(dir)

	candidateDirs := []string{"app/pages", "src/pages", "pages", "src/views", "src/screens", "src/app", "app"}
	var activeDirs []string
	for _, cand := range candidateDirs {
		p := filepath.Join(dir, cand)
		if dirExists(p) {
			isSubOrParent := false
			for _, existing := range activeDirs {
				if strings.HasPrefix(cand, existing+"/") || strings.HasPrefix(existing, cand+"/") {
					isSubOrParent = true
					break
				}
			}
			if !isSubOrParent {
				activeDirs = append(activeDirs, cand)
			}
		}
	}

	for _, candRel := range activeDirs {
		candFull := filepath.Join(dir, candRel)
		if !dirExists(candFull) {
			continue
		}

		_ = filepath.WalkDir(candFull, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".vue" && ext != ".tsx" && ext != ".jsx" && ext != ".svelte" && ext != ".html" {
				return nil
			}

			relToCand, _ := filepath.Rel(candFull, path)
			relNoExt := strings.TrimSuffix(relToCand, ext)
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

	return routes, nil
}
