package diagnostics

import (
	"os"
	"testing"
)

// Rute login POST tanpa limiter apa pun di berkas harus memicu SEC-RATE-LIMIT-MISSING.
func TestRateLimitMissing_Detected(t *testing.T) {
	p := t.TempDir() + "/auth.go"
	src := "package api\n\nfunc setup(app *App) {\n\tapp.Post(\"/api/auth/login\", loginHandler)\n\tapp.Post(\"/api/auth/register\", registerHandler)\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasRuleID(inspectRuleIDs(p, "internal/api/auth.go"), "SEC-RATE-LIMIT-MISSING") {
		t.Error("rute autentikasi tanpa rate limiter harus terdeteksi SEC-RATE-LIMIT-MISSING")
	}
}

// Bila limiter sudah ada di berkas, aturan tidak boleh memicu (hindari false positive).
func TestRateLimitPresent_NotFlagged(t *testing.T) {
	p := t.TempDir() + "/auth.go"
	src := "package api\n\nimport \"github.com/gofiber/fiber/v2/middleware/limiter\"\n\nfunc setup(app *App) {\n\tauth := app.Group(\"/api/auth\", limiter.New(limiter.Config{Max: 5}))\n\tauth.Post(\"/login\", loginHandler)\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if hasRuleID(inspectRuleIDs(p, "internal/api/auth.go"), "SEC-RATE-LIMIT-MISSING") {
		t.Error("rute autentikasi dengan limiter tidak boleh ditandai SEC-RATE-LIMIT-MISSING")
	}
}

// Rute non-auth (mutasi biasa) tidak boleh memicu aturan ini.
func TestRateLimit_NonAuthRouteIgnored(t *testing.T) {
	p := t.TempDir() + "/products.go"
	src := "package api\n\nfunc setup(app *App) {\n\tapp.Post(\"/api/products\", createProduct)\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if hasRuleID(inspectRuleIDs(p, "internal/api/products.go"), "SEC-RATE-LIMIT-MISSING") {
		t.Error("rute non-autentikasi tidak boleh ditandai SEC-RATE-LIMIT-MISSING")
	}
}

// Sumber aturan tidak boleh memicu dirinya sendiri.
func TestRateLimit_NoSelfTrigger(t *testing.T) {
	for _, f := range []string{"code_inspector.go", "code_line_rules.go"} {
		if hasRuleID(inspectRuleIDs(f, "pkg/diagnostics/"+f), "SEC-RATE-LIMIT-MISSING") {
			t.Errorf("%s memicu SEC-RATE-LIMIT-MISSING pada dirinya sendiri", f)
		}
	}
}
