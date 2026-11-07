package api

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestProjectDetailOmitsDiagnosticsAndListPaginates(t *testing.T) {
	s := syncServer(t)
	if err := s.db.AutoMigrate(&models.Diagnostic{}); err != nil {
		t.Fatal(err)
	}
	p := models.Project{Name: "demo", Slug: "demo"}
	if err := s.db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	base := time.Now()
	for i := 0; i < 5; i++ {
		d := models.Diagnostic{ProjectID: p.ID, Score: i, Summary: fmt.Sprintf("s%d", i),
			FindingsJSON: "[" + strings.Repeat("{}", 1) + "]", CreatedAt: base.Add(time.Duration(i) * time.Minute)}
		if err := s.db.Create(&d).Error; err != nil {
			t.Fatal(err)
		}
	}

	code, detail := syncCall(t, s, "GET", fmt.Sprintf("/api/projects/%d", p.ID), "")
	if code != 200 {
		t.Fatalf("detail status %d", code)
	}
	if _, ok := detail["diagnostics"]; ok {
		t.Fatal("detail must not contain diagnostics")
	}

	code, res := syncCall(t, s, "GET", fmt.Sprintf("/api/projects/%d/diagnostics?limit=2&offset=1", p.ID), "")
	if code != 200 {
		t.Fatalf("list status %d", code)
	}
	items := res["items"].([]any)
	if len(items) != 2 || res["total"].(float64) != 5 || res["limit"].(float64) != 2 || res["offset"].(float64) != 1 {
		t.Fatalf("unexpected page: %v", res)
	}
	first := items[0].(map[string]any)
	if first["score"].(float64) != 3 { // terbaru dulu, offset 1 -> skor 3
		t.Fatalf("order wrong: %v", first)
	}
	if first["findings_json"] != "" {
		t.Fatalf("lite list must omit findings_json: %v", first["findings_json"])
	}

	_, res = syncCall(t, s, "GET", fmt.Sprintf("/api/projects/%d/diagnostics?limit=1&full=1", p.ID), "")
	if res["items"].([]any)[0].(map[string]any)["findings_json"] == "" {
		t.Fatal("full=1 must include findings_json")
	}
	_, res = syncCall(t, s, "GET", fmt.Sprintf("/api/projects/%d/diagnostics", p.ID), "")
	if res["limit"].(float64) != 100 || len(res["items"].([]any)) != 5 {
		t.Fatalf("default limit: %v", res["limit"])
	}
}

func TestLatestProjectDiagnosticOmitsDuplicateFindingsJSON(t *testing.T) {
	s := syncServer(t)
	if err := s.db.AutoMigrate(&models.Diagnostic{}, &models.DeepCheckReport{}); err != nil {
		t.Fatal(err)
	}
	p := models.Project{Name: "lat", Slug: "lat"}
	if err := s.db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	d := models.Diagnostic{ProjectID: p.ID, Score: 90, Summary: "x",
		FindingsJSON: `[{"rule_id":"R1","title":"t","target_file":"a.go","line":1}]`}
	if err := s.db.Create(&d).Error; err != nil {
		t.Fatal(err)
	}
	url := fmt.Sprintf("/api/projects/%d/diagnostics/latest", p.ID)
	code, res := syncCall(t, s, "GET", url, "")
	if code != 200 {
		t.Fatalf("status %d", code)
	}
	if len(res["findings"].([]any)) != 1 {
		t.Fatalf("findings must stay: %v", res["findings"])
	}
	if res["diagnostic"].(map[string]any)["findings_json"] != "" {
		t.Fatal("diagnostic.findings_json must be omitted by default")
	}
	_, res = syncCall(t, s, "GET", url+"?full=1", "")
	if res["diagnostic"].(map[string]any)["findings_json"] == "" {
		t.Fatal("full=1 must keep findings_json")
	}
}
