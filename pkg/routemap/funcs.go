package routemap

import (
	"sort"
	"strconv"
	"strings"
)

// FuncRef adalah satu fungsi/method lokal di call graph bertipe (posisi relatif root proyek).
type FuncRef struct {
	Key   string `json:"key"`
	Name  string `json:"name"` // pkg.Func / pkg.Type.Method
	File  string `json:"file"`
	Line  int    `json:"line"`
	End   int    `json:"end"`
	Calls int    `json:"calls,omitempty"` // jumlah fungsi lokal yang dipanggilnya
}

func (g *Graph) ref(k string) (FuncRef, bool) {
	f := g.funcs[k]
	if f == nil {
		return FuncRef{}, false
	}
	file, ln := splitPos(f.Pos)
	name := f.file.PkgName + "." + f.Name
	if f.Recv != "" {
		name = f.file.PkgName + "." + f.Recv + "." + f.Name
	}
	return FuncRef{Key: k, Name: name, File: file, Line: ln, End: f.End, Calls: len(g.callees[k])}, true
}

func splitPos(pos string) (string, int) {
	i := strings.LastIndex(pos, ":")
	if i < 0 {
		return pos, 0
	}
	n, _ := strconv.Atoi(pos[i+1:])
	return pos[:i], n
}

// FuncAt mencari fungsi lokal yang memuat berkas:baris (berkas relatif root proyek atau akhiran path, mis.
// "handlers/x.go"). line ≤ 0 → nil. Dipakai audit untuk membaca fungsi utuh beserta pemanggil & yang dipanggil.
func FuncAt(graphs []*Graph, file string, line int) (*Graph, *FuncRef) {
	file = strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "./")
	if file == "" || line <= 0 {
		return nil, nil
	}
	for _, g := range graphs {
		for k, f := range g.funcs {
			fp, ln := splitPos(f.Pos)
			if (fp == file || strings.HasSuffix(fp, "/"+file)) && line >= ln && line <= f.End {
				r, _ := g.ref(k)
				return g, &r
			}
		}
	}
	return nil, nil
}

// FuncNamed mencari fungsi lokal berdasarkan nama (Func, Type.Method, pkg.Type.Method) — untuk temuan yang
// TargetFile-nya nama handler, bukan berkas.
func FuncNamed(graphs []*Graph, name string) (*Graph, *FuncRef) {
	if name == "" {
		return nil, nil
	}
	for _, g := range graphs {
		for k := range g.funcs {
			r, _ := g.ref(k)
			if r.Name == name || strings.HasSuffix(r.Name, "."+name) {
				return g, &r
			}
		}
	}
	return nil, nil
}

func (g *Graph) refs(keys []string) []FuncRef {
	out := make([]FuncRef, 0, len(keys))
	seen := map[string]bool{}
	for _, k := range keys {
		if seen[k] {
			continue
		}
		seen[k] = true
		if r, ok := g.ref(k); ok {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].File+strconv.Itoa(out[i].Line) < out[j].File+strconv.Itoa(out[j].Line)
	})
	return out
}

// Callees: fungsi lokal yang dipanggil langsung oleh fungsi key (go/types, termasuk implementasi interface).
func (g *Graph) Callees(key string) []FuncRef { return g.refs(g.callees[key]) }

// Callers: fungsi lokal yang memanggil fungsi key secara langsung.
func (g *Graph) Callers(key string) []FuncRef { return g.refs(g.callers[key]) }

// RoutesReaching: rute yang handler-nya (langsung atau lewat rantai pemanggilan) menjangkau fungsi key.
func (g *Graph) RoutesReaching(key string) []*Route {
	reach := g.reaching(key)
	var out []*Route
	for _, rt := range g.Routes {
		if reach[rt.handlerKey] {
			out = append(out, rt)
		}
	}
	return out
}
