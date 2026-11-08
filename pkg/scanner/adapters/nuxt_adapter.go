package adapters

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
)

type NuxtAdapter struct{}

func NewNuxtAdapter() *NuxtAdapter {
	return &NuxtAdapter{}
}

func (a *NuxtAdapter) Name() string {
	return "Nuxt (Nuxt 3 / Nuxt 2)"
}

func (a *NuxtAdapter) Type() string {
	return "frontend"
}

func (a *NuxtAdapter) Detect(dir string) bool {
	for _, f := range []string{"nuxt.config.ts", "nuxt.config.js", "nuxt.config.mjs"} {
		if util.FileExists(filepath.Join(dir, f)) {
			return true
		}
	}
	pkgPath := filepath.Join(dir, "package.json")
	if util.FileExists(pkgPath) {
		content := util.ReadFileString(pkgPath)
		if strings.Contains(content, `"nuxt"`) || strings.Contains(content, `"@nuxt/`) {
			return true
		}
	}
	// Nuxt 3 directory conventions
	if util.FileExists(filepath.Join(dir, "app.vue")) || dirExists(filepath.Join(dir, "app/pages")) || dirExists(filepath.Join(dir, "app/composables")) {
		return true
	}
	return false
}

// activePageDirs memilih direktori kandidat yang ada dan bukan induk/anak dari yang sudah terpilih.
func activePageDirs(dir string, candidates []string) []string {
	var active []string
	for _, cand := range candidates {
		if !dirExists(filepath.Join(dir, cand)) {
			continue
		}
		isSubOrParent := false
		for _, existing := range active {
			if strings.HasPrefix(cand, existing+"/") || strings.HasPrefix(existing, cand+"/") {
				isSubOrParent = true
				break
			}
		}
		if !isSubOrParent {
			active = append(active, cand)
		}
	}
	return active
}

// nuxtScan = keadaan satu kali pemindaian proyek Nuxt.
type nuxtScan struct {
	dir        string
	apiFuncMap map[string]string
	seen       map[string]bool
	routes     []models.RouteInfo
}

var nuxtAuthMarkers = []string{
	"middleware: 'auth'", `middleware: "auth"`, "middleware: ['auth']", `middleware: ["auth"]`,
	"useAuth(", "requireAuth",
}

func nuxtMiddleware(content string) string {
	for _, m := range nuxtAuthMarkers {
		if strings.Contains(content, m) {
			return "Protected (Auth Guard)"
		}
	}
	return "Public"
}

// nuxtPageRoutePath mengubah path berkas halaman (tanpa ekstensi) menjadi path rute Nuxt.
func nuxtPageRoutePath(relNoExt string) string {
	routePath := "/" + filepath.ToSlash(relNoExt)
	if routePath == "/index" || routePath == "/page" {
		routePath = "/"
	} else if strings.HasSuffix(routePath, "/index") {
		routePath = strings.TrimSuffix(routePath, "/index")
	} else if strings.HasSuffix(routePath, "/page") {
		routePath = strings.TrimSuffix(routePath, "/page")
	}

	// Nuxt 3 params: [id] -> :id, [...slug] -> *
	// Nuxt 2 params: _id -> :id
	routePath = strings.ReplaceAll(routePath, "[...", "*")
	routePath = strings.ReplaceAll(routePath, "[", ":")
	routePath = strings.ReplaceAll(routePath, "]", "")
	if strings.Contains(routePath, "/_") {
		routePath = strings.ReplaceAll(routePath, "/_", "/:")
	}
	return routePath
}

func (n *nuxtScan) addPage(fullPageDir, path string) {
	ext := filepath.Ext(path)
	if ext != ".vue" && ext != ".tsx" && ext != ".jsx" {
		return
	}
	relToPages, err := filepath.Rel(fullPageDir, path)
	if err != nil {
		return
	}
	routePath := nuxtPageRoutePath(strings.TrimSuffix(relToPages, ext))
	routeKey := "PAGE " + routePath
	if n.seen[routeKey] {
		return
	}
	n.seen[routeKey] = true

	fileData, _ := os.ReadFile(path)
	contentStr := string(fileData)
	relHandler, _ := filepath.Rel(n.dir, path)
	n.routes = append(n.routes, models.RouteInfo{
		Method:         "PAGE (GET)",
		Path:           routePath,
		Handler:        filepath.ToSlash(relHandler),
		Description:    inferFrontendPageDescription(routePath),
		Middleware:     nuxtMiddleware(contentStr),
		TargetRequests: ExtractPageTargetRequests(path, contentStr, n.dir, n.apiFuncMap),
	})
}

func (n *nuxtScan) scanPages() {
	for _, relPageDir := range activePageDirs(n.dir, []string{"app/pages", "pages", "src/pages", "src/app", "app"}) {
		fullPageDir := filepath.Join(n.dir, relPageDir)
		_ = filepath.WalkDir(fullPageDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if shouldIgnoreName(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			n.addPage(fullPageDir, path)
			return nil
		})
	}
}

// scanAppFallback: bila tidak ada folder pages, pakai app.vue sebagai halaman tunggal.
func (n *nuxtScan) scanAppFallback() {
	for _, appFile := range []string{"app.vue", "app/app.vue", "src/App.vue"} {
		appPath := filepath.Join(n.dir, appFile)
		if !util.FileExists(appPath) {
			continue
		}
		contentBytes, _ := os.ReadFile(appPath)
		n.routes = append(n.routes, models.RouteInfo{
			Method:         "PAGE (GET)",
			Path:           "/",
			Handler:        filepath.ToSlash(appFile),
			Description:    "Halaman tunggal utama aplikasi (Single Page)",
			Middleware:     "Public",
			TargetRequests: ExtractPageTargetRequests(appPath, string(contentBytes), n.dir, n.apiFuncMap),
		})
		return
	}
}

var nitroMethodSuffixes = []struct{ suffix, method string }{
	{".get", "GET"}, {".post", "POST"}, {".put", "PUT"}, {".delete", "DELETE"},
}

// nitroEndpoint menurunkan method dan path endpoint dari nama berkas server/api (tanpa ekstensi).
func nitroEndpoint(relNoExt string) (method, endpointPath string) {
	for _, m := range nitroMethodSuffixes {
		if strings.HasSuffix(relNoExt, m.suffix) {
			return m.method, "/api/" + strings.TrimSuffix(relNoExt, m.suffix)
		}
	}
	return "ALL", "/api/" + filepath.ToSlash(relNoExt)
}

// scanNitroAPI memindai route API server Nitro (server/api).
func (n *nuxtScan) scanNitroAPI() {
	nitroApiDir := filepath.Join(n.dir, "server/api")
	if !dirExists(nitroApiDir) {
		return
	}
	_ = filepath.WalkDir(nitroApiDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".ts" && ext != ".js" {
			return nil
		}
		rel, _ := filepath.Rel(nitroApiDir, path)
		method, endpointPath := nitroEndpoint(strings.TrimSuffix(rel, ext))

		routeKey := method + " " + endpointPath
		if n.seen[routeKey] {
			return nil
		}
		n.seen[routeKey] = true
		relHandler, _ := filepath.Rel(n.dir, path)
		n.routes = append(n.routes, models.RouteInfo{
			Method:      method,
			Path:        endpointPath,
			Handler:     filepath.ToSlash(relHandler),
			Description: "Endpoint API Nitro server internal",
			Middleware:  "Nitro Server",
		})
		return nil
	})
}

func (a *NuxtAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	n := &nuxtScan{dir: dir, apiFuncMap: BuildFrontendApiFunctionMap(dir), seen: make(map[string]bool)}
	n.scanPages()
	if len(n.routes) == 0 {
		n.scanAppFallback()
	}
	n.scanNitroAPI()
	return n.routes, nil
}

// pageDescriptionRules diperiksa berurutan; kecocokan pertama menang.
var pageDescriptionRules = []struct {
	keys []string
	desc string
}{
	{[]string{"projects/:id", "projects/[id]"}, "Detail proyek, topologi arsitektur, diagnostik & AI Fast Scan"},
	{[]string{"projects"}, "Daftar repositori proyek dan ringkasan arsitektur service"},
	{[]string{"ports"}, "Cockpit monitoring status port listening dan pendeteksi tabrakan port"},
	{[]string{"settings"}, "Pengaturan sistem, konfigurasi LLM multi-provider & Sensored Keys Security Vault"},
	{[]string{"scan"}, "Scanner direktori lokal untuk registrasi repositori baru"},
	{[]string{"mcp"}, "Katalog server Model Context Protocol (MCP) untuk AI Agent Antigravity"},
	{[]string{"login", "auth"}, "Halaman login dan autentikasi pengguna aplikasi"},
	{[]string{"register"}, "Halaman pendaftaran pengguna baru"},
	{[]string{"profile", "user"}, "Halaman profil dan pengaturan akun pengguna"},
	{[]string{"asset", "aset"}, "Halaman manajemen portofolio aset dan kekayaan"},
	{[]string{"family"}, "Halaman manajemen anggota keluarga & undangan"},
	{[]string{"finance", "keuangan"}, "Halaman pembukuan kas, catatan transaksi & keuangan keluarga"},
	{[]string{"sync"}, "Halaman monitor integritas brankas & delta sync real-time"},
	{[]string{"categor"}, "Halaman pengelolaan kategori aset"},
}

// InferFrontendPageDescription provides human-readable descriptions for common frontend routes
func InferFrontendPageDescription(routePath string) string {
	if routePath == "/" {
		return "Dashboard utama katalog proyek, filter pencarian & pemantauan bundle"
	}
	for _, r := range pageDescriptionRules {
		for _, k := range r.keys {
			if strings.Contains(routePath, k) {
				return r.desc
			}
		}
	}
	return "Halaman antarmuka interaktif pengguna (" + routePath + ")"
}

func inferFrontendPageDescription(routePath string) string {
	return InferFrontendPageDescription(routePath)
}
