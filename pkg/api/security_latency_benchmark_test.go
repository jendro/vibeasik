package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/security"
)

func TestGetCensoredKeywords_ContractAndProjection(t *testing.T) {
	s := newGuardTestServer(t)
	if err := s.db.AutoMigrate(&models.CensoredKeyword{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Buat beberapa fixture keyword langsung via SQL terparameterisasi agar nilai boolean tepat
	fixtures := []struct {
		kw, cat, desc string
		isRegex       int
		isActive      int
	}{
		{"STRIPE_LIVE_KEY_TEST", "API_KEY", "Stripe production key", 0, 1},
		{"PRIVATE_KEY_TEST", "AUTH_TOKEN", "RSA private header", 0, 0},
		{"SECRET_REGEX_[0-9]+", "ENVIRONMENT_SECRET", "Numbered token pattern", 1, 1},
	}

	for _, f := range fixtures {
		err := s.db.Exec("INSERT INTO censored_keywords (keyword, category, description, is_regex, is_active) VALUES (?, ?, ?, ?, ?)",
			f.kw, f.cat, f.desc, f.isRegex, f.isActive).Error
		if err != nil {
			t.Fatalf("failed to insert fixture: %v", err)
		}
	}

	tok := map[string]string{localauth.Header: testLocalToken}
	resp := doReq(t, s, "GET", "/api/security/censored-keywords", "localhost", tok)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", resp.StatusCode)
	}

	var body struct {
		Keywords        []models.CensoredKeyword `json:"keywords"`
		Total           int                      `json:"total"`
		ActiveCount     int                      `json:"active_count"`
		BuiltInPatterns []string                 `json:"built_in_patterns"`
		DoubleSecurity  bool                     `json:"double_security"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Total != 3 {
		t.Errorf("expected total 3, got %d", body.Total)
	}
	if body.ActiveCount != 2 {
		t.Errorf("expected active_count 2, got %d", body.ActiveCount)
	}
	if !body.DoubleSecurity {
		t.Errorf("expected double_security true")
	}
	if len(body.BuiltInPatterns) == 0 {
		t.Errorf("expected non-empty built_in_patterns")
	}

	// Verifikasi projection: kolom id, keyword, category, description, is_regex, is_active terisi utuh
	foundRegex := false
	for _, kw := range body.Keywords {
		if kw.ID == 0 {
			t.Errorf("keyword ID missing or 0: %+v", kw)
		}
		if kw.Keyword == "" {
			t.Errorf("keyword string missing: %+v", kw)
		}
		if kw.Category == "" {
			t.Errorf("keyword category missing (UI regression): %+v", kw)
		}
		if kw.Description == "" {
			t.Errorf("keyword description missing (UI regression): %+v", kw)
		}
		if kw.IsRegex {
			foundRegex = true
		}
	}

	if !foundRegex {
		t.Errorf("expected at least one regex keyword preserved")
	}
}

func TestCreateCensoredKeyword_SyncGlobalSanitizer(t *testing.T) {
	s := newGuardTestServer(t)
	if err := s.db.AutoMigrate(&models.CensoredKeyword{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	tok := map[string]string{localauth.Header: testLocalToken}

	// 1. Uji POST keyword literal
	payloadLiteral := `{"keyword":"LIVE_SUPER_SECRET_PAYMENT_XYZ","category":"API_KEY","description":"Payment token","is_regex":false}`
	resp1 := doReqWithBody(t, s, "POST", "/api/security/censored-keywords", "localhost", payloadLiteral, tok)
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created for literal, got: %d", resp1.StatusCode)
	}

	// Verifikasi GlobalSanitizer langsung tersinkronisasi (tanpa latency lag)
	sampleText := "The api secret is LIVE_SUPER_SECRET_PAYMENT_XYZ for integration"
	sanitizedText, audit := security.GlobalSanitizer.Sanitize(sampleText)
	if audit.RedactedCount == 0 {
		t.Errorf("expected GlobalSanitizer to detect LIVE_SUPER_SECRET_PAYMENT_XYZ after 201 response")
	}
	if strings.Contains(sanitizedText, "LIVE_SUPER_SECRET_PAYMENT_XYZ") {
		t.Errorf("sanitized text still contains raw secret: %s", sanitizedText)
	}

	// 2. Uji POST keyword regex
	payloadRegex := `{"keyword":"VIBE_KEY_[0-9A-F]{8}","category":"AUTH_TOKEN","description":"Hex key pattern","is_regex":true}`
	resp2 := doReqWithBody(t, s, "POST", "/api/security/censored-keywords", "localhost", payloadRegex, tok)
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created for regex, got: %d", resp2.StatusCode)
	}

	sampleRegexText := "Found token VIBE_KEY_ABC12345 in header"
	sanitizedRegexText, regexAudit := security.GlobalSanitizer.Sanitize(sampleRegexText)
	if regexAudit.RedactedCount == 0 {
		t.Errorf("expected GlobalSanitizer to detect regex pattern after 201 response")
	}
	if strings.Contains(sanitizedRegexText, "VIBE_KEY_ABC12345") {
		t.Errorf("sanitized text still contains matched regex pattern: %s", sanitizedRegexText)
	}
}

// TestLatencySLO_CensoredKeywordsEndpoints mengukur distribusi latensi p50, p95, p99
// dan memverifikasi batas kenyamanan interaktif 200ms SLO (K2213, K2266).
func TestLatencySLO_CensoredKeywordsEndpoints(t *testing.T) {
	s := newGuardTestServer(t)
	if err := s.db.AutoMigrate(&models.CensoredKeyword{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Seed 50 keyword untuk mensimulasikan dataset riil
	for i := 0; i < 50; i++ {
		kw := fmt.Sprintf("SAMPLE_KEYWORD_PATTERN_%03d", i)
		cat := "custom"
		desc := fmt.Sprintf("Description for keyword %d", i)
		isRegex := 0
		if i%5 == 0 {
			isRegex = 1
		}
		isActive := 0
		if i%2 == 0 {
			isActive = 1
		}
		_ = s.db.Exec("INSERT INTO censored_keywords (keyword, category, description, is_regex, is_active) VALUES (?, ?, ?, ?, ?)",
			kw, cat, desc, isRegex, isActive)
	}

	tok := map[string]string{localauth.Header: testLocalToken}
	iterations := 100
	latencies := make([]time.Duration, 0, iterations)

	// Ukur GET /api/security/censored-keywords
	for i := 0; i < iterations; i++ {
		start := time.Now()
		resp := doReq(t, s, "GET", "/api/security/censored-keywords", "localhost", tok)
		elapsed := time.Since(start)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got: %d", resp.StatusCode)
		}
		_ = resp.Body.Close()
		latencies = append(latencies, elapsed)
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	p50 := latencies[iterations*50/100]
	p95 := latencies[iterations*95/100]
	p99 := latencies[iterations*99/100]

	t.Logf("📊 GET /api/security/censored-keywords Latency Metrics (N=%d):", iterations)
	t.Logf("   p50: %v", p50)
	t.Logf("   p95: %v", p95)
	t.Logf("   p99: %v", p99)

	// Pastikan p95 jauh di bawah ambang 200ms
	if p95 > 200*time.Millisecond {
		t.Errorf("p95 latency %v melebihi SLO 200ms", p95)
	}

	// Ukur POST /api/security/censored-keywords (insert + SyncCensoredKeywords)
	postLatencies := make([]time.Duration, 0, 30)
	for i := 0; i < 30; i++ {
		body := fmt.Sprintf(`{"keyword":"LATENCY_TEST_KEY_%03d","category":"API_KEY","description":"Test"}`, i)
		start := time.Now()
		resp := doReqWithBody(t, s, "POST", "/api/security/censored-keywords", "localhost", body, tok)
		elapsed := time.Since(start)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got: %d", resp.StatusCode)
		}
		_ = resp.Body.Close()
		postLatencies = append(postLatencies, elapsed)
	}

	sort.Slice(postLatencies, func(i, j int) bool {
		return postLatencies[i] < postLatencies[j]
	})

	postP50 := postLatencies[len(postLatencies)*50/100]
	postP95 := postLatencies[len(postLatencies)*95/100]
	postP99 := postLatencies[len(postLatencies)*99/100]

	t.Logf("📊 POST /api/security/censored-keywords Latency Metrics (N=%d):", len(postLatencies))
	t.Logf("   p50: %v", postP50)
	t.Logf("   p95: %v", postP95)
	t.Logf("   p99: %v", postP99)

	if postP95 > 200*time.Millisecond {
		t.Errorf("POST p95 latency %v melebihi SLO 200ms", postP95)
	}
}
