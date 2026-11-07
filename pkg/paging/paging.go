// Package paging berisi patokan "berapa baris per load": ukuran halaman tidak tetap, melainkan diturunkan dari
// BOBOT BARIS (byte JSON hasil serialisasi satu baris di list) supaya muatan per permintaan tetap kecil dan cepat.
// Dipakai oleh MCP `vibeasik_paging_plan`, prompt agent (AgentGuidance), dan boleh dipanggil handler vibeasik
// sendiri (ClampLimit). Aturan lengkap: AI_CONTEXT.md Aturan 37; patokan audit: guidelines NET-007..NET-009, FE-014, MOB-007.
//
// Dasar angka (heuristik rekayasa, bukan hukum):
//   - Serialisasi JSON linier terhadap byte (±10 ns/byte decode, ±2 ns/byte encode di Go): 100 KB < 1 ms, 1 MB ≈ 9 ms
//     — jadi yang mahal adalah transfer jaringan dan render DOM, bukan JSON itu sendiri.
//   - Target 100 KB mentah per halaman (≈ 15–25 KB setelah gzip; <0,2 dtk di 4G lambat), batas keras 512 KB.
//   - DOM: Lighthouse memperingatkan > 800 node dan menggagalkan > 1400 node; baris di layar × node per baris ≤ 800.
//   - RAIL: respons < 100 ms terasa instan, < 1 dtk terasa mengalir; p95 API list ≤ 300 ms.
//   - OFFSET membaca & membuang N baris: biaya O(offset). Di atas ±1000 baris dalam / tabel ≥ 10 rb baris → keyset.
package paging

import (
	"fmt"
	"strings"
)

// Anggaran dan batas (byte).
const (
	TargetPayload      = 100 * 1024 // target muatan JSON mentah satu halaman (web)
	MobilePayload      = 50 * 1024  // jaringan seluler / aplikasi mobile
	HardPayload        = 512 * 1024 // batas keras satu respons list, berapa pun yang diminta klien
	MaxPageSize        = 200        // batas atas jumlah baris per permintaan
	DOMNodeBudget      = 800        // simpul DOM yang dirender sekaligus (ambang peringatan Lighthouse)
	DefaultNodesPerRow = 20         // perkiraan simpul DOM satu baris/kartu bila tidak diketahui
	HeavyColumnBytes   = 1024       // kolom yang rata-rata > 1 KB = berat, jangan ikut di list
	KeysetTableRows    = 10_000     // tabel sebesar ini ke atas → keyset
	KeysetDeepOffset   = 1_000      // offset sedalam ini → keyset
	ExactCountMaxRows  = 100_000    // di atas ini jangan COUNT(*) per request
	minRowBytes        = 64
)

// nicePageSizes: ukuran halaman "bulat" yang dipakai (dibulatkan ke bawah).
var nicePageSizes = []int{1, 2, 3, 5, 10, 15, 20, 25, 50, 100, 150, 200}

// Class: kelas bobot baris untuk bahasa percakapan & tabel patokan.
type Class struct {
	Name     string `json:"name"`
	MaxBytes int    `json:"max_bytes"` // batas atas bobot baris (JSON, byte); 0 = tak terbatas
	Example  string `json:"example"`
}

var Classes = []Class{
	{"XS", 500, "id + nama + status"},
	{"S", 1024, "baris tabel biasa (10–15 kolom pendek)"},
	{"M", 2048, "kartu dengan deskripsi singkat"},
	{"L", 4096, "kartu kaya (ringkasan, tag, relasi ringkas)"},
	{"XL", 8192, "baris dengan teks panjang / JSON bersarang"},
	{"XXL", 16384, "baris dengan blob teks/JSON besar"},
	{"HUGE", 0, "baris berisi longtext/JSON/gambar base64 — list WAJIB proyeksi tipis"},
}

// ClassOf mengembalikan kelas bobot baris.
func ClassOf(rowBytes int) Class {
	for _, c := range Classes {
		if c.MaxBytes == 0 || rowBytes <= c.MaxBytes {
			return c
		}
	}
	return Classes[len(Classes)-1]
}

// Options mengatur anggaran; nol = bawaan.
type Options struct {
	Mobile      bool
	NodesPerRow int
}

func (o Options) payload() int {
	if o.Mobile {
		return MobilePayload
	}
	return TargetPayload
}

// PageSize: ukuran halaman yang disarankan = anggaran muatan ÷ bobot baris, dibulatkan ke bawah ke angka bulat,
// dibatasi 1..MaxPageSize (mobile ≤ 100). Baris sangat besar menghasilkan 1–3 — sinyal bahwa list harus memakai
// proyeksi tipis, bukan terus memperkecil halaman.
func PageSize(rowBytes int, o Options) int {
	if rowBytes < minRowBytes {
		rowBytes = minRowBytes
	}
	base := o.payload() / rowBytes
	max := MaxPageSize
	if o.Mobile {
		max = 100
	}
	if base > max {
		base = max
	}
	return floorNice(base)
}

func floorNice(n int) int {
	best := nicePageSizes[0]
	for _, v := range nicePageSizes {
		if v <= n {
			best = v
		}
	}
	return best
}

// ClampLimit membatasi `limit` dari klien di sisi server: default bila kosong, tak pernah melewati MaxPageSize
// maupun HardPayload ÷ bobot baris. rowBytes ≤ 0 → hanya batas jumlah baris.
func ClampLimit(requested, rowBytes int, o Options) int {
	def := MaxPageSize
	if rowBytes > 0 {
		def = PageSize(rowBytes, o)
	}
	if requested <= 0 {
		return def
	}
	hard := MaxPageSize
	if rowBytes > 0 {
		if byBytes := HardPayload / max(rowBytes, minRowBytes); byBytes < hard {
			hard = byBytes
		}
	}
	if hard < 1 {
		hard = 1
	}
	if requested > hard {
		return hard
	}
	return requested
}

// Column mendeskripsikan satu kolom list untuk memperkirakan bobot baris.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`           // int, bool, float, uuid, timestamp/date, string/varchar, text/json/blob/longtext, enum
	Size int    `json:"size,omitempty"` // panjang maksimum untuk varchar/string
	Avg  int    `json:"avg,omitempty"`  // rata-rata byte nilai bila diketahui (menimpa tebakan)
}

// EstimateRow memperkirakan bobot satu baris JSON (nama kunci + nilai + tanda baca) dan kolom yang berat.
func EstimateRow(cols []Column) (bytes int, heavy []string) {
	bytes = 2 // { }
	for i, c := range cols {
		v := c.Avg
		if v <= 0 {
			v = typicalValueBytes(c)
		}
		bytes += len(c.Name) + 4 + v // "nama":nilai,
		if i == len(cols)-1 {
			bytes--
		}
		if v > HeavyColumnBytes {
			heavy = append(heavy, c.Name)
		}
	}
	return bytes, heavy
}

func typicalValueBytes(c Column) int {
	t := strings.ToLower(c.Type)
	switch {
	case strings.Contains(t, "bool"):
		return 5
	case strings.Contains(t, "uuid"):
		return 38
	case strings.Contains(t, "time") || strings.Contains(t, "date"):
		return 27
	case strings.Contains(t, "int") || strings.Contains(t, "serial"):
		return 8
	case strings.Contains(t, "float") || strings.Contains(t, "decimal") || strings.Contains(t, "numeric") || strings.Contains(t, "double"):
		return 10
	case strings.Contains(t, "enum"):
		return 12
	case strings.Contains(t, "blob") || strings.Contains(t, "longtext") || strings.Contains(t, "json") || strings.Contains(t, "base64"):
		return 4096
	case strings.Contains(t, "text"):
		return 1200
	}
	if c.Size > 0 { // varchar(n)/string: rata-rata ±60% dari panjang maksimum, dibatasi 2 KB
		return min(max(c.Size*6/10, 8), 2048)
	}
	return 40
}

// Input adalah pertanyaan perencanaan paging untuk satu list.
type Input struct {
	RowBytes    int      `json:"row_bytes,omitempty"`     // bobot baris terukur/diperkirakan (JSON)
	Columns     []Column `json:"columns,omitempty"`       // atau daftar kolom untuk diperkirakan
	TableRows   int      `json:"table_rows,omitempty"`    // jumlah baris tabel (0 = tak diketahui)
	Mobile      bool     `json:"mobile,omitempty"`        // klien seluler / aplikasi mobile
	NodesPerRow int      `json:"nodes_per_row,omitempty"` // simpul DOM per baris di UI
	DeepPaging  bool     `json:"deep_paging,omitempty"`   // pengguna perlu loncat jauh (halaman ke-50+)
}

// Plan: keputusan paging lengkap.
type Plan struct {
	RowBytes      int      `json:"row_bytes"`
	Class         string   `json:"class"`
	PageSize      int      `json:"page_size"`     // default per permintaan
	MaxPageSize   int      `json:"max_page_size"` // batas server (clamp)
	PagePayload   int      `json:"page_payload_bytes"`
	PayloadBudget int      `json:"payload_budget_bytes"`
	Strategy      string   `json:"strategy"` // offset | keyset
	Count         string   `json:"count"`    // exact | has_more
	UIMode        string   `json:"ui_mode"`  // pages | load_more | virtual_list
	UIRows        int      `json:"ui_rows_in_dom"`
	SlimList      bool     `json:"slim_list"`
	HeavyColumns  []string `json:"heavy_columns,omitempty"`
	Notes         []string `json:"notes"`
}

// Recommend menurunkan rencana paging dari bobot baris, ukuran tabel, dan konteks UI.
func Recommend(in Input) Plan {
	o := Options{Mobile: in.Mobile, NodesPerRow: in.NodesPerRow}
	row, heavy := in.RowBytes, []string(nil)
	if row <= 0 && len(in.Columns) > 0 {
		row, heavy = EstimateRow(in.Columns)
	}
	var notes []string
	if row <= 0 {
		row = 1024
		notes = append(notes, "Bobot baris tidak diketahui — dipakai 1 KB. Ukur sungguhan: serialisasi 20 baris contoh dan ambil p90 (bukan rata-rata).")
	}
	page := PageSize(row, o)
	maxPage := ClampLimit(MaxPageSize, row, o)
	if page > maxPage {
		page = maxPage
	}
	p := Plan{
		RowBytes: row, Class: ClassOf(row).Name, PageSize: page, MaxPageSize: maxPage,
		PagePayload: page * row, PayloadBudget: o.payload(), Strategy: "offset", Count: "exact", HeavyColumns: heavy,
	}
	if row > HeavyColumnBytes*4 || len(heavy) > 0 {
		p.SlimList = true
		if len(heavy) > 0 {
			notes = append(notes, fmt.Sprintf("Kolom berat (>%d B): %s — keluarkan dari list, ambil lewat endpoint detail atau tautan.", HeavyColumnBytes, strings.Join(heavy, ", ")))
		} else {
			notes = append(notes, "Baris >4 KB: list memakai proyeksi tipis (hanya kolom yang tampil) dan detail diambil saat dibuka.")
		}
	}
	if row > o.payload()/3 {
		notes = append(notes, fmt.Sprintf("Satu halaman hanya muat %d baris — itu tanda proyeksi terlalu gemuk; tipiskan dulu sebelum memperkecil halaman.", page))
	}
	if in.TableRows >= KeysetTableRows || in.DeepPaging {
		p.Strategy = "keyset"
		notes = append(notes, "Keyset/cursor: WHERE (sort_key, id) > (:last_sort, :last_id) ORDER BY sort_key, id LIMIT n+1 — waktu tetap berapa pun kedalamannya; wajib indeks pada (sort_key, id).")
	} else {
		notes = append(notes, fmt.Sprintf("OFFSET masih wajar untuk tabel kecil; pindah ke keyset bila offset >%d atau tabel ≥ %d baris. Selalu ORDER BY kolom unik/stabil (tambahkan id).", KeysetDeepOffset, KeysetTableRows))
	}
	if in.TableRows > ExactCountMaxRows || p.Strategy == "keyset" {
		p.Count = "has_more"
		notes = append(notes, "Jangan COUNT(*) tiap request: ambil limit+1 baris untuk has_more/next_cursor; total hanya perkiraan/di-cache bila UI benar-benar butuh.")
	}
	nodes := in.NodesPerRow
	if nodes <= 0 {
		nodes = DefaultNodesPerRow
	}
	p.UIRows = DOMNodeBudget / nodes
	switch {
	case page <= p.UIRows && p.Strategy == "offset" && in.TableRows > 0 && in.TableRows <= page*20:
		p.UIMode = "pages"
	case page <= p.UIRows:
		p.UIMode = "load_more"
	default:
		p.UIMode = "virtual_list"
		notes = append(notes, fmt.Sprintf("%d baris × %d simpul > %d simpul DOM: render per jendela (virtual scroll) atau turunkan jumlah baris di layar ke ≤ %d.", page, nodes, DOMNodeBudget, p.UIRows))
	}
	notes = append(notes, fmt.Sprintf("Respons list p95 ≤ 300 ms; muatan >%d KB per respons dipotong server (ClampLimit). Hitung bobot baris dari JSON sebenarnya (termasuk nama kunci).", HardPayload/1024))
	p.Notes = notes
	return p
}

// Table menghasilkan tabel Markdown bobot baris → ukuran halaman (untuk dokumentasi & prompt).
func Table() string {
	var b strings.Builder
	b.WriteString("| Bobot baris (JSON) | Kelas | Web (100 KB) | Mobile (50 KB) |\n| :--- | :--- | :--- | :--- |\n")
	prev := 0
	for _, c := range Classes {
		rb := c.MaxBytes
		label := fmt.Sprintf("%s–%s", human(prev), human(c.MaxBytes))
		if c.MaxBytes == 0 {
			rb, label = 32768, "> 16 KB"
		}
		fmt.Fprintf(&b, "| %s | %s | %d baris | %d baris |\n", label, c.Name, PageSize(rb, Options{}), PageSize(rb, Options{Mobile: true}))
		prev = c.MaxBytes
	}
	return b.String()
}

func human(n int) string {
	if n >= 1024 {
		return fmt.Sprintf("%d KB", n/1024)
	}
	return fmt.Sprintf("%d B", n)
}

// Markdown merangkum Plan untuk dibaca agent.
func (p Plan) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "### Rencana paging\n- Bobot baris ≈ %d B (kelas %s) → **%d baris per permintaan** (≈ %.0f KB; anggaran %d KB), batas server %d baris.\n",
		p.RowBytes, p.Class, p.PageSize, float64(p.PagePayload)/1024, p.PayloadBudget/1024, p.MaxPageSize)
	fmt.Fprintf(&b, "- Strategi: **%s**, total: **%s**, UI: **%s** (maks %d baris di DOM).\n", p.Strategy, p.Count, p.UIMode, p.UIRows)
	if p.SlimList {
		b.WriteString("- List WAJIB memakai proyeksi tipis (select kolom yang tampil saja).\n")
	}
	for _, n := range p.Notes {
		b.WriteString("- " + n + "\n")
	}
	return b.String()
}
