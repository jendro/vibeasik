package diagnostics

import (
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Area perbaikan sebuah temuan, dari sudut pandang pemilik aplikasi: apa yang membaik bila temuan ini dikerjakan.
const (
	AreaSpeed    = "speed"    // performa & kecepatan: query/N+1, Big-O, latensi rute, ukuran aset, daftar panjang
	AreaData     = "data"     // kualitas data: validasi input, transaksi
	AreaUI       = "ui"       // tampilan: frontend, mobile, halaman yang rusak saat dijalankan
	AreaSecurity = "security" // keamanan
	AreaCode     = "code"     // kode & arsitektur: kompleksitas, duplikasi, struktur
	AreaOps      = "ops"      // operasional: git, env, port, skrip jalan/deploy, risiko proyek
)

// Areas: urutan tampil area perbaikan.
var Areas = []string{AreaSpeed, AreaData, AreaUI, AreaSecurity, AreaCode, AreaOps}

// rule (awalan rule_id) → area; dicek sebelum kategori karena satu kategori bisa berisi beberapa area
// (mis. kategori query memuat N+1 = performa dan SQL injection = keamanan). Awalan terpanjang menang.
var areaByRule = map[string]string{
	"COST-": AreaSpeed, "QUERY-NPLUS1": AreaSpeed, "DEEP-QUERY-UNPAGINATED": AreaSpeed, "PERF-": AreaSpeed,
	"DEEP-ROUTE-LATENCY": AreaSpeed, "ASSET-OVERSIZED": AreaSpeed, "DEEP-ASSET-WEIGHT": AreaSpeed,
	"DEEP-FE-CSS-BLOAT": AreaSpeed, "DEEP-FE-LARGE-CHUNK": AreaSpeed, "FE-IMG-NO-LAZY": AreaSpeed,
	"FE-HEAVY-LIB-IMPORT": AreaSpeed, "FE-LISTENER-LEAK": AreaSpeed, "DEEP-AND-APK-SIZE": AreaSpeed,
	"DEEP-AND-RELEASE-NOT-OPTIMIZED": AreaSpeed, "MOB-LISTVIEW-NON-BUILDER": AreaSpeed,
	"MOB-SHRINKWRAP-LIST": AreaSpeed, "MOB-OPACITY-WIDGET": AreaSpeed, "SMELL-MISSING-TIMEOUT": AreaSpeed,
	"DEEP-UI-COMPONENT-HOTSPOT": AreaSpeed, "RUNTIME-ROUTE": AreaSpeed, "SEC-RATE-LIMIT-MISSING": AreaSpeed,

	"DEEP-FORM-MISSING-VALIDATION": AreaData, "ROUTER-DUPLICATE-VALIDATION": AreaData,
	"QUERY-MISSING-TX": AreaData,

	"SEC-": AreaSecurity, "QUERY-SQLI": AreaSecurity, "ROUTER-UNGUARDED-MUTATION": AreaSecurity,
	"ROUTER-WILDCARD-CORS": AreaSecurity, "ENV-FILE-TRACKED": AreaSecurity, "ENV-IGNORE-INCOMPLETE": AreaSecurity,
	"SMELL-HARDCODED-SECRET": AreaSecurity, "DEEP-AND-CLEARTEXT": AreaSecurity,

	"FE-": AreaUI, "DESIGN-": AreaUI, "RUNTIME-PAGE-": AreaUI, "MOB-": AreaUI,

	"SMELL-": AreaCode, "DEEP-FUNC-": AreaCode, "ROUTER-": AreaCode,

	"GIT-": AreaOps, "PORT-": AreaOps, "ENV-": AreaOps, "DEPLOY-": AreaOps, "DEV-SCRIPT": AreaOps,
	"MOBILE-DEV-SCRIPT": AreaOps, "STORAGE-": AreaOps, "PROJ-": AreaOps, "RUNTIME-": AreaOps,
}

var areaByCategory = map[string]string{
	"query": AreaSpeed, "security": AreaSecurity, "frontend": AreaUI, "mobile": AreaUI,
	"code_smell": AreaCode, "architecture": AreaCode, "quality": AreaCode, "router": AreaCode,
	"git": AreaOps, "port": AreaOps, "config": AreaOps, "deployment": AreaOps, "documentation": AreaOps,
	"runtime": AreaOps, "risk": AreaOps,
}

// TagFindingAreas mengisi Finding.Area setiap temuan (untuk respons ke Studio).
func TagFindingAreas(findings []models.Finding) {
	for i := range findings {
		findings[i].Area = FindingArea(findings[i])
	}
}

// FindingArea: area perbaikan temuan (bawaan AreaCode bila rule & kategorinya tidak dikenal).
func FindingArea(f models.Finding) string {
	rule := strings.ToUpper(f.RuleID)
	best, area := 0, ""
	for prefix, a := range areaByRule {
		if len(prefix) > best && strings.HasPrefix(rule, prefix) {
			best, area = len(prefix), a
		}
	}
	if area != "" {
		return area
	}
	if a, ok := areaByCategory[strings.ToLower(f.Category)]; ok {
		return a
	}
	return AreaCode
}
