package adapters

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

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
	if util.FileExists(pkgPath) {
		content := util.ReadFileString(pkgPath)
		if strings.Contains(content, `"react-router"`) ||
			strings.Contains(content, `"react-router-dom"`) ||
			strings.Contains(content, `"@tanstack/react-router"`) ||
			(strings.Contains(content, `"react"`) && !strings.Contains(content, `"next"`)) {
			return true
		}
	}
	return false
}

var (
	reactRouteTagRegex      = regexp.MustCompile(`<Route\s+[^>]*path=["']([^"']+)["'][^>]*element=\{<([a-zA-Z0-9_]+)`)
	reactRouterObjectRegex  = regexp.MustCompile(`path:\s*['"]([^'"]+)['"]\s*,\s*element:\s*<([a-zA-Z0-9_]+)`)
	reactCandidateRouterSrc = []string{
		"src/App.tsx", "src/App.jsx", "src/routes.tsx", "src/routes.jsx",
		"src/router.tsx", "src/router.jsx", "src/router/index.tsx", "src/router/index.ts",
		"src/main.tsx", "src/main.jsx", "src/index.tsx", "src/index.jsx",
	}
)

// reactScan = keadaan satu kali pemindaian proyek React.
type reactScan struct {
	dir        string
	apiFuncMap map[string]string
	seen       map[string]bool
	routes     []models.RouteInfo
}

// scanRouterFile membaca <Route path=... element=<Comp />> dan createBrowserRouter([{ path, element }]).
func (r *reactScan) scanRouterFile(rf string) {
	fullPath := filepath.Join(r.dir, rf)
	if !util.FileExists(fullPath) {
		return
	}
	contentBytes, _ := os.ReadFile(fullPath)
	contentStr := string(contentBytes)

	for _, re := range []*regexp.Regexp{reactRouteTagRegex, reactRouterObjectRegex} {
		for _, m := range re.FindAllStringSubmatch(contentStr, -1) {
			path := m[1]
			routeKey := "PAGE " + path
			if r.seen[routeKey] {
				continue
			}
			r.seen[routeKey] = true
			r.routes = append(r.routes, models.RouteInfo{
				Method:         "PAGE (GET)",
				Path:           path,
				Handler:        m[2] + " (" + rf + ")",
				Description:    inferFrontendPageDescription(path),
				Middleware:     "Public",
				TargetRequests: ExtractPageTargetRequests(fullPath, contentStr, r.dir, r.apiFuncMap),
			})
		}
	}
}

func (r *reactScan) addPageFile(fullPageDir, path string) {
	ext := filepath.Ext(path)
	if ext != ".tsx" && ext != ".jsx" && ext != ".js" {
		return
	}
	relToDir, _ := filepath.Rel(fullPageDir, path)
	routePath := indexRoutePath("/" + filepath.ToSlash(strings.TrimSuffix(relToDir, ext)))

	routeKey := "PAGE " + routePath
	if r.seen[routeKey] {
		return
	}
	r.seen[routeKey] = true
	contentBytes, _ := os.ReadFile(path)
	relHandler, _ := filepath.Rel(r.dir, path)
	r.routes = append(r.routes, models.RouteInfo{
		Method:         "PAGE (GET)",
		Path:           routePath,
		Handler:        filepath.ToSlash(relHandler),
		Description:    inferFrontendPageDescription(routePath),
		Middleware:     "Public",
		TargetRequests: ExtractPageTargetRequests(path, string(contentBytes), r.dir, r.apiFuncMap),
	})
}

// scanPageDirs memindai direktori halaman berbasis berkas: src/pages, src/views, src/screens.
func (r *reactScan) scanPageDirs() {
	for _, pageDirRel := range []string{"src/pages", "pages", "src/views", "src/screens"} {
		fullPageDir := filepath.Join(r.dir, pageDirRel)
		if !dirExists(fullPageDir) {
			continue
		}
		_ = filepath.WalkDir(fullPageDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			r.addPageFile(fullPageDir, path)
			return nil
		})
	}
}

func (a *ReactRouterAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	r := &reactScan{dir: dir, apiFuncMap: BuildFrontendApiFunctionMap(dir), seen: make(map[string]bool)}
	for _, rf := range reactCandidateRouterSrc {
		r.scanRouterFile(rf)
	}
	r.scanPageDirs()
	return r.routes, nil
}
