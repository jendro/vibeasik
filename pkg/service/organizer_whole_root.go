package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// pathMove adalah satu awalan path yang berpindah (folder lama -> folder baru).
type pathMove struct{ from, to string }

// rebasePath memindahkan p ke awalan baru bila p berada di salah satu folder yang pindah.
func rebasePath(p string, moves []pathMove) (string, bool) {
	for _, m := range moves {
		if p == m.from {
			return m.to, true
		}
		if pathWithin(m.from, p) {
			rel, _ := filepath.Rel(m.from, p)
			return filepath.Join(m.to, rel), true
		}
	}
	return p, false
}

// itemPathMoves mengumpulkan awalan path yang berubah dari item penyatuan, awalan terpanjang lebih dulu agar
// folder bersarang dipetakan oleh item terdekatnya.
func itemPathMoves(items []models.FolderMoveItem) []pathMove {
	var moves []pathMove
	for _, item := range items {
		if item.CurrentPath == "" || item.TargetPath == "" || item.CurrentPath == item.TargetPath {
			continue
		}
		moves = append(moves, pathMove{from: item.CurrentPath, to: item.TargetPath})
	}
	sort.SliceStable(moves, func(i, j int) bool { return len(moves[i].from) > len(moves[j].from) })
	return moves
}

// wholeRootSource mengembalikan root proyek bila proyek sudah berada di satu folder miliknya sendiri, sehingga
// folder itu dipindahkan utuh (berkas di luar folder komponen ikut pindah). "" berarti komponen dipindahkan satu
// per satu: komponen terpencar, atau root hanyalah folder induk umum yang juga memuat hal lain.
func (s *organizerService) wholeRootSource(ctx context.Context, p *models.Project, preview *models.ConsolidatePreview, sources []consolidateSource) string {
	root := strings.TrimSpace(p.RootPath)
	if root == "" || root == "-" || len(sources) == 0 {
		return ""
	}
	root = filepath.Clean(root)
	for _, src := range sources {
		if src.path != root && !pathWithin(root, src.path) {
			return ""
		}
	}
	// Root yang sudah berada di dalam folder tujuan (mis. root = folder komponen) tidak bisa dinaikkan utuh.
	if pathWithin(preview.ProjectTargetDir, root) {
		return ""
	}
	if s.isSharedParentDir(ctx, root, preview.ProjectTargetDir) || !ownsProjectRoot(p, root, preview.ProjectTargetDir) {
		return ""
	}
	if s.rootHoldsOtherProjects(ctx, p.ID, root) {
		return ""
	}
	return root
}

// isSharedParentDir menolak folder yang lazim menjadi induk banyak proyek: akar sistem berkas, folder home dan
// leluhurnya, folder proyek bawaan, serta folder yang memuat tujuan pemindahan itu sendiri.
func (s *organizerService) isSharedParentDir(ctx context.Context, root, projectTargetDir string) bool {
	if filepath.Dir(root) == root || pathWithin(root, projectTargetDir) {
		return true
	}
	shared := []string{models.UserHome(), models.DefaultProjectsRoot()}
	if ws, _ := s.GetWorkspaceSettings(ctx); ws != nil {
		shared = append(shared, ws.DefaultProjectsRoot)
	}
	for _, dir := range shared {
		if dir = strings.TrimSpace(dir); dir == "" {
			continue
		}
		dir = filepath.Clean(dir)
		if root == dir || pathWithin(root, dir) {
			return true
		}
	}
	return false
}

// ownsProjectRoot mencari bukti bahwa root memang folder proyek ini: sudah pernah disatukan, repo git sendiri,
// atau penanda .vibeasik.json milik proyek ini. Bila root sudah tidak ada di disk (pemindahan yang belum tercatat),
// buktinya dicari di folder tujuan.
func ownsProjectRoot(p *models.Project, root, projectTargetDir string) bool {
	probe := root
	if !isDirPath(probe) && isDirPath(projectTargetDir) {
		probe = projectTargetDir
	}
	owner := identifierProjectID(probe)
	if owner != 0 && owner != p.ID {
		return false
	}
	if p.IsUnified {
		return true
	}
	if _, err := os.Stat(filepath.Join(probe, ".git")); err == nil {
		return true
	}
	// Penanda saja belum cukup untuk folder tepat di bawah home (mis. ~/Documents): proyek bundel ber-root folder
	// umum juga mendapat penanda di sana.
	return owner == p.ID && filepath.Dir(root) != filepath.Clean(models.UserHome())
}

// rootHoldsOtherProjects melaporkan apakah proyek lain terdaftar di dalam root (root = folder induk bersama).
func (s *organizerService) rootHoldsOtherProjects(ctx context.Context, projectID uint, root string) bool {
	others, err := s.projectRepo.FindAll(ctx, "", "")
	if err != nil {
		return true
	}
	inside := func(path string) bool {
		path = strings.TrimSpace(path)
		if path == "" || path == "-" {
			return false
		}
		path = filepath.Clean(path)
		return path == root || pathWithin(root, path)
	}
	for _, o := range others {
		if o.ID == projectID {
			continue
		}
		if inside(o.RootPath) {
			return true
		}
		for _, c := range o.Components {
			if inside(c.AbsolutePath) {
				return true
			}
		}
	}
	return false
}

// previewWholeRootMove menyusun satu item yang memindahkan seluruh folder root proyek ke folder tujuan.
func previewWholeRootMove(preview *models.ConsolidatePreview, p *models.Project, slug, root string) int64 {
	item := models.FolderMoveItem{
		ID:                 "item-1",
		ProjectID:          p.ID,
		ProjectName:        p.Name,
		CurrentPath:        root,
		TargetPath:         preview.ProjectTargetDir,
		RelativeTargetPath: slug,
		Status:             "pending",
	}
	if assessConsolidateMove(&item, p.ID) {
		preview.HasConflict = true
		preview.CanConsolidate = false
	}
	if item.Status == "pending" {
		note := "Seluruh folder proyek dipindahkan utuh, termasuk berkas di luar folder komponen; susunan di dalamnya tidak berubah."
		item.Warning = strings.TrimSpace(note + " " + item.Warning)
	}
	preview.WholeRoot = true
	preview.Items = append(preview.Items, item)
	return item.SizeBytes
}

// saveWholeRootComponentPaths menyimpan path komponen setelah root pindah utuh: posisi relatif terhadap root tetap,
// hanya awalannya yang berganti. Mengembalikan folder komponen (lama -> baru).
func (s *organizerService) saveWholeRootComponentPaths(ctx context.Context, p *models.Project, oldRoot, newRoot string) ([]pathMove, error) {
	var dirs []pathMove
	for _, c := range p.Components {
		src := strings.TrimSpace(c.AbsolutePath)
		if src == "" || src == "-" {
			continue
		}
		src = filepath.Clean(src)
		rel, err := filepath.Rel(oldRoot, src)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		dst := filepath.Join(newRoot, rel)
		if err := s.componentRepo.UpdateFields(ctx, c.ID, map[string]interface{}{
			"absolute_path": dst,
			"relative_path": filepath.ToSlash(rel),
		}); err != nil {
			return nil, fmt.Errorf("folder proyek sudah berada di %s tetapi path komponen #%d gagal disimpan: %w", newRoot, c.ID, err)
		}
		dirs = append(dirs, pathMove{from: src, to: dst})
	}
	return dirs, nil
}

// rebaseCodeMapPaths mengganti awalan path lama di peta kode (root peta dan path absolut tiap berkas) agar peta
// tetap menunjuk berkas yang benar tanpa menunggu pemindaian ulang.
func (s *organizerService) rebaseCodeMapPaths(ctx context.Context, projectID uint, moves []pathMove) error {
	if s.db == nil || len(moves) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maps []models.CodeMap
		if err := tx.Where("project_id = ?", projectID).Find(&maps).Error; err != nil {
			return err
		}
		for _, cm := range maps {
			if cm.RootPath == "" {
				continue
			}
			if moved, ok := rebasePath(filepath.Clean(cm.RootPath), moves); ok {
				if err := tx.Model(&models.CodeMap{}).Where("id = ?", cm.ID).UpdateColumn("root_path", moved).Error; err != nil {
					return err
				}
			}
		}
		var entries []models.CodeMapEntry
		if err := tx.Select("id", "abs_path").Where("project_id = ?", projectID).Find(&entries).Error; err != nil {
			return err
		}
		for _, e := range entries {
			if e.AbsPath == "" {
				continue
			}
			moved, ok := rebasePath(filepath.Clean(e.AbsPath), moves)
			if !ok {
				continue
			}
			if err := tx.Model(&models.CodeMapEntry{}).Where("id = ?", e.ID).UpdateColumn("abs_path", moved).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// repairGitWorktrees menjalankan `git worktree repair` di repo git yang pindah dan punya worktree tertaut.
// Worktree tertaut menyimpan path absolut repo utamanya, sehingga tanpa perbaikan ia tidak lagi mengenali repo
// itu. Kegagalan hanya menjadi peringatan: folder sudah pindah dan perbaikan bisa diulang dengan tangan.
func repairGitWorktrees(repos, moves []pathMove) []string {
	var warnings []string
	seen := map[string]bool{}
	for _, repo := range repos {
		if seen[repo.to] {
			continue
		}
		seen[repo.to] = true
		adminDir := filepath.Join(repo.to, ".git", "worktrees")
		entries, err := os.ReadDir(adminDir)
		if err != nil || len(entries) == 0 {
			continue
		}
		args := []string{"-C", repo.to, "worktree", "repair"}
		// Worktree yang ikut pindah bersama repo utamanya harus disebut dengan path barunya.
		for _, e := range entries {
			data, err := os.ReadFile(filepath.Join(adminDir, e.Name(), "gitdir"))
			if err != nil {
				continue
			}
			linked := filepath.Dir(filepath.Clean(strings.TrimSpace(string(data))))
			if moved, ok := rebasePath(linked, moves); ok && isDirPath(moved) {
				args = append(args, moved)
			}
		}
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			warnings = append(warnings, fmt.Sprintf("git worktree repair di %s gagal: %v %s", repo.to, err, strings.TrimSpace(string(out))))
		}
	}
	return warnings
}
