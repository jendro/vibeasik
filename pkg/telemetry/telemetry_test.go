package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestInstallIDStable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a, b := InstallID(), InstallID()
	if a != b || !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(a) {
		t.Fatalf("id tidak stabil/valid: %q %q", a, b)
	}
}

func TestSendAndOptOut(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var got map[string]any
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(204)
	}))
	defer srv.Close()
	Send(srv.URL, "start", Info{Version: "v1.2.3", Edition: "community", DBDriver: "sqlite", Projects: 3})
	if n != 1 || got["version"] != "v1.2.3" || got["install_id"] == "" || got["os"] == "" {
		t.Fatalf("payload: %v n=%d", got, n)
	}
	for k := range got {
		switch k {
		case "path", "hostname", "user", "project", "ip":
			t.Fatalf("bidang terlarang %q ikut terkirim", k)
		}
	}
	t.Setenv("VIBEASIK_NO_TELEMETRY", "1")
	Send(srv.URL, "start", Info{})
	if n != 1 {
		t.Fatal("opt-out diabaikan")
	}
}
