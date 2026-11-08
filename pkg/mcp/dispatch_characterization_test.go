package mcp

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/models"
)

var updateGolden = flag.Bool("update", false, "tulis ulang berkas golden")

var (
	reTime = regexp.MustCompile(`"(created_at|updated_at|CreatedAt|UpdatedAt|completed_at|last_[a-z_]+|built_at)": "[^"]*"`)
	reDate = regexp.MustCompile(`\d{1,2} [A-Z][a-z]{2} \d{4} \d{2}:\d{2}`)
	reTmp  = regexp.MustCompile(`/tmp/[^\s"]+|/var/folders/[^\s"]+`)
)

func newCharServer(t *testing.T) *Server {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv(envOrchestratorRunID, "")
	db, err := database.InitDB(filepath.Join(t.TempDir(), "char.db"))
	if err != nil {
		t.Fatal(err)
	}
	return NewServer(db)
}

func renderResult(res *ToolCallResult, err error) string {
	var b strings.Builder
	if err != nil {
		b.WriteString("ERR: " + err.Error() + "\n")
	}
	if res != nil {
		for _, c := range res.Content {
			b.WriteString(c.Type + ": " + c.Text + "\n")
		}
	}
	out := reTime.ReplaceAllString(b.String(), `"$1": "<t>"`)
	out = reDate.ReplaceAllString(out, "<date>")
	return reTmp.ReplaceAllString(out, "<tmp>")
}

// TestCallToolCharacterization mengunci hasil/error tool representatif lewat callTool.
func TestCallToolCharacterization(t *testing.T) {
	s := newCharServer(t)
	root := t.TempDir()
	p := models.Project{Name: "Demo", Slug: "demo", RootPath: root}
	if err := s.db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	type call struct {
		name string
		args map[string]interface{}
	}
	calls := []call{
		{"no_such_tool", nil},
		{"list_projects", nil},
		{"list_projects", map[string]interface{}{"detail": true}},
		{"get_project_detail", map[string]interface{}{"slug": "demo"}},
		{"get_project_detail", map[string]interface{}{"slug": "nope"}},
		{"scan_directory", nil},
		{"run_diagnostics", map[string]interface{}{"project_id": "999"}},
		{"get_ai_recommendations", map[string]interface{}{"project_id": 999.0}},
		{"vibeasik_search_domain", nil},
		{"vibeasik_inject_ai_template", nil},
		{"vibeasik_check_ports", nil},
		{"vibeasik_check_ports", map[string]interface{}{"ports": []interface{}{0.0, -1.0}}},
		{"vibeasik_rea_analyze", nil},
		{"link_project_workspace", nil},
		{"create_todo", nil},
		{"create_todo", map[string]interface{}{"title": " Tulis dokumentasi API ", "project_id": 1.0, "category": " Docs ", "tags": "a,b", "priority": "high"}},
		{"create_todo", map[string]interface{}{"title": "Tulis dokumentasi API", "project_id": 1.0}},
		{"create_todo", map[string]interface{}{"title": "Perbaiki login", "is_ai_suggestion": true, "level": "global", "completed_by_agent": "x", "completed_at": "2026-01-02T03:04:05Z"}},
		{"update_todo", nil},
		{"update_todo", map[string]interface{}{"todo_id": 9999.0}},
		{"update_todo", map[string]interface{}{"todo_id": "1", "status": "in_progress", "title": "Judul Baru", "category": "Backend", "priority": "low"}},
		{"update_todo", map[string]interface{}{"todo_id": 1.0, "status": "completed", "completion_note": "selesai", "completed_by_agent": "bot"}},
		{"update_todo", map[string]interface{}{"todo_id": 1, "status": "pending"}},
		{"list_todos", nil},
		{"list_todos", map[string]interface{}{"project_id": 1.0, "detail": true, "status": "pending"}},
		{"list_todos", map[string]interface{}{"include_closed": true, "limit": 1.0, "category": "all", "source": "all", "tag": "all"}},
		{"list_todos", map[string]interface{}{"query": "dokumentasi", "project_id": "1"}},
		{"list_todos", map[string]interface{}{"creator_type": "ai", "is_ai_suggestion": false, "component_id": 5.0}},
		{"list_guidelines", map[string]interface{}{"category": "zzz-none"}},
		{"suggest_todos", map[string]interface{}{"project_id": 1.0, "save_to_db": false}},
		{"suggest_todos", map[string]interface{}{"save_to_db": false}},
		{"delete_todo", nil},
		{"delete_todo", map[string]interface{}{"todo_id": 9999.0}},
		{"update_project_status", map[string]interface{}{"project_id": 1.0, "action": "bogus"}},
		{"update_project_status", map[string]interface{}{"project_id": 1.0, "action": "hold"}},
		{"fast_scan", nil},
		{"evaluate_architecture", nil},
		{"consolidate_project", nil},
		{"set_finding_status", map[string]interface{}{"project_id": 1.0, "finding_id": "x", "status": "bad"}},
		{"get_latest_diagnostic", nil},
		{"get_todo_impact_dimensions", nil},
		{"vibeasik_assign_port", nil},
		{"vibeasik_run_component", nil},
		{"vibeasik_stop_component", nil},
		{"vibeasik_run_status", nil},
		{"vibeasik_code_map_search", map[string]interface{}{"project_id": 1.0}},
		{"vibeasik_code_map_status", map[string]interface{}{"project_id": 1.0}},
		{"vibeasik_code_map_file", map[string]interface{}{"project_id": 1.0, "file": "x.go"}},
		{"vibeasik_code_map_routes", map[string]interface{}{"project_id": 1.0}},
		{"vibeasik_code_map_unknown", map[string]interface{}{"project_id": 1.0}},
	}
	var out strings.Builder
	for _, c := range calls {
		res, err := s.callTool(c.name, c.args)
		out.WriteString("### " + c.name + "\n" + renderResult(res, err))
	}
	golden := filepath.Join("testdata", "calltool.golden")
	if *updateGolden {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(golden, []byte(out.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != string(want) {
		_ = os.WriteFile(golden+".got", []byte(out.String()), 0o644)
		t.Fatalf("hasil callTool berubah; bandingkan %s dengan %s.got", golden, golden)
	}
}

// TestCommunityToolListLocked mengunci daftar nama tool Community (urutan diabaikan).
func TestCommunityToolListLocked(t *testing.T) {
	s := &Server{}
	var names []string
	for _, x := range s.getToolsList() {
		names = append(names, x.Name)
	}
	sort.Strings(names)
	got := strings.Join(names, "\n") + "\n"
	golden := filepath.Join("testdata", "toollist.golden")
	if *updateGolden {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("daftar tool berubah:\n%s", got)
	}
}
