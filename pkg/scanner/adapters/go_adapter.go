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

// goScan = keadaan satu kali pemindaian proyek Go.
type goScan struct {
	routes       []models.RouteInfo
	scannedFiles map[string]bool
}

func (g *goScan) parseOnce(p string) {
	g.scannedFiles[p] = true
	g.routes = append(g.routes, ParseGoRoutesFromFile(p)...)
}

var goCandidateFiles = []string{
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

func (g *goScan) scanCandidateFiles(dir string) {
	for _, cf := range goCandidateFiles {
		p := filepath.Join(dir, cf)
		if util.FileExists(p) && !g.scannedFiles[p] {
			g.parseOnce(p)
		}
	}
}

// scanRouteFolders memindai folder routes/ dan router/.
func (g *goScan) scanRouteFolders(dir string) {
	for _, sub := range []string{"routes", "router", "routing", "api", "internal/routes", "internal/router", "internal/api"} {
		subPath := filepath.Join(dir, sub)
		if !dirExists(subPath) {
			continue
		}
		entries, _ := os.ReadDir(subPath)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			p := filepath.Join(subPath, e.Name())
			if !g.scannedFiles[p] {
				g.parseOnce(p)
			}
		}
	}
}

// walkAll menelusuri seluruh repo untuk berkas .go yang tampak memuat rute.
func (g *goScan) walkAll(dir string) {
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
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") && !g.scannedFiles[path] && fileMayContainRoutes(path) {
			g.parseOnce(path)
		}
		return nil
	})
}

func (a *GoAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	g := &goScan{scannedFiles: make(map[string]bool)}
	g.scanCandidateFiles(dir)
	g.scanRouteFolders(dir)
	// If routes are still empty, walk the entire repo for .go files
	if len(g.routes) == 0 {
		g.walkAll(dir)
	}
	return g.routes, nil
}

var routeMarkers = []string{".Get(", ".Post(", ".GET(", ".POST(", ".HandleFunc(", ".Handle(", ".Group(", "Route("}

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
		for _, m := range routeMarkers {
			if strings.Contains(line, m) {
				return true
			}
		}
	}
	return false
}

var (
	goGroupRegex          = regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\s*:?=\s*[a-zA-Z0-9_.]+\.Group\(\s*["']([^"']+)["']`)
	goRouteMethodRegex    = regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\.(Get|Post|Put|Delete|Patch|Options|Head|GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)\(\s*["']([^"']+)["']\s*,\s*([^)]+)\)`)
	goHandleFuncMethodsRe = regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\.HandleFunc\(\s*["']([^"']+)["']\s*,\s*([a-zA-Z0-9_.]+)\s*\)\.Methods\(\s*([^)]+)\)`)
	goNetHTTPHandleFuncRe = regexp.MustCompile(`(?m)(?:http|[a-zA-Z0-9_]+)\.HandleFunc\(\s*["'](?:(GET|POST|PUT|DELETE|PATCH)\s+)?([^"']+)["']\s*,\s*([a-zA-Z0-9_.]+)`)
	goInlineRouteRegex    = regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\.(Get|Post|Put|Delete|Patch|Options|Head|GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)\(\s*["']([^"']+)["']\s*,`)
	goMethodListRegex     = regexp.MustCompile(`["'](GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)["']`)
)

// goRouteParser mem-parse satu berkas Go baris demi baris sambil melacak grup rute.
type goRouteParser struct {
	filePath string
	groupMap map[string]string
	routes   []models.RouteInfo
}

// trackGroup mencatat deklarasi grup rute; true bila baris adalah deklarasi grup.
func (p *goRouteParser) trackGroup(line string) bool {
	gm := goGroupRegex.FindStringSubmatch(line)
	if len(gm) < 3 {
		return false
	}
	varName := gm[1]
	prefix := gm[2]
	for parentVar, parentPrefix := range p.groupMap {
		if strings.Contains(line, parentVar+".Group") {
			prefix = CleanURLPath(parentPrefix + "/" + prefix)
			break
		}
	}
	p.groupMap[varName] = CleanURLPath(prefix)
	return true
}

// withGroupPrefix menambahkan prefiks grup milik varName (bila ada) ke path.
func (p *goRouteParser) withGroupPrefix(varName, path string) string {
	if prefix, ok := p.groupMap[varName]; ok {
		return CleanURLPath(prefix + "/" + path)
	}
	return CleanURLPath(path)
}

func goMiddleware(line string) string {
	if strings.Contains(line, "auth") || strings.Contains(line, "jwt") || strings.Contains(line, "Protect") {
		return "Protected (Auth Guard)"
	}
	return "Public"
}

// tryRouteMethod: Fiber / Gin / Echo routes.
func (p *goRouteParser) tryRouteMethod(line string) bool {
	rm := goRouteMethodRegex.FindStringSubmatch(line)
	if len(rm) < 5 {
		return false
	}
	method := strings.ToUpper(rm[2])
	path := p.withGroupPrefix(rm[1], rm[3])
	handler := cleanGoHandlerName(strings.TrimSpace(rm[4]))
	p.routes = append(p.routes, models.RouteInfo{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Description: describeGoHandler(handler, method, path),
		Middleware:  goMiddleware(line),
	})
	return true
}

// tryInlineRoute: Fiber / Gin / Echo inline closure routes: app.Get("/path", func(...)
func (p *goRouteParser) tryInlineRoute(line string) bool {
	im := goInlineRouteRegex.FindStringSubmatch(line)
	if len(im) < 4 {
		return false
	}
	method := strings.ToUpper(im[2])
	path := p.withGroupPrefix(im[1], im[3])
	p.routes = append(p.routes, models.RouteInfo{
		Method:      method,
		Path:        path,
		Handler:     filepath.Base(p.filePath) + ":inline_handler",
		Description: describeGoHandler("inline_handler", method, path),
		Middleware:  goMiddleware(line),
	})
	return true
}

// tryMuxRoute: Gorilla Mux: r.HandleFunc("/path", handler).Methods("GET", "POST")
func (p *goRouteParser) tryMuxRoute(line string) bool {
	mm := goHandleFuncMethodsRe.FindStringSubmatch(line)
	if len(mm) < 5 {
		return false
	}
	path := p.withGroupPrefix(mm[1], mm[2])
	handler := cleanGoHandlerName(mm[3])

	methods := goMethodListRegex.FindAllStringSubmatch(mm[4], -1)
	if len(methods) == 0 {
		p.routes = append(p.routes, models.RouteInfo{
			Method:      "ALL",
			Path:        path,
			Handler:     handler,
			Description: inferEndpointDescription("ALL", path),
			Middleware:  "Public",
		})
		return true
	}
	for _, m := range methods {
		p.routes = append(p.routes, models.RouteInfo{
			Method:      strings.ToUpper(m[1]),
			Path:        path,
			Handler:     handler,
			Description: inferEndpointDescription(strings.ToUpper(m[1]), path),
			Middleware:  "Public",
		})
	}
	return true
}

// tryNetHTTPRoute: Standard net/http: http.HandleFunc("GET /api/v1/ping", PingHandler)
func (p *goRouteParser) tryNetHTTPRoute(line string) bool {
	nh := goNetHTTPHandleFuncRe.FindStringSubmatch(line)
	if len(nh) < 4 {
		return false
	}
	method := "ALL"
	if nh[1] != "" {
		method = strings.ToUpper(nh[1])
	}
	path := CleanURLPath(nh[2])
	p.routes = append(p.routes, models.RouteInfo{
		Method:      method,
		Path:        path,
		Handler:     cleanGoHandlerName(nh[3]),
		Description: inferEndpointDescription(method, path),
		Middleware:  "Public",
	})
	return true
}

// handleLine mencoba tiap pola berurutan; pola pertama yang cocok menang.
func (p *goRouteParser) handleLine(line string) {
	for _, try := range []func(string) bool{p.trackGroup, p.tryRouteMethod, p.tryInlineRoute, p.tryMuxRoute, p.tryNetHTTPRoute} {
		if try(line) {
			return
		}
	}
}

// ParseGoRoutesFromFile parses routes from a single Go source file
func ParseGoRoutesFromFile(filePath string) []models.RouteInfo {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	p := &goRouteParser{filePath: filePath, groupMap: make(map[string]string)}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}
		p.handleLine(line)
	}
	return p.routes
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

// endpointDescRule: contains = salah satu substring path; method "" berarti semua metode.
type endpointDescRule struct {
	contains []string
	method   string
	desc     string
}

var endpointDescRules = []endpointDescRule{
	{[]string{"/health", "/ping"}, "", "Health check status & liveness probe"},
	{[]string{"/auth/login", "/login"}, "", "Otentikasi & pembuatan session token pengguna"},
	{[]string{"/auth/register", "/register"}, "", "Pendaftaran akun pengguna baru"},
	{[]string{"/auth/refresh"}, "", "Refresh access token kadaluarsa"},
	{[]string{"/auth/logout"}, "", "Revokasi token & logout pengguna"},
	{[]string{"/users"}, "GET", "Daftar pengguna terdaftar"},
	{[]string{"/users"}, "POST", "Pembuatan pengguna baru"},
	{[]string{"/assets"}, "GET", "Daftar portofolio aset & kekayaan"},
	{[]string{"/assets"}, "POST", "Pencatatan aset baru ke dalam database"},
	{[]string{"/family"}, "", "Manajemen data keluarga & anggota"},
	{[]string{"/projects"}, "GET", "Daftar repositori proyek tersimpan"},
	{[]string{"/projects"}, "POST", "Pendaftaran repositori proyek baru"},
}

func inferEndpointDescription(method, path string) string {
	for _, r := range endpointDescRules {
		if r.method != "" && r.method != method {
			continue
		}
		for _, c := range r.contains {
			if strings.Contains(path, c) {
				return r.desc
			}
		}
	}
	switch method {
	case "GET":
		return "Pengambilan data " + path
	case "POST":
		return "Pembuatan sumber daya baru pada " + path
	case "PUT", "PATCH":
		return "Pembaruan data pada " + path
	case "DELETE":
		return "Penghapusan sumber daya pada " + path
	}
	return "Endpoint handler untuk " + path
}
