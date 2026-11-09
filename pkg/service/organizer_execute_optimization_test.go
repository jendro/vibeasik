package service_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
	"github.com/vibeasik/vibeasik/pkg/service"
)

// spyComponentRepo membungkus ComponentRepository untuk menghitung jumlah FindByID
type spyComponentRepo struct {
	repository.ComponentRepository
	findByIDCalls atomic.Int64
	updateCalls   atomic.Int64
}

func (s *spyComponentRepo) FindByID(ctx context.Context, id uint) (*models.Component, error) {
	s.findByIDCalls.Add(1)
	return s.ComponentRepository.FindByID(ctx, id)
}

func (s *spyComponentRepo) Update(ctx context.Context, comp *models.Component) error {
	s.updateCalls.Add(1)
	return s.ComponentRepository.Update(ctx, comp)
}

// spyProjectRepo membungkus ProjectRepository untuk menghitung jumlah FindByID
type spyProjectRepo struct {
	repository.ProjectRepository
	findByIDCalls atomic.Int64
	updateCalls   atomic.Int64
}

func (s *spyProjectRepo) FindByID(ctx context.Context, id uint) (*models.Project, error) {
	s.findByIDCalls.Add(1)
	return s.ProjectRepository.FindByID(ctx, id)
}

func (s *spyProjectRepo) Update(ctx context.Context, p *models.Project) error {
	s.updateCalls.Add(1)
	return s.ProjectRepository.Update(ctx, p)
}

func TestExecuteMoves_IdentityMapCacheVerification(t *testing.T) {
	_, db := setupTestServices(t)
	ctx := context.Background()

	// 1. Siapkan data fixture di DB
	proj := models.Project{
		Name:     "Test Project",
		Slug:     "test-project",
		RootPath: "/tmp/old-proj-root",
	}
	if err := db.Create(&proj).Error; err != nil {
		t.Fatalf("gagal membuat project fixture: %v", err)
	}

	comp := models.Component{
		ProjectID:    proj.ID,
		Name:         "Frontend",
		Role:         models.RoleFrontendClient,
		AbsolutePath: "/tmp/old-proj-root/fe",
		RelativePath: "fe",
	}
	if err := db.Create(&comp).Error; err != nil {
		t.Fatalf("gagal membuat component fixture: %v", err)
	}

	// 2. Buat repository asli dan bungkus dengan spy
	baseRepos := repository.NewRepositories(db)
	spyComp := &spyComponentRepo{ComponentRepository: baseRepos.Components}
	spyProj := &spyProjectRepo{ProjectRepository: baseRepos.Projects}

	customRepos := &repository.Repositories{
		Projects:    spyProj,
		Components:  spyComp,
		Routes:      baseRepos.Routes,
		Diagnostics: baseRepos.Diagnostics,
		Settings:    baseRepos.Settings,
		Security:    baseRepos.Security,
	}

	customServices := service.NewServices(customRepos, db)

	// 3. Buat folder sementara di disk untuk 5 operasi move
	tempDir := t.TempDir()
	moves := make([]models.FolderMoveItem, 0, 5)

	for i := 1; i <= 5; i++ {
		srcDir := filepath.Join(tempDir, fmt.Sprintf("src_%d", i))
		dstDir := filepath.Join(tempDir, fmt.Sprintf("dst_%d", i))
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			t.Fatalf("gagal membuat direktori sumber: %v", err)
		}
		moves = append(moves, models.FolderMoveItem{
			ID:          fmt.Sprintf("move-%d", i),
			ProjectID:   proj.ID,
			ComponentID: comp.ID,
			CurrentPath: srcDir,
			TargetPath:  dstDir,
			Status:      "pending",
		})
	}

	payload := models.ExecuteMovePayload{
		Moves:    moves,
		UpdateDB: true,
	}

	// 4. Eksekusi pemindahan
	result, err := customServices.Organizer.ExecuteMoves(ctx, payload)
	if err != nil {
		t.Fatalf("ExecuteMoves error: %v", err)
	}

	if result.SuccessCount != 5 {
		t.Fatalf("harapan 5 sukses, dapat: %d (gagal: %d)", result.SuccessCount, result.FailedCount)
	}

	// 5. Verifikasi pemanggilan FindByID (Identity Map Cache)
	compFindCalls := spyComp.findByIDCalls.Load()
	projFindCalls := spyProj.findByIDCalls.Load()

	t.Logf("✅ Component FindByID dipanggil: %d kali (dari 5 move items)", compFindCalls)
	t.Logf("✅ Project FindByID dipanggil: %d kali (dari 5 move items)", projFindCalls)

	if compFindCalls != 1 {
		t.Errorf("Ekspektasi Component FindByID dipanggil tepat 1 kali, tapi dipanggil %d kali (N+1 query tidak tereliminasi)", compFindCalls)
	}
	if projFindCalls != 1 {
		t.Errorf("Ekspektasi Project FindByID dipanggil tepat 1 kali, tapi dipanggil %d kali (N+1 query tidak tereliminasi)", projFindCalls)
	}
}

func TestExecuteMoves_NegativeCacheVerification(t *testing.T) {
	_, db := setupTestServices(t)
	ctx := context.Background()

	baseRepos := repository.NewRepositories(db)
	spyComp := &spyComponentRepo{ComponentRepository: baseRepos.Components}
	spyProj := &spyProjectRepo{ProjectRepository: baseRepos.Projects}

	customRepos := &repository.Repositories{
		Projects:    spyProj,
		Components:  spyComp,
		Routes:      baseRepos.Routes,
		Diagnostics: baseRepos.Diagnostics,
		Settings:    baseRepos.Settings,
		Security:    baseRepos.Security,
	}

	customServices := service.NewServices(customRepos, db)

	tempDir := t.TempDir()
	moves := make([]models.FolderMoveItem, 0, 4)

	// Item dengan ComponentID & ProjectID yang tidak ada di DB (negative caching)
	for i := 1; i <= 4; i++ {
		srcDir := filepath.Join(tempDir, fmt.Sprintf("src_neg_%d", i))
		dstDir := filepath.Join(tempDir, fmt.Sprintf("dst_neg_%d", i))
		_ = os.MkdirAll(srcDir, 0755)
		moves = append(moves, models.FolderMoveItem{
			ID:          fmt.Sprintf("move-neg-%d", i),
			ProjectID:   99999, // tidak ada di DB
			ComponentID: 88888, // tidak ada di DB
			CurrentPath: srcDir,
			TargetPath:  dstDir,
			Status:      "pending",
		})
	}

	payload := models.ExecuteMovePayload{
		Moves:    moves,
		UpdateDB: true,
	}

	result, err := customServices.Organizer.ExecuteMoves(ctx, payload)
	if err != nil {
		t.Fatalf("ExecuteMoves error: %v", err)
	}
	if result.SuccessCount != 4 {
		t.Fatalf("harapan 4 sukses, dapat: %d", result.SuccessCount)
	}

	// Negative cache harus memastikan query tidak diulang walau nil
	if calls := spyComp.findByIDCalls.Load(); calls != 1 {
		t.Errorf("Ekspektasi negative cache mencegah query berulang untuk ComponentID 88888 (panggilan: %d)", calls)
	}
	if calls := spyProj.findByIDCalls.Load(); calls != 1 {
		t.Errorf("Ekspektasi negative cache mencegah query berulang untuk ProjectID 99999 (panggilan: %d)", calls)
	}
}

func BenchmarkExecuteMoves_IdentityMapCache(b *testing.B) {
	_, db := setupTestServices(&testing.T{})
	ctx := context.Background()

	proj := models.Project{
		Name:     "Benchmark Project",
		Slug:     "bench-proj",
		RootPath: "/tmp/bench",
	}
	_ = db.Create(&proj)

	comp := models.Component{
		ProjectID:    proj.ID,
		Name:         "Bench Component",
		Role:         models.RoleBackend,
		AbsolutePath: "/tmp/bench/be",
	}
	_ = db.Create(&comp)

	repos := repository.NewRepositories(db)
	services := service.NewServices(repos, db)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tempDir, _ := os.MkdirTemp("", "bench_move_*")
		moves := make([]models.FolderMoveItem, 0, 10)
		for j := 0; j < 10; j++ {
			s := filepath.Join(tempDir, fmt.Sprintf("s_%d", j))
			d := filepath.Join(tempDir, fmt.Sprintf("d_%d", j))
			_ = os.MkdirAll(s, 0755)
			moves = append(moves, models.FolderMoveItem{
				ID:          fmt.Sprintf("bm-%d", j),
				ProjectID:   proj.ID,
				ComponentID: comp.ID,
				CurrentPath: s,
				TargetPath:  d,
				Status:      "pending",
			})
		}
		payload := models.ExecuteMovePayload{
			Moves:    moves,
			UpdateDB: true,
		}
		b.StartTimer()

		_, _ = services.Organizer.ExecuteMoves(ctx, payload)

		b.StopTimer()
		_ = os.RemoveAll(tempDir)
		b.StartTimer()
	}
}
