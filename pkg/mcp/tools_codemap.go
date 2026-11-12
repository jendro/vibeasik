package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/codemap"
	"github.com/vibeasik/vibeasik/pkg/insight"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/routemap"
	"github.com/vibeasik/vibeasik/pkg/workspace"
)

// Peta kode: indeks per berkas (peran, simbol, impor, dipakai-oleh, rute, ringkasan) yang dicari ala RAG
// agar agent menemukan lokasi kode tanpa menjelajah seluruh repositori.

const codeMapProjectHint = "Project ID. Omit to resolve from workspace_path (or the current directory / .vibeasik.json)."

func (s *Server) codeMapToolsList() []Tool {
	projectProps := func(extra map[string]interface{}) map[string]interface{} {
		p := map[string]interface{}{
			"project_id":     prop("integer", codeMapProjectHint),
			"workspace_path": prop("string", "Directory inside the project, used when project_id is omitted"),
		}
		for k, v := range extra {
			p[k] = v
		}
		return p
	}
	return []Tool{
		{
			Name: "vibeasik_code_map_search",
			Description: "RAG-style search over the project's code map: returns the most relevant files (path, role/kind, summary, matching symbols with line numbers, routes, who uses the file) for a free-text query in Indonesian or English. " +
				"Use it BEFORE grepping or reading many files to locate where something lives. snippet=true adds the code around the best matching symbol for the top hits. The map refreshes itself incrementally.",
			InputSchema: ToolSchema{Type: "object", Properties: projectProps(map[string]interface{}{
				"query":        prop("string", "What you are looking for, e.g. 'hapus todo', 'exec guard localhost', 'useBackgroundJobs', '/api/queue'"),
				"kind":         prop("string", "Optional role filter: page, component, composable, store, handler, route, service, repository, model, type, middleware, util, config, entry, test, doc, migration, module"),
				"component_id": prop("integer", "Optional: only files of this component/app"),
				"limit":        prop("integer", "Max results (default 10, max 50)"),
				"snippet":      prop("boolean", "Include code snippets for the top 5 hits"),
			}), Required: []string{"query"}},
		},
		{
			Name: "vibeasik_code_map_overview",
			Description: "Top-level map of a project for AI agents: apps/components and their directories (file counts, roles, key files), core modules (most imported), hotspots (files too large to read whole), entry points/route files, and context docs (AI_CONTEXT.md, README). " +
				"detail=files also lists every file with its summary and symbols (large).",
			InputSchema: ToolSchema{Type: "object", Properties: projectProps(map[string]interface{}{
				"component_id": prop("integer", "Optional: only this component/app"),
				"detail":       prop("string", "'dirs' (default) or 'files'"),
			})},
		},
		{
			Name:        "vibeasik_code_map_file",
			Description: "Details of one file from the code map: all symbols with line numbers, local imports, files that use it (estimate the blast radius before editing), routes it defines or API endpoints it calls, and its summary.",
			InputSchema: ToolSchema{Type: "object", Properties: projectProps(map[string]interface{}{
				"file": prop("string", "Project-relative path, absolute path, or unique suffix such as 'TodoCard.vue'"),
			}), Required: []string{"file"}},
		},
		{
			Name: "vibeasik_code_map_build",
			Description: "Rebuilds the project's code map now (incremental: only changed files are parsed; force=true reparses everything). Search/overview already refresh maps older than 30 seconds, so this is rarely needed. " +
				"export=true also rewrites .vibeasik/CODEMAP.md + codemap.jsonl when files changed (same as the auto-sync of initialized projects).",
			InputSchema: ToolSchema{Type: "object", Properties: projectProps(map[string]interface{}{
				"force":  prop("boolean", "Reparse every file"),
				"export": prop("boolean", "Rewrite .vibeasik/CODEMAP.md + codemap.jsonl if anything changed (ignored with force)"),
			})},
		},
		{
			Name: "vibeasik_code_map_routes",
			Description: "Traces HTTP routes of Go backends statically (Fiber/Gin/Echo/Chi/gorilla/net/http) from func main: setup call stack, group prefixes, the EFFECTIVE middleware chain per route (global → group → prefix → route, in execution order, incl. middleware that is registered but inactive because it was added after the route or skipped via Next/skip paths), whether auth is enforced, handler declaration, response statuses (also via helpers such as fail(c, 400, ...)), and local functions the handler reaches. " +
				"Filter by path substring, method, or function (name, pkg.Func, or file) to go backwards: function → handler → route → group → main. Use it before concluding a route lacks auth/validation: middleware applied at group or global level is invisible on the route line itself.",
			InputSchema: ToolSchema{Type: "object", Properties: projectProps(map[string]interface{}{
				"path":     prop("string", "Route path substring, e.g. '/api/admin/users'"),
				"method":   prop("string", "HTTP method filter, e.g. POST"),
				"function": prop("string", "Function name / pkg.Func / file path: list routes whose handler reaches it"),
				"limit":    prop("integer", "Max routes (default 20, max 200)"),
				"format":   prop("string", "'text' (default, readable trace) or 'json'"),
			})},
		},
		{
			Name: "vibeasik_code_map_cost",
			Description: "Process cost per Go route or function, computed statically from the AST: tick formula T(n) (1 tick = one executed statement; n = number of data rows), Big-O class, T(1), and the database query count Q(n) — a query inside a data loop yields Q(n) = n + 1 and is flagged N+1. Sorted most expensive first as data grows. " +
				"Use it before and after touching a handler to prove the change did not raise its growth class (e.g. O(n) → O(n²)) or introduce an N+1 query, and to find the routes worth optimising. Worst case per branch; all collections share one size n; recursion is flagged and its formula is a lower bound.",
			InputSchema: ToolSchema{Type: "object", Properties: projectProps(map[string]interface{}{
				"kind":       prop("string", "'routes' (default) or 'funcs'"),
				"q":          prop("string", "Substring filter: route path/handler, or function name/file"),
				"class":      prop("string", "Growth class filter: constant, linear, linearithmic, quadratic, cubic, polynomial"),
				"n_plus_one": prop("boolean", "Only entries with a query inside a data loop"),
				"limit":      prop("integer", "Max entries (default 20, max 200)"),
			})},
		},
		{
			Name:        "vibeasik_code_map_export",
			Description: "Writes the code map into the repository as .vibeasik/CODEMAP.md (readable map for agents without MCP) and .vibeasik/codemap.jsonl (one chunk per file for grep/RAG).",
			InputSchema: ToolSchema{Type: "object", Properties: projectProps(nil)},
		},
		{
			Name: "vibeasik_code_map_status",
			Description: "Health of the project's code map: effective root, file/symbol counts, languages and roles, last build time and staleness, whether it was initialized (CODEMAP.md written and kept in sync by auto-sync) and how many files still lack an LLM summary. " +
				"Use it to decide whether to run link_project_workspace (Init vibeasik) or vibeasik_code_map_enrich.",
			InputSchema: ToolSchema{Type: "object", Properties: projectProps(nil)},
		},
	}
}

func (s *Server) codeMapProjectID(args map[string]interface{}) (uint, error) {
	if pid := argUint(args, "project_id"); pid > 0 {
		return pid, nil
	}
	wp, _ := args["workspace_path"].(string)
	match, err := workspace.IdentifyWorkspace(s.db, wp, "http://127.0.0.1:8765")
	if err != nil || match == nil || match.Project == nil {
		return 0, fmt.Errorf("proyek tidak dikenali dari path; berikan project_id (lihat list_projects)")
	}
	return match.Project.ID, nil
}

// codeMapHandler menjalankan satu tool peta kode untuk proyek pid yang sudah dipastikan.
type codeMapHandler func(s *Server, pid uint, args map[string]interface{}) (*ToolCallResult, error)

var codeMapToolHandlers = map[string]codeMapHandler{
	"vibeasik_code_map_search":   (*Server).codeMapSearch,
	"vibeasik_code_map_overview": (*Server).codeMapOverview,
	"vibeasik_code_map_file":     (*Server).codeMapFile,
	"vibeasik_code_map_build":    (*Server).codeMapBuild,
	"vibeasik_code_map_export":   (*Server).codeMapExport,
	"vibeasik_code_map_routes":   codeMapRoutes,
	"vibeasik_code_map_cost":     codeMapCost,
	"vibeasik_code_map_status": func(s *Server, pid uint, _ map[string]interface{}) (*ToolCallResult, error) {
		return s.codeMapStatus(pid)
	},
}

// callCodeMapTool returns handled=false when name is not a code map tool.
func (s *Server) callCodeMapTool(name string, args map[string]interface{}) (*ToolCallResult, bool, error) {
	if !strings.HasPrefix(name, "vibeasik_code_map_") {
		return nil, false, nil
	}
	pid, err := s.codeMapProjectID(args)
	if err != nil {
		return nil, true, err
	}
	h, ok := codeMapToolHandlers[name]
	if !ok {
		return nil, false, nil
	}
	res, err := h(s, pid, args)
	return res, true, err
}

func (s *Server) codeMapSearch(pid uint, args map[string]interface{}) (*ToolCallResult, error) {
	q, _ := args["query"].(string)
	if strings.TrimSpace(q) == "" {
		return nil, fmt.Errorf("query is required")
	}
	kind, _ := args["kind"].(string)
	snippet, _ := args["snippet"].(bool)
	limit := int(argUint(args, "limit"))
	hits, err := codemap.Search(s.db, pid, codemap.SearchOptions{Query: q, Kind: kind, ComponentID: argUint(args, "component_id"), Limit: limit, Snippet: snippet})
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"query": q, "hits": hits})
}

func (s *Server) codeMapOverview(pid uint, args map[string]interface{}) (*ToolCallResult, error) {
	detail, _ := args["detail"].(string)
	ov, err := codemap.BuildOverview(s.db, pid, argUint(args, "component_id"), detail == "files")
	if err != nil {
		return nil, err
	}
	return jsonResult(ov)
}

func (s *Server) codeMapFile(pid uint, args map[string]interface{}) (*ToolCallResult, error) {
	file, _ := args["file"].(string)
	fd, candidates, err := codemap.File(s.db, pid, file)
	switch {
	case err != nil:
		return nil, err
	case len(candidates) > 1:
		return jsonResult(map[string]interface{}{"error": "ambiguous path, pick one", "candidates": candidates})
	case fd == nil:
		return nil, fmt.Errorf("berkas %q tidak ada di peta kode (coba vibeasik_code_map_search)", file)
	}
	return jsonResult(fd)
}

func (s *Server) codeMapBuild(pid uint, args map[string]interface{}) (*ToolCallResult, error) {
	force, _ := args["force"].(bool)
	export, _ := args["export"].(bool)
	if export && !force {
		sr, err := codemap.Sync(s.db, pid, false)
		if err != nil {
			return nil, err
		}
		return jsonResult(sr)
	}
	br, err := codemap.Build(s.db, pid, force)
	if err != nil {
		return nil, err
	}
	return jsonResult(br)
}

func (s *Server) codeMapExport(pid uint, _ map[string]interface{}) (*ToolCallResult, error) {
	er, err := codemap.Export(s.db, pid)
	if err != nil {
		return nil, err
	}
	return jsonResult(er)
}

func codeMapRoutes(s *Server, pid uint, args map[string]interface{}) (*ToolCallResult, error) {
	graphs, root, err := codemap.RouteGraphs(s.db, pid)
	if err != nil {
		return nil, err
	}
	pathQ, _ := args["path"].(string)
	method, _ := args["method"].(string)
	fn, _ := args["function"].(string)
	limit := int(argUint(args, "limit"))
	if limit <= 0 {
		limit = 20
	} else if limit > 200 {
		limit = 200
	}
	routes := routemap.FindRoutes(graphs, method, pathQ, fn)
	total := len(routes)
	if len(routes) > limit {
		routes = routes[:limit]
	}
	var warnings []string
	for _, g := range graphs {
		warnings = append(warnings, g.Warnings...)
	}
	if f, _ := args["format"].(string); f == "json" {
		return jsonResult(map[string]interface{}{"root": root, "total": total, "routes": routes, "warnings": warnings})
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d rute cocok (ditampilkan %d) — root %s\n", total, len(routes), root)
	for _, w := range warnings {
		b.WriteString("⚠ " + w + "\n")
	}
	for _, rt := range routes {
		b.WriteString("\n" + routemap.Trace(rt))
	}
	if total == 0 {
		b.WriteString("Tidak ada rute cocok. Proyek tanpa backend Go, atau filter terlalu sempit (path = substring, function = nama fungsi/berkas).\n")
	}
	return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: b.String()}}}, nil
}

// codeMapCost: rumus tick T(n), Big-O, dan query Q(n) tiap rute/fungsi — untuk membuktikan perubahan dengan angka.
func codeMapCost(s *Server, pid uint, args map[string]interface{}) (*ToolCallResult, error) {
	limit := int(argUint(args, "limit"))
	if limit <= 0 {
		limit = 20
	}
	q := insight.CostQuery{Limit: limit}
	q.Kind, _ = args["kind"].(string)
	q.Search, _ = args["q"].(string)
	q.Class, _ = args["class"].(string)
	q.NPlusOne, _ = args["n_plus_one"].(bool)
	page, err := insight.ProcessCost(s.db, pid, q, time.Now())
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	sum := page.Summary
	if !sum.Supported {
		b.WriteString("Tidak ada rute backend Go atau PHP yang bisa dihitung di proyek ini.\n")
		return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: b.String()}}}, nil
	}
	fmt.Fprintf(&b, "%d rute, %d fungsi — rute N+1 query: %d — kelas rute: %v\n", sum.Routes, sum.Funcs, sum.NPlusOneRoutes, sum.RouteClasses)
	if h := page.Handling; h.Sources > 0 {
		fmt.Fprintf(&b, "Sumber biaya tinggi: %d — di todo %d, di Audit %d, bukan masalah %d, perlu cek ulang %d, belum ditangani %d\n",
			h.Sources, h.InTodo, h.InAudit, h.Ignored, h.Recheck, h.Open)
	}
	fmt.Fprintf(&b, "%d %s cocok (ditampilkan %d), urut dari yang termahal saat data tumbuh:\n", page.Total, page.Kind, len(page.Routes)+len(page.Funcs))
	handled := func(src *insight.CostSource, h *insight.CostHandling) string {
		out := ""
		if h != nil {
			out += fmt.Sprintf("\n  ditangani (%s): %s", h.Where, h.Next)
		}
		if src != nil {
			out += fmt.Sprintf("\n  belum ditangani: sumber %s (%s:%d)", src.Func, src.File, src.Line)
		}
		return out
	}
	line := func(title, where string, c routemap.Cost) {
		flags := ""
		if c.NPlusOne {
			flags += " [N+1 QUERY]"
		}
		if c.Recursive {
			flags += " [rekursif: batas bawah]"
		}
		if c.Unbounded {
			flags += " [loop tanpa batas]"
		}
		fmt.Fprintf(&b, "\n%s\n  T(n) = %s → %s, T(1) = %d tick; Q(n) = %s%s\n  %s\n", title, c.Formula, c.BigO, c.Ticks1, c.QueryFormula, flags, where)
	}
	for _, r := range page.Routes {
		line(r.Method+" "+r.Path, "handler "+r.Handler+" ("+r.HandlerDecl+")"+handled(r.Source, r.Handled), r.Cost)
	}
	for _, f := range page.Funcs {
		line(f.Name, fmt.Sprintf("%s:%d — dijangkau %d rute", f.File, f.Line, f.Routes)+handled(f.Source, f.Handled), f.Cost)
	}
	return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: b.String()}}}, nil
}

// codeMapStatus merangkum kondisi peta proyek; peta yang sudah pernah dibangun disegarkan dulu (inkremental)
// agar hitungan berkas tanpa ringkasan LLM akurat.
func (s *Server) codeMapStatus(pid uint) (*ToolCallResult, error) {
	pending := -1
	if s.db.Where("project_id = ?", pid).First(&models.CodeMap{}).Error == nil {
		if _, n, err := codemap.EnrichCandidates(s.db, pid, 1); err == nil {
			pending = n
		}
	}
	var meta models.CodeMap
	if s.db.Where("project_id = ?", pid).First(&meta).Error != nil {
		return jsonResult(map[string]interface{}{
			"project_id": pid, "built": false,
			"hint": "Peta belum pernah dibangun: jalankan link_project_workspace (Init vibeasik) atau vibeasik_code_map_build.",
		})
	}
	var languages, kinds map[string]int
	_ = json.Unmarshal([]byte(meta.LanguagesJSON), &languages)
	_ = json.Unmarshal([]byte(meta.KindsJSON), &kinds)
	age := time.Since(meta.BuiltAt)
	out := map[string]interface{}{
		"project_id": pid, "built": true, "root_path": meta.RootPath,
		"file_count": meta.FileCount, "symbol_count": meta.SymbolCount, "enriched_count": meta.EnrichedCount,
		"languages": languages, "kinds": kinds, "build_ms": meta.BuildMs,
		"built_at": meta.BuiltAt, "age_seconds": int(age.Seconds()), "stale": age > codemap.StaleAfter,
		"initialized": meta.ExportedAt != nil, "exported_at": meta.ExportedAt,
	}
	if md := codemap.ExportedMarkdown(s.db, pid); md != "" {
		out["codemap_md"] = md
	}
	if meta.ExportedAt != nil {
		out["auto_sync"] = fmt.Sprintf("setiap %s selama `vibeasik serve` berjalan", codemap.AutoSyncInterval)
	} else {
		out["hint"] = "Belum di-init: link_project_workspace (tanpa component_id) menulis CODEMAP.md & mengaktifkan auto-sync."
	}
	if pending >= 0 {
		out["files_without_llm_summary"] = pending
	}
	return jsonResult(out)
}
