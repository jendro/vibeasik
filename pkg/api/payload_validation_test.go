package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/localauth"
)

type payloadCase struct {
	method, path, body string
	wantErr            string // potongan pesan error 400 yang diharapkan
}

// Todo #3279: payload tak valid ke endpoint pengubah data dijawab 400 dengan pesan jelas.
func TestPayloadValidationRejects(t *testing.T) {
	s := newGuardTestServer(t)
	long := strings.Repeat("x", 21000)
	cases := []payloadCase{
		// Pusat Pengetahuan sync
		{"POST", "/api/knowledge-sync/consent", ``, "body JSON wajib"},
		{"POST", "/api/knowledge-sync/consent", `{"accept":false}`, "accept: true"},
		{"POST", "/api/knowledge-sync/consent", `{"accept":true}`, "version"},
		{"POST", "/api/knowledge-sync/mode", `{"mode":"bogus"}`, "salah satu dari"},
		{"POST", "/api/knowledge-sync/mode", `{}`, "mode wajib"},
		{"POST", "/api/knowledge-sync/revoke", `{"server_id":0}`, "server_id"},
		// Proyek & organizer
		{"POST", "/api/projects", ``, "body JSON wajib"},
		{"POST", "/api/projects", `{"name":"  "}`, "name wajib"},
		{"POST", "/api/projects", `{"name":"` + strings.Repeat("a", 300) + `"}`, "maksimal"},
		{"POST", "/api/projects", `{"name":`, "tidak valid"},
		{"PUT", "/api/projects/1", `{"name":"` + strings.Repeat("a", 300) + `"}`, "maksimal"},
		{"POST", "/api/projects/0/consolidate", `{}`, "invalid project id"},
		{"POST", "/api/projects/1/consolidate", `{"target_parent_dir":"relatif/x"}`, "path absolut"},
		{"POST", "/api/organizer/plan", `{"target_directory":"relatif"}`, "path absolut"},
		{"POST", "/api/organizer/plan", `[`, "tidak valid"},
		{"POST", "/api/organizer/execute", `{"moves":[]}`, "tidak ada folder"},
		{"PUT", "/api/settings/workspace", `{"default_projects_root":"rel"}`, "path absolut"},
		// Todo
		{"POST", "/api/todos", `{"title":"  "}`, "title"},
		{"POST", "/api/todos", `{"title":"ok","status":"aneh"}`, "status"},
		{"POST", "/api/todos", `{"title":"ok","priority":"urgent"}`, "priority"},
		{"POST", "/api/todos", `{"title":"ok","description":"` + long + `"}`, "description"},
		{"POST", "/api/todos", `{"title":"ok","level":"galaxy"}`, "level"},
		{"PUT", "/api/todos/1", `{"status":"aneh"}`, "status"},
		{"PUT", "/api/todos/1", `{"title":123}`, "title harus berupa teks"},
		{"PUT", "/api/todos/1", `{"title":" "}`, "title"},
		{"POST", "/api/todos/1/approve", `{"status":"aneh"}`, "status"},
		{"POST", "/api/todos/suggest", `{"project_id":"x"}`, "tidak valid"},
		// Pengetahuan
		{"POST", "/api/knowledge/nodes", `{"title":"t"}`, "content"},
		{"POST", "/api/knowledge/nodes", `{"content":"c"}`, "title"},
		{"POST", "/api/knowledge/nodes/0/feedback", `{"helpful":true}`, "invalid node id"},
		{"POST", "/api/knowledge/nodes/1/feedback", ``, "body JSON wajib"},
		{"POST", "/api/knowledge/recall", `{"text":""}`, "text"},
		{"POST", "/api/knowledge/recall", `{"text":"a","limit":9999}`, "limit"},
		{"POST", "/api/knowledge/gaps", `{"topic":""}`, "topic"},
		{"POST", "/api/knowledge/gaps/0/dismiss", ``, "invalid gap id"},
		// Komponen, rute, port, runner
		{"PUT", "/api/components/1/routes/1", `{"description":"` + long + `"}`, "description"},
		{"POST", "/api/components/1/routes/sync", ``, ""}, // tanpa payload: id valid, lolos validasi (bukan 400 payload)
		{"POST", "/api/components/1/port", `{"port":70000}`, "port"},
		{"POST", "/api/components/1/port", `{rusak`, "tidak valid"},
		{"POST", "/api/projects/abc/ensure-unique-ports", ``, "invalid project id"},
		{"POST", "/api/projects/0/ensure-unique-ports", ``, "invalid project id"},
		// Pengaturan LLM
		{"PUT", "/api/settings/llm", ``, "body JSON wajib"},
		{"POST", "/api/settings/llm/active-provider", `{"provider":"zzz"}`, "provider"},
		{"POST", "/api/settings/llm/select-model", `{"model":" "}`, "model"},
		{"POST", "/api/settings/llm/configs", `{"name":"a","url":"ftp://x"}`, "http"},
		{"POST", "/api/settings/llm/configs", `{"name":"a","url":"http://x","provider":"zzz"}`, "provider"},
		{"PUT", "/api/settings/llm/configs/1", `{"name":"` + strings.Repeat("a", 300) + `"}`, "name"},
		{"PUT", "/api/settings/llm/configs/1/vision", `{"model":""}`, "model"},
		{"PUT", "/api/settings/llm/configs/1/alias", `{"model":"m","alias":"` + strings.Repeat("a", 300) + `"}`, "alias"},
		// vibeasik.com
		{"POST", "/api/vibeasik/deploy/generate-plan", ``, "body JSON wajib"},
		{"POST", "/api/vibeasik/security/cloud-audit", `{`, "tidak valid"},
		{"POST", "/api/vibeasik/ai-templates/inject", `{"target_path":"/tmp/x"}`, "template_id"},
		{"POST", "/api/vibeasik/ai-templates/inject", `{"target_path":"rel","template_id":"a"}`, "path absolut"},
		// Traffic
		{"POST", "/api/traffic/sources", `{"mode":"zzz"}`, "mode"},
		{"POST", "/api/traffic/sources", `{"sync_interval_sec":-5}`, "sync_interval_sec"},
		{"PUT", "/api/traffic/sources/1", ``, "body JSON wajib"},
		{"PUT", "/api/traffic/sources/abc", `{}`, "invalid traffic source id"},
		{"POST", "/api/traffic/sources/0/sync", ``, "invalid traffic source id"},
		{"POST", "/api/traffic/sources/1%20OR%201=1/rotate-key", ``, "invalid traffic source id"},
		// Setup & update
		{"POST", "/api/setup/account", `{"choice":"zzz"}`, "choice"},
		{"POST", "/api/setup/terms", `{"accept":true}`, "version"},
		{"POST", "/api/setup/database/test", `{"driver":"oracle","name":"x"}`, "driver"},
		{"POST", "/api/setup/database", `{"driver":"sqlite"}`, "name"},
		{"PUT", "/api/update/auto", ``, "body JSON wajib"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Host = "localhost"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(localauth.Header, testLocalToken)
		resp, err := s.app.Test(req, -1)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		code, out := decodeResp(t, resp)
		if tc.wantErr == "" {
			if code == 400 {
				t.Errorf("%s %s → 400 %v, want bukan 400", tc.method, tc.path, out)
			}
			continue
		}
		msg, _ := out["error"].(string)
		if code != 400 || !strings.Contains(msg, tc.wantErr) {
			t.Errorf("%s %s → %d %q, want 400 memuat %q", tc.method, tc.path, code, msg, tc.wantErr)
		}
	}
}

// Endpoint tanpa body tetap wajib id > 0 (dan bukan teks) — tanpa menyentuh DB.
func TestMutationIDsMustBePositive(t *testing.T) {
	s := newGuardTestServer(t)
	routes := []string{
		"POST /api/components/%s/routes/sync", "POST /api/components/%s/stop",
		"POST /api/projects/%s/stop", "POST /api/todos/%s/reject", "POST /api/todos/%s/approve",
		"POST /api/settings/llm/configs/%s/activate", "DELETE /api/todos/%s", "DELETE /api/projects/%s",
		"POST /api/projects/%s/deprecate", "POST /api/projects/%s/consolidate", "DELETE /api/components/%s",
	}
	for _, r := range routes {
		method, tmpl, _ := strings.Cut(r, " ")
		for _, bad := range []string{"0", "-1", "abc"} {
			req := httptest.NewRequest(method, strings.Replace(tmpl, "%s", bad, 1), strings.NewReader(`{}`))
			req.Host = "localhost"
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(localauth.Header, testLocalToken)
			resp, err := s.app.Test(req, -1)
			if err != nil {
				t.Fatal(err)
			}
			code, out := decodeResp(t, resp)
			if code != 400 || !strings.Contains(out["error"].(string), "invalid") {
				t.Errorf("%s dengan id %q → %d %v, want 400 invalid ... id", r, bad, code, out)
			}
		}
	}
}

// Payload sah tetap diproses seperti sebelumnya (respons sukses tidak berubah).
func TestPayloadValidationAcceptsValid(t *testing.T) {
	s := newGuardTestServer(t)
	var lastErr string
	do := func(method, path, body string) int {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Host = "localhost"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(localauth.Header, testLocalToken)
		resp, err := s.app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		code, out := decodeResp(t, resp)
		lastErr, _ = out["error"].(string)
		return code
	}
	if c := do("POST", "/api/todos", `{"title":"tes validasi","status":"pending","priority":"high","level":"global"}`); c != 201 {
		t.Errorf("createTodo sah → %d, want 201", c)
	}
	if c := do("POST", "/api/projects", `{"name":"proyek-sah","root_path":"-"}`); c != 201 {
		t.Errorf("createProject sah → %d, want 201", c)
	}
	if c := do("PUT", "/api/todos/1", `{"priority":"low"}`); c != 200 && c != 404 {
		t.Errorf("updateTodo sah → %d", c)
	}
	// Body kosong tetap diterima di endpoint dengan payload opsional.
	do("POST", "/api/projects/1/consolidate", ``)
	if strings.Contains(lastErr, "body JSON") {
		t.Errorf("consolidate tanpa body tidak boleh ditolak sebagai payload: %q", lastErr)
	}
}
