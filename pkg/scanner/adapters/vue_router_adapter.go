package adapters

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

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
	if util.FileExists(filepath.Join(dir, "nuxt.config.ts")) || util.FileExists(filepath.Join(dir, "nuxt.config.js")) {
		return false
	}
	pkgPath := filepath.Join(dir, "package.json")
	if util.FileExists(pkgPath) {
		content := util.ReadFileString(pkgPath)
		if strings.Contains(content, `"vue-router"`) || (strings.Contains(content, `"vue"`) && !strings.Contains(content, `"nuxt"`)) {
			return true
		}
	}
	for _, rf := range []string{"src/router/index.ts", "src/router/index.js", "src/router.ts", "src/router.js"} {
		if util.FileExists(filepath.Join(dir, rf)) {
			return true
		}
	}
	return false
}

var vueRouteObjRegex = regexp.MustCompile(`path:\s*['"]([^'"]+)['"](?:\s*,\s*name:\s*['"]([^'"]+)['"])?(?:\s*,\s*component:\s*([^,}]+))?`)

// vueScan = keadaan satu kali pemindaian proyek Vue.
type vueScan struct {
	dir        string
	apiFuncMap map[string]string
	seen       map[string]bool
	routes     []models.RouteInfo
}

// vueRouteHandler memilih nama handler: name, lalu component, lalu "VueComponent".
func vueRouteHandler(name string, m []string) string {
	handler := name
	if handler == "" && len(m) >= 4 {
		handler = strings.TrimSpace(m[3])
	}
	if handler == "" {
		handler = "VueComponent"
	}
	return handler
}

// scanRouterFiles membaca definisi Vue Router.
func (v *vueScan) scanRouterFiles() {
	routerFiles := []string{
		"src/router/index.ts", "src/router/index.js",
		"src/router.ts", "src/router.js", "router/index.js",
	}
	for _, rf := range routerFiles {
		fullPath := filepath.Join(v.dir, rf)
		if !util.FileExists(fullPath) {
			continue
		}
		contentBytes, _ := os.ReadFile(fullPath)
		for _, m := range vueRouteObjRegex.FindAllStringSubmatch(string(contentBytes), -1) {
			path := m[1]
			routeKey := "PAGE " + path
			if v.seen[routeKey] {
				continue
			}
			v.seen[routeKey] = true
			v.routes = append(v.routes, models.RouteInfo{
				Method:      "PAGE (GET)",
				Path:        path,
				Handler:     vueRouteHandler(m[2], m) + " (" + rf + ")",
				Description: inferFrontendPageDescription(path),
				Middleware:  "Public",
			})
		}
	}
}

// enrichTargets mengisi TargetRequests rute yang sudah ada dari berkas router.
func (v *vueScan) enrichTargets(routePath string, targetRequests []string) {
	for idx, existing := range v.routes {
		if existing.Path == routePath && len(existing.TargetRequests) == 0 {
			v.routes[idx].TargetRequests = targetRequests
			break
		}
	}
}

func (v *vueScan) addViewFile(fullViewDir, path string) {
	if filepath.Ext(path) != ".vue" {
		return
	}
	relToDir, _ := filepath.Rel(fullViewDir, path)
	routePath := indexRoutePath("/" + filepath.ToSlash(strings.TrimSuffix(relToDir, ".vue")))

	contentBytes, _ := os.ReadFile(path)
	targetRequests := ExtractPageTargetRequests(path, string(contentBytes), v.dir, v.apiFuncMap)

	routeKey := "PAGE " + routePath
	if v.seen[routeKey] {
		// Enrich target requests if already added via router file
		v.enrichTargets(routePath, targetRequests)
		return
	}
	v.seen[routeKey] = true
	relHandler, _ := filepath.Rel(v.dir, path)
	v.routes = append(v.routes, models.RouteInfo{
		Method:         "PAGE (GET)",
		Path:           routePath,
		Handler:        filepath.ToSlash(relHandler),
		Description:    inferFrontendPageDescription(routePath),
		Middleware:     "Public",
		TargetRequests: targetRequests,
	})
}

// scanViewDirs memindai views/ dan pages/.
func (v *vueScan) scanViewDirs() {
	for _, viewDirRel := range []string{"src/views", "src/pages", "views", "pages"} {
		fullViewDir := filepath.Join(v.dir, viewDirRel)
		if !dirExists(fullViewDir) {
			continue
		}
		_ = filepath.WalkDir(fullViewDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			v.addViewFile(fullViewDir, path)
			return nil
		})
	}
}

func (a *VueRouterAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	v := &vueScan{dir: dir, apiFuncMap: BuildFrontendApiFunctionMap(dir), seen: make(map[string]bool)}
	v.scanRouterFiles()
	v.scanViewDirs()
	return v.routes, nil
}
