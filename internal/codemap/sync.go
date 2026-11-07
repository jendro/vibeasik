package codemap

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/models"
)

// AutoSyncInterval adalah jeda pemeriksaan peta proyek yang sudah di-init (pernah ditulis ke repo).
const AutoSyncInterval = 2 * time.Minute

// MarkdownPath mengembalikan lokasi CODEMAP.md untuk root efektif proyek.
func MarkdownPath(root string) string {
	return filepath.Join(root, ".vibeasik", "CODEMAP.md")
}

// RootOf mengembalikan root efektif peta proyek yang tercatat pada build terakhir.
func RootOf(db *gorm.DB, projectID uint) string {
	var meta models.CodeMap
	db.Select("root_path").Where("project_id = ?", projectID).First(&meta)
	return meta.RootPath
}

// ExportedMarkdown mengembalikan path CODEMAP.md proyek bila peta pernah ditulis ke repo dan berkasnya ada.
func ExportedMarkdown(db *gorm.DB, projectID uint) string {
	var meta models.CodeMap
	if db.Where("project_id = ?", projectID).First(&meta).Error != nil || meta.ExportedAt == nil || meta.RootPath == "" {
		return ""
	}
	if p := MarkdownPath(meta.RootPath); exists(p) {
		return p
	}
	return ""
}

// EnsureGitignore menambahkan ".vibeasik/" ke .gitignore root repo git agar peta yang diperbarui otomatis
// tidak mengotori git status, commit agent, maupun merge worktree. Mengembalikan true bila berkas diubah.
func EnsureGitignore(root string) (bool, error) {
	if !exists(filepath.Join(root, ".git")) {
		return false, nil
	}
	p := filepath.Join(root, ".gitignore")
	data, err := os.ReadFile(p)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		switch strings.TrimSpace(line) {
		case ".vibeasik/", ".vibeasik", "/.vibeasik/", "/.vibeasik":
			return false, nil
		}
	}
	add := "# vibeasik: peta kode & cache yang dibuat ulang otomatis\n.vibeasik/\n"
	switch {
	case len(data) == 0:
	case strings.HasSuffix(string(data), "\n"):
		add = "\n" + add
	default:
		add = "\n\n" + add
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.WriteString(add)
	return err == nil, err
}

// SyncResult menjelaskan hasil satu sinkronisasi peta proyek.
type SyncResult struct {
	Build    *BuildResult `json:"build"`
	Exported bool         `json:"exported"`
}

// Sync membangun ulang peta secara inkremental lalu menulis ulang CODEMAP.md hanya bila ada berkas yang berubah
// (atau alwaysExport). Dipakai init, auto-sync, dan setelah ringkasan LLM diperbarui.
func Sync(db *gorm.DB, projectID uint, alwaysExport bool) (*SyncResult, error) {
	res, err := Build(db, projectID, false)
	if err != nil {
		return nil, err
	}
	out := &SyncResult{Build: res}
	if alwaysExport || res.Parsed > 0 || res.Removed > 0 {
		if _, err := Export(db, projectID); err != nil {
			return out, err
		}
		out.Exported = true
	}
	return out, nil
}

// StartAutoSync menjaga peta proyek yang sudah di-init tetap mutakhir saat aplikasi berubah: tiap interval
// berkas yang berubah di-parse ulang dan CODEMAP.md/codemap.jsonl ditulis ulang bila ada perubahan.
// Hanya dijalankan oleh `vibeasik serve`.
func StartAutoSync(ctx context.Context, db *gorm.DB, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				syncInitialized(db)
			}
		}
	}()
}

func syncInitialized(db *gorm.DB) {
	var metas []models.CodeMap
	if db.Where("exported_at IS NOT NULL").Find(&metas).Error != nil {
		return
	}
	for _, m := range metas {
		if m.RootPath == "" || !isDir(m.RootPath) {
			continue
		}
		if res, err := Sync(db, m.ProjectID, false); err != nil {
			log.Printf("[codemap] auto-sync proyek %d gagal: %v", m.ProjectID, err)
		} else if res.Exported {
			log.Printf("[codemap] proyek %d: %d berkas berubah, %d dihapus → CODEMAP.md diperbarui", m.ProjectID, res.Build.Parsed, res.Build.Removed)
		}
	}
}
