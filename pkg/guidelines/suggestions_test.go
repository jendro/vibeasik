package guidelines

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

var updateGolden = flag.Bool("update", false, "tulis ulang berkas golden")

func writeFiles(t *testing.T, root string, files ...string) {
	t.Helper()
	for _, f := range files {
		p := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestGenerateSuggestionsForProjectGolden mengunci seluruh saran todo untuk beberapa topologi proyek.
func TestGenerateSuggestionsForProjectGolden(t *testing.T) {
	type scenario struct {
		name string
		make func(t *testing.T) *models.Project
	}
	scenarios := []scenario{
		{"nil", func(t *testing.T) *models.Project { return nil }},
		{"kosong_tanpa_root", func(t *testing.T) *models.Project {
			return &models.Project{ID: 1, Name: "Kosong", RootPath: "-"}
		}},
		{"bundle_penuh_kosong_berkas", func(t *testing.T) *models.Project {
			root, be, fe, mob := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
			writeFiles(t, root, ".env")
			return &models.Project{ID: 2, Name: "Bundle", RootPath: root, Components: []models.Component{
				{ID: 10, Name: "api", Role: models.RoleBackend, AbsolutePath: be, Port: 8080},
				{ID: 11, Name: "web", Role: models.RoleFrontendClient, Framework: "nuxt", AbsolutePath: fe},
				{ID: 12, Name: "app", Role: models.RoleFrontendMobile, Framework: "flutter", AbsolutePath: mob},
			}}
		}},
		{"bundle_lengkap", func(t *testing.T) *models.Project {
			root, be, fe := t.TempDir(), t.TempDir(), t.TempDir()
			writeFiles(t, root, ".env", ".env.example", ".gitignore", "AI_CONTEXT.md", "CHANGELOG.md", "Dockerfile")
			writeFiles(t, be, "internal/x_test.go")
			writeFiles(t, fe, "composables/useApi.ts", "a.spec.ts")
			return &models.Project{ID: 3, Name: "Lengkap", RootPath: root, Components: []models.Component{
				{ID: 20, Name: "api", Role: models.RoleBackend, AbsolutePath: be, HealthURL: "http://x/h"},
				{ID: 21, Name: "web", Role: models.RoleFrontendAdmin, Framework: "vue", AbsolutePath: fe},
			}}
		}},
		{"dockerfile_di_komponen_route_health", func(t *testing.T) *models.Project {
			root, be, fe := t.TempDir(), t.TempDir(), t.TempDir()
			writeFiles(t, be, "dockerfile", "main_test.go")
			writeFiles(t, fe, "__tests__/a.ts")
			return &models.Project{ID: 4, Name: "Dock", RootPath: root, Components: []models.Component{
				{ID: 30, Name: "api", Role: models.RoleBackend, AbsolutePath: be, Routes: []models.ComponentRoute{{Path: "/api/ping"}}},
				{ID: 31, Name: "web", Role: models.RoleFrontendClient, Framework: "nuxt", AbsolutePath: fe},
			}}
		}},
	}
	var out strings.Builder
	for _, sc := range scenarios {
		p := sc.make(t)
		got := GenerateSuggestionsForProject(p, nil)
		b, err := json.MarshalIndent(got, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		if p != nil && p.RootPath != "" {
			s = strings.ReplaceAll(s, p.RootPath, "<root>")
		}
		out.WriteString("### " + sc.name + "\n" + s + "\n")
	}
	golden := filepath.Join("testdata", "suggestions.golden")
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
		t.Fatalf("saran berubah; bandingkan %s dengan %s.got", golden, golden)
	}
}

func TestFilePredicates(t *testing.T) {
	var buf bytes.Buffer
	origWriter := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(origWriter)

	tmp := t.TempDir()
	existing := filepath.Join(tmp, "ada.txt")
	if err := os.WriteFile(existing, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(tmp, "tidak-ada.txt")

	if !fileExists(existing) {
		t.Errorf("fileExists(%q) = false, want true", existing)
	}
	if fileExists(missing) {
		t.Errorf("fileExists(%q) = true, want false", missing)
	}
	if fileMissing(existing) {
		t.Errorf("fileMissing(%q) = true, want false", existing)
	}
	if !fileMissing(missing) {
		t.Errorf("fileMissing(%q) = false, want true", missing)
	}
	if buf.Len() > 0 {
		t.Errorf("expected no warning logs for normal missing/existing files, got: %s", buf.String())
	}

	// Normal missing directory for dirHasFileSuffix -> return false tanpa warning log
	buf.Reset()
	missingDir := filepath.Join(tmp, "missing_dir")
	if dirHasFileSuffix(missingDir, ".go") {
		t.Errorf("dirHasFileSuffix(%q) = true, want false", missingDir)
	}
	if buf.Len() > 0 {
		t.Errorf("expected no warning logs for normal missing dir, got: %s", buf.String())
	}

	// Normal existing directory with file matching suffix
	subDir := filepath.Join(tmp, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !dirHasFileSuffix(subDir, ".go") {
		t.Errorf("dirHasFileSuffix(%q, .go) = false, want true", subDir)
	}

	// Error sistem non-IsNotExist (permission denied) -> log [WARN] dan perilaku aman
	if os.Geteuid() != 0 {
		forbiddenDir := filepath.Join(tmp, "forbidden")
		if err := os.Mkdir(forbiddenDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(forbiddenDir, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(forbiddenDir, 0o700)
		})

		// 1. fileExists pada unreadable path -> false dan log [WARN]
		buf.Reset()
		unreadable := filepath.Join(forbiddenDir, "secret.txt")
		if fileExists(unreadable) {
			t.Errorf("fileExists(%q) = true, want false on permission error", unreadable)
		}
		if !strings.Contains(buf.String(), "[WARN] fileExists: os.Stat") {
			t.Errorf("expected log to contain '[WARN] fileExists: os.Stat', got: %s", buf.String())
		}

		// 2. fileMissing pada unreadable path -> false dan log [WARN] (fail-safe: jangan memicu saran keliru)
		buf.Reset()
		if fileMissing(unreadable) {
			t.Errorf("fileMissing(%q) = true, want false on permission error", unreadable)
		}
		if !strings.Contains(buf.String(), "[WARN] fileMissing: os.Stat") {
			t.Errorf("expected log to contain '[WARN] fileMissing: os.Stat', got: %s", buf.String())
		}

		// 3. dirHasFileSuffix pada forbidden directory -> false dan log [WARN]
		buf.Reset()
		if dirHasFileSuffix(forbiddenDir, ".go") {
			t.Errorf("dirHasFileSuffix(%q) = true, want false on permission error", forbiddenDir)
		}
		if !strings.Contains(buf.String(), "[WARN] dirHasFileSuffix: os.ReadDir") {
			t.Errorf("expected log to contain '[WARN] dirHasFileSuffix: os.ReadDir', got: %s", buf.String())
		}
	}
}

