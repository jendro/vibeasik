package diagnostics

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.email=t@t", "-c", "user.name=t", "-c", "core.excludesFile=/dev/null"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func ruleIDs(findings []models.Finding) []string {
	var ids []string
	for _, f := range findings {
		ids = append(ids, f.RuleID+" "+f.TargetFile)
	}
	sort.Strings(ids)
	return ids
}

func assertRules(t *testing.T, got []models.Finding, want ...string) {
	t.Helper()
	ids := ruleIDs(got)
	sort.Strings(want)
	if len(ids) != len(want) {
		t.Fatalf("temuan = %v, ingin %v", ids, want)
	}
	for i := range ids {
		if ids[i] != want[i] {
			t.Fatalf("temuan = %v, ingin %v", ids, want)
		}
	}
}

func TestEnvHygiene_MultiEnvLeaks(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	writeFiles(t, dir, map[string]string{
		".gitignore":      ".env\n",
		".env.server.jkt": "DB_URL=postgres://jkt\nSTORAGE_DIR=/var/lib/app\n",
	})
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "init")
	writeFiles(t, dir, map[string]string{
		".env.local":   "DB_URL=postgres://local\nexport JWT_SECRET=rahasia\n",
		".env.example": "DB_URL=\n",
	})

	got := inspectEnvHygiene(dir, "be")
	assertRules(t, got,
		"ENV-FILE-TRACKED be/.env.server.jkt",
		"ENV-NOT-IGNORED be/.env.local",
		"ENV-IGNORE-INCOMPLETE be/.gitignore",
		"ENV-EXAMPLE-DRIFT be/.env.example",
	)
	for _, f := range got {
		if f.RuleID == "ENV-EXAMPLE-DRIFT" && (f.Title == "" || !strings.Contains(f.Description, "JWT_SECRET") || strings.Contains(f.Description, "rahasia")) {
			t.Fatalf("drift harus menyebut nama kunci tanpa nilai: %q", f.Description)
		}
	}
}

func TestEnvHygiene_WildcardBlockClean(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	writeFiles(t, dir, map[string]string{
		".gitignore":    envGitignoreBlock + "\n",
		".env.example":  "PORT=\nDB_URL=\n",
		".env.local":    "PORT=8080\nDB_URL=x\n",
		".env.server.a": "PORT=80\n",
		".env.server.b": "DB_URL=y\n",
	})
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "init")
	assertRules(t, inspectEnvHygiene(dir, ""))
}

func TestEnvHygiene_TemplateIgnoredAndMissing(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	writeFiles(t, dir, map[string]string{".gitignore": ".env*\n", ".env.example": "A=\n", ".env": "A=1\n"})
	assertRules(t, inspectEnvHygiene(dir, ""), "ENV-EXAMPLE-IGNORED .gitignore")

	// Di luar repo git: hanya template yang dinilai
	plain := t.TempDir()
	writeFiles(t, plain, map[string]string{".env.server": "A=1\n"})
	assertRules(t, inspectEnvHygiene(plain, ""), "ENV-EXAMPLE-MISSING .env.example")
}

func TestStoragePaths_GoBinary(t *testing.T) {
	src := `package main

import "os"

func main() {
	uploadDir := "./uploads"
	os.MkdirAll("storage/avatars", 0o755)
	app.Static("/files", "./uploads")
	c.SaveFile(file, "uploads/"+name)
	db, _ := gorm.Open(sqlite.Open("app.db"), cfg)
	os.WriteFile(filepath.Join(os.TempDir(), "uploads", name), b, 0o644)
	logo := "/home/dev/project/logo.png"
	_, b, _, _ := runtime.Caller(0)
	base := filepath.Dir(b)

	// aman: berasal dari env / folder binary
	dir := filepath.Join(storageDir(), "uploads")
	os.MkdirAll(os.Getenv("STORAGE_DIR"), 0o755)
	tmp, _ := os.CreateTemp("", "upload-*")
	// os.MkdirAll("uploads", 0o755)
	doc := "contoh: /home/user/apps/notion"
	win := "C:\\Users\\budi\\data"
}
`
	got := inspectStoragePaths(src, "main.go", "go", strings.Split(src, "\n"))
	want := map[int]string{
		6: "STORAGE-RELATIVE-PATH", 7: "STORAGE-RELATIVE-PATH", 8: "STORAGE-RELATIVE-PATH",
		9: "STORAGE-RELATIVE-PATH", 10: "STORAGE-RELATIVE-PATH", 11: "STORAGE-TEMP-DIR",
		12: "STORAGE-MACHINE-PATH", 13: "STORAGE-SOURCE-RELATIVE", 22: "STORAGE-MACHINE-PATH",
	}
	if len(got) != len(want) {
		t.Fatalf("temuan = %v", ruleIDsLines(got))
	}
	for _, f := range got {
		if want[f.Line] != f.RuleID {
			t.Fatalf("baris %d: %s, ingin %s (semua: %v)", f.Line, f.RuleID, want[f.Line], ruleIDsLines(got))
		}
	}
}

func TestStoragePaths_SkippedWithObjectStorage(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"upload.go": "package main\n\nfunc save() {\n\tuploadDir := \"uploads\"\n\t_ = uploadDir\n}\n",
		"s3.go":     "package main\n\nfunc put() { client.PutObject(ctx, in) }\n",
	})
	for _, f := range InspectDirectory(dir, "") {
		if f.RuleID == "STORAGE-RELATIVE-PATH" {
			t.Fatalf("proyek ber-S3 tidak boleh mendapat %s", f.RuleID)
		}
	}
	os.Remove(filepath.Join(dir, "s3.go"))
	found := false
	for _, f := range InspectDirectory(dir, "") {
		found = found || f.RuleID == "STORAGE-RELATIVE-PATH"
	}
	if !found {
		t.Fatal("tanpa object storage, path relatif harus dilaporkan")
	}
}

func ruleIDsLines(fs []models.Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.RuleID+":"+strconv.Itoa(f.Line))
	}
	return out
}
