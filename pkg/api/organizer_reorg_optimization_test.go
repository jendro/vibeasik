package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
)

// TestProjectsTablePrimaryKeyAndExplainPlan memverifikasi:
// 1. Kolom id pada tabel projects memiliki constraint primary key.
// 2. Query planner SQLite menggunakan primary key index saat WHERE id IN (...).
func TestProjectsTablePrimaryKeyAndExplainPlan(t *testing.T) {
	s := newGuardTestServer(t)

	// 1. Periksa PRAGMA table_info('projects')
	type tableInfoRow struct {
		Cid       int     `gorm:"column:cid"`
		Name      string  `gorm:"column:name"`
		Type      string  `gorm:"column:type"`
		NotNull   int     `gorm:"column:notnull"`
		DfltValue *string `gorm:"column:dflt_value"`
		Pk        int     `gorm:"column:pk"`
	}

	var cols []tableInfoRow
	if err := s.db.Raw("PRAGMA table_info('projects')").Scan(&cols).Error; err != nil {
		t.Fatalf("gagal membaca PRAGMA table_info('projects'): %v", err)
	}

	var idCol *tableInfoRow
	for i := range cols {
		if cols[i].Name == "id" {
			idCol = &cols[i]
			break
		}
	}

	if idCol == nil {
		t.Fatalf("kolom 'id' tidak ditemukan pada tabel projects")
	}
	if idCol.Pk == 0 {
		t.Fatalf("kolom 'id' bukan primary key (pk=%d)", idCol.Pk)
	}
	t.Logf("✅ Kolom 'id' terverifikasi sebagai PRIMARY KEY (type: %s, pk: %d)", idCol.Type, idCol.Pk)

	// 2. EXPLAIN QUERY PLAN untuk query WHERE id IN (?, ?)
	type explainPlanRow struct {
		ID     int    `gorm:"column:id"`
		Parent int    `gorm:"column:parent"`
		NotUsed int   `gorm:"column:notused"`
		Detail string `gorm:"column:detail"`
	}

	var planRows []explainPlanRow
	query := "EXPLAIN QUERY PLAN SELECT * FROM projects WHERE id IN (?, ?, ?)"
	if err := s.db.Raw(query, 1, 2, 3).Scan(&planRows).Error; err != nil {
		t.Fatalf("gagal menjalankan EXPLAIN QUERY PLAN: %v", err)
	}

	if len(planRows) == 0 {
		t.Fatalf("hasil EXPLAIN QUERY PLAN kosong")
	}

	usesPKIndex := false
	for _, row := range planRows {
		t.Logf("EXPLAIN QUERY PLAN detail: %s", row.Detail)
		// SQLite menggunakan "SEARCH TABLE projects USING INTEGER PRIMARY KEY (rowid=?)" atau "SEARCH projects USING PRIMARY KEY"
		detailUpper := strings.ToUpper(row.Detail)
		if strings.Contains(detailUpper, "PRIMARY KEY") || strings.Contains(detailUpper, "ROWID") {
			usesPKIndex = true
		}
	}

	if !usesPKIndex {
		t.Errorf("EXPLAIN QUERY PLAN tidak menggunakan PRIMARY KEY index: %+v", planRows)
	}
}

// TestOrganizerPlanOptimization_SemanticAndLatency memverifikasi:
// 1. Semantik hasil: request dengan ProjectIDs hanya memproses project yang diminta.
// 2. Efisiensi pembacaan baris: FindByIDs hanya membaca N baris yang diminta, bukan total proyek.
// 3. Distribusi latensi p50 & p95 tetap jauh di bawah ambang batas responsif 200ms SLO.
func TestOrganizerPlanOptimization_SemanticAndLatency(t *testing.T) {
	s := newGuardTestServer(t)
	ctx := context.Background()
	baseDir := t.TempDir()
	targetDir := filepath.Join(baseDir, "reorg_target")

	// Bersihkan tabel mock awal agar dataset bersih
	s.db.Exec("DELETE FROM components")
	s.db.Exec("DELETE FROM projects")

	// Siapkan 50 project dengan masing-masing 2 komponen
	totalProjects := 50
	createdProjectIDs := make([]uint, 0, totalProjects)

	for i := 1; i <= totalProjects; i++ {
		projDir := filepath.Join(baseDir, fmt.Sprintf("proj_%02d", i))
		proj := models.Project{
			Name:        fmt.Sprintf("Project %02d", i),
			Slug:        fmt.Sprintf("proj-%02d", i),
			RootPath:    projDir,
			ProjectType: models.TypeBundle,
			Status:      models.StatusActive,
			Components: []models.Component{
				{
					Name:         fmt.Sprintf("be_%02d", i),
					Role:         models.RoleBackend,
					AbsolutePath: filepath.Join(projDir, "be"),
				},
				{
					Name:         fmt.Sprintf("fe_%02d", i),
					Role:         models.RoleFrontendClient,
					AbsolutePath: filepath.Join(projDir, "fe"),
				},
			},
		}
		if err := s.db.Create(&proj).Error; err != nil {
			t.Fatalf("gagal membuat mock project %d: %v", i, err)
		}
		createdProjectIDs = append(createdProjectIDs, proj.ID)
	}

	newPlanReq := func(body []byte) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "http://localhost/api/organizer/plan", bytes.NewReader(body))
		req.Host = "localhost"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(localauth.Header, testLocalToken)
		return req
	}

	// 1. Uji Semantik: Request dengan sebagian project (misal 3 project pertama)
	selectedIDs := createdProjectIDs[:3]
	reqBodyWithIDs, err := json.Marshal(models.ReorgPlanRequest{
		TargetDirectory: targetDir,
		ProjectIDs:      selectedIDs,
		Strategy:        models.StrategyProductCentric,
	})
	if err != nil {
		t.Fatal(err)
	}

	reqWithIDs := newPlanReq(reqBodyWithIDs)
	respWithIDs, err := s.app.Test(reqWithIDs, 5000)
	if err != nil {
		t.Fatalf("request /api/organizer/plan gagal: %v", err)
	}
	if respWithIDs.StatusCode != http.StatusOK {
		t.Fatalf("status code salah: want 200, got %d", respWithIDs.StatusCode)
	}

	var planFiltered models.ReorgPlan
	if err := json.NewDecoder(respWithIDs.Body).Decode(&planFiltered); err != nil {
		t.Fatalf("gagal decode response json: %v", err)
	}

	// Masing-masing dari 3 project memiliki 2 komponen -> 6 planned moves
	wantFilteredMoves := len(selectedIDs) * 2
	if planFiltered.TotalItems != wantFilteredMoves {
		t.Errorf("TotalItems tidak cocok: want %d, got %d", wantFilteredMoves, planFiltered.TotalItems)
	}

	// Verifikasi bahwa move yang ada hanya untuk project yang dipilih
	idMap := make(map[uint]bool)
	for _, id := range selectedIDs {
		idMap[id] = true
	}
	for _, m := range planFiltered.PlannedMoves {
		if !idMap[m.ProjectID] {
			t.Errorf("ditemukan planned move untuk project ID %d di luar filter %v", m.ProjectID, selectedIDs)
		}
	}

	// 2. Uji Semantik: Request tanpa ProjectIDs (harus memuat seluruh 50 project)
	reqBodyAll, err := json.Marshal(models.ReorgPlanRequest{
		TargetDirectory: targetDir,
		Strategy:        models.StrategyProductCentric,
	})
	if err != nil {
		t.Fatal(err)
	}

	reqAll := newPlanReq(reqBodyAll)
	respAll, err := s.app.Test(reqAll, 5000)
	if err != nil {
		t.Fatalf("request /api/organizer/plan all gagal: %v", err)
	}
	if respAll.StatusCode != http.StatusOK {
		t.Fatalf("status code salah: want 200, got %d", respAll.StatusCode)
	}

	var planAll models.ReorgPlan
	if err := json.NewDecoder(respAll.Body).Decode(&planAll); err != nil {
		t.Fatalf("gagal decode response json: %v", err)
	}

	wantAllMoves := totalProjects * 2
	if planAll.TotalItems != wantAllMoves {
		t.Errorf("TotalItems tanpa filter tidak cocok: want %d, got %d", wantAllMoves, planAll.TotalItems)
	}

	// 3. Verifikasi Database Read Reduction via FindByIDs vs FindAll
	repo := repository.NewProjectRepository(s.db)
	allProjectsLoaded, err := repo.FindAll(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	filteredProjectsLoaded, err := repo.FindByIDs(ctx, selectedIDs)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("📊 Analisis Baris Database:")
	t.Logf("   - Tanpa filter (FindAll): %d baris project dimuat", len(allProjectsLoaded))
	t.Logf("   - Dengan filter (FindByIDs): %d baris project dimuat (penghematan: %d baris / %.1f%%)",
		len(filteredProjectsLoaded),
		len(allProjectsLoaded)-len(filteredProjectsLoaded),
		float64(len(allProjectsLoaded)-len(filteredProjectsLoaded))/float64(len(allProjectsLoaded))*100.0)

	if len(filteredProjectsLoaded) != len(selectedIDs) {
		t.Errorf("FindByIDs mengembalikan jumlah baris yang salah: want %d, got %d", len(selectedIDs), len(filteredProjectsLoaded))
	}

	// 4. Pengukuran Latensi p50 & p95 (50 iterasi)
	iterations := 50
	latenciesFiltered := make([]time.Duration, iterations)
	latenciesAll := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		// Filtered (3 of 50)
		start := time.Now()
		reqF := newPlanReq(reqBodyWithIDs)
		resF, err := s.app.Test(reqF, 5000)
		if err != nil || resF.StatusCode != 200 {
			t.Fatalf("error during latency benchmark: %v", err)
		}
		latenciesFiltered[i] = time.Since(start)

		// All (50 of 50)
		startAll := time.Now()
		reqA := newPlanReq(reqBodyAll)
		resA, err := s.app.Test(reqA, 5000)
		if err != nil || resA.StatusCode != 200 {
			t.Fatalf("error during latency benchmark: %v", err)
		}
		latenciesAll[i] = time.Since(startAll)
	}

	sort.Slice(latenciesFiltered, func(i, j int) bool { return latenciesFiltered[i] < latenciesFiltered[j] })
	sort.Slice(latenciesAll, func(i, j int) bool { return latenciesAll[i] < latenciesAll[j] })

	p50Filtered := latenciesFiltered[iterations*50/100]
	p95Filtered := latenciesFiltered[iterations*95/100]

	p50All := latenciesAll[iterations*50/100]
	p95All := latenciesAll[iterations*95/100]

	t.Logf("📈 Distribusi Latensi (/api/organizer/plan):")
	t.Logf("   - Dengan ProjectIDs (3/50): p50 = %v, p95 = %v", p50Filtered, p95Filtered)
	t.Logf("   - Tanpa ProjectIDs (50/50): p50 = %v, p95 = %v", p50All, p95All)

	if p95Filtered > 200*time.Millisecond {
		t.Errorf("p95 latency (%v) melebihi batas kenyamanan 200ms", p95Filtered)
	}
}

func BenchmarkOrganizerPlan(b *testing.B) {
	s := newGuardTestServer(&testing.T{})
	baseDir := b.TempDir()
	targetDir := filepath.Join(baseDir, "reorg_target")

	s.db.Exec("DELETE FROM components")
	s.db.Exec("DELETE FROM projects")

	totalProjects := 50
	createdProjectIDs := make([]uint, 0, totalProjects)

	for i := 1; i <= totalProjects; i++ {
		projDir := filepath.Join(baseDir, fmt.Sprintf("proj_%02d", i))
		proj := models.Project{
			Name:        fmt.Sprintf("Project %02d", i),
			Slug:        fmt.Sprintf("proj-%02d", i),
			RootPath:    projDir,
			ProjectType: models.TypeBundle,
			Status:      models.StatusActive,
			Components: []models.Component{
				{
					Name:         fmt.Sprintf("be_%02d", i),
					Role:         models.RoleBackend,
					AbsolutePath: filepath.Join(projDir, "be"),
				},
				{
					Name:         fmt.Sprintf("fe_%02d", i),
					Role:         models.RoleFrontendClient,
					AbsolutePath: filepath.Join(projDir, "fe"),
				},
			},
		}
		_ = s.db.Create(&proj)
		createdProjectIDs = append(createdProjectIDs, proj.ID)
	}

	selectedIDs := createdProjectIDs[:3]
	reqBodyWithIDs, _ := json.Marshal(models.ReorgPlanRequest{
		TargetDirectory: targetDir,
		ProjectIDs:      selectedIDs,
		Strategy:        models.StrategyProductCentric,
	})
	reqBodyAll, _ := json.Marshal(models.ReorgPlanRequest{
		TargetDirectory: targetDir,
		Strategy:        models.StrategyProductCentric,
	})

	newPlanReq := func(body []byte) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "http://localhost/api/organizer/plan", bytes.NewReader(body))
		req.Host = "localhost"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(localauth.Header, testLocalToken)
		return req
	}

	b.Run("WithProjectIDs_3of50", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req := newPlanReq(reqBodyWithIDs)
			resp, err := s.app.Test(req, 5000)
			if err != nil || resp.StatusCode != http.StatusOK {
				b.Fatalf("failed request: %v", err)
			}
		}
	})

	b.Run("WithoutProjectIDs_All50", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req := newPlanReq(reqBodyAll)
			resp, err := s.app.Test(req, 5000)
			if err != nil || resp.StatusCode != http.StatusOK {
				b.Fatalf("failed request: %v", err)
			}
		}
	})
}
