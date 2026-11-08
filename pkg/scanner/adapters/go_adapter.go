package adapters

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
)

type GoAdapter struct{}

func NewGoAdapter() *GoAdapter {
	return &GoAdapter{}
}

func (a *GoAdapter) Name() string {
	return "Go (Fiber / Gin / Echo / Chi / Mux / net/http)"
}

func (a *GoAdapter) Type() string {
	return "backend"
}

func (a *GoAdapter) Detect(dir string) bool {
	if util.FileExists(filepath.Join(dir, "go.mod")) {
		return true
	}
	// Check if any .go file exists
	found := false
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() && shouldIgnoreName(d.Name()) {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".go") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func (a *GoAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	var routes []models.RouteInfo
	scannedFiles := make(map[string]bool)

	candidateFiles := []string{
		"main.go",
		"server.go",
		"routes.go",
		"router.go",
		"app.go",
		"api/server.go",
		"internal/api/server.go",
		"internal/routes/routes.go",
		"internal/router/router.go",
		"cmd/server/main.go",
		"cmd/main.go",
	}

	for _, cf := range candidateFiles {
		p := filepath.Join(dir, cf)
		if util.FileExists(p) && !scannedFiles[p] {
			scannedFiles[p] = true
			r := ParseGoRoutesFromFile(p)
			if len(r) > 0 {
				routes = append(routes, r...)
			}
		}
	}

	// Scan routes/ and router/ folders
	for _, sub := range []string{"routes", "router", "routing", "api", "internal/routes", "internal/router", "internal/api"} {
		subPath := filepath.Join(dir, sub)
		if dirExists(subPath) {
			entries, _ := os.ReadDir(subPath)
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
					p := filepath.Join(subPath, e.Name())
					if !scannedFiles[p] {
						scannedFiles[p] = true
						r := ParseGoRoutesFromFile(p)
						if len(r) > 0 {
							routes = append(routes, r...)
						}
					}
				}
			}
		}
	}

	// If routes are still empty, walk the entire repo for .go files
	if len(routes) == 0 {
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if shouldIgnoreName(d.Name()) || d.Name() == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") && !scannedFiles[path] {
				if fileMayContainRoutes(path) {
					scannedFiles[path] = true
					r := ParseGoRoutesFromFile(path)
					if len(r) > 0 {
						routes = append(routes, r...)
					}
				}
			}
			return nil
		})
	}

	return routes, nil
}

func fileMayContainRoutes(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineCount := 0
	for scanner.Scan() && lineCount < 200 {
		line := scanner.Text()
		lineCount++
		if strings.Contains(line, ".Get(") || strings.Contains(line, ".Post(") ||
			strings.Contains(line, ".GET(") || strings.Contains(line, ".POST(") ||
			strings.Contains(line, ".HandleFunc(") || strings.Contains(line, ".Handle(") ||
			strings.Contains(line, ".Group(") || strings.Contains(line, "Route(") {
			return true
		}
	}
	return false
}

// ParseGoRoutesFromFile parses routes from a single Go source file
func ParseGoRoutesFromFile(filePath string) []models.RouteInfo {
	var routes []models.RouteInfo

	file, err := os.Open(filePath)
	if err != nil {
		return routes
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	groupMap := make(map[string]string)

	groupRegex := regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\s*:?=\s*[a-zA-Z0-9_.]+\.Group\(\s*["']([^"']+)["']`)
	routeMethodRegex := regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\.(Get|Post|Put|Delete|Patch|Options|Head|GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)\(\s*["']([^"']+)["']\s*,\s*([^)]+)\)`)
	handleFuncMethodsRegex := regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\.HandleFunc\(\s*["']([^"']+)["']\s*,\s*([a-zA-Z0-9_.]+)\s*\)\.Methods\(\s*([^)]+)\)`)
	netHttpHandleFuncRegex := regexp.MustCompile(`(?m)(?:http|[a-zA-Z0-9_]+)\.HandleFunc\(\s*["'](?:(GET|POST|PUT|DELETE|PATCH)\s+)?([^"']+)["']\s*,\s*([a-zA-Z0-9_.]+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}

		// Track route groups
		if gm := groupRegex.FindStringSubmatch(line); len(gm) >= 3 {
			varName := gm[1]
			prefix := gm[2]
			for parentVar, parentPrefix := range groupMap {
				if strings.Contains(line, parentVar+".Group") {
					prefix = CleanURLPath(parentPrefix + "/" + prefix)
					break
				}
			}
			groupMap[varName] = CleanURLPath(prefix)
			continue
		}

		// Fiber / Gin / Echo routes
		if rm := routeMethodRegex.FindStringSubmatch(line); len(rm) >= 5 {
			varName := rm[1]
			method := strings.ToUpper(rm[2])
			path := rm[3]
			handlerPart := strings.TrimSpace(rm[4])

			if prefix, ok := groupMap[varName]; ok {
				path = CleanURLPath(prefix + "/" + path)
			} else {
				path = CleanURLPath(path)
			}

			handler := cleanGoHandlerName(handlerPart)
			middleware := "Public"
			if strings.Contains(line, "auth") || strings.Contains(line, "jwt") || strings.Contains(line, "Protect") {
				middleware = "Protected (Auth Guard)"
			}

			routes = append(routes, models.RouteInfo{
				Method:      method,
				Path:        path,
				Handler:     handler,
				Description: describeGoHandler(handler, method, path),
				Middleware:  middleware,
			})
			continue
		}

		// Fiber / Gin / Echo inline closure routes: app.Get("/path", func(...)
		inlineRouteRegex := regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\.(Get|Post|Put|Delete|Patch|Options|Head|GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)\(\s*["']([^"']+)["']\s*,`)
		if im := inlineRouteRegex.FindStringSubmatch(line); len(im) >= 4 {
			varName := im[1]
			method := strings.ToUpper(im[2])
			path := im[3]

			if prefix, ok := groupMap[varName]; ok {
				path = CleanURLPath(prefix + "/" + path)
			} else {
				path = CleanURLPath(path)
			}

			middleware := "Public"
			if strings.Contains(line, "auth") || strings.Contains(line, "jwt") || strings.Contains(line, "Protect") {
				middleware = "Protected (Auth Guard)"
			}

			routes = append(routes, models.RouteInfo{
				Method:      method,
				Path:        path,
				Handler:     filepath.Base(filePath) + ":inline_handler",
				Description: describeGoHandler("inline_handler", method, path),
				Middleware:  middleware,
			})
			continue
		}

		// Gorilla Mux: r.HandleFunc("/path", handler).Methods("GET", "POST")
		if mm := handleFuncMethodsRegex.FindStringSubmatch(line); len(mm) >= 5 {
			varName := mm[1]
			path := mm[2]
			handler := cleanGoHandlerName(mm[3])
			rawMethods := mm[4]

			if prefix, ok := groupMap[varName]; ok {
				path = CleanURLPath(prefix + "/" + path)
			} else {
				path = CleanURLPath(path)
			}

			methodListRegex := regexp.MustCompile(`["'](GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)["']`)
			methods := methodListRegex.FindAllStringSubmatch(rawMethods, -1)
			if len(methods) == 0 {
				routes = append(routes, models.RouteInfo{
					Method:      "ALL",
					Path:        path,
					Handler:     handler,
					Description: inferEndpointDescription("ALL", path),
					Middleware:  "Public",
				})
			} else {
				for _, m := range methods {
					routes = append(routes, models.RouteInfo{
						Method:      strings.ToUpper(m[1]),
						Path:        path,
						Handler:     handler,
						Description: inferEndpointDescription(strings.ToUpper(m[1]), path),
						Middleware:  "Public",
					})
				}
			}
			continue
		}

		// Standard net/http: http.HandleFunc("GET /api/v1/ping", PingHandler)
		if nh := netHttpHandleFuncRegex.FindStringSubmatch(line); len(nh) >= 4 {
			method := "ALL"
			if nh[1] != "" {
				method = strings.ToUpper(nh[1])
			}
			path := CleanURLPath(nh[2])
			handler := cleanGoHandlerName(nh[3])

			routes = append(routes, models.RouteInfo{
				Method:      method,
				Path:        path,
				Handler:     handler,
				Description: inferEndpointDescription(method, path),
				Middleware:  "Public",
			})
			continue
		}
	}

	return routes
}

func cleanGoHandlerName(raw string) string {
	parts := strings.Split(raw, ",")
	lastPart := strings.TrimSpace(parts[len(parts)-1])
	lastPart = strings.TrimSuffix(lastPart, ")")
	lastPart = strings.TrimSpace(lastPart)
	if strings.HasPrefix(lastPart, "func(") {
		return "Inline Handler"
	}
	return lastPart
}

func describeGoHandler(handler, method, path string) string {
	dict := map[string]string{
		"controllers.Register":           "Mendaftarkan akun user baru, inisialisasi vault keluarga E2EE, dan seeder dompet kas",
		"controllers.Login":              "Autentikasi akun via email dan password, menghasilkan JWT access token & refresh token",
		"controllers.GoogleLogin":        "Autentikasi SSO pihak ketiga via Google ID token / Firebase Auth",
		"controllers.RefreshToken":       "Memperbarui access token kadaluarsa menggunakan refresh token valid",
		"controllers.Logout":             "Mencabut (revoke) refresh token dan mengakhiri sesi aktif pengguna",
		"controllers.GetCurrentUser":     "Mengambil data profil pengguna terotentikasi, keluarga aktif, dan undangan pending",
		"controllers.UpdateUserKeys":     "Memperbarui public key kriptografi dan encrypted family key pengguna",
		"controllers.SearchUser":         "Mencari pengguna terdaftar berdasarkan email/username untuk undangan keluarga",
		"controllers.RegisterFCMToken":   "Mendaftarkan token perangkat FCM untuk menerima notifikasi push",
		"controllers.UploadAvatar":       "Mengunggah atau memperbarui foto profil avatar pengguna",
		"controllers.DeleteAvatar":       "Menghapus foto profil avatar kustom pengguna",
		"controllers.GetFamilyDetails":   "Mengambil detail informasi keluarga, kuota storage, dan seluruh anggota",
		"controllers.GetUserInvitations": "Melihat daftar undangan bergabung ke keluarga yang berstatus pending",
		"controllers.AcceptInvitation":   "Menerima undangan keluarga dan membuka akses ke enkripsi family key",
		"controllers.RejectInvitation":   "Menolak undangan bergabung ke grup keluarga",
		"controllers.AddFamilyMember":    "Mengundang anggota baru dengan enkripsi family key menggunakan public key user",
		"controllers.UpdateMemberKey":    "Memperbarui encrypted family key anggota saat rotasi kunci",
		"controllers.DeleteFamilyMember": "Menghapus anggota dari grup keluarga (khusus admin)",
		"controllers.DeleteImages":       "Menghapus berkas gambar fisik terenkripsi (.enc) dari disk server dan database",
		"controllers.CreateAsset":        "Menambahkan data aset keluarga baru dengan payload terenkripsi E2EE",
		"controllers.GetAssets":          "Mengambil daftar seluruh aset keluarga dengan filter dan paginasi",
		"controllers.GetAsset":           "Mengambil detail satu data aset keluarga beserta daftar gambar lampirannya",
		"controllers.UpdateAsset":        "Memperbarui metadata dan lampiran gambar aset terenkripsi",
		"controllers.DeleteAsset":        "Menghapus data aset keluarga dan mengembalikan sisa kuota penyimpanan",
		"controllers.UploadImages":       "Mengunggah gambar lampiran terenkripsi (.enc) ke direktori server",
	}

	if desc, found := dict[handler]; found {
		return desc
	}

	return inferEndpointDescription(method, path)
}

func inferEndpointDescription(method, path string) string {
	switch {
	case strings.Contains(path, "/health") || strings.Contains(path, "/ping"):
		return "Health check status & liveness probe"
	case strings.Contains(path, "/auth/login") || strings.Contains(path, "/login"):
		return "Otentikasi & pembuatan session token pengguna"
	case strings.Contains(path, "/auth/register") || strings.Contains(path, "/register"):
		return "Pendaftaran akun pengguna baru"
	case strings.Contains(path, "/auth/refresh"):
		return "Refresh access token kadaluarsa"
	case strings.Contains(path, "/auth/logout"):
		return "Revokasi token & logout pengguna"
	case strings.Contains(path, "/users") && method == "GET":
		return "Daftar pengguna terdaftar"
	case strings.Contains(path, "/users") && method == "POST":
		return "Pembuatan pengguna baru"
	case strings.Contains(path, "/assets") && method == "GET":
		return "Daftar portofolio aset & kekayaan"
	case strings.Contains(path, "/assets") && method == "POST":
		return "Pencatatan aset baru ke dalam database"
	case strings.Contains(path, "/family"):
		return "Manajemen data keluarga & anggota"
	case strings.Contains(path, "/projects") && method == "GET":
		return "Daftar repositori proyek tersimpan"
	case strings.Contains(path, "/projects") && method == "POST":
		return "Pendaftaran repositori proyek baru"
	case method == "GET":
		return "Pengambilan data " + path
	case method == "POST":
		return "Pembuatan sumber daya baru pada " + path
	case method == "PUT" || method == "PATCH":
		return "Pembaruan data pada " + path
	case method == "DELETE":
		return "Penghapusan sumber daya pada " + path
	default:
		return "Endpoint handler untuk " + path
	}
}
