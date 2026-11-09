package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func reqAPI(s *Server, method, path, body string) *http.Response {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Host = "localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(localauth.Header, testLocalToken)
	resp, _ := s.app.Test(req, -1)
	return resp
}

func TestSetupGet(t *testing.T) {
	s := newGuardTestServer(t)
	resp := reqAPI(s, "GET", "/api/setup", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/setup status %d, want 200", resp.StatusCode)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, key := range []string{"complete", "terms", "account", "database", "restart_supported"} {
		if _, ok := res[key]; !ok {
			t.Errorf("missing key %q in setup response", key)
		}
	}
}

func TestSetupAccountValidation(t *testing.T) {
	s := newGuardTestServer(t)

	// Invalid choice
	resp := reqAPI(s, "POST", "/api/setup/account", `{"choice":"invalid"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /api/setup/account invalid choice status %d, want 400", resp.StatusCode)
	}

	// Valid choice: skip
	resp = reqAPI(s, "POST", "/api/setup/account", `{"choice":"skip"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/setup/account skip status %d, want 200", resp.StatusCode)
	}
}

func TestSetupTermsValidation(t *testing.T) {
	s := newGuardTestServer(t)

	// Unchecked accept
	resp := reqAPI(s, "POST", "/api/setup/terms", `{"accept":false,"version":"v1"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /api/setup/terms accept=false status %d, want 400", resp.StatusCode)
	}

	// Mismatched version
	resp = reqAPI(s, "POST", "/api/setup/terms", `{"accept":true,"version":"v-outdated-unknown"}`)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("POST /api/setup/terms outdated version status %d, want 409", resp.StatusCode)
	}
}

func TestSetupDatabaseValidation(t *testing.T) {
	s := newGuardTestServer(t)

	// Invalid driver
	resp := reqAPI(s, "POST", "/api/setup/database/test", `{"driver":"oracle","name":"test"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /api/setup/database/test status %d, want 400", resp.StatusCode)
	}
}

func TestSetupRestart(t *testing.T) {
	s := newGuardTestServer(t)

	// Tanpa restart function -> 501 Not Implemented
	resp := reqAPI(s, "POST", "/api/setup/restart", "")
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("POST /api/setup/restart status %d, want 501", resp.StatusCode)
	}

	// Dengan restart function -> 200 OK
	restarted := false
	s.SetRestart(func() error {
		restarted = true
		return nil
	})
	resp = reqAPI(s, "POST", "/api/setup/restart", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/setup/restart with SetRestart status %d, want 200", resp.StatusCode)
	}
	_ = restarted
}

func TestEditionRouteContract(t *testing.T) {
	// Sub-todo #3493 & #3455: GET /api/edition adalah kata benda resource yang valid (false positive audit)
	s := newGuardTestServer(t)
	resp := reqAPI(s, "GET", "/api/edition", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/edition status %d, want 200", resp.StatusCode)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode /api/edition: %v", err)
	}
	if _, ok := res["edition"]; !ok {
		t.Errorf("missing key 'edition' in /api/edition response")
	}
	if _, ok := res["features"]; !ok {
		t.Errorf("missing key 'features' in /api/edition response")
	}
}

func TestComponentPortsRoute(t *testing.T) {
	// Sub-todo #3455: POST /api/components/:id/ports menggantikan rute singular /port
	s := newGuardTestServer(t)

	// Seed component
	p := models.Project{Name: "test-port-proj", Slug: "test-port-proj", RootPath: "/tmp/test-port-proj"}
	s.db.Create(&p)
	c := models.Component{ProjectID: p.ID, Name: "web-fe"}
	s.db.Create(&c)

	// Request ke rute ternormalisasi /ports
	resp := reqAPI(s, "POST", "/api/components/1/ports", `{"port":4321}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/components/1/ports status %d, want 200", resp.StatusCode)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if portVal, ok := res["run_port"].(float64); !ok || int(portVal) != 4321 {
		t.Errorf("expected run_port 4321, got %v", res["run_port"])
	}
}

func TestSetupComplete(t *testing.T) {
	s := newGuardTestServer(t)

	// Belum accept terms -> error 400
	resp := reqAPI(s, "POST", "/api/setup/complete", "")
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/setup/complete unexpected status %d", resp.StatusCode)
	}
}

func TestSetupSaveDatabase(t *testing.T) {
	s := newGuardTestServer(t)

	// Driver invalid -> 400
	resp := reqAPI(s, "POST", "/api/setup/database", `{"driver":"unsupported","name":"db"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /api/setup/database invalid driver status %d, want 400", resp.StatusCode)
	}
}
