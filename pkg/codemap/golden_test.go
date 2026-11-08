package codemap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "tulis ulang testdata/*.json")

func hash8(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// checkGolden membandingkan peta nama→hash dengan berkas testdata (atau menulisnya dengan -update-golden).
func checkGolden(t *testing.T, file string, got map[string]string) {
	t.Helper()
	path := filepath.Join("testdata", file)
	if *updateGolden {
		_ = os.MkdirAll("testdata", 0o755)
		b, _ := json.MarshalIndent(got, "", " ")
		if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]string
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	var bad []string
	for k, v := range got {
		if want[k] != v {
			bad = append(bad, k)
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 || len(want) != len(got) {
		t.Fatalf("%s: %d berubah (want %d, got %d): %v", file, len(bad), len(want), len(got), bad[:min(len(bad), 10)])
	}
}

func TestClassifyGolden(t *testing.T) {
	dirs := []string{"", "pages", "views", "screens", "app/routes", "layouts", "composables", "hooks", "stores", "store", "components",
		"widgets", "models", "model", "entities", "entity", "schemas", "types", "middleware", "handlers", "controllers", "server/api",
		"routes", "services", "service", "usecase", "usecases", "repository", "repositories", "cmd", "utils", "helpers", "lib", "pkg",
		"config", "scripts", "migrations", "migration", "tests", "__tests__", "x", "Pages/Sub", "src/Components"}
	files := []string{"a.go", "a_test.go", "a.test.ts", "a.spec.ts", "test_a.py", "main.go", "main.dart", "main.py", "app.vue", "index.php",
		"main.ts", "a.tsx", "a.jsx", "a.vue", "a.d.ts", "types.ts", "useFoo.ts", "usea.ts", "use.ts", "useFoo", "user.go", "authMiddleware.go",
		"guard.ts", "handler.go", "controller.go", "route.go", "router.ts", "service.go", "repository.go", "repo.go", "util.go", "helpers.go",
		"config.go", "settings.py", "package.json", "go.mod", "nuxt.config.ts", "vite.config.js", "tailwind.config.js", "next.config.mjs",
		"a_test.dart", "a.css", "a.sql", "README.md", "a.sh", "Makefile", "Dockerfile", "a.dart", "Test.java", "UserTest.kt", "x_test.dart",
		"Dockerfile", ".env.example", "tsconfig.json", "usage.go", "UseCase.ts"}
	langs := []string{"go", "typescript", "vue", "svelte", "markdown", "sql", "css", "dart", "shell", "python", "php", ""}
	var sb strings.Builder
	h := sha256.New()
	for _, d := range dirs {
		for _, f := range files {
			p := f
			if d != "" {
				p = d + "/" + f
			}
			for _, l := range langs {
				sb.Reset()
				fmt.Fprintf(&sb, "%s|%s=%s\n", p, l, classify(p, l))
				h.Write([]byte(sb.String()))
			}
		}
	}
	got := map[string]string{"classify": hex.EncodeToString(h.Sum(nil))[:16]}
	checkGolden(t, "classify_golden.json", got)
}

var parseSamples = map[string]string{
	"go": `// Package demo menjelaskan sesuatu.
package demo

import (
	"fmt"
	alias "example.com/x/y"
	_ "embed"
)

import "os"

// Server adalah server.
type Server struct{ a int }

type Iface interface{ M() }

type Alias = int

type Gen[T any] struct{}

const Max = 3
var Exported = 4
const hidden = 5

// Handle menangani.
func (s *Server) Handle(c *Ctx) error { fmt.Println(alias.Foo, os.Args); return nil }
func (s Server) other() {}
func (g *Gen[T]) Get() {}

func Run(r *Router) {
	r.Get("/a", h)
	r.Post("/b/:id", h)
	app.Delete("/c", h)
}
func private() {}
`,
	"ts": `// Modul util.
import { a } from './a'
import type { T } from '~/types'
const x = require('./cjs')
/** Hitung. */
export function calc() {}
export default async function* gen() {}
export const arrow = async (a: number): Promise<void> => {}
export const val = 3
export class Foo {}
export interface Bar {}
export type Baz = string
export enum E {}
function local() {}
const loc = (x) => x
class Priv {}
interface PI {}
export const useStore = defineStore('main', () => {})
const r = useFoo()
app.get('/x', h)
router.post('/y', h)
fetch('/api/things/' + id)
$fetch(` + "`/api/items/${id}/detail`" + `)
axios.get("/api/ab")
`,
	"vue": `<!-- Komponen kartu -->
<script setup lang="ts">
import Foo from './Foo.vue'
const props = defineProps<{ a: string }>()
const emit = defineEmits(['x'])
definePageMeta({ layout: 'x' })
const m = defineModel()
const { data } = useFetch('/api/list')
const t = useTodo()
</script>
<template><MyCard /><div><OtherThing/></div></template>
<script>
export default { name: 'x' }
</script>
`,
	"py": `#!/usr/bin/env python
# komentar
"""Modul python."""
from a.b import c
import os.path
class A:
    def m(self): pass
    async def n(self): pass
def f(): pass
async def g(): pass
def _h(): pass
class _P: pass
@app.get("/p/q")
def route(): pass
`,
	"php": `<?php
use App\Models\User;
use Foo\Bar;
abstract class Ctl extends Base {
    public static function make() {}
    private function hid() {}
}
interface I {}
trait T {}
enum Suit {}
function top() {}
Route::get('/users', [C::class, 'i']);
Route::resource('photos', P::class);
$routes->get('kontingen', 'Api\KontingenController::index');
$routes->post('/auth', 'AuthController::login');
$routes->get('plain');
`,
	"dart": `import 'package:flutter/material.dart';
export 'src/a.dart';
part 'b.g.dart';
abstract class Foo {}
mixin M {}
enum E { a }
extension X on int {}
class _Priv {}
void main() {
}
Future<void> load() async {
}
Widget build(BuildContext c) {
}
`,
	"java": `package x;
import java.util.List;
import static a.b.C.d;
@RestController
public class UserCtl {
    @GetMapping("/users")
    public List<User> list() { return null; }
    @RequestMapping(value = "/x")
    private void helper(int a) {}
}
interface Repo {}
`,
	"kotlin": `import a.b.C
class K
data class D(val a: Int)
object O
interface I
fun top() {}
suspend fun <T> gen(): T {}
private fun String.ext() {}
@PostMapping(path = "/k")
`,
	"rust":   "pub fn a() {}\nasync fn b() {}\npub(crate) struct S;\nenum E {}\ntrait T {}\n    fn inner() {}\n",
	"ruby":   "class A\n  def m; end\n  def self.s; end\n  def q?; end\nend\nmodule M\nend\nclass A::B\nend\n",
	"csharp": "public class A {\n  public static async Task<int> M(int a) { }\n  private void P() {}\n}\ninterface I {}\nrecord R();\nstruct S {}\nenum E {}\n",
	"swift":  "public class A {}\nstruct S {}\nprotocol P {}\nenum E {}\nextension A {}\n  public static func f() {}\nfunc g() {}\n",
	"sql":    "CREATE TABLE users (id int);\ncreate table if not exists `orders` (id int);\nCREATE TABLE \"x\" ();\n",
	"md":     "# Judul\n\nParagraf pertama dengan teks.\n\n## Sub\n\n- item\n\n```go\ncode\n```\n### Sub3 ###\n#### none\n",
	"md2":    "Tanpa judul\nbaris dua\n\n| a | b |\n![img](x)\n<div>\n",
	"md3":    "# Hanya judul\n",
	"css":    ".a { color: red }\n#b, .c { }\n.d-e:hover {\n}\nbody { }\n",
	"lic": `/*
 * Copyright 2020
 * License MIT
 */
// eslint-disable-next-line
// @ts-ignore
//go:generate x
// nolint
// prettier-ignore

// Deskripsi sebenarnya
// baris dua
package x
`,
	"lic2":  "#!/bin/sh\n'use strict'\n<?php\nimport a from 'b'\nfrom x import y\n// Ringkas\n// ok\nfunction f() {}\n",
	"lic3":  "// SPDX-License\nconst a = 1\n",
	"lic4":  "const a = 1\n// bukan awal\n",
	"lic5":  "/* blok\n * lanjut\n */\nfunction q() {}\n",
	"lic6":  "# py komentar\n# dua\nx = 1\n",
	"empty": "",
	"nl":    "\n\n\n",
	"doc":   "// Pertama.\nfunc A() {}\n\n// Kedua\n// panjang\nfunc B() {}\n#[derive(X)]\nfn z() {}\n",
	"long":  "// " + strings.Repeat("kata ", 120) + "\npackage p\n",
	"many":  strings.Repeat("func F() {}\n", 100),
}

func TestParseSourceGolden(t *testing.T) {
	langs := []string{"go", "typescript", "javascript", "vue", "svelte", "python", "php", "dart", "java", "kotlin", "rust", "ruby",
		"csharp", "swift", "sql", "markdown", "css", "text", ""}
	got := map[string]string{}
	for sk, src := range parseSamples {
		for _, l := range langs {
			p := parseSource(l, "a/b/file.x", src)
			b, _ := json.Marshal(map[string]interface{}{"s": p.symbols, "r": p.refs, "o": p.routes, "m": p.summary})
			got[sk+"/"+l] = hash8(string(b))
		}
	}
	for sk, src := range parseSamples {
		lines := strings.Split(src, "\n")
		got["leading/"+sk] = hash8(leadingComment(lines))
		got["mdsum/"+sk] = hash8(markdownSummary(lines))
		got["docabove/"+sk] = hash8(fmt.Sprint(len(lines), docAbove(lines, len(lines)/2+1), docAbove(lines, len(lines))))
		rs := applyRules(lines, goRules, goExported)
		b, _ := json.Marshal(rs)
		got["apply-go/"+sk] = hash8(string(b))
		rs = applyRules(lines, jsRules, nil)
		b, _ = json.Marshal(rs)
		got["apply-js/"+sk] = hash8(string(b))
		rs = applyRules(lines, pyRules, func(n string) bool { return !strings.HasPrefix(n, "_") })
		b, _ = json.Marshal(rs)
		got["apply-py/"+sk] = hash8(string(b))
		rs = applyRules(lines, jvmRules, nil)
		b, _ = json.Marshal(rs)
		got["apply-jvm/"+sk] = hash8(string(b))
	}
	checkGolden(t, "parse_golden.json", got)
}
