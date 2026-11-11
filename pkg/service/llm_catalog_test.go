package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverLLMModelsReadsDisplayName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer k1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"object":"list","data":[
			{"id":"vendor/model-b","display_name":"Model B"},
			{"id":"vendor/model-a","name":"Model A"},
			{"id":"vendor/model-a","display_name":"duplikat"},
			{"id":"plain","display_name":"plain"},
			{"id":""}]}`))
	}))
	defer srv.Close()

	got, err := DiscoverLLMModels(context.Background(), srv.URL+"/v1/chat/completions", "k1")
	if err != nil {
		t.Fatal(err)
	}
	want := []DiscoveredModel{{ID: "plain"}, {ID: "vendor/model-a", DisplayName: "Model A"}, {ID: "vendor/model-b", DisplayName: "Model B"}}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].DisplayName != want[i].DisplayName {
			t.Fatalf("item %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestDiscoverLLMModelsRejectsErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	if _, err := DiscoverLLMModels(context.Background(), srv.URL+"/v1", ""); err == nil {
		t.Fatal("want error on 401")
	}
}

func TestParseModelCatalogPlainArray(t *testing.T) {
	got, err := parseModelCatalog([]byte(`[{"id":"x","display_name":"X"}]`))
	if err != nil || len(got) != 1 || got[0].DisplayName != "X" {
		t.Fatalf("got %+v err %v", got, err)
	}
}
