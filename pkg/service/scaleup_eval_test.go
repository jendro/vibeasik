package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestEvaluateComponentGolden(t *testing.T) {
	services, db := setupTestServices(t)
	ctx := context.Background()
	fat := strings.Repeat("x\n", 320)

	cases := map[string]struct {
		path  string
		files map[string]string
	}{
		"empty":   {path: "-"},
		"missing": {path: "/nonexistent/zzz-dir"},
		"plain":   {files: map[string]string{"main.go": "package main\n"}},
		"mcp_native": {files: map[string]string{
			"service/a.go": "x", "repository/b.go": "x", "mcp/c.go": "x", "planners/d.go": "x",
			"api/routes.go": fat, "a.go": fat, "b.go": fat, "c.go": fat,
		}},
		"clean": {files: map[string]string{"core/a.go": "x", "adapters/b.go": "x", "api/r.ts": fat}},
		"layered": {files: map[string]string{
			"internal/services/a.go": "x", "internal/repo/b.go": "x", "node_modules/x/y.js": fat,
		}},
		"vertical": {files: map[string]string{"features/a.py": fat, "ai/b.py": "x", "db/c.php": "x"}},
		"agents":   {files: map[string]string{"agents/a.go": "x", "usecases/b.go": "x", "slices/z.go": "x"}},
	}
	names := []string{"empty", "missing", "plain", "mcp_native", "clean", "layered", "vertical", "agents"}
	for _, n := range names {
		c := cases[n]
		dir := c.path
		if dir == "" {
			dir = t.TempDir()
			writeFiles(t, dir, c.files)
		}
		comp := models.Component{Name: n, Role: models.RoleBackend, Language: "go", AbsolutePath: dir}
		if err := db.Create(&comp).Error; err != nil {
			t.Fatal(err)
		}
		ev, err := services.ScaleUp.EvaluateComponent(ctx, comp.ID)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		golden(t, "scaleup_eval_"+n, ev, dir)
	}
	if _, err := services.ScaleUp.EvaluateComponent(ctx, 99999); err == nil {
		t.Fatal("expected error")
	}
}
