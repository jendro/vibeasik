package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Galeri Desain bertingkat: Website (DesignReference) → Halaman (DesignPage) → Komponen (MeasuredComponent).
// Halaman menyimpan hasil UKUR browser sungguhan (getBoundingClientRect + getComputedStyle), bukan tebakan
// LLM: layout (container, section, ritme spacing), komponen (posisi, ukuran, margin, padding, style) per
// viewport, dan fakta pengamatan (ObservedFact) yang dihitung dari hasil ukur itu. Galeri tidak menilai website:
// fakta dari banyak website diringkas jadi kesimpulan per jenis website (designlib/conclusions.go).

const (
	DesignPageHome    = "home"
	DesignPagePricing = "pricing"
	DesignPageProduct = "product"
	DesignPageAbout   = "about"
	DesignPageAuth    = "auth"
	DesignPageDocs    = "docs"
	DesignPageBlog    = "blog"
	DesignPageContact = "contact"
	DesignPageOther   = "other"
)

// DesignPageKinds adalah jenis halaman yang dikenali (urutan tampil di UI).
var DesignPageKinds = []string{DesignPageHome, DesignPagePricing, DesignPageProduct, DesignPageAbout, DesignPageAuth, DesignPageDocs, DesignPageBlog, DesignPageContact, DesignPageOther}

type DesignPage struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	ReferenceID uint   `gorm:"index" json:"reference_id"`
	URL         string `gorm:"size:1000" json:"url"`
	Kind        string `gorm:"size:20" json:"kind"`
	Title       string `gorm:"size:500" json:"title,omitempty"`
	Auto        bool   `json:"auto"` // ditemukan otomatis dari navigasi halaman utama

	Status     string     `gorm:"size:20" json:"status"` // new, captured, failed
	LastError  string     `gorm:"type:text" json:"last_error,omitempty"`
	MeasuredAt *time.Time `json:"measured_at,omitempty"`

	DesktopShot string `gorm:"size:255" json:"desktop_shot,omitempty"`
	MobileShot  string `gorm:"size:255" json:"mobile_shot,omitempty"`

	Desktop     *PageMeasure `gorm:"-" json:"desktop,omitempty"`
	DesktopJSON string       `gorm:"type:longtext" json:"-"`
	Mobile      *PageMeasure `gorm:"-" json:"mobile,omitempty"`
	MobileJSON  string       `gorm:"type:longtext" json:"-"`
	// Observations: fakta terukur halaman apa adanya (tanpa target/skor), dihitung dari Desktop/Mobile saat dibaca.
	Observations []ObservedFact `gorm:"-" json:"observations,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Box adalah kotak elemen dalam piksel CSS relatif terhadap atas halaman (bukan viewport).
type Box struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// Sides adalah nilai atas, kanan, bawah, kiri (px) — urutan sama dengan CSS.
type Sides [4]int

// ComponentStyle adalah style hasil getComputedStyle sebuah komponen.
type ComponentStyle struct {
	Font          string  `json:"font,omitempty"`
	FontSize      float64 `json:"font_size,omitempty"`
	FontWeight    int     `json:"font_weight,omitempty"`
	LineHeight    float64 `json:"line_height,omitempty"`
	LetterSpacing float64 `json:"letter_spacing,omitempty"`
	TextTransform string  `json:"text_transform,omitempty"`
	TextAlign     string  `json:"text_align,omitempty"`
	Color         string  `json:"color,omitempty"`
	Background    string  `json:"background,omitempty"` // hex warna latar efektif (latar transparan → leluhur)
	Gradient      bool    `json:"gradient,omitempty"`
	Border        string  `json:"border,omitempty"` // "1px solid #e5e7eb"
	Radius        float64 `json:"radius,omitempty"`
	Shadow        string  `json:"shadow,omitempty"`
	Display       string  `json:"display,omitempty"`
	Contrast      float64 `json:"contrast,omitempty"` // rasio kontras teks vs latar efektif
}

// MeasuredComponent adalah satu komponen yang terukur di halaman. Posisi & ukuran = layout (urusan halaman),
// Style = tampilan komponen itu sendiri.
type MeasuredComponent struct {
	ID        int            `json:"id"`              // nomor urut di halaman (K1, K2, …) — dipakai juga di label LLM
	Role      string         `json:"role"`            // navbar, hero, heading, button, input, card, image, form, footer, section
	Level     int            `json:"level,omitempty"` // heading 1–6
	Label     string         `json:"label,omitempty"`
	Tag       string         `json:"tag"`
	Section   int            `json:"section"` // indeks section induk (-1 = di luar section)
	Box       Box            `json:"box"`
	XPct      float64        `json:"x_pct"` // posisi & lebar relatif lebar viewport (%)
	WPct      float64        `json:"w_pct"`
	Align     string         `json:"align,omitempty"` // left, center, right, full — relatif container
	Margin    Sides          `json:"margin"`
	Padding   Sides          `json:"padding"`
	Gap       int            `json:"gap,omitempty"`      // gap flex/grid
	Children  int            `json:"children,omitempty"` // anak langsung yang terlihat
	Columns   int            `json:"columns,omitempty"`  // jumlah kolom anak (grid/flex baris)
	Fold      bool           `json:"above_fold"`         // terlihat di layar pertama
	Style     ComponentStyle `json:"style"`
	Signature string         `json:"signature"` // kunci varian (peran + style + ukuran dibulatkan)
}

// SectionBox adalah satu blok vertikal halaman (header, hero, section, footer).
type SectionBox struct {
	Index         int    `json:"index"`
	Role          string `json:"role"`
	Label         string `json:"label,omitempty"` // heading pertama di section
	Y             int    `json:"y"`
	H             int    `json:"h"`
	PaddingTop    int    `json:"padding_top"`    // jarak tepi section ke konten pertama
	PaddingBottom int    `json:"padding_bottom"` // jarak konten terakhir ke tepi bawah
	Background    string `json:"background,omitempty"`
	Columns       int    `json:"columns,omitempty"` // kolom konten terlebar
	Components    int    `json:"components"`
}

// PageLayout adalah ringkasan tata letak halaman pada satu viewport.
type PageLayout struct {
	ContainerWidth  int          `json:"container_width"` // lebar konten utama paling umum
	ContainerLeft   int          `json:"container_left"`
	SidePadding     int          `json:"side_padding"` // jarak konten ke tepi layar
	Sections        []SectionBox `json:"sections"`
	HeaderHeight    int          `json:"header_height,omitempty"`
	HeaderSticky    bool         `json:"header_sticky,omitempty"`
	NavItems        int          `json:"nav_items,omitempty"`
	HOverflow       bool         `json:"h_overflow"` // scroll horizontal
	FoldCTA         bool         `json:"fold_cta"`   // tombol aksi terlihat di layar pertama
	FoldH1          bool         `json:"fold_h1"`
	H1Count         int          `json:"h1_count"`
	LargestTextIsH1 bool         `json:"largest_text_is_h1"`
}

// MeasureStats adalah statistik terukur seluruh halaman untuk penilaian patokan.
type MeasureStats struct {
	FontSizes      []DesignValueCount `json:"font_sizes"` // ukuran font teks terlihat (px)
	Spacing        []DesignValueCount `json:"spacing"`    // margin/padding/gap ≠ 0 (px)
	Radii          []DesignValueCount `json:"radii"`
	Fonts          []DesignValueCount `json:"fonts"`
	TextColors     []DesignValueCount `json:"text_colors"`
	BgColors       []DesignValueCount `json:"bg_colors"`
	LeftEdges      []DesignValueCount `json:"left_edges"` // tepi kiri blok teks (px)
	BodySize       float64            `json:"body_size"`  // ukuran font paragraf paling umum
	BodyLineHeight float64            `json:"body_line_height"`
	H1Size         float64            `json:"h1_size"`
	MaxTextSize    float64            `json:"max_text_size"`
	CharsPerLine   float64            `json:"chars_per_line"` // rata-rata karakter per baris paragraf
	TextSamples    int                `json:"text_samples"`
	LowContrast    int                `json:"low_contrast"` // sampel teks di bawah WCAG AA
	LowContrastEx  []string           `json:"low_contrast_examples,omitempty"`
	Interactive    int                `json:"interactive"`   // tombol/input terukur
	SmallTargets   int                `json:"small_targets"` // < 44px (mobile) / < 24px (desktop)
}

// PageMeasure adalah hasil ukur satu halaman pada satu viewport.
type PageMeasure struct {
	ViewportW  int                 `json:"viewport_w"`
	ViewportH  int                 `json:"viewport_h"`
	PageHeight int                 `json:"page_height"`
	Layout     PageLayout          `json:"layout"`
	Components []MeasuredComponent `json:"components"`
	Stats      MeasureStats        `json:"stats"`
}

// ObservedFact adalah satu fakta terukur halaman pada satu viewport — apa adanya, tanpa target, skor, atau lulus/gagal.
type ObservedFact struct {
	Key      string  `json:"key"`
	Area     string  `json:"area"` // area DesignAreas
	Title    string  `json:"title"`
	Viewport string  `json:"viewport"` // desktop, mobile
	Value    string  `json:"value"`    // teks tampil, mis. "1280px"
	Num      float64 `json:"num"`      // angka utama (diringkas lintas website); ya/tidak = 1/0
	Unit     string  `json:"unit,omitempty"`
	Detail   string  `json:"detail,omitempty"`
}

func (p *DesignPage) BeforeSave(*gorm.DB) error {
	p.DesktopJSON = marshalPtr(p.Desktop)
	p.MobileJSON = marshalPtr(p.Mobile)
	return nil
}

func (p *DesignPage) AfterFind(*gorm.DB) error {
	if p.DesktopJSON != "" {
		p.Desktop = &PageMeasure{}
		_ = json.Unmarshal([]byte(p.DesktopJSON), p.Desktop)
	}
	if p.MobileJSON != "" {
		p.Mobile = &PageMeasure{}
		_ = json.Unmarshal([]byte(p.MobileJSON), p.Mobile)
	}
	return nil
}

func marshalPtr[T any](v *T) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
