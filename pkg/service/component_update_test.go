package service_test

import (
	"context"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestComponentUpdateGolden(t *testing.T) {
	services, db := setupTestServices(t)
	ctx := context.Background()
	c := models.Component{Name: "n", Role: models.RoleBackend, RelativePath: "r", AbsolutePath: "/a", Language: "go",
		Framework: "f", PackageManager: "pm", DevCommand: "dev", BuildCommand: "b", Port: 1, HealthURL: "h", EnvFile: "e", EnvSummary: "s"}
	if err := db.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	var out []any
	for _, u := range []models.Component{
		{},
		{Name: "N2", Role: models.RoleCLI, RelativePath: "R2", AbsolutePath: "/A2", Language: "ts"},
		{Framework: "F3", PackageManager: "PM3", DevCommand: "D3", BuildCommand: "B3", Port: 33, HealthURL: "H3", EnvFile: "E3", EnvSummary: "S3"},
	} {
		u := u
		r, err := services.Components.Update(ctx, c.ID, &u)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if _, err := services.Components.Update(ctx, 9999, &models.Component{}); err == nil {
		t.Fatal("want error")
	}
	golden(t, "component_update", out, "")
}
