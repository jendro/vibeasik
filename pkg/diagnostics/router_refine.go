package diagnostics

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/routemap"
)

// refineRouterFindings memeriksa ulang temuan ROUTER-UNGUARDED-MUTATION (aturan per baris) dengan peta rute
// statis (routemap: main → group → middleware efektif). Rute yang rantainya memuat middleware auth wajib
// (dipasang di global/group/prefix) dibuang; rute dari fungsi setup yang tidak terjangkau dari main dibuang
// dan dilaporkan sekali sebagai kode mati; sisanya diperkaya rantai middleware efektifnya. Pemeriksaan ganda
// di jalur rute (middleware/validasi, patokan ARCH-004) ikut dilaporkan.
func refineRouterFindings(findings []models.Finding, dir, baseRel string, hasGo bool) []models.Finding {
	if !hasGo {
		return findings
	}
	graphs := routemap.Analyze(dir)
	if len(graphs) == 0 {
		return findings
	}
	out := findings[:0]
	for _, f := range findings {
		if f.RuleID != "ROUTER-UNGUARDED-MUTATION" || !strings.HasSuffix(f.TargetFile, ".go") {
			out = append(out, f)
			continue
		}
		rt := routemap.RouteAt(graphs, filepath.ToSlash(f.TargetFile), f.Line)
		switch {
		case rt == nil:
			out = append(out, f)
		case rt.Auth || rt.Unreachable:
			continue
		default:
			var chain []string
			for _, mw := range rt.Chain {
				chain = append(chain, mw.Name+" {"+mw.Scope+"}")
			}
			if len(chain) == 0 {
				chain = []string{"(tidak ada)"}
			}
			f.Description += fmt.Sprintf(" Peta rute (dari main): %s %s, rantai middleware efektif: %s.", rt.Method, rt.Path, strings.Join(chain, " → "))
			out = append(out, f)
		}
	}
	for _, g := range graphs {
		out = append(out, duplicateFindings(g, baseRel)...)
		for _, w := range g.Warnings {
			if !strings.HasPrefix(w, "Fungsi setup ") {
				continue
			}
			file, line := warningPos(w)
			if baseRel != "" {
				file = filepath.ToSlash(filepath.Join(baseRel, file))
			}
			out = append(out, models.Finding{
				Category: "router", Severity: "INFO", RuleID: "ROUTER-UNREACHABLE-SETUP",
				Title:       "Fungsi Pendaftaran Rute Tidak Terjangkau dari main",
				Description: w,
				Suggestion:  "Pasang fungsi setup ini dari alur router utama, atau hapus bila memang kode mati agar AI/manusia tidak mengira rutenya aktif.",
				TargetFile:  file, Line: line,
				TheoryBasis: "Dead code elimination & single source of truth routing (Martin Fowler, Refactoring: Remove Dead Code).",
			})
		}
	}
	return out
}

// warningPos mengambil "berkas:baris" dari teks peringatan "Fungsi setup x (berkas.go:12) …".
func warningPos(w string) (string, int) {
	i, j := strings.Index(w, "("), strings.Index(w, ")")
	if i < 0 || j < i {
		return "", 0
	}
	return splitPos(w[i+1 : j])
}

// duplicateFindings: satu temuan per pemeriksaan ganda, di lokasi salinannya (yang kandidat dihapus).
func duplicateFindings(g *routemap.Graph, baseRel string) []models.Finding {
	var out []models.Finding
	for _, d := range g.Duplicates {
		file, line := splitPos(d.Pos)
		if baseRel != "" {
			file = filepath.ToSlash(filepath.Join(baseRel, file))
		}
		routes := strings.Join(d.Routes, ", ")
		if d.Count > len(d.Routes) {
			routes += fmt.Sprintf(" (+%d rute lain)", d.Count-len(d.Routes))
		}
		f := models.Finding{
			Category: "router", Severity: "INFO", TargetFile: file, Line: line,
			TheoryBasis: "KISS & single source of truth (patokan ARCH-004; Martin Fowler, Refactoring: Remove Duplicated Code).",
		}
		if d.Kind == "middleware" {
			f.RuleID, f.Severity = "ROUTER-DUPLICATE-MIDDLEWARE", "WARNING"
			f.Title = "Middleware Terpasang Ganda di Rantai Rute"
			f.Description = fmt.Sprintf("Middleware `%s` sudah aktif di %s (%s) lalu dipasang lagi di sini, sehingga berjalan dua kali per request di: %s.", d.Name, d.First, d.Detail, routes)
			f.Suggestion = "Hapus pemasangan ulang ini dan biarkan middleware di lapisan yang lebih luas (global/group). Bila konfigurasinya memang berbeda, beri nama/konfigurasi yang jelas berbeda."
		} else {
			f.RuleID = "ROUTER-DUPLICATE-VALIDATION"
			f.Title = "Validasi Ganda di Jalur Rute"
			f.Description = fmt.Sprintf("Kondisi `%s` sudah diperiksa di %s lalu diperiksa lagi di sini (%s) pada jalur: %s.", d.Name, d.First, d.Detail, routes)
			f.Suggestion = "Pilih satu lapisan validasi (handler/validator untuk bentuk input, service untuk aturan domain) lalu hapus salinannya. Pertahankan salinan service bila fungsinya juga dipanggil dari jalur lain tanpa validasi handler."
		}
		out = append(out, f)
	}
	return out
}

// splitPos memecah "berkas.go:12" (posisi routemap) menjadi berkas & baris.
func splitPos(pos string) (string, int) {
	file, ln, _ := strings.Cut(pos, ":")
	n := 0
	fmt.Sscanf(ln, "%d", &n)
	return file, n
}
