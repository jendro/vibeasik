package knowledge

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Representasi vektor: TF-IDF jarang (sparse) yang dinormalisasi L2, dihitung murni di Go tanpa model embedding
// atau layanan luar (Aturan 2: zero CGO). Kata dipecah termasuk camelCase/snake_case, plus jembatan istilah
// Indonesia ↔ Inggris agar "validasi" menemukan "validation". Indeks disimpan di memori dan dibangun ulang
// hanya bila teks node berubah (jumlah node / updated_at terakhir berbeda) — penghitung helpful/uses diubah
// lewat UpdateColumn sehingga tidak memicu pembangunan ulang.

type vec map[string]float64

func (a vec) dot(b vec) float64 {
	if len(a) > len(b) {
		a, b = b, a
	}
	s := 0.0
	for t, w := range a {
		s += w * b[t]
	}
	return s
}

var stopWords = map[string]bool{
	"dan": true, "yang": true, "di": true, "ke": true, "dari": true, "untuk": true, "pada": true, "dengan": true,
	"adalah": true, "ini": true, "itu": true, "agar": true, "bisa": true, "wajib": true, "selalu": true, "jangan": true,
	"atau": true, "bila": true, "saat": true, "oleh": true, "akan": true, "juga": true, "tidak": true, "harus": true,
	"lebih": true, "dalam": true, "sebagai": true, "secara": true, "setiap": true, "semua": true, "sudah": true,
	"belum": true, "karena": true, "tanpa": true, "hanya": true, "tetap": true, "lalu": true, "jika": true, "ada": true,
	"perlu": true, "baru": true, "apa": true, "bagaimana": true, "kapan": true, "mana": true, "sangat": true, "masih": true,
	"pakai": true, "memakai": true, "gunakan": true, "buat": true, "membuat": true, "dapat": true, "kita": true, "kami": true,
	"anda": true, "kamu": true, "sebuah": true, "suatu": true, "hal": true, "cara": true, "tersebut": true, "namun": true,
	"tetapi": true, "serta": true, "sehingga": true, "supaya": true, "tiap": true, "per": true, "nya": true, "bukan": true,
	"the": true, "and": true, "in": true, "on": true, "at": true, "to": true, "for": true, "of": true, "with": true,
	"a": true, "an": true, "is": true, "are": true, "be": true, "or": true, "by": true, "as": true, "it": true,
	"this": true, "that": true, "not": true, "use": true, "using": true, "from": true,
}

// bridges: istilah Indonesia ↔ Inggris yang sering bertukar di patokan, pelajaran, dan todo.
var bridges = map[string]string{
	"keamanan": "security", "validasi": "validation", "kueri": "query", "uji": "test", "pengujian": "test",
	"galat": "error", "kesalahan": "error", "rute": "route", "berkas": "file", "tampilan": "ui", "antarmuka": "ui",
	"performa": "performance", "kinerja": "performance", "autentikasi": "authentication", "otorisasi": "authorization",
	"rahasia": "secret", "kredensial": "credential", "penyimpanan": "storage", "migrasi": "migration", "indeks": "index",
	"basis": "database", "pengguna": "user", "masuk": "login", "hapus": "delete", "simpan": "save", "ubah": "update",
	"komponen": "component", "halaman": "page", "tombol": "button", "proyek": "project", "pengaturan": "settings",
	"pencatatan": "logging", "catatan": "log", "lambat": "slow", "bocor": "leak", "kebocoran": "leak",
	"perencanaan": "planning", "rencana": "plan", "dokumentasi": "docs", "penanganan": "handling", "antrean": "queue",
	// desain (facet Galeri Desain): kata Indonesia → istilah baku facet agar recall menemukan spesifikasi per jenis
	"desain": "design", "warna": "color", "tipografi": "typography", "ilustrasi": "illustration", "toko": "ecommerce", "belanja": "ecommerce",
	"serius": "serious", "formal": "serious", "ceria": "playful", "ramah": "friendly", "mewah": "premium", "elegan": "premium",
	"minimalis": "minimal", "padat": "dense", "lega": "airy", "gelap": "dark", "terang": "light", "korporat": "corporate",
	"pendidikan": "education", "edukasi": "education", "kesehatan": "health", "keuangan": "fintech", "portofolio": "portfolio",
	"berita": "news", "kuliner": "food", "anak": "kids", "komunitas": "community", "pemerintah": "government",
	// teori desain & warna (designlib/theory.json)
	"teori": "theory", "harmoni": "harmony", "komplementer": "complementary", "analog": "analogous", "monokromatik": "monochromatic",
	"triadik": "triadic", "tetradik": "tetradic", "kontras": "contrast", "palet": "palette", "hangat": "warm", "dingin": "cool",
	"hierarki": "hierarchy", "kedekatan": "proximity", "kesamaan": "similarity", "keseimbangan": "balance", "perataan": "alignment",
	"saturasi": "saturation", "kejenuhan": "saturation", "rona": "hue",
}

// tokenize: kata utuh + potongan camelCase/snake_case, huruf kecil, tanpa stopword & angka murni.
func tokenize(s string) []string {
	var out []string
	add := func(w string) {
		w = strings.ToLower(w)
		if len([]rune(w)) < 2 || stopWords[w] || isDigits(w) {
			return
		}
		out = append(out, w)
		if b, ok := bridges[w]; ok {
			out = append(out, b)
		}
	}
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		add(w)
		rs := []rune(w)
		start, split := 0, false
		for i := 1; i < len(rs); i++ {
			if unicode.IsLower(rs[i-1]) && unicode.IsUpper(rs[i]) {
				add(string(rs[start:i]))
				start, split = i, true
			}
		}
		if split {
			add(string(rs[start:]))
		}
	}
	return out
}

func isDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// termFreq: judul & domain/tag berbobot 2, isi berbobot 1.
func termFreq(title, content, domain string, tags []string) map[string]float64 {
	tf := map[string]float64{}
	for _, t := range tokenize(title) {
		tf[t] += 2
	}
	for _, t := range tokenize(domain + " " + strings.Join(tags, " ")) {
		tf[t] += 2
	}
	for _, t := range tokenize(content) {
		tf[t]++
	}
	return tf
}

// entry adalah satu node aktif di indeks.
type entry struct {
	id     uint
	owner  uint // 0 = pustaka bersama
	kind   string
	domain string
	source string
	key    string
	v      vec
}

type index struct {
	sig      string
	n        int
	df       map[string]int
	entries  []*entry
	byID     map[uint]*entry
	domainN  map[string]int // jumlah node aktif per domain (deteksi celah)
	postings map[string][]int
}

var (
	idxMu  sync.Mutex
	cached *index
)

// signature: berubah bila ada node bertambah/berkurang atau teks/status node diperbarui.
func signature(db *gorm.DB) string {
	var row struct {
		N int64
		M string
	}
	db.Raw("SELECT COUNT(*) AS n, COALESCE(MAX(updated_at), '') AS m FROM knowledge_nodes").Scan(&row)
	return strconv.FormatInt(row.N, 10) + "|" + row.M
}

// current mengembalikan indeks terbaru (dibangun ulang bila node berubah).
func current(db *gorm.DB) *index {
	sig := signature(db)
	idxMu.Lock()
	defer idxMu.Unlock()
	if cached != nil && cached.sig == sig {
		return cached
	}
	cached = build(db, sig)
	return cached
}

// invalidate memaksa indeks dibangun ulang pada pemakaian berikutnya (dipakai setelah tulis dalam proses yang sama
// ketika updated_at bisa sama persis dengan tulis sebelumnya).
func invalidate() {
	idxMu.Lock()
	cached = nil
	idxMu.Unlock()
}

func build(db *gorm.DB, sig string) *index {
	var rows []models.KnowledgeNode
	db.Select("id", "source_key", "kind", "domain", "tags", "title", "content", "source", "owner_employee_id").
		Where("status = ?", models.KnowledgeNodeActive).Order("id").Find(&rows)
	ix := &index{sig: sig, n: len(rows), df: map[string]int{}, byID: map[uint]*entry{}, domainN: map[string]int{}, postings: map[string][]int{}}
	tfs := make([]map[string]float64, len(rows))
	for i, r := range rows {
		tfs[i] = termFreq(r.Title, r.Content, r.Domain, r.Tags)
		for t := range tfs[i] {
			ix.df[t]++
		}
	}
	for i, r := range rows {
		e := &entry{id: r.ID, kind: r.Kind, domain: r.Domain, source: r.Source, key: r.Key, v: ix.weigh(tfs[i], false)}
		if r.OwnerEmployeeID != nil {
			e.owner = *r.OwnerEmployeeID
		}
		ix.entries = append(ix.entries, e)
		ix.byID[r.ID] = e
		if r.Domain != "" {
			ix.domainN[r.Domain]++
		}
		for t := range e.v {
			ix.postings[t] = append(ix.postings[t], i)
		}
	}
	return ix
}

func (ix *index) idf(t string) float64 {
	return math.Log(float64(ix.n+1)/(float64(ix.df[t])+0.5)) + 1
}

// weigh mengubah frekuensi kata menjadi vektor TF-IDF ternormalisasi. Untuk kueri, kata yang tidak ada di korpus
// dibuang (tidak mungkin cocok); untuk node baru (keepUnseen) kata itu justru pembeda yang harus ikut dihitung.
func (ix *index) weigh(tf map[string]float64, keepUnseen bool) vec {
	v := vec{}
	norm := 0.0
	for t, f := range tf {
		if ix.df[t] == 0 && !keepUnseen {
			continue
		}
		w := (1 + math.Log(f)) * ix.idf(t)
		v[t] = w
		norm += w * w
	}
	if norm == 0 {
		return v
	}
	norm = math.Sqrt(norm)
	for t := range v {
		v[t] /= norm
	}
	return v
}

// queryVector: vektor kueri memakai IDF korpus saat ini.
func (ix *index) queryVector(title, content, domain string, tags []string) vec {
	return ix.weigh(termFreq(title, content, domain, tags), false)
}

// noteVector: vektor butir baru (untuk cek duplikat & tautan), kata barunya ikut dihitung.
func (ix *index) noteVector(title, content, domain string, tags []string) vec {
	return ix.weigh(termFreq(title, content, domain, tags), true)
}

type neighbor struct {
	e   *entry
	sim float64
}

// nearest: k node paling mirip dengan v (kosinus ≥ min), lewat posting list agar tidak membandingkan semua pasangan.
func (ix *index) nearest(v vec, k int, min float64, skip func(*entry) bool) []neighbor {
	acc := map[int]float64{}
	for t, w := range v {
		ps := ix.postings[t]
		if len(ps)*2 > ix.n && ix.n > 20 {
			continue // kata yang ada di lebih dari separuh node tidak membedakan
		}
		for _, i := range ps {
			acc[i] += w * ix.entries[i].v[t]
		}
	}
	out := make([]neighbor, 0, len(acc))
	for i, s := range acc {
		if s >= min && (skip == nil || !skip(ix.entries[i])) {
			out = append(out, neighbor{ix.entries[i], s})
		}
	}
	sort.Slice(out, func(a, b int) bool {
		return out[a].sim > out[b].sim || out[a].sim == out[b].sim && out[a].e.id < out[b].e.id
	})
	if k > 0 && len(out) > k {
		out = out[:k]
	}
	return out
}

// TextDoc / TextIndex: indeks TF-IDF generik dengan tokenizer & pembobotan yang sama seperti Pusat Pengetahuan,
// untuk korpus lain yang perlu pencarian "yang mirip" tanpa LLM (mis. todo serupa per proyek, internal/todorag).
type TextDoc struct {
	ID      uint
	Title   string
	Content string
	Tags    string
}

type TextIndex struct{ ix *index }

type TextHit struct {
	ID  uint
	Sim float64
}

// NewTextIndex membangun indeks dari dokumen (judul & tag berbobot 2, isi berbobot 1).
func NewTextIndex(docs []TextDoc) *TextIndex {
	ix := &index{n: len(docs), df: map[string]int{}, byID: map[uint]*entry{}, domainN: map[string]int{}, postings: map[string][]int{}}
	tfs := make([]map[string]float64, len(docs))
	for i, d := range docs {
		tfs[i] = termFreq(d.Title, d.Content, d.Tags, nil)
		for t := range tfs[i] {
			ix.df[t]++
		}
	}
	for i, d := range docs {
		e := &entry{id: d.ID, v: ix.weigh(tfs[i], false)}
		ix.entries = append(ix.entries, e)
		ix.byID[d.ID] = e
		for t := range e.v {
			ix.postings[t] = append(ix.postings[t], i)
		}
	}
	return &TextIndex{ix: ix}
}

// Len: jumlah dokumen di indeks.
func (t *TextIndex) Len() int { return t.ix.n }

// Similar: k dokumen paling mirip dengan kueri (kosinus ≥ min), skip menyaring id yang tidak diinginkan.
func (t *TextIndex) Similar(title, content string, k int, min float64, skip func(id uint) bool) []TextHit {
	var sk func(*entry) bool
	if skip != nil {
		sk = func(e *entry) bool { return skip(e.id) }
	}
	ns := t.ix.nearest(t.ix.queryVector(title, content, "", nil), k, min, sk)
	out := make([]TextHit, len(ns))
	for i, n := range ns {
		out[i] = TextHit{ID: n.e.id, Sim: n.sim}
	}
	return out
}
