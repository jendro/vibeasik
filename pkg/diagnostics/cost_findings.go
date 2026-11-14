package diagnostics

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/routemap"
)

// Temuan biaya proses: rumus tick T(n) & query Q(n) hasil analisis statis (routemap) dilaporkan di FUNGSI
// tempat biayanya berasal, bukan per rute, supaya satu perbaikan menutup semua rute yang terdampak.
//
//	QUERY-NPLUS1-CHAIN      query dikalikan loop data lewat rantai pemanggilan (tidak terlihat per baris)
//	COST-POLYNOMIAL-GROWTH  tick tumbuh O(n²) ke atas
//
// Rumusnya batas atas kasus terburuk (semua koleksi dianggap berukuran n), jadi bobot skornya sengaja ringan
// dan jumlah temuan dibatasi per jenis: yang dilaporkan hanya sumber termahal.
const (
	ruleNPlusOneChain    = "QUERY-NPLUS1-CHAIN"
	rulePolynomialGrowth = "COST-POLYNOMIAL-GROWTH"
	// N+1 lebih pasti dan lebih mahal (I/O) daripada perkiraan pertumbuhan tick, jadi jatahnya lebih besar.
	maxNPlusOneFindings = 12
	maxGrowthFindings   = 8
)

func costFindings(graphs []*routemap.Graph, baseRel string, existing []models.Finding) []models.Finding {
	return hotspotFindings(routemap.Hotspots(graphs), baseRel, existing)
}

// phpCostFindings: sumber biaya rute Laravel/Lumen, CodeIgniter 4, Slim, dan halaman PHP biasa di dir.
func phpCostFindings(dir, baseRel string, existing []models.Finding) []models.Finding {
	return hotspotFindings(routemap.PHPHotspots(dir), baseRel, existing)
}

func hotspotFindings(hots []routemap.Hotspot, baseRel string, existing []models.Finding) []models.Finding {
	var out []models.Finding
	left := map[string]int{"nplus1": maxNPlusOneFindings, "growth": maxGrowthFindings}
	for _, h := range actionableFirst(hots) {
		if left[h.Kind] <= 0 {
			continue
		}
		f := HotspotFinding(h, baseRel)
		// Query yang tertulis langsung di dalam loop fungsi ini sudah dilaporkan aturan per baris.
		if h.Kind == "nplus1" && hasFindingIn(existing, "QUERY-NPLUS1", f.TargetFile, h.Func.Line, h.Func.End) {
			continue
		}
		left[h.Kind]--
		out = append(out, f)
	}
	return out
}

// Aturan audit untuk sumber biaya proses (dipakai juga oleh tab Biaya Proses untuk menautkan todo).
const (
	RuleNPlusOneChain    = ruleNPlusOneChain
	RulePolynomialGrowth = rulePolynomialGrowth
)

// HotspotRule: aturan audit untuk satu jenis sumber biaya.
func HotspotRule(kind string) string {
	if kind == "nplus1" {
		return ruleNPlusOneChain
	}
	return rulePolynomialGrowth
}

// HotspotFinding menyusun temuan audit untuk satu sumber biaya; baseRel = folder aplikasi relatif root proyek.
func HotspotFinding(h routemap.Hotspot, baseRel string) models.Finding {
	file := h.Func.File
	if baseRel != "" {
		file = filepath.ToSlash(filepath.Join(baseRel, file))
	}
	routes := strings.Join(h.Routes, ", ")
	if h.RouteCount > len(h.Routes) {
		routes += fmt.Sprintf(" (+%d rute lain)", h.RouteCount-len(h.Routes))
	}
	f := models.Finding{TargetFile: file, Line: h.Func.Line, RuleID: HotspotRule(h.Kind)}
	// Bukti: baris yang menaikkan biaya (loop atas data, query di dalam loop, pemanggilan berbiaya), diambil dari
	// penghitung yang sama dengan rumusnya. Baris terpenting menjadi potongan kode temuan.
	f.Snippet = hotspotEvidence(h)
	if h.Kind == "nplus1" {
		via := "di dalam loop atas data"
		if h.Via != "" {
			via = fmt.Sprintf("lewat pemanggilan `%s` di dalam loop atas data", h.Via)
		}
		f.Category, f.Severity = "query", "WARNING"
		f.Title = "N+1 Query Lewat Rantai Pemanggilan"
		f.Description = fmt.Sprintf("Fungsi `%s` menjalankan query %s: jumlah query Q(n) = %s (n = jumlah data). Untuk 1.000 data berarti %s query. Rute terdampak: %s.",
			h.Func.Name, via, h.Func.QueryFormula, growthAt(h.Func.Cost, 1000, true), routes)
		f.Suggestion = nplusOneFix(h.Func.Lang) + " Bila loop-nya memang kecil dan tetap (mis. ≤ 5 jenis), tandai temuan ini sebagai bukan masalah."
		if !routemap.OnlyQueries(h.Func.IOKinds) {
			// I/O selain query (HTTP, berkas, email, antrean, goroutine) per item: aturannya sama — jangan per item.
			kinds := strings.Join(routemap.IOLabels(h.Func.IOKinds), ", ")
			f.Title = "I/O per Item di Dalam Loop (" + kinds + ")"
			f.Description = fmt.Sprintf("Fungsi `%s` menjalankan operasi I/O (%s) %s: jumlah operasi Q(n) = %s (n = jumlah data). Untuk 1.000 data berarti %s operasi. Rute terdampak: %s.",
				h.Func.Name, kinds, via, h.Func.QueryFormula, growthAt(h.Func.Cost, 1000, true), routes)
			f.Suggestion = "Keluarkan I/O dari loop: ambil/baca sekali sebelum loop lalu cocokkan di memori (map), kirim secara batch (satu request/insert/publish untuk banyak item), atau batasi konkurensi dengan satu worker/pool, bukan satu goroutine per item. Bila loop-nya kecil dan tetap, tandai temuan ini sebagai bukan masalah."
			f.TheoryBasis = "Biaya I/O (jaringan, disk, SMTP) per operasi jauh di atas biaya perhitungan di memori; satu operasi per item membuat latensi tumbuh linear terhadap jumlah data."
		}
		f.TheoryBasis = "N+1 Selects (Martin Fowler, Patterns of Enterprise Application Architecture): biaya I/O tumbuh linear terhadap jumlah baris."
		f.Description += evidenceSentence(h)
		return f
	}
	// Pangkat 3 atau loop bersarang di badan fungsi sendiri cukup dapat dipercaya untuk jadi peringatan;
	// pangkat lebih tinggi lewat rantai pemanggilan panjang hanyalah petunjuk tempat memeriksa.
	sev := "INFO"
	if h.Func.Degree == 3 || (h.Func.Degree >= 3 && h.Via == "") {
		sev = "WARNING"
	}
	via := "loop bersarang atas data di badan fungsi"
	if h.Via != "" {
		via = fmt.Sprintf("loop atas data yang memanggil `%s` (yang juga mengulang data)", h.Via)
	}
	caveat := "Hitungan ini batas atas kasus terburuk dengan semua koleksi dianggap berukuran n."
	if h.Func.Degree >= 4 && h.Via != "" {
		caveat = "Pangkat setinggi ini muncul karena rantai pemanggilan panjang dan semua koleksi dianggap berukuran n: perlakukan sebagai petunjuk tempat memeriksa, bukan angka pasti."
	}
	f.Category, f.Severity = "code_smell", sev
	f.Title = fmt.Sprintf("Biaya Proses Tumbuh %s", h.Func.BigO)
	f.Description = fmt.Sprintf("Fungsi `%s` memuat %s: jumlah langkah T(n) = %s. Untuk 1 data %d langkah, untuk 100 data sekitar %s langkah. Rute terdampak: %s. %s",
		h.Func.Name, via, leadingTerms(h.Func.Formula, 3), h.Func.Ticks1, growthAt(h.Func.Cost, 100, false), routes, caveat)
	f.Suggestion = "Ganti pencarian di loop dalam dengan map/set yang dibangun sekali (O(n)), urutkan lalu gabung, atau batasi ukuran data lewat paging. Bila salah satu koleksinya selalu kecil dan tetap, tandai temuan ini sebagai bukan masalah."
	f.TheoryBasis = "Analisis kompleksitas waktu (Cormen dkk., Introduction to Algorithms): suku tertinggi menentukan pertumbuhan biaya saat data membesar."
	f.Description += evidenceSentence(h)
	return f
}

// hotspotEvidence memilih baris bukti terpenting sebuah sumber biaya sebagai potongan kode temuan: query
// terdalam untuk N+1, loop terdalam untuk pertumbuhan tick (bila tidak ada, bukti pertama).
func hotspotEvidence(h routemap.Hotspot) string {
	want := "loop"
	if h.Kind == "nplus1" {
		want = "query"
	}
	ev := h.Func.Evidence
	best := -1
	for i, e := range ev {
		if e.Kind == want && (best < 0 || e.Depth > ev[best].Depth) {
			best = i
		}
	}
	if best < 0 && len(ev) > 0 {
		best = 0
	}
	if best < 0 {
		return ""
	}
	return ev[best].Code
}

// evidenceSentence: daftar singkat baris bukti untuk deskripsi temuan ("Bukti: L12 `for …` (loop atas data) …").
func evidenceSentence(h routemap.Hotspot) string {
	if len(h.Func.Evidence) == 0 {
		return ""
	}
	var parts []string
	for _, e := range h.Func.Evidence {
		if len(parts) == 4 {
			parts = append(parts, fmt.Sprintf("… (+%d baris lain)", len(h.Func.Evidence)-4))
			break
		}
		code := e.Code
		if len(code) > 90 {
			code = code[:90] + "…"
		}
		parts = append(parts, fmt.Sprintf("L%d `%s` (%s)", e.Line, code, e.Note))
	}
	return " Bukti di " + h.Func.File + ": " + strings.Join(parts, "; ") + "."
}

// growthAt: nilai T(n) atau Q(n) pada ukuran n dari tabel pertumbuhan, diformat ringkas.
func growthAt(c routemap.Cost, n int64, queries bool) string {
	for _, g := range c.Growth {
		if g.N == n {
			v := g.Ticks
			if queries {
				v = g.Queries
			}
			return compactNumber(v)
		}
	}
	return "?"
}

func compactNumber(v float64) string {
	switch {
	case v >= 1e15:
		return "lebih dari 1.000 triliun"
	case v >= 1e12:
		return fmt.Sprintf("%.1f triliun", v/1e12)
	case v >= 1e9:
		return fmt.Sprintf("%.1f miliar", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.1f juta", v/1e6)
	case v >= 1e4:
		return fmt.Sprintf("%.0f ribu", v/1e3)
	}
	return fmt.Sprintf("%.0f", v)
}

func hasFindingIn(fs []models.Finding, ruleID, file string, from, to int) bool {
	for _, f := range fs {
		if f.RuleID == ruleID && filepath.ToSlash(f.TargetFile) == file && f.Line >= from && f.Line <= to {
			return true
		}
	}
	return false
}

// leadingTerms menyisakan suku-suku tertinggi rumus (penentu pertumbuhan) agar deskripsi tetap terbaca.
func leadingTerms(formula string, n int) string {
	parts := strings.Split(formula, " + ")
	if len(parts) <= n+1 {
		return formula
	}
	return strings.Join(parts[:n], " + ") + " + …"
}

// actionableFirst mengurutkan sumber biaya menurut seberapa bisa ditindaklanjuti: N+1 apa adanya (termahal dulu);
// pertumbuhan tick mendahulukan loop bersarang di badan fungsi sendiri, lalu pangkat terendah lewat rantai
// pemanggilan (pangkat tinggi lewat rantai panjang paling tidak pasti). Urutan asal dipertahankan di dalam kelompok.
func actionableFirst(hs []routemap.Hotspot) []routemap.Hotspot {
	group := func(h routemap.Hotspot) int {
		switch {
		case h.Kind != "growth":
			return 0
		case h.Via == "":
			return 1
		}
		return h.Func.Degree
	}
	out := append([]routemap.Hotspot(nil), hs...)
	sort.SliceStable(out, func(i, j int) bool { return group(out[i]) < group(out[j]) })
	return out
}

// nplusOneFix: cara memperbaiki N+1 dalam istilah ekosistemnya.
func nplusOneFix(lang string) string {
	if lang == "php" {
		return "Ambil semua data yang dibutuhkan loop dalam satu query sebelum loop (Eloquent `with()`/`load()`, JOIN, atau `whereIn('id', ...)`), lalu cocokkan di memori lewat array berkunci id."
	}
	return "Ambil semua data yang dibutuhkan loop dalam satu query sebelum loop (Preload, JOIN, atau WHERE id IN (...)), lalu cocokkan di memori lewat map."
}
