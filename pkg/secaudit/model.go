// Package secaudit menjalankan uji keamanan proyek: memeriksa kode router/sumber apakah kontrol
// keamanan standar dipasang (rate limit, header keamanan, CORS, query berparameter, validasi input,
// autentikasi rute mutasi) lalu membuktikannya lewat respons HTTP nyata dari server proyek sendiri
// (lokal atau live). Kontrol yang hanya terlihat di kode berstatus "declared"; kontrol yang terbukti
// di respons berstatus "pass". Probe live hanya mengirim request biasa ke server milik pengguna.
package secaudit

import "time"

// Status hasil satu pemeriksaan.
const (
	StatusPass     = "pass"     // terbukti lewat respons HTTP nyata (atau kode aman tanpa pola berbahaya)
	StatusDeclared = "declared" // terlihat dipasang di kode, belum dibuktikan di server berjalan
	StatusWarn     = "warn"     // sebagian / lemah
	StatusFail     = "fail"     // tidak ada atau terbukti tidak berjalan
	StatusSkipped  = "skipped"  // tidak bisa diuji (mis. server tidak menyala, bukan backend)
)

// Kategori kontrol keamanan. Tiap kategori punya bukti sendiri (statis dan/atau live), tidak digabung.
const (
	CatRateLimit  = "rate_limit"  // DDoS & brute-force
	CatHeaders    = "headers"     // header keamanan respons
	CatCORS       = "cors"        // asal lintas domain
	CatInjection  = "injection"   // SQL / command injection
	CatInput      = "input"       // validasi input & batas body
	CatAuth       = "auth"        // autentikasi rute mutasi
	CatTransport  = "transport"   // HTTPS / HSTS
	CatErrorLeaks = "error_leaks" // galat internal bocor ke klien
	CatXSS        = "xss"         // cross-site scripting
	CatTraversal  = "traversal"   // path traversal & berkas sensitif terbuka
)

// Severity mengikuti tingkat temuan diagnostik vibeasik.
const (
	SevCritical = "CRITICAL"
	SevWarning  = "WARNING"
	SevInfo     = "INFO"
)

// Check = satu kontrol keamanan beserta buktinya.
type Check struct {
	ID          string     `json:"id"`
	Category    string     `json:"category"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	Severity    string     `json:"severity"`
	Source      string     `json:"source"` // static | live
	Evidence    []string   `json:"evidence,omitempty"`
	Suggestion  string     `json:"suggestion,omitempty"`
	Locations   []Location `json:"locations,omitempty"`
	GuidelineID string     `json:"guideline_id,omitempty"`
}

// Location menunjuk baris kode yang menjadi bukti.
type Location struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet,omitempty"`
}

// LiveOptions mengatur probe ke server berjalan.
type LiveOptions struct {
	BaseURL     string        // mis. http://127.0.0.1:4101 atau https://app.contoh.com
	Paths       []string      // path GET aman untuk diuji (dari daftar rute); kosong = "/"
	AuthPaths   []string      // path login/autentikasi (diuji burst POST kecil)
	Burst       int           // jumlah request burst rate limit (default 25, maks 60)
	Timeout     time.Duration // per request (default 5 detik)
	SkipBurst   bool          // lewati uji burst (mis. server produksi yang rapuh)
	ProbeLabel  string        // label sasaran untuk laporan
	Routes      []RouteRef    // semua rute yang dikenal: sasaran uji serangan
	AuthHeader  string        // opsional: "Bearer …" / "Cookie: …" agar endpoint berlogin ikut diserang
	SkipAttacks bool          // lewati uji serangan (SQLi, XSS, traversal, …)
	// AttackBudget = batas waktu fase serangan termasuk jeda rate limit (bawaan 8 menit).
	AttackBudget time.Duration
	// OnLog (opsional) menerima catatan fase serangan (mis. jeda karena rate limit).
	OnLog func(level, msg string)
	// OnPhase (opsional) dipanggil sebelum fase "attack" (serangan) dan "burst" (rate limit).
	OnPhase func(phase string)
	// OnRequest (opsional) dipanggil setiap request probe selesai — untuk memantau uji yang sedang berjalan.
	OnRequest func(method, path string, status int, latency time.Duration, err error)
}

// LiveResult merangkum probe server berjalan.
type LiveResult struct {
	BaseURL     string            `json:"base_url"`
	Reachable   bool              `json:"reachable"`
	Error       string            `json:"error,omitempty"`
	StatusCode  int               `json:"status_code,omitempty"`
	LatencyMs   int64             `json:"latency_ms,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Paths       []string          `json:"paths,omitempty"` // rute yang menjawab dan ikut diuji
	RequestsRun int               `json:"requests_run"`
	Attacks     []Attack          `json:"attacks,omitempty"` // serangan yang dikirim beserta vonisnya
	Notes       []string          `json:"-"`
}

// Report = hasil uji keamanan satu proyek / komponen.
type Report struct {
	ProjectID   uint          `json:"project_id"`
	ComponentID uint          `json:"component_id,omitempty"`
	Target      string        `json:"target"`
	Score       int           `json:"score"`
	Grade       string        `json:"grade"`
	Summary     ReportSummary `json:"summary"`
	Checks      []Check       `json:"checks"`
	Live        *LiveResult   `json:"live,omitempty"`
	FilesRead   int           `json:"files_read"`
	RoutesSeen  int           `json:"routes_seen"`
	StartedAt   time.Time     `json:"started_at"`
	DurationMs  int64         `json:"duration_ms"`
	Notes       []string      `json:"notes,omitempty"`
}

// ReportSummary menghitung jumlah per status.
type ReportSummary struct {
	Pass     int `json:"pass"`
	Declared int `json:"declared"`
	Warn     int `json:"warn"`
	Fail     int `json:"fail"`
	Skipped  int `json:"skipped"`
}
