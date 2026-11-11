package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestGetTodosSendsUnpagedStatusCounts(t *testing.T) {
	s := syncServer(t)
	if err := s.db.AutoMigrate(&models.Todo{}); err != nil {
		t.Fatal(err)
	}
	p := models.Project{Name: "Hitung", Slug: "hitung-todo"}
	s.db.Create(&p)
	for i := 0; i < 3; i++ {
		s.db.Create(&models.Todo{ProjectID: &p.ID, Title: "selesai", Status: models.TodoStatusCompleted})
	}
	parent := models.Todo{ProjectID: &p.ID, Title: "induk", Status: models.TodoStatusPending}
	s.db.Create(&parent)
	s.db.Create(&models.Todo{ProjectID: &p.ID, Title: "sub", Status: models.TodoStatusPending, ParentID: &parent.ID})

	resp := doReq(t, s, "GET", fmt.Sprintf("/api/todos?limit=1&project_id=%d", p.ID), "localhost", map[string]string{localauth.Header: testLocalToken})
	var counts map[string]int
	if err := json.Unmarshal([]byte(resp.Header.Get("X-Todo-Status-Counts")), &counts); err != nil {
		t.Fatalf("header hitungan tidak valid: %q", resp.Header.Get("X-Todo-Status-Counts"))
	}
	if counts[models.TodoStatusCompleted] != 3 || counts[models.TodoStatusPending] != 1 {
		t.Fatalf("hitungan salah (harus tanpa batas halaman & tanpa sub todo): %v", counts)
	}
}
