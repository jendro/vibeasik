package service_test

import (
	"context"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestDiagnoseProjectSummary(t *testing.T) {
	services, db := setupTestServices(t)
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	writeFiles(t, dir, map[string]string{"main.go": "package main\nfunc main(){}\n"})
	p := models.Project{Name: "Diag", Slug: "diag", RootPath: dir, Components: []models.Component{
		{Name: "api", Role: models.RoleBackend, Language: "go", AbsolutePath: dir},
	}}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	var out []any
	for i := 0; i < 2; i++ { // kedua: menggabung dengan temuan sebelumnya
		d, err := services.Diagnostics.DiagnoseProject(ctx, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, map[string]any{"score": d.Score, "summary": d.Summary, "findings": d.FindingsJSON})
	}
	if _, err := services.Diagnostics.DiagnoseProject(ctx, 9999); err == nil {
		t.Fatal("want error")
	} else {
		out = append(out, err.Error())
	}
	golden(t, "diagnose_project", out, dir)
}
