package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestRouterUnguardedMutation_GroupMiddleware(t *testing.T) {
	src := `package routes

func Setup(app *fiber.App) {
	api := app.Group("/api")
	api.Post("/login", handlers.Login)
	admin := api.Group("/admin", middleware.RequireAuth(), middleware.RequireAdmin())
	admin.Post("/users", handlers.AdminCreateUser)
	sub := admin.Group("/queue")
	sub.Delete("/dlq", handlers.AdminClearDLQ)
	me := api.Group("/me")
	me.Use(middleware.RequireAuth())
	me.Put("/profile", handlers.UpdateProfile)
	api.Post("/orders", handlers.CreateOrder)
}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "routes", "routes.go")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var hits []int
	for _, f := range InspectFile(path, "routes/routes.go", "go") {
		if f.RuleID == "ROUTER-UNGUARDED-MUTATION" {
			hits = append(hits, f.Line)
		}
	}
	if len(hits) != 1 || hits[0] != 13 {
		t.Fatalf("hanya /orders (L13) yang tidak terjaga, dapat baris %v", hits)
	}
}

func TestDuplicateMiddlewareFinding(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module demo\n\ngo 1.22\n",
		"main.go": `package main

import "github.com/gofiber/fiber/v2"

func auth() fiber.Handler { return func(c *fiber.Ctx) error { return c.Next() } }

func main() {
	app := fiber.New()
	api := app.Group("/api", auth())
	api.Post("/orders", auth(), func(c *fiber.Ctx) error { return c.SendStatus(201) })
	app.Listen(":3000")
}
`,
	}
	for p, src := range files {
		if err := os.WriteFile(filepath.Join(dir, p), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var got []string
	for _, f := range InspectDirectory(dir, "be") {
		if f.RuleID == "ROUTER-DUPLICATE-MIDDLEWARE" {
			got = append(got, fmt.Sprintf("%s:%d", f.TargetFile, f.Line))
		}
	}
	if len(got) != 1 || got[0] != "be/main.go:10" {
		t.Fatalf("auth() dipasang ulang di rute (be/main.go:10), dapat %v", got)
	}
}

// TestRouterGlobalMiddlewareLocalAccessRecognized memverifikasi bahwa middleware global seperti s.localAccess
// dihitung sebagai security boundary efektif oleh routemap & diagnostics sehingga rute mutasi tidak
// menghasilkan false positive ROUTER-UNGUARDED-MUTATION (todo #3503).
func TestRouterGlobalMiddlewareLocalAccessRecognized(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module demo\n\ngo 1.22\n",
		"main.go": `package main

import "github.com/gofiber/fiber/v2"

type Server struct{}

func (s *Server) localAccess(c *fiber.Ctx) error { return c.Next() }

func (s *Server) setupRoutes(app *fiber.App) {
	app.Use(s.localAccess)
	api := app.Group("/api")
	api.Post("/setup/database/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	api.Post("/projects", func(c *fiber.Ctx) error { return c.SendStatus(201) })
}

func main() {
	app := fiber.New()
	s := &Server{}
	s.setupRoutes(app)
	app.Listen(":3000")
}
`,
	}
	for p, src := range files {
		if err := os.WriteFile(filepath.Join(dir, p), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var hits []models.Finding
	for _, f := range InspectDirectory(dir, "demo") {
		if f.RuleID == "ROUTER-UNGUARDED-MUTATION" {
			hits = append(hits, f)
		}
	}
	if len(hits) != 0 {
		t.Fatalf("diharapkan 0 temuan ROUTER-UNGUARDED-MUTATION karena dilindungi s.localAccess global, tapi dapat %d: %+v", len(hits), hits)
	}
}

// TestRouterUnguardedMutation_SkipsRawStringCodeSamples memverifikasi bahwa string literal yang berisi
// potongan kode rute (seperti di scaleup_service.go) tidak memicu temuan ROUTER-UNGUARDED-MUTATION (todo #3503).
func TestRouterUnguardedMutation_SkipsRawStringCodeSamples(t *testing.T) {
	src := "package service\n\n" +
		"type Snippet struct { Code string }\n\n" +
		"var Sample = Snippet{\n" +
		"\tCode: `// router sample\n" +
		"func setup(app *fiber.App) {\n" +
		"\tprojects := app.Group(\"/projects\")\n" +
		"\tprojects.Post(\"/\", createProject)\n" +
		"\tprojects.Post(\"/projects\", createProject)\n" +
		"}`,\n" +
		"}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "service", "scaleup_service.go")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. InspectFile langsung (tanpa routemap)
	var fileHits []models.Finding
	for _, f := range InspectFile(path, "service/scaleup_service.go", "go") {
		if f.RuleID == "ROUTER-UNGUARDED-MUTATION" {
			fileHits = append(fileHits, f)
		}
	}
	if len(fileHits) != 0 {
		t.Fatalf("InspectFile seharusnya mengabaikan rute di dalam raw string, dapat %d temuan: %+v", len(fileHits), fileHits)
	}

	// 2. InspectDirectory
	var dirHits []models.Finding
	for _, f := range InspectDirectory(dir, "demo") {
		if f.RuleID == "ROUTER-UNGUARDED-MUTATION" {
			dirHits = append(dirHits, f)
		}
	}
	if len(dirHits) != 0 {
		t.Fatalf("InspectDirectory seharusnya tidak melaporkan rute di berkas service/non-route, dapat %d temuan: %+v", len(dirHits), dirHits)
	}
}
