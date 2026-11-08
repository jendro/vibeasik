package adapters

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

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
		if util.FileExists(filepath.Join(dir, f)) {
			return true
		}
	}
	pkgPath := filepath.Join(dir, "package.json")
	if util.FileExists(pkgPath) {
		content := util.ReadFileString(pkgPath)
		if strings.Contains(content, `"next"`) {
			return true
		}
	}
	return false
}

var (
	nextGroupRegex  = regexp.MustCompile(`/\([^)]+\)`)
	nextMethodRegex = regexp.MustCompile(`export\s+(?:async\s+)?function\s+(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\b`)
)

// nextScan = keadaan satu kali pemindaian proyek Next.js.
type nextScan struct {
	dir        string
	apiFuncMap map[string]string
	seen       map[string]bool
	routes     []models.RouteInfo
}

// nextAppRoutePath menurunkan path rute App Router dari lokasi berkas relatif terhadap app/.
func nextAppRoutePath(relToApp string) string {
	routeDir := filepath.Dir(relToApp)
	routePath := "/"
	if routeDir != "." {
		routePath = "/" + filepath.ToSlash(routeDir)
	}
	// Remove Route Groups: (group) -> empty
	routePath = nextGroupRegex.ReplaceAllString(routePath, "")
	if routePath == "" {
		routePath = "/"
	}
	// Transform parameters: [id] -> :id, [...slug] -> *
	routePath = strings.ReplaceAll(routePath, "[[...", "*")
	routePath = strings.ReplaceAll(routePath, "[...", "*")
	routePath = strings.ReplaceAll(routePath, "[", ":")
	routePath = strings.ReplaceAll(routePath, "]", "")
	return routePath
}

func nextAppMiddleware(content string) string {
	if strings.Contains(content, "getServerSession") || strings.Contains(content, "auth()") {
		return "Protected (NextAuth)"
	}
	return "Public"
}

func (n *nextScan) addAppPage(path, routePath, contentStr, relHandler string) {
	routeKey := "PAGE " + routePath
	if n.seen[routeKey] {
		return
	}
	n.seen[routeKey] = true
	n.routes = append(n.routes, models.RouteInfo{
		Method:         "PAGE (GET)",
		Path:           routePath,
		Handler:        filepath.ToSlash(relHandler),
		Description:    inferFrontendPageDescription(routePath),
		Middleware:     nextAppMiddleware(contentStr),
		TargetRequests: ExtractPageTargetRequests(path, contentStr, n.dir, n.apiFuncMap),
	})
}

// addAppRouteHandler: Next.js Route Handler: export async function GET/POST/etc
func (n *nextScan) addAppRouteHandler(routePath, contentStr, relHandler string) {
	for _, mm := range nextMethodRegex.FindAllStringSubmatch(contentStr, -1) {
		m := strings.ToUpper(mm[1])
		routeKey := m + " " + routePath
		if n.seen[routeKey] {
			continue
		}
		n.seen[routeKey] = true
		n.routes = append(n.routes, models.RouteInfo{
			Method:      m,
			Path:        routePath,
			Handler:     filepath.ToSlash(relHandler),
			Description: inferEndpointDescription(m, routePath),
			Middleware:  "Route Handler",
		})
	}
}

func (n *nextScan) visitAppFile(appFull, path string, d os.DirEntry) {
	baseName := d.Name()
	isPage := strings.HasPrefix(baseName, "page.") && (strings.HasSuffix(baseName, ".tsx") || strings.HasSuffix(baseName, ".jsx") || strings.HasSuffix(baseName, ".js"))
	isRoute := strings.HasPrefix(baseName, "route.") && (strings.HasSuffix(baseName, ".ts") || strings.HasSuffix(baseName, ".js"))
	if !isPage && !isRoute {
		return
	}
	relToApp, _ := filepath.Rel(appFull, path)
	routePath := nextAppRoutePath(relToApp)
	fileData, _ := os.ReadFile(path)
	relHandler, _ := filepath.Rel(n.dir, path)
	if isPage {
		n.addAppPage(path, routePath, string(fileData), relHandler)
		return
	}
	n.addAppRouteHandler(routePath, string(fileData), relHandler)
}

// scanAppRouter memindai App Router (app/ atau src/app/).
func (n *nextScan) scanAppRouter() {
	for _, appRel := range []string{"app", "src/app"} {
		appFull := filepath.Join(n.dir, appRel)
		if !dirExists(appFull) {
			continue
		}
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
			n.visitAppFile(appFull, path, d)
			return nil
		})
	}
}

// nextPagesRoutePath menurunkan path rute Pages Router dari nama berkas tanpa ekstensi.
func nextPagesRoutePath(relNoExt string) string {
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
	return routePath
}

func (n *nextScan) visitPagesFile(pageFull, path string) {
	ext := filepath.Ext(path)
	if ext != ".tsx" && ext != ".jsx" && ext != ".js" {
		return
	}
	relToPages, _ := filepath.Rel(pageFull, path)
	relNoExt := strings.TrimSuffix(relToPages, ext)
	if strings.HasPrefix(relNoExt, "_") {
		// Skip _app, _document
		return
	}
	// Check if it's api route in pages/api
	isApi := strings.HasPrefix(relNoExt, "api/") || strings.HasPrefix(relNoExt, "api\\")
	routePath := nextPagesRoutePath(relNoExt)

	fileData, _ := os.ReadFile(path)
	relHandler, _ := filepath.Rel(n.dir, path)

	if isApi {
		routeKey := "ALL " + routePath
		if n.seen[routeKey] {
			return
		}
		n.seen[routeKey] = true
		n.routes = append(n.routes, models.RouteInfo{
			Method:      "ALL",
			Path:        routePath,
			Handler:     filepath.ToSlash(relHandler),
			Description: inferEndpointDescription("ALL", routePath),
			Middleware:  "API Handler",
		})
		return
	}
	routeKey := "PAGE " + routePath
	if n.seen[routeKey] {
		return
	}
	n.seen[routeKey] = true
	n.routes = append(n.routes, models.RouteInfo{
		Method:         "PAGE (GET)",
		Path:           routePath,
		Handler:        filepath.ToSlash(relHandler),
		Description:    inferFrontendPageDescription(routePath),
		Middleware:     "Public",
		TargetRequests: ExtractPageTargetRequests(path, string(fileData), n.dir, n.apiFuncMap),
	})
}

// scanPagesRouter memindai Pages Router (pages/ atau src/pages/).
func (n *nextScan) scanPagesRouter() {
	for _, pageRel := range []string{"pages", "src/pages"} {
		pageFull := filepath.Join(n.dir, pageRel)
		if !dirExists(pageFull) {
			continue
		}
		_ = filepath.WalkDir(pageFull, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			n.visitPagesFile(pageFull, path)
			return nil
		})
	}
}

func (a *NextJsAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	n := &nextScan{dir: dir, apiFuncMap: BuildFrontendApiFunctionMap(dir), seen: make(map[string]bool)}
	n.scanAppRouter()
	n.scanPagesRouter()
	return n.routes, nil
}
