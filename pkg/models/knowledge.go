package models

import "time"

// Pusat Pengetahuan (satu pintu): semua patokan & pengetahuan UMUM pekerja AI vibeasik — rubrik guideline,
// checklist audit, pelajaran vonis audit, keahlian pegawai, hasil belajar teknologi baru — disimpan sebagai
// node yang bisa dicari (vektor TF-IDF), bertaut (link), berkelompok (klaster), dan belajar dari hasil kerja
// (penghitung helpful/harmful). Fakta khusus satu proyek tetap milik dewan eksekutif (ExecutiveKnowledge).

// Jenis node (tipe memori ala CoALA: semantik = rule/concept, prosedural = procedure, episodik → lesson).
const (
	KnowledgeKindRule      = "rule"      // patokan normatif: yang wajib/dilarang
	KnowledgeKindProcedure = "procedure" // cara mengerjakan: langkah, pola, perintah
	KnowledgeKindLesson    = "lesson"    // pelajaran dari pengalaman (berhasil/gagal, audit keliru/benar)
	KnowledgeKindConcept   = "concept"   // pengetahuan dasar: bahasa, framework, metode
)

// KnowledgeKinds: jenis node yang sah.
var KnowledgeKinds = []string{KnowledgeKindRule, KnowledgeKindProcedure, KnowledgeKindLesson, KnowledgeKindConcept}

// Asal node.
const (
	KnowledgeSrcGuideline = "guideline"       // berkas guidelines/*.json (sumber kebenaran tetap berkasnya)
	KnowledgeSrcChecklist = "audit_checklist" // checklist review audit LLM per ekosistem
	KnowledgeSrcVerdict   = "audit_verdict"   // pelajaran dari vonis audit keliru/benar
	KnowledgeSrcEmployee  = "employee"        // cermin pengetahuan pribadi pegawai (employee_knowledge)
	KnowledgeSrcPromoted  = "promoted"        // pengetahuan pribadi yang terbukti berguna lalu dibagikan ke kantor
	KnowledgeSrcGap       = "gap_research"    // hasil pegawai mempelajari teknologi/metode baru (celah pengetahuan)
	KnowledgeSrcReview    = "qa_review"       // best practice dari penilai QA (model ≥ Opus 5.5) atas hasil kerja
	KnowledgeSrcDesign    = "design_library"  // cermin Galeri Desain: kesimpulan per jenis website, pola teramati, referensi
	KnowledgeSrcTheory    = "design_theory"   // teori desain & teori warna (designlib/theory.json) bertaut ke contoh di Galeri Desain
	KnowledgeSrcLesson    = "design_lesson"   // pelajaran desain before/after (gambar awal, gambar akhir, narasi konten)
	KnowledgeSrcManual    = "manual"          // diajarkan manusia
	KnowledgeSrcAgent     = "agent"           // dicatat agent coding lewat MCP
)

// AuthoritativeSources: sumber yang berkedudukan di atas pengetahuan umum LLM dan pegawai. Galeri Desain adalah
// pengetahuan hasil mempelajari website nyata, jadi bukti ✓/✗ dari nilai QA pegawai tidak mengubah bobotnya dan
// temu-kembali memberinya utilitas penuh. Yang dinilai QA adalah kepatuhan pegawai pada galeri, bukan galerinya.
var AuthoritativeSources = []string{KnowledgeSrcDesign}

// KnowledgeAuthoritative: node dari sumber ini tidak ikut dinilai oleh hasil kerja pegawai.
func KnowledgeAuthoritative(source string) bool {
	for _, s := range AuthoritativeSources {
		if s == source {
			return true
		}
	}
	return false
}

// Status node.
const (
	KnowledgeNodeActive  = "active"
	KnowledgeNodeMerged  = "merged"  // dilebur ke node lain yang hampir sama
	KnowledgeNodeRetired = "retired" // terbukti merugikan / sumbernya hilang / dihapus manusia
)

// KnowledgeNode adalah satu butir pengetahuan di Pusat Pengetahuan.
type KnowledgeNode struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// Key: identitas di sumbernya (guideline:SEC-001, audit:go:2, employee:15, promoted:40); kosong = node asli pusat.
	Key     string   `gorm:"column:source_key;size:191;index" json:"key,omitempty"` // `key` kata tercadang MySQL
	Kind    string   `gorm:"size:16;index" json:"kind"`
	Domain  string   `gorm:"size:48;index" json:"domain"` // ekosistem (go, web, flutter, …) atau area (security, testing, …)
	Tags    []string `gorm:"type:text;serializer:json" json:"tags"`
	Title   string   `gorm:"size:191" json:"title"`
	Content string   `gorm:"type:text" json:"content"`
	Source  string   `gorm:"size:32;index" json:"source"`
	// OwnerEmployeeID: pengetahuan pribadi pegawai; nil = pustaka bersama seluruh kantor.
	OwnerEmployeeID *uint  `gorm:"index" json:"owner_employee_id,omitempty"`
	OwnerName       string `gorm:"size:128" json:"owner_name,omitempty"`
	// Weight: kepentingan awal 0–100. Helpful/Harmful: bukti dari hasil kerja (nilai QA, vonis audit, umpan balik).
	Weight       int        `json:"weight"`
	Helpful      int        `json:"helpful"`
	Harmful      int        `json:"harmful"`
	Uses         int        `json:"uses"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	ClusterID    uint       `gorm:"index" json:"cluster_id"`
	MergedIntoID *uint      `json:"merged_into_id,omitempty"`
	// NeedsReview: patokan bersumber berkas yang berkali-kali terbukti keliru — berkasnya perlu ditinjau manusia/agent.
	NeedsReview  bool      `gorm:"index" json:"needs_review"`
	Status       string    `gorm:"size:16;index;default:'active'" json:"status"`
	StatusReason string    `gorm:"type:text" json:"status_reason,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Utility: rata-rata posterior Beta(helpful+1, harmful+1) — 0.5 untuk node yang belum teruji.
func (n KnowledgeNode) Utility() float64 {
	return float64(n.Helpful+1) / float64(n.Helpful+n.Harmful+2)
}

// KnowledgeLink: tautan antar node yang isinya berdekatan (gaya Zettelkasten / A-MEM), dipakai memperluas pencarian.
type KnowledgeLink struct {
	ID       uint    `gorm:"primaryKey" json:"id"`
	FromID   uint    `gorm:"index" json:"from_id"`
	ToID     uint    `gorm:"index" json:"to_id"`
	Strength float64 `json:"strength"` // kemiripan kosinus 0–1
}

// KnowledgeCluster: kelompok node yang membahas konsep yang sama (dihitung ulang tiap kurasi).
type KnowledgeCluster struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Label     string    `gorm:"size:191" json:"label"`
	Domain    string    `gorm:"size:48" json:"domain"`
	Terms     []string  `gorm:"type:text;serializer:json" json:"terms"`
	Size      int       `json:"size"`
	Utility   float64   `json:"utility"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Hasil pemakaian node.
const (
	KnowledgeOutcomeHelpful = "helpful"
	KnowledgeOutcomeHarmful = "harmful"
	KnowledgeOutcomeNeutral = "neutral"
)

// KnowledgeUse: node yang disisipkan ke prompt sebuah pekerjaan; diberi nilai saat hasil kerjanya diketahui.
type KnowledgeUse struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	NodeID     uint      `gorm:"index" json:"node_id"`
	Context    string    `gorm:"size:191;index" json:"context"` // todo:12, audit:3:path/berkas.go, plan:4
	EmployeeID *uint     `json:"employee_id,omitempty"`
	Outcome    string    `gorm:"size:16;index" json:"outcome,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// Status celah pengetahuan.
const (
	KnowledgeGapOpen      = "open"      // terdeteksi, belum ada yang mempelajari
	KnowledgeGapLearning  = "learning"  // agenda belajar pegawai sudah dibuat
	KnowledgeGapLearned   = "learned"   // node baru sudah masuk pusat pengetahuan
	KnowledgeGapDismissed = "dismissed" // diabaikan manusia / gagal berulang
)

// KnowledgeGap: teknologi/metode yang muncul di pekerjaan tetapi belum punya pengetahuan di kantor.
type KnowledgeGap struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Domain       string     `gorm:"size:48;index" json:"domain"`
	Topic        string     `gorm:"size:191" json:"topic"`
	Sample       string     `gorm:"type:text" json:"sample,omitempty"` // contoh konteks tempat celah terlihat
	Hits         int        `json:"hits"`
	Auto         bool       `json:"auto"` // terdeteksi otomatis: ekosistem yang muncul di pekerjaan tanpa satu node pun
	Attempts     int        `json:"attempts"`
	Status       string     `gorm:"size:16;index;default:'open'" json:"status"`
	EmployeeID   *uint      `json:"employee_id,omitempty"`
	EmployeeName string     `gorm:"size:128" json:"employee_name,omitempty"`
	AgendaID     *uint      `json:"agenda_id,omitempty"`
	NodesLearned int        `json:"nodes_learned"`
	Note         string     `gorm:"type:text" json:"note,omitempty"`
	LearnedAt    *time.Time `json:"learned_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
