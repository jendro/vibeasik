package codemap

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"unicode"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Layer arsitektur yang dipakai panduan penempatan logika (urut dari luar ke dalam).
const (
	LayerHandler    = "handler"    // adapter HTTP/controller: parse & validasi input dasar, panggil service, bentuk response
	LayerRoute      = "route"      // registrasi rute
	LayerMiddleware = "middleware" // auth, rate limit, CORS
	LayerService    = "service"    // aturan bisnis & orkestrasi use case (termasuk paket domain/usecase)
	LayerRepository = "repository" // query, transaksi, akses data
	LayerModel      = "model"      // entitas & struktur data
	LayerUtil       = "util"       // helper lintas fitur
)

var layerOrder = []string{LayerHandler, LayerRoute, LayerMiddleware, LayerService, LayerRepository, LayerModel, LayerUtil}

var layerRole = map[string]string{
	LayerHandler:    "adapter HTTP tipis: parse & validasi input dasar, panggil service, bentuk response",
	LayerRoute:      "registrasi rute & grup middleware",
	LayerMiddleware: "auth, rate limit, CORS",
	LayerService:    "aturan bisnis, orkestrasi use case, transaksi lintas repository",
	LayerRepository: "query & akses data (SQL/ORM), tanpa aturan bisnis",
	LayerModel:      "entitas & struktur data",
	LayerUtil:       "helper lintas fitur (dipakai bersama, satu tempat)",
}

// LayerDir adalah satu direktori yang memuat berkas layer tertentu.
type LayerDir struct {
	Dir   string   `json:"dir"`
	Files int      `json:"files"`
	Lines int      `json:"lines"`
	Some  []string `json:"some,omitempty"` // contoh nama berkas
}

// InventoryItem adalah satu simbol berkas target beserta rentang barisnya.
type InventoryItem struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	From      int    `json:"from"`
	To        int    `json:"to"`
	Signature string `json:"signature,omitempty"`
}

// SymbolRef menunjuk satu simbol di satu berkas.
type SymbolRef struct {
	Path  string `json:"path"`
	Layer string `json:"layer"`
	Name  string `json:"name"`
	Line  int    `json:"line"`
}

// ReuseFile adalah berkas layer dalam (service/repository/model/util) yang relevan dengan domain berkas target.
type ReuseFile struct {
	Path    string   `json:"path"`
	Layer   string   `json:"layer"`
	Symbols []string `json:"symbols"`
}

// Twin adalah fungsi di berkas target yang namanya sama/setara dengan fungsi di berkas lain (kandidat fungsi kembar).
type Twin struct {
	Name   string      `json:"name"`
	Line   int         `json:"line"`
	Others []SymbolRef `json:"others"`
}

// LayerGuide adalah konteks peta kode untuk perbaikan audit: di mana tiap layer benar-benar tinggal di proyek,
// isi berkas target, fungsi yang sudah ada untuk dipakai ulang, dan kandidat fungsi kembar. Dengan ini LLM
// menaruh logika di folder yang ada (bukan mengarang usecase/ atau application/) dan tidak menulis ulang fungsi.
type LayerGuide struct {
	Target    string                `json:"target"`
	Layer     string                `json:"layer"`
	Lang      string                `json:"lang"`
	Scope     string                `json:"scope"`
	Lines     int                   `json:"lines"`
	Layers    map[string][]LayerDir `json:"layers"`
	Inventory []InventoryItem       `json:"inventory,omitempty"`
	Reuse     []ReuseFile           `json:"reuse,omitempty"`
	Twins     []Twin                `json:"twins,omitempty"`
}

// skipKinds bukan kode aplikasi: tidak masuk peta layer maupun pencarian fungsi kembar.
var skipKinds = map[string]bool{"test": true, "doc": true, "migration": true, "config": true, "script": true, "style": true}

// LayerOf memetakan kind peta kode ke layer arsitektur. Paket domain/usecase dihitung layer service; helper di
// folder handler tetap util; berkas *_service/*_repository di folder handler dihitung sesuai namanya.
func LayerOf(e models.CodeMapEntry) string {
	p := "/" + strings.ToLower(e.Path)
	base := path.Base(p)
	stem := strings.TrimSuffix(base, path.Ext(base))
	switch {
	case skipKinds[e.Kind]:
		return ""
	case strings.Contains(stem, "repositor") || strings.HasSuffix(stem, "_repo") || strings.HasSuffix(stem, ".repo"):
		return LayerRepository
	case strings.Contains(stem, "service") || strings.Contains(stem, "usecase"):
		return LayerService
	case strings.Contains(stem, "helper") || strings.Contains(stem, "util"):
		return LayerUtil
	}
	switch e.Kind {
	case LayerHandler, LayerRoute, LayerMiddleware, LayerService, LayerRepository, LayerModel, LayerUtil:
		return e.Kind
	case "module":
		if containsAny(p, []string{"/domain/", "/domains/", "/usecase", "/application/", "/core/", "/business/", "/logic/", "/services/"}) {
			return LayerService
		}
	}
	return ""
}

// genericNameWords bukan penanda domain: kata ini muncul di nama berkas/fungsi semua fitur.
var genericNameWords = map[string]bool{
	"admin": true, "handler": true, "handlers": true, "controller": true, "controllers": true, "api": true, "http": true,
	"list": true, "write": true, "read": true, "actions": true, "action": true, "dto": true, "manual": true, "detail": true,
	"mutations": true, "stats": true, "export": true, "copy": true, "type": true, "types": true, "delete": true, "move": true,
	"create": true, "update": true, "get": true, "set": true, "new": true, "routes": true, "route": true, "router": true,
	"main": true, "base": true, "common": true, "service": true, "repository": true, "repo": true, "model": true,
	"models": true, "v1": true, "v2": true, "handle": true, "request": true, "response": true, "public": true,
	"by": true, "id": true, "ids": true, "all": true, "to": true, "from": true, "of": true, "find": true, "save": true,
	"register": true, "serve": true, "json": true, "table": true, "name": true, "string": true, "error": true, "init": true,
}

// nameParts memecah nama snake_case/camelCase/kebab menjadi kata huruf kecil (AdminListWebinars → admin list webinars).
func nameParts(name string) []string {
	var out []string
	rs := []rune(name)
	start := -1
	flush := func(end int) {
		if start >= 0 && end > start {
			out = append(out, strings.ToLower(string(rs[start:end])))
		}
		start = -1
	}
	for i, r := range rs {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush(i)
			continue
		}
		if start < 0 {
			start = i
			continue
		}
		prev := rs[i-1]
		nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
		if (unicode.IsLower(prev) && unicode.IsUpper(r)) || (unicode.IsUpper(prev) && unicode.IsUpper(r) && nextLower) {
			flush(i)
			start = i
		}
	}
	flush(len(rs))
	return out
}

// domainWords: kata penanda domain dari nama berkas (admin_webinar_export.go → webinar).
func domainWords(file string) []string {
	base := path.Base(file)
	var out []string
	for _, w := range nameParts(strings.TrimSuffix(base, path.Ext(base))) {
		if !genericNameWords[w] && len(w) >= 3 {
			out = append(out, w)
		}
	}
	return uniqueStrings(out)
}

// twinPrefixes/twinSuffixes dibuang sebelum membandingkan nama: AdminListWebinars ≈ ListWebinars ≈ handleListWebinars.
var twinPrefixes = []string{"handle", "admin", "do", "api"}
var twinSuffixes = []string{"handler", "func", "impl", "fn", "ctrl"}

// twinKey menormalkan nama fungsi jadi kumpulan kata terurut. Nama yang seluruhnya kata generik (GetByID,
// RegisterRoutes, TableName) atau terlalu pendek diabaikan karena wajar ada di banyak berkas.
func twinKey(name string) string {
	words := nameParts(name)
	for len(words) > 1 && containsWord(twinPrefixes, words[0]) {
		words = words[1:]
	}
	for len(words) > 1 && containsWord(twinSuffixes, words[len(words)-1]) {
		words = words[:len(words)-1]
	}
	specific := false
	for _, w := range words {
		if !genericNameWords[w] && len(w) >= 3 {
			specific = true
		}
	}
	words = uniqueStrings(words)
	sort.Strings(words)
	key := strings.Join(words, " ")
	if !specific || (len(words) < 2 && len(key) < 8) {
		return ""
	}
	return key
}

func containsWord(list []string, w string) bool {
	for _, x := range list {
		if x == w {
			return true
		}
	}
	return false
}

func isCallable(kind string) bool { return kind == "func" || kind == "method" }

// BuildLayerGuide menyusun LayerGuide untuk berkas target (path relatif root proyek, relatif komponen, atau absolut).
// withInventory menambahkan daftar fungsi berkas beserta signature (untuk temuan God Controller / berkas besar).
// Mengembalikan nil bila peta kode belum ada atau berkas tidak dikenal.
func BuildLayerGuide(db *gorm.DB, projectID uint, file string, withInventory bool) *LayerGuide {
	if db == nil || projectID == 0 || strings.TrimSpace(file) == "" {
		return nil
	}
	if err := EnsureFresh(db, projectID, StaleAfter); err != nil {
		return nil
	}
	ix, err := loadIndex(db, projectID)
	if err != nil {
		return nil
	}
	target := lookupDoc(ix, file)
	if target == nil {
		return nil
	}
	g := &LayerGuide{Target: target.entry.Path, Layer: LayerOf(target.entry), Lang: target.entry.Lang, Lines: target.entry.Lines, Layers: map[string][]LayerDir{}}
	inScope := scopeFilter(target)
	g.Scope = target.entry.ComponentName
	if g.Scope == "" {
		g.Scope = topDir(target.entry.Path)
	}

	// 1. Peta layer: direktori nyata tiap layer di komponen & bahasa yang sama.
	dirs := map[string]map[string]*LayerDir{}
	var scoped []*doc
	for _, d := range ix.docs {
		if !inScope(d) || skipKinds[d.entry.Kind] {
			continue
		}
		scoped = append(scoped, d)
		layer := LayerOf(d.entry)
		if layer == "" {
			continue
		}
		dir := path.Dir(d.entry.Path) + "/"
		if dirs[layer] == nil {
			dirs[layer] = map[string]*LayerDir{}
		}
		ld := dirs[layer][dir]
		if ld == nil {
			ld = &LayerDir{Dir: dir}
			dirs[layer][dir] = ld
		}
		ld.Files++
		ld.Lines += d.entry.Lines
		if len(ld.Some) < 3 {
			ld.Some = append(ld.Some, path.Base(d.entry.Path))
		}
	}
	for layer, m := range dirs {
		list := make([]LayerDir, 0, len(m))
		for _, ld := range m {
			list = append(list, *ld)
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].Files != list[j].Files {
				return list[i].Files > list[j].Files
			}
			return list[i].Dir < list[j].Dir
		})
		if len(list) > 5 {
			list = list[:5]
		}
		g.Layers[layer] = list
	}

	// 2. Inventaris berkas target: rentang baris tiap simbol + signature dari disk.
	if withInventory {
		g.Inventory = inventoryOf(target)
	}

	// 3. Fungsi yang sudah ada di layer dalam untuk domain yang sama (pakai ulang, jangan tulis ulang).
	words := domainWords(target.entry.Path)
	for _, d := range scoped {
		if d == target {
			continue
		}
		layer := LayerOf(d.entry)
		if layer != LayerService && layer != LayerRepository && layer != LayerModel && layer != LayerUtil {
			continue
		}
		related := layer == LayerUtil
		for _, w := range words {
			if strings.Contains(strings.ToLower(d.entry.Path), w) {
				related = true
				break
			}
		}
		var names []string
		for _, s := range d.symbols {
			if !isCallable(s.Kind) && s.Kind != "type" && s.Kind != "interface" && s.Kind != "class" {
				continue
			}
			if !related {
				for _, w := range words {
					if strings.Contains(strings.ToLower(s.Name), w) {
						names = append(names, s.Name)
						break
					}
				}
				continue
			}
			names = append(names, s.Name)
		}
		if len(names) == 0 {
			continue
		}
		if len(names) > 20 {
			names = append(names[:20], fmt.Sprintf("…(+%d)", len(names)-20))
		}
		g.Reuse = append(g.Reuse, ReuseFile{Path: d.entry.Path, Layer: layer, Symbols: names})
	}
	sort.SliceStable(g.Reuse, func(i, j int) bool { return layerRank(g.Reuse[i].Layer) < layerRank(g.Reuse[j].Layer) })
	if len(g.Reuse) > 14 {
		g.Reuse = g.Reuse[:14]
	}

	// 4. Kandidat fungsi kembar: fungsi berkas target yang nama ternormalisasinya sama dengan fungsi berkas lain.
	byKey := map[string][]SymbolRef{}
	for _, d := range scoped {
		if d == target {
			continue
		}
		for _, s := range d.symbols {
			if !isCallable(s.Kind) {
				continue
			}
			if k := twinKey(s.Name); k != "" {
				byKey[k] = append(byKey[k], SymbolRef{Path: d.entry.Path, Layer: LayerOf(d.entry), Name: s.Name, Line: s.Line})
			}
		}
	}
	for _, s := range target.symbols {
		if !isCallable(s.Kind) {
			continue
		}
		others := byKey[twinKey(s.Name)]
		if twinKey(s.Name) == "" || len(others) == 0 {
			continue
		}
		if len(others) > 4 {
			others = others[:4]
		}
		g.Twins = append(g.Twins, Twin{Name: s.Name, Line: s.Line, Others: others})
		if len(g.Twins) >= 15 {
			break
		}
	}
	return g
}

func layerRank(layer string) int {
	for i, l := range layerOrder {
		if l == layer {
			return i
		}
	}
	return len(layerOrder)
}

// lookupDoc mencari berkas berdasarkan path relatif root, path absolut, atau akhiran path (terpendek bila ganda).
func lookupDoc(ix *index, file string) *doc {
	file = strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(file), "\\", "/"), "./")
	if d := ix.byPath[file]; d != nil {
		return d
	}
	var best *doc
	for _, d := range ix.docs {
		if d.entry.AbsPath == file || strings.HasSuffix(d.entry.Path, "/"+file) || strings.HasSuffix(file, "/"+d.entry.Path) {
			if best == nil || len(d.entry.Path) < len(best.entry.Path) {
				best = d
			}
		}
	}
	return best
}

func topDir(p string) string {
	if i := strings.Index(p, "/"); i > 0 {
		return p[:i]
	}
	return ""
}

// scopeFilter membatasi peta ke komponen (atau direktori teratas) dan bahasa yang sama dengan berkas target,
// agar backend Go tidak disarankan memakai composable frontend atau kode legacy.
func scopeFilter(target *doc) func(*doc) bool {
	comp, top, lang := target.entry.ComponentName, topDir(target.entry.Path), target.entry.Lang
	return func(d *doc) bool {
		if d.entry.Lang != lang {
			return false
		}
		if comp != "" {
			return d.entry.ComponentName == comp
		}
		return topDir(d.entry.Path) == top
	}
}

func inventoryOf(d *doc) []InventoryItem {
	syms := make([]models.CodeSymbol, 0, len(d.symbols))
	for _, s := range d.symbols {
		if isCallable(s.Kind) || s.Kind == "type" || s.Kind == "interface" || s.Kind == "class" || s.Kind == "const" {
			syms = append(syms, s)
		}
	}
	sort.SliceStable(syms, func(i, j int) bool { return syms[i].Line < syms[j].Line })
	var lines []string
	if data, err := os.ReadFile(d.entry.AbsPath); err == nil {
		lines = strings.Split(string(data), "\n")
	}
	total := d.entry.Lines
	if len(lines) > 0 {
		total = len(lines)
	}
	out := make([]InventoryItem, 0, len(syms))
	for i, s := range syms {
		to := total
		if i+1 < len(syms) && syms[i+1].Line > s.Line {
			to = syms[i+1].Line - 1
		}
		it := InventoryItem{Name: s.Name, Kind: s.Kind, From: s.Line, To: to}
		if s.Line >= 1 && s.Line <= len(lines) {
			it.Signature = truncate(strings.TrimSpace(lines[s.Line-1]), 160)
		}
		out = append(out, it)
		if len(out) >= 80 {
			break
		}
	}
	return out
}

// HasDir melaporkan apakah direktori (relatif root proyek, diakhiri "/") ada di peta layer mana pun.
func (g *LayerGuide) HasDir(dir string) bool {
	dir = strings.TrimSuffix(strings.TrimPrefix(dir, "./"), "/") + "/"
	for _, list := range g.Layers {
		for _, ld := range list {
			if ld.Dir == dir {
				return true
			}
		}
	}
	return false
}

// MainDir mengembalikan direktori utama sebuah layer (paling banyak berkas), "" bila layer belum ada.
func (g *LayerGuide) MainDir(layer string) string {
	if list := g.Layers[layer]; len(list) > 0 {
		return list[0].Dir
	}
	return ""
}

// Prompt merender panduan menjadi blok teks ringkas untuk prompt LLM.
func (g *LayerGuide) Prompt() string {
	if g == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\nPeta Layer Proyek (dari peta kode vibeasik, komponen %q, bahasa %s; path relatif root proyek). Berkas target: %s (layer %s, %d baris).\n",
		g.Scope, g.Lang, g.Target, orDash(g.Layer), g.Lines)
	for _, layer := range layerOrder {
		list := g.Layers[layer]
		if len(list) == 0 {
			if layer == LayerService || layer == LayerRepository {
				fmt.Fprintf(&b, "- %s (%s): BELUM ADA di proyek ini.\n", layer, layerRole[layer])
			}
			continue
		}
		parts := make([]string, 0, len(list))
		for _, ld := range list {
			parts = append(parts, fmt.Sprintf("%s (%d berkas, mis. %s)", ld.Dir, ld.Files, strings.Join(ld.Some, ", ")))
		}
		fmt.Fprintf(&b, "- %s (%s): %s\n", layer, layerRole[layer], strings.Join(parts, "; "))
	}
	if len(g.Inventory) > 0 {
		b.WriteString("\nInventaris Fungsi Berkas Target (nama, baris, signature) — rencana pemindahan hanya boleh memakai nama di daftar ini:\n")
		for _, it := range g.Inventory {
			fmt.Fprintf(&b, "- %s %s L%d-%d: %s\n", it.Kind, it.Name, it.From, it.To, it.Signature)
		}
	}
	if len(g.Reuse) > 0 {
		b.WriteString("\nFungsi/tipe yang SUDAH ADA di layer dalam untuk domain ini (pakai ulang; jangan tulis ulang):\n")
		for _, r := range g.Reuse {
			fmt.Fprintf(&b, "- [%s] %s: %s\n", r.Layer, r.Path, strings.Join(r.Symbols, ", "))
		}
	}
	if len(g.Twins) > 0 {
		b.WriteString("\nKandidat Fungsi Kembar (nama setara di berkas lain — periksa apakah tugasnya sama; bila ya, satukan ke satu tempat lalu panggil ulang):\n")
		for _, t := range g.Twins {
			refs := make([]string, 0, len(t.Others))
			for _, o := range t.Others {
				refs = append(refs, fmt.Sprintf("%s (%s:%d, layer %s)", o.Name, o.Path, o.Line, orDash(o.Layer)))
			}
			fmt.Fprintf(&b, "- %s (L%d) ≈ %s\n", t.Name, t.Line, strings.Join(refs, "; "))
		}
	}
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// TwinKey adalah kunci nama fungsi ternormalisasi untuk pencarian fungsi kembar lintas paket ("" = terlalu umum).
func TwinKey(name string) string { return twinKey(name) }
