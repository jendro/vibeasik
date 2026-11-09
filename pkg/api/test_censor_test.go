package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/localauth"
)

func TestTestCensorPayload_ValidationAndCaching(t *testing.T) {
	s := newGuardTestServer(t)

	// 1. Missing body (empty) -> 400 Bad Request
	req := httptest.NewRequest("POST", "/api/security/test-censor", nil)
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(localauth.Header, testLocalToken)
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", resp.StatusCode)
	}

	// 2. Empty text field -> 400 Bad Request
	emptyBody := bytes.NewBufferString(`{"text": ""}`)
	req = httptest.NewRequest("POST", "/api/security/test-censor", emptyBody)
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(localauth.Header, testLocalToken)
	resp, err = s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty text, got %d", resp.StatusCode)
	}

	// 3. Valid text with sensitive pattern -> 200 OK + Sanitized + Cache-Control header
	payload := map[string]string{"text": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.dummy secret"}
	bodyBytes, _ := json.Marshal(payload)
	req = httptest.NewRequest("POST", "/api/security/test-censor", bytes.NewBuffer(bodyBytes))
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(localauth.Header, testLocalToken)
	resp, err = s.app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl != "private, max-age=60" {
		t.Errorf("expected Cache-Control 'private, max-age=60', got %q", cacheControl)
	}

	var res1 map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&res1)
	if res1["sanitized_text"] == "" {
		t.Errorf("expected sanitized_text to be present, got %+v", res1)
	}

	// 4. Repeated call -> Fast In-Memory Cache Hit
	req2 := httptest.NewRequest("POST", "/api/security/test-censor", bytes.NewBuffer(bodyBytes))
	req2.Host = "localhost"
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(localauth.Header, testLocalToken)
	resp2, err := s.app.Test(req2, -1)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for cached call, got %d", resp2.StatusCode)
	}
	var res2 map[string]interface{}
	_ = json.NewDecoder(resp2.Body).Decode(&res2)
	if res2["sanitized_text"] != res1["sanitized_text"] {
		t.Errorf("expected identical sanitized text from cache, got %v vs %v", res1["sanitized_text"], res2["sanitized_text"])
	}
}
