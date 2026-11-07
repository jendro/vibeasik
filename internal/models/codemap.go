package models

import "time"

// CodeMap adalah metadata peta kode sebuah proyek: indeks per berkas (gaya RAG) yang membantu AI agent
// menemukan lokasi kode tanpa menjelajah seluruh repositori. Isi per berkas ada di CodeMapEntry.
type CodeMap struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ProjectID     uint       `gorm:"uniqueIndex;not null" json:"project_id"`
	RootPath      string     `gorm:"size:1024" json:"root_path"` // root efektif peta (leluhur komponen bila root proyek terlalu umum)
	FileCount     int        `json:"file_count"`
	SymbolCount   int        `json:"symbol_count"`
	EnrichedCount int        `json:"enriched_count"` // berkas yang sudah punya ringkasan LLM
	LanguagesJSON string     `gorm:"type:text" json:"languages_json"`
	KindsJSON     string     `gorm:"type:text" json:"kinds_json"`
	BuildMs       int64      `json:"build_ms"`
	ParserVersion int        `json:"parser_version"`
	BuiltAt       time.Time  `json:"built_at"`
	ExportedAt    *time.Time `json:"exported_at,omitempty"`
}

// CodeSymbol adalah simbol penting dalam berkas (fungsi, tipe, komponen, heading dokumen).
type CodeSymbol struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"` // func, method, type, interface, class, const, component, store, heading, table
	Line     int    `json:"line"`
	Exported bool   `json:"exported,omitempty"`
	Doc      string `json:"doc,omitempty"`
}

// CodeMapEntry adalah satu "chunk" peta kode: satu berkas beserta peran, simbol, dependensi, rute, dan ringkasannya.
type CodeMapEntry struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProjectID     uint      `gorm:"index:idx_codemap_project_path,priority:1;not null" json:"project_id"`
	ComponentID   uint      `gorm:"index" json:"component_id,omitempty"`
	ComponentName string    `gorm:"size:255" json:"component_name,omitempty"`
	Path          string    `gorm:"size:512;index:idx_codemap_project_path,priority:2;not null" json:"path"` // relatif terhadap root proyek, slash
	AbsPath       string    `gorm:"size:1024" json:"abs_path"`
	Lang          string    `gorm:"size:32" json:"lang"`
	Kind          string    `gorm:"size:32;index" json:"kind"` // page, component, composable, handler, service, model, ...
	Lines         int       `json:"lines"`
	SizeBytes     int64     `json:"size_bytes"`
	SymbolsJSON   string    `gorm:"type:text" json:"-"`
	ImportsJSON   string    `gorm:"type:text" json:"-"` // path berkas lokal yang dipakai; diakhiri "/" untuk paket/direktori
	RefsJSON      string    `gorm:"type:text" json:"-"` // spesifikasi impor mentah, di-resolve ulang tiap build
	RoutesJSON    string    `gorm:"type:text" json:"-"`
	Summary       string    `gorm:"type:text" json:"summary,omitempty"`    // dari komentar dokumen berkas (heuristik)
	AISummary     string    `gorm:"type:text" json:"ai_summary,omitempty"` // ringkasan tujuan berkas dari LLM (job code_map_enrich)
	Keywords      string    `gorm:"type:text" json:"keywords,omitempty"`   // kata kunci tambahan dari LLM, dipisah koma
	ContentHash   string    `gorm:"size:64" json:"content_hash"`
	ModTime       int64     `json:"mod_time"`
	AIHash        string    `gorm:"size:64" json:"-"` // ContentHash saat AISummary dibuat; beda = ringkasan perlu diperbarui
	UpdatedAt     time.Time `json:"updated_at"`
}
