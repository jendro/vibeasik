package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAgentRulesManagedBlock(t *testing.T) {
	root := t.TempDir()
	user := "# Aturan tim\n\nPakai bahasa Indonesia.\n"
	if err := os.WriteFile(filepath.Join(root, "GEMINI.md"), []byte(user), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := WriteAgentRules(root, 7, "Toko Kopi")
	if err != nil || len(files) != 2 { // GEMINI.md diperbarui, AGENTS.md dibuat, CLAUDE.md tidak ada → dilewati
		t.Fatalf("berkas: %v %v", files, err)
	}
	if _, err := os.Stat(filepath.Join(root, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Error("CLAUDE.md tidak boleh dibuat bila proyek belum memakainya")
	}
	g, _ := os.ReadFile(filepath.Join(root, "GEMINI.md"))
	gs := string(g)
	for _, want := range []string{rulesBegin, "vibeasik_knowledge_recall", "vibeasik_design_guidance", "project_id: 7", "Toko Kopi", user} {
		if !strings.Contains(gs, want) {
			t.Errorf("GEMINI.md harus memuat %q:\n%s", want, gs)
		}
	}
	if strings.Index(gs, rulesBegin) > strings.Index(gs, "# Aturan tim") {
		t.Error("blok protokol harus di awal berkas")
	}

	// Init ulang: tidak ada yang ditulis; ganti nama proyek → blok diganti, bukan digandakan, teks pengguna utuh.
	if files, _ = WriteAgentRules(root, 7, "Toko Kopi"); len(files) != 0 {
		t.Errorf("tanpa perubahan tidak menulis: %v", files)
	}
	_ = os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# Claude\n"), 0o644)
	if _, err := WriteAgentRules(root, 7, "Kopi Senja"); err != nil {
		t.Fatal(err)
	}
	g, _ = os.ReadFile(filepath.Join(root, "GEMINI.md"))
	gs = string(g)
	if strings.Count(gs, rulesBegin) != 1 || strings.Contains(gs, "Toko Kopi") || !strings.Contains(gs, "Kopi Senja") || !strings.HasSuffix(gs, user) {
		t.Errorf("blok harus diganti di tempat:\n%s", gs)
	}
	if c, _ := os.ReadFile(filepath.Join(root, "CLAUDE.md")); !strings.Contains(string(c), rulesBegin) || !strings.Contains(string(c), "# Claude") {
		t.Errorf("CLAUDE.md yang sudah ada diberi blok: %s", c)
	}
}
