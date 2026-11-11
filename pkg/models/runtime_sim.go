package models

import "time"

// Status sesi Simulasi Runtime.
const (
	RuntimeSimRunning   = "running"
	RuntimeSimCompleted = "completed"
	RuntimeSimFailed    = "failed"
	RuntimeSimCancelled = "cancelled"
)

// RuntimeSimOptions mengatur satu sesi Simulasi Runtime. Batas & nilai teknis (jumlah endpoint, halaman,
// nilai contoh parameter, batas tunggu boot) ditentukan otomatis; field-nya hanya override opsional (0/"" = otomatis).
type RuntimeSimOptions struct {
	KeepRunning bool `json:"keep_running"` // biarkan aplikasi tetap menyala setelah simulasi selesai
	// SkipComponentIDs: komponen yang tidak dicentang di checklist. Kosong = semua komponen yang bisa dijalankan.
	// Backend yang dibutuhkan frontend tercentang tetap ikut dinyalakan.
	SkipComponentIDs []uint `json:"skip_component_ids,omitempty"`

	MaxRoutes      int `json:"max_routes,omitempty"`       // override batas rute per komponen (otomatis: semua, maks. 300)
	BootTimeoutSec int `json:"boot_timeout_sec,omitempty"` // override batas tunggu tetap (otomatis: menyesuaikan aktivitas log)

	// ParamValue memaksa satu nilai untuk semua parameter rute (:id, {id}, [id]). Kosong = otomatis: nilai
	// diambil dari data nyata respons endpoint daftar (mis. id pertama /api/users), lalu tebakan per nama parameter.
	ParamValue string `json:"param_value,omitempty"`
	// TestMutations ikut mengirim POST/PUT/PATCH berbody `{}` (1x, id dinamis diisi id yang tidak ada) untuk
	// menguji validasi & penanganan error. Bisa mengubah data: hanya untuk database development.
	TestMutations bool `json:"test_mutations,omitempty"`
	TestDelete    bool `json:"test_delete,omitempty"` // DELETE ikut diuji (butuh TestMutations)
	// SkipPatterns: endpoint mutasi yang path-nya mengandung salah satu kata ini tidak dikirim.
	SkipPatterns []string `json:"skip_patterns,omitempty"`

	Auth *RuntimeSimAuth `json:"auth,omitempty"` // login dulu agar endpoint & halaman terproteksi ikut diuji

	SkipPages bool     `json:"skip_pages,omitempty"` // lewati uji halaman frontend (kecepatan & screenshot)
	MaxPages  int      `json:"max_pages,omitempty"`  // override batas halaman per frontend (otomatis: semua, maks. 60)
	Viewports []string `json:"viewports,omitempty"`  // mobile, tablet, desktop (default semuanya)
}

// RuntimeSimPortConflict menjelaskan port komponen yang bentrok beserta port pengganti yang disarankan.
type RuntimeSimPortConflict struct {
	Port          int    `json:"port"`
	Kind          string `json:"kind"` // busy (dipakai proses lain), duplicate (sama dengan komponen lain), reserved (milik vibeasik)
	Message       string `json:"message"`
	SuggestedPort int    `json:"suggested_port,omitempty"`
}

// RuntimeSimPlanComponent adalah satu baris checklist Uji Runtime: bisa dijalankan atau tidak, port, dan cakupannya.
type RuntimeSimPlanComponent struct {
	ComponentID uint                    `json:"component_id"`
	Name        string                  `json:"name"`
	Role        string                  `json:"role"`
	DevCommand  string                  `json:"dev_command,omitempty"`
	Runnable    bool                    `json:"runnable"`
	Reason      string                  `json:"reason,omitempty"` // alasan tidak bisa dijalankan
	Running     bool                    `json:"running"`          // sudah berjalan (dipakai apa adanya)
	Port        int                     `json:"port,omitempty"`
	Endpoints   int                     `json:"endpoints"` // endpoint API yang akan diuji
	Pages       int                     `json:"pages"`     // halaman frontend yang akan dibuka di browser
	Dynamic     int                     `json:"dynamic"`   // rute berparameter (nilai diisi otomatis)
	NeededBy    []string                `json:"needed_by,omitempty"`
	Conflict    *RuntimeSimPortConflict `json:"conflict,omitempty"`
}

// RuntimeSimPlan adalah pratinjau Uji Runtime: checklist komponen dan pengaturan otomatis yang akan dipakai.
type RuntimeSimPlan struct {
	ProjectID  uint                      `json:"project_id"`
	Components []RuntimeSimPlanComponent `json:"components"`
	Conflicts  int                       `json:"conflicts"`
	Auto       []string                  `json:"auto"` // penjelasan singkat pengaturan otomatis
}

// RuntimeSimAuth adalah akun uji untuk login sebelum menguji endpoint & halaman yang butuh login.
type RuntimeSimAuth struct {
	Identifier      string `json:"identifier"`                 // email / username akun uji
	Password        string `json:"password,omitempty"`         // tidak pernah ikut tersimpan di laporan
	IdentifierField string `json:"identifier_field,omitempty"` // nama field JSON login API (default dicoba: email, username, identifier)
	PasswordField   string `json:"password_field,omitempty"`   // default "password"
	LoginPath       string `json:"login_path,omitempty"`       // endpoint login API (default dideteksi dari rute POST */login)
	LoginBody       string `json:"login_body,omitempty"`       // template JSON kustom, placeholder {{identifier}} & {{password}}
	LoginPagePath   string `json:"login_page_path,omitempty"`  // halaman login frontend (default dideteksi, mis. /login)
	HasPassword     bool   `json:"has_password,omitempty"`     // hanya untuk tampilan: password tersimpan di profil
}

// RuntimeTestProfile menyimpan konfigurasi Uji Runtime terakhir per proyek (termasuk akun uji) agar
// tidak perlu diketik ulang. Password tidak pernah dikembalikan lewat API.
type RuntimeTestProfile struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ProjectID   uint      `gorm:"uniqueIndex;not null" json:"project_id"`
	OptionsJSON string    `gorm:"type:longtext" json:"options_json"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RuntimeSimLogin adalah hasil satu percobaan login (API atau browser frontend).
type RuntimeSimLogin struct {
	Kind        string `json:"kind"` // api, browser
	ComponentID uint   `json:"component_id"`
	Component   string `json:"component"`
	Target      string `json:"target"` // endpoint atau URL halaman login
	OK          bool   `json:"ok"`
	StatusCode  int    `json:"status_code,omitempty"`
	Method      string `json:"method,omitempty"` // bearer, cookie, bearer+cookie, session
	Message     string `json:"message"`
	DurationMs  int64  `json:"duration_ms,omitempty"`
	Screenshot  string `json:"screenshot,omitempty"` // bukti halaman login bila gagal
}

// RuntimeSimShot adalah screenshot satu halaman pada satu viewport.
type RuntimeSimShot struct {
	Viewport    string   `json:"viewport"` // mobile, tablet, desktop
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	File        string   `json:"file"` // nama berkas, diunduh lewat .../runtime-sim/shots/:session/:file
	LoadMs      float64  `json:"load_ms"`
	ScrollWidth int      `json:"scroll_width"`
	PageHeight  int      `json:"page_height"`
	OverflowX   bool     `json:"overflow_x"`                  // konten lebih lebar dari layar → scroll horizontal (tidak responsif)
	Offenders   []string `json:"offenders,omitempty"`         // elemen yang melewati lebar layar
	SmallTap    int      `json:"small_tap_targets,omitempty"` // tombol/tautan < 32px di mobile
	SmallText   int      `json:"small_text,omitempty"`        // teks < 12px di mobile
}

// RuntimeSimPage adalah hasil membuka satu halaman frontend di browser sungguhan.
type RuntimeSimPage struct {
	ComponentID   uint             `json:"component_id"`
	Component     string           `json:"component"`
	Path          string           `json:"path"`
	SourceFile    string           `json:"source_file,omitempty"` // berkas halaman di kode (untuk diperbaiki AI)
	URL           string           `json:"url"`
	Auth          string           `json:"auth"` // guest, auth
	FinalURL      string           `json:"final_url,omitempty"`
	Title         string           `json:"title,omitempty"`
	StatusCode    int              `json:"status_code,omitempty"`
	TTFBMs        float64          `json:"ttfb_ms"`
	FCPMs         float64          `json:"fcp_ms"`
	DOMReadyMs    float64          `json:"dom_ready_ms"`
	LoadMs        float64          `json:"load_ms"`      // load terbaik setelah warm (navigasi ulang per viewport)
	ColdLoadMs    float64          `json:"cold_load_ms"` // kunjungan pertama, termasuk kompilasi on-demand dev server
	Requests      int              `json:"requests"`
	TransferKB    float64          `json:"transfer_kb"`
	FailedRequest []string         `json:"failed_requests,omitempty"`
	ConsoleErrors []string         `json:"console_errors,omitempty"`
	Shots         []RuntimeSimShot `json:"shots"`
	Verdict       string           `json:"verdict"` // OK, SLOW, REDIRECT_LOGIN, ERROR, FAILED
	Note          string           `json:"note,omitempty"`
	Error         string           `json:"error,omitempty"`
}

// RuntimeSimStep adalah satu tahap pada timeline simulasi.
type RuntimeSimStep struct {
	Key        string     `json:"key"`
	Title      string     `json:"title"`
	Status     string     `json:"status"` // pending, running, done, failed, skipped
	Detail     string     `json:"detail,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	DurationMs int64      `json:"duration_ms,omitempty"`
}

// RuntimeSimComponent adalah kondisi satu komponen selama simulasi.
type RuntimeSimComponent struct {
	ComponentID uint     `json:"component_id"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	State       string   `json:"state"`            // pending, booting, running, crashed, timeout, failed, skipped, stopped
	Source      string   `json:"source,omitempty"` // launched (dinyalakan simulasi), reused (sudah jalan via vibeasik), external (port sudah listening)
	Command     string   `json:"command,omitempty"`
	Port        int      `json:"port,omitempty"`
	URL         string   `json:"url,omitempty"`
	PID         int      `json:"pid,omitempty"`
	BootMs      int64    `json:"boot_ms,omitempty"`
	MemoryMB    float64  `json:"memory_mb,omitempty"`
	RoutesFound int      `json:"routes_found"`
	ErrorLines  int      `json:"error_lines"`
	Note        string   `json:"note,omitempty"`
	Logs        []string `json:"logs,omitempty"`
	Screenshot  string   `json:"screenshot,omitempty"` // berkas screenshot beranda (desktop), lihat RuntimeSimShot.File
}

// RuntimeSimProbe adalah hasil satu request HTTP nyata ke aplikasi yang sedang berjalan.
type RuntimeSimProbe struct {
	ComponentID uint      `json:"component_id"`
	Component   string    `json:"component"`
	Method      string    `json:"method"`
	Path        string    `json:"path"`            // path yang dikirim (parameter sudah diisi)
	Route       string    `json:"route,omitempty"` // pola rute di kode, mis. /api/users/:id
	Handler     string    `json:"handler,omitempty"`
	Auth        string    `json:"auth"` // guest, auth
	URL         string    `json:"url"`
	StatusCode  int       `json:"status_code,omitempty"`
	ColdMs      float64   `json:"cold_ms"` // request pertama (termasuk kompilasi on-demand dev server)
	WarmMs      float64   `json:"warm_ms"` // rata-rata request berikutnya
	Samples     int       `json:"samples"`
	SizeBytes   int64     `json:"size_bytes,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	Verdict     string    `json:"verdict"` // OK, SLOW, REDIRECT, PROTECTED, NOT_FOUND, CLIENT_ERROR, VALIDATION, STREAM, SERVER_ERROR, FAILED
	Error       string    `json:"error,omitempty"`
	At          time.Time `json:"at"`
}

// RuntimeSimEvent adalah satu baris aktivitas ("apa yang sedang dicek").
type RuntimeSimEvent struct {
	At      time.Time `json:"at"`
	Level   string    `json:"level"` // info, ok, warn, error
	Message string    `json:"message"`
}

// RuntimeSimSession adalah snapshot lengkap sebuah Simulasi Runtime (live maupun tersimpan).
type RuntimeSimSession struct {
	ID          string                `json:"id"`
	ProjectID   uint                  `json:"project_id"`
	ProjectName string                `json:"project_name"`
	Status      string                `json:"status"`
	Options     RuntimeSimOptions     `json:"options"`
	StartedAt   time.Time             `json:"started_at"`
	FinishedAt  *time.Time            `json:"finished_at,omitempty"`
	DurationMs  int64                 `json:"duration_ms,omitempty"`
	Score       int                   `json:"score"`
	Summary     string                `json:"summary,omitempty"`
	Steps       []RuntimeSimStep      `json:"steps"`
	Components  []RuntimeSimComponent `json:"components"`
	Probes      []RuntimeSimProbe     `json:"probes"`
	Logins      []RuntimeSimLogin     `json:"logins"`
	Pages       []RuntimeSimPage      `json:"pages"`
	Events      []RuntimeSimEvent     `json:"events"`
	Findings    []Finding             `json:"findings"`
	ReportID    uint                  `json:"report_id,omitempty"`
}

// RuntimeSimReport menyimpan hasil akhir Simulasi Runtime.
type RuntimeSimReport struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ProjectID  uint      `gorm:"index;not null" json:"project_id"`
	Status     string    `gorm:"size:32" json:"status"`
	Score      int       `json:"score"`
	Summary    string    `gorm:"type:longtext" json:"summary"`
	ResultJSON string    `gorm:"type:longtext" json:"result_json"` // RuntimeSimSession lengkap
	CreatedAt  time.Time `json:"created_at"`
}
