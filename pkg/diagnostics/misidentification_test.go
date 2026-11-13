package diagnostics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// ruleLines menjalankan InspectFile pada sumber sementara dan mengembalikan baris temuan per rule.
func ruleLines(t *testing.T, rel, lang, src string) map[string][]int {
	t.Helper()
	full := filepath.Join(t.TempDir(), filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out := map[string][]int{}
	for _, f := range InspectFile(full, rel, lang) {
		out[f.RuleID] = append(out[f.RuleID], f.Line)
	}
	return out
}

func wantLines(t *testing.T, got map[string][]int, rule string, want ...int) {
	t.Helper()
	g := got[rule]
	if len(g) != len(want) {
		t.Errorf("%s: baris temuan %v, seharusnya %v", rule, g, want)
		return
	}
	for i := range want {
		if g[i] != want[i] {
			t.Errorf("%s: baris temuan %v, seharusnya %v", rule, g, want)
			return
		}
	}
}

func TestSwallowedErrorGoOnlyWhenErrIsNeverRead(t *testing.T) {
	src := `package x

func a() error {
	_, err := os.Stat("a")
	if err != nil {
		return err
	}
	_, err = db.Exec(` + "`" + `
		UPDATE t SET a = 1
	` + "`" + `)
	return err
}

func b() {
	_, err := os.Stat("b")
	fmt.Println("selesai")
}

func c() {
	v, err := load()
	_ = err
	use(v)
}

func d() bool {
	_, err := os.Stat("d")
	return err == nil
}
`
	got := ruleLines(t, "internal/store/x.go", "go", src)
	wantLines(t, got, "SMELL-SWALLOWED-ERROR", 15, 21)
}

func TestLineRulesIgnoreStringsCommentsAndDocs(t *testing.T) {
	ts := `export const rules = [
  { rule: 'Dilarang catch (e) {} kosong dan AllowOrigins: "*" pada API', check: 'app.Post("/users", handler)' },
  { fix: "import moment from 'moment'" },
]
function load() {
  try { run() } catch (e) {}
}
`
	got := ruleLines(t, "server/rules.ts", "typescript", ts)
	wantLines(t, got, "SMELL-SWALLOWED-ERROR", 6)
	wantLines(t, got, "ROUTER-WILDCARD-CORS")
	wantLines(t, got, "ROUTER-UNGUARDED-MUTATION")

	goSrc := `package svc

const doc = ` + "`" + `
	app.Post("/items", createItem)
	db.Raw(fmt.Sprintf("SELECT * FROM t WHERE id = %s", id))
` + "`" + `

// app.Delete("/items/:id", deleteItem)
var hint = "exec.Command(\"sh\", \"-c\", cmd + input)"
`
	got = ruleLines(t, "internal/handlers/doc.go", "go", goSrc)
	for _, rule := range []string{"ROUTER-UNGUARDED-MUTATION", "QUERY-SQLI", "SEC-COMMAND-INJECTION"} {
		wantLines(t, got, rule)
	}
}

func TestUnguardedMutationSeesMiddlewareAliasAndClientCalls(t *testing.T) {
	src := `package api

func register(app *fiber.App, s *Server) {
	v1 := app.Group("/v1")
	a := s.requirePantauSource
	v1.Post("/auth/device/revoke", s.requireDevice, s.revokeDevice)
	v1.Post("/pantau/push", a, s.pushPantau)
	v1.Post("/tokens", s.createToken)
	v1.Delete("/items/:id", requireRole("admin")(s.deleteItem))
	v1.Put("/multi",
		s.requireUser, s.updateMulti)
	resp, err := s.client.Post("/remote/items", payload)
	kv.Delete("user:profile", id)
	v1.Post("/orders", s.createOrder)
}
`
	got := ruleLines(t, "internal/api/routes.go", "go", src)
	wantLines(t, got, "ROUTER-UNGUARDED-MUTATION", 8, 14)

	ts := `import axios from 'axios'
export async function save(payload) {
  const res = await axios.post('/api/items', payload)
  return api.post("https://example.test/items", payload)
}
`
	got = ruleLines(t, "src/services/items.ts", "typescript", ts)
	wantLines(t, got, "ROUTER-UNGUARDED-MUTATION")
}

func TestVerbInPathMatchesWholeWordsOnly(t *testing.T) {
	src := `package api

func register(app *fiber.App) {
	app.Get("/edition", h.edition)
	app.Get("/billing/credits", h.credits)
	app.Get("/addresses/:id", h.address)
	app.Get("/getUser/:id", h.user)
	app.Post("/orders/create-order", auth, h.create)
	app.Get("/users/:id/edit", h.editForm)
}
`
	got := ruleLines(t, "internal/api/routes.go", "go", src)
	wantLines(t, got, "ROUTER-VERB-IN-PATH", 7, 8)
}

func TestNPlusOneNeedsExecutedQueryInsideDataLoop(t *testing.T) {
	src := `package repo

func a(db *gorm.DB, users []User) {
	for _, u := range users {
		if u.Active {
			log.Println(u.ID)
		}
		db.Where("user_id = ?", u.ID).Find(&orders)
	}
	db.Find(&all)
}

func b(db *gorm.DB, filters []Filter) *gorm.DB {
	for _, f := range filters {
		db = db.Where(f.Cond, f.Val)
	}
	return db
}

func c(db *gorm.DB) {
	for _, where := range []string{"a = 1", ""} {
		db.Where(where).Find(&rows)
	}
}

func d(db *gorm.DB, ids []uint) *User {
	for _, id := range ids {
		if db.First(&u, id).Error == nil {
			return &u
		}
	}
	return nil
}
`
	got := ruleLines(t, "internal/repo/x.go", "go", src)
	wantLines(t, got, "QUERY-NPLUS1", 8)

	py := `def load(session, items):
    for item in items:
        session.query(Order).filter_by(item_id=item.id).all()
    session.query(Order).all()
    return {}
`
	got = ruleLines(t, "app/services/load.py", "python", py)
	wantLines(t, got, "QUERY-NPLUS1", 3)

	bench := "package main\n\nfunc main() {\n\tfor i := 0; i < n; i++ {\n\t\tdb.Exec(\"INSERT INTO t VALUES (?)\", i)\n\t}\n}\n"
	got = ruleLines(t, "cmd/bench_insert/main.go", "go", bench)
	wantLines(t, got, "QUERY-NPLUS1")
}

func TestCommandInjectionOnlyDynamicShell(t *testing.T) {
	src := `package x

func run(n int, input string) {
	exec.Command("git", "log", fmt.Sprintf("-%d", n)).Run()
	total := c.eval(a) + c.eval(b)
	exec.Command("sh", "-c", "ls -la").Run()
	exec.Command("sh", "-c", "echo "+input).Run()
	_ = total
}
`
	got := ruleLines(t, "pkg/x/run.go", "go", src)
	wantLines(t, got, "SEC-COMMAND-INJECTION", 7)

	php := "<?php\n$found = shell_exec('command -v mysqldump');\n$name = $_GET['n'];\n$out = shell_exec('ls ' . $name);\n"
	got = ruleLines(t, "app/Libraries/Backup.php", "php", php)
	wantLines(t, got, "SEC-COMMAND-INJECTION", 4)
}

func TestStubHandlerIgnoresPlainPythonPass(t *testing.T) {
	py := `class NotFound(Exception):
    pass


def parse(v):
    try:
        return int(v)
    except ValueError:
        pass


def create_item(request):
    pass
`
	got := ruleLines(t, "app/handlers/items.py", "python", py)
	wantLines(t, got, "ROUTER-STUB-HANDLER", 13)

	goSrc := "package x\n\nfunc f() {\n\tctx := context.TODO()\n\t_ = ctx\n}\n"
	got = ruleLines(t, "internal/handlers/x.go", "go", goSrc)
	wantLines(t, got, "ROUTER-STUB-HANDLER")
}

func TestHandlerPathUsesSegmentsNotSubstrings(t *testing.T) {
	for path, want := range map[string]bool{
		"backend/handlers/admin.go":           true,
		"internal/api/server.go":              true,
		"app/Controllers/UserController.php":  true,
		"src/user_handler.ts":                 true,
		"frontend/types/api.ts":               false,
		"frontend/data/api-docs.ts":           false,
		"lib/services/api_service.dart":       false,
		"pkg/capital/rapid.go":                false,
		"backend/internal/store/apikeys.go":   false,
		"server/routes/items.ts":              true,
		"backend/internal/router/router.go":   true,
		"backend/internal/service/payment.go": false,
	} {
		if got := isHandlerPath(path); got != want {
			t.Errorf("isHandlerPath(%q) = %v, seharusnya %v", path, got, want)
		}
	}
	if !isTypeOrDataFile("frontend/types/api.ts") || !isTypeOrDataFile("frontend/data/api-docs.ts") || isTypeOrDataFile("backend/handlers/admin.go") {
		t.Error("berkas tipe/data harus dikenali dan handler tidak")
	}
}

func TestSecretAndBodyReadContext(t *testing.T) {
	vue := "<template>\n  <ApiPanel :active-api-key=\"activeApiKeyValue\" />\n</template>\n<script setup>\nconst apiKey = \"sk_live_9f8a7b6c5d4e3f2a1b\"\n</script>\n"
	got := ruleLines(t, "app/pages/docs/api.vue", "vue", vue)
	wantLines(t, got, "SMELL-HARDCODED-SECRET", 5)

	goSrc := `package api

func a(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	data, _ := io.ReadAll(r.Body)
	_ = data
}
`
	got = ruleLines(t, "internal/handlers/a.go", "go", goSrc)
	wantLines(t, got, "PERF-UNBOUNDED-BODY-READ")

	goSrc = "package api\n\nfunc b(r *http.Request) {\n\tdata, _ := io.ReadAll(r.Body)\n\tout, _ := io.ReadAll(resp.Body)\n\tres := s.http.Get(ctx, url)\n\t_, _, _ = data, out, res\n}\n"
	got = ruleLines(t, "internal/handlers/b.go", "go", goSrc)
	wantLines(t, got, "PERF-UNBOUNDED-BODY-READ", 4)
	wantLines(t, got, "SMELL-MISSING-TIMEOUT")
}

func TestNativeDialogSkipsCommentsAndLocalFunctions(t *testing.T) {
	vue := `<template>
  <!-- Konfirmasi (pengganti confirm() bawaan browser) -->
  <p>Jangan pakai alert() di aplikasi</p>
  <button @click="confirm('Hapus?') && remove()">Hapus</button>
</template>
<script setup>
function save() {
  alert('tersimpan')
}
</script>
`
	got := ruleLines(t, "app/pages/items.vue", "vue", vue)
	wantLines(t, got, "DESIGN-NATIVE-DIALOG", 4, 8)

	vue = `<template>
  <button @click="confirm('Hapus?')">Hapus</button>
</template>
<script setup>
const { confirm } = useConfirm()
async function remove() {
  if (await confirm('Yakin?')) run()
}
</script>
`
	got = ruleLines(t, "app/pages/items2.vue", "vue", vue)
	wantLines(t, got, "DESIGN-NATIVE-DIALOG")
}

func TestXSSSkipsSanitizedComputed(t *testing.T) {
	vue := `<template>
  <div v-html="safeHtml" />
  <div v-html="item.content" />
</template>
<script setup>
const safeHtml = computed(() => {
  const raw = md.render(props.source)
  return DOMPurify.sanitize(raw)
})
</script>
`
	got := ruleLines(t, "app/components/MarkdownViewer.vue", "vue", vue)
	wantLines(t, got, "SEC-XSS-UNSAFE-HTML", 3)
}

func hasTitle(fs []string, sub string) bool {
	for _, f := range fs {
		if strings.Contains(f, sub) {
			return true
		}
	}
	return false
}

func TestProjectLevelChecksUseRealContext(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("be/tests/unit/UserTest.php", "<?php\nclass UserTest {}\n")
	write("be/app/User.php", "<?php\nclass User {}\n")
	write("docs/README.md", "# dokumen\n")
	write("web/tests/docs.test.mjs", "test('x', () => {})\n")
	write("web/app.js", "console.log(1)\n")

	// Komponen hanya menyimpan path relatif (absolute_path kosong); root bukan repo git.
	project := &models.Project{ID: 1, Name: "Demo", RootPath: root, Components: []models.Component{
		{Name: "be", RelativePath: "be", Role: models.RoleBackend, Language: "php"},
		{Name: "docs", RelativePath: "docs", Role: models.RoleOther},
		{Name: "web", RelativePath: "web", Role: models.RoleFrontendClient},
	}}
	diag, err := AnalyzeProject(project, nil)
	if err != nil {
		t.Fatal(err)
	}
	var findings []models.Finding
	if err := json.Unmarshal([]byte(diag.FindingsJSON), &findings); err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, f := range findings {
		titles = append(titles, f.RuleID+" "+f.Title)
	}
	for _, unwanted := range []string{"Component Directory Missing", "Component Path Not Set", "No Automated Tests", ".gitignore", "README"} {
		if hasTitle(titles, unwanted) {
			t.Errorf("%q tidak boleh dilaporkan untuk proyek ini: %v", unwanted, titles)
		}
	}
	if project.Components[0].AbsolutePath != "" {
		t.Error("AnalyzeProject tidak boleh mengubah komponen milik pemanggil")
	}

	// Komponen tanpa test tetap dilaporkan dengan ID aturan tetap dan kunci per komponen.
	write("api/main.go", "package main\n\nfunc main() {}\n")
	project.Components = append(project.Components, models.Component{Name: "api", RelativePath: "api", Role: models.RoleBackend, Language: "go"})
	diag, _ = AnalyzeProject(project, nil)
	_ = json.Unmarshal([]byte(diag.FindingsJSON), &findings)
	found := false
	for _, f := range findings {
		if f.RuleID == RuleNoTests && f.TargetFile == "api" {
			found = true
		}
	}
	if !found {
		t.Errorf("komponen api tanpa test harus dilaporkan sebagai %s", RuleNoTests)
	}
}

func TestAdvisoryFindingsAndPathAliasing(t *testing.T) {
	for _, f := range []models.Finding{
		{Category: "git", RuleID: RuleGitUncommitted},
		{RuleID: "PORT-SHARED-3000", Category: "port"},
		{RuleID: rulePolynomialGrowth},
	} {
		if !IsAdvisoryFinding(f) {
			t.Errorf("%s harus berupa catatan (bukan bahan todo)", f.RuleID)
		}
	}
	if IsAdvisoryFinding(models.Finding{RuleID: "QUERY-SQLI", Category: "query"}) {
		t.Error("temuan kode bukan catatan")
	}
	// Path berkas panjang tidak boleh disamarkan sanitizer rahasia.
	long := "database/migrations/2025_01_01_000000_create_webinar_participants_table.php"
	d := buildDiagnostic(1, []models.Finding{{RuleID: "X", TargetFile: long, Title: "t " + long}}, nil, func(int) string { return "" }, "")
	if !strings.Contains(d.FindingsJSON, long) {
		t.Errorf("path target harus utuh di laporan: %s", d.FindingsJSON)
	}
}
