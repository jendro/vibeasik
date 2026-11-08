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

	// Auto-detect language and framework if not set or set to auto
	if comp.Language == "auto" || comp.Language == "" || comp.Framework == "" {
		if util.PathExists(filepath.Join(absDir, "spark")) || util.PathExists(filepath.Join(absDir, "app/Config/Routes.php")) {
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
	}

	// 1. Extract routes from source code via static code inspection
	scanned := extractRoutes(absDir, comp.Role)

	// 2. Load existing routes from DB
	var existing []models.ComponentRoute
	if err := db.Where("component_id = ?", comp.ID).Find(&existing).Error; err != nil {
		return nil, fmt.Errorf("failed to load existing routes: %w", err)
	}

	// If DB is completely empty but FastScanData already contains routes, merge them first
	if len(existing) == 0 && comp.FastScanData != "" {
		var fastScan models.FastScanResult
		if err := json.Unmarshal([]byte(comp.FastScanData), &fastScan); err == nil && len(fastScan.Routes) > 0 {
			for _, r := range fastScan.Routes {
				dom, fnGrp := InferRouteDomainAndFunction(r.Path, r.Method)
				desc := r.Description
				if desc == "" {
					desc = InferRouteDescription(r.Method, r.Path)
				}
				targetReqsJSON := "[]"
				if len(r.TargetRequests) > 0 {
					b, _ := json.Marshal(r.TargetRequests)
					targetReqsJSON = string(b)
				}
				cr := models.ComponentRoute{
					ComponentID:    comp.ID,
					Method:         r.Method,
					Path:           r.Path,
					Handler:        r.Handler,
					Middleware:     r.Middleware,
					Description:    desc,
					Domain:         dom,
					FunctionGroup:  fnGrp,
					TargetRequests: targetReqsJSON,
				}
				_ = db.Create(&cr).Error
			}
			_ = db.Where("component_id = ?", comp.ID).Find(&existing)
		}
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

		targetReqsJSON := "[]"
		if len(sc.TargetRequests) > 0 {
			b, _ := json.Marshal(sc.TargetRequests)
			targetReqsJSON = string(b)
		}

		if cur, found := existingMap[key]; found {
			// In source code and in DB: keep, do not duplicate
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
			if needsSave {
				_ = db.Save(cur).Error
				updatedCount++
			}
		} else {
			// In source code but NOT in DB: add new
			domain, fnGroup := InferRouteDomainAndFunction(sc.Path, sc.Method)
			desc := sc.Description
			if desc == "" {
				desc = InferRouteDescription(sc.Method, sc.Path)
			}

			newRoute := models.ComponentRoute{
				ComponentID:    comp.ID,
				Method:         sc.Method,
				Path:           sc.Path,
				Handler:        sc.Handler,
				Middleware:     sc.Middleware,
				Description:    desc,
				Domain:         domain,
				FunctionGroup:  fnGroup,
				TargetRequests: targetReqsJSON,
			}
			if err := db.Create(&newRoute).Error; err == nil {
				addedCount++
			}
		}
	}

	// 4. "jika hilang dihapus": delete routes present in DB that no longer exist in code
	// Only delete if scanned found at least one route from code (avoid accidental wiping if dir was temporarily unreadable)
	if len(scanned) > 0 {
		for key, cur := range existingMap {
			if !codeKeySet[key] {
				db.Delete(cur)
				deletedCount++
			}
		}
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

// InferRouteDomainAndFunction uses intelligent semantic heuristics to assign domain and function
func InferRouteDomainAndFunction(path, method string) (string, string) {
	p := strings.ToLower(path)
	m := strings.ToUpper(method)

	// Determine Domain
	domain := "Umum"
	functionGroup := "Lainnya"

	switch {
	case strings.Contains(p, "auth") || strings.Contains(p, "login") || strings.Contains(p, "register") || strings.Contains(p, "token") || strings.Contains(p, "logout") || strings.Contains(p, "password") || strings.Contains(p, "oauth"):
		domain = "Autentikasi & Keamanan"
		if strings.Contains(p, "login") || strings.Contains(p, "token") {
			functionGroup = "Login & Otorisasi Sesi"
		} else if strings.Contains(p, "register") {
			functionGroup = "Registrasi Pengguna"
		} else if strings.Contains(p, "logout") {
			functionGroup = "Terminasi Sesi"
		} else {
			functionGroup = "Manajemen Kredensial"
		}

	case strings.Contains(p, "user") || strings.Contains(p, "member") || strings.Contains(p, "profile") || strings.Contains(p, "role") || strings.Contains(p, "permission") || strings.Contains(p, "account"):
		domain = "Manajemen Pengguna & Akun"
		if m == "GET" {
			functionGroup = "Pengambilan Data Pengguna"
		} else if m == "POST" || m == "PUT" || m == "PATCH" {
			functionGroup = "Pembaruan Profil & Akses"
		} else {
			functionGroup = "Operasi Akun"
		}

	case strings.Contains(p, "project") || strings.Contains(p, "bundle") || strings.Contains(p, "component") || strings.Contains(p, "app"):
		domain = "Manajemen Proyek & Komponen"
		if strings.Contains(p, "component") {
			functionGroup = "Manajemen Service & Komponen"
		} else if m == "GET" {
			functionGroup = "Katalog & Detail Proyek"
		} else {
			functionGroup = "Registrasi & Modifikasi Proyek"
		}

	case strings.Contains(p, "asset") || strings.Contains(p, "product") || strings.Contains(p, "item") || strings.Contains(p, "inventory") || strings.Contains(p, "category"):
		domain = "Aset & Inventori"
		if m == "GET" {
			functionGroup = "Katalog & Pencarian Aset"
		} else {
			functionGroup = "Mutasi & Kelola Aset"
		}

	case strings.Contains(p, "diag") || strings.Contains(p, "health") || strings.Contains(p, "ping") || strings.Contains(p, "metric") || strings.Contains(p, "log") || strings.Contains(p, "port") || strings.Contains(p, "monitor"):
		domain = "Diagnostik & Monitoring"
		if strings.Contains(p, "port") {
			functionGroup = "Pemeriksaan Status Port"
		} else if strings.Contains(p, "health") || strings.Contains(p, "ping") {
			functionGroup = "Health Check & Liveness"
		} else {
			functionGroup = "Audit Kesehatan Kode"
		}

	case strings.Contains(p, "llm") || strings.Contains(p, "ai") || strings.Contains(p, "scan") || strings.Contains(p, "review") || strings.Contains(p, "advisor"):
		domain = "Layanan AI & Fast Scan"
		if strings.Contains(p, "fast-scan") || strings.Contains(p, "scan") {
			functionGroup = "Pemindaian Blueprint & Arsitektur"
		} else if strings.Contains(p, "config") || strings.Contains(p, "model") {
			functionGroup = "Konfigurasi Multi-Model AI"
		} else {
			functionGroup = "Analisis Cerdas & Audit AI"
		}

	case strings.Contains(p, "security") || strings.Contains(p, "censor") || strings.Contains(p, "secret") || strings.Contains(p, "vault"):
		domain = "Keamanan & Redaksi Rahasia"
		functionGroup = "Vault Kata Kunci Terenkripsi"

	case strings.Contains(p, "setting") || strings.Contains(p, "config") || strings.Contains(p, "preference"):
		domain = "Pengaturan & Konfigurasi"
		functionGroup = "Preferensi Sistem"

	case strings.Contains(p, "file") || strings.Contains(p, "upload") || strings.Contains(p, "download") || strings.Contains(p, "media") || strings.Contains(p, "document") || strings.Contains(p, "dir"):
		domain = "Media & File Storage"
		functionGroup = "Manajemen Berkas & Direktori"

	case strings.Contains(p, "kontingen") || strings.Contains(p, "official"):
		domain = "Portal & Manajemen Kontingen"
		if strings.Contains(p, "official") {
			functionGroup = "Manajemen Official Kontingen"
		} else if strings.Contains(p, "daftar") || strings.Contains(p, "ikuti") {
			functionGroup = "Pendaftaran Kejuaraan"
		} else if strings.Contains(p, "tagihan") || strings.Contains(p, "rekap") {
			functionGroup = "Rekapitulasi & Tagihan Kontingen"
		} else {
			functionGroup = "Profil & Manajemen Kontingen"
		}

	case strings.Contains(p, "atlet"):
		domain = "Portal Atlet & Peserta"
		if strings.Contains(p, "pertandingan") {
			functionGroup = "Jadwal & Hasil Atlet"
		} else {
			functionGroup = "Biodata & Status Atlet"
		}

	case strings.Contains(p, "kejuaraan") || strings.Contains(p, "pertandingan") || strings.Contains(p, "bagan") || strings.Contains(p, "bracket") || strings.Contains(p, "kelas") || strings.Contains(p, "kategori"):
		domain = "Kompetisi & Pertandingan"
		if strings.Contains(p, "bagan") || strings.Contains(p, "bracket") {
			functionGroup = "Bagan Pertandingan & Bracket Tree"
		} else if strings.Contains(p, "kelas") || strings.Contains(p, "kategori") {
			functionGroup = "Kategori & Kelas Tanding"
		} else {
			functionGroup = "Manajemen Event & Jadwal Tanding"
		}

	case strings.Contains(p, "bayar") || strings.Contains(p, "pembayaran") || strings.Contains(p, "tagihan") || strings.Contains(p, "rekening") || strings.Contains(p, "invoice") || strings.Contains(p, "billing"):
		domain = "Keuangan & Pembayaran"
		functionGroup = "Transaksi & Verifikasi Pembayaran"

	case strings.Contains(p, "public") || strings.Contains(p, "portal") || strings.Contains(p, "gallery") || strings.Contains(p, "galeri") || strings.Contains(p, "blog") || strings.Contains(p, "news") || strings.Contains(p, "lokasi"):
		domain = "Portal Informasi Publik"
		if strings.Contains(p, "blog") || strings.Contains(p, "news") {
			functionGroup = "Berita & Artikel"
		} else if strings.Contains(p, "gallery") || strings.Contains(p, "galeri") {
			functionGroup = "Dokumentasi & Galeri"
		} else if strings.Contains(p, "lokasi") {
			functionGroup = "Informasi Lokasi & Venue"
		} else {
			functionGroup = "Informasi & Publikasi Umum"
		}

	case strings.Contains(p, "admin"):
		domain = "Administrasi & Master Data"
		if strings.Contains(p, "verify") || strings.Contains(p, "verifikasi") {
			functionGroup = "Verifikasi & Validasi Data"
		} else if strings.Contains(p, "dashboard") {
			functionGroup = "Dashboard Eksekutif Admin"
		} else {
			functionGroup = "Master Data & Pengawasan"
		}

	case strings.HasPrefix(m, "PAGE"):
		switch {
		case strings.Contains(p, "project") || strings.Contains(p, "bundle") || strings.Contains(p, "app"):
			domain = "Manajemen Proyek"
			functionGroup = "Katalog & Detail Proyek"
		case strings.Contains(p, "port") || strings.Contains(p, "monitor"):
			domain = "Monitoring & Jaringan"
			functionGroup = "Pemeriksaan Port"
		case strings.Contains(p, "setting") || strings.Contains(p, "config"):
			domain = "Pengaturan & Konfigurasi"
			functionGroup = "Pengaturan Sistem & LLM"
		case strings.Contains(p, "scan"):
			domain = "Registrasi & Scanner Proyek"
			functionGroup = "Pindai Direktori Lokal"
		case strings.Contains(p, "mcp"):
			domain = "Integrasi AI & MCP Hub"
			functionGroup = "MCP Agent Hub"
		case strings.Contains(p, "auth") || strings.Contains(p, "login"):
			domain = "Autentikasi & Akun"
			functionGroup = "Login & Otorisasi"
		case p == "/":
			domain = "Dashboard & Pemantauan"
			functionGroup = "Ringkasan Workspace"
		default:
			domain = "Fitur Antarmuka"
			functionGroup = "Navigasi Halaman"
		}

	default:
		domain = "Layanan & Endpoint API"
		if m == "GET" {
			functionGroup = "Query & Pengambilan Data"
		} else if m == "POST" {
			functionGroup = "Pembuatan Data Baru"
		} else if m == "PUT" || m == "PATCH" {
			functionGroup = "Pembaruan Data"
		} else if m == "DELETE" {
			functionGroup = "Penghapusan Entitas"
		}
	}

	return domain, functionGroup
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

	switch {
	case strings.Contains(p, "auth/login") || strings.Contains(p, "login"):
		return "Memvalidasi kredensial pengguna dan mengembalikan token otentikasi JWT untuk otorisasi sesi."
	case strings.Contains(p, "auth/register") || strings.Contains(p, "register"):
		return "Mendaftarkan akun pengguna baru ke dalam database sistem."
	case strings.Contains(p, "auth/logout") || strings.Contains(p, "logout"):
		return "Mengakhiri sesi aktif dan mencabut masa berlaku token otorisasi pengguna."
	case strings.Contains(p, "health") || strings.Contains(p, "ping"):
		return "Memeriksa status ketersediaan dan kesiapan layanan server (health check)."
	case strings.Contains(p, "ports"):
		return "Memeriksa status listening port sistem dan mendeteksi potensi tabrakan port."
	case strings.Contains(p, "fast-scan"):
		return "Menjalankan pemindaian mendalam AI terhadap blueprint arsitektur, rute, dan environment."
	case strings.Contains(p, "diagnose"):
		return "Melakukan audit statis integritas kode, git status, dan kesehatan dependensi proyek."
	case strings.Contains(p, "ai-review"):
		return "Menghasilkan rekomendasi arsitektur, rencana aksi P0/P1, dan perbaikan dari AI."
	case strings.Contains(p, "routes/sync"):
		return "Menyelaraskan daftar rute di database dengan kode sumber aktual pada repositori."
	case strings.Contains(p, "routes/enrich"):
		return "Mengirim daftar rute ke LLM untuk pengelompokan domain bisnis dan pembuatan keterangan otomatis."
	case strings.Contains(p, "censored-keywords"):
		return "Mengelola daftar kata kunci rahasia yang disensor secara global sebelum dikirim ke AI."
	case strings.Contains(p, "settings/llm"):
		return "Mengambil dan memperbarui konfigurasi provider LLM aktif di sistem."
	default:
		switch m {
		case "GET":
			if strings.Contains(p, ":id") || strings.Contains(p, "[id]") {
				return fmt.Sprintf("Mengambil rincian data tunggal untuk entitas '%s' berdasarkan ID.", cleanPath)
			}
			return fmt.Sprintf("Mengambil daftar koleksi data untuk layanan '%s'.", cleanPath)
		case "POST":
			return fmt.Sprintf("Membuat entitas baru pada endpoint '%s'.", cleanPath)
		case "PUT":
			return fmt.Sprintf("Memperbarui data entitas '%s' secara menyeluruh berdasarkan ID.", cleanPath)
		case "PATCH":
			return fmt.Sprintf("Memperbarui sebagian atribut data pada entitas '%s'.", cleanPath)
		case "DELETE":
			return fmt.Sprintf("Menghapus entitas data '%s' dari database berdasarkan ID.", cleanPath)
		default:
			return fmt.Sprintf("Menangani request HTTP %s pada endpoint '%s'.", method, path)
		}
	}
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
