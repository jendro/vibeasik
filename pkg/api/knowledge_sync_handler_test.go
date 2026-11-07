package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/knowledgesync"
	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
)

func syncServer(t *testing.T) *Server {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv(localauth.EnvToken, testLocalToken)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}, &models.CensoredKeyword{}, &models.KnowledgeNode{}, &models.KnowledgeLink{}); err != nil {
		t.Fatal(err)
	}
	registry.SetEntitlement(func(f string) bool { return f == "" })
	t.Cleanup(func() { registry.SetEntitlement(func(f string) bool { return f == "" }) })
	return NewServer(db)
}

func syncCall(t *testing.T, s *Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "localhost"
	req.Header.Set(localauth.Header, testLocalToken)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func TestKnowledgeSyncViewAndConsent(t *testing.T) {
	s := syncServer(t)

	code, v := syncCall(t, s, "GET", "/api/knowledge-sync", "")
	if code != http.StatusOK || v["mode"] != "auto" || v["can_control"] != false {
		t.Fatalf("tampilan awal: %d %v", code, v)
	}
	if txt, _ := v["consent_text"].(string); !strings.Contains(txt, "Prinsip utama") {
		t.Fatal("teks persetujuan harus ikut dikirim ke Studio")
	}
	if c := v["consent"].(map[string]any); c["accepted"] != false {
		t.Fatalf("belum ada persetujuan: %v", c)
	}

	// Persetujuan harus eksplisit dan sesuai versi.
	if code, _ := syncCall(t, s, "POST", "/api/knowledge-sync/consent", `{}`); code != http.StatusBadRequest {
		t.Fatalf("tanpa accept harus 400, dapat %d", code)
	}
	if code, _ := syncCall(t, s, "POST", "/api/knowledge-sync/consent", `{"accept":true,"version":"versi-lama"}`); code != http.StatusConflict {
		t.Fatalf("versi usang harus 409, dapat %d", code)
	}
	body := `{"accept":true,"version":"` + knowledgesync.ConsentVersion + `"}`
	code, v = syncCall(t, s, "POST", "/api/knowledge-sync/consent", body)
	if code != http.StatusOK || v["consent"].(map[string]any)["accepted"] != true {
		t.Fatalf("persetujuan: %d %v", code, v)
	}
}

func TestKnowledgeSyncModeLockedOnFreeAllowedOnPro(t *testing.T) {
	s := syncServer(t)
	code, v := syncCall(t, s, "POST", "/api/knowledge-sync/mode", `{"mode":"off"}`)
	if code != http.StatusForbidden || v["code"] != "locked" {
		t.Fatalf("Free harus terkunci: %d %v", code, v)
	}
	registry.SetEntitlement(func(f string) bool { return f == knowledgesync.FeatureAutoControl })
	if code, v = syncCall(t, s, "POST", "/api/knowledge-sync/mode", `{"mode":"off"}`); code != http.StatusOK || v["mode"] != "off" {
		t.Fatalf("Pro boleh mengubah: %d %v", code, v)
	}
	if code, _ = syncCall(t, s, "POST", "/api/knowledge-sync/mode", `{"mode":"sembarang"}`); code != http.StatusBadRequest {
		t.Fatalf("mode tidak dikenal harus 400, dapat %d", code)
	}
}

func TestKnowledgeSyncPreviewAndPullGuard(t *testing.T) {
	s := syncServer(t)
	code, v := syncCall(t, s, "GET", "/api/knowledge-sync/preview", "")
	if code != http.StatusOK {
		t.Fatalf("preview: %d", code)
	}
	if _, ok := v["will_send"].([]any); !ok {
		t.Fatalf("will_send harus berupa array (bukan null): %v", v)
	}
	// Tarik paket tanpa persetujuan ditolak sebelum menyentuh jaringan.
	if code, _ = syncCall(t, s, "POST", "/api/knowledge-sync/pull", ""); code != http.StatusForbidden {
		t.Fatalf("pull tanpa persetujuan harus 403, dapat %d", code)
	}
}

func TestKnowledgeSyncRequiresLocalToken(t *testing.T) {
	s := syncServer(t)
	req := httptest.NewRequest("GET", "/api/knowledge-sync", nil)
	req.Host = "localhost"
	resp, _ := s.app.Test(req, -1)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("tanpa token harus 401, dapat %d", resp.StatusCode)
	}
}
