package models

import "time"

// Pegawai vibeasik: agent AI berkepribadian & berpengetahuan sendiri (desainer, PM, engineer, QA, HR, …).
// Agent coding (Claude Code, Antigravity, orkestrator) yang mengerjakan todo vibeasik "memakai" pegawai:
// persona + pengetahuannya disuntikkan ke prompt, lalu hasil kerjanya dinilai pegawai QA dan pengetahuan
// pegawai bertambah (yang bagus) atau berkurang (yang buruk).

// Status pegawai.
const (
	EmployeeActive   = "active"
	EmployeeResigned = "resigned" // dinonaktifkan; riwayat tetap disimpan
)

// EmployeeAvatar adalah penampilan karakter pegawai (avatar profil & karakter di Kantor Agent).
type EmployeeAvatar struct {
	Skin      string `json:"skin"`
	Hair      string `json:"hair"`
	HairStyle string `json:"hair_style"` // short, long, bun, bald, curly, hijab, ponytail, spiky
	Shirt     string `json:"shirt"`
	Pants     string `json:"pants"`
	Accessory string `json:"accessory"` // none, glasses, headset, cap, beanie, tie, scarf
	Facial    string `json:"facial"`    // none, beard, mustache
	Emoji     string `json:"emoji"`
}

// Employee adalah satu pegawai agent vibeasik.
type Employee struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Name      string `gorm:"size:128" json:"name"`
	Role      string `gorm:"size:64;index" json:"role"` // key peran (roles.go paket staff)
	Title     string `gorm:"size:128" json:"title"`     // jabatan tampilan, mis. "Senior Backend Engineer"
	Division  string `gorm:"size:64;index" json:"division"`
	Seniority string `gorm:"size:32" json:"seniority"` // junior, mid, senior, lead
	// Specialty: kekhususan teknis, mis. "Go · Fiber · GORM" (dicocokkan dengan teks todo saat penugasan)
	Specialty string `gorm:"size:255" json:"specialty"`
	// ProfileType: specialist (fokus mendalam pada 1-2 keahlian dekat) atau generalist (luas di divisinya)
	ProfileType string   `gorm:"size:32;default:'specialist'" json:"profile_type"`
	Stack       []string `gorm:"type:text;serializer:json" json:"stack"`
	Brief       string   `gorm:"type:text" json:"brief"`       // tanggung jawab & cara kerja
	Personality string   `gorm:"type:text" json:"personality"` // gambaran kepribadian
	Motto       string   `gorm:"size:255" json:"motto"`
	// Skills: bobot keahlian 0–100 per kemampuan (bertambah/berkurang dari review QA)
	Skills map[string]int `gorm:"type:text;serializer:json" json:"skills"`
	// Traits: bobot kepribadian 0–100 (ketelitian, kreativitas, kecepatan, komunikasi, keberanian)
	Traits map[string]int `gorm:"type:text;serializer:json" json:"traits"`
	// Principles: kunci prinsip pegangan kerja pegawai (dipilih saat direkrut; menentukan urutan & pilihan langkah kerjanya)
	Principles []string `gorm:"type:text;serializer:json" json:"principles"`
	// Mind: pola pikir pegawai sebagai sumbu dua kutub 0–100 (cara menangkap, menimbang, bekerja, berhubungan).
	// Taste: selera tampilan sebagai sumbu dua kutub 0–100 (kepadatan, bentuk, warna, kontras, …).
	Mind   map[string]int `gorm:"type:text;serializer:json" json:"mind"`
	Taste  map[string]int `gorm:"type:text;serializer:json" json:"taste"`
	Avatar EmployeeAvatar `gorm:"type:text;serializer:json" json:"avatar"`
	Desk   int            `json:"desk"` // urutan kursi di zona divisinya
	// LeadsDivision: divisi yang dipimpin pegawai ini (leader/manager). Leader memahami spesialisasi timnya
	// dan menimbang siapa yang mengerjakan setiap todo divisinya.
	LeadsDivision string `gorm:"size:64;index" json:"leads_division,omitempty"`
	Status        string `gorm:"size:32;index;default:'active'" json:"status"`
	// WarningLevel: SP yang sedang berlaku dari HR (0 = tidak ada, 1–3 = SP1–SP3)
	WarningLevel int `gorm:"default:0" json:"warning_level"`
	// ProjectID & ProjectName: untuk eksekutif proyek (CEO, CTO, PO) yang terikat ke proyek tertentu
	ProjectID   *uint  `gorm:"index" json:"project_id,omitempty"`
	ProjectName string `gorm:"size:255" json:"project_name,omitempty"`

	XP            int        `json:"xp"`
	Level         int        `gorm:"default:1" json:"level"`
	TasksDone     int        `json:"tasks_done"`
	TasksReviewed int        `json:"tasks_reviewed"`
	AvgScore      float64    `json:"avg_score"`
	LastReviewAt  *time.Time `json:"last_review_at,omitempty"`
	// ReplacedEmployeeID & ReplacedEmployeeName: catatan eks pegawai yang digantikan pegawai baru ini
	ReplacedEmployeeID   *uint     `gorm:"index" json:"replaced_employee_id,omitempty"`
	ReplacedEmployeeName string    `gorm:"size:128" json:"replaced_employee_name,omitempty"`
	ReplacementReason    string    `gorm:"size:512" json:"replacement_reason,omitempty"`
	InheritedNotesCount  int       `json:"inherited_notes_count,omitempty"`
	InheritedSummary     string    `gorm:"type:text" json:"inherited_summary,omitempty"`
	HiredAt              time.Time `json:"hired_at"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Asal & status satu butir pengetahuan pegawai.
const (
	KnowledgeSeed   = "seed"      // pengetahuan bawaan saat direkrut
	KnowledgeQA     = "qa_review" // ditambahkan QA dari hasil kerja yang dinilai bagus
	KnowledgeManual = "manual"    // diajarkan langsung oleh manusia
	KnowledgeAgenda = "agenda"    // dipelajari pegawai sendiri lewat agenda pribadinya
	KnowledgeChat   = "chat"      // dipetik dari obrolan santai dengan rekan pegawai
	KnowledgeTeam   = "team"      // catatan leader tentang keahlian anggota timnya (diperbarui otomatis)
	KnowledgeAlumni = "alumni"    // diwariskan dari eks pegawai yang dipecat/berhenti

	KnowledgeActive  = "active"
	KnowledgeRemoved = "removed" // dihapus QA / manusia karena dinilai buruk; disimpan sebagai jejak
	KnowledgeMerged  = "merged"  // dilebur ke pengetahuan lain yang mirip (konsolidasi memori)
)

// Profil keahlian pegawai.
const (
	ProfileSpecialist = "specialist" // fokus mendalam pada 1-2 keahlian dekat
	ProfileGeneralist = "generalist" // luas di divisinya, adaptif untuk todo umum/integrasi
)

// EmployeeKnowledge adalah satu butir pengetahuan yang dibawa pegawai ke setiap tugas dan obrolan.
type EmployeeKnowledge struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	EmployeeID uint   `gorm:"index" json:"employee_id"`
	Topic      string `gorm:"size:191" json:"topic"`
	Content    string `gorm:"type:text" json:"content"`
	Kind       string `gorm:"size:32" json:"kind"` // technical, process, lesson, preference
	Source     string `gorm:"size:32;index" json:"source"`
	// Weight: keyakinan/kepentingan 0–100; pengetahuan paling berbobot didahulukan di prompt
	Weight       int   `json:"weight"`
	SourceTodoID *uint `json:"source_todo_id,omitempty"`
	// MergedIntoID: id butir pengetahuan utama tempat butir ini dileburkan
	MergedIntoID *uint `gorm:"index" json:"merged_into_id,omitempty"`
	// AboutEmployeeID: untuk catatan tim (source team) — anggota tim yang dicatat leader
	AboutEmployeeID *uint      `gorm:"index" json:"about_employee_id,omitempty"`
	ReviewID        *uint      `json:"review_id,omitempty"`
	Status          string     `gorm:"size:32;index;default:'active'" json:"status"`
	RemovedReason   string     `gorm:"type:text" json:"removed_reason,omitempty"`
	RemovedAt       *time.Time `json:"removed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Status tugas pegawai.
const (
	EmployeeTaskAssigned = "assigned" // todo sedang/akan dikerjakan agent atas nama pegawai
	EmployeeTaskDone     = "completed"
	EmployeeTaskReviewed = "reviewed"
)

// EmployeeTask adalah satu todo yang dikerjakan atas nama pegawai, beserta nilai QA-nya.
type EmployeeTask struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	EmployeeID  uint   `gorm:"index" json:"employee_id"`
	TodoID      uint   `gorm:"index" json:"todo_id"`
	Title       string `gorm:"size:512" json:"title"`
	ProjectName string `gorm:"size:255" json:"project_name,omitempty"`
	Category    string `gorm:"size:64" json:"category"`
	AssignedBy  string `gorm:"size:128" json:"assigned_by,omitempty"` // agent coding / orkestrator / manusia
	// Area spesialisasi todo (mis. "backend_engineer/go"): pengalaman per area membuat leader makin sering
	// memberi todo sejenis ke pegawai yang sama sehingga ia makin tajam di bidang itu.
	Area            string     `gorm:"size:128;index" json:"area,omitempty"`
	LeaderID        *uint      `json:"leader_id,omitempty"`
	LeaderName      string     `gorm:"size:128" json:"leader_name,omitempty"`
	AssignReason    string     `gorm:"type:text" json:"assign_reason,omitempty"`
	Status          string     `gorm:"size:32;index" json:"status"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ReviewID        *uint      `json:"review_id,omitempty"`
	Score           *int       `json:"score,omitempty"` // 0–100 dari QA
	QAComment       string     `gorm:"type:text" json:"qa_comment,omitempty"`
	HighScoreSkills []string   `gorm:"type:text;serializer:json" json:"high_score_skills,omitempty"` // Skill penentu nilai tinggi
	HighScoreNotes  string     `gorm:"type:text" json:"high_score_notes,omitempty"`                  // Catatan penyebab nilai tinggi / praktik baik
	SuccessFactors  []string   `gorm:"type:text;serializer:json" json:"success_factors,omitempty"`   // List faktor penentu keberhasilan
	// QCFields: nilai panel QC per bidang (keamanan, performa, arsitektur, data, testing, UI/UX)
	QCFields map[string]QCFieldScore `gorm:"type:text;serializer:json" json:"qc_fields,omitempty"`
	// Rework: nasib todo setelah nilai QA rendah — "reopened" (dibuka lagi lewat leader), "limit" (batas pengerjaan ulang habis)
	Rework string `gorm:"size:16" json:"rework,omitempty"`
	// Impact: level dampak todo ke pengguna akhir & performa aplikasi (0–4, Aturan 38) — bobot rata-rata nilai pegawai.
	Impact       *int   `json:"impact,omitempty"`
	ImpactReason string `gorm:"size:512" json:"impact_reason,omitempty"`
	PrevScore    *int   `json:"prev_score,omitempty"` // nilai sebelum dinilai ulang dengan rubrik berpatokan pengguna
	// Relevance: seberapa mewakili kondisi pegawai sekarang (0,15–1; dihitung, tidak disimpan) — tugas yang dikerjakan
	// sebelum sebagian besar pengetahuannya sekarang ada berbobot kecil di avg_score (staff/relevance.go).
	Relevance float64   `gorm:"-" json:"relevance,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// KnowledgeChange adalah satu perubahan pengetahuan hasil review QA.
type KnowledgeChange struct {
	KnowledgeID uint   `json:"knowledge_id,omitempty"`
	Topic       string `json:"topic"`
	Content     string `json:"content,omitempty"`
	Reason      string `json:"reason,omitempty"`
	TodoID      uint   `json:"todo_id,omitempty"`
}

// SkillChange adalah perubahan bobot keahlian hasil review QA.
type SkillChange struct {
	Skill  string `json:"skill"`
	Delta  int    `json:"delta"`
	Reason string `json:"reason,omitempty"`
}

// Status review QA.
const (
	ReviewQueued    = "queued"
	ReviewRunning   = "running"
	ReviewCompleted = "completed"
	ReviewFailed    = "failed"
)

// EmployeeReview adalah satu sesi penilaian QA atas kumpulan tugas yang diselesaikan seorang pegawai.
type EmployeeReview struct {
	ID           uint              `gorm:"primaryKey" json:"id"`
	EmployeeID   uint              `gorm:"index" json:"employee_id"`
	EmployeeName string            `gorm:"size:128" json:"employee_name"`
	ReviewerID   uint              `json:"reviewer_id"`
	ReviewerName string            `gorm:"size:128" json:"reviewer_name"`
	Status       string            `gorm:"size:32;index" json:"status"`
	TaskIDs      []uint            `gorm:"type:text;serializer:json" json:"task_ids"`
	Score        int               `json:"score"`
	Summary      string            `gorm:"type:text" json:"summary,omitempty"`
	Added        []KnowledgeChange `gorm:"type:text;serializer:json" json:"added"`
	Removed      []KnowledgeChange `gorm:"type:text;serializer:json" json:"removed"`
	SkillChanges []SkillChange     `gorm:"type:text;serializer:json" json:"skill_changes"`
	XPGained     int               `json:"xp_gained"`
	// ReworkTodoIDs: todo bernilai rendah yang dibuka lagi lewat leader setelah review ini
	ReworkTodoIDs []uint     `gorm:"type:text;serializer:json" json:"rework_todo_ids,omitempty"`
	Relevance     float64    `gorm:"-" json:"relevance,omitempty"` // relevansi terhadap kondisi pegawai sekarang (lihat EmployeeTask.Relevance)
	Model         string     `gorm:"size:128" json:"model,omitempty"`
	Error         string     `gorm:"type:text" json:"error,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Status pesan obrolan dengan pegawai.
const (
	ChatPending  = "pending" // pesan pengguna menunggu dijawab (job antrean staff_chat)
	ChatAnswered = "answered"
	ChatFailed   = "failed"
)

// EmployeeChatMessage adalah satu pesan obrolan antara manusia dan pegawai.
type EmployeeChatMessage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	EmployeeID uint      `gorm:"index" json:"employee_id"`
	Role       string    `gorm:"size:16" json:"role"` // user, assistant
	Content    string    `gorm:"type:text" json:"content"`
	Status     string    `gorm:"size:16;index" json:"status"`
	Model      string    `gorm:"size:128" json:"model,omitempty"`
	Error      string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// StaffSettings mengatur pemakaian pegawai (satu baris).
type StaffSettings struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// Enabled: todo vibeasik yang dikerjakan agent coding ditugaskan ke pegawai (persona + pengetahuan masuk prompt)
	Enabled bool `gorm:"default:true" json:"enabled"`
	// AutoReview: QA menilai otomatis tugas yang sudah selesai begitu tumpukannya tenang
	AutoReview bool `gorm:"default:true" json:"auto_review"`
	// ReviewMinTasks: jumlah tugas selesai minimal sebelum review otomatis
	ReviewMinTasks int `gorm:"default:1" json:"review_min_tasks"`
	// ReviewSettleMin: menunggu sekian menit tanpa todo baru selesai, agar semua todo terkumpul dulu
	ReviewSettleMin int `gorm:"default:3" json:"review_settle_min"`
	// MaxKnowledge: batas pengetahuan aktif per pegawai; yang paling ringan dilepas bila penuh
	MaxKnowledge int `gorm:"default:40" json:"max_knowledge"`
	// Agenda pribadi pegawai (EmployeeAgenda): dibuat dari pemicu otomatis & dikerjakan saat senggang
	AgendaEnabled bool `gorm:"default:true" json:"agenda_enabled"`
	// AgendaAutoRun: agenda dikerjakan otomatis saat pegawai tidak punya todo aktif
	AgendaAutoRun bool `gorm:"default:true" json:"agenda_auto_run"`
	// AgendaAllowCLI: agenda investigasi boleh memakai agent CLI lokal (read-only, terminal dibuka di laptop)
	AgendaAllowCLI bool `gorm:"default:true" json:"agenda_allow_cli"`
	// AgendaDailyLimit: disimpan untuk kompatibilitas ke belakang (sebelumnya batas harian)
	AgendaDailyLimit int `gorm:"default:6" json:"agenda_daily_limit,omitempty"`
	// AgendaRate: seberapa sering agenda dikerjakan per pegawai — sedikit (60m), normal (20m), sering (8m), sangat_sering (3m)
	AgendaRate string `gorm:"size:16;default:'normal'" json:"agenda_rate"`
	// AgendaLowScore: nilai QA di bawah ambang ini memicu agenda belajar
	AgendaLowScore int `gorm:"default:70" json:"agenda_low_score"`
	// ReworkEnabled: todo yang nilai QA-nya di bawah ReworkScore langsung dibuka lagi dan diserahkan ke leader
	// divisinya, yang menulis arahan lebih tepat & memilih pegawai yang paling mungkin mendapat nilai tinggi
	ReworkEnabled bool `gorm:"default:true" json:"rework_enabled"`
	ReworkScore   int  `gorm:"default:60" json:"rework_score"`
	// ReworkMax: batas pengerjaan ulang per todo (mencegah putaran tanpa akhir)
	ReworkMax int `gorm:"default:2" json:"rework_max"`
	// QAGateEnabled: hasil kerja todo di branch-nya dinilai QA SEBELUM digabung ke branch utama; nilai di bawah
	// QAGateScore dikerjakan ulang di branch yang sama, maksimal QAGateMaxRounds putaran, lalu dieskalasi ke leader
	QAGateEnabled   bool `gorm:"default:true" json:"qa_gate_enabled"`
	QAGateScore     int  `gorm:"default:75" json:"qa_gate_score"`
	QAGateMaxRounds int  `gorm:"default:3" json:"qa_gate_max_rounds"`
	// RevisitEnabled: todo selesai bernilai kurang masuk antrean perbaikan tertunda dan baru dibuka lagi
	// setelah ada bukti skill naik (lihat EmployeeRevisit)
	RevisitEnabled bool `gorm:"default:true" json:"revisit_enabled"`
	// RevisitScore: nilai di bawah ambang ini (tapi ≥ rework_score) dianggap lulus tipis
	RevisitScore int `gorm:"default:70" json:"revisit_score"`
	// RevisitDaily: todo yang dibuka lagi dari antrean per hari (agar tidak mendesak pekerjaan baru)
	RevisitDaily int `gorm:"default:3" json:"revisit_daily"`
	// RevisitMaxAgeDays: todo yang selesai lebih lama dari ini dianggap usang
	RevisitMaxAgeDays int `gorm:"default:60" json:"revisit_max_age_days"`
	// RevisitMaxCycles: batas berapa kali satu todo bisa masuk antrean perbaikan tertunda (siklus ulang setelah nilai baru masih rendah)
	RevisitMaxCycles int `gorm:"default:3" json:"revisit_max_cycles"`
	// IdleSocial: pegawai yang santai (tanpa todo aktif) boleh mengobrol sesama pegawai & upgrade skill sendiri
	IdleSocial bool `gorm:"default:true" json:"idle_social"`
	// IdleSocialRate: seberapa sering kegiatan santai dipicu — sedikit, normal, banyak, sangat_sering
	IdleSocialRate string `gorm:"size:16;default:'normal'" json:"idle_social_rate"`
	// ConversationModel: model ringan untuk obrolan & upgrade skill santai ("<config_id>:<model>");
	// kosong = otomatis (9Router lokal deepseek flash bila ada, selain itu model teringan di pool)
	ConversationModel string `gorm:"size:255" json:"conversation_model"`
	// HREnabled: pegawai HR mengevaluasi kinerja (SP), relevansi (usulan pecat), dan beban tim (usulan rekrut)
	HREnabled bool `gorm:"default:true" json:"hr_enabled"`
	// HRAutoApprove: usulan rekrut & pecat dari HR langsung dijalankan tanpa persetujuan pengguna
	HRAutoApprove bool `gorm:"default:false" json:"hr_auto_approve"`
	// HRWarnScore: rata-rata nilai QA di bawah ambang ini (min. 3 todo dinilai) memicu SP
	HRWarnScore int `gorm:"default:60" json:"hr_warn_score"`
	// HRIdleDays: pegawai teknis tanpa todo selama sekian hari (peran masih diisi rekan) diusulkan dipecat
	HRIdleDays int `gorm:"default:30" json:"hr_idle_days"`
	// HRHireLoad: rata-rata todo aktif per pegawai seperan di atas ambang ini memicu usulan rekrut
	HRHireLoad int       `gorm:"default:3" json:"hr_hire_load"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Agenda pegawai: todo milik pegawai SENDIRI, terpisah dari todo proyek (tabel `todos`). Pegawai membuatnya
// untuk dirinya (refleksi todo selesai, belajar dari nilai QA rendah, menyelidiki run gagal, usulan
// perbaikan cara kerja vibeasik) lalu mengerjakannya saat senggang: lewat LLM pool (model dipilih sesuai
// kebutuhan dengan cadangan provider lain) atau lewat agent CLI lokal (orkestrator, read-only, jendela
// terminal selalu dibuka di laptop). Hasilnya menjadi pengetahuan pegawai atau usulan untuk pengguna.

// Jenis agenda.
const (
	AgendaReflect     = "reflect"     // menarik pelajaran dari todo yang baru selesai
	AgendaLearn       = "learn"       // mempelajari topik (kelemahan dari QA, stack baru)
	AgendaInvestigate = "investigate" // memeriksa kode/proyek langsung (CLI agent, read-only)
	AgendaImprove     = "improve"     // usulan perbaikan cara kerja/alat vibeasik untuk pengguna
)

// Pemicu agenda.
const (
	AgendaTriggerTodoDone  = "todo_completed" // todo pegawai selesai
	AgendaTriggerLowScore  = "qa_low_score"   // nilai QA di bawah ambang
	AgendaTriggerRunFailed = "run_failed"     // run orkestrator todo pegawai gagal
	AgendaTriggerSelf      = "self"           // dibuat pegawai saat mengerjakan agenda lain
	AgendaTriggerAgent     = "agent"          // agent coding lewat MCP
	AgendaTriggerManual    = "manual"         // pengguna
	AgendaTriggerIdle      = "idle"           // upgrade skill saat santai (model obrolan, tanpa agenda lanjutan)
	AgendaTriggerHR        = "hr_warning"     // rencana perbaikan setelah menerima SP dari HR
	AgendaTriggerGap       = "knowledge_gap"  // teknologi/metode baru yang belum dikenal Pusat Pengetahuan
)

// Status agenda.
const (
	AgendaPending = "pending"
	AgendaRunning = "running" // dikerjakan job antrean (LLM) atau run orkestrator (CLI)
	AgendaDone    = "done"
	AgendaFailed  = "failed"
	AgendaDropped = "dropped" // dibatalkan pengguna / kedaluwarsa
)

// Pelaksana agenda.
const (
	AgendaExecLLM = "llm" // LLM pool
	AgendaExecCLI = "cli" // agent coding CLI lokal lewat orkestrator (terminal dibuka di laptop)
)

// EmployeeAgenda adalah satu todo pribadi pegawai.
type EmployeeAgenda struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	EmployeeID   uint   `gorm:"index" json:"employee_id"`
	EmployeeName string `gorm:"size:128" json:"employee_name"`
	Kind         string `gorm:"size:24;index" json:"kind"`
	Title        string `gorm:"size:255" json:"title"`
	Detail       string `gorm:"type:text" json:"detail"`
	Trigger      string `gorm:"column:trigger_source;size:32;index" json:"trigger"` // kolom bukan `trigger` (kata tercadang MySQL)
	// SourceRef: asal pemicu, mis. "todo:12", "review:4", "run:31", "agenda:7" (unik per pegawai+jenis agar tidak ganda)
	SourceRef string `gorm:"size:64;index" json:"source_ref,omitempty"`
	// Proyek hanya sebagai konteks (direktori kerja CLI); agenda bukan milik proyek.
	ProjectID   *uint  `json:"project_id,omitempty"`
	ProjectName string `gorm:"size:255" json:"project_name,omitempty"`
	WorkDir     string `gorm:"size:1024" json:"work_dir,omitempty"`
	Priority    int    `gorm:"default:50" json:"priority"` // 0–100
	Status      string `gorm:"size:16;index;default:'pending'" json:"status"`
	Executor    string `gorm:"size:8;default:'llm'" json:"executor"` // llm, cli
	// Need: kebutuhan model pilihan pegawai (light/standard/deep); PreferModel: model tertentu "<config_id>:<model>"
	Need        string     `gorm:"size:16;default:'standard'" json:"need"`
	PreferModel string     `gorm:"size:255" json:"prefer_model,omitempty"`
	ModelUsed   string     `gorm:"size:255" json:"model_used,omitempty"`
	Attempts    string     `gorm:"type:text" json:"attempts,omitempty"` // jejak percobaan model (JSON)
	RunID       *uint      `gorm:"index" json:"run_id,omitempty"`       // run orkestrator untuk executor cli
	Result      string     `gorm:"type:text" json:"result,omitempty"`
	Error       string     `gorm:"type:text" json:"error,omitempty"`
	CreatedBy   string     `gorm:"size:128" json:"created_by,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Obrolan santai antar pegawai: dua pegawai yang tidak punya todo aktif mengobrol (dibuat satu kali oleh
// model obrolan yang ringan), masing-masing boleh memetik satu pengetahuan dari rekannya.

// Tempat obrolan santai.
const (
	ConvAtDesk   = "desk"
	ConvAtPantry = "pantry"
)

// Status obrolan santai.
const (
	ConvQueued  = "queued"
	ConvRunning = "running"
	ConvDone    = "done"
	ConvFailed  = "failed"
)

// ConvLine adalah satu ucapan dalam obrolan santai.
type ConvLine struct {
	EmployeeID uint   `json:"employee_id"`
	Name       string `json:"name"`
	Text       string `json:"text"`
}

// EmployeeConversation adalah satu obrolan santai antara dua pegawai.
type EmployeeConversation struct {
	ID      uint       `gorm:"primaryKey" json:"id"`
	AID     uint       `gorm:"column:a_id;index" json:"a_id"`
	AName   string     `gorm:"column:a_name;size:128" json:"a_name"`
	BID     uint       `gorm:"column:b_id;index" json:"b_id"`
	BName   string     `gorm:"column:b_name;size:128" json:"b_name"`
	Topic   string     `gorm:"size:255" json:"topic"`
	Lines   []ConvLine `gorm:"type:text;serializer:json" json:"lines"`
	Learned string     `gorm:"type:text" json:"learned,omitempty"`           // ringkasan pengetahuan yang dipetik
	Trigger string     `gorm:"column:trigger_source;size:16" json:"trigger"` // idle / manual
	// Place: tempat mengobrol di Kantor Agent — "desk" (A menghampiri meja tetangganya B) atau "pantry"
	Place     string     `gorm:"size:16" json:"place"`
	Status    string     `gorm:"size:16;index" json:"status"`
	Model     string     `gorm:"size:255" json:"model,omitempty"`
	Error     string     `gorm:"type:text" json:"error,omitempty"`
	Finished  *time.Time `json:"finished_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Tindakan HR: pegawai HR (People Partner) mengevaluasi kantor secara berkala — memberi Surat Peringatan
// (SP1–SP3) kepada pegawai yang nilai QA-nya terus rendah, mengusulkan pemecatan pegawai yang tetap buruk
// setelah SP3 atau sudah tidak relevan (lama tanpa todo sementara perannya diisi rekan), dan memberi tahu
// bahwa kantor butuh pegawai baru (beban peran terlalu tinggi / todo terus dipinjamkan ke divisi lain).
// SP langsung berlaku; rekrut & pecat menunggu persetujuan pengguna kecuali hr_auto_approve aktif.

// Jenis tindakan HR.
const (
	HRHire    = "hire"    // usulan rekrut pegawai baru untuk satu peran
	HRWarning = "warning" // Surat Peringatan (Level 1–3)
	HRDismiss = "dismiss" // usulan pemecatan
)

// Status tindakan HR.
const (
	HRProposed   = "proposed"   // usulan rekrut/pecat menunggu keputusan pengguna
	HRActive     = "active"     // SP sedang berlaku
	HRDone       = "done"       // usulan disetujui & dijalankan
	HRRejected   = "rejected"   // usulan ditolak / SP dicabut pengguna
	HRExpired    = "expired"    // SP habis masa berlakunya
	HRCleared    = "cleared"    // SP dicabut HR karena performa pulih
	HRSuperseded = "superseded" // SP digantikan SP tingkat berikutnya
)

// Alasan pemecatan / pemicu rekrutmen.
const (
	DismissPerformance = "performance"    // tetap buruk setelah SP3
	DismissIrrelevant  = "irrelevant"     // sudah tidak relevan
	CauseUnmatchedTodo = "unmatched_todo" // tidak ada pegawai yang mampu menangani todo
)

// HRMetrics adalah angka yang dibaca HR saat mengambil tindakan (disimpan sebagai bukti).
type HRMetrics struct {
	Reviewed             int      `json:"reviewed,omitempty"`              // todo yang dinilai QA dalam jendela evaluasi
	AvgScore             float64  `json:"avg_score,omitempty"`             // rata-rata nilai QA todo tersebut
	IdleDays             int      `json:"idle_days,omitempty"`             // hari sejak todo terakhir
	Members              int      `json:"members,omitempty"`               // pegawai aktif seperan
	ActiveTasks          int      `json:"active_tasks,omitempty"`          // todo aktif seluruh pegawai seperan
	Borrowed             int      `json:"borrowed,omitempty"`              // todo peran ini yang dikerjakan pegawai peran lain (14 hari)
	PeerTasks            int      `json:"peer_tasks,omitempty"`            // todo rekan seperan dalam jendela idle
	UnmatchedTodoID      *uint    `json:"unmatched_todo_id,omitempty"`     // todo yang memicu rekrutmen karena tidak ada yang cocok
	MissingSkills        []string `json:"missing_skills,omitempty"`        // keahlian yang belum ada di kantor
	RecommendedSpecialty string   `json:"recommended_specialty,omitempty"` // saran spesialisasi untuk pegawai baru
}

// EmployeeHRAction adalah satu tindakan/usulan HR.
type EmployeeHRAction struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Kind         string    `gorm:"size:16;index" json:"kind"`
	Level        int       `json:"level,omitempty"` // tingkat SP (1–3)
	EmployeeID   *uint     `gorm:"index" json:"employee_id,omitempty"`
	EmployeeName string    `gorm:"size:128" json:"employee_name,omitempty"`
	Role         string    `gorm:"size:64;index" json:"role"`
	Division     string    `gorm:"size:64" json:"division"`
	Cause        string    `gorm:"size:32" json:"cause,omitempty"` // alasan pemecatan: performance / irrelevant
	Title        string    `gorm:"size:255" json:"title"`
	Reason       string    `gorm:"type:text" json:"reason"` // alasan berbasis angka (deterministik)
	Letter       string    `gorm:"type:text" json:"letter"` // surat/pemberitahuan HR (ditulis ulang LLM persona HR)
	Metrics      HRMetrics `gorm:"type:text;serializer:json" json:"metrics"`
	Status       string    `gorm:"size:16;index" json:"status"`
	IssuerID     *uint     `json:"issuer_id,omitempty"`
	IssuerName   string    `gorm:"size:128" json:"issuer_name,omitempty"`
	// HiredID: pegawai yang direkrut saat usulan rekrut disetujui
	HiredID   *uint  `json:"hired_id,omitempty"`
	HiredName string `gorm:"size:128" json:"hired_name,omitempty"`
	Model     string `gorm:"size:255" json:"model,omitempty"` // model yang menulis surat (kosong = templat)
	// TaskIDs: ID tugas (EmployeeTask) yang menjadi dasar evaluasi SP ini (agar tidak dihitung ulang di SP berikutnya)
	TaskIDs    []uint     `gorm:"type:text;serializer:json" json:"task_ids,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	DecidedAt  *time.Time `json:"decided_at,omitempty"`
	DecidedBy  string     `gorm:"size:128" json:"decided_by,omitempty"`
	DecideNote string     `gorm:"type:text" json:"decide_note,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Perbaikan tertunda: todo yang sudah selesai tetapi nilai QA-nya kurang (mentok batas pengerjaan ulang,
// lulus tipis, atau tidak sempat dikerjakan ulang) disimpan di antrean dan "tidur" sampai ada bukti skill
// naik — pegawainya naik level/keahlian, mendapat pengetahuan baru, atau ada rekan seperan yang terbukti
// unggul di area itu. Sebelum dibuka lagi todo dicek basi (berkasnya sudah diubah commit lain, todo
// sudah diselesaikan ulang, terlalu lama). Nilai lama tidak pernah ditimpa: tersimpan di sini & di EmployeeTask.

// Sumber perbaikan tertunda.
const (
	RevisitLimit      = "limit"      // batas pengerjaan ulang (rework_max) habis, nilai masih rendah
	RevisitBorderline = "borderline" // lulus tipis: rework_score ≤ nilai < revisit_score
	RevisitSkipped    = "skipped"    // nilai di bawah rework_score tetapi tidak pernah dikerjakan ulang
)

// Status perbaikan tertunda.
const (
	RevisitWaiting   = "waiting"   // menunggu bukti skill naik
	RevisitReady     = "ready"     // pemicu terpenuhi, menunggu kuota harian
	RevisitHold      = "hold"      // di-hold: batas siklus ulang tercapai tanpa progress cukup
	RevisitReopened  = "reopened"  // todo dibuka lagi, menunggu dikerjakan & dinilai
	RevisitDone      = "done"      // nilai baru sudah masuk
	RevisitStale     = "stale"     // usang: tidak layak dikerjakan ulang lagi
	RevisitDismissed = "dismissed" // dibatalkan pengguna
)

// RevisitBaseline: keadaan pegawai saat masuk antrean (pembanding pemicu kenaikan skill).
type RevisitBaseline struct {
	Level       int  `json:"level"`
	SkillSum    int  `json:"skill_sum"`
	KnowledgeID uint `json:"knowledge_id"` // id pengetahuan terakhir pegawai
}

// EmployeeRevisit adalah satu todo di antrean perbaikan tertunda (maks satu per todo).
type EmployeeRevisit struct {
	ID           uint            `gorm:"primaryKey" json:"id"`
	TodoID       uint            `gorm:"index" json:"todo_id"`
	TaskID       uint            `gorm:"index" json:"task_id"` // EmployeeTask yang dinilai rendah
	Title        string          `gorm:"size:512" json:"title"`
	ProjectName  string          `gorm:"size:255" json:"project_name,omitempty"`
	Area         string          `gorm:"size:128;index" json:"area,omitempty"`
	EmployeeID   uint            `gorm:"index" json:"employee_id"` // pegawai yang mengerjakan dulu
	EmployeeName string          `gorm:"size:128" json:"employee_name"`
	Cause        string          `gorm:"size:16;index" json:"cause"`
	OldScore     int             `json:"old_score"`
	OldComment   string          `gorm:"type:text" json:"old_comment,omitempty"`
	Priority     int             `json:"priority"` // makin besar makin didahulukan (dampak, jarak ke ambang, umur)
	Baseline     RevisitBaseline `gorm:"type:text;serializer:json" json:"baseline"`
	Status       string          `gorm:"size:16;index" json:"status"`
	Trigger      string          `gorm:"type:text" json:"trigger,omitempty"`      // bukti skill naik yang membangunkannya
	StaleReason  string          `gorm:"type:text" json:"stale_reason,omitempty"` // alasan usang / dibatalkan
	AssigneeID   *uint           `json:"assignee_id,omitempty"`
	AssigneeName string          `gorm:"size:128" json:"assignee_name,omitempty"`
	NewTaskID    *uint           `json:"new_task_id,omitempty"`
	NewScore     *int            `json:"new_score,omitempty"`
	ReopenedAt   *time.Time      `json:"reopened_at,omitempty"`
	FinishedAt   *time.Time      `json:"finished_at,omitempty"`
	// RevisitCycle: siklus ke-N perbaikan tertunda untuk todo ini (1 = pertama, 2 = setelah siklus 1 masih rendah, dst.)
	RevisitCycle int `gorm:"default:1" json:"revisit_cycle"`
	// TodoStatus: status todo saat ini — virtual field, diisi dari join ke tabel todos (tidak disimpan di DB revisit)
	TodoStatus string    `gorm:"-" json:"todo_status,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// QA gate: penilaian QA atas diff branch todo sebelum di-squash ke branch utama. Satu baris = satu putaran.
// Cycle = Todo.rework_count saat dinilai, sehingga putaran dihitung ulang setelah todo dieskalasi ke leader.
type TodoQAGate struct {
	ID        uint     `gorm:"primaryKey" json:"id"`
	TodoID    uint     `gorm:"index" json:"todo_id"`
	Cycle     int      `json:"cycle"`
	Round     int      `json:"round"`
	Score     int      `json:"score"`
	Threshold int      `json:"threshold"`
	Passed    bool     `json:"passed"`
	Escalated bool     `json:"escalated"`
	Feedback  string   `gorm:"type:text" json:"feedback"`
	Checklist []string `gorm:"type:text;serializer:json" json:"checklist"`
	// Fields: nilai panel QC per bidang dari Senior Developer (QC lead)
	Fields       map[string]QCFieldScore `gorm:"type:text;serializer:json" json:"fields,omitempty"`
	EmployeeID   *uint                   `json:"employee_id,omitempty"`
	ReviewerName string                  `gorm:"size:128" json:"reviewer_name"`
	Model        string                  `gorm:"size:160" json:"model"`
	Branch       string                  `gorm:"size:255" json:"branch"`
	Agent        string                  `gorm:"size:128" json:"agent"`
	DiffLines    int                     `json:"diff_lines"`
	DiffFiles    int                     `json:"diff_files"`
	CreatedAt    time.Time               `json:"created_at"`
}

// Fase konsultasi agent coding dengan pegawai pendamping.
const (
	ConsultPlan   = "plan"   // sebelum mengedit: rencana & berkas target
	ConsultStuck  = "stuck"  // di tengah pengerjaan: macet atau ragu
	ConsultReview = "review" // sebelum melapor: pegawai meninjau diff
)

// Putusan pegawai atas satu konsultasi.
const (
	ConsultProceed = "proceed" // lanjut
	ConsultRevise  = "revise"  // ubah dulu sesuai butir arahan
	ConsultStop    = "stop"    // berhenti: salah arah / perlu keputusan pengguna
)

// Seberapa jauh hasil kerja mengikuti arahan (dinilai QA gate dari diff).
const (
	ConsultFollowedYes     = "yes"
	ConsultFollowedPartial = "partial"
	ConsultFollowedNo      = "no"
)

// EmployeeConsult adalah satu konsultasi agent coding dengan pegawai yang mendampingi todo: agent mengetik,
// pegawai menjawab dengan model, pengetahuan, dan riwayatnya sendiri. QA gate lalu menilai apakah arahan itu
// diikuti dan berguna; pemilik bisa menambah penilaiannya sendiri.
type EmployeeConsult struct {
	ID           uint     `gorm:"primaryKey" json:"id"`
	TodoID       uint     `gorm:"index" json:"todo_id"`
	TodoTitle    string   `gorm:"size:512" json:"todo_title"`
	ProjectName  string   `gorm:"size:255" json:"project_name,omitempty"`
	EmployeeID   uint     `gorm:"index" json:"employee_id"`
	EmployeeName string   `gorm:"size:128" json:"employee_name"`
	Cycle        int      `json:"cycle"` // siklus pengerjaan todo (rework_count) saat konsultasi
	Phase        string   `gorm:"size:16;index" json:"phase"`
	Question     string   `gorm:"type:text" json:"question"`
	Verdict      string   `gorm:"size:16" json:"verdict"`
	Advice       string   `gorm:"type:text" json:"advice"`
	Points       []string `gorm:"type:text;serializer:json" json:"points"`
	Risks        []string `gorm:"type:text;serializer:json" json:"risks"`
	Agent        string   `gorm:"size:128" json:"agent"`
	Model        string   `gorm:"size:160" json:"model"`
	Branch       string   `gorm:"size:255" json:"branch,omitempty"`
	DiffLines    int      `json:"diff_lines,omitempty"`
	StepsFailed  int      `json:"steps_failed,omitempty"` // langkah pegangan yang gagal saat fase review
	// Penilaian QA gate: apakah diff mengikuti arahan, dan apakah arahan itu benar & berguna.
	Followed   string     `gorm:"size:16;index" json:"followed,omitempty"`
	Helpful    *bool      `json:"helpful,omitempty"`
	JudgeNote  string     `gorm:"type:text" json:"judge_note,omitempty"`
	GateScore  *int       `json:"gate_score,omitempty"`
	GatePassed *bool      `json:"gate_passed,omitempty"`
	JudgedBy   string     `gorm:"size:128" json:"judged_by,omitempty"`
	JudgedAt   *time.Time `json:"judged_at,omitempty"`
	// Penilaian pemilik: 1 = arahan bagus, -1 = arahan buruk, 0 = belum dinilai.
	OwnerRating int       `json:"owner_rating"`
	OwnerNote   string    `gorm:"type:text" json:"owner_note,omitempty"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

// QCFieldScore adalah nilai satu bidang panel QC: level 0–4 sesuai deskripsi level bidang + catatan konkret.
type QCFieldScore struct {
	Level int    `json:"level"`
	Note  string `json:"note,omitempty"`
}

// EmployeeTurnover mencatat riwayat suksesi / penggantian pegawai kantor: siapa digantikan oleh siapa,
// alasan pemberhentian/pemecatan, kapan pergantian terjadi, serta warisan catatan nilai tinggi & ilmu yang diserahkan.
type EmployeeTurnover struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	FormerEmployeeID    uint       `gorm:"index" json:"former_employee_id"`
	FormerEmployeeName  string     `gorm:"size:128" json:"former_employee_name"`
	FormerRole          string     `gorm:"size:64;index" json:"former_role"`
	FormerDivision      string     `gorm:"size:64" json:"former_division"`
	DismissReason       string     `gorm:"type:text" json:"dismiss_reason"`
	DismissedAt         time.Time  `json:"dismissed_at"`
	SuccessorID         *uint      `gorm:"index" json:"successor_id,omitempty"`
	SuccessorName       string     `gorm:"size:128" json:"successor_name,omitempty"`
	HiredAt             *time.Time `json:"hired_at,omitempty"`
	InheritedNotesCount int        `json:"inherited_notes_count"`
	InheritedSkills     []string   `gorm:"type:text;serializer:json" json:"inherited_skills,omitempty"`
	InheritedSummary    string     `gorm:"type:text" json:"inherited_summary,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// EmployeeStepStat mencatat riwayat satu langkah pegangan kerja pada satu pegawai: berapa kali lolos/gagal saat
// hasil kerjanya diperiksa, dan kotak ingatannya (1 = baru gagal, selalu diingat; makin tinggi makin jarang diulang).
type EmployeeStepStat struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	EmployeeID uint       `gorm:"uniqueIndex:idx_employee_step" json:"employee_id"`
	StepID     string     `gorm:"size:32;uniqueIndex:idx_employee_step" json:"step_id"`
	Box        int        `json:"box"`
	Passed     int        `json:"passed"`
	Failed     int        `json:"failed"`
	LastTodoID uint       `json:"last_todo_id"`
	LastNote   string     `gorm:"type:text" json:"last_note,omitempty"`
	LastAt     *time.Time `json:"last_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Status langkah kerja yang tumbuh dari pengalaman.
const (
	LearnedStepTrial    = "trial"    // sedang dicoba: dihitung diam-diam, belum memengaruhi nilai
	LearnedStepActive   = "active"   // dipakai di lembar kerja & pemeriksaan hasil
	LearnedStepRetired  = "retired"  // dihentikan (salah sasaran, tidak berguna, atau sudah naik menjadi langkah peran)
	LearnedStepRejected = "rejected" // usulan ditolak saat divalidasi; disimpan sebagai jejak
)

// EmployeeLearnedStep adalah langkah kerja yang tumbuh dari pengalaman, bukan bawaan: diusulkan dari keluhan
// penilai yang berulang, divalidasi & dicoba dulu oleh program, lalu dipakai pegawai pemiliknya (OwnerEmployeeID)
// atau seluruh perannya (OwnerEmployeeID kosong).
type EmployeeLearnedStep struct {
	ID              uint     `gorm:"primaryKey" json:"id"`
	Code            string   `gorm:"size:16;uniqueIndex" json:"code"`
	Role            string   `gorm:"size:64;index" json:"role"`
	OwnerEmployeeID *uint    `gorm:"index" json:"owner_employee_id,omitempty"`
	OwnerName       string   `gorm:"size:128" json:"owner_name,omitempty"`
	Phase           string   `gorm:"size:16" json:"phase"`
	Title           string   `gorm:"size:160" json:"title"`
	Instruction     string   `gorm:"type:text" json:"instruction"`
	Evidence        string   `gorm:"type:text" json:"evidence,omitempty"`
	Risk            int      `json:"risk"`
	Tags            []string `gorm:"type:text;serializer:json" json:"tags"`
	Skill           string   `gorm:"size:64" json:"skill"`
	// Areas & Words: kapan langkah berlaku (area berkas yang disentuh, atau kata di tugas bila berkas belum diketahui)
	Areas []string `gorm:"type:text;serializer:json" json:"areas"`
	Words []string `gorm:"type:text;serializer:json" json:"words"`
	// Pattern: pola yang tidak boleh muncul di baris baru pada berkas area FileArea ("" = hanya dinilai penilai).
	// Context: pola yang harus ada di berkas agar ia diperiksa (opsional).
	Pattern      string   `gorm:"type:text" json:"pattern,omitempty"`
	FileArea     string   `gorm:"size:16" json:"file_area,omitempty"`
	Context      string   `gorm:"type:text" json:"context,omitempty"`
	Message      string   `gorm:"size:255" json:"message,omitempty"`
	BadExamples  []string `gorm:"type:text;serializer:json" json:"bad_examples,omitempty"`
	GoodExamples []string `gorm:"type:text;serializer:json" json:"good_examples,omitempty"`
	Status       string   `gorm:"size:16;index" json:"status"`
	StatusReason string   `gorm:"type:text" json:"status_reason,omitempty"`
	// Asal usulan: keluhan penilai yang memicunya (Trigger), sumbernya, dan model penulis usulan
	Source     string `gorm:"size:32" json:"source"`
	SourceRef  string `gorm:"size:255" json:"source_ref,omitempty"`
	Trigger    string `gorm:"type:text" json:"trigger,omitempty"`
	Complaints int    `json:"complaints"`
	Model      string `gorm:"size:160" json:"model,omitempty"`
	// Angka masa percobaan: berapa kali diperiksa, berapa kali menandai, dan jumlah nilai penilai saat menandai/tidak
	TrialChecks   int        `json:"trial_checks"`
	TrialHits     int        `json:"trial_hits"`
	HitScoreSum   int        `json:"hit_score_sum"`
	MissScoreSum  int        `json:"miss_score_sum"`
	BacktestFiles int        `json:"backtest_files"`
	BacktestHits  int        `json:"backtest_hits"`
	ActivatedAt   *time.Time `json:"activated_at,omitempty"`
	RetiredAt     *time.Time `json:"retired_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Status peran pegawai yang tumbuh dari pekerjaan.
const (
	CustomRoleActive   = "active"
	CustomRoleRejected = "rejected" // usulan ditolak saat divalidasi; disimpan sebagai jejak
	CustomRoleRetired  = "retired"
)

// RoleSeed adalah satu butir pengetahuan bawaan sebuah peran.
type RoleSeed struct {
	Topic   string `json:"topic"`
	Content string `json:"content"`
	Kind    string `json:"kind"`
	Weight  int    `json:"weight"`
}

// EmployeeCustomRole adalah peran pegawai yang tidak ada di katalog bawaan: lahir dari sekelompok todo yang tidak
// punya peran pas. Keywords mengarahkan todo ke peran ini.
type EmployeeCustomRole struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Key          string         `gorm:"size:64;uniqueIndex" json:"key"`
	Title        string         `gorm:"size:128" json:"title"`
	Division     string         `gorm:"size:32" json:"division"`
	Icon         string         `gorm:"size:64" json:"icon,omitempty"`
	Technical    bool           `json:"technical"`
	Brief        string         `gorm:"type:text" json:"brief"`
	Specialty    string         `gorm:"size:255" json:"specialty"`
	Stack        []string       `gorm:"type:text;serializer:json" json:"stack"`
	Skills       map[string]int `gorm:"type:text;serializer:json" json:"skills"`
	Categories   []string       `gorm:"type:text;serializer:json" json:"categories"`
	Keywords     []string       `gorm:"type:text;serializer:json" json:"keywords"`
	Knowledge    []RoleSeed     `gorm:"type:text;serializer:json" json:"knowledge"`
	Status       string         `gorm:"size:16;index" json:"status"`
	StatusReason string         `gorm:"type:text" json:"status_reason,omitempty"`
	Trigger      string         `gorm:"type:text" json:"trigger,omitempty"` // judul todo yang memicunya
	SourceRef    string         `gorm:"size:255" json:"source_ref,omitempty"`
	Todos        int            `json:"todos"`
	Coverage     int            `json:"coverage"` // persen todo pemicu yang tertangkap kata kuncinya
	Overlap      int            `json:"overlap"`  // persen todo peran lain yang ikut tertangkap
	Model        string         `gorm:"size:160" json:"model,omitempty"`
	HiredID      *uint          `json:"hired_id,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// Arah paket pegawai.
const (
	PassportExport = "export"
	PassportImport = "import"
)

// EmployeePassport mencatat paket pegawai yang diekspor dari atau diimpor ke kantor ini: nomor seri tiap salinan,
// sidik jari kunci penanda tangan asalnya, dan rekam jejak yang dibawa paket itu.
type EmployeePassport struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	EmployeeID uint      `gorm:"index" json:"employee_id"`
	Name       string    `gorm:"size:128" json:"name"`
	Serial     string    `gorm:"size:64;uniqueIndex" json:"serial"`
	Direction  string    `gorm:"size:16;index" json:"direction"`
	CopyNo     int       `json:"copy_no"`
	License    string    `gorm:"size:32" json:"license"`
	Origin     string    `gorm:"size:64" json:"origin"` // sidik jari kunci publik penanda tangan
	Record     string    `gorm:"type:text" json:"record,omitempty"`
	File       string    `gorm:"size:512" json:"file,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
