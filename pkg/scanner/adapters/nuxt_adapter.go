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

func (a *NuxtAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	var routes []models.RouteInfo
	seen := make(map[string]bool)

	apiFuncMap := BuildFrontendApiFunctionMap(dir)

	// Candidate page directories
	var activePageDirs []string
	for _, cand := range []string{"app/pages", "pages", "src/pages", "src/app", "app"} {
		p := filepath.Join(dir, cand)
		if dirExists(p) {
			isSubOrParent := false
			for _, existing := range activePageDirs {
				if strings.HasPrefix(cand, existing+"/") || strings.HasPrefix(existing, cand+"/") {
					isSubOrParent = true
					break
				}
			}
			if !isSubOrParent {
				activePageDirs = append(activePageDirs, cand)
			}
		}
	}

	for _, relPageDir := range activePageDirs {
		fullPageDir := filepath.Join(dir, relPageDir)
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

			ext := filepath.Ext(path)
			if ext != ".vue" && ext != ".tsx" && ext != ".jsx" {
				return nil
			}

			relToPages, err := filepath.Rel(fullPageDir, path)
			if err != nil {
				return nil
			}

			relNoExt := strings.TrimSuffix(relToPages, ext)
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

			routeKey := "PAGE " + routePath
			if seen[routeKey] {
				return nil
			}
			seen[routeKey] = true

			fileData, _ := os.ReadFile(path)
			contentStr := string(fileData)

			middleware := "Public"
			if strings.Contains(contentStr, "middleware: 'auth'") ||
				strings.Contains(contentStr, `middleware: "auth"`) ||
				strings.Contains(contentStr, "middleware: ['auth']") ||
				strings.Contains(contentStr, `middleware: ["auth"]`) ||
				strings.Contains(contentStr, "useAuth(") ||
				strings.Contains(contentStr, "requireAuth") {
				middleware = "Protected (Auth Guard)"
			}

			desc := inferFrontendPageDescription(routePath)
			relHandler, _ := filepath.Rel(dir, path)
			targetRequests := ExtractPageTargetRequests(path, contentStr, dir, apiFuncMap)

			routes = append(routes, models.RouteInfo{
				Method:         "PAGE (GET)",
				Path:           routePath,
				Handler:        filepath.ToSlash(relHandler),
				Description:    desc,
				Middleware:     middleware,
				TargetRequests: targetRequests,
			})

			return nil
		})
	}

	// Fallback if no pages folder: check app.vue
	if len(routes) == 0 {
		for _, appFile := range []string{"app.vue", "app/app.vue", "src/App.vue"} {
			appPath := filepath.Join(dir, appFile)
			if util.FileExists(appPath) {
				contentBytes, _ := os.ReadFile(appPath)
				contentStr := string(contentBytes)
				targetRequests := ExtractPageTargetRequests(appPath, contentStr, dir, apiFuncMap)
				routes = append(routes, models.RouteInfo{
					Method:         "PAGE (GET)",
					Path:           "/",
					Handler:        filepath.ToSlash(appFile),
					Description:    "Halaman tunggal utama aplikasi (Single Page)",
					Middleware:     "Public",
					TargetRequests: targetRequests,
				})
				break
			}
		}
	}

	// Scan Nitro server API routes (server/api)
	nitroApiDir := filepath.Join(dir, "server/api")
	if dirExists(nitroApiDir) {
		_ = filepath.WalkDir(nitroApiDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".ts" && ext != ".js" {
				return nil
			}
			rel, _ := filepath.Rel(nitroApiDir, path)
			relNoExt := strings.TrimSuffix(rel, ext)

			method := "ALL"
			endpointPath := "/api/" + filepath.ToSlash(relNoExt)
			if strings.HasSuffix(relNoExt, ".get") {
				method = "GET"
				endpointPath = "/api/" + strings.TrimSuffix(relNoExt, ".get")
			} else if strings.HasSuffix(relNoExt, ".post") {
				method = "POST"
				endpointPath = "/api/" + strings.TrimSuffix(relNoExt, ".post")
			} else if strings.HasSuffix(relNoExt, ".put") {
				method = "PUT"
				endpointPath = "/api/" + strings.TrimSuffix(relNoExt, ".put")
			} else if strings.HasSuffix(relNoExt, ".delete") {
				method = "DELETE"
				endpointPath = "/api/" + strings.TrimSuffix(relNoExt, ".delete")
			}

			routeKey := method + " " + endpointPath
			if !seen[routeKey] {
				seen[routeKey] = true
				relHandler, _ := filepath.Rel(dir, path)
				routes = append(routes, models.RouteInfo{
					Method:      method,
					Path:        endpointPath,
					Handler:     filepath.ToSlash(relHandler),
					Description: "Endpoint API Nitro server internal",
					Middleware:  "Nitro Server",
				})
			}
			return nil
		})
	}

	return routes, nil
}

// InferFrontendPageDescription provides human-readable descriptions for common frontend routes
func InferFrontendPageDescription(routePath string) string {
	switch {
	case routePath == "/":
		return "Dashboard utama katalog proyek, filter pencarian & pemantauan bundle"
	case strings.Contains(routePath, "projects/:id") || strings.Contains(routePath, "projects/[id]"):
		return "Detail proyek, topologi arsitektur, diagnostik & AI Fast Scan"
	case strings.Contains(routePath, "projects"):
		return "Daftar repositori proyek dan ringkasan arsitektur service"
	case strings.Contains(routePath, "ports"):
		return "Cockpit monitoring status port listening dan pendeteksi tabrakan port"
	case strings.Contains(routePath, "settings"):
		return "Pengaturan sistem, konfigurasi LLM multi-provider & Sensored Keys Security Vault"
	case strings.Contains(routePath, "scan"):
		return "Scanner direktori lokal untuk registrasi repositori baru"
	case strings.Contains(routePath, "mcp"):
		return "Katalog server Model Context Protocol (MCP) untuk AI Agent Antigravity"
	case strings.Contains(routePath, "login") || strings.Contains(routePath, "auth"):
		return "Halaman login dan autentikasi pengguna aplikasi"
	case strings.Contains(routePath, "register"):
		return "Halaman pendaftaran pengguna baru"
	case strings.Contains(routePath, "profile") || strings.Contains(routePath, "user"):
		return "Halaman profil dan pengaturan akun pengguna"
	case strings.Contains(routePath, "asset") || strings.Contains(routePath, "aset"):
		return "Halaman manajemen portofolio aset dan kekayaan"
	case strings.Contains(routePath, "family"):
		return "Halaman manajemen anggota keluarga & undangan"
	case strings.Contains(routePath, "finance") || strings.Contains(routePath, "keuangan"):
		return "Halaman pembukuan kas, catatan transaksi & keuangan keluarga"
	case strings.Contains(routePath, "sync"):
		return "Halaman monitor integritas brankas & delta sync real-time"
	case strings.Contains(routePath, "categor"):
		return "Halaman pengelolaan kategori aset"
	default:
		return "Halaman antarmuka interaktif pengguna (" + routePath + ")"
	}
}

func inferFrontendPageDescription(routePath string) string {
	return InferFrontendPageDescription(routePath)
}
