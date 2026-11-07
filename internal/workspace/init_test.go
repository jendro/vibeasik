package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/internal/codemap"
	"github.com/vibeasik/vibeasik/internal/models"
)

func TestInitProjectWritesMapIdentifierAndGitignore(t *testing.T) {
	root := t.TempDir()
	for rel, content := range map[string]string{
		".git/HEAD":     "ref: refs/heads/main\n",
		".gitignore":    "node_modules",
		"be/main.go":    "package main\n\n// Run menjalankan server.\nfunc Run() {}\n",
		"fe/app.vue":    "<template><div /></template>\n",
		"AI_CONTEXT.md": "# Konteks\n",
	} {
		p := filepath.Join(root, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	_ = db.AutoMigrate(&models.Project{}, &models.Component{}, &models.ComponentRoute{}, &models.CodeMap{}, &models.CodeMapEntry{})
	p := models.Project{Name: "demo", Slug: "demo", RootPath: root, Components: []models.Component{
		{Name: "BE", AbsolutePath: filepath.Join(root, "be")}, {Name: "FE", AbsolutePath: filepath.Join(root, "fe")}}}
	db.Create(&p)

	res, err := InitProject(db, p.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.CodeMapMarkdown != filepath.Join(root, ".vibeasik", "CODEMAP.md") || !res.GitignoreUpdated || len(res.FilesCreated) < 2 {
		t.Fatalf("unexpected init result: %+v", res)
	}
	gi, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if string(gi) != "node_modules\n\n# vibeasik: peta kode & cache yang dibuat ulang otomatis\n.vibeasik/\n" {
		t.Fatalf("gitignore: %q", gi)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "be", IdentifierFileName))
	var idf ProjectIdentifier
	_ = json.Unmarshal(raw, &idf)
	if idf.CodeMap == nil || idf.CodeMap.Markdown != res.CodeMapMarkdown || !strings.Contains(idf.AIAgentInstructions, "vibeasik_code_map_search") {
		t.Fatalf("identifier must point agents to the code map: %+v", idf)
	}

	// Init ulang tidak menggandakan baris .gitignore.
	if res, _ = InitProject(db, p.ID, ""); res.GitignoreUpdated {
		t.Fatal(".gitignore must not be appended twice")
	}

	// Aplikasi berubah → Sync mem-parse ulang berkas itu dan menulis ulang CODEMAP.md; tanpa perubahan tidak ditulis.
	if s, _ := codemap.Sync(db, p.ID, false); s.Exported {
		t.Fatalf("no change must not rewrite the map: %+v", s.Build)
	}
	later := time.Now().Add(time.Minute)
	newFile := filepath.Join(root, "be", "billing.go")
	_ = os.WriteFile(newFile, []byte("package main\n\n// HitungTagihan menghitung tagihan bulanan.\nfunc HitungTagihan() {}\n"), 0o644)
	_ = os.Chtimes(newFile, later, later)
	s, err := codemap.Sync(db, p.ID, false)
	if err != nil || !s.Exported || s.Build.Parsed != 1 {
		t.Fatalf("changed app must refresh the map: %+v %v", s, err)
	}
	md, _ := os.ReadFile(res.CodeMapMarkdown)
	if !strings.Contains(string(md), "billing.go") {
		t.Fatal("CODEMAP.md must list the new file")
	}
}
