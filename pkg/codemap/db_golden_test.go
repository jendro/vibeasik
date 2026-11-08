package codemap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

var volatileKeys = map[string]bool{"id": true, "ID": true, "built_at": true, "exported_at": true, "duration_ms": true,
	"updated_at": true, "created_at": true, "BuiltAt": true, "build_ms": true, "BuildMs": true, "UpdatedAt": true}

func scrub(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		for k := range x {
			if volatileKeys[k] {
				delete(x, k)
				continue
			}
			x[k] = scrub(x[k])
		}
	case []interface{}:
		for i := range x {
			x[i] = scrub(x[i])
		}
	}
	return v
}

func scrubJSON(v interface{}, root string) string {
	b, _ := json.Marshal(v)
	var g interface{}
	_ = json.Unmarshal([]byte(strings.ReplaceAll(string(b), root, "<ROOT>")), &g)
	out, _ := json.Marshal(scrub(g))
	return string(out)
}

var dateRe = regexp.MustCompile(`pada \d{4}-\d{2}-\d{2} \d{2}:\d{2}`)

func goldenFixture() map[string]string {
	return map[string]string{
		"README.md":          "# Proyek\n\nAplikasi contoh untuk golden.\n",
		"AI_CONTEXT.md":      "# Konteks\n\nAturan kerja.\n",
		"docs/guide.md":      "# Panduan\n\nIsi.\n",
		"Caddyfile":          "example.com {\n}\n",
		"docker-compose.yml": "services:\n  a: {}\n",
		"be/go.mod":          "module example.com/be\n\ngo 1.22\n",
		"be/main.go":         "package main\n\nimport (\n\t\"example.com/be/internal/api\"\n)\n\nfunc main() { api.Register(nil) }\n",
		"be/internal/models/todo.go": `// Package models berisi entitas.
package models

// Todo adalah satu tugas.
type Todo struct{ ID uint }

// EstimateImpact menaksir dampak todo.
func EstimateImpact(t *Todo) int { return 1 }

func (t *Todo) Title() string { return "" }
`,
		"be/internal/models/user.go": "package models\n\ntype User struct{}\n\nfunc NewUser() *User { return nil }\n",
		"be/internal/api/todo_handler.go": `package api

import (
	"example.com/be/internal/models"
	"example.com/be/internal/service"
	m2 "example.com/be/internal/models"
)

func createTodo(c *Ctx) error { _ = models.Todo{}; _ = service.PublishTodo(); _ = m2.NewUser(); return nil }

func Register(api *Router) {
	api.Post("/todos", createTodo)
	api.Get("/todos/:id", getTodo)
}
`,
		"be/internal/api/admin_todo.go":                   "package api\n\nimport \"example.com/be/internal/service\"\n\nfunc AdminCreateTodo() { service.PublishTodo() }\nfunc ListTodos() {}\nfunc RegisterRoutes() {}\n",
		"be/internal/api/todo_handler_test.go":            "package api\n\nfunc TestCreateTodo(t *testing.T) {}\n",
		"be/internal/service/todo_service.go":             "package service\n\nimport \"example.com/be/internal/repository\"\n\nfunc PublishTodo() error { return repository.SaveTodo() }\nfunc CreateTodo() {}\n",
		"be/internal/repository/todo_repo.go":             "package repository\n\nfunc SaveTodo() error { return nil }\nfunc ListTodos() {}\n",
		"be/internal/util/helpers.go":                     "package util\n\nfunc Paginate(q int) int { return q }\n",
		"be/migrations/001_init.sql":                      "CREATE TABLE todos (id int);\n",
		"be/vendor/x/y.go":                                "package y\n",
		"be/api.pb.go":                                    "package be\n",
		"fe/nuxt.config.ts":                               "export default defineNuxtConfig({})\n",
		"fe/package.json":                                 "{\"name\":\"fe\"}\n",
		"fe/app/app.vue":                                  "<template><NuxtPage /></template>\n",
		"fe/app/composables/useTodoImpact.ts":             "import type { Todo } from '~/types'\n\n/** Definisi dimensi dampak todo. */\nexport const useTodoImpact = () => {\n  return $fetch(`${apiBase}/api/todos/impact-dimensions`)\n}\n",
		"fe/app/composables/useAuth.ts":                   "export function useAuth() { return $fetch('/api/auth/login') }\n",
		"fe/app/types/index.ts":                           "export interface Todo { id: number }\n",
		"fe/app/stores/todo.ts":                           "import { useTodoImpact } from '~/composables/useTodoImpact'\nexport const useTodoStore = defineStore('todo', () => { useTodoImpact() })\n",
		"fe/app/components/todos/ImpactBadge.vue":         "<script setup lang=\"ts\">\nimport type { Todo } from '~/types'\n/** Chip skor dampak todo. */\nconst props = defineProps<{ todo: Todo }>()\nconst { info } = useTodoImpact()\n</script>\n<template><span>{{ info }}</span></template>\n",
		"fe/app/components/todos/todo-card.vue":           "<template><div /></template>\n",
		"fe/app/pages/index.vue":                          "<template><TodosTodoCard /></template>\n",
		"fe/app/pages/todos/[id].vue":                     "<template><TodosImpactBadge :todo=\"t\" /></template>\n<script setup>\nconst a = useAuth()\n</script>\n",
		"fe/app/layouts/default.vue":                      "<template><slot /></template>\n",
		"fe/app/middleware/auth.ts":                       "export default defineNuxtRouteMiddleware(() => {})\n",
		"fe/app/utils/format.ts":                          "export function fmt(a: number) { return a }\nimport { x } from '@/types'\nimport z from '../types/index'\n",
		"fe/tests/a.spec.ts":                              "import { fmt } from '~/utils/format'\n",
		"fe/node_modules/x/index.js":                      "export const ignored = 1\n",
		"fe/.nuxt/skip.ts":                                "export const s = 1\n",
		"fe/dist/bundle.min.js":                           "var a=1\n",
		"fe/app/styles/main.css":                          ".root { color: red }\n#id { }\n",
		"py/pyproject.toml":                               "[tool]\n",
		"py/app/__init__.py":                              "",
		"py/app/main.py":                                  "\"\"\"Aplikasi py.\"\"\"\nfrom app.util import helper\nfrom . import sibling\nimport os\n\n@app.get(\"/py\")\ndef index(): pass\n",
		"py/app/util.py":                                  "def helper(): pass\nclass _Hidden: pass\n",
		"py/app/sibling.py":                               "def s(): pass\n",
		"php/composer.json":                               "{\"require\":{}}\n",
		"php/app/Controllers/Api/KontingenController.php": "<?php\nnamespace App\\Controllers\\Api;\nuse App\\Models\\KontingenModel;\nclass KontingenController { public function index() {} }\n",
		"php/app/Models/KontingenModel.php":               "<?php\nclass KontingenModel {}\n",
		"php/app/Config/Routes.php":                       "<?php\n$routes->get('kontingen', 'Api\\KontingenController::index');\n$routes->post('/login', 'AuthController::login');\n",
		"dart/pubspec.yaml":                               "name: myapp\n",
		"dart/lib/main.dart":                              "import 'package:myapp/src/a.dart';\nimport './src/b.dart';\nimport 'dart:io';\nvoid main() {\n}\n",
		"dart/lib/src/a.dart":                             "class A {}\n",
		"dart/lib/src/b.dart":                             "class B {}\n",
		"jv/src/main/java/com/x/UserCtl.java":             "package com.x;\nimport com.x.Repo;\npublic class UserCtl {\n  @GetMapping(\"/u\")\n  public String list() { return \"\"; }\n}\n",
		"jv/src/main/java/com/x/Repo.java":                "package com.x;\ninterface Repo {}\n",
		"jv/src/main/kotlin/com/x/Svc.kt":                 "package com.x\nimport com.x.Repo\nclass Svc\nfun top() {}\n",
	}
}

func setupGolden(t *testing.T) (*gorm.DB, models.Project, string) {
	t.Helper()
	root := t.TempDir()
	writeFiles(t, root, goldenFixture())
	big := strings.Repeat("// filler\n", 50)
	writeFiles(t, root, map[string]string{"be/internal/api/big_handler.go": "package api\n\n// Besar.\nfunc Big() {}\n" + strings.Repeat("func F() {}\n", 450) + big})
	fixed := time.Unix(1700000000, 0)
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			_ = os.Chtimes(p, fixed, fixed)
		}
		return nil
	})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}, &models.ComponentRoute{}, &models.CodeMap{}, &models.CodeMapEntry{}); err != nil {
		t.Fatal(err)
	}
	p := models.Project{Name: "golden", Slug: "golden", RootPath: root, Components: []models.Component{
		{Name: "BE", Role: "backend", AbsolutePath: filepath.Join(root, "be")},
		{Name: "FE", Role: "frontend_admin", AbsolutePath: filepath.Join(root, "fe")},
		{Name: "PY", Role: "backend", AbsolutePath: filepath.Join(root, "py")},
		{Name: "PHP", Role: "backend", AbsolutePath: filepath.Join(root, "php")},
		{Name: "APP", Role: "mobile", AbsolutePath: filepath.Join(root, "dart")},
		{Name: "JV", Role: "backend", AbsolutePath: filepath.Join(root, "jv")},
		{Name: "GONE", Role: "backend", AbsolutePath: filepath.Join(root, "missing")},
	}}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	return db, p, root
}

func dumpEntries(db *gorm.DB, pid uint) []map[string]interface{} {
	var es []models.CodeMapEntry
	db.Where("project_id = ?", pid).Order("path").Find(&es)
	var out []map[string]interface{}
	for _, e := range es {
		out = append(out, map[string]interface{}{"e": e, "sym": e.SymbolsJSON, "imp": e.ImportsJSON, "refs": e.RefsJSON, "routes": e.RoutesJSON, "ai": e.AIHash})
	}
	return out
}

func TestCodemapDBGolden(t *testing.T) {
	db, p, root := setupGolden(t)
	got := map[string]string{}
	rec := func(name string, v interface{}) { name = strings.ReplaceAll(name, root, "<ROOT>"); got[name] = hash8(scrubJSON(v, root)) }

	res, err := Build(db, p.ID, false)
	rec("build/first", res)
	rec("build/err", fmt.Sprint(err))
	rec("entries/first", dumpEntries(db, p.ID))
	var meta models.CodeMap
	db.Where("project_id = ?", p.ID).First(&meta)
	rec("meta", meta)
	res, _ = Build(db, p.ID, false)
	rec("build/second", res)
	res, _ = Build(db, p.ID, true)
	rec("build/force", res)
	rec("entries/force", dumpEntries(db, p.ID))
	rec("effroot", EffectiveRoot(func() models.Project { var pr models.Project; db.Preload("Components").First(&pr, p.ID); return pr }()))

	var paths []string
	var es []models.CodeMapEntry
	db.Where("project_id = ?", p.ID).Order("path").Find(&es)
	for _, e := range es {
		paths = append(paths, e.Path)
	}
	for _, pa := range paths {
		fd, cands, err := File(db, p.ID, pa)
		rec("file/"+pa, []interface{}{fd, cands, fmt.Sprint(err)})
		for _, withInv := range []bool{false, true} {
			rec(fmt.Sprintf("layer/%s/%v", pa, withInv), BuildLayerGuide(db, p.ID, pa, withInv))
		}
	}
	for _, f := range []string{"ImpactBadge.vue", "index.ts", "main.go", "todo_handler.go", "missing.go", "", "./be/main.go", "be\\main.go", filepath.Join(root, "be/main.go"), "types/index.ts", "handlers/x.go", "internal/api/admin_todo.go", "api/todo_handler.go", "service/todo_service.go"} {
		fd, cands, err := File(db, p.ID, f)
		rec("filelookup/"+f, []interface{}{fd, cands, fmt.Sprint(err)})
		for _, inv := range []bool{false, true} {
			rec(fmt.Sprintf("layerlookup/%s/%v", f, inv), BuildLayerGuide(db, p.ID, f, inv))
		}
	}
	rec("layer/nil-db", BuildLayerGuide(nil, p.ID, "be/main.go", true))
	rec("layer/zero", BuildLayerGuide(db, 0, "be/main.go", true))
	rec("layer/blank", BuildLayerGuide(db, p.ID, "  ", true))
	for _, g := range []*LayerGuide{BuildLayerGuide(db, p.ID, "be/internal/api/todo_handler.go", true), BuildLayerGuide(db, p.ID, "fe/app/pages/index.vue", false), nil} {
		rec(fmt.Sprintf("prompt/%v", g != nil && g.Target != ""), g.Prompt()+"|"+fmt.Sprint(g != nil && g.HasDir("be/internal/api"))+"|"+func() string {
			if g == nil {
				return ""
			}
			return g.MainDir(LayerService) + g.MainDir(LayerHandler)
		}())
	}

	queries := []string{"estimate impact", "dampak", "/api/todos/impact-dimensions", "todo", "hapus todo", "delete todo", "createTodo", "login", "ImpactBadge",
		"nuxt page", "kontingen", "routes", "", "   ", "zzzzqqq", "test", "publish todo service", "paginate", "orchestr", "main", "user", "pengguna", "simpan",
		"CreateTodo", "be/internal/api", "helper", "a", "the and for", "todo todo todo", "TodoCard", "config", "migrations"}
	for _, q := range queries {
		for _, o := range []SearchOptions{{Query: q}, {Query: q, Snippet: true, Limit: 3}, {Query: q, Kind: "handler"}, {Query: q, Kind: "test", Limit: 100},
			{Query: q, ComponentID: p.Components[1].ID, Snippet: true}, {Query: q, Limit: 1}} {
			o := o
			hits, err := Search(db, p.ID, o)
			rec(fmt.Sprintf("search/%q/%q/%d/%v/%d", q, o.Kind, o.ComponentID, o.Snippet, o.Limit), []interface{}{hits, fmt.Sprint(err)})
		}
	}
	fe := p.Components[1].ID
	for _, cid := range []uint{0, fe, 999} {
		for _, wf := range []bool{false, true} {
			ov, err := BuildOverview(db, p.ID, cid, wf)
			rec(fmt.Sprintf("overview/%d/%v", cid, wf), []interface{}{ov, fmt.Sprint(err)})
		}
	}
	for _, lim := range []int{0, 3, 100} {
		c, total, err := EnrichCandidates(db, p.ID, lim)
		rec(fmt.Sprintf("enrich/%d", lim), []interface{}{c, total, fmt.Sprint(err)})
	}
	// ringkasan LLM sebagian
	for i, e := range es {
		if i%4 == 0 {
			SaveEnrichment(db, p.ID, e.Path, e.ContentHash, "Ringkasan AI untuk "+e.Path, []string{"alpha", " beta "})
		}
	}
	RefreshEnrichedCount(db, p.ID)
	c, total, err := EnrichCandidates(db, p.ID, 5)
	rec("enrich/after", []interface{}{c, total, fmt.Sprint(err)})
	hits, _ := Search(db, p.ID, SearchOptions{Query: "alpha beta ringkasan"})
	rec("search/after-enrich", hits)
	ov, _ := BuildOverview(db, p.ID, 0, true)
	rec("overview/after-enrich", ov)
	res, _ = Build(db, p.ID, false)
	rec("build/after-enrich", res)

	// perubahan berkas: ubah, sentuh (hash sama), hapus, tambah
	later := time.Unix(1700001000, 0)
	writeFiles(t, root, map[string]string{
		"be/internal/models/todo.go": "package models\n\n// Todo baru.\ntype Todo struct{}\n",
		"fe/app/utils/format.ts":     "export function fmt(a: number) { return a }\nimport { x } from '@/types'\nimport z from '../types/index'\n",
		"be/extra.go":                "package main\n\nfunc Extra() {}\n",
	})
	_ = os.Chtimes(filepath.Join(root, "be/internal/models/todo.go"), later, later)
	_ = os.Chtimes(filepath.Join(root, "fe/app/utils/format.ts"), later, later)
	_ = os.Chtimes(filepath.Join(root, "be/extra.go"), later, later) // waktu tetap agar golden stabil
	_ = os.Remove(filepath.Join(root, "README.md"))
	res, err = Build(db, p.ID, false)
	rec("build/changed", []interface{}{res, fmt.Sprint(err)})
	rec("entries/changed", dumpEntries(db, p.ID))
	db.Model(&models.CodeMap{}).Where("project_id = ?", p.ID).Update("parser_version", 0)
	res, _ = Build(db, p.ID, false)
	rec("build/parserversion", res)

	exp, err := Export(db, p.ID)
	rec("export/res", []interface{}{exp, fmt.Sprint(err)})
	if exp != nil {
		for _, f := range []string{exp.MarkdownPath, exp.JSONLPath, exp.RoutesPath} {
			if f == "" {
				continue
			}
			b, _ := os.ReadFile(f)
			s := dateRe.ReplaceAllString(string(b), "pada DATE")
			rec("export/file/"+filepath.Base(f), s)
		}
	}
	ov, _ = BuildOverview(db, p.ID, 0, false)
	rec("overview/after-export", ov)

	// proyek tanpa direktori / tanpa root
	empty := models.Project{Name: "kosong", Slug: "kosong", RootPath: "/nonexistent-dir-xyz"}
	db.Create(&empty)
	_, err = Build(db, empty.ID, false)
	rec("build/empty-err", fmt.Sprint(err))
	_, err = Build(db, 99999, false)
	rec("build/notfound", fmt.Sprint(err != nil))
	rec("effroot/empty", EffectiveRoot(empty))

	checkGolden(t, "db_golden.json", got)
	_ = sort.Strings
}

func TestResolverGolden(t *testing.T) {
	paths := []string{
		"be/main.go", "be/internal/models/todo.go", "be/internal/api/h.go", "be/internal/api/h2.go", "main.go",
		"fe/app/pages/index.vue", "fe/app/composables/useThing.ts", "fe/app/components/todos/TodoCard.vue", "fe/app/components/base/app-button.vue",
		"fe/app/components/Orchestrator/OrchestratorRunList.vue", "fe/app/components/Foo/index.vue", "fe/app/utils/a.ts", "fe/app/utils/index.ts",
		"fe/src/lib/b.ts", "fe/c.ts", "fe/app/types/index.ts", "fe/app/x.tsx", "fe/app/y.mjs",
		"py/app/__init__.py", "py/app/main.py", "py/app/util/__init__.py", "py/app/util/helper.py", "py/other.py",
		"php/app/Models/User.php", "php/app/Controllers/Api/KontingenController.php",
		"jv/src/main/java/com/x/UserCtl.java", "jv/src/main/kotlin/com/x/Svc.kt", "jv/src/main/java/com/x/Both.java", "jv/src/main/kotlin/com/x/Both.kt",
		"dart/lib/main.dart", "dart/lib/src/a.dart", "dart/lib/src/b.dart",
	}
	roots := []rootInfo{
		{relBase: "be", goModule: "example.com/be"}, {relBase: "fe", srcDir: "app"}, {relBase: "py"}, {relBase: "php"}, {relBase: "dart", dartPkg: "myapp"},
		{relBase: "jv"}, {relBase: "", goModule: "example.com/root"}, {relBase: "fe/nested", srcDir: ""},
	}
	rs := newResolver(paths, roots)
	froms := []string{"be/main.go", "be/internal/api/h.go", "fe/app/pages/index.vue", "fe/app/components/todos/TodoCard.vue", "fe/c.ts", "py/app/main.py",
		"py/app/util/helper.py", "php/app/Controllers/Api/KontingenController.php", "jv/src/main/java/com/x/UserCtl.java", "dart/lib/main.dart", "main.go", "fe/nested/z.ts", "other/x.txt"}
	specs := []string{"auto:useThing", "auto:useMissing", "auto:TodosTodoCard", "auto:TodoCard", "auto:LazyTodoCard", "auto:Missing", "auto:BaseAppButton", "auto:OrchestratorRunList",
		"auto:Foo", "py:app.util", "py:app.util.helper", "py:.util", "py:..other", "py:.", "py:app", "py:other", "py:missing", "php:App\\Models\\User", "php:App/Models/User",
		"php:Models\\User", "php:Nope", "php:Api\\KontingenController", "jvm:com.x.UserCtl", "jvm:com.x.Svc", "jvm:com.x.Both", "jvm:com.y.Nope", "dart:package:myapp/src/a.dart",
		"dart:package:other/src/a.dart", "dart:package:myapp", "dart:dart:io", "dart:./src/b.dart", "dart:src/a.dart", "dart:missing.dart", "./h2", "./h2.go", "../models/todo",
		"../models/todo.go", "./utils/a", "~/utils/a", "@/utils/a", "~/lib/b", "~/c", "~/types", "~~/c", "@@/c", "~~/app/utils/a", "example.com/be/internal/models", "example.com/be",
		"example.com/root/be", "example.com/root", "example.com/other", "lodash", "react", "", "./missing", "~/missing"}
	got := map[string]string{}
	var sb strings.Builder
	for _, f := range froms {
		for _, s := range specs {
			fmt.Fprintf(&sb, "%s|%s=%s\n", f, s, rs.resolve(f, s))
		}
	}
	got["resolve"] = hash8(sb.String())
	b, _ := json.Marshal([]interface{}{rs.files, rs.dirs, rs.nuxtComps, rs.composable, rs.suffix})
	got["newresolver"] = hash8(string(b))
	var nb strings.Builder
	for _, s := range []string{"todos/TodoCard", "orchestrator/OrchestratorRunList", "base/app-button", "Foo/index", "index", "a/b/c", "a_b.c d/e", "", "x/x", "Todo/todo-card"} {
		fmt.Fprintf(&nb, "%s=%s;", s, nuxtName(s))
	}
	got["nuxtName"] = hash8(nb.String())
	var tb strings.Builder
	for _, s := range []string{"createTodoItem", "HTTPServer2go", "snake_case_name a1b2 X", "ünïcode Wörd", "The and for", "", "ab", "a", "IDs", "getHTTPResponse", "v2Api", "ABC", "aB", "A1", "1A", "file.name/with-parts"} {
		fmt.Fprintf(&tb, "%s=%v;", s, tokenize(s))
		fmt.Fprintf(&tb, "q=%v;", queryConcepts(s))
	}
	got["tokenize"] = hash8(tb.String())
	checkGolden(t, "resolver_golden.json", got)
}
