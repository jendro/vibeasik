package analysis

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/scanner/adapters"
)

// SyncComponentRoutes synchronizes routes between the source code on disk and the database.
// Rules:
// 1. If route exists in both source code and DB: do not duplicate; preserve enriched description/domain; update handler/middleware if changed.
// 2. If route is in DB but no longer exists in source code: delete from DB.
// 3. If route is in source code but not yet in DB: insert into DB with initial inferred domain & description.
func SyncComponentRoutes(db *gorm.DB, comp *models.Component, proj *models.Project) (*models.RouteSyncResult, error) {
	if comp == nil {
		return nil, fmt.Errorf("component is nil")
	}

	absDir := comp.AbsolutePath
	if absDir == "" && proj != nil && proj.RootPath != "" {
		absDir = filepath.Join(proj.RootPath, comp.RelativePath)
	}

	autoDetectCodeIgniter(db, comp, absDir)

	// 1. Extract routes from source code via static code inspection
	scanned := extractRoutes(absDir, comp.Role)

	// 2. Load existing routes from DB
	var existing []models.ComponentRoute
	if err := db.Where("component_id = ?", comp.ID).Find(&existing).Error; err != nil {
		return nil, fmt.Errorf("failed to load existing routes: %w", err)
	}

	// If DB is completely empty but FastScanData already contains routes, merge them first
	if len(existing) == 0 && comp.FastScanData != "" {
		seedRoutesFromFastScan(db, comp, &existing)
	}

	// 3. Build lookup maps
	// Key: METHOD + " " + PATH (normalized uppercase method and trimmed path)
	existingMap := make(map[string]*models.ComponentRoute)
	for i := range existing {
		key := normalizeRouteKey(existing[i].Method, existing[i].Path)
		existingMap[key] = &existing[i]
	}

	codeKeySet := make(map[string]bool)
	addedCount := 0
	updatedCount := 0
	deletedCount := 0

	for _, sc := range scanned {
		key := normalizeRouteKey(sc.Method, sc.Path)
		codeKeySet[key] = true
		targetReqsJSON := marshalTargetRequests(sc.TargetRequests)

		if cur, found := existingMap[key]; found {
			// In source code and in DB: keep, do not duplicate
			if mergeScannedRoute(cur, sc, targetReqsJSON) {
				_ = db.Save(cur).Error
				updatedCount++
			}
		} else {
			// In source code but NOT in DB: add new
			if err := db.Create(newComponentRoute(comp.ID, sc, targetReqsJSON)).Error; err == nil {
				addedCount++
			}
		}
	}

	// 4. "jika hilang dihapus": delete routes present in DB that no longer exist in code
	// Only delete if scanned found at least one route from code (avoid accidental wiping if dir was temporarily unreadable)
	if len(scanned) > 0 {
		deletedCount = deleteMissingRoutes(db, existingMap, codeKeySet)
	}

	// 5. Query final routes
	var finalRoutes []models.ComponentRoute
	_ = db.Where("component_id = ?", comp.ID).Order("domain asc, path asc").Find(&finalRoutes)

	// 6. Update FastScanData cache if exists
	syncFastScanDataCache(db, comp, finalRoutes)

	return &models.RouteSyncResult{
		ComponentID:  comp.ID,
		AddedCount:   addedCount,
		UpdatedCount: updatedCount,
		DeletedCount: deletedCount,
		TotalCount:   len(finalRoutes),
		Routes:       finalRoutes,
	}, nil
}

// deleteMissingRoutes menghapus rute DB yang tak ada lagi di kode; mengembalikan jumlah yang dihapus.
func deleteMissingRoutes(db *gorm.DB, existingMap map[string]*models.ComponentRoute, codeKeySet map[string]bool) int {
	deleted := 0
	for key, cur := range existingMap {
		if !codeKeySet[key] {
			db.Delete(cur)
			deleted++
		}
	}
	return deleted
}

// autoDetectCodeIgniter mengisi bahasa/framework komponen CodeIgniter bila belum ditentukan.
func autoDetectCodeIgniter(db *gorm.DB, comp *models.Component, absDir string) {
	if !(comp.Language == "auto" || comp.Language == "" || comp.Framework == "") {
		return
	}
	if !(util.PathExists(filepath.Join(absDir, "spark")) || util.PathExists(filepath.Join(absDir, "app/Config/Routes.php"))) {
		return
	}
	comp.Language = "php"
	comp.Framework = "codeigniter"
	comp.PackageManager = "composer"
	if comp.DevCommand == "" {
		comp.DevCommand = "php spark serve"
	}
	if comp.Port == 0 {
		comp.Port = 8080
	}
	_ = db.Model(comp).Updates(map[string]interface{}{
		"language":        comp.Language,
		"framework":       comp.Framework,
		"package_manager": comp.PackageManager,
		"dev_command":     comp.DevCommand,
		"port":            comp.Port,
	})
}

// marshalTargetRequests menyandikan daftar target request menjadi JSON ("[]" bila kosong).
func marshalTargetRequests(reqs []string) string {
	if len(reqs) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(reqs)
	return string(b)
}

// newComponentRoute membuat rute baru dengan domain & deskripsi hasil inferensi.
func newComponentRoute(componentID uint, sc models.RouteInfo, targetReqsJSON string) *models.ComponentRoute {
	domain, fnGroup := InferRouteDomainAndFunction(sc.Path, sc.Method)
	desc := sc.Description
	if desc == "" {
		desc = InferRouteDescription(sc.Method, sc.Path)
	}
	return &models.ComponentRoute{
		ComponentID:    componentID,
		Method:         sc.Method,
		Path:           sc.Path,
		Handler:        sc.Handler,
		Middleware:     sc.Middleware,
		Description:    desc,
		Domain:         domain,
		FunctionGroup:  fnGroup,
		TargetRequests: targetReqsJSON,
	}
}

// seedRoutesFromFastScan memasukkan rute dari FastScanData ke DB kosong lalu memuat ulang existing.
func seedRoutesFromFastScan(db *gorm.DB, comp *models.Component, existing *[]models.ComponentRoute) {
	var fastScan models.FastScanResult
	if err := json.Unmarshal([]byte(comp.FastScanData), &fastScan); err != nil || len(fastScan.Routes) == 0 {
		return
	}
	for _, r := range fastScan.Routes {
		_ = db.Create(newComponentRoute(comp.ID, r, marshalTargetRequests(r.TargetRequests))).Error
	}
	_ = db.Where("component_id = ?", comp.ID).Find(existing)
}

// mergeScannedRoute memperbarui rute DB dari hasil scan; true bila ada perubahan yang harus disimpan.
func mergeScannedRoute(cur *models.ComponentRoute, sc models.RouteInfo, targetReqsJSON string) bool {
	needsSave := false
	if sc.Handler != "" && cur.Handler != sc.Handler {
		cur.Handler = sc.Handler
		needsSave = true
	}
	if sc.Middleware != "" && cur.Middleware != sc.Middleware {
		cur.Middleware = sc.Middleware
		needsSave = true
	}
	if targetReqsJSON != "[]" && cur.TargetRequests != targetReqsJSON {
		cur.TargetRequests = targetReqsJSON
		needsSave = true
	}
	if cur.Description == "" && sc.Description != "" {
		cur.Description = sc.Description
		needsSave = true
	}
	if cur.Domain == "" || cur.Domain == "Umum" {
		d, f := InferRouteDomainAndFunction(cur.Path, cur.Method)
		cur.Domain = d
		if cur.FunctionGroup == "" || cur.FunctionGroup == "Lainnya" {
			cur.FunctionGroup = f
		}
		needsSave = true
	}
	return needsSave
}

// maxEnrichRoutes: rute per pemanggilan pengayaan LLM; sisanya menunggu pemanggilan berikutnya.
const maxEnrichRoutes = 80

// routeNeedsEnrich: rute belum diperkaya LLM (tag kosong & deskripsi masih bawaan/heuristik). Rute yang
// deskripsinya diubah pengguna tidak disentuh.
func routeNeedsEnrich(r models.ComponentRoute) bool {
	if r.Tags != "" {
		return false
	}
	return r.Description == "" || r.Description == "Endpoint API" || r.Description == InferRouteDescription(r.Method, r.Path)
}

// criticalPurposes: pengecekan quality/security — model aktif (≥ Opus 5.5 menurut Aturan 26), tidak ditimbang turun.
var criticalPurposes = map[string]bool{"audit_review": true, "audit_verify": true}

// cleanJSONResponse strips markdown codeblocks (```json ... ```) and excess whitespace
func cleanJSONResponse(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

func normalizeRouteKey(method, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}


// routeSub: aturan fungsi di dalam sebuah domain. Cocok bila (keys kosong atau path memuat salah satu keys)
// dan (methods kosong atau method sama dengan salah satunya).
type routeSub struct {
	keys    []string
	methods []string
	group   string
}

func (s routeSub) match(p, m string) bool {
	if len(s.keys) > 0 && !util.ContainsAny(p, s.keys...) {
		return false
	}
	if len(s.methods) == 0 {
		return true
	}
	for _, x := range s.methods {
		if m == x {
			return true
		}
	}
	return false
}

// routeDomainRule: domain terpilih bila path memuat salah satu keys; fungsinya ditentukan subs (urut, pertama cocok).
type routeDomainRule struct {
	keys   []string
	domain string
	subs   []routeSub
}

func (r routeDomainRule) group(p, m string) string {
	for _, s := range r.subs {
		if s.match(p, m) {
			return s.group
		}
	}
	return "Lainnya"
}

var routeDomainRules = []routeDomainRule{
	{[]string{"auth", "login", "register", "token", "logout", "password", "oauth"}, "Autentikasi & Keamanan", []routeSub{
		{keys: []string{"login", "token"}, group: "Login & Otorisasi Sesi"},
		{keys: []string{"register"}, group: "Registrasi Pengguna"},
		{keys: []string{"logout"}, group: "Terminasi Sesi"},
		{group: "Manajemen Kredensial"},
	}},
	{[]string{"user", "member", "profile", "role", "permission", "account"}, "Manajemen Pengguna & Akun", []routeSub{
		{methods: []string{"GET"}, group: "Pengambilan Data Pengguna"},
		{methods: []string{"POST", "PUT", "PATCH"}, group: "Pembaruan Profil & Akses"},
		{group: "Operasi Akun"},
	}},
	{[]string{"project", "bundle", "component", "app"}, "Manajemen Proyek & Komponen", []routeSub{
		{keys: []string{"component"}, group: "Manajemen Service & Komponen"},
		{methods: []string{"GET"}, group: "Katalog & Detail Proyek"},
		{group: "Registrasi & Modifikasi Proyek"},
	}},
	{[]string{"asset", "product", "item", "inventory", "category"}, "Aset & Inventori", []routeSub{
		{methods: []string{"GET"}, group: "Katalog & Pencarian Aset"},
		{group: "Mutasi & Kelola Aset"},
	}},
	{[]string{"diag", "health", "ping", "metric", "log", "port", "monitor"}, "Diagnostik & Monitoring", []routeSub{
		{keys: []string{"port"}, group: "Pemeriksaan Status Port"},
		{keys: []string{"health", "ping"}, group: "Health Check & Liveness"},
		{group: "Audit Kesehatan Kode"},
	}},
	{[]string{"llm", "ai", "scan", "review", "advisor"}, "Layanan AI & Fast Scan", []routeSub{
		{keys: []string{"fast-scan", "scan"}, group: "Pemindaian Blueprint & Arsitektur"},
		{keys: []string{"config", "model"}, group: "Konfigurasi Multi-Model AI"},
		{group: "Analisis Cerdas & Audit AI"},
	}},
	{[]string{"security", "censor", "secret", "vault"}, "Keamanan & Redaksi Rahasia", []routeSub{
		{group: "Vault Kata Kunci Terenkripsi"},
	}},
	{[]string{"setting", "config", "preference"}, "Pengaturan & Konfigurasi", []routeSub{
		{group: "Preferensi Sistem"},
	}},
	{[]string{"file", "upload", "download", "media", "document", "dir"}, "Media & File Storage", []routeSub{
		{group: "Manajemen Berkas & Direktori"},
	}},
	{[]string{"kontingen", "official"}, "Portal & Manajemen Kontingen", []routeSub{
		{keys: []string{"official"}, group: "Manajemen Official Kontingen"},
		{keys: []string{"daftar", "ikuti"}, group: "Pendaftaran Kejuaraan"},
		{keys: []string{"tagihan", "rekap"}, group: "Rekapitulasi & Tagihan Kontingen"},
		{group: "Profil & Manajemen Kontingen"},
	}},
	{[]string{"atlet"}, "Portal Atlet & Peserta", []routeSub{
		{keys: []string{"pertandingan"}, group: "Jadwal & Hasil Atlet"},
		{group: "Biodata & Status Atlet"},
	}},
	{[]string{"kejuaraan", "pertandingan", "bagan", "bracket", "kelas", "kategori"}, "Kompetisi & Pertandingan", []routeSub{
		{keys: []string{"bagan", "bracket"}, group: "Bagan Pertandingan & Bracket Tree"},
		{keys: []string{"kelas", "kategori"}, group: "Kategori & Kelas Tanding"},
		{group: "Manajemen Event & Jadwal Tanding"},
	}},
	{[]string{"bayar", "pembayaran", "tagihan", "rekening", "invoice", "billing"}, "Keuangan & Pembayaran", []routeSub{
		{group: "Transaksi & Verifikasi Pembayaran"},
	}},
	{[]string{"public", "portal", "gallery", "galeri", "blog", "news", "lokasi"}, "Portal Informasi Publik", []routeSub{
		{keys: []string{"blog", "news"}, group: "Berita & Artikel"},
		{keys: []string{"gallery", "galeri"}, group: "Dokumentasi & Galeri"},
		{keys: []string{"lokasi"}, group: "Informasi Lokasi & Venue"},
		{group: "Informasi & Publikasi Umum"},
	}},
	{[]string{"admin"}, "Administrasi & Master Data", []routeSub{
		{keys: []string{"verify", "verifikasi"}, group: "Verifikasi & Validasi Data"},
		{keys: []string{"dashboard"}, group: "Dashboard Eksekutif Admin"},
		{group: "Master Data & Pengawasan"},
	}},
}

// routePageRule: aturan halaman frontend (method PAGE*); exact berarti path harus persis sama.
type routePageRule struct {
	keys          []string
	exact         string
	domain, group string
}

var routePageRules = []routePageRule{
	{keys: []string{"project", "bundle", "app"}, domain: "Manajemen Proyek", group: "Katalog & Detail Proyek"},
	{keys: []string{"port", "monitor"}, domain: "Monitoring & Jaringan", group: "Pemeriksaan Port"},
	{keys: []string{"setting", "config"}, domain: "Pengaturan & Konfigurasi", group: "Pengaturan Sistem & LLM"},
	{keys: []string{"scan"}, domain: "Registrasi & Scanner Proyek", group: "Pindai Direktori Lokal"},
	{keys: []string{"mcp"}, domain: "Integrasi AI & MCP Hub", group: "MCP Agent Hub"},
	{keys: []string{"auth", "login"}, domain: "Autentikasi & Akun", group: "Login & Otorisasi"},
	{exact: "/", domain: "Dashboard & Pemantauan", group: "Ringkasan Workspace"},
}

func inferPageDomain(p string) (string, string) {
	for _, r := range routePageRules {
		if (r.exact != "" && p == r.exact) || util.ContainsAny(p, r.keys...) {
			return r.domain, r.group
		}
	}
	return "Fitur Antarmuka", "Navigasi Halaman"
}

var routeAPIMethodGroups = map[string]string{
	"GET":    "Query & Pengambilan Data",
	"POST":   "Pembuatan Data Baru",
	"PUT":    "Pembaruan Data",
	"PATCH":  "Pembaruan Data",
	"DELETE": "Penghapusan Entitas",
}

// InferRouteDomainAndFunction uses intelligent semantic heuristics to assign domain and function
func InferRouteDomainAndFunction(path, method string) (string, string) {
	p := strings.ToLower(path)
	m := strings.ToUpper(method)

	for _, r := range routeDomainRules {
		if util.ContainsAny(p, r.keys...) {
			return r.domain, r.group(p, m)
		}
	}
	if strings.HasPrefix(m, "PAGE") {
		return inferPageDomain(p)
	}
	group := "Lainnya"
	if g, ok := routeAPIMethodGroups[m]; ok {
		group = g
	}
	return "Layanan & Endpoint API", group
}

type routeDescRule struct {
	keys []string
	desc string
}

var routeDescRules = []routeDescRule{
	{[]string{"auth/login", "login"}, "Memvalidasi kredensial pengguna dan mengembalikan token otentikasi JWT untuk otorisasi sesi."},
	{[]string{"auth/register", "register"}, "Mendaftarkan akun pengguna baru ke dalam database sistem."},
	{[]string{"auth/logout", "logout"}, "Mengakhiri sesi aktif dan mencabut masa berlaku token otorisasi pengguna."},
	{[]string{"health", "ping"}, "Memeriksa status ketersediaan dan kesiapan layanan server (health check)."},
	{[]string{"ports"}, "Memeriksa status listening port sistem dan mendeteksi potensi tabrakan port."},
	{[]string{"fast-scan"}, "Menjalankan pemindaian mendalam AI terhadap blueprint arsitektur, rute, dan environment."},
	{[]string{"diagnose"}, "Melakukan audit statis integritas kode, git status, dan kesehatan dependensi proyek."},
	{[]string{"ai-review"}, "Menghasilkan rekomendasi arsitektur, rencana aksi P0/P1, dan perbaikan dari AI."},
	{[]string{"routes/sync"}, "Menyelaraskan daftar rute di database dengan kode sumber aktual pada repositori."},
	{[]string{"routes/enrich"}, "Mengirim daftar rute ke LLM untuk pengelompokan domain bisnis dan pembuatan keterangan otomatis."},
	{[]string{"censored-keywords"}, "Mengelola daftar kata kunci rahasia yang disensor secara global sebelum dikirim ke AI."},
	{[]string{"settings/llm"}, "Mengambil dan memperbarui konfigurasi provider LLM aktif di sistem."},
}

// routeMethodDescFormats: templat deskripsi per method untuk rute yang tak dikenali (satu %s = path bersih).
var routeMethodDescFormats = map[string]string{
	"POST":   "Membuat entitas baru pada endpoint '%s'.",
	"PUT":    "Memperbarui data entitas '%s' secara menyeluruh berdasarkan ID.",
	"PATCH":  "Memperbarui sebagian atribut data pada entitas '%s'.",
	"DELETE": "Menghapus entitas data '%s' dari database berdasarkan ID.",
}

// InferRouteDescription generates human-readable Indonesian description for routes
func InferRouteDescription(method, path string) string {
	m := strings.ToUpper(method)
	p := strings.ToLower(path)

	cleanPath := strings.TrimPrefix(path, "/api")
	cleanPath = strings.TrimPrefix(cleanPath, "/")

	if strings.HasPrefix(m, "PAGE") {
		return adapters.InferFrontendPageDescription(path)
	}
	for _, r := range routeDescRules {
		if util.ContainsAny(p, r.keys...) {
			return r.desc
		}
	}
	if m == "GET" {
		if util.ContainsAny(p, ":id", "[id]") {
			return fmt.Sprintf("Mengambil rincian data tunggal untuk entitas '%s' berdasarkan ID.", cleanPath)
		}
		return fmt.Sprintf("Mengambil daftar koleksi data untuk layanan '%s'.", cleanPath)
	}
	if f, ok := routeMethodDescFormats[m]; ok {
		return fmt.Sprintf(f, cleanPath)
	}
	return fmt.Sprintf("Menangani request HTTP %s pada endpoint '%s'.", method, path)
}

// syncFastScanDataCache updates the FastScanData JSON on Component with current DB routes
func syncFastScanDataCache(db *gorm.DB, comp *models.Component, routes []models.ComponentRoute) {
	if comp.FastScanData == "" {
		return
	}

	var fastScan models.FastScanResult
	if err := json.Unmarshal([]byte(comp.FastScanData), &fastScan); err == nil {
		converted := make([]models.RouteInfo, len(routes))
		for i, r := range routes {
			var reqs []string
			if r.TargetRequests != "" {
				_ = json.Unmarshal([]byte(r.TargetRequests), &reqs)
			}
			converted[i] = models.RouteInfo{
				Method:         r.Method,
				Path:           r.Path,
				Handler:        r.Handler,
				Description:    r.Description,
				Middleware:     r.Middleware,
				TargetRequests: reqs,
			}
		}
		fastScan.Routes = converted
		dataBytes, _ := json.Marshal(fastScan)
		comp.FastScanData = string(dataBytes)
		_ = db.Model(comp).Update("fast_scan_data", comp.FastScanData).Error
	}
}
