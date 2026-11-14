package routemap

import (
	"go/ast"
	"regexp"
	"strings"
)

// Fungsi di luar rute HTTP juga ikut biaya proses: worker antrean (`queue.RegisterHandler(T, handleX)`), goroutine
// dan job yang dijalankan dari main/init, serta callback yang diteruskan sebagai nilai. Akarnya fungsi main & init
// ditambah setiap fungsi lokal yang dirujuk sebagai nilai (bukan dipanggil) dari fungsi yang terjangkau; dari situ
// diikuti pemanggilannya. Skrip sekali jalan (benchmark, seed, migrasi, alat bantu) tidak ikut: loop & query di
// sana disengaja.

// oneOffToolRe: sama dengan pemeriksaan skrip sekali jalan di paket diagnostics (tidak diimpor agar tidak siklik).
var oneOffToolRe = regexp.MustCompile(`(^|/)(cmd/bench[^/]*|cmd/[^/]*(?:check|fix|backfill|cleanup|repair|reindex|maint)[^/]*|bench(marks?)?|scripts?|seeds?|seeders?|migrations?|fixtures|tools)(/|$)|(^|[_/-])(seed|seeder|migrate|migration|bench)[_.-]`)

func oneOffFile(fi *fileInfo) bool {
	return fi != nil && oneOffToolRe.MatchString(strings.ToLower(fi.Rel))
}

// backgroundReach: fungsi yang dijalankan di luar rute (lihat atas), tanpa skrip sekali jalan.
func (g *Graph) backgroundReach(m *module) map[string]bool {
	seen := map[string]bool{}
	var stack []string
	push := func(k string) {
		if f := m.funcs[k]; f != nil && !seen[k] && !oneOffFile(f.file) {
			seen[k] = true
			stack = append(stack, k)
		}
	}
	for k, f := range m.funcs {
		if f.Recv == "" && (f.Name == "main" || f.Name == "init") && !(f.Name == "main" && g.toolMain(f)) {
			push(k)
		}
	}
	for len(stack) > 0 {
		k := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, c := range g.callees[k] {
			push(c)
		}
		for _, r := range valueRefs(m, m.funcs[k]) {
			push(r)
		}
	}
	return seen
}

// valueRefs: fungsi lokal yang dirujuk sebagai nilai di badan f (argumen, assignment, `go`/`defer` lewat nilai),
// bukan dipanggil langsung.
func valueRefs(m *module, f *funcInfo) []string {
	if f == nil || f.Decl == nil || f.Decl.Body == nil {
		return nil
	}
	called := map[ast.Expr]bool{}
	ast.Inspect(f.Decl.Body, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok {
			called[ce.Fun] = true
		}
		return true
	})
	var out []string
	ast.Inspect(f.Decl.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident, *ast.SelectorExpr:
			e := x.(ast.Expr)
			if called[e] {
				return true
			}
			for _, r := range m.resolveFuncs(f.file, e) {
				out = append(out, r.Key)
			}
			if _, isSel := x.(*ast.SelectorExpr); isSel {
				return false // jangan menelusuri X sebagai rujukan terpisah
			}
		}
		return true
	})
	return out
}

// toolMain: `main` di cmd/<nama>/ yang tidak membuat router HTTP = alat bantu/skrip (bandingkan data, unduh aset,
// ambil token), bukan aplikasi yang melayani pengguna.
func (g *Graph) toolMain(f *funcInfo) bool {
	if f.file == nil {
		return false
	}
	rel := strings.ToLower(f.file.Rel)
	if !strings.HasPrefix(rel, "cmd/") && !strings.Contains(rel, "/cmd/") {
		return false
	}
	dir := f.file.Rel[:strings.LastIndex(f.file.Rel, "/")+1]
	for _, r := range g.Routers {
		if strings.HasPrefix(r.Pos, dir) {
			return false
		}
		for _, st := range r.Stack {
			if strings.HasPrefix(st, dir) {
				return false
			}
		}
	}
	return true
}
