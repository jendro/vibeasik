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

var svelteMethodRegex = regexp.MustCompile(`export\s+(?:async\s+)?function\s+(GET|POST|PUT|DELETE|PATCH)\b`)

// svelteScan = keadaan satu kali pemindaian proyek SvelteKit.
type svelteScan struct {
	dir        string
	routesDir  string
	apiFuncMap map[string]string
	seen       map[string]bool
	routes     []models.RouteInfo
}

// svelteRoutePath menurunkan path rute dari lokasi berkas +page/+server.
func svelteRoutePath(relToRoutes string) string {
	routeDir := filepath.Dir(relToRoutes)
	routePath := "/"
	if routeDir != "." {
		routePath = "/" + filepath.ToSlash(routeDir)
	}
	// Param transforms: [id] -> :id
	routePath = strings.ReplaceAll(routePath, "[...", "*")
	routePath = strings.ReplaceAll(routePath, "[", ":")
	routePath = strings.ReplaceAll(routePath, "]", "")
	return routePath
}

func (s *svelteScan) addPage(path, routePath, contentStr, relHandler string) {
	routeKey := "PAGE " + routePath
	if s.seen[routeKey] {
		return
	}
	s.seen[routeKey] = true
	s.routes = append(s.routes, models.RouteInfo{
		Method:         "PAGE (GET)",
		Path:           routePath,
		Handler:        filepath.ToSlash(relHandler),
		Description:    inferFrontendPageDescription(routePath),
		Middleware:     "Public",
		TargetRequests: ExtractPageTargetRequests(path, contentStr, s.dir, s.apiFuncMap),
	})
}

// addServer: SvelteKit +server: export function GET/POST
func (s *svelteScan) addServer(routePath, contentStr, relHandler string) {
	for _, mm := range svelteMethodRegex.FindAllStringSubmatch(contentStr, -1) {
		m := strings.ToUpper(mm[1])
		routeKey := m + " " + routePath
		if s.seen[routeKey] {
			continue
		}
		s.seen[routeKey] = true
		s.routes = append(s.routes, models.RouteInfo{
			Method:      m,
			Path:        routePath,
			Handler:     filepath.ToSlash(relHandler),
			Description: inferEndpointDescription(m, routePath),
			Middleware:  "SvelteKit Endpoint",
		})
	}
}

func (s *svelteScan) visit(path string, d os.DirEntry) {
	baseName := d.Name()
	isPage := strings.HasPrefix(baseName, "+page.") && strings.HasSuffix(baseName, ".svelte")
	isServer := strings.HasPrefix(baseName, "+server.") && (strings.HasSuffix(baseName, ".js") || strings.HasSuffix(baseName, ".ts"))
	if !isPage && !isServer {
		return
	}
	relToRoutes, _ := filepath.Rel(s.routesDir, path)
	routePath := svelteRoutePath(relToRoutes)
	fileData, _ := os.ReadFile(path)
	relHandler, _ := filepath.Rel(s.dir, path)
	if isPage {
		s.addPage(path, routePath, string(fileData), relHandler)
		return
	}
	s.addServer(routePath, string(fileData), relHandler)
}

func (a *SvelteKitAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	s := &svelteScan{dir: dir, apiFuncMap: BuildFrontendApiFunctionMap(dir), seen: make(map[string]bool)}

	s.routesDir = filepath.Join(dir, "src/routes")
	if !dirExists(s.routesDir) {
		s.routesDir = filepath.Join(dir, "routes")
	}
	if dirExists(s.routesDir) {
		_ = filepath.WalkDir(s.routesDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			s.visit(path, d)
			return nil
		})
	}
	return s.routes, nil
}
