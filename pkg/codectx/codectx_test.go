package codectx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func setup(t *testing.T, files map[string]string, comps ...models.Component) (*gorm.DB, *models.Project) {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}, &models.ComponentRoute{}, &models.CodeMap{}, &models.CodeMapEntry{}); err != nil {
		t.Fatal(err)
	}
	for i := range comps {
		comps[i].AbsolutePath = filepath.Join(root, comps[i].AbsolutePath)
	}
	p := models.Project{Name: "uji", Slug: "uji", RootPath: root, Components: comps}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	return db, &p
}

const laravelController = `<?php
namespace App\Http\Controllers;

class UserController extends Controller
{
    public function index(Request $request)
    {
        $users = $this->userService->listActive($request->input('q'));
        return response()->json($users);
    }

    private function unusedHelper()
    {
        return 1;
    }
}
`

const laravelService = `<?php
namespace App\Services;

class UserService
{
    public function listActive($q)
    {
        return $this->repo->activeUsers($q);
    }
}
`

const laravelRepo = `<?php
namespace App\Repositories;

class UserRepository
{
    public function activeUsers($q)
    {
        return User::where('active', 1)->where('name', 'like', "%$q%")->get();
    }
}
`

func TestBuildLaravelFollowsRouteControllerServiceRepository(t *testing.T) {
	db, p := setup(t, map[string]string{
		"api/composer.json":                           `{"require":{"laravel/framework":"^10"}}`,
		"api/app/Http/Controllers/UserController.php": laravelController,
		"api/app/Services/UserService.php":            laravelService,
		"api/app/Repositories/UserRepository.php":     laravelRepo,
		"api/routes/api.php":                          "<?php\nRoute::get('/users', [UserController::class, 'index']);\n",
		"api/tests/Feature/UserControllerTest.php":    "<?php\nuse App\\Http\\Controllers\\UserController;\nclass UserControllerTest { public function testIndex() { $this->get('/users'); } }\n",
	}, models.Component{Name: "API", Role: "backend", AbsolutePath: "api"})
	db.Create(&models.ComponentRoute{ComponentID: p.Components[0].ID, Method: "GET", Path: "/api/users", Handler: "UserController@index", Middleware: "auth:sanctum"})

	c := Build(db, p, "app/Http/Controllers/UserController.php", 8)
	if c == nil || c.Unit == nil || c.Unit.Name != "index" || c.Unit.From != 6 || c.Unit.To != 10 {
		t.Fatalf("enclosing controller method expected: %+v", c)
	}
	names := map[string]int{}
	for _, s := range c.Callees {
		names[s.Name] = s.Depth
	}
	if names["listActive"] != 1 || names["activeUsers"] != 2 {
		t.Fatalf("controller → service (depth 1) → repository (depth 2) expected: %+v", c.Callees)
	}
	if len(c.Routes) != 1 || !strings.Contains(c.Routes[0], "GET /api/users") || !strings.Contains(c.Routes[0], "auth:sanctum") {
		t.Fatalf("route with middleware expected: %v", c.Routes)
	}
	if c.CallerCount == 0 || !strings.Contains(c.Callers[0].Path, "routes/api.php") {
		t.Fatalf("route file must count as caller ('index' reference): %+v", c.Callers)
	}
	if len(c.Tests) != 1 {
		t.Fatalf("test file expected: %v", c.Tests)
	}
	prompt := c.Prompt()
	for _, s := range []string{"Konteks Kode Menyeluruh", "GET /api/users", "listActive — app/Services/UserService.php", "User::where", "Diuji oleh"} {
		if !strings.Contains(prompt, s) {
			t.Fatalf("prompt missing %q:\n%s", s, prompt)
		}
	}

	// berkas utuh: fungsi eksternal & kode mati
	fc := BuildFile(db, p, "app/Http/Controllers/UserController.php")
	if fc == nil || len(fc.External) == 0 || fc.External[0].Name != "listActive" {
		t.Fatalf("external callee expected: %+v", fc)
	}
	if len(fc.Unused) != 1 || !strings.HasPrefix(fc.Unused[0], "unusedHelper") {
		t.Fatalf("unused private method expected: %v", fc.Unused)
	}
}

func TestGlobalFindsDuplicateAndUnusedFunctions(t *testing.T) {
	body := `
	if v == "" {
		return 0, errors.New("kosong")
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, errors.New("tidak valid")
	}
	return n, nil
}
`
	db, p := setup(t, map[string]string{
		"be/go.mod":             "module example.com/be\n\ngo 1.22\n",
		"be/handlers/a.go":      "package handlers\n\nfunc parseUserID(v string) (int, error) {" + body + "\nfunc UseA() { parseUserID(\"1\") }\n",
		"be/handlers/b.go":      "package handlers\n\nfunc parseOrderID(v string) (int, error) {" + body + "\nfunc UseB() { parseOrderID(\"1\") }\n",
		"be/handlers/dead.go":   "package handlers\n\nfunc neverCalled(v string) (int, error) {" + strings.Replace(body, "kosong", "beda", 1) + "\n",
		"be/service/doc.go":     "package service\n\nvar sample = `\n// CONTOH KODE DOKUMENTASI — bukan deklarasi fungsi Go\nfunc registerFeatures() {}\n`\n",
		"legacy/old/x.go":       "package old\n\nfunc parseUserID(v string) (int, error) {" + body,
		"be/handlers/a_test.go": "package handlers\n\nfunc TestA(t *testing.T) { UseA() }\n",
	}, models.Component{Name: "BE", Role: "backend", AbsolutePath: "be"})

	fs := GlobalFunctionFindings(db, p)
	var dup, unused []models.Finding
	for _, f := range fs {
		switch f.RuleID {
		case RuleDuplicateFunction:
			dup = append(dup, f)
		case RuleUnusedFunction:
			unused = append(unused, f)
			if strings.Contains(f.Title, "registerFeatures") {
				t.Fatalf("registerFeatures in documentation snippet must not be reported as unused: %+v", f)
			}
		}
	}
	if len(dup) != 1 || !strings.Contains(dup[0].Description, "parseOrderID") || !strings.Contains(dup[0].Title, "badan identik") || strings.Contains(dup[0].Description, "legacy/") {
		t.Fatalf("one identical-body group in component only (legacy excluded): %+v", dup)
	}
	if !strings.Contains(dup[0].Description, "neverCalled") || !strings.Contains(dup[0].Title, "3 tempat") {
		t.Fatalf("same shape with different literal must join the group: %+v", dup[0])
	}
	if len(unused) != 1 || !strings.HasSuffix(unused[0].TargetFile, "handlers/dead.go") {
		t.Fatalf("neverCalled must be reported unused: %+v", unused)
	}
}

func TestBlockEnd(t *testing.T) {
	goSrc := strings.Split("func a() {\n\ts := \"}\"\n\t// }\n\tif x {\n\t}\n}\nfunc b() {}", "\n")
	if got := blockEnd(goSrc, 1, "go", 0); got != 6 {
		t.Fatalf("go braces with string/comment: got %d", got)
	}
	py := strings.Split("def a():\n    x = 1\n\n    return x\ndef b():\n    pass", "\n")
	if got := blockEnd(py, 1, "python", 0); got != 4 {
		t.Fatalf("python indentation: got %d", got)
	}
	tpl := strings.Split("const a = () => {\n  const s = `\n}\n`\n  return s\n}\nconst b = 1", "\n")
	if got := blockEnd(tpl, 1, "typescript", 0); got != 6 {
		t.Fatalf("multi-line template literal: got %d", got)
	}
}

func TestSplitHandler(t *testing.T) {
	cases := map[string][2]string{
		`App\Http\Controllers\UserController@index`:  {"usercontroller", "index"},
		`App\Controllers\Api\AdminController::stats`: {"admincontroller", "stats"},
		"userController.list":                        {"usercontroller", "list"},
		"list_users":                                 {"", "list_users"},
	}
	for in, want := range cases {
		if c, m := splitHandler(in); c != want[0] || m != want[1] {
			t.Fatalf("splitHandler(%q) = %q,%q", in, c, m)
		}
	}
}
