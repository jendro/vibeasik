package codectx

import (
	"crypto/sha1"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/codemap"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// Pengecekan fungsi global: satu tugas = satu fungsi di satu tempat. Seluruh fungsi aplikasi dibandingkan
// lintas berkas (per komponen & bahasa): badan identik (setelah normalisasi literal/spasi/komentar) atau nama
// setara dengan badan sangat mirip dilaporkan sebagai fungsi kembar; fungsi privat yang tidak dirujuk di mana
// pun dilaporkan sebagai kode mati.
const (
	RuleDuplicateFunction = "SMELL-DUPLICATE-FUNCTION"
	RuleUnusedFunction    = "SMELL-UNUSED-FUNCTION"

	minTwinBodyLines   = 6    // fungsi pendek (getter/wrapper) wajar mirip
	twinSimilarity     = 0.72 // Jaccard shingle token untuk nama setara
	shapeScore         = 0.99 // penanda kelompok "pola sama beda literal"
	maxGlobalFindings  = 25
	maxUnusedFindings  = 30
	maxGlobalFuncLines = 400
)

type fn struct {
	file   *codemap.FileSymbols
	sym    models.CodeSymbol
	from   int
	to     int
	exact  string // hash badan apa adanya (tanpa spasi/komentar): identik
	shape  string // hash struktur (literal dinormalkan): pola sama, beda nilai
	shingl map[string]bool
	key    string
}

var (
	strLitRe  = regexp.MustCompile("\"(?:\\\\.|[^\"\\\\])*\"|'(?:\\\\.|[^'\\\\])*'|`[^`]*`")
	numLitRe  = regexp.MustCompile(`\b\d+(\.\d+)?\b`)
	lineCmtRe = regexp.MustCompile(`(//|#).*$`)
	tokenRe   = regexp.MustCompile(`[A-Za-z_$][\w$]*|[^\s\w]`)
)

// normalizeBody: token badan fungsi tanpa signature, komentar, dan spasi; literal=true menormalkan literal
// string/angka (untuk mendeteksi pola yang sama dengan nilai berbeda).
func normalizeBody(lines []string, from, to int, literal bool) []string {
	var toks []string
	for n := from + 1; n <= to && n <= len(lines); n++ {
		l := strings.TrimSpace(lines[n-1])
		if l == "" || strings.HasPrefix(l, "*") || strings.HasPrefix(l, "/*") || strings.HasPrefix(l, "//") || strings.HasPrefix(l, "#") {
			continue
		}
		if literal {
			l = strLitRe.ReplaceAllString(l, "S")
			l = lineCmtRe.ReplaceAllString(l, "")
			l = numLitRe.ReplaceAllString(l, "N")
		}
		toks = append(toks, tokenRe.FindAllString(l, -1)...)
	}
	return toks
}

func hashOf(toks []string) string {
	h := sha1.Sum([]byte(strings.Join(toks, " ")))
	return fmt.Sprintf("%x", h[:8])
}

func shingles(toks []string) map[string]bool {
	out := map[string]bool{}
	for i := 0; i+3 <= len(toks); i++ {
		out[strings.Join(toks[i:i+3], " ")] = true
	}
	return out
}

func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if b[k] {
			inter++
		}
	}
	return float64(inter) / float64(len(a)+len(b)-inter)
}

// GlobalFunctionFindings menjalankan pengecekan fungsi global untuk seluruh proyek.
func GlobalFunctionFindings(db *gorm.DB, project *models.Project) []models.Finding {
	b := newBuilder(db, project, DefaultOptions())
	if b == nil {
		return nil
	}
	groups := map[string][]*fn{} // komponen|bahasa → fungsi
	hasComponents := false
	for i := range b.files {
		hasComponents = hasComponents || b.files[i].Component != ""
	}
	for i := range b.files {
		f := &b.files[i]
		// berkas di luar komponen terdaftar (mis. folder legacy/ rujukan) tidak ikut diaudit
		if !f.IsAppCode() || f.Lines > 5000 || len(f.Symbols) == 0 || (hasComponents && f.Component == "") {
			continue
		}
		lines := b.src.lines(f.AbsPath)
		if len(lines) == 0 {
			continue
		}
		for _, s := range f.Symbols {
			if !isCallable(s.Kind) {
				continue
			}
			from, to := b.spanOf(f, s)
			if to-from < minTwinBodyLines || to-from > maxGlobalFuncLines {
				continue
			}
			toks := normalizeBody(lines, from, to, true)
			if len(toks) < 25 {
				continue
			}
			groups[f.Component+"|"+f.Lang] = append(groups[f.Component+"|"+f.Lang], &fn{
				file: f, sym: s, from: from, to: to, exact: hashOf(normalizeBody(lines, from, to, false)), shape: hashOf(toks),
				shingl: shingles(toks), key: codemap.TwinKey(s.Name),
			})
		}
	}
	var out []models.Finding
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, duplicateFindings(groups[k])...)
		if len(out) >= maxGlobalFindings {
			out = out[:maxGlobalFindings]
			break
		}
	}
	return append(out, b.unusedFindings()...)
}

// duplicateFindings mengelompokkan fungsi kembar (union-find) dan membuat satu temuan per kelompok.
func duplicateFindings(fns []*fn) []models.Finding {
	parent := make([]int, len(fns))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	union := func(a, b int) { parent[find(a)] = find(b) }
	sim := map[[2]int]float64{}
	byExact, byShape, byKey := map[string][]int{}, map[string][]int{}, map[string][]int{}
	for i, f := range fns {
		byExact[f.exact] = append(byExact[f.exact], i)
		byShape[f.shape] = append(byShape[f.shape], i)
		if f.key != "" {
			byKey[f.key] = append(byKey[f.key], i)
		}
	}
	link := func(groups map[string][]int, score float64, minLines int) {
		for _, idx := range groups {
			for _, j := range idx[1:] {
				i := idx[0]
				if fns[j].to-fns[j].from < minLines {
					continue
				}
				if _, done := sim[[2]int{i, j}]; !done {
					union(i, j)
					sim[[2]int{i, j}] = score
				}
			}
		}
	}
	link(byExact, 1, 0)
	link(byShape, shapeScore, 8) // pola sama beda literal: hanya fungsi yang cukup panjang
	for _, idx := range byKey {
		if len(idx) > 12 {
			continue // nama generik berulang di banyak berkas (mis. setiap handler punya List)
		}
		for a := 0; a < len(idx); a++ {
			for c := a + 1; c < len(idx); c++ {
				i, j := idx[a], idx[c]
				if fns[i].file == fns[j].file {
					continue
				}
				if s := jaccard(fns[i].shingl, fns[j].shingl); s >= twinSimilarity {
					union(i, j)
					sim[[2]int{i, j}] = s
				}
			}
		}
	}
	members := map[int][]int{}
	for i := range fns {
		members[find(i)] = append(members[find(i)], i)
	}
	var out []models.Finding
	for _, idx := range members {
		if len(idx) < 2 {
			continue
		}
		sort.Slice(idx, func(a, c int) bool {
			return fns[idx[a]].file.Path+fmt.Sprint(fns[idx[a]].from) < fns[idx[c]].file.Path+fmt.Sprint(fns[idx[c]].from)
		})
		best := 0.0
		for pair, s := range sim {
			if find(pair[0]) == find(idx[0]) && s > best {
				best = s
			}
		}
		kind := fmt.Sprintf("kemiripan %d%%", int(best*100+0.5))
		switch best {
		case 1:
			kind = "badan identik"
		case shapeScore:
			kind = "pola sama, beda nilai literal — kandidat diparameterisasi"
		}
		first := fns[idx[0]]
		var locs []string
		for _, i := range idx {
			f := fns[i]
			locs = append(locs, fmt.Sprintf("%s (%s:%d-%d, layer %s)", f.sym.Name, f.file.Path, f.from, f.to, orDash(f.file.Layer)))
		}
		target := innermost(fns, idx)
		out = append(out, models.Finding{
			Category: "code_smell", Severity: "WARNING", RuleID: RuleDuplicateFunction,
			Title:       fmt.Sprintf("Fungsi Kembar: %s di %d tempat (%s)", first.sym.Name, len(idx), kind),
			Description: "Logika yang sama ditulis berulang di beberapa berkas: " + strings.Join(locs, "; ") + ". Perubahan aturan harus diingat di semua salinan sehingga mudah tidak konsisten.",
			Suggestion: fmt.Sprintf("Satukan ke satu fungsi di layer yang tepat (kandidat: %s di %s), lalu panggil ulang dari tempat lain. Pertahankan salinan hanya bila ada alasan teknis nyata (beda lapisan keamanan, hot path, beda kontrak).",
				fns[target].sym.Name, fns[target].file.Path),
			TargetFile: first.file.Path, Line: first.from, Snippet: strings.TrimSpace(first.sym.Name),
			TheoryBasis: "DRY — The Pragmatic Programmer (Hunt & Thomas): every piece of knowledge must have a single, unambiguous representation; Martin Fowler Refactoring: Duplicated Code.",
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TargetFile+fmt.Sprint(out[i].Line) < out[j].TargetFile+fmt.Sprint(out[j].Line)
	})
	return out
}

// innermost: anggota kelompok di layer terdalam (repository → service → util → model) sebagai tempat tinggal tunggal.
func innermost(fns []*fn, idx []int) int {
	rank := map[string]int{codemap.LayerRepository: 0, codemap.LayerService: 1, codemap.LayerUtil: 2, codemap.LayerModel: 3}
	best, bestRank := idx[0], 99
	for _, i := range idx {
		r, ok := rank[fns[i].file.Layer]
		if !ok {
			r = 9
		}
		if r < bestRank {
			best, bestRank = i, r
		}
	}
	return best
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// unusedFindings: fungsi privat (Go huruf kecil non-method, PHP private, Python _nama) yang identifier-nya
// tidak muncul di mana pun selain definisinya — termasuk test & pemakaian sebagai nilai (callback).
func (b *builder) unusedFindings() []models.Finding {
	counts := map[string]map[string]int{} // komponen|bahasa → identifier → jumlah baris
	for i := range b.files {
		f := &b.files[i]
		if !f.IsAppCode() && !f.IsTest() {
			continue
		}
		key := f.Component + "|" + f.Lang
		if counts[key] == nil {
			counts[key] = map[string]int{}
		}
		for _, l := range b.src.lines(f.AbsPath) {
			seen := map[string]bool{}
			for _, id := range identRe.FindAllString(l, -1) {
				if !seen[id] {
					seen[id] = true
					counts[key][id]++
				}
			}
		}
	}
	var out []models.Finding
	for i := range b.files {
		f := &b.files[i]
		if !f.IsAppCode() {
			continue
		}
		lines := b.src.lines(f.AbsPath)
		for _, s := range f.Symbols {
			if !isCallable(s.Kind) || !isPrivate(f, s, lines) || counts[f.Component+"|"+f.Lang][s.Name] > 1 {
				continue
			}
			out = append(out, models.Finding{
				Category: "code_smell", Severity: "INFO", RuleID: RuleUnusedFunction,
				Title:       fmt.Sprintf("Fungsi Tidak Dipakai: %s", s.Name),
				Description: fmt.Sprintf("Fungsi privat '%s' di %s tidak dirujuk di mana pun (kode aplikasi maupun test) dalam komponen ini.", s.Name, path.Base(f.Path)),
				Suggestion:  "Hapus bila memang kode mati, atau sambungkan ke alur yang seharusnya memakainya. Kode mati membuat AI agent & developer salah mengira alur yang aktif.",
				TargetFile:  f.Path, Line: s.Line, Snippet: strings.TrimSpace(lineAt(lines, s.Line)),
				TheoryBasis: "Martin Fowler Refactoring: Remove Dead Code; Robert C. Martin Clean Code: dead functions.",
			})
			if len(out) >= maxUnusedFindings {
				return out
			}
		}
	}
	return out
}

func lineAt(lines []string, n int) string {
	if n >= 1 && n <= len(lines) {
		return lines[n-1]
	}
	return ""
}

// isDocSymbol mengecek apakah simbol atau barisnya ditandai sebagai dokumentasi/contoh.
func isDocSymbol(doc, sig string) bool {
	return strings.Contains(doc, "DOKUMENTASI") || strings.Contains(doc, "CONTOH KODE") ||
		strings.Contains(sig, "DOKUMENTASI") || strings.Contains(sig, "CONTOH KODE")
}

func isGoPrivate(s models.CodeSymbol, sig string) bool {
	return s.Kind == "func" && !strings.HasPrefix(s.Name, "Test") && !strings.HasPrefix(s.Name, "Benchmark") &&
		s.Name[0] >= 'a' && s.Name[0] <= 'z' && !strings.Contains(sig, "//go:")
}

// isPrivate: fungsi yang secara bahasa tidak bisa dipakai dari luar berkas/paket/kelasnya.
func isPrivate(f *codemap.FileSymbols, s models.CodeSymbol, lines []string) bool {
	if frameworkEntry(f, s) {
		return false
	}
	sig := lineAt(lines, s.Line)
	if isDocSymbol(s.Doc, sig) {
		return false
	}
	switch f.Lang {
	case "go":
		return isGoPrivate(s, sig)
	case "php":
		return strings.Contains(sig, "private function")
	case "python":
		return strings.HasPrefix(s.Name, "_") && !strings.HasPrefix(s.Name, "__")
	}
	return false
}
