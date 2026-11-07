package codectx

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/codemap"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/routemap"
)

// FileContext adalah gambaran di sekitar satu berkas yang direview utuh: fungsi di berkas lain yang dipakai
// berkas ini (dengan kodenya), siapa yang memakai fungsi-fungsi berkas ini, test, dan rute.
type FileContext struct {
	Target   string   `json:"target"`
	Precise  bool     `json:"precise"`
	External []Span   `json:"external,omitempty"`
	Users    []string `json:"users,omitempty"` // "Fungsi ← pemanggil (berkas:baris)"
	Unused   []string `json:"unused,omitempty"`
	Tests    []string `json:"tests,omitempty"`
	Routes   []string `json:"routes,omitempty"`
}

// BuildFile menyusun FileContext untuk review LLM berkas utuh.
func BuildFile(db *gorm.DB, project *models.Project, file string) *FileContext {
	opt := DefaultOptions()
	opt.MaxCallees, opt.MaxCalleeLines = 6, 40
	b := newBuilder(db, project, opt)
	if b == nil {
		return nil
	}
	f := b.lookup(file)
	if f == nil || !f.IsAppCode() {
		return nil
	}
	b.target = f
	b.indexDefs()
	fc := &FileContext{Target: f.Path}

	// 1. Fungsi di berkas lain yang dipanggil dari berkas ini, diurutkan dari yang paling sering dipakai.
	type ext struct {
		d     def
		uses  int
		first int
	}
	exts := map[string]*ext{}
	var order []string
	add := func(d def) {
		key := d.file.Path + ":" + d.sym.Name
		if e := exts[key]; e != nil {
			e.uses++
			return
		}
		exts[key] = &ext{d: d, uses: 1, first: len(order)}
		order = append(order, key)
	}
	var graphs []*routemap.Graph
	if f.Lang == "go" {
		graphs = routemap.Analyze(codemap.EffectiveRoot(*project))
	}
	for _, s := range f.Symbols {
		if !isCallable(s.Kind) {
			continue
		}
		from, to := b.spanOf(f, s)
		if g, fr := routemap.FuncAt(graphs, f.Path, s.Line); fr != nil {
			fc.Precise = true
			for _, ce := range g.Callees(fr.Key) {
				if cf := b.lookup(ce.File); cf != nil && cf != f {
					add(def{file: cf, sym: models.CodeSymbol{Name: ce.Name, Kind: "func", Line: ce.Line}})
				}
			}
			continue
		}
		for _, call := range callsIn(b.src.lines(f.AbsPath), from, to) {
			for _, d := range b.resolve(call, f, s.Name) {
				if d.file != f {
					add(d)
				}
			}
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return exts[order[i]].uses > exts[order[j]].uses })
	for _, key := range order {
		if len(fc.External) >= opt.MaxCallees {
			break
		}
		e := exts[key]
		from, to := b.spanOf(e.d.file, e.d.sym)
		s := b.span(e.d.file, e.d.sym.Name, from, to, 0, opt.MaxCalleeLines, 1)
		fc.External = append(fc.External, s)
	}
	sortSpans(fc.External)

	// 2. Pemakai fungsi berkas ini di berkas lain + fungsi yang tidak dirujuk di mana pun (indeks identifier
	// sekali jalan, bukan regex per simbol).
	refs := b.identIndex()
	for _, s := range f.Symbols {
		if !isCallable(s.Kind) || len(s.Name) < 3 {
			continue
		}
		uses, shown := 0, 0
		for _, r := range refs[s.Name] {
			if r.file == f && r.line == s.Line {
				continue
			}
			if r.test {
				if len(fc.Tests) < 6 && !contains(fc.Tests, r.file.Path) {
					fc.Tests = append(fc.Tests, r.file.Path)
				}
				continue
			}
			uses++
			if r.file != f && shown < 2 && len(fc.Users) < 14 {
				caller := "(level berkas)"
				if es, _, _, ok := b.enclosing(r.file, r.line); ok {
					caller = es.Name
				}
				fc.Users = append(fc.Users, fmt.Sprintf("%s ← %s (%s:%d)", s.Name, caller, r.file.Path, r.line))
				shown++
			}
		}
		if uses == 0 && !frameworkEntry(f, s) {
			fc.Unused = append(fc.Unused, fmt.Sprintf("%s (L%d)", s.Name, s.Line))
		}
	}
	if f.Lang != "go" {
		for _, s := range f.Symbols {
			if isCallable(s.Kind) && len(fc.Routes) < 10 {
				tmp := &Context{}
				b.routesFor(tmp, f, s.Name)
				fc.Routes = append(fc.Routes, tmp.Routes...)
			}
		}
	}
	return fc
}

type identRef struct {
	file *codemap.FileSymbols
	line int
	test bool
}

var identRe = regexp.MustCompile(`[A-Za-z_][\w]*`)

// identIndex: identifier → tempat kemunculannya (satu per baris) di berkas lingkup & test.
func (b *builder) identIndex() map[string][]identRef {
	out := map[string][]identRef{}
	for i := range b.files {
		o := &b.files[i]
		test := b.testInScope(o)
		if !test && !b.inScope(o) {
			continue
		}
		for n, l := range b.src.lines(o.AbsPath) {
			seen := map[string]bool{}
			for _, id := range identRe.FindAllString(l, -1) {
				if len(id) >= 3 && !seen[id] {
					seen[id] = true
					out[id] = append(out[id], identRef{file: o, line: n + 1, test: test})
				}
			}
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// frameworkEntry: fungsi yang dipanggil framework/runtime, bukan kode aplikasi (tidak dianggap tak terpakai).
func frameworkEntry(f *codemap.FileSymbols, s models.CodeSymbol) bool {
	n := s.Name
	switch {
	case n == "main" || n == "init" || n == "__construct" || n == "__invoke" || strings.HasPrefix(n, "__"):
		return true
	case f.Lang == "go" && len(n) > 0 && n[0] >= 'A' && n[0] <= 'Z':
		return true // simbol ekspor bisa dipakai paket lain di luar lingkup
	case s.Exported:
		return true
	}
	return false
}

// Prompt merender FileContext untuk review LLM.
func (fc *FileContext) Prompt() string {
	if fc == nil || (len(fc.External) == 0 && len(fc.Users) == 0 && len(fc.Routes) == 0 && len(fc.Tests) == 0) {
		return ""
	}
	var b strings.Builder
	mode := "peta kode + resolusi nama leksikal"
	if fc.Precise {
		mode = "call graph bertipe go/types"
	}
	fmt.Fprintf(&b, "\nKonteks Lintas Berkas (%s) — nilai berkas ini sebagai bagian dari alur aplikasi, seperti senior developer: periksa juga apakah validasi/otorisasi/transaksi sudah ditangani di fungsi lain sebelum melaporkannya.\n", mode)
	if len(fc.Routes) > 0 {
		b.WriteString("Rute yang ditangani berkas ini:\n")
		for _, r := range fc.Routes {
			b.WriteString("- " + r + "\n")
		}
	}
	if len(fc.External) > 0 {
		b.WriteString("Fungsi di berkas lain yang dipakai berkas ini (layer dalam lebih dulu):\n")
		for _, s := range fc.External {
			fmt.Fprintf(&b, "· %s — %s:%d-%d", s.Name, s.Path, s.From, s.To)
			if s.Layer != "" {
				b.WriteString(" (layer " + s.Layer + ")")
			}
			b.WriteString("\n" + s.Code)
		}
	}
	if len(fc.Users) > 0 {
		b.WriteString("Dipakai di berkas lain:\n")
		for _, u := range fc.Users {
			b.WriteString("- " + u + "\n")
		}
	}
	if len(fc.Unused) > 0 {
		b.WriteString("Fungsi berkas ini yang tidak dirujuk di mana pun (kandidat kode mati): " + strings.Join(fc.Unused, ", ") + "\n")
	}
	if len(fc.Tests) > 0 {
		b.WriteString("Diuji oleh: " + strings.Join(fc.Tests, ", ") + "\n")
	} else {
		b.WriteString("Diuji oleh: belum ada test yang merujuk fungsi berkas ini.\n")
	}
	return b.String()
}
