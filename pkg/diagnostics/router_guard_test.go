package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
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
