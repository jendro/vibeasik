package models

// Facet desain: cara desainer grafis + frontend developer membaca sebuah tampilan. Satu referensi digambarkan oleh
// KEPERLUAN (jenis situs/aplikasi), TUJUAN (apa yang ingin dicapai pengunjung/pemilik), NADA (kepribadian visual),
// AUDIENS, GAYA visual, ARKETIPE layout, KEPADATAN informasi, dan SKEMA warna. Agent memakainya untuk mengambil
// spesifikasi yang tepat ("desain e-commerce yang playful" ≠ "dashboard admin yang serius").

// Kunci dimensi facet.
const (
	FacetPurpose   = "purposes"
	FacetGoal      = "goals"
	FacetTone      = "tones"
	FacetAudience  = "audiences"
	FacetStyle     = "styles"
	FacetArchetype = "archetypes"
	FacetDensity   = "density"
	FacetScheme    = "scheme"
)

// FacetDims: urutan dimensi (tampilan & prompt).
var FacetDims = []string{FacetPurpose, FacetGoal, FacetTone, FacetAudience, FacetArchetype, FacetStyle, FacetDensity, FacetScheme}

// Nilai baku tiap dimensi (kunci kecil, huruf latin). Label Indonesia ada di FacetLabels.
var (
	// Keperluan: jenis situs/aplikasi.
	DesignPurposes = []string{"ecommerce", "marketplace", "admin_dashboard", "saas_marketing", "web_app", "docs_devtools",
		"content_media", "portfolio_creative", "corporate", "fintech", "education", "social_community", "booking_travel",
		"health", "food_delivery", "government_ngo"}
	// Tujuan: hasil yang dikejar desain.
	DesignGoals = []string{"convert", "discover", "inform", "trust", "engage", "productivity", "showcase"}
	// Nada: kepribadian visual (maks 3 per referensi).
	DesignTones = []string{"serious", "playful", "friendly", "premium", "bold", "minimal", "techy", "warm", "editorial"}
	// Audiens.
	DesignAudiences = []string{"b2b", "b2c", "developer", "enterprise", "mass_market", "kids_family", "creative_pro", "internal_staff"}
	// Arketipe layout.
	DesignArchetypes = []string{"landing_page", "dashboard_sidebar", "catalog_grid", "docs_sidebar", "feed_timeline", "form_flow", "editorial_article", "app_shell_bottom_nav"}
	// Gaya visual (daftar saran; gaya lain boleh ditambahkan bebas).
	DesignStyles = []string{"dark_mode", "glassmorphism", "bento_grid", "brutalist", "flat", "3d", "gradient", "neumorphism",
		"illustration", "photography_led", "typography_led", "monochrome", "colorful", "rounded", "sharp_corners", "material", "ios_native"}
	DesignDensities = []string{"airy", "balanced", "dense"}
	DesignSchemes   = []string{"light", "dark", "both"}
)

// FacetLabels: label Indonesia untuk UI & prompt.
var FacetLabels = map[string]string{
	"ecommerce": "E-commerce / toko online", "marketplace": "Marketplace", "admin_dashboard": "Aplikasi admin / dashboard",
	"saas_marketing": "Landing SaaS / produk", "web_app": "Aplikasi web", "docs_devtools": "Dokumentasi / developer tools",
	"content_media": "Konten / media / blog", "portfolio_creative": "Portofolio / agensi kreatif", "corporate": "Company profile",
	"fintech": "Fintech / keuangan", "education": "Edukasi", "social_community": "Sosial / komunitas",
	"booking_travel": "Pemesanan / travel", "health": "Kesehatan", "food_delivery": "Makanan / pesan-antar", "government_ngo": "Pemerintah / lembaga",
	"convert": "Konversi (jual/daftar)", "discover": "Jelajah & cari", "inform": "Informasi & edukasi", "trust": "Kepercayaan & kredibilitas",
	"engage": "Keterlibatan & retensi", "productivity": "Produktivitas kerja", "showcase": "Pamer brand/karya",
	"serious": "Serius / formal", "playful": "Playful / ceria", "friendly": "Ramah", "premium": "Premium / elegan", "bold": "Berani / ekspresif",
	"minimal": "Minimalis / tenang", "techy": "Teknis / futuristik", "warm": "Hangat / organik", "editorial": "Editorial",
	"b2b": "B2B", "b2c": "B2C", "developer": "Developer", "enterprise": "Enterprise", "mass_market": "Masyarakat umum",
	"kids_family": "Anak & keluarga", "creative_pro": "Profesional kreatif", "internal_staff": "Staf internal",
	"landing_page": "Landing page", "dashboard_sidebar": "Dashboard + sidebar", "catalog_grid": "Katalog / grid produk", "docs_sidebar": "Docs + sidebar",
	"feed_timeline": "Feed / timeline", "form_flow": "Alur form / wizard", "editorial_article": "Artikel editorial", "app_shell_bottom_nav": "Aplikasi mobile (bottom nav)",
	"airy": "Lega", "balanced": "Seimbang", "dense": "Padat", "light": "Terang", "dark": "Gelap", "both": "Terang & gelap",
}

// DesignFacets: profil desain sebuah referensi. Dimensi yang dipasang kurator (Pinned) tidak ditimpa analisis ulang.
type DesignFacets struct {
	Purposes   []string `json:"purposes,omitempty"`
	Goals      []string `json:"goals,omitempty"`
	Tones      []string `json:"tones,omitempty"`
	Audiences  []string `json:"audiences,omitempty"`
	Archetypes []string `json:"archetypes,omitempty"`
	Styles     []string `json:"styles,omitempty"`
	Density    string   `json:"density,omitempty"`
	Scheme     string   `json:"scheme,omitempty"`
	Pinned     []string `json:"pinned,omitempty"` // dimensi yang ditetapkan kurator
	Source     string   `json:"source,omitempty"` // llm | heuristic | manual (asal dominan)
}

// Empty: tak ada satu pun dimensi terisi.
func (f *DesignFacets) Empty() bool {
	return f == nil || (len(f.Purposes)+len(f.Goals)+len(f.Tones)+len(f.Audiences)+len(f.Archetypes)+len(f.Styles) == 0 && f.Density == "" && f.Scheme == "")
}

// Values mengembalikan nilai sebuah dimensi sebagai daftar (density/scheme = satu nilai).
func (f *DesignFacets) Values(dim string) []string {
	if f == nil {
		return nil
	}
	switch dim {
	case FacetPurpose:
		return f.Purposes
	case FacetGoal:
		return f.Goals
	case FacetTone:
		return f.Tones
	case FacetAudience:
		return f.Audiences
	case FacetArchetype:
		return f.Archetypes
	case FacetStyle:
		return f.Styles
	case FacetDensity:
		if f.Density != "" {
			return []string{f.Density}
		}
	case FacetScheme:
		if f.Scheme != "" {
			return []string{f.Scheme}
		}
	}
	return nil
}

// SetValues mengisi sebuah dimensi.
func (f *DesignFacets) SetValues(dim string, vals []string) {
	switch dim {
	case FacetPurpose:
		f.Purposes = vals
	case FacetGoal:
		f.Goals = vals
	case FacetTone:
		f.Tones = vals
	case FacetAudience:
		f.Audiences = vals
	case FacetArchetype:
		f.Archetypes = vals
	case FacetStyle:
		f.Styles = vals
	case FacetDensity:
		f.Density = ""
		if len(vals) > 0 {
			f.Density = vals[0]
		}
	case FacetScheme:
		f.Scheme = ""
		if len(vals) > 0 {
			f.Scheme = vals[0]
		}
	}
}

// IsPinned: dimensi ini ditetapkan kurator.
func (f *DesignFacets) IsPinned(dim string) bool {
	if f == nil {
		return false
	}
	for _, p := range f.Pinned {
		if p == dim {
			return true
		}
	}
	return false
}

// AppDesignTags: tag desain satu aplikasi FE/mobile proyek — kosakata facet SAMA dengan referensi Galeri Desain plus
// platform. Disimpan di Component.DesignTags hanya bagian yang dikunci pengguna/agent (Facets.Pinned,
// PlatformPinned); selebihnya diturunkan heuristik saat dibaca (designlib.AppProfile).
type AppDesignTags struct {
	Platform       string       `json:"platform,omitempty"`
	PlatformPinned bool         `json:"platform_pinned,omitempty"`
	Facets         DesignFacets `json:"facets"`
}
