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

func (s *organizerService) loadProjectsForPlan(ctx context.Context, projectIDs []uint) ([]models.Project, error) {
	if len(projectIDs) > 0 {
		return s.projectRepo.FindByIDs(ctx, projectIDs)
	}
	return s.projectRepo.FindAll(ctx, "", "")
}

func (s *organizerService) GenerateReorgPlan(ctx context.Context, req models.ReorgPlanRequest) (*models.ReorgPlan, error) {
	targetDir := strings.TrimSpace(req.TargetDirectory)
	if targetDir == "" {
		targetDir = models.DefaultProjectsRoot()
	}
	targetDir = filepath.Clean(targetDir)

	projects, err := s.loadProjectsForPlan(ctx, req.ProjectIDs)
	if err != nil {
		return nil, err
	}

	filterMap := make(map[uint]bool, len(req.ProjectIDs))
	for _, id := range req.ProjectIDs {
		filterMap[id] = true
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

	planner := &reorgPlanner{plan: plan, targetDir: targetDir, moveCounter: 1}
	for _, p := range projects {
		if len(filterMap) > 0 && !filterMap[p.ID] {
			continue
		}
		planner.planProject(p)
	}

	plan.TotalItems = len(plan.PlannedMoves)
	plan.TotalSize = planner.totalSize
	plan.GeneratedScript = buildReorgScript(plan.PlannedMoves, targetDir)

	return plan, nil
}

// reorgPlanner mengakumulasi langkah pemindahan ke dalam satu ReorgPlan.
type reorgPlanner struct {
	plan        *models.ReorgPlan
	targetDir   string
	moveCounter int
	totalSize   int64
}

func (pl *reorgPlanner) nextID() string {
	id := fmt.Sprintf("move-%d", pl.moveCounter)
	pl.moveCounter++
	return id
}

func (pl *reorgPlanner) planProject(p models.Project) {
	cleanBaseName := normalizeProjectName(p.Name)
	if cleanBaseName == "" {
		cleanBaseName = Slugify(p.Name)
	}
	if len(p.Components) > 0 {
		for _, c := range p.Components {
			pl.planComponent(p, c, cleanBaseName)
		}
		return
	}
	pl.planStandalone(p, cleanBaseName)
}

func (pl *reorgPlanner) planComponent(p models.Project, c models.Component, cleanBaseName string) {
	srcPath := strings.TrimSpace(c.AbsolutePath)
	if srcPath == "" || srcPath == "-" {
		return
	}
	srcPath = filepath.Clean(srcPath)

	roleSub := reorgRoleSubfolder(c)
	destPath := filepath.Join(pl.targetDir, cleanBaseName, roleSub)
	if srcPath == destPath {
		return
	}

	item := models.FolderMoveItem{
		ID:                 pl.nextID(),
		ProjectID:          p.ID,
		ProjectName:        p.Name,
		ComponentID:        c.ID,
		ComponentName:      c.Name,
		Role:               c.Role,
		CurrentPath:        srcPath,
		TargetPath:         destPath,
		RelativeTargetPath: filepath.Join(cleanBaseName, roleSub),
		Status:             "pending",
	}
	pl.assessAndAppend(item, "Ada uncommitted git changes pada folder ini. Disarankan git commit terlebih dahulu.")
}

func (pl *reorgPlanner) planStandalone(p models.Project, cleanBaseName string) {
	srcPath := strings.TrimSpace(p.RootPath)
	if srcPath == "" || srcPath == "-" {
		return
	}
	srcPath = filepath.Clean(srcPath)
	destPath := filepath.Join(pl.targetDir, cleanBaseName)
	if srcPath == destPath {
		return
	}

	item := models.FolderMoveItem{
		ID:                 pl.nextID(),
		ProjectID:          p.ID,
		ProjectName:        p.Name,
		CurrentPath:        srcPath,
		TargetPath:         destPath,
		RelativeTargetPath: cleanBaseName,
		Status:             "pending",
	}
	pl.assessAndAppend(item, "Ada uncommitted git changes pada folder ini.")
}

// assessAndAppend memeriksa kondisi disk sumber/tujuan, lalu menambahkan item ke rencana.
func (pl *reorgPlanner) assessAndAppend(item models.FolderMoveItem, dirtyWarning string) {
	if isDirPath(item.CurrentPath) {
		item.SizeBytes = getFolderSize(item.CurrentPath)
		pl.totalSize += item.SizeBytes
		item.IsGitRepo, item.HasUncommittedChanges = checkIsGitRepo(item.CurrentPath)
		if item.HasUncommittedChanges {
			item.Warning = dirtyWarning
		}
	} else {
		item.Status = "conflict"
		item.Error = "Folder sumber tidak ditemukan di komputer: " + item.CurrentPath
	}
	if _, err := os.Stat(item.TargetPath); err == nil {
		item.Status = "conflict"
		item.Error = "Target folder sudah ada di komputer: " + item.TargetPath
	}
	pl.plan.PlannedMoves = append(pl.plan.PlannedMoves, item)
}

// roleSubfolders memetakan peran komponen ke nama subfolder target.
var roleSubfolders = map[string]string{
	models.RoleBackend:        "backend",
	models.RoleFrontendClient: "frontend",
	models.RoleFrontendAdmin:  "frontend-admin",
	models.RoleFrontendMobile: "mobile",
	models.RoleDocs:           "docs",
	models.RoleCLI:            "cli",
	models.RoleWorker:         "worker",
}

// roleSubfolder mengembalikan subfolder untuk peran yang dikenal atau yang
// mengandung "front"/"mobile"; ok=false bila perlu memakai nama komponen.
func roleSubfolder(role string) (string, bool) {
	if sub, ok := roleSubfolders[role]; ok {
		return sub, true
	}
	if strings.Contains(role, "front") {
		return "frontend", true
	}
	if strings.Contains(role, "mobile") {
		return "mobile", true
	}
	return "", false
}

func reorgRoleSubfolder(c models.Component) string {
	if sub, ok := roleSubfolder(c.Role); ok {
		return sub
	}
	return Slugify(c.Name)
}

func buildReorgScript(moves []models.FolderMoveItem, targetDir string) string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("# vibeasik Workspace Reorganization Script\n")
	sb.WriteString("# Dibuat otomatis pada: " + time.Now().Format(time.RFC1123) + "\n")
	sb.WriteString("# Target Folder: " + targetDir + "\n\n")
	sb.WriteString("set -e\n\n")
	sb.WriteString("echo '=== Memulai Reorganisasi Folder Proyek vibeasik ==='\n")
	sb.WriteString("mkdir -p \"" + targetDir + "\"\n\n")

	for _, m := range moves {
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
	return sb.String()
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

		if status, msg := performMove(src, dst); status != "success" {
			m.Status = status
			m.Error = msg
			result.FailedCount++
			result.Moves = append(result.Moves, m)
			continue
		}

		m.Status = "success"
		m.Error = ""
		result.SuccessCount++

		if payload.UpdateDB {
			s.syncMovedPaths(ctx, m, src, dst)
		}
		result.Moves = append(result.Moves, m)
	}

	result.Message = fmt.Sprintf("Berhasil memindahkan %d folder, %d gagal, %d dilewati.", result.SuccessCount, result.FailedCount, result.SkippedCount)
	return result, nil
}

// performMove memverifikasi lalu memindahkan src ke dst. Mengembalikan status
// "success", atau status gagal ("failed"/"conflict") beserta pesan galatnya.
func performMove(src, dst string) (status, errMsg string) {
	if !isDirPath(src) {
		return "failed", "Folder sumber tidak ada di disk: " + src
	}
	if _, err := os.Stat(dst); err == nil {
		return "conflict", "Folder tujuan sudah ada di disk: " + dst
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return "failed", "Gagal membuat direktori induk tujuan: " + err.Error()
	}
	if err := moveOrCopy(src, dst); err != nil {
		return "failed", "Gagal memindahkan folder: " + err.Error()
	}
	return "success", ""
}

// syncMovedPaths memperbarui path komponen dan root proyek di database setelah pemindahan.
func (s *organizerService) syncMovedPaths(ctx context.Context, m models.FolderMoveItem, src, dst string) {
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

func getComponentSubfolder(c models.Component) string {
	if sub, ok := roleSubfolder(c.Role); ok {
		return sub
	}
	clean := Slugify(c.Name)
	if clean == "" {
		clean = fmt.Sprintf("component-%d", c.ID)
	}
	return clean
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

	targetParentDir = s.resolveConsolidateParent(ctx, targetParentDir)

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

	var totalSize int64
	if len(p.Components) > 0 {
		totalSize = previewComponentMoves(preview, p, slug)
	} else {
		totalSize = previewStandaloneMove(preview, p, slug)
	}

	preview.TotalSizeBytes = totalSize
	return preview, nil
}

func (s *organizerService) resolveConsolidateParent(ctx context.Context, targetParentDir string) string {
	targetParentDir = strings.TrimSpace(targetParentDir)
	if targetParentDir == "" {
		ws, _ := s.GetWorkspaceSettings(ctx)
		if ws != nil && ws.DefaultProjectsRoot != "" {
			targetParentDir = ws.DefaultProjectsRoot
		} else {
			targetParentDir = models.DefaultProjectsRoot()
		}
	}
	return filepath.Clean(targetParentDir)
}

// previewStandaloneMove menyusun item tunggal untuk proyek tanpa komponen.
func previewStandaloneMove(preview *models.ConsolidatePreview, p *models.Project, slug string) int64 {
	srcPath := strings.TrimSpace(p.RootPath)
	if srcPath == "" || srcPath == "-" {
		return 0
	}
	item := models.FolderMoveItem{
		ID:                 "item-1",
		ProjectID:          p.ID,
		ProjectName:        p.Name,
		CurrentPath:        filepath.Clean(srcPath),
		TargetPath:         preview.ProjectTargetDir,
		RelativeTargetPath: slug,
		Status:             "pending",
	}
	if assessConsolidateMove(&item, p.ID) {
		preview.HasConflict = true
		preview.CanConsolidate = false
	}
	preview.Items = append(preview.Items, item)
	return item.SizeBytes
}

// newConsolidateItem membuat item awal untuk satu folder sumber (beserta komponen kembarannya).
func newConsolidateItem(p *models.Project, idx int, src consolidateSource) models.FolderMoveItem {
	c := src.components[0]
	item := models.FolderMoveItem{
		ID:            fmt.Sprintf("item-%d", idx+1),
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
	return item
}

// enclosingSourcePath mencari folder komponen terdalam yang memuat src ("" bila tidak ada).
func enclosingSourcePath(sources []consolidateSource, src consolidateSource) string {
	parent := ""
	for _, other := range sources {
		if pathWithin(other.path, src.path) && (parent == "" || len(other.path) > len(parent)) {
			parent = other.path
		}
	}
	return parent
}

// previewComponentMoves menyusun item per folder sumber komponen dan mengembalikan total ukuran.
func previewComponentMoves(preview *models.ConsolidatePreview, p *models.Project, slug string) int64 {
	var totalSize int64
	usedSubfolders := make(map[string]int)
	sources := groupConsolidateSources(p.Components)
	// targetOf menyimpan tujuan setiap folder sumber agar komponen bersarang (folder di dalam komponen lain)
	// ikut pindah bersama induknya alih-alih dipindahkan dua kali.
	targetOf := map[string]string{}
	for i, src := range sources {
		item := newConsolidateItem(p, i, src)

		if parent := enclosingSourcePath(sources, src); parent != "" {
			// Diselesaikan setelah semua induk punya tujuan (lihat resolveNestedTargets).
			item.Status = "moves_with_parent"
			item.Warning = "Berada di dalam folder komponen lain; ikut pindah bersama induknya."
			item.Error = parent
			preview.Items = append(preview.Items, item)
			continue
		}

		subfolder := getComponentSubfolder(src.components[0])
		usedSubfolders[subfolder]++
		if usedSubfolders[subfolder] > 1 {
			subfolder = fmt.Sprintf("%s-%s", subfolder, Slugify(src.components[0].Name))
		}
		item.TargetPath = filepath.Join(preview.ProjectTargetDir, subfolder)
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
	resolveNestedTargets(preview, targetOf)
	return totalSize
}

// resolveNestedTargets mengisi tujuan item "moves_with_parent" dari tujuan induknya.
func resolveNestedTargets(preview *models.ConsolidatePreview, targetOf map[string]string) {
	for i := range preview.Items {
		item := &preview.Items[i]
		if item.Status != "moves_with_parent" {
			continue
		}
		parent := item.Error
		item.Error = ""
		rel, _ := filepath.Rel(parent, item.CurrentPath)
		item.TargetPath = filepath.Join(targetOf[parent], rel)
		item.RelativeTargetPath, _ = filepath.Rel(preview.TargetParentDir, item.TargetPath)
	}
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
		warnings = append(warnings, moveVibeasikEntries(srcDir, dstDir, entries)...)
		// Hanya dihapus bila sudah kosong; sisa berkas (duplikat yang sudah ada di root) dibiarkan.
		if err := os.Remove(srcDir); err != nil && !os.IsNotExist(err) {
			warnings = append(warnings, fmt.Sprintf("%s tidak kosong, dibiarkan (berkas yang sama sudah ada di root)", srcDir))
		}
	}
	return warnings
}

// moveVibeasikEntries memindahkan isi srcDir ke dstDir tanpa menimpa; peta kode lama dibuang.
func moveVibeasikEntries(srcDir, dstDir string, entries []os.DirEntry) []string {
	var warnings []string
	for _, e := range entries {
		from := filepath.Join(srcDir, e.Name())
		if e.Name() == "CODEMAP.md" || e.Name() == "codemap.jsonl" {
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

	movedItems, movedCount, err := moveConsolidateItems(preview.Items)
	if err != nil {
		return nil, err
	}
	// Komponen bersarang baru ada di tujuannya setelah induknya pindah, jadi path disimpan setelah semua pemindahan.
	for _, item := range movedItems {
		if err := s.saveComponentPaths(ctx, item); err != nil {
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
	linkedFiles, codeMapMarkdown, warnings := s.initUnifiedWorkspace(projectID, apiBase, warnings)

	return &models.ConsolidateResult{
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
	}, nil
}

// moveConsolidateItems memindahkan item berstatus "pending" dan mengembalikan semua item (status diperbarui).
func moveConsolidateItems(items []models.FolderMoveItem) ([]models.FolderMoveItem, int, error) {
	var movedItems []models.FolderMoveItem
	movedCount := 0
	for _, item := range items {
		if item.Status == "pending" {
			if err := os.MkdirAll(filepath.Dir(item.TargetPath), 0755); err != nil {
				return nil, 0, fmt.Errorf("gagal membuat folder tujuan %s: %w", item.TargetPath, err)
			}
			if err := moveOrCopy(item.CurrentPath, item.TargetPath); err != nil {
				return nil, 0, fmt.Errorf("gagal memindahkan %s ke %s: %w", item.CurrentPath, item.TargetPath, err)
			}
			item.Status = "success"
			movedCount++
		}
		movedItems = append(movedItems, item)
	}
	return movedItems, movedCount, nil
}

// saveComponentPaths menyimpan path komponen setelah foldernya dipindah, agar database tidak tertinggal dari isi
// disk bila pemindahan berikutnya gagal (penyatuan ulang lalu mengenali folder yang sudah di target).
func (s *organizerService) saveComponentPaths(ctx context.Context, item models.FolderMoveItem) error {
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

// initUnifiedWorkspace menjalankan Init vibeasik di root proyek tersatukan (bila db tersedia).
func (s *organizerService) initUnifiedWorkspace(projectID uint, apiBase string, warnings []string) (linkedFiles []string, codeMap string, out []string) {
	if s.db == nil {
		return nil, "", warnings
	}
	init, err := workspace.InitProject(s.db, projectID, apiBase)
	if err == nil {
		return init.FilesCreated, init.CodeMapMarkdown, warnings
	}
	warnings = append(warnings, "Init vibeasik di root gagal: "+err.Error())
	if created, _, linkErr := workspace.LinkProjectWorkspaces(s.db, projectID, 0, apiBase); linkErr == nil {
		linkedFiles = created
	}
	return linkedFiles, "", warnings
}
