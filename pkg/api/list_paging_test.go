package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// Koleksi tetap array; ?limit/&offset opsional; total di X-Total-Count.
func TestProjectListOptionalPaging(t *testing.T) {
	s := syncServer(t)
	for i := 0; i < 5; i++ {
		if err := s.db.Create(&models.Project{Name: fmt.Sprintf("p%d", i), Slug: fmt.Sprintf("p%d", i)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	get := func(path string) ([]models.Project, string) {
		req := httptest.NewRequest("GET", path, nil)
		req.Host = "localhost"
		req.Header.Set(localauth.Header, testLocalToken)
		resp, err := s.app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(resp.Body)
		var out []models.Project
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("bukan array: %s", raw)
		}
		return out, resp.Header.Get("X-Total-Count")
	}
	if all, tot := get("/api/projects"); len(all) != 5 || tot != "5" {
		t.Fatalf("tanpa limit harus utuh: %d %s", len(all), tot)
	}
	if pg, tot := get("/api/projects?limit=2&offset=1"); len(pg) != 2 || tot != "5" {
		t.Fatalf("paged: %d %s", len(pg), tot)
	}
	if pg, _ := get("/api/projects?offset=9"); len(pg) != 0 {
		t.Fatalf("offset lewat batas: %d", len(pg))
	}
}
