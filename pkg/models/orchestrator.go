package models

import "time"

// Status run orkestrator agent
const (
	AgentRunQueued    = "queued"
	AgentRunRunning   = "running"
	AgentRunCompleted = "completed"
	AgentRunFailed    = "failed"
	AgentRunCancelled = "cancelled"
)

// Jenis run: tugas (agent mengerjakan todo/prompt) atau merge (agent lain menggabungkan & memeriksa branch tugas)
const (
	AgentRunKindTask  = "task"
	AgentRunKindMerge = "merge"
)

// Status merge run tugas yang dikerjakan di git worktree terpisah
const (
	MergeStatePending   = "pending"    // branch siap, run merge antre
	MergeStateMerging   = "merging"    // agent merge sedang bekerja
	MergeStateMerged    = "merged"     // branch sudah masuk ke branch dasar
	MergeStateFailed    = "failed"     // merge gagal/dibatalkan; branch disimpan untuk diulang
	MergeStateNoChanges = "no_changes" // agent tidak mengubah berkas apa pun
	MergeStateKept      = "kept"       // run gagal tetapi perubahan sebagian disimpan di branch
	MergeStateQAReview  = "qa_review"  // branch sedang dinilai QA gate sebelum di-squash
	MergeStateQARework  = "qa_rework"  // nilai QA gate kurang; dikerjakan ulang di branch yang sama (merge_run_id = run ulang)
)

// Mode isolasi run tugas
const (
	IsolationWorktree = "worktree" // run todo di repo git dikerjakan di worktree + branch sendiri, lalu di-merge agent lain
	IsolationShared   = "shared"   // semua agent bekerja langsung di checkout utama
)

// Strategi router: siapa yang memilih profil worker ketika run dikirim dalam mode otomatis
const (
	RouterRules = "rules" // heuristik lokal (kategori, prioritas, sisa kuota)
	RouterJev   = "jev"   // System One / TypeSafe Jev primitive "choice"
	RouterLLM   = "llm"   // LLM eksternal (OpenAI-compatible) aktif
)

// AgentProfile adalah satu "kursi pekerja" yang bisa disuruh orkestrator:
// CLI agent + model + effort + akun. Contoh: "Antigravity · Akun 3 · gemini-3.8-flash-high"
// atau "Claude Code · opus · high".
type AgentProfile struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"size:160;not null" json:"name"`
	AgentKey     string `gorm:"size:80;index;not null" json:"agent_key"` // claude, agy, opencode, copilot, aider, atau key custom AI Hub
	Model        string `gorm:"size:160" json:"model"`
	Effort       string `gorm:"size:32" json:"effort"`       // low, medium, high, xhigh, max (kosong = bawaan CLI)
	AccountRef   string `gorm:"size:512" json:"account_ref"` // agy: id profil agy-profile; claude: CLAUDE_CONFIG_DIR
	AccountLabel string `gorm:"size:160" json:"account_label"`
	// Autonomy: "full" = agent boleh menjalankan perintah tanpa bertanya (skip permissions), "edits" = hanya edit file
	Autonomy string `gorm:"size:16;default:'full'" json:"autonomy"`
	// Strengths dipakai router: kategori todo (feature, security, ui_ux, ...) dan tingkat ("quick", "complex"), dipisah koma
	Strengths     string     `gorm:"size:512" json:"strengths"`
	Notes         string     `gorm:"type:text" json:"notes"`
	SortOrder     int        `gorm:"default:100" json:"sort_order"`
	MaxConcurrent int        `gorm:"default:1" json:"max_concurrent"`
	DailyLimit    int        `gorm:"default:0" json:"daily_limit"` // 0 = tanpa batas run per hari
	Enabled       bool       `gorm:"default:true" json:"enabled"`
	CooldownUntil *time.Time `json:"cooldown_until,omitempty"` // diisi saat kuota habis terdeteksi
	LastError     string     `gorm:"type:text" json:"last_error,omitempty"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AgentRun adalah satu tugas yang dikirim ke agent coding di laptop (tersimpan, bertahan saat engine restart).
type AgentRun struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	TodoID      *uint  `gorm:"index" json:"todo_id,omitempty"`
	ProjectID   *uint  `gorm:"index" json:"project_id,omitempty"`
	ComponentID *uint  `json:"component_id,omitempty"`
	ProjectName string `gorm:"size:255" json:"project_name,omitempty"`
	WorkDir     string `gorm:"size:1024" json:"work_dir"`
	// TargetFiles: berkas target todo (path absolut, dipisah baris baru); dipakai penjadwal untuk menjalankan
	// beberapa agent paralel di repositori yang sama selama berkas targetnya tidak bertabrakan.
	TargetFiles string `gorm:"type:text" json:"target_files,omitempty"`
	// TodoSnapshot: salinan todo saat dispatch agar todo bisa dipulihkan bila terhapus selama run.
	TodoSnapshot string `gorm:"type:text" json:"-"`
	// Isolasi git: run tugas bekerja di ExecDir (worktree) pada Branch dari BaseBranch@BaseCommit milik RepoRoot;
	// run merge (Kind=merge, ParentRunID) menggabungkan Branch itu ke BaseBranch di checkout utama.
	Kind        string `gorm:"size:16;default:'task'" json:"kind"`
	ParentRunID *uint  `gorm:"index" json:"parent_run_id,omitempty"`
	RepoRoot    string `gorm:"size:1024" json:"repo_root,omitempty"`
	ExecDir     string `gorm:"size:1024" json:"exec_dir,omitempty"`
	Branch      string `gorm:"size:255" json:"branch,omitempty"`
	BaseBranch  string `gorm:"size:255" json:"base_branch,omitempty"`
	BaseCommit  string `gorm:"size:64" json:"base_commit,omitempty"`
	MergeState  string `gorm:"size:16" json:"merge_state,omitempty"`
	MergeRunID  *uint  `json:"merge_run_id,omitempty"`
	Title       string `gorm:"size:512" json:"title"`
	Prompt      string `gorm:"type:text" json:"prompt"`
	Category    string `gorm:"size:64" json:"category,omitempty"`
	Priority    string `gorm:"size:64" json:"priority,omitempty"`
	Status      string `gorm:"size:32;index;default:'queued'" json:"status"`
	// Pemilihan worker: profil yang diminta (kosong = otomatis lewat router) dan yang akhirnya dipakai
	RequestedProfileID *uint      `json:"requested_profile_id,omitempty"`
	PoolAgentKey       string     `gorm:"size:80" json:"pool_agent_key,omitempty"` // mode otomatis dibatasi ke satu agent (mis. semua akun agy)
	Router             string     `gorm:"size:32" json:"router"`
	ProfileID          *uint      `gorm:"index" json:"profile_id,omitempty"`
	ProfileName        string     `gorm:"size:160" json:"profile_name,omitempty"`
	AgentKey           string     `gorm:"size:80" json:"agent_key,omitempty"`
	Model              string     `gorm:"size:160" json:"model,omitempty"`
	Effort             string     `gorm:"size:32" json:"effort,omitempty"`
	RoutedBy           string     `gorm:"size:32" json:"routed_by,omitempty"` // manual, rules, jev, llm
	RouteReason        string     `gorm:"type:text" json:"route_reason,omitempty"`
	ComplexityLevel    string     `gorm:"size:16" json:"complexity_level,omitempty"` // quick, normal, complex (dari todo atau taksiran prompt)
	ComplexityScore    float64    `json:"complexity_score"`
	TimeoutMin         int        `gorm:"default:60" json:"timeout_min"`
	Attempts           int        `gorm:"default:0" json:"attempts"`
	ExcludedProfiles   string     `gorm:"size:512" json:"-"`                 // profil yang kehabisan kuota pada run ini (dipisah koma)
	Output             string     `gorm:"type:text" json:"output,omitempty"` // ekor log (dibatasi)
	ResultText         string     `gorm:"type:text" json:"result_text,omitempty"`
	ExitCode           int        `json:"exit_code"`
	Error              string     `gorm:"type:text" json:"error,omitempty"`
	QuotaHit           bool       `json:"quota_hit"`
	CreatedAt          time.Time  `json:"created_at"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	DurationMs         int64      `json:"duration_ms"`
	// ReadOnly: run pemeriksaan (agenda pegawai) — tanpa worktree/merge, agent dilarang mengubah berkas.
	ReadOnly bool `gorm:"default:false" json:"read_only,omitempty"`
	// ShowTerminal: buka jendela terminal di laptop untuk run ini walau pengaturan global show_terminal mati.
	ShowTerminal bool `gorm:"default:false" json:"show_terminal,omitempty"`
	// WaitReason: alasan run masih antre (diisi penjadwal di memori, tidak disimpan)
	WaitReason string `gorm:"-" json:"wait_reason,omitempty"`
}

// OrchestratorSettings menyimpan aturan global orkestrator (satu baris).
type OrchestratorSettings struct {
	ID                uint   `gorm:"primaryKey" json:"id"`
	Router            string `gorm:"size:32;default:'rules'" json:"router"`
	MaxConcurrent     int    `gorm:"default:2" json:"max_concurrent"`
	DefaultTimeoutMin int    `gorm:"default:60" json:"default_timeout_min"`
	CooldownMin       int    `gorm:"default:60" json:"cooldown_min"`
	AutoRotate        bool   `gorm:"default:true" json:"auto_rotate"`        // pindah ke profil lain saat kuota habis
	AutoCompleteTodo  bool   `gorm:"default:true" json:"auto_complete_todo"` // tandai todo selesai bila agent sukses
	ExtraInstructions string `gorm:"type:text" json:"extra_instructions"`    // ditambahkan ke setiap prompt
	// PreferredAgent: mode otomatis hanya memakai profil agent ini (mis. "agy" yang punya banyak akun) selama ada profilnya
	PreferredAgent string `gorm:"size:80;default:'agy'" json:"preferred_agent"`
	// EscalateMinScore: skor kerumitan minimal agar run boleh dikerjakan agent lain (0 = tidak pernah eskalasi)
	EscalateMinScore float64 `gorm:"default:0" json:"escalate_min_score"`
	// AdaptiveEffort: profil tanpa effort tetap memakai effort (dan varian model flash) sesuai kerumitan tugas
	AdaptiveEffort bool `gorm:"default:true" json:"adaptive_effort"`
	// ParallelSameDir: run di repositori yang sama boleh paralel bila berkas targetnya diketahui dan tidak bertabrakan
	ParallelSameDir bool `gorm:"default:true" json:"parallel_same_dir"`
	// Isolation: worktree (default; tiap run todo di branch sendiri lalu di-merge agent lain) atau shared
	Isolation string `gorm:"size:16;default:'worktree'" json:"isolation"`
	// MergeProfileID: profil worker khusus untuk agent merge (nil = dipilih router)
	MergeProfileID *uint `json:"merge_profile_id,omitempty"`
	// ShowTerminal: tiap run membuka jendela terminal di desktop laptop yang menampilkan log live (juga untuk run dari LAN)
	ShowTerminal bool `gorm:"default:false" json:"show_terminal"`
	// MergeSimilar: sebelum dispatch, todo pending yang serupa dilebur menjadi satu todo gabungan (Aturan 43)
	MergeSimilar bool      `gorm:"default:true" json:"merge_similar"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AgentDispatchRequest dikirim UI/MCP untuk menyuruh agent bekerja.
type AgentDispatchRequest struct {
	TodoID       uint   `json:"todo_id"`
	ProjectID    uint   `json:"project_id"`
	ComponentID  uint   `json:"component_id"`
	WorkDir      string `json:"work_dir"`
	Title        string `json:"title"`
	Prompt       string `json:"prompt"`         // wajib bila tanpa todo
	Instructions string `json:"instructions"`   // catatan tambahan dari pengguna
	ProfileID    uint   `json:"profile_id"`     // 0 = otomatis (router)
	PoolAgentKey string `json:"pool_agent_key"` // mode otomatis hanya memilih dari profil agent ini
	Router       string `json:"router"`         // override strategi router
	TimeoutMin   int    `json:"timeout_min"`
	ReadOnly     bool   `json:"read_only"`     // pemeriksaan tanpa mengubah berkas (tanpa worktree/merge)
	ShowTerminal bool   `json:"show_terminal"` // paksa jendela terminal di laptop untuk run ini
}

// AgentBatchDispatchRequest mengirim banyak todo sekaligus ke orkestrator (tombol "Suruh Agent Otomatis").
type AgentBatchDispatchRequest struct {
	TodoIDs      []uint `json:"todo_ids"`
	ProjectID    uint   `json:"project_id"` // tanpa todo_ids: semua todo pending proyek ini
	PoolAgentKey string `json:"pool_agent_key"`
	Router       string `json:"router"`
	Instructions string `json:"instructions"`
	TimeoutMin   int    `json:"timeout_min"`
}
