package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Galeri Desain: database referensi tampilan website yang bagus. Setiap referensi menyimpan screenshot
// desktop/mobile, design token hasil ekstraksi CSS/DOM, dan analisis AI (gaya, palet, tipografi, pola layout,
// prompt siap pakai) yang dirangkum menjadi Design Brief untuk agent coding saat memodernkan tampilan proyek.

const (
	DesignStatusNew      = "new"      // baru ditambahkan, belum dipotret
	DesignStatusCaptured = "captured" // screenshot & token ada, belum dianalisis
	DesignStatusAnalyzed = "analyzed"
	DesignStatusFailed   = "failed"

	DesignSourceSeed   = "seed"   // kurasi bawaan vibeasik
	DesignSourceManual = "manual" // ditambahkan pengguna dari Studio
	DesignSourceAgent  = "agent"  // ditambahkan agent AI lewat MCP

	DesignAnalysisVision    = "llm_vision" // LLM membaca screenshot + token
	DesignAnalysisText      = "llm_text"   // LLM tanpa gambar (model tidak mendukung vision)
	DesignAnalysisHeuristic = "heuristic"  // tanpa LLM: diturunkan dari token & struktur halaman

	DesignPlatformWeb     = "web"     // dipotret otomatis dari URL
	DesignPlatformAndroid = "android" // screenshot diunggah pengguna
	DesignPlatformIOS     = "ios"
)

// DesignPlatforms adalah platform referensi yang dikenali.
var DesignPlatforms = []string{DesignPlatformWeb, DesignPlatformAndroid, DesignPlatformIOS}

// DesignAreas adalah area prinsip desain yang diekstrak dari screenshot (urutan tampil di UI & brief).
var DesignAreas = []string{
	"spacing", "layout", "navigation", "typography", "color", "buttons", "forms",
	"content", "iconography", "imagery", "feedback", "accessibility", "responsive", "motion",
}

// DesignCategories adalah kategori yang dikenali UI/MCP (bebas diisi lain, ini hanya saran).
var DesignCategories = []string{
	"saas", "devtools", "ecommerce", "marketplace", "fintech", "dashboard", "portfolio",
	"agency", "product", "docs", "education", "media", "company", "other",
}

type DesignReference struct {
	ID       uint     `gorm:"primaryKey" json:"id"`
	Name     string   `gorm:"size:200" json:"name"`
	Platform string   `gorm:"size:20;default:web" json:"platform"`
	URL      string   `gorm:"size:1000" json:"url"` // tanpa index: utf8mb4 1000 char > batas kunci MySQL 3072 byte
	Category string   `gorm:"size:60;index" json:"category"`
	Tags     []string `gorm:"-" json:"tags"`
	TagsJSON string   `gorm:"type:text" json:"-"`
	Notes    string   `gorm:"type:text" json:"notes"`
	Source   string   `gorm:"size:20" json:"source"`
	Rating   int      `json:"rating"` // 0-5 dari pengguna
	Favorite bool     `json:"favorite"`

	Status    string `gorm:"size:20;index" json:"status"`
	LastError string `gorm:"type:text" json:"last_error,omitempty"`

	DesktopShot     string       `gorm:"size:255" json:"desktop_shot,omitempty"` // nama berkas di DataDir
	MobileShot      string       `gorm:"size:255" json:"mobile_shot,omitempty"`
	Uploads         []DesignShot `gorm:"-" json:"uploads"` // screenshot unggahan (aplikasi Android/iOS, halaman di balik login)
	UploadsJSON     string       `gorm:"type:text" json:"-"`
	CapturedAt      *time.Time   `json:"captured_at,omitempty"`
	PageTitle       string       `gorm:"size:500" json:"page_title,omitempty"`
	PageDescription string       `gorm:"type:text" json:"page_description,omitempty"`

	Tokens        *DesignTokens   `gorm:"-" json:"tokens,omitempty"`
	TokensJSON    string          `gorm:"type:text" json:"-"`
	Structure     *PageStructure  `gorm:"-" json:"structure,omitempty"`
	StructureJSON string          `gorm:"type:text" json:"-"`
	Analysis      *DesignAnalysis `gorm:"-" json:"analysis,omitempty"`
	AnalysisJSON  string          `gorm:"type:text" json:"-"`

	Facets     *DesignFacets `gorm:"-" json:"facets,omitempty"` // profil desain efektif (kurator > LLM > heuristik)
	FacetsJSON string        `gorm:"type:text" json:"-"`

	AnalysisSource string     `gorm:"size:20" json:"analysis_source,omitempty"`
	AnalysisModel  string     `gorm:"size:200" json:"analysis_model,omitempty"`
	AnalyzedAt     *time.Time `json:"analyzed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DesignColor adalah satu warna yang dipakai halaman (hex #rrggbb) beserta frekuensi & perkiraan perannya.
type DesignColor struct {
	Hex     string `json:"hex"`
	Count   int    `json:"count,omitempty"`
	Role    string `json:"role,omitempty"` // primary, accent, background, surface, text, border, neutral
	Name    string `json:"name,omitempty"` // nama CSS variable bila berasal dari token
	Neutral bool   `json:"neutral,omitempty"`
}

type DesignValueCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// DesignTokens adalah token desain yang terukur dari CSS & DOM (bukan tebakan LLM).
type DesignTokens struct {
	Colors       []DesignColor      `json:"colors"`
	CSSVariables []DesignColor      `json:"css_variables,omitempty"` // custom property berwarna, mis. --color-primary
	Fonts        []DesignValueCount `json:"fonts"`
	HeadingFont  string             `json:"heading_font,omitempty"` // font yang dipakai aturan h1/h2
	GoogleFonts  []string           `json:"google_fonts,omitempty"`
	FontSizes    []DesignValueCount `json:"font_sizes,omitempty"`
	Radii        []DesignValueCount `json:"radii,omitempty"`
	Shadows      []DesignValueCount `json:"shadows,omitempty"`
	Breakpoints  []int              `json:"breakpoints,omitempty"`
	Frameworks   []string           `json:"frameworks,omitempty"`
	HasDarkMode  bool               `json:"has_dark_mode"`
	UsesGradient bool               `json:"uses_gradient"`
	UsesBlur     bool               `json:"uses_blur"` // backdrop-filter (glassmorphism)
	UsesGrid     bool               `json:"uses_grid"`
	UsesAnim     bool               `json:"uses_animation"`
	ThemeColor   string             `json:"theme_color,omitempty"`
	CSSBytes     int                `json:"css_bytes"`
	Stylesheets  int                `json:"stylesheets"`
}

// PageStructure adalah kerangka konten halaman dari DOM hasil render.
type PageStructure struct {
	Headings   []string       `json:"headings"`  // h1-h2 berurutan
	NavLinks   []string       `json:"nav_links"` // label menu navigasi
	CTAs       []string       `json:"ctas,omitempty"`
	Counts     map[string]int `json:"counts"` // section, img, svg, button, form, video, ...
	TextLength int            `json:"text_length"`
	Language   string         `json:"language,omitempty"`
}

// DesignShot adalah satu screenshot unggahan pengguna.
type DesignShot struct {
	File  string `json:"file"`
	Label string `json:"label,omitempty"`
	MIME  string `json:"mime"`
}

// DesignPrinciple adalah satu best practice yang terlihat di screenshot dan bisa diterapkan ke aplikasi lain.
type DesignPrinciple struct {
	Area       string `json:"area"`               // salah satu DesignAreas
	Rule       string `json:"rule"`               // prinsip yang bisa ditiru, mis. "Satu CTA utama per layar"
	Evidence   string `json:"evidence,omitempty"` // apa yang terlihat di screenshot
	HowToApply string `json:"how_to_apply,omitempty"`
	Priority   string `json:"priority,omitempty"` // must, should, nice
}

type DesignNavigation struct {
	Type          string `json:"type,omitempty"`      // top bar, sidebar, bottom navigation, tab bar, drawer, ...
	Placement     string `json:"placement,omitempty"` // posisi & perataan
	Shape         string `json:"shape,omitempty"`     // bentuk item: pill, underline, ikon+label, ...
	Items         string `json:"items,omitempty"`     // jumlah & pengelompokan menu
	ActiveState   string `json:"active_state,omitempty"`
	Secondary     string `json:"secondary,omitempty"` // aksi sekunder: profil, pencarian, CTA di nav
	MobilePattern string `json:"mobile_pattern,omitempty"`
}

type DesignSpacing struct {
	BaseUnit         string `json:"base_unit,omitempty"` // mis. 4px / 8dp
	Scale            string `json:"scale,omitempty"`
	SectionPadding   string `json:"section_padding,omitempty"`
	ComponentPadding string `json:"component_padding,omitempty"`
	Gaps             string `json:"gaps,omitempty"`
	Density          string `json:"density,omitempty"`
}

type DesignForms struct {
	LabelPlacement  string `json:"label_placement,omitempty"`
	InputStyle      string `json:"input_style,omitempty"`
	InputHeight     string `json:"input_height,omitempty"`
	Grouping        string `json:"grouping,omitempty"`
	ActionPlacement string `json:"action_placement,omitempty"` // letak tombol simpan/kirim
	Validation      string `json:"validation,omitempty"`
}

type DesignButtons struct {
	Primary   string `json:"primary,omitempty"`
	Secondary string `json:"secondary,omitempty"`
	Shape     string `json:"shape,omitempty"`
	Sizing    string `json:"sizing,omitempty"`
	Placement string `json:"placement,omitempty"`
}

type DesignColorUsage struct {
	Distribution string `json:"distribution,omitempty"` // mis. 60-30-10 netral/permukaan/aksen
	AccentUsage  string `json:"accent_usage,omitempty"`
	Semantic     string `json:"semantic,omitempty"` // sukses/peringatan/error
	Contrast     string `json:"contrast,omitempty"`
}

type DesignComponentNote struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DesignTypography struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
	Mono    string `json:"mono,omitempty"`
	Scale   string `json:"scale,omitempty"`
	Notes   string `json:"notes,omitempty"`
}

type DesignLayout struct {
	Grid     string   `json:"grid,omitempty"`
	Spacing  string   `json:"spacing,omitempty"`
	Density  string   `json:"density,omitempty"` // airy, balanced, dense
	Patterns []string `json:"patterns,omitempty"`
}

// DesignAnalysis adalah pengamatan desain oleh AI (atau heuristik bila tidak ada LLM): deskripsi apa yang terlihat,
// tanpa skor atau penilaian baik/buruk.
type DesignAnalysis struct {
	Summary       string                `json:"summary"`
	StyleKeywords []string              `json:"style_keywords"`
	Mood          string                `json:"mood,omitempty"`
	Palette       []DesignColor         `json:"palette"`
	Typography    DesignTypography      `json:"typography"`
	Layout        DesignLayout          `json:"layout"`
	Sections      []string              `json:"sections,omitempty"`
	Components    []DesignComponentNote `json:"components,omitempty"`
	Effects       []string              `json:"effects,omitempty"`
	ReuseTips     []string              `json:"reuse_tips,omitempty"`
	TailwindHint  string                `json:"tailwind_hint,omitempty"`
	AgentPrompt   string                `json:"agent_prompt,omitempty"`

	// Best practice terstruktur dari pembacaan screenshot.
	Navigation *DesignNavigation `json:"navigation,omitempty"`
	Spacing    *DesignSpacing    `json:"spacing,omitempty"`
	Forms      *DesignForms      `json:"forms,omitempty"`
	Buttons    *DesignButtons    `json:"buttons,omitempty"`
	ColorUsage *DesignColorUsage `json:"color_usage,omitempty"`
	Principles []DesignPrinciple `json:"principles,omitempty"`

	// Facets: pembacaan LLM atas keperluan/tujuan/nada/audiens/arketipe (dinormalisasi ke kosakata baku).
	Facets *DesignFacets `json:"facets,omitempty"`
}

func (d *DesignReference) BeforeSave(*gorm.DB) error {
	if d.Platform == "" {
		d.Platform = DesignPlatformWeb
	}
	d.TagsJSON = marshalOrEmpty(d.Tags)
	if d.Uploads == nil {
		d.Uploads = []DesignShot{}
	}
	if b, err := json.Marshal(d.Uploads); err == nil {
		d.UploadsJSON = string(b)
	}
	d.TokensJSON = marshalOrEmpty(d.Tokens)
	d.StructureJSON = marshalOrEmpty(d.Structure)
	d.AnalysisJSON = marshalOrEmpty(d.Analysis)
	d.FacetsJSON = ""
	if !d.Facets.Empty() {
		if b, err := json.Marshal(d.Facets); err == nil {
			d.FacetsJSON = string(b)
		}
	}
	return nil
}

func (d *DesignReference) AfterFind(*gorm.DB) error {
	d.Tags = []string{}
	_ = json.Unmarshal([]byte(d.TagsJSON), &d.Tags)
	d.Uploads = []DesignShot{}
	_ = json.Unmarshal([]byte(d.UploadsJSON), &d.Uploads)
	if d.Platform == "" {
		d.Platform = DesignPlatformWeb
	}
	if d.TokensJSON != "" {
		d.Tokens = &DesignTokens{}
		_ = json.Unmarshal([]byte(d.TokensJSON), d.Tokens)
	}
	if d.StructureJSON != "" {
		d.Structure = &PageStructure{}
		_ = json.Unmarshal([]byte(d.StructureJSON), d.Structure)
	}
	if d.AnalysisJSON != "" {
		d.Analysis = &DesignAnalysis{}
		_ = json.Unmarshal([]byte(d.AnalysisJSON), d.Analysis)
	}
	if d.FacetsJSON != "" {
		d.Facets = &DesignFacets{}
		_ = json.Unmarshal([]byte(d.FacetsJSON), d.Facets)
	}
	return nil
}

func marshalOrEmpty(v interface{}) string {
	switch t := v.(type) {
	case []string:
		if len(t) == 0 {
			return "[]"
		}
	case *DesignTokens:
		if t == nil {
			return ""
		}
	case *PageStructure:
		if t == nil {
			return ""
		}
	case *DesignAnalysis:
		if t == nil {
			return ""
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
