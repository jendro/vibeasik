package routemap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for p, src := range files {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func find(t *testing.T, gs []*Graph, method, path string) *Route {
	t.Helper()
	for _, g := range gs {
		for _, rt := range g.Routes {
			if rt.Method == method && rt.Path == path {
				return rt
			}
		}
	}
	t.Fatalf("rute %s %s tidak ditemukan", method, path)
	return nil
}

func chainNames(rt *Route) string {
	var n []string
	for _, mw := range rt.Chain {
		n = append(n, mw.Name)
	}
	return strings.Join(n, " | ")
}

func TestFiberTrace(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod": "module demo\n\ngo 1.22\n",
		"main.go": `package main

import (
	"demo/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func main() {
	app := fiber.New()
	app.Use(recover.New())
	app.Use(limiter.New(limiter.Config{Next: func(c *fiber.Ctx) bool { return strings.HasPrefix(c.Path(), "/health") }}))
	routes.Setup(app)
	app.Listen(":3000")
}
`,
		"routes/routes.go": `package routes

import (
	"demo/handlers"
	"demo/middleware"
	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App) {
	app.Get("/health", handlers.Health)
	api := app.Group("/api")
	api.Post("/login", handlers.Login)
	user := api.Group("", middleware.RequireAuth())
	user.Post("/orders", handlers.CreateOrder)
	admin := api.Group("/admin", middleware.RequireAdmin())
	admin.Delete("/users/:id", handlers.DeleteUser)
	api.Post("/late", handlers.Login)
	app.Use(middleware.Audit())
	handlers.NewAdmin().Register(admin)
}
`,
		"middleware/mw.go": `package middleware

import "github.com/gofiber/fiber/v2"

func RequireAuth() fiber.Handler { return func(c *fiber.Ctx) error { return c.Next() } }
func RequireAdmin() fiber.Handler { return func(c *fiber.Ctx) error { return c.Next() } }
func Audit() fiber.Handler { return func(c *fiber.Ctx) error { return c.Next() } }
`,
		"handlers/h.go": `package handlers

import (
	"demo/services"
	"github.com/gofiber/fiber/v2"
)

func fail(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"success": false, "message": msg})
}

func Health(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) }

func Login(c *fiber.Ctx) error {
	if c.Body() == nil {
		return fail(c, fiber.StatusBadRequest, "body kosong")
	}
	return c.SendStatus(204)
}

func CreateOrder(c *fiber.Ctx) error {
	if err := services.PlaceOrder(); err != nil {
		return fail(c, 500, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": 1})
}

func DeleteUser(c *fiber.Ctx) error { return fiber.NewError(fiber.StatusNotFound, "tidak ada") }

type Module interface{ Register(r fiber.Router) }
type A struct{}
type B struct{}

func (A) Register(r fiber.Router) { r.Put("/a", Health) }
func (B) Register(r fiber.Router) { r.Put("/b", Health) }

type Admin struct{ mods []Module }

func NewAdmin() *Admin { return &Admin{mods: []Module{A{}, B{}}} }
func (h *Admin) Register(r fiber.Router) {
	for _, m := range h.mods {
		m.Register(r)
	}
}

func Orphan(app *fiber.App) { app.Post("/dead", Health) }
`,
		"services/s.go": `package services

func PlaceOrder() error { return save() }
func save() error       { return nil }
`,
	})
	gs := Analyze(root)
	if len(gs) != 1 || gs[0].Framework != "fiber" {
		t.Fatalf("graf = %+v", gs)
	}

	health := find(t, gs, "GET", "/health")
	if health.Auth || len(health.Inactive) == 0 || !strings.Contains(health.Inactive[0].Reason, "dilewati") {
		t.Errorf("/health: limiter harus dilewati lewat Next, inactive=%+v", health.Inactive)
	}

	login := find(t, gs, "POST", "/api/login")
	if login.Auth {
		t.Error("/api/login didaftarkan sebelum group auth: harus publik")
	}
	if len(login.Responses) != 2 || login.Responses[0].Status != 204 || login.Responses[1].Status != 400 || login.Responses[1].Via != "fail" {
		t.Errorf("response login = %+v", login.Responses)
	}

	orders := find(t, gs, "POST", "/api/orders")
	if !orders.Auth || !strings.Contains(chainNames(orders), "recover.New()") {
		t.Errorf("/api/orders harus auth lewat group dan memuat middleware global: %s", chainNames(orders))
	}
	if len(orders.Stack) != 2 || !strings.Contains(orders.Stack[1], "routes.Setup") {
		t.Errorf("stack = %v", orders.Stack)
	}
	if len(orders.Calls) < 2 || !strings.Contains(strings.Join(orders.Calls, ","), "services.save") {
		t.Errorf("calls = %v", orders.Calls)
	}

	// Fiber: Group("") ber-middleware memasang auth di prefix /api untuk rute sesudahnya, lintas group.
	late := find(t, gs, "POST", "/api/late")
	if !late.Auth {
		t.Error("/api/late sesudah Group(\"\", RequireAuth) harus ikut auth (semantik prefix fiber)")
	}

	del := find(t, gs, "DELETE", "/api/admin/users/:id")
	if !strings.Contains(chainNames(del), "RequireAdmin") || del.Responses[0].Status != 404 || del.Responses[0].Kind != "error" {
		t.Errorf("delete: chain=%s resp=%+v", chainNames(del), del.Responses)
	}
	inactive := false
	for _, mw := range del.Inactive {
		if strings.Contains(mw.Name, "Audit") && strings.Contains(mw.Reason, "setelah") {
			inactive = true
		}
	}
	if !inactive {
		t.Errorf("Audit() didaftarkan setelah rute: harus tidak aktif, inactive=%+v", del.Inactive)
	}

	// dispatch interface (CHA): A.Register & B.Register di bawah /api/admin
	if a := find(t, gs, "PUT", "/api/admin/a"); !a.Auth {
		t.Error("/api/admin/a harus auth")
	}
	find(t, gs, "PUT", "/api/admin/b")

	dead := find(t, gs, "POST", "/dead")
	if !dead.Unreachable || len(gs[0].Warnings) == 0 {
		t.Errorf("rute Orphan harus ditandai tidak terjangkau, warnings=%v", gs[0].Warnings)
	}

	// penelusuran balik: fungsi → rute
	back := FindRoutes(gs, "", "", "save")
	if len(back) != 1 || back[0].Path != "/api/orders" {
		t.Errorf("FindRoutes(save) = %v", back)
	}
	if rt := RouteAt(gs, "routes/routes.go", 14); rt == nil || rt.Path != "/api/orders" {
		t.Errorf("RouteAt = %+v", rt)
	}
	md := Markdown(gs)
	for _, want := range []string{"## Peta Rute", "`POST /api/orders`", "tidak berjalan untuk", "Tidak terpasang"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown tanpa %q", want)
		}
	}
	if !strings.Contains(Trace(orders), "Jalur setup: main.go:10 main()") {
		t.Errorf("trace:\n%s", Trace(orders))
	}
}

func TestChiGinEcho(t *testing.T) {
	root := writeTree(t, map[string]string{
		"chi/go.mod": "module chiapp\n",
		"chi/main.go": `package main

import "github.com/go-chi/chi/v5"

func main() {
	r := chi.NewRouter()
	r.Use(Logger)
	r.Route("/api", func(r chi.Router) {
		r.Get("/pub", h)
		r.Group(func(r chi.Router) {
			r.Use(JWTAuth)
			r.Post("/priv", h)
		})
		r.With(JWTAuth).Delete("/x", h)
	})
}
`,
		"gin/go.mod": "module ginapp\n",
		"gin/main.go": `package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	v1 := r.Group("/v1", AuthRequired())
	v1.POST("/a", h)
	pub := r.Group("/v1")
	pub.POST("/b", h)
	r.Run()
}
`,
		"echo/go.mod": "module echoapp\n",
		"echo/main.go": `package main

import "github.com/labstack/echo/v4"

func main() {
	e := echo.New()
	e.POST("/items", create, JWT())
	e.Use(Logger())
}

func create(c echo.Context) error { return c.JSON(201, map[string]int{"id": 1}) }
`,
	})
	gs := Analyze(root)
	if len(gs) != 3 {
		t.Fatalf("modul = %d", len(gs))
	}
	if find(t, gs, "GET", "/api/pub").Auth || !find(t, gs, "POST", "/api/priv").Auth || !find(t, gs, "DELETE", "/api/x").Auth {
		t.Error("chi: auth group/With salah")
	}
	if !find(t, gs, "POST", "/v1/a").Auth || find(t, gs, "POST", "/v1/b").Auth {
		t.Error("gin: middleware group hanya untuk turunan group itu")
	}
	items := find(t, gs, "POST", "/items")
	if !items.Auth || items.Responses[0].Status != 201 || !strings.Contains(chainNames(items), "Logger") {
		t.Errorf("echo: chain=%s resp=%+v (e.Use berlaku walau didaftarkan sesudah rute)", chainNames(items), items.Responses)
	}
}

func TestDuplicateMiddlewareAndValidation(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod": "module demo\n\ngo 1.22\n",
		"main.go": `package main

import (
	"demo/handlers"
	"demo/middleware"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()
	api := app.Group("/api", middleware.RequireAuth())
	api.Post("/users", middleware.RequireAuth(), handlers.CreateUser)
	api.Post("/notes", handlers.CreateNote)
	app.Listen(":3000")
}
`,
		"middleware/auth.go": `package middleware

import "github.com/gofiber/fiber/v2"

func RequireAuth() fiber.Handler { return func(c *fiber.Ctx) error { return c.Next() } }
`,
		"handlers/user.go": `package handlers

import (
	"strings"

	"demo/service"
	"github.com/gofiber/fiber/v2"
)

type UserReq struct{ Email string }

func CreateUser(c *fiber.Ctx) error {
	var req UserReq
	if err := c.BodyParser(&req); err != nil {
		return c.SendStatus(400)
	}
	if strings.TrimSpace(req.Email) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "email wajib"})
	}
	if err := service.CreateUser(req.Email); err != nil {
		return c.SendStatus(500)
	}
	return c.SendStatus(201)
}

func CreateNote(c *fiber.Ctx) error {
	var req UserReq
	if req.Email == "" {
		return c.SendStatus(400)
	}
	return c.SendStatus(201)
}
`,
		"service/user.go": `package service

import (
	"errors"
	"strings"
)

type input struct{ Email string }

func CreateUser(email string) error {
	in := input{Email: email}
	if strings.TrimSpace(in.Email) == "" {
		return errors.New("email wajib")
	}
	if err := save(in); err != nil {
		return err
	}
	return nil
}

func save(in input) error { return nil }
`,
	})
	gs := Analyze(root)
	if len(gs) != 1 {
		t.Fatalf("butuh 1 graph, dapat %d", len(gs))
	}
	var mw, val []Duplicate
	for _, d := range gs[0].Duplicates {
		switch d.Kind {
		case "middleware":
			mw = append(mw, d)
		case "validation":
			val = append(val, d)
		}
	}
	if len(mw) != 1 || !strings.Contains(mw[0].Name, "RequireAuth") || mw[0].Detail != "group → route" || mw[0].Routes[0] != "POST /api/users" {
		t.Fatalf("middleware ganda RequireAuth group → route di /api/users, dapat %+v", mw)
	}
	if len(val) != 1 || val[0].Name != `TrimSpace(Email) == ""` || !strings.HasPrefix(val[0].Pos, "service/user.go:") {
		t.Fatalf("validasi email ganda handler → service, dapat %+v", val)
	}
}

func TestContextForNarrowsToFunctionAtLine(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod": "module app\n",
		"main.go": `package main

import "github.com/labstack/echo/v4"

func main() {
	e := echo.New()
	e.POST("/items", create)
	e.GET("/users", listUsers)
}
`,
		"handlers.go": `package main

import "github.com/labstack/echo/v4"

func create(c echo.Context) error {
	return c.JSON(201, nil)
}

func listUsers(c echo.Context) error {
	return c.JSON(200, nil)
}
`,
	})
	gs := Analyze(root)
	got := ContextFor(gs, "handlers.go", 6)
	if !strings.Contains(got, "/items") || strings.Contains(got, "/users") {
		t.Errorf("temuan di create harus hanya membawa rute /items:\n%s", got)
	}
	if file := ContextFor(gs, "handlers.go", 0); !strings.Contains(file, "/items") || !strings.Contains(file, "/users") {
		t.Errorf("temuan tingkat berkas tetap membawa semua rute berkas:\n%s", file)
	}
}
