package api

import (
	"html"
	"net"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/localauth"
)

// Akses lokal engine (lapisan pertama, sebelum exec guard):
//   - Host harus localhost / IP loopback (atau IP LAN bila engine sengaja dibuka ke LAN) — menolak DNS rebinding.
//   - Setiap request API, MCP, dan halaman Studio wajib membawa token lokal (header, Bearer, atau cookie Studio).
//     Pengecualian: /health, /api/traffic/ingest (berkunci sendiri per sumber), dan aset statis Studio.
//   - Studio mendapat cookie lewat /auth?t=<token> (tautan dari `vibeasik serve` / `vibeasik open`).

// SetLAN menandai engine dibuka ke jaringan lokal (bind bukan loopback): Host berupa IP LAN diterima.
func (s *Server) SetLAN(on bool) { s.lan = on }

func securityHeaders(c *fiber.Ctx) error {
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "SAMEORIGIN")
	c.Set("Referrer-Policy", "no-referrer")
	return c.Next()
}

// allowedHost menerima nama localhost & IP loopback; dengan LAN juga IP privat/link-local dan nama .local.
func allowedHost(host string, lan bool) bool {
	h := host
	if hh, _, err := net.SplitHostPort(host); err == nil {
		h = hh
	}
	h = strings.ToLower(strings.Trim(h, "[]"))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	ip := net.ParseIP(h)
	if ip != nil && ip.IsLoopback() {
		return true
	}
	if !lan {
		return false
	}
	if ip != nil {
		return ip.IsPrivate() || ip.IsLinkLocalUnicast()
	}
	return strings.HasSuffix(h, ".local")
}

func requestToken(c *fiber.Ctx) string {
	if t := strings.TrimSpace(c.Get(localauth.Header)); t != "" {
		return t
	}
	if a := c.Get(fiber.HeaderAuthorization); strings.HasPrefix(a, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(a, "Bearer "))
	}
	return c.Cookies(localauth.Cookie)
}

func isAPIPath(p string) bool {
	return strings.HasPrefix(p, "/api/") || p == "/api" || strings.HasPrefix(p, "/mcp") || strings.HasPrefix(p, "/vibeasik/")
}

// isStaticAsset: berkas build Studio (JS, CSS, font, gambar) tidak rahasia dan dimuat sebelum cookie ada.
func isStaticAsset(p string) bool {
	if strings.HasPrefix(p, "/_nuxt/") || strings.HasPrefix(p, "/_fonts/") {
		return true
	}
	last := p[strings.LastIndex(p, "/")+1:]
	dot := strings.LastIndex(last, ".")
	if dot < 0 {
		return false
	}
	switch strings.ToLower(last[dot+1:]) {
	case "js", "css", "woff", "woff2", "ttf", "png", "jpg", "jpeg", "webp", "svg", "ico", "gif", "map":
		return true
	}
	return false
}

func (s *Server) localAccess(c *fiber.Ctx) error {
	if !allowedHost(c.Hostname(), s.lan) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "host_forbidden",
			"hint":  "Buka engine lewat http://localhost atau 127.0.0.1 (akses LAN: jalankan serve --host 0.0.0.0).",
		})
	}
	p := c.Path()
	if c.Method() == fiber.MethodOptions || p == "/health" || p == "/api/traffic/ingest" {
		return c.Next()
	}
	if p == "/auth" {
		return s.authHandshake(c)
	}
	if localauth.Equal(requestToken(c), s.token) {
		return c.Next()
	}
	if isAPIPath(p) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
			"hint":  "Kirim header " + localauth.Header + " berisi token dari `vibeasik token` (berkas " + localauth.Path() + ").",
		})
	}
	if isStaticAsset(p) {
		return c.Next()
	}
	return gatePage(c, fiber.StatusUnauthorized, "")
}

// authHandshake menukar token di URL dengan cookie Studio lalu mengarahkan ke halaman tujuan.
func (s *Server) authHandshake(c *fiber.Ctx) error {
	if !localauth.Equal(c.Query("t"), s.token) {
		return gatePage(c, fiber.StatusUnauthorized, "Tautan masuk tidak valid atau sudah kedaluwarsa.")
	}
	c.Cookie(&fiber.Cookie{
		Name:     localauth.Cookie,
		Value:    s.token,
		Path:     "/",
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
	next := c.Query("next", "/cockpit")
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.Contains(next, "\\") {
		next = "/cockpit"
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Redirect(next, fiber.StatusFound)
}

func gatePage(c *fiber.Ctx, status int, msg string) error {
	if msg == "" {
		msg = "Studio butuh tautan masuk dari laptop ini."
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Type("html", "utf-8")
	return c.Status(status).SendString(`<!doctype html><html lang="id"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>vibeasik — masuk</title>
<style>body{font-family:system-ui,sans-serif;background:#0b1220;color:#e2e8f0;display:grid;place-items:center;min-height:100vh;margin:0}main{max-width:32rem;padding:2rem}code{background:#1e293b;padding:.15rem .4rem;border-radius:.3rem}h1{font-size:1.3rem}</style></head>
<body><main><h1>🔒 ` + html.EscapeString(msg) + `</h1>
<p>Jalankan perintah ini di terminal laptop yang menjalankan vibeasik:</p>
<p><code>vibeasik open</code></p>
<p>atau buka tautan <code>/auth?t=…</code> yang dicetak <code>vibeasik serve</code>.</p></main></body></html>`)
}
