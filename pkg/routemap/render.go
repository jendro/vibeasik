package routemap

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// maxMarkdownRoutes membatasi rute per modul di CODEMAP.md; sisanya ada lengkap di routes.jsonl.
const maxMarkdownRoutes = 400

func mwLabel(mw Middleware) string {
	s := mw.Name
	if mw.Decl != "" && !strings.Contains(s, mw.Decl) && mw.Name != "func literal" {
		s += " [" + mw.Decl + "]"
	} else if mw.Name == "func literal" {
		s = "func literal [" + mw.Pos + "]"
	}
	if mw.Auth {
		if mw.Optional {
			s += " (auth opsional)"
		} else {
			s += " (auth)"
		}
	}
	return s
}

func respLabel(r Response) string {
	st := "?"
	if r.Status > 0 {
		st = strconv.Itoa(r.Status)
	}
	s := st + " " + r.Kind
	if r.Via != "" {
		s += " via " + r.Via
	}
	return s
}

// globalChain: middleware global modul (scope global, urut pendaftaran) — ditulis sekali per modul.
func globalChain(g *Graph) []Middleware {
	var out []Middleware
	seen := map[string]bool{}
	for _, r := range g.Routers {
		if r.Parent >= 0 {
			continue
		}
		for _, mw := range r.Middlewares {
			k := mw.Pos + mw.Name
			if !seen[k] {
				seen[k] = true
				out = append(out, mw)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].seq < out[j].seq })
	return out
}

// statusList: "200,400 via fail,500" ringkas & unik.
func statusList(rs []Response) string {
	var out []string
	seen := map[string]bool{}
	for _, r := range rs {
		if r.Kind == "next" {
			continue
		}
		st := "?"
		if r.Status > 0 {
			st = strconv.Itoa(r.Status)
		}
		if r.Kind != "json" && r.Kind != "error" && r.Kind != "status" {
			st += " " + r.Kind
		}
		if !seen[st] {
			seen[st] = true
			out = append(out, st)
		}
	}
	return strings.Join(out, ",")
}

// redundant: middleware rute yang namanya sudah aktif lebih dulu di rantai (dari global/group).
func redundant(rt *Route) map[int]bool {
	out := map[int]bool{}
	for i := range repeated(rt) {
		if rt.Chain[i].Scope != "global" {
			out[i] = true
		}
	}
	return out
}

// repeated: indeks middleware di rantai rute → indeks kemunculan pertama middleware yang sama. Func literal
// anonim dan nama terpotong tidak bisa dibuktikan sama, jadi tidak dihitung.
func repeated(rt *Route) map[int]int {
	out := map[int]int{}
	first := map[string]int{}
	for i, mw := range rt.Chain {
		if mw.Name == "func literal" || strings.HasSuffix(mw.Name, "…") {
			continue
		}
		key := mw.Name + "|" + mw.Decl
		if j, ok := first[key]; ok {
			out[i] = j
			continue
		}
		first[key] = i
	}
	return out
}

// Markdown menulis bagian "Peta Rute" untuk CODEMAP.md: rantai middleware ditulis sekali per kelompok,
// lalu satu baris per rute (detail lengkap di routes.jsonl) agar hemat token.
func Markdown(graphs []*Graph) string {
	if len(graphs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Peta Rute (router → middleware → handler → response)\n\n")
	b.WriteString("> Ditelusuri statis dari `func main` dengan go/parser + go/types: prefix group, middleware efektif per rute sesuai semantik framework ")
	b.WriteString("(urutan pendaftaran, prefix Use/Group, skip `Next`), status response handler (termasuk lewat helper), fungsi yang dijangkau. ")
	b.WriteString("Sebelum menyimpulkan \"rute tanpa auth/validasi\", cek rantai di sini: middleware global & group tidak terlihat di baris rute. ")
	b.WriteString("Detail per rute (jalur setup dari main, response + body, semua fungsi yang dipanggil) ada di `.vibeasik/routes.jsonl`; ")
	b.WriteString("MCP `vibeasik_code_map_routes` (filter path/method, atau `function` untuk menelusur balik fungsi → rute).\n\n")
	for _, g := range graphs {
		writeModuleMarkdown(&b, g)
	}
	return b.String()
}

func writeModuleMarkdown(b *strings.Builder, g *Graph) {
	fmt.Fprintf(b, "### Modul `%s` (`%s`, %s) — %d rute\n\n", g.Module, g.Dir, g.Framework, len(g.Routes))
	if len(g.Entries) > 0 {
		fmt.Fprintf(b, "- Titik masuk: %s\n", "`"+strings.Join(g.Entries, "`, `")+"`")
	}
	globals := writeGlobalMiddleware(b, g)
	writeGroupMiddleware(b, g)
	open := openMutatingRoutes(g)
	for _, w := range g.Warnings {
		b.WriteString("- ⚠ " + w + "\n")
	}
	writeDuplicateWarnings(b, g)
	writeOpenRoutes(b, open)
	b.WriteString("\n")

	order, buckets := bucketRoutes(g, globals)
	written := 0
	for _, k := range order {
		written += writeBucket(b, buckets[k], k, globals, written)
	}
	if written < len(g.Routes) {
		fmt.Fprintf(b, "- … %d rute lain di `.vibeasik/routes.jsonl`\n\n", len(g.Routes)-written)
	}
}

// writeGlobalMiddleware menulis daftar middleware global dan mengembalikan kumpulan kuncinya.
func writeGlobalMiddleware(b *strings.Builder, g *Graph) map[string]bool {
	globals := map[string]bool{}
	gc := globalChain(g)
	if len(gc) == 0 {
		return globals
	}
	b.WriteString("- **Middleware global** (urut eksekusi, berlaku untuk rute yang didaftarkan sesudahnya):\n")
	for i, mw := range gc {
		globals[mw.Pos+mw.Name] = true
		note := ""
		if late := lateRouteCount(g, mw); late > 0 {
			note = fmt.Sprintf(" — tidak berjalan untuk %d rute (didaftarkan sesudahnya/dilewati)", late)
		}
		fmt.Fprintf(b, "  %d. %s — `%s`%s\n", i+1, mwLabel(mw), mw.Pos, note)
	}
	return globals
}

// lateRouteCount: jumlah rute yang middleware-nya tercatat tidak aktif.
func lateRouteCount(g *Graph, mw Middleware) int {
	late := 0
	for _, rt := range g.Routes {
		for _, in := range rt.Inactive {
			if in.Pos == mw.Pos && in.Name == mw.Name {
				late++
			}
		}
	}
	return late
}

func writeGroupMiddleware(b *strings.Builder, g *Graph) {
	var groups []string
	for _, r := range g.Routers {
		if r.Parent >= 0 && len(r.Middlewares) > 0 {
			var names []string
			for _, mw := range r.Middlewares {
				names = append(names, mwLabel(mw))
			}
			groups = append(groups, fmt.Sprintf("  - `%s` (`%s`): %s", r.Prefix, r.Pos, strings.Join(names, " → ")))
		}
	}
	if len(groups) > 0 {
		b.WriteString("- **Group bermiddleware**:\n" + strings.Join(groups, "\n") + "\n")
	}
}

// openMutatingRoutes: rute pengubah data tanpa auth wajib.
func openMutatingRoutes(g *Graph) []string {
	var open []string
	for _, rt := range g.Routes {
		if !rt.Auth && !rt.Unreachable && rt.Method != "GET" && rt.Method != "HEAD" && rt.Method != "OPTIONS" {
			open = append(open, "`"+rt.Method+" "+rt.Path+"`")
		}
	}
	return open
}

func writeDuplicateWarnings(b *strings.Builder, g *Graph) {
	for _, d := range g.Duplicates {
		what := "Middleware `" + d.Name + "` terpasang ulang"
		if d.Kind == "validation" {
			what = "Validasi `" + d.Name + "` diperiksa ulang"
		}
		fmt.Fprintf(b, "- ⚠ %s di `%s` (sudah di `%s`, %s) — %d rute, mis. `%s` (redundan, patokan ARCH-004).\n", what, d.Pos, d.First, d.Detail, d.Count, d.Routes[0])
	}
}

func writeOpenRoutes(b *strings.Builder, open []string) {
	if len(open) == 0 {
		return
	}
	more := ""
	if len(open) > 25 {
		more = fmt.Sprintf(" … +%d lagi (lihat routes.jsonl, `auth:false`)", len(open)-25)
		open = open[:25]
	}
	fmt.Fprintf(b, "- Rute pengubah data **tanpa** middleware auth wajib (pastikan memang publik): %s%s\n", strings.Join(open, ", "), more)
}

// routeBucket = kelompok rute yang berbagi rantai non-global (group/prefix/with).
type routeBucket struct {
	label  string
	routes []*Route
}

func bucketRoutes(g *Graph, globals map[string]bool) ([]string, map[string]*routeBucket) {
	var order []string
	buckets := map[string]*routeBucket{}
	for _, rt := range g.Routes {
		key, label := bucketKey(rt, globals)
		if buckets[key] == nil {
			buckets[key] = &routeBucket{label: label}
			order = append(order, key)
		}
		buckets[key].routes = append(buckets[key].routes, rt)
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i] != "dead" && order[j] == "dead" })
	return order, buckets
}

func bucketKey(rt *Route, globals map[string]bool) (key, label string) {
	if rt.Unreachable {
		return "dead", "Tidak terpasang (fungsi setup tidak terjangkau dari main — kemungkinan kode mati)"
	}
	var parts []string
	for _, mw := range rt.Chain {
		if !globals[mw.Pos+mw.Name] && mw.Scope != "route" {
			parts = append(parts, mwLabel(mw)+" {"+mw.Scope+" "+mw.ScopePath+"}")
		}
	}
	key = strings.Join(parts, " → ")
	label = "global"
	if key != "" {
		label = "global → " + key
	}
	return key, label
}

// writeBucket menulis satu kelompok rute; written = rute yang sudah ditulis sebelumnya (batas
// maxMarkdownRoutes). Mengembalikan jumlah rute yang ditulis di kelompok ini.
func writeBucket(b *strings.Builder, bk *routeBucket, k string, globals map[string]bool, written int) int {
	auth := "publik"
	if len(bk.routes) > 0 && bk.routes[0].Auth {
		auth = "auth"
	}
	if k == "dead" {
		auth = "tidak aktif"
	}
	fmt.Fprintf(b, "#### Rantai: %s · %s (%d rute)\n\n", bk.label, auth, len(bk.routes))
	n := 0
	for _, rt := range bk.routes {
		if written+n >= maxMarkdownRoutes {
			break
		}
		n++
		writeRouteLine(b, rt, auth, k, globals)
	}
	b.WriteString("\n")
	return n
}

func writeRouteLine(b *strings.Builder, rt *Route, auth, k string, globals map[string]bool) {
	fmt.Fprintf(b, "- `%s %s` → `%s`", rt.Method, rt.Path, rt.Handler)
	if rt.HandlerDecl != "" {
		fmt.Fprintf(b, " `%s`", rt.HandlerDecl)
	}
	if st := statusList(rt.Responses); st != "" {
		b.WriteString(" · " + st)
	}
	red := redundant(rt)
	var extra []string
	for i, mw := range rt.Chain {
		if mw.Scope == "route" {
			l := mwLabel(mw)
			if red[i] {
				l += " (redundan)"
			}
			extra = append(extra, l)
		}
	}
	if len(extra) > 0 {
		b.WriteString(" · +rute: " + strings.Join(extra, ", "))
	}
	if rt.Auth != (auth == "auth") && k != "dead" {
		if rt.Auth {
			b.WriteString(" · **auth**")
		} else {
			b.WriteString(" · **publik**")
		}
	}
	for _, mw := range rt.Inactive {
		if mw.Auth || !globals[mw.Pos+mw.Name] { // global yang telat sudah dicatat sekali di daftar global
			b.WriteString(" · ✗ " + mw.Name + ": " + mw.Reason)
		}
	}
	b.WriteString("\n")
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

type jsonlRoute struct {
	Module string `json:"module"`
	*Route
}

// JSONL: satu rute per baris (lengkap: stack setup, rantai, middleware tidak aktif, response, pemanggilan).
func JSONL(graphs []*Graph) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	for _, g := range graphs {
		for _, rt := range g.Routes {
			_ = enc.Encode(jsonlRoute{Module: g.Module, Route: rt})
		}
	}
	return b.String()
}

// Trace = penjelasan teks satu rute dari main sampai response (untuk MCP & prompt agent).
func Trace(rt *Route) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s  (didaftarkan %s)\n", rt.Method, rt.Path, rt.Pos)
	if rt.Unreachable {
		b.WriteString("⚠ Fungsi setup rute ini tidak terjangkau dari main — kemungkinan kode mati.\n")
	}
	if len(rt.Stack) > 0 {
		b.WriteString("Jalur setup: " + strings.Join(rt.Stack, " → ") + " → " + rt.Pos + "\n")
	}
	b.WriteString("Rantai middleware (urut eksekusi):\n")
	for i, mw := range rt.Chain {
		scope := mw.Scope
		if mw.ScopePath != "" {
			scope += " " + mw.ScopePath
		}
		fmt.Fprintf(&b, "  %d. %s {%s} — %s\n", i+1, mwLabel(mw), scope, mw.Pos)
	}
	for _, mw := range rt.Inactive {
		fmt.Fprintf(&b, "  ✗ %s — %s (%s)\n", mwLabel(mw), mw.Reason, mw.Pos)
	}
	fmt.Fprintf(&b, "Auth wajib: %v\n", rt.Auth)
	fmt.Fprintf(&b, "Handler: %s", rt.Handler)
	if rt.HandlerDecl != "" {
		fmt.Fprintf(&b, " (%s)", rt.HandlerDecl)
	}
	b.WriteString("\n")
	for _, r := range rt.Responses {
		fmt.Fprintf(&b, "  → %s", respLabel(r))
		if r.Body != "" {
			b.WriteString(" " + r.Body)
		}
		fmt.Fprintf(&b, " (%s)\n", r.Pos)
	}
	if len(rt.Calls) > 0 {
		b.WriteString("Memanggil: " + strings.Join(rt.Calls, ", ") + "\n")
	}
	return b.String()
}

// ContextFor: jejak rute (maks. 3) yang relevan dengan temuan di berkas:baris, untuk prompt LLM audit/agent.
// Urutan: rute yang didaftarkan tepat di baris itu, lalu rute yang handler-nya menjangkau fungsi/berkas target
// (target bisa "handlers/x.go" atau nama handler "handlers.AdminResendCertMail").
func ContextFor(graphs []*Graph, target string, line int) string {
	if len(graphs) == 0 || target == "" {
		return ""
	}
	var routes []*Route
	if rt := RouteAt(graphs, target, line); rt != nil {
		routes = append(routes, rt)
	} else if _, fr := FuncAt(graphs, target, line); fr != nil {
		// temuan di dalam fungsi: hanya rute yang menjangkau fungsi itu, bukan semua rute berkasnya
		routes = FindRoutes(graphs, "", "", fr.Key)
	} else {
		q := target
		if i := strings.LastIndex(q, "."); i > 0 && !strings.Contains(q, "/") && !strings.HasSuffix(q, ".go") {
			q = q[i+1:] // handlers.AdminResendCertMail → AdminResendCertMail
		}
		routes = FindRoutes(graphs, "", "", q)
	}
	if len(routes) == 0 {
		return ""
	}
	var b strings.Builder
	for i, rt := range routes {
		if i == 3 {
			fmt.Fprintf(&b, "(+%d rute lain menjangkau target ini)\n", len(routes)-3)
			break
		}
		b.WriteString(Trace(rt) + "\n")
	}
	return b.String()
}
