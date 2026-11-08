package routemap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Golden test karakterisasi: mengunci seluruh keluaran analisis (graf, Markdown, JSONL, Trace, query) untuk
// kumpulan fixture lintas framework. UPDATE_GOLDEN=1 menulis ulang berkas testdata.

var goldenFixtures = map[string]map[string]string{
	"fiber_mix": {
		"go.mod": "module demo\n\ngo 1.22\n",
		"main.go": `package main

import (
	"demo/h"
	"demo/mw"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

var optionalAuth = mw.OptionalAuth()

func main() {
	app := fiber.New()
	app.Use(mw.Logger())
	app.Use("/static-ish", mw.CORS())
	app.Use(limiter.New(limiter.Config{Next: func(c *fiber.Ctx) bool {
		return c.Path() == "/ping" || strings.HasPrefix(c.Path(), "/health")
	}}))
	app.Use(func(c *fiber.Ctx) error { return c.Next() })
	app.Get("/ping", h.Ping)
	app.Get("/health/live", h.Ping)
	app.Static("/assets", "./public")
	api := app.Group("/api", mw.JWTProtected())
	api.Get("/me", optionalAuth, h.Me)
	api.Post("/things", mw.JWTProtected(), h.CreateThing)
	api.Put("/things/:id", h.UpdateThing)
	api.Delete("/things/:id", h.DeleteThing)
	api.Patch("/things/:id", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusAccepted) })
	api.Add("TRACE", "/trace", h.Ping)
	api.All("/any", h.Ping)
	sub := fiber.New()
	sub.Get("/x", h.Ping)
	sub.Post("/y", h.Ping)
	app.Mount("/sub", sub)
	app.Route("/r", func(r fiber.Router) {
		r.Get("/a", h.Ping)
		r.Post("/b", mw.JWTProtected(), h.Ping)
	})
	app.Use(mw.Late())
	app.Post("/after", h.Ping)
	app.Listen(":3000")
}
`,
		"mw/mw.go": `package mw

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func Logger() fiber.Handler        { return func(c *fiber.Ctx) error { return c.Next() } }
func CORS() fiber.Handler          { return func(c *fiber.Ctx) error { return c.Next() } }
func Late() fiber.Handler          { return func(c *fiber.Ctx) error { return c.Next() } }
func OptionalAuth() fiber.Handler  { return func(c *fiber.Ctx) error { return c.Next() } }
func JWTProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/open") || c.Path() == "/api/things/skip" {
			return c.Next()
		}
		return c.SendStatus(401)
	}
}
`,
		"h/h.go": `package h

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func respond(c *fiber.Ctx, code int, body any) error { return c.Status(code).JSON(body) }
func fail(c *fiber.Ctx, code int, msg string) error { return respond(c, code, fiber.Map{"error": msg}) }

func Ping(c *fiber.Ctx) error { return c.SendString("pong") }
func Me(c *fiber.Ctx) error {
	if c.Locals("u") == nil {
		return fail(c, http.StatusUnauthorized, "no")
	}
	return c.JSON(fiber.Map{"id": 1, "name": "x"})
}
func CreateThing(c *fiber.Ctx) error {
	return c.Status(fiber.StatusCreated).JSON(&Thing{ID: 1})
}
func UpdateThing(c *fiber.Ctx) error {
	if c.Params("id") == "" {
		return fiber.NewError(fiber.StatusBadRequest, "id")
	}
	return c.Redirect("/x", fiber.StatusSeeOther)
}
func DeleteThing(c *fiber.Ctx) error { return c.SendStatus(204) }

type Thing struct{ ID int }
`,
	},
	"gin_mix": {
		"go.mod": "module ginmix\n",
		"main.go": `package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc   { return func(c *gin.Context) { c.Next() } }
func AdminOnly() gin.HandlerFunc      { return func(c *gin.Context) { c.Next() } }
func Logger() gin.HandlerFunc         { return func(c *gin.Context) { c.Next() } }
func TrySoftAuth() gin.HandlerFunc    { return func(c *gin.Context) { c.Next() } }

func main() {
	r := gin.Default()
	r.Use(Logger())
	r.GET("/", index)
	r.Static("/s", "./s")
	r.StaticFile("/f", "./f")
	r.Match([]string{"GET", "POST"}, "/m", index)
	r.Any("/any", index)
	v1 := r.Group("/v1", AuthRequired())
	{
		v1.GET("/a", a)
		v1.POST("/b", b)
		adm := v1.Group("/admin", AdminOnly())
		adm.DELETE("/c/:id", c2)
	}
	soft := r.Group("/soft", TrySoftAuth())
	soft.GET("/s", index)
	pub := r.Group("/pub")
	pub.GET("/p", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	r.Run()
}

func index(c *gin.Context) { c.String(200, "hi") }
func a(c *gin.Context)     { c.JSON(http.StatusOK, gin.H{"a": 1}) }
func b(c *gin.Context) {
	if c.Query("x") == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "x"})
		return
	}
	c.Redirect(http.StatusFound, "/y")
}
func c2(c *gin.Context) { c.AbortWithStatus(404) }
`,
	},
	"echo_mix": {
		"go.mod": "module echomix\n",
		"main.go": `package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func JWT() echo.MiddlewareFunc    { return func(next echo.HandlerFunc) echo.HandlerFunc { return next } }
func Logger() echo.MiddlewareFunc { return func(next echo.HandlerFunc) echo.HandlerFunc { return next } }
func Recover() echo.MiddlewareFunc { return nil }

func main() {
	e := echo.New()
	e.Pre(Recover())
	e.GET("/", home)
	e.Add(http.MethodPost, "/add", home, Logger())
	e.POST("/items", create, JWT())
	g := e.Group("/api", JWT())
	g.GET("/list", list)
	g.DELETE("/items/:id", func(c echo.Context) error { return echo.NewHTTPError(http.StatusForbidden, "no") })
	g.Use(Logger())
	e.Use(Logger())
	e.Start(":1")
}

func home(c echo.Context) error { return c.NoContent(204) }
func create(c echo.Context) error { return c.JSON(201, map[string]int{"id": 1}) }
func list(c echo.Context) error {
	return c.JSON(http.StatusOK, []string{"a"})
}
`,
	},
	"chi_mix": {
		"go.mod": "module chimix\n",
		"main.go": `package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func JWTAuth(next http.Handler) http.Handler    { return next }
func Logger(next http.Handler) http.Handler     { return next }
func AdminGuard(next http.Handler) http.Handler { return next }

func main() {
	r := chi.NewRouter()
	r.Use(Logger)
	r.Get("/", home)
	r.Method("PUT", "/m", http.HandlerFunc(home))
	r.MethodFunc("PATCH", "/mf", home)
	r.Handle("/h", http.HandlerFunc(home))
	r.HandleFunc("/hf", home)
	r.Route("/api", func(r chi.Router) {
		r.Use(JWTAuth)
		r.Get("/x", home)
		r.Route("/admin", adminRoutes)
		r.With(AdminGuard).Post("/w", home)
	})
	r.Mount("/sub", subRouter())
	http.ListenAndServe(":1", r)
}

func adminRoutes(r chi.Router) {
	r.Use(AdminGuard)
	r.Delete("/{id}", home)
}

func subRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/z", home)
	return r
}

func home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "" {
		http.Error(w, "bad", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
`,
	},
	"gorilla_nethttp": {
		"go.mod": "module gor\n",
		"main.go": `package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func Auth(next http.Handler) http.Handler { return next }

func main() {
	r := mux.NewRouter()
	r.Use(Auth)
	r.HandleFunc("/a", home).Methods("GET", "POST")
	r.HandleFunc("/b", home)
	s := r.PathPrefix("/api").Subrouter()
	s.HandleFunc("/c", home).Methods("DELETE")
	http.ListenAndServe(":1", r)
}

func home(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
`,
		"std/go.mod": "module std\n",
		"std/main.go": `package main

import "net/http"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /items", create)
	mux.HandleFunc("/plain", create)
	mux.Handle("GET /h/{id}", http.HandlerFunc(create))
	http.ListenAndServe(":1", mux)
}

func create(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "x", 500)
	w.WriteHeader(201)
}
`,
	},
	"structured": {
		"go.mod": "module st\n",
		"main.go": `package main

import (
	"st/server"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.New().Run()
	},
}

func main() { rootCmd.Execute() }
`,
		"server/server.go": `package server

import (
	"st/handlers"

	"github.com/gofiber/fiber/v2"
)

type Server struct {
	app  *fiber.App
	mods []Module
}

type Module interface{ Register(r fiber.Router) }

func New() *Server {
	s := &Server{app: fiber.New()}
	s.mods = []Module{handlers.Users{}, handlers.Orders{}}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.app.Use(handlers.RequireAuth())
	g := s.app.Group("/v1")
	for _, m := range s.mods {
		m.Register(g)
	}
	s.app.Get("/factory", handlers.Make("x"))
	s.app.Get("/generic", handlers.Gen[int])
	s.app.Get("/var", handlers.Var)
}

func (s *Server) Run() error { return s.app.Listen(":1") }

func Orphan(app *fiber.App) {
	app.Get("/orphan", handlers.Make("o"))
}

func OrphanGroup(r fiber.Router) {
	r.Post("/orphan2", handlers.Make("o"))
}
`,
		"handlers/h.go": `package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func RequireAuth() fiber.Handler { return func(c *fiber.Ctx) error { return c.Next() } }

type Users struct{}
type Orders struct{}

func (Users) Register(r fiber.Router)  { r.Get("/users", list) }
func (Orders) Register(r fiber.Router) { r.Get("/orders", list) }

func list(c *fiber.Ctx) error { return c.JSON([]int{}) }

func Make(name string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if strings.TrimSpace(name) == "" {
			return c.SendStatus(400)
		}
		return c.SendString(name)
	}
}

func Gen[T any](c *fiber.Ctx) error { return c.JSON(fiber.Map{"t": 1}) }

var Var = func(c *fiber.Ctx) error { return c.SendStatus(200) }
`,
	},
	"duplicates": {
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
	api.Post("/validated", handlers.Validated)
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

type UserReq struct{ Email string; Age int }

func CreateUser(c *fiber.Ctx) error {
	var req UserReq
	if err := c.BodyParser(&req); err != nil {
		return c.SendStatus(400)
	}
	if strings.TrimSpace(req.Email) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "email wajib"})
	}
	if req.Age < 0 {
		return c.SendStatus(400)
	}
	if err := service.CreateUser(req.Email, req.Age); err != nil {
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

func Validated(c *fiber.Ctx) error {
	var req UserReq
	if err := validateReq(req); err != nil {
		return c.SendStatus(400)
	}
	if v := service.IsValid(req.Email); !v {
		return c.SendStatus(400)
	}
	return nil
}

func validateReq(r UserReq) error { return nil }
`,
		"service/user.go": `package service

import (
	"errors"
	"strings"
)

type input struct{ Email string; Age int }

func CreateUser(email string, age int) error {
	in := input{Email: email, Age: age}
	if strings.TrimSpace(in.Email) == "" {
		return errors.New("email wajib")
	}
	if in.Age < 0 {
		return errors.New("umur")
	}
	if in.Email == "" || in.Age == 0 {
		return nil
	}
	if err := save(in); err != nil {
		return err
	}
	return nil
}

func IsValid(s string) bool { return s != "" }

func save(in input) error { return nil }
`,
	},
	"stmts": {
		"go.mod": "module stm\n",
		"main.go": `package main

import (
	"github.com/gofiber/fiber/v2"
)

var (
	DeclRouter = fiber.New()
	jwtGuard   = auth()
)

func auth() fiber.Handler { return nil }
func h(c *fiber.Ctx) error { return c.SendStatus(200) }

func setup() *fiber.App {
	app := fiber.New()
	app.Get("/in-setup", h)
	return app
}

func wire(r fiber.Router, extra fiber.Handler) fiber.Router {
	r.Get("/wired", h, extra)
	return r
}

func main() {
	var app = fiber.New()
	var a, b = fiber.New(), fiber.New()
	_, _ = a, b
	x := setup()
	x.Get("/x", h)
	if y := wire(app, jwtGuard); y != nil {
		y.Get("/if-init", h)
	} else {
		app.Get("/else", h)
	}
	for i := 0; i < 2; i++ {
		app.Get("/for", h)
	}
	for _, v := range []string{"a"} {
		_ = v
		app.Get("/range", h)
	}
	switch {
	case true:
		app.Get("/switch", h)
	default:
		app.Get("/switch-default", h)
	}
	var iface interface{}
	switch iface.(type) {
	case int:
		app.Get("/tswitch", h)
	}
outer:
	for {
		app.Get("/labeled", h)
		break outer
	}
	go app.Get("/go", h)
	defer app.Get("/defer", h)
	func() {
		app.Get("/iife", h)
	}()
	{
		app.Get("/block", h)
	}
	app.Get(dynamicPath(), h)
	app.Get("/nohandler")
	app.Get("/lit", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"k": 1}) })
	grp := app.Group("/g", jwtGuard)
	grp.Get("/a", h)
	app.Use(jwtGuard, grp)
	log(setupOther(app))
	app.Listen(":1")
}

func dynamicPath() string { return "/dyn" }
func log(v ...interface{}) {}
func setupOther(app *fiber.App) *fiber.App { app.Get("/other", h); return app }
`,
	},
	"multi_module": {
		"a/go.mod":           "module a\n",
		"a/main.go":          "package main\n\nimport \"github.com/labstack/echo/v4\"\n\nfunc main() {\n\te := echo.New()\n\te.GET(\"/a\", func(c echo.Context) error { return c.NoContent(204) })\n}\n",
		"b/go.mod":           "module b\n",
		"b/main.go":          "package main\n\nimport \"github.com/gin-gonic/gin\"\n\nfunc main() {\n\tr := gin.New()\n\tr.POST(\"/b\", func(c *gin.Context) { c.Status(201) })\n}\n",
		"b/x/go.mod":         "module bx\n",
		"empty/go.mod":       "module empty\n",
		"empty/main.go":      "package main\nfunc main() {}\n",
		"skipme/vendor/v.go": "package v\n",
		"c/go.mod":           "module c\n",
		"c/main.go":          "package main\n\nimport \"github.com/labstack/echo/v4\"\n\nfunc main() {\n\te := echo.New()\n\te.POST(\"/c\", create)\n}\nfunc create(c echo.Context) error { return c.JSON(201, nil) }\n",
		"c/main_test.go":     "package main\n",
		"c/broken.go":        "package main\nfunc (\n",
	},
}

func manyRoutesFixture() map[string]string {
	var b strings.Builder
	b.WriteString("package main\n\nimport \"github.com/gofiber/fiber/v2\"\n\nfunc main() {\n\tapp := fiber.New()\n\tapi := app.Group(\"/api\", requireAuth())\n")
	for i := 0; i < 430; i++ {
		fmt.Fprintf(&b, "\tapp.Post(\"/open/r%03d\", h)\n", i)
		if i < 30 {
			fmt.Fprintf(&b, "\tapi.Put(\"/priv/r%03d\", h)\n", i)
		}
	}
	b.WriteString("}\n\nfunc requireAuth() fiber.Handler { return nil }\nfunc h(c *fiber.Ctx) error { return c.SendStatus(200) }\n")
	return map[string]string{"go.mod": "module many\n", "main.go": b.String()}
}

func goldenCompare(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden %s: %v", name, err)
	}
	if string(want) != got {
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := 0; i < len(gl) && i < len(wl); i++ {
			if gl[i] != wl[i] {
				t.Fatalf("golden %s berbeda di baris %d:\nwant: %s\ngot:  %s", name, i+1, wl[i], gl[i])
			}
		}
		t.Fatalf("golden %s berbeda panjang: want %d baris, got %d", name, len(wl), len(gl))
	}
}

func dumpAnalysis(gs []*Graph) string {
	var b strings.Builder
	b.WriteString("=== JSON\n")
	j, _ := json.MarshalIndent(gs, "", " ")
	b.Write(j)
	b.WriteString("\n=== MARKDOWN\n")
	b.WriteString(Markdown(gs))
	b.WriteString("\n=== JSONL\n")
	b.WriteString(JSONL(gs))
	b.WriteString("\n=== TRACE\n")
	for _, g := range gs {
		for _, rt := range g.Routes {
			b.WriteString(Trace(rt) + "\n")
		}
	}
	b.WriteString("=== FILEROUTES\n")
	fr := FileRoutes(gs)
	keys := make([]string, 0, len(fr))
	for k := range fr {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := append([]string(nil), fr[k]...)
		sort.Strings(v)
		fmt.Fprintf(&b, "%s: %s\n", k, strings.Join(v, "; "))
	}
	b.WriteString("=== QUERIES\n")
	for _, q := range [][3]string{{"", "", ""}, {"get", "", ""}, {"", "/a", ""}, {"POST", "", "list"}, {"", "", "handlers.CreateUser"}, {"", "", "save"}, {"", "", "home"}, {"", "", "h/h.go"}, {"", "", "create"}} {
		var ps []string
		for _, rt := range FindRoutes(gs, q[0], q[1], q[2]) {
			ps = append(ps, rt.Method+" "+rt.Path)
		}
		fmt.Fprintf(&b, "%v => %s\n", q, strings.Join(ps, ", "))
	}
	for _, g := range gs {
		for _, rt := range g.Routes {
			file, line := goldenSplitPos(rt.Pos)
			if at := RouteAt(gs, file, line); at != nil {
				fmt.Fprintf(&b, "RouteAt(%s) => %s %s\n", rt.Pos, at.Method, at.Path)
			}
			if rt.HandlerDecl != "" {
				f, l := goldenSplitPos(rt.HandlerDecl)
				fmt.Fprintf(&b, "ContextFor(%s) => %q\n", rt.HandlerDecl, ContextFor(gs, f, l))
				fmt.Fprintf(&b, "ContextFor(%s,0) => %q\n", f, ContextFor(gs, f, 0))
			}
		}
	}
	fmt.Fprintf(&b, "ContextFor(handlers.CreateUser) => %q\n", ContextFor(gs, "handlers.CreateUser", 0))
	fmt.Fprintf(&b, "ContextFor(empty) => %q\n", ContextFor(gs, "", 0))
	return b.String()
}

func goldenSplitPos(p string) (string, int) {
	i := strings.LastIndex(p, ":")
	n := 0
	fmt.Sscanf(p[i+1:], "%d", &n)
	return p[:i], n
}

func TestGoldenAnalysis(t *testing.T) {
	names := make([]string, 0, len(goldenFixtures)+1)
	for n := range goldenFixtures {
		names = append(names, n)
	}
	names = append(names, "many_routes")
	sort.Strings(names)
	for _, n := range names {
		files := goldenFixtures[n]
		if n == "many_routes" {
			files = manyRoutesFixture()
		}
		t.Run(n, func(t *testing.T) {
			root := writeTree(t, files)
			gs := Analyze(root)
			// urutan modul mengikuti WalkDir (leksikal) sehingga deterministik
			if n == "many_routes" { // terlalu besar untuk dump penuh: kunci Markdown saja
				goldenCompare(t, "routemap_"+n, Markdown(gs))
				return
			}
			goldenCompare(t, "routemap_"+n, dumpAnalysis(gs))
		})
	}
}
