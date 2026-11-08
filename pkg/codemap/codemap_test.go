package codemap

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func setup(t *testing.T) (*gorm.DB, models.Project, string) {
	t.Helper()
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"README.md": "# Contoh\n\nAplikasi todo untuk tes peta kode.\n",
		"be/go.mod": "module example.com/be\n\ngo 1.22\n",
		"be/internal/models/todo.go": `// Package models berisi entitas domain todo.
package models

// Todo adalah satu tugas.
type Todo struct{ ID uint }

// EstimateImpact menaksir dampak todo.
func EstimateImpact(t *Todo) int { return 1 }
`,
		"be/internal/api/todo_handler.go": `package api

import (
	"example.com/be/internal/models"
)

func (s *Server) createTodo(c *Ctx) error { _ = models.Todo{}; return nil }

func register(api *Router) {
	api.Post("/todos", s.createTodo)
	api.Get("/todos/:id", s.getTodo)
}
`,
		"be/internal/api/todo_handler_test.go": "package api\n\nfunc TestCreateTodo(t *testing.T) {}\n",
		"fe/nuxt.config.ts":                    "export default defineNuxtConfig({})\n",
		"fe/app/composables/useTodoImpact.ts": `import type { Todo } from '~/types'

/** Definisi dimensi dampak todo. */
export const useTodoImpact = () => {
  return $fetch(` + "`${apiBase}/api/todos/impact-dimensions`" + `)
}
`,
		"fe/app/types/index.ts": "export interface Todo { id: number }\n",
		"fe/app/components/todos/ImpactBadge.vue": `<script setup lang="ts">
import type { Todo } from '~/types'
/** Chip skor dampak todo. */
const props = defineProps<{ todo: Todo }>()
const { info } = useTodoImpact()
</script>
<template><span>{{ info }}</span></template>
`,
		"fe/app/pages/todos/[id].vue": "<template><TodosImpactBadge :todo=\"t\" /></template>\n",
		"fe/node_modules/x/index.js":  "export const ignored = 1\n",
	})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}, &models.ComponentRoute{}, &models.CodeMap{}, &models.CodeMapEntry{}); err != nil {
		t.Fatal(err)
	}
	p := models.Project{Name: "contoh", Slug: "contoh", RootPath: root, Components: []models.Component{
		{Name: "BE", Role: "backend", AbsolutePath: filepath.Join(root, "be")},
		{Name: "FE", Role: "frontend_admin", AbsolutePath: filepath.Join(root, "fe")},
	}}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	return db, p, root
}

func TestBuildResolvesImportsAndDependents(t *testing.T) {
	db, p, _ := setup(t)
	res, err := Build(db, p.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.FileCount != 10 || res.Parsed != 10 {
		t.Fatalf("expected 10 files parsed (node_modules ignored), got %+v", res)
	}
	model, _, err := File(db, p.ID, "be/internal/models/todo.go")
	if err != nil || model == nil {
		t.Fatalf("model file missing: %v", err)
	}
	if model.Kind != "model" || model.Summary != "Package models berisi entitas domain todo." || len(model.UsedBy) != 1 || model.UsedBy[0] != "be/internal/api/todo_handler.go" {
		t.Fatalf("Go package import must link handler → model: %+v", model)
	}
	badge, _, _ := File(db, p.ID, "ImpactBadge.vue")
	if badge == nil || badge.ComponentName != "FE" || badge.Summary != "Chip skor dampak todo." {
		t.Fatalf("suffix lookup + vue summary failed: %+v", badge)
	}
	wantImports := map[string]bool{"fe/app/types/index.ts": true, "fe/app/composables/useTodoImpact.ts": true}
	for _, imp := range badge.Imports {
		delete(wantImports, imp)
	}
	if len(wantImports) > 0 || len(badge.UsedBy) != 1 || badge.UsedBy[0] != "fe/app/pages/todos/[id].vue" {
		t.Fatalf("nuxt ~/ + auto-import composable/component must resolve: imports=%v usedBy=%v", badge.Imports, badge.UsedBy)
	}
	page, _, _ := File(db, p.ID, "fe/app/pages/todos/[id].vue")
	if page == nil || len(page.Routes) == 0 || page.Routes[0] != "PAGE /todos/:id" {
		t.Fatalf("nuxt page route expected: %+v", page)
	}
	handler, _, _ := File(db, p.ID, "be/internal/api/todo_handler.go")
	if handler == nil || handler.Kind != "handler" || strings.Join(handler.Routes, ",") != "POST /todos,GET /todos/:id" {
		t.Fatalf("fiber routes expected: %+v", handler)
	}
}

func TestSearchRanksSymbolsAndSynonyms(t *testing.T) {
	db, p, _ := setup(t)
	if _, err := Build(db, p.ID, false); err != nil {
		t.Fatal(err)
	}
	hits, err := Search(db, p.ID, SearchOptions{Query: "estimate impact", Snippet: true})
	if err != nil || len(hits) == 0 {
		t.Fatalf("no hits: %v", err)
	}
	if hits[0].Path != "be/internal/models/todo.go" || len(hits[0].Symbols) == 0 || hits[0].Symbols[0].Name != "EstimateImpact" || !strings.Contains(hits[0].Snippet, "func EstimateImpact") {
		t.Fatalf("camelCase symbol should rank first with snippet: %+v", hits[0])
	}
	hits, _ = Search(db, p.ID, SearchOptions{Query: "dampak"})
	if len(hits) == 0 {
		t.Fatal("Indonesian synonym 'dampak' should find impact files")
	}
	hits, _ = Search(db, p.ID, SearchOptions{Query: "/api/todos/impact-dimensions"})
	if len(hits) == 0 || hits[0].Path != "fe/app/composables/useTodoImpact.ts" {
		t.Fatalf("API call path should find the calling composable: %+v", hits)
	}
	hits, _ = Search(db, p.ID, SearchOptions{Query: "todo", Kind: "test"})
	if len(hits) != 1 || hits[0].Kind != "test" {
		t.Fatalf("kind filter: %+v", hits)
	}
}

func TestIncrementalBuildAndExport(t *testing.T) {
	db, p, root := setup(t)
	if _, err := Build(db, p.ID, false); err != nil {
		t.Fatal(err)
	}
	res, _ := Build(db, p.ID, false)
	if res.Parsed != 0 || res.Unchanged != 10 {
		t.Fatalf("second build should reuse entries: %+v", res)
	}
	later := time.Now().Add(time.Minute)
	writeFiles(t, root, map[string]string{"be/internal/models/todo.go": "package models\n\n// Todo baru.\ntype Todo struct{}\n"})
	_ = os.Chtimes(filepath.Join(root, "be/internal/models/todo.go"), later, later)
	_ = os.Remove(filepath.Join(root, "README.md"))
	res, _ = Build(db, p.ID, false)
	if res.Parsed != 1 || res.Removed != 1 || res.FileCount != 9 {
		t.Fatalf("changed file reparsed, deleted file removed: %+v", res)
	}
	ov, err := BuildOverview(db, p.ID, 0, false)
	if err != nil || ov.FileCount != 9 || len(ov.Components) != 2 {
		t.Fatalf("overview: %+v %v", ov, err)
	}
	exp, err := Export(db, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	md, _ := os.ReadFile(exp.MarkdownPath)
	jl, _ := os.ReadFile(exp.JSONLPath)
	if !strings.Contains(string(md), "ImpactBadge.vue") || strings.Count(string(jl), "\n") != 9 {
		t.Fatalf("export content unexpected:\n%s", md)
	}
	// Berkas ekspor di .vibeasik tidak boleh ikut terindeks.
	res, _ = Build(db, p.ID, false)
	if res.FileCount != 9 {
		t.Fatalf(".vibeasik must be ignored: %+v", res)
	}
}

func TestNuxtName(t *testing.T) {
	cases := map[string]string{"todos/TodoCard": "TodosTodoCard", "orchestrator/OrchestratorRunList": "OrchestratorRunList", "base/app-button": "BaseAppButton", "Foo/index": "Foo"}
	for in, want := range cases {
		if got := nuxtName(in); got != want {
			t.Errorf("nuxtName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGenericRootUsesComponentAncestor(t *testing.T) {
	docs := t.TempDir()
	writeFiles(t, docs, map[string]string{
		"README.md":              "# Folder umum\n",
		"other-project/main.go":  "package main\n",
		"myapp/AI_CONTEXT.md":    "# Konteks\n\nAturan.\n",
		"myapp/be/main.go":       "package main\n\nfunc main() {}\n",
		"myapp/fe/app.vue":       "<template><div /></template>\n",
		"myapp/nested/.git/HEAD": "ref\n",
		"myapp/nested/skip.go":   "package nested\n",
	})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	_ = db.AutoMigrate(&models.Project{}, &models.Component{}, &models.ComponentRoute{}, &models.CodeMap{}, &models.CodeMapEntry{})
	p := models.Project{Name: "bundel", Slug: "bundel", RootPath: docs, Components: []models.Component{
		{Name: "BE", AbsolutePath: filepath.Join(docs, "myapp/be")}, {Name: "FE", AbsolutePath: filepath.Join(docs, "myapp/fe")}}}
	db.Create(&p)
	res, err := Build(db, p.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.FileCount != 3 {
		t.Fatalf("only myapp files (no sibling project, no nested repo) expected: %+v", res)
	}
	ov, _ := BuildOverview(db, p.ID, 0, false)
	if ov.RootPath != filepath.Join(docs, "myapp") || len(ov.Docs) != 1 || ov.Docs[0].Path != "AI_CONTEXT.md" {
		t.Fatalf("effective root should be the components' ancestor: %+v", ov)
	}
}

func TestParsePHPRoutesAndSymbols(t *testing.T) {
	src := "<?php\nuse App\\Models\\UserModel;\n\n$routes->group('api', static function ($routes) {\n    $routes->get('kontingen', 'KontingenController::index');\n    $routes->post('/auth/login', 'Api\\AuthController::login');\n});\n\nclass Routes {\n    public function boot() {}\n}\n"
	p := parseSource("php", "app/Config/Routes.php", src)
	if strings.Join(p.routes, ",") != "GET /kontingen,POST /auth/login" {
		t.Fatalf("ci4 routes: %v", p.routes)
	}
	want := map[string]bool{"php:App\\Models\\UserModel": true, "php:KontingenController": true, "php:Api\\AuthController": true}
	for _, r := range p.refs {
		delete(want, r)
	}
	if len(want) > 0 || len(p.symbols) != 2 {
		t.Fatalf("refs missing %v, symbols %+v", want, p.symbols)
	}
}

func TestEnsureFresh(t *testing.T) {
	db, p, _ := setup(t)

	// Panggilan pertama membangun code map
	if err := EnsureFresh(db, p.ID, time.Minute); err != nil {
		t.Fatalf("EnsureFresh gagal saat initial build: %v", err)
	}

	// Panggilan kedua saat masih segar (< maxAge) tidak mengembalikan error
	if err := EnsureFresh(db, p.ID, time.Minute); err != nil {
		t.Fatalf("EnsureFresh gagal saat cache hit: %v", err)
	}

	// Panggilan untuk project tidak ada harus mengembalikan error yang dibungkus (K1651)
	nonExistentID := uint(999999)
	err := EnsureFresh(db, nonExistentID, time.Minute)
	if err == nil {
		t.Fatal("ekspektasi error untuk proyek yang tidak ada, dapat nil")
	}
	expectedMsg := "gagal memperbarui code map proyek 999999"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Fatalf("error tidak memuat pesan pembungkus %q, didapat: %v", expectedMsg, err)
	}
}

func TestExistsPredicateContract(t *testing.T) {
	// 1. Berkas ada -> return true, tidak ada log peringatan
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("ok"), 0o644); err != nil {
		t.Fatalf("gagal membuat berkas uji: %v", err)
	}

	var logBuf bytes.Buffer
	origWriter := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(origWriter)

	if !exists(existingFile) {
		t.Fatalf("exists(%q) = false, ekspektasi true untuk berkas yang ada", existingFile)
	}
	if logBuf.Len() > 0 {
		t.Fatalf("ekspektasi tidak ada log pada berkas yang ada, didapat: %s", logBuf.String())
	}

	// 2. Berkas tidak ada (os.IsNotExist) -> return false tanpa error/panic dan tanpa log [WARN] (jalur normal predikat)
	logBuf.Reset()
	missingFile := filepath.Join(tmpDir, "missing.txt")
	if exists(missingFile) {
		t.Fatalf("exists(%q) = true, ekspektasi false untuk berkas yang tidak ada", missingFile)
	}
	if logBuf.Len() > 0 {
		t.Fatalf("ekspektasi tidak ada log pada IsNotExist (jalur normal), didapat: %s", logBuf.String())
	}

	// 3. Error sistem non-IsNotExist (misal akses izin terlarang 0000 di Unix)
	// -> return false dan log menangkap peringatan [WARN]
	if os.Geteuid() != 0 {
		logBuf.Reset()
		forbiddenDir := filepath.Join(tmpDir, "forbidden")
		if err := os.Mkdir(forbiddenDir, 0o700); err != nil {
			t.Fatalf("gagal membuat direktori uji: %v", err)
		}
		if err := os.Chmod(forbiddenDir, 0o000); err != nil {
			t.Fatalf("gagal chmod direktori uji: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(forbiddenDir, 0o700)
		})

		unreadablePath := filepath.Join(forbiddenDir, "unreachable.txt")
		got := exists(unreadablePath)
		if got {
			t.Fatalf("exists(%q) = true, ekspektasi false untuk path dengan permission denied", unreadablePath)
		}
		logOutput := logBuf.String()
		if !strings.Contains(logOutput, "[WARN] exists: os.Stat") {
			t.Fatalf("ekspektasi log memuat '[WARN] exists: os.Stat', didapat: %q", logOutput)
		}
	}
}

