package secaudit

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
)

func serve(t *testing.T, app *fiber.App) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	t.Cleanup(func() { _ = app.ShutdownWithTimeout(2 * time.Second) })
	return "http://" + ln.Addr().String()
}

// testLimiter: limiter per IP sederhana (429 + Retry-After setelah max request).
func testLimiter(max int) fiber.Handler {
	var mu sync.Mutex
	hits := map[string]int{}
	return func(c *fiber.Ctx) error {
		mu.Lock()
		hits[c.IP()]++
		n := hits[c.IP()]
		mu.Unlock()
		if n > max {
			c.Set("Retry-After", "60")
			return c.SendStatus(fiber.StatusTooManyRequests)
		}
		return c.Next()
	}
}

// hardenedApp memasang helmet, CORS allowlist, limiter, dan validasi input lewat router nyata.
func hardenedApp() *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(helmet.New(helmet.Config{ContentSecurityPolicy: "default-src 'self'; frame-ancestors 'none'"}))
	app.Use(cors.New(cors.Config{AllowOrigins: "https://app.contoh.test", AllowCredentials: true}))
	app.Use(testLimiter(8))
	app.Get("/api/items", func(c *fiber.Ctx) error {
		if id := c.Query("id"); id != "" {
			if _, err := strconv.Atoi(id); err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "id tidak valid"})
			}
		}
		return c.JSON(fiber.Map{"items": []string{}})
	})
	app.Post("/api/login", func(c *fiber.Ctx) error { return c.Status(401).JSON(fiber.Map{"error": "salah"}) })
	return app
}

// bareApp tanpa kontrol: memantulkan Origin + credentials dan membocorkan galat SQL.
func bareApp() *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		if o := c.Get("Origin"); o != "" {
			c.Set("Access-Control-Allow-Origin", o)
			c.Set("Access-Control-Allow-Credentials", "true")
		}
		c.Set("X-Powered-By", "Express 4.18.2")
		return c.Next()
	})
	app.Get("/api/items", func(c *fiber.Ctx) error {
		if id := c.Query("id"); id != "" {
			if _, err := strconv.Atoi(id); err != nil {
				return c.Status(500).SendString("Error 1064: You have an error in your SQL syntax near '" + id + "'")
			}
		}
		return c.JSON(fiber.Map{"items": []string{}})
	})
	app.Post("/api/login", func(c *fiber.Ctx) error { return c.Status(401).JSON(fiber.Map{"error": "salah"}) })
	return app
}

func byID(r *Report) map[string]Check {
	m := map[string]Check{}
	for _, c := range r.Checks {
		m[c.ID] = c
	}
	return m
}

var routes = []RouteRef{{"GET", "/api/items"}, {"POST", "/api/login"}, {"DELETE", "/api/items/:id"}}

func TestLiveHardenedServerPasses(t *testing.T) {
	base := serve(t, hardenedApp())
	rep := Run(context.Background(), Input{Target: "hardened", Routes: routes, Live: &LiveOptions{BaseURL: base, Burst: 20}})
	if rep.Live == nil || !rep.Live.Reachable {
		t.Fatalf("server tidak terjangkau: %+v", rep.Live)
	}
	c := byID(rep)
	for _, id := range []string{"LIVE-HEADERS", "LIVE-CORS", "LIVE-INPUT-BOUNDARY", "LIVE-RATE-LIMIT", "LIVE-DISCLOSURE"} {
		if c[id].Status != StatusPass {
			t.Errorf("%s = %s, ingin pass; bukti %v", id, c[id].Status, c[id].Evidence)
		}
	}
	// Limiter global sudah habis oleh burst GET, jadi POST login juga dibalas 429.
	if c["LIVE-AUTH-LIMIT"].Status != StatusPass {
		t.Errorf("LIVE-AUTH-LIMIT = %s; bukti %v", c["LIVE-AUTH-LIMIT"].Status, c["LIVE-AUTH-LIMIT"].Evidence)
	}
	if c["LIVE-TRANSPORT"].Status != StatusSkipped {
		t.Errorf("loopback HTTP seharusnya skipped, dapat %s", c["LIVE-TRANSPORT"].Status)
	}
	if rep.Grade != "A" {
		t.Errorf("grade %s (skor %d), ingin A", rep.Grade, rep.Score)
	}
}

func TestLiveBareServerFails(t *testing.T) {
	base := serve(t, bareApp())
	rep := Run(context.Background(), Input{Target: "bare", Routes: routes, Live: &LiveOptions{BaseURL: base, Burst: 15}})
	c := byID(rep)
	want := map[string]string{
		"LIVE-HEADERS":        StatusFail,
		"LIVE-CORS":           StatusFail,
		"LIVE-INPUT-BOUNDARY": StatusFail,
		"LIVE-RATE-LIMIT":     StatusFail,
		"LIVE-AUTH-LIMIT":     StatusFail,
		"LIVE-DISCLOSURE":     StatusWarn,
	}
	for id, st := range want {
		if c[id].Status != st {
			t.Errorf("%s = %s, ingin %s; bukti %v", id, c[id].Status, st, c[id].Evidence)
		}
	}
	if c["LIVE-AUTH-LIMIT"].Severity != SevCritical {
		t.Errorf("login tanpa limit harus CRITICAL")
	}
	if rep.Score >= 60 {
		t.Errorf("skor server tanpa kontrol terlalu tinggi: %d", rep.Score)
	}
	if rep.Checks[0].Status != StatusFail {
		t.Errorf("urutan: temuan gagal harus di atas, dapat %s", rep.Checks[0].Status)
	}
}

// Rute health yang bersih tidak boleh menutupi rute API yang bocor: tiap rute dinilai, yang terburuk dilaporkan.
func TestLiveWorstRouteIsReported(t *testing.T) {
	app := bareApp()
	app.Get("/health", func(c *fiber.Ctx) error { return c.SendString("ok") })
	base := serve(t, app)
	rep := Run(context.Background(), Input{Routes: []RouteRef{{"GET", "/health"}, {"GET", "/api/items"}}, Live: &LiveOptions{BaseURL: base, SkipBurst: true}})
	if got := rep.Live.Paths; len(got) != 2 || got[0] != "/health" || got[1] != "/api/items" {
		t.Fatalf("rute diuji = %v", got)
	}
	c := byID(rep)
	// Galat SQL hanya ada di /api/items, padahal /health dijawab bersih.
	if b := c["LIVE-INPUT-BOUNDARY"]; b.Status != StatusFail || len(b.Evidence) == 0 || b.Evidence[0] != "Rute /api/items:" {
		t.Errorf("LIVE-INPUT-BOUNDARY = %s %v, ingin fail dengan rute /api/items", b.Status, b.Evidence)
	}
	// CORS dipasang global di bareApp, jadi gagal (rute mana pun boleh disebut).
	if c["LIVE-CORS"].Status != StatusFail {
		t.Errorf("LIVE-CORS = %s", c["LIVE-CORS"].Status)
	}
}

func TestLiveUnreachableIsSkipped(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	rep := Run(context.Background(), Input{Live: &LiveOptions{BaseURL: addr, Timeout: time.Second}})
	if rep.Live.Reachable || len(rep.Checks) != 1 || rep.Checks[0].Status != StatusSkipped {
		t.Fatalf("server mati harus skipped, dapat %+v", rep.Checks)
	}
}

func TestNormalizeBaseURLRejectsOtherSchemes(t *testing.T) {
	if ValidateBaseURL("") != "" || ValidateBaseURL("https://api.contoh.test") != "" || ValidateBaseURL("file:///x") == "" {
		t.Errorf("ValidateBaseURL keliru")
	}
	for _, raw := range []string{"", "ftp://x", "file:///etc/passwd", "http://"} {
		if _, err := normalizeBaseURL(raw); err == nil {
			t.Errorf("%q seharusnya ditolak", raw)
		}
	}
	u, err := normalizeBaseURL("127.0.0.1:4101/api/?x=1")
	if err != nil || u.String() != "http://127.0.0.1:4101/api" {
		t.Errorf("normalisasi keliru: %v %v", u, err)
	}
}

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, name)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestStaticInsecureRouter(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"main.go": `package main
func setup(app *fiber.App, db *gorm.DB) {
	app.Use(cors.New(cors.Config{AllowOrigins: "*", AllowCredentials: true}))
	app.Post("/api/login", login)
	app.Delete("/api/users/:id", deleteUser)
	app.Get("/api/users", func(c *fiber.Ctx) error {
		db.Raw(fmt.Sprintf("SELECT * FROM users WHERE name = '%s'", c.Query("q"))).Scan(&u)
		return c.JSON(fiber.Map{"error": err.Error()})
	})
}`,
		".env.server":             "APP_DEBUG=true\nDB_PASSWORD=rahasia\n",
		"node_modules/x/index.js": `app.post("/login", h)`,
	})
	rep := Run(context.Background(), Input{Dirs: []SourceDir{{Name: "api", Path: dir}}})
	c := byID(rep)
	want := map[string]string{
		"RATE-LIMIT":    StatusFail,
		"SEC-HEADERS":   StatusFail,
		"CORS":          StatusFail,
		"SQL-INJECTION": StatusFail,
		"AUTH-MUTATION": StatusFail,
		"ERROR-LEAK":    StatusFail,
	}
	for id, st := range want {
		if c[id].Status != st {
			t.Errorf("%s = %s, ingin %s; bukti %v", id, c[id].Status, st, c[id].Evidence)
		}
	}
	if c["RATE-LIMIT"].Severity != SevCritical {
		t.Errorf("login tanpa limiter harus CRITICAL")
	}
	if l := c["SQL-INJECTION"].Locations; len(l) == 0 || l[0].File != "main.go" || l[0].Line != 7 {
		t.Errorf("lokasi SQL injection keliru: %+v", l)
	}
	for _, l := range c["ERROR-LEAK"].Locations {
		if l.Snippet != "APP_DEBUG=…" && l.File == ".env.server" {
			t.Errorf("nilai env tidak boleh ikut dilaporkan: %q", l.Snippet)
		}
	}
	if rep.FilesRead != 1 {
		t.Errorf("node_modules harus dilewati; files_read=%d", rep.FilesRead)
	}
	if n := len(c["CORS"].Locations); n != 1 {
		t.Errorf("lokasi CORS di baris sama harus satu, dapat %d", n)
	}
	if rep.RoutesSeen != 3 {
		t.Errorf("rute dari sumber = %d, ingin 3", rep.RoutesSeen)
	}
}

func TestStaticHardenedRouter(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"server.go": `package main
func setup(app *fiber.App, db *gorm.DB) {
	app := fiber.New(fiber.Config{BodyLimit: 1 << 20})
	app.Use(helmet.New(helmet.Config{ContentSecurityPolicy: "default-src 'self'"}))
	app.Use(cors.New(cors.Config{AllowOrigins: "https://app.contoh.test"}))
	app.Use(limiter.New(limiter.Config{Max: 100}))
	api := app.Group("/api", jwtware.New(jwtware.Config{}))
	api.Post("/items", create)
	api.Get("/items", func(c *fiber.Ctx) error {
		db.Where("name = ?", c.Query("q")).Find(&items)
		return validate.Struct(in)
	})
}
var validate = validator.New()`,
	})
	rep := Run(context.Background(), Input{Dirs: []SourceDir{{Path: dir}}})
	c := byID(rep)
	for id, st := range map[string]string{
		"RATE-LIMIT": StatusDeclared, "SEC-HEADERS": StatusDeclared, "CORS": StatusDeclared,
		"SQL-INJECTION": StatusPass, "AUTH-MUTATION": StatusDeclared, "INPUT-VALIDATION": StatusDeclared,
		"ERROR-LEAK": StatusPass, "CMD-INJECTION": StatusPass,
	} {
		if c[id].Status != st {
			t.Errorf("%s = %s, ingin %s; bukti %v", id, c[id].Status, st, c[id].Evidence)
		}
	}
	if rep.Grade != "A" {
		t.Errorf("grade %s (skor %d)", rep.Grade, rep.Score)
	}
}

func TestLivePathsSkipSideEffects(t *testing.T) {
	got := LivePaths([]RouteRef{
		{"GET", "/api/logout"}, {"GET", "/api/export/csv"}, {"GET", "/api/users/:id"},
		{"GET", "/health"}, {"POST", "/api/items"}, {"GET", "/files/*"},
	})
	want := []string{"/health", "/api/users/1"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("LivePaths = %v, ingin %v", got, want)
	}
	if a := AuthPaths([]RouteRef{{"POST", "/api/auth/login"}, {"GET", "/login"}, {"POST", "/api/items"}}); len(a) != 1 || a[0] != "/api/auth/login" {
		t.Errorf("AuthPaths = %v", a)
	}
}
