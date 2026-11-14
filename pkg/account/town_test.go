package account

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/config"
)

func TestTownURLUsesDeviceToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/town/ticket" {
			w.WriteHeader(404)
			return
		}
		w.Write([]byte(`{"url":"https://town.example/login?ticket=abc"}`))
	}))
	defer srv.Close()
	old := config.GetConfig().VibeasikCloudURL
	config.GetConfig().VibeasikCloudURL = srv.URL
	mu.Lock()
	oldState := state
	state = nil
	mu.Unlock()
	t.Cleanup(func() {
		config.GetConfig().VibeasikCloudURL = old
		mu.Lock()
		state = oldState
		mu.Unlock()
	})

	if _, err := TownURL(); err != ErrNotLoggedIn {
		t.Fatalf("tanpa login: %v", err)
	}
	mu.Lock()
	state = &State{DeviceToken: "vbd_uji"}
	mu.Unlock()
	u, err := TownURL()
	if err != nil || u != "https://town.example/login?ticket=abc" || gotAuth != "Bearer vbd_uji" {
		t.Fatalf("url=%q err=%v auth=%q", u, err, gotAuth)
	}
}
