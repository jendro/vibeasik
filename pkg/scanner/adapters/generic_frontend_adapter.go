package adapters

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
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
	return util.FileExists(filepath.Join(dir, "package.json")) || util.FileExists(filepath.Join(dir, "index.html"))
}

var genericPageExts = map[string]bool{".vue": true, ".tsx": true, ".jsx": true, ".svelte": true, ".html": true}

// indexRoutePath: "<dir>/index" menjadi "<dir>" ("/" bila kosong); jalur lain dikembalikan apa adanya.
func indexRoutePath(routePath string) string {
	if strings.HasSuffix(routePath, "/index") {
		routePath = strings.TrimSuffix(routePath, "/index")
		if routePath == "" {
			routePath = "/"
		}
	}
	return routePath
}

func (a *GenericFrontendAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	var routes []models.RouteInfo
	seen := make(map[string]bool)

	apiFuncMap := BuildFrontendApiFunctionMap(dir)

	candidateDirs := []string{"app/pages", "src/pages", "pages", "src/views", "src/screens", "src/app", "app"}
	for _, candRel := range activePageDirs(dir, candidateDirs) {
		candFull := filepath.Join(dir, candRel)
		if !dirExists(candFull) {
			continue
		}

		_ = filepath.WalkDir(candFull, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			if !genericPageExts[ext] {
				return nil
			}

			relToCand, _ := filepath.Rel(candFull, path)
			routePath := indexRoutePath("/" + filepath.ToSlash(strings.TrimSuffix(relToCand, ext)))

			routeKey := "PAGE " + routePath
			if seen[routeKey] {
				return nil
			}
			seen[routeKey] = true
			contentBytes, _ := os.ReadFile(path)
			relHandler, _ := filepath.Rel(dir, path)

			routes = append(routes, models.RouteInfo{
				Method:         "PAGE (GET)",
				Path:           routePath,
				Handler:        filepath.ToSlash(relHandler),
				Description:    inferFrontendPageDescription(routePath),
				Middleware:     "Public",
				TargetRequests: ExtractPageTargetRequests(path, string(contentBytes), dir, apiFuncMap),
			})
			return nil
		})
	}

	return routes, nil
}
