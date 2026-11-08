package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
	"github.com/vibeasik/vibeasik/pkg/workspace"
)

// OrganizerService plans and executes folder reorganization for cleaner project management
type OrganizerService interface {
	GenerateReorgPlan(ctx context.Context, req models.ReorgPlanRequest) (*models.ReorgPlan, error)
	ExecuteMoves(ctx context.Context, payload models.ExecuteMovePayload) (*models.ExecuteMoveResult, error)
	PreviewConsolidate(ctx context.Context, projectID uint, targetParentDir string) (*models.ConsolidatePreview, error)
	Consolidate(ctx context.Context, projectID uint, targetParentDir string, apiBase string) (*models.ConsolidateResult, error)
	GetWorkspaceSettings(ctx context.Context) (*models.WorkspaceSettings, error)
	UpdateWorkspaceSettings(ctx context.Context, defaultProjectsRoot string) (*models.WorkspaceSettings, error)
}

type organizerService struct {
	projectRepo   repository.ProjectRepository
	componentRepo repository.ComponentRepository
	db            *gorm.DB
}

// NewOrganizerService instantiates a new OrganizerService
func NewOrganizerService(projectRepo repository.ProjectRepository, componentRepo repository.ComponentRepository, db *gorm.DB) OrganizerService {
	return &organizerService{
		projectRepo:   projectRepo,
		componentRepo: componentRepo,
		db:            db,
	}
}

func getFolderSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func checkIsGitRepo(path string) (bool, bool) {
	gitDir := filepath.Join(path, ".git")
	if fi, err := os.Stat(gitDir); err == nil && (fi.IsDir() || !fi.IsDir()) {
		// Check uncommitted changes
		cmd := exec.Command("git", "-C", path, "status", "--porcelain")
		out, err := cmd.Output()
		hasUncommitted := err == nil && len(strings.TrimSpace(string(out))) > 0
		return true, hasUncommitted
	}
	return false, false
}

func (s *organizerService) GenerateReorgPlan(ctx context.Context, req models.ReorgPlanRequest) (*models.ReorgPlan, error) {
	targetDir := strings.TrimSpace(req.TargetDirectory)
	if targetDir == "" {
		targetDir = models.DefaultProjectsRoot()
	}
	targetDir = filepath.Clean(targetDir)

	projects, err := s.projectRepo.FindAll(ctx, "", "")
	if err != nil {
		return nil, err
	}

	filterMap := make(map[uint]bool)
	if len(req.ProjectIDs) > 0 {
		for _, id := range req.ProjectIDs {
			filterMap[id] = true
		}
	}

	plan := &models.ReorgPlan{
		TargetDirectory: targetDir,
		Strategy:        req.Strategy,
		PlannedMoves:    make([]models.FolderMoveItem, 0),
		CreatedAt:       time.Now(),
	}
	if plan.Strategy == "" {
		plan.Strategy = models.StrategyProductCentric
	}

	moveCounter := 1
	var totalSize int64

	for _, p := range projects {
		if len(filterMap) > 0 && !filterMap[p.ID] {
			continue
		}

		cleanBaseName := normalizeProjectName(p.Name)
		if cleanBaseName == "" {
			cleanBaseName = Slugify(p.Name)
		}

		// If project has components, plan move per component
		if len(p.Components) > 0 {
			for _, c := range p.Components {
				srcPath := strings.TrimSpace(c.AbsolutePath)
				if srcPath == "" || srcPath == "-" {
					continue
				}
				srcPath = filepath.Clean(srcPath)

				// Determine friendly subfolder name based on role
				roleSub := "backend"
				switch c.Role {
				case models.RoleBackend:
					roleSub = "backend"
				case models.RoleFrontendAdmin:
					roleSub = "frontend-admin"
				case models.RoleFrontendClient:
					roleSub = "frontend"
				case models.RoleFrontendMobile:
					roleSub = "mobile"
				case models.RoleCLI:
					roleSub = "cli"
				case models.RoleWorker:
					roleSub = "worker"
				case models.RoleDocs:
					roleSub = "docs"
				default:
					if strings.Contains(c.Role, "front") {
						roleSub = "frontend"
					} else if strings.Contains(c.Role, "mobile") {
						roleSub = "mobile"
					} else {
						roleSub = Slugify(c.Name)
					}
				}

				destPath := filepath.Join(targetDir, cleanBaseName, roleSub)
				relTarget := filepath.Join(cleanBaseName, roleSub)

				// If source is already at destination, skip
				if srcPath == destPath {
					continue
				}

				item := models.FolderMoveItem{
					ID:                 fmt.Sprintf("move-%d", moveCounter),
					ProjectID:          p.ID,
					ProjectName:        p.Name,
					ComponentID:        c.ID,
					ComponentName:      c.Name,
					Role:               c.Role,
					CurrentPath:        srcPath,
					TargetPath:         destPath,
					RelativeTargetPath: relTarget,
					Status:             "pending",
				}
				moveCounter++

				if fi, err := os.Stat(srcPath); err == nil && fi.IsDir() {
					item.SizeBytes = getFolderSize(srcPath)
					totalSize += item.SizeBytes
					item.IsGitRepo, item.HasUncommittedChanges = checkIsGitRepo(srcPath)
					if item.HasUncommittedChanges {
						item.Warning = "Ada uncommitted git changes pada folder ini. Disarankan git commit terlebih dahulu."
					}
				} else {
					item.Status = "conflict"
					item.Error = "Folder sumber tidak ditemukan di komputer: " + srcPath
				}

				if _, err := os.Stat(destPath); err == nil {
					item.Status = "conflict"
					item.Error = "Target folder sudah ada di komputer: " + destPath
				}

				plan.PlannedMoves = append(plan.PlannedMoves, item)
			}
		} else {
			// Standalone single project folder
			srcPath := strings.TrimSpace(p.RootPath)
			if srcPath == "" || srcPath == "-" {
				continue
			}
			srcPath = filepath.Clean(srcPath)
			destPath := filepath.Join(targetDir, cleanBaseName)

			if srcPath == destPath {
				continue
			}

			item := models.FolderMoveItem{
				ID:                 fmt.Sprintf("move-%d", moveCounter),
				ProjectID:          p.ID,
				ProjectName:        p.Name,
				CurrentPath:        srcPath,
				TargetPath:         destPath,
				RelativeTargetPath: cleanBaseName,
				Status:             "pending",
			}
			moveCounter++

			if fi, err := os.Stat(srcPath); err == nil && fi.IsDir() {
				item.SizeBytes = getFolderSize(srcPath)
				totalSize += item.SizeBytes
				item.IsGitRepo, item.HasUncommittedChanges = checkIsGitRepo(srcPath)
				if item.HasUncommittedChanges {
					item.Warning = "Ada uncommitted git changes pada folder ini."
				}
			} else {
				item.Status = "conflict"
				item.Error = "Folder sumber tidak ditemukan di komputer: " + srcPath
			}

			if _, err := os.Stat(destPath); err == nil {
				item.Status = "conflict"
				item.Error = "Target folder sudah ada di komputer: " + destPath
			}

			plan.PlannedMoves = append(plan.PlannedMoves, item)
		}
	}

	plan.TotalItems = len(plan.PlannedMoves)
	plan.TotalSize = totalSize

	// Generate executable bash script as a safe, transparent alternative
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("# vibeasik Workspace Reorganization Script\n")
	sb.WriteString("# Dibuat otomatis pada: " + time.Now().Format(time.RFC1123) + "\n")
	sb.WriteString("# Target Folder: " + targetDir + "\n\n")
	sb.WriteString("set -e\n\n")
	sb.WriteString("echo '=== Memulai Reorganisasi Folder Proyek vibeasik ==='\n")
	sb.WriteString("mkdir -p \"" + targetDir + "\"\n\n")

	for _, m := range plan.PlannedMoves {
		if m.Status == "conflict" {
			sb.WriteString("# [DILEWATI KARENA KONFLIK] " + m.CurrentPath + " -> " + m.TargetPath + "\n")
			sb.WriteString("# Alasan: " + m.Error + "\n\n")
			continue
		}
		sb.WriteString("# Pindahkan: " + m.ProjectName + " (" + m.ComponentName + ")\n")
		sb.WriteString("if [ -d \"" + m.CurrentPath + "\" ]; then\n")
		sb.WriteString("  if [ ! -d \"" + m.TargetPath + "\" ]; then\n")
		sb.WriteString("    mkdir -p \"" + filepath.Dir(m.TargetPath) + "\"\n")
		sb.WriteString("    mv \"" + m.CurrentPath + "\" \"" + m.TargetPath + "\"\n")
		sb.WriteString("    echo ' [OK] Berhasil memindahkan: " + m.RelativeTargetPath + "'\n")
		sb.WriteString("  else\n")
		sb.WriteString("    echo ' [SKIP] Target sudah ada: " + m.TargetPath + "'\n")
		sb.WriteString("  fi\n")
		sb.WriteString("else\n")
		sb.WriteString("  echo ' [SKIP] Sumber tidak ditemukan: " + m.CurrentPath + "'\n")
		sb.WriteString("fi\n\n")
	}

	sb.WriteString("echo '=== Reorganisasi Selesai! Silakan sinkronkan path di vibeasik. ==='\n")
	plan.GeneratedScript = sb.String()

	return plan, nil
}

func moveOrCopy(src, dst string) error {
	// Try standard atomic os.Rename first
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Fallback to copy and remove across device boundaries
	cmd := exec.Command("cp", "-a", src, dst)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gagal menyalin folder: %w", err)
	}
	return os.RemoveAll(src)
}

func (s *organizerService) ExecuteMoves(ctx context.Context, payload models.ExecuteMovePayload) (*models.ExecuteMoveResult, error) {
	result := &models.ExecuteMoveResult{
		Moves: make([]models.FolderMoveItem, 0, len(payload.Moves)),
	}

	for _, m := range payload.Moves {
		if m.Status == "skipped" {
			result.SkippedCount++
			result.Moves = append(result.Moves, m)
			continue
		}

		src := filepath.Clean(m.CurrentPath)
		dst := filepath.Clean(m.TargetPath)

		// Verification
		fi, err := os.Stat(src)
		if err != nil || !fi.IsDir() {
			m.Status = "failed"
			m.Error = "Folder sumber tidak ada di disk: " + src
			result.FailedCount++
			result.Moves = append(result.Moves, m)
			continue
		}

		if _, err := os.Stat(dst); err == nil {
			m.Status = "conflict"
			m.Error = "Folder tujuan sudah ada di disk: " + dst
			result.FailedCount++
			result.Moves = append(result.Moves, m)
			continue
		}

		// Ensure parent exists
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			m.Status = "failed"
			m.Error = "Gagal membuat direktori induk tujuan: " + err.Error()
			result.FailedCount++
			result.Moves = append(result.Moves, m)
			continue
		}

		// Perform move
		if err := moveOrCopy(src, dst); err != nil {
			m.Status = "failed"
			m.Error = "Gagal memindahkan folder: " + err.Error()
			result.FailedCount++
			result.Moves = append(result.Moves, m)
			continue
		}

		m.Status = "success"
		m.Error = ""
		result.SuccessCount++

		// Update Database if requested
		if payload.UpdateDB {
			if m.ComponentID > 0 {
				comp, err := s.componentRepo.FindByID(ctx, m.ComponentID)
				if err == nil && comp != nil {
					comp.AbsolutePath = dst
					comp.RelativePath = filepath.Base(dst)
					_ = s.componentRepo.Update(ctx, comp)
				}
			}
			if m.ProjectID > 0 {
				p, err := s.projectRepo.FindByID(ctx, m.ProjectID)
				if err == nil && p != nil {
					// If project has no components or this was the root
					if len(p.Components) <= 1 || p.RootPath == src {
						p.RootPath = filepath.Dir(dst)
						_ = s.projectRepo.Update(ctx, p)
					}
				}
			}
		}

		result.Moves = append(result.Moves, m)
	}

	result.Message = fmt.Sprintf("Berhasil memindahkan %d folder, %d gagal, %d dilewati.", result.SuccessCount, result.FailedCount, result.SkippedCount)
	return result, nil
}

func getComponentSubfolder(c models.Component) string {
	switch c.Role {
	case models.RoleBackend:
		return "backend"
	case models.RoleFrontendClient:
		return "frontend"
	case models.RoleFrontendAdmin:
		return "frontend-admin"
	case models.RoleFrontendMobile:
		return "mobile"
	case models.RoleDocs:
		return "docs"
	case models.RoleCLI:
		return "cli"
	case models.RoleWorker:
		return "worker"
	default:
		if strings.Contains(c.Role, "front") {
			return "frontend"
		} else if strings.Contains(c.Role, "mobile") {
			return "mobile"
		}
		clean := Slugify(c.Name)
		if clean == "" {
			clean = fmt.Sprintf("component-%d", c.ID)
		}
		return clean
	}
}

func (s *organizerService) GetWorkspaceSettings(ctx context.Context) (*models.WorkspaceSettings, error) {
	var ws models.WorkspaceSettings
	if s.db != nil {
		if err := s.db.WithContext(ctx).First(&ws).Error; err == nil {
			return &ws, nil
		}
	}
	defaultRoot := models.DefaultProjectsRoot()
	ws = models.WorkspaceSettings{
		DefaultProjectsRoot: defaultRoot,
		UpdatedAt:           time.Now(),
	}
	if s.db != nil {
		_ = s.db.WithContext(ctx).Create(&ws).Error
	}
	return &ws, nil
}

func (s *organizerService) UpdateWorkspaceSettings(ctx context.Context, defaultProjectsRoot string) (*models.WorkspaceSettings, error) {
	defaultProjectsRoot = strings.TrimSpace(defaultProjectsRoot)
	if defaultProjectsRoot == "" {
		defaultProjectsRoot = models.DefaultProjectsRoot()
	}
	defaultProjectsRoot = filepath.Clean(defaultProjectsRoot)

	var ws models.WorkspaceSettings
	if s.db != nil {
		if err := s.db.WithContext(ctx).First(&ws).Error; err == nil {
			ws.DefaultProjectsRoot = defaultProjectsRoot
			ws.UpdatedAt = time.Now()
			if err := s.db.WithContext(ctx).Save(&ws).Error; err != nil {
				return nil, err
			}
			return &ws, nil
		}
	}
	ws = models.WorkspaceSettings{
		DefaultProjectsRoot: defaultProjectsRoot,
		UpdatedAt:           time.Now(),
	}
	if s.db != nil {
		if err := s.db.WithContext(ctx).Create(&ws).Error; err != nil {
			return nil, err
		}
	}
	return &ws, nil
}

// consolidateSource adalah satu folder sumber beserta semua komponen yang menunjuk ke sana (komponen ganda hasil
// scan ulang sering berbagi folder yang sama) — folder itu hanya boleh dipindahkan sekali.
type consolidateSource struct {
	path       string
	components []models.Component
}

func groupConsolidateSources(comps []models.Component) []consolidateSource {
	var out []consolidateSource
	index := map[string]int{}
	for _, c := range comps {
		src := strings.TrimSpace(c.AbsolutePath)
		if src == "" || src == "-" {
			continue
		}
		src = filepath.Clean(src)
		if i, ok := index[src]; ok {
			out[i].components = append(out[i].components, c)
			continue
		}
		index[src] = len(out)
		out = append(out, consolidateSource{path: src, components: []models.Component{c}})
	}
	return out
}

func pathWithin(parent, child string) bool {
	r, err := filepath.Rel(parent, child)
	return err == nil && r != "." && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}

func isDirPath(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// identifierProjectID membaca project_id dari .vibeasik.json di dir (0 bila tidak ada/tidak terbaca).
func identifierProjectID(dir string) uint {
	data, err := os.ReadFile(filepath.Join(dir, workspace.IdentifierFileName))
	if err != nil {
		return 0
	}
	var idf workspace.ProjectIdentifier
	if json.Unmarshal(data, &idf) != nil {
		return 0
	}
	return idf.ProjectID
}

// assessConsolidateMove mengisi status item dari kondisi disk. Mengembalikan true bila item konflik.
func assessConsolidateMove(item *models.FolderMoveItem, projectID uint) bool {
	src, dst := item.CurrentPath, item.TargetPath
	if src == dst {
		item.Status = "already_in_place"
		item.SizeBytes = getFolderSize(src)
		return false
	}
	srcOK := isDirPath(src)
	_, dstErr := os.Stat(dst)
	dstExists := dstErr == nil
	switch {
	case !srcOK && dstExists && isDirPath(dst):
		// Sisa penyatuan yang terhenti di tengah jalan: folder sudah di target tetapi database masih menyimpan path lama.
		if id := identifierProjectID(dst); id != 0 && id != projectID {
			item.Status = "conflict"
			item.Error = fmt.Sprintf("Target folder sudah ada dan milik proyek lain (ID %d): %s", id, dst)
			return true
		}
		item.Status = "already_in_place"
		item.Warning = "Folder sudah berada di target (penyatuan sebelumnya belum tercatat); path di database akan diperbarui."
		item.SizeBytes = getFolderSize(dst)
		item.IsGitRepo, item.HasUncommittedChanges = checkIsGitRepo(dst)
		return false
	case !srcOK:
		item.Status = "conflict"
		item.Error = "Folder sumber tidak ditemukan di komputer: " + src
		return true
	case dstExists:
		item.Status = "conflict"
		item.Error = "Target folder sudah ada di komputer: " + dst
		return true
	}
	item.SizeBytes = getFolderSize(src)
	item.IsGitRepo, item.HasUncommittedChanges = checkIsGitRepo(src)
	if item.HasUncommittedChanges {
		item.Warning = "Terdapat perubahan git yang belum di-commit."
	}
	return false
}

func (s *organizerService) PreviewConsolidate(ctx context.Context, projectID uint, targetParentDir string) (*models.ConsolidatePreview, error) {
	p, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil || p == nil {
		return nil, fmt.Errorf("project not found (ID: %d)", projectID)
	}

	targetParentDir = strings.TrimSpace(targetParentDir)
	if targetParentDir == "" {
		ws, _ := s.GetWorkspaceSettings(ctx)
		if ws != nil && ws.DefaultProjectsRoot != "" {
			targetParentDir = ws.DefaultProjectsRoot
		} else {
			targetParentDir = models.DefaultProjectsRoot()
		}
	}
	targetParentDir = filepath.Clean(targetParentDir)

	slug := p.Slug
	if slug == "" {
		slug = Slugify(p.Name)
	}
	if slug == "" {
		slug = fmt.Sprintf("project-%d", p.ID)
	}

	projectTargetDir := filepath.Join(targetParentDir, slug)

	preview := &models.ConsolidatePreview{
		ProjectID:        p.ID,
		ProjectName:      p.Name,
		ProjectSlug:      slug,
		IsUnified:        p.IsUnified,
		TargetParentDir:  targetParentDir,
		ProjectTargetDir: projectTargetDir,
		Items:            make([]models.FolderMoveItem, 0),
		CanConsolidate:   true,
	}

	usedSubfolders := make(map[string]int)
	var totalSize int64

	if len(p.Components) > 0 {
		sources := groupConsolidateSources(p.Components)
		// targetOf menyimpan tujuan setiap folder sumber agar komponen bersarang (folder di dalam komponen lain)
		// ikut pindah bersama induknya alih-alih dipindahkan dua kali.
		targetOf := map[string]string{}
		for i, src := range sources {
			c := src.components[0]
			item := models.FolderMoveItem{
				ID:            fmt.Sprintf("item-%d", i+1),
				ProjectID:     p.ID,
				ProjectName:   p.Name,
				ComponentID:   c.ID,
				ComponentName: c.Name,
				Role:          c.Role,
				CurrentPath:   src.path,
				Status:        "pending",
			}
			for _, dup := range src.components {
				item.ComponentIDs = append(item.ComponentIDs, dup.ID)
			}
			if len(src.components) > 1 {
				names := make([]string, 0, len(src.components)-1)
				for _, dup := range src.components[1:] {
					names = append(names, dup.Name)
				}
				item.Warning = fmt.Sprintf("Folder ini juga dipakai komponen %s; dipindahkan sekali untuk semuanya.", strings.Join(names, ", "))
			}

			parent := ""
			for _, other := range sources {
				if pathWithin(other.path, src.path) && (parent == "" || len(other.path) > len(parent)) {
					parent = other.path
				}
			}
			if parent != "" {
				// Diselesaikan setelah semua induk punya tujuan (lihat di bawah).
				item.Status = "moves_with_parent"
				item.Warning = "Berada di dalam folder komponen lain; ikut pindah bersama induknya."
				item.Error = parent
				preview.Items = append(preview.Items, item)
				continue
			}

			subfolder := getComponentSubfolder(c)
			usedSubfolders[subfolder]++
			if usedSubfolders[subfolder] > 1 {
				subfolder = fmt.Sprintf("%s-%s", subfolder, Slugify(c.Name))
			}
			item.TargetPath = filepath.Join(projectTargetDir, subfolder)
			item.RelativeTargetPath = filepath.Join(slug, subfolder)
			targetOf[src.path] = item.TargetPath

			warning := item.Warning
			if assessConsolidateMove(&item, p.ID) {
				preview.HasConflict = true
				preview.CanConsolidate = false
			}
			if warning != "" && item.Warning != warning {
				item.Warning = strings.TrimSpace(warning + " " + item.Warning)
			}
			totalSize += item.SizeBytes
			preview.Items = append(preview.Items, item)
		}
		for i := range preview.Items {
			item := &preview.Items[i]
			if item.Status != "moves_with_parent" {
				continue
			}
			parent := item.Error
			item.Error = ""
			rel, _ := filepath.Rel(parent, item.CurrentPath)
			item.TargetPath = filepath.Join(targetOf[parent], rel)
			item.RelativeTargetPath, _ = filepath.Rel(targetParentDir, item.TargetPath)
		}
	} else {
		// Standalone project without components
		srcPath := strings.TrimSpace(p.RootPath)
		if srcPath != "" && srcPath != "-" {
			item := models.FolderMoveItem{
				ID:                 "item-1",
				ProjectID:          p.ID,
				ProjectName:        p.Name,
				CurrentPath:        filepath.Clean(srcPath),
				TargetPath:         projectTargetDir,
				RelativeTargetPath: slug,
				Status:             "pending",
			}
			if assessConsolidateMove(&item, p.ID) {
				preview.HasConflict = true
				preview.CanConsolidate = false
			}
			totalSize += item.SizeBytes
			preview.Items = append(preview.Items, item)
		}
	}

	preview.TotalSizeBytes = totalSize
	return preview, nil
}

// relocateVibeasikDirs memindahkan isi folder .vibeasik/ (peta kode, base_knowledge, cache) dari folder komponen ke
// root proyek tersatukan, sehingga artefak vibeasik hanya ada satu di folder induk. Berkas yang sudah ada di root
// tidak ditimpa; CODEMAP.md/codemap.jsonl lama dibuang karena dibuat ulang oleh Init vibeasik.
func relocateVibeasikDirs(root string, dirs []string) []string {
	var warnings []string
	dstDir := filepath.Join(root, ".vibeasik")
	seen := map[string]bool{}
	for _, d := range dirs {
		d = filepath.Clean(d)
		if d == root || seen[d] {
			continue
		}
		seen[d] = true
		srcDir := filepath.Join(d, ".vibeasik")
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			continue
		}
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return append(warnings, fmt.Sprintf("gagal membuat %s: %v", dstDir, err))
		}
		for _, e := range entries {
			from := filepath.Join(srcDir, e.Name())
			switch e.Name() {
			case "CODEMAP.md", "codemap.jsonl":
				_ = os.Remove(from)
				continue
			}
			to := filepath.Join(dstDir, e.Name())
			if _, err := os.Stat(to); err == nil {
				continue
			}
			if err := os.Rename(from, to); err != nil {
				warnings = append(warnings, fmt.Sprintf("gagal memindahkan %s ke root: %v", from, err))
			}
		}
		// Hanya dihapus bila sudah kosong; sisa berkas (duplikat yang sudah ada di root) dibiarkan.
		if err := os.Remove(srcDir); err != nil && !os.IsNotExist(err) {
			warnings = append(warnings, fmt.Sprintf("%s tidak kosong, dibiarkan (berkas yang sama sudah ada di root)", srcDir))
		}
	}
	return warnings
}

func (s *organizerService) Consolidate(ctx context.Context, projectID uint, targetParentDir string, apiBase string) (*models.ConsolidateResult, error) {
	preview, err := s.PreviewConsolidate(ctx, projectID, targetParentDir)
	if err != nil {
		return nil, err
	}

	if preview.HasConflict {
		return nil, fmt.Errorf("tidak dapat menyatukan proyek karena terdapat konflik target atau folder sumber tidak ditemukan")
	}

	// Make sure target project directory exists
	if err := os.MkdirAll(preview.ProjectTargetDir, 0755); err != nil {
		return nil, fmt.Errorf("gagal membuat direktori proyek induk (%s): %w", preview.ProjectTargetDir, err)
	}

	// Path komponen disimpan segera setelah foldernya dipindah, agar database tidak tertinggal dari isi disk bila
	// pemindahan berikutnya gagal (penyatuan ulang lalu mengenali folder yang sudah di target).
	savePaths := func(item models.FolderMoveItem) error {
		for _, cid := range item.ComponentIDs {
			if err := s.componentRepo.UpdateFields(ctx, cid, map[string]interface{}{
				"absolute_path": item.TargetPath,
				"relative_path": filepath.ToSlash(item.RelativeTargetPath),
			}); err != nil {
				return fmt.Errorf("folder %s sudah dipindah ke %s tetapi path komponen #%d gagal disimpan: %w", item.CurrentPath, item.TargetPath, cid, err)
			}
		}
		return nil
	}

	var movedItems []models.FolderMoveItem
	movedCount := 0

	for _, item := range preview.Items {
		if item.Status == "pending" {
			if err := os.MkdirAll(filepath.Dir(item.TargetPath), 0755); err != nil {
				return nil, fmt.Errorf("gagal membuat folder tujuan %s: %w", item.TargetPath, err)
			}
			if err := moveOrCopy(item.CurrentPath, item.TargetPath); err != nil {
				return nil, fmt.Errorf("gagal memindahkan %s ke %s: %w", item.CurrentPath, item.TargetPath, err)
			}
			item.Status = "success"
			movedCount++
		}
		movedItems = append(movedItems, item)
	}
	// Komponen bersarang baru ada di tujuannya setelah induknya pindah, jadi path disimpan setelah semua pemindahan.
	for _, item := range movedItems {
		if err := savePaths(item); err != nil {
			return nil, err
		}
	}

	if err := s.projectRepo.UpdateFields(ctx, projectID, map[string]interface{}{
		"root_path":  preview.ProjectTargetDir,
		"is_unified": true,
	}); err != nil {
		return nil, fmt.Errorf("folder sudah disatukan di %s tetapi root proyek gagal disimpan: %w", preview.ProjectTargetDir, err)
	}

	// Artefak vibeasik (peta kode, base knowledge, tautan AI) dipusatkan di root folder induk.
	var compDirs []string
	for _, item := range movedItems {
		compDirs = append(compDirs, item.TargetPath)
	}
	warnings := relocateVibeasikDirs(preview.ProjectTargetDir, compDirs)

	var linkedFiles []string
	codeMapMarkdown := ""
	if s.db != nil {
		if init, err := workspace.InitProject(s.db, projectID, apiBase); err != nil {
			warnings = append(warnings, "Init vibeasik di root gagal: "+err.Error())
			if created, _, linkErr := workspace.LinkProjectWorkspaces(s.db, projectID, 0, apiBase); linkErr == nil {
				linkedFiles = created
			}
		} else {
			linkedFiles = init.FilesCreated
			codeMapMarkdown = init.CodeMapMarkdown
		}
	}

	result := &models.ConsolidateResult{
		Success:         true,
		Message:         fmt.Sprintf("Proyek '%s' berhasil disatukan ke dalam %s; peta kode & tautan AI kini berada di root folder induk.", preview.ProjectName, preview.ProjectTargetDir),
		ProjectID:       preview.ProjectID,
		ProjectName:     preview.ProjectName,
		ProjectRoot:     preview.ProjectTargetDir,
		IsUnified:       true,
		MovedCount:      movedCount,
		Items:           movedItems,
		LinkedFiles:     linkedFiles,
		CodeMapMarkdown: codeMapMarkdown,
		Warnings:        warnings,
	}

	return result, nil
}
