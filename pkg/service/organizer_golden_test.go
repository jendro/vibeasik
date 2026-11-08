package service_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
)

var scriptStamp = regexp.MustCompile(`(?m)^# Dibuat otomatis pada: .*$`)

func mkdirs(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func gitInit(t *testing.T, dir string, dirty bool) {
	t.Helper()
	if err := exec.Command("git", "-C", dir, "init", "-q").Run(); err != nil {
		t.Fatal(err)
	}
	if dirty {
		writeFiles(t, dir, map[string]string{"new.txt": "x"})
	}
}

func TestGenerateReorgPlanGolden(t *testing.T) {
	services, db := setupTestServices(t)
	_ = db.AutoMigrate(&models.WorkspaceSettings{})
	ctx := context.Background()
	base := t.TempDir()
	t.Setenv("HOME", filepath.Join(base, "home"))
	target := filepath.Join(base, "ws")

	d := func(n string) string { return filepath.Join(base, "src", n) }
	for _, n := range []string{"be", "fe", "adm", "mob", "cli", "wrk", "doc", "x1", "x2", "x3", "solo", "solo2", "inplace"} {
		mkdirs(t, d(n))
	}
	writeFiles(t, d("be"), map[string]string{"a.txt": "12345"})
	gitInit(t, d("be"), true)
	gitInit(t, d("fe"), false)
	gitInit(t, d("solo"), true)
	mkdirs(t, filepath.Join(target, "shop", "mobile"), filepath.Join(target, "dup"))
	mkdirs(t, filepath.Join(target, "shop", "frontend"))

	comp := func(name, role, path string) models.Component {
		return models.Component{Name: name, Role: role, AbsolutePath: path}
	}
	projects := []models.Project{
		{Name: "ShopBackend", Slug: "p1", Components: []models.Component{
			comp("api", models.RoleBackend, d("be")),
			comp("web", models.RoleFrontendClient, d("fe")),
			comp("adm", models.RoleFrontendAdmin, d("adm")),
			comp("mob", models.RoleFrontendMobile, d("mob")),
			comp("cli", models.RoleCLI, d("cli")),
			comp("wrk", models.RoleWorker, d("wrk")),
			comp("doc", models.RoleDocs, d("doc")),
			comp("gone", models.RoleBackend, d("missing")),
			comp("nopath", models.RoleBackend, "-"),
			comp("blank", models.RoleBackend, ""),
			comp("fx", "frontend_x", d("x1")),
			comp("mx", "xmobile", d("x2")),
			comp("Other Thing", "weird", d("x3")),
		}},
		{Name: "Solo", Slug: "p2", RootPath: d("solo")},
		{Name: "SoloMissing", Slug: "p3", RootPath: d("nope")},
		{Name: "Dash", Slug: "p4", RootPath: "-"},
		{Name: "---", Slug: "p5", RootPath: d("solo2")},
		{Name: "Dup", Slug: "p6", RootPath: d("inplace")},
		{Name: "InPlace", Slug: "p7", RootPath: filepath.Join(target, "inplace")},
	}
	mkdirs(t, filepath.Join(target, "inplace"))
	for i := range projects {
		if err := db.Create(&projects[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	norm := func(plan *models.ReorgPlan) *models.ReorgPlan {
		plan.CreatedAt = time.Time{}
		// ukuran folder git tidak deterministik; periksa konsistensi lalu nolkan
		var sum int64
		for i := range plan.PlannedMoves {
			sum += plan.PlannedMoves[i].SizeBytes
			if plan.PlannedMoves[i].IsGitRepo {
				plan.PlannedMoves[i].SizeBytes = 0
			}
		}
		if sum != plan.TotalSize {
			t.Fatalf("total size %d != sum %d", plan.TotalSize, sum)
		}
		plan.TotalSize = 0
		for i := range plan.PlannedMoves {
			plan.TotalSize += plan.PlannedMoves[i].SizeBytes
		}
		plan.GeneratedScript = scriptStamp.ReplaceAllString(plan.GeneratedScript, "# Dibuat otomatis pada: <T>")
		return plan
	}
	var out []any
	for _, req := range []models.ReorgPlanRequest{
		{TargetDirectory: target},
		{TargetDirectory: target + "/", Strategy: "custom", ProjectIDs: []uint{1, 3}},
		{ProjectIDs: []uint{2}},
	} {
		plan, err := services.Organizer.GenerateReorgPlan(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, norm(plan))
	}
	golden(t, "reorg_plan", out, base)
}

func TestPreviewConsolidateGolden(t *testing.T) {
	services, db := setupTestServices(t)
	_ = db.AutoMigrate(&models.WorkspaceSettings{})
	ctx := context.Background()
	base := t.TempDir()
	t.Setenv("HOME", filepath.Join(base, "home"))
	target := filepath.Join(base, "ws")
	d := func(n string) string { return filepath.Join(base, "src", n) }

	for _, n := range []string{"be", "be2", "fe", "fe2", "gone", "conf", "solo", "solo-conf"} {
		mkdirs(t, d(n))
	}
	mkdirs(t, filepath.Join(d("be"), "docs"), filepath.Join(d("be"), "docs", "deep"))
	writeFiles(t, d("be"), map[string]string{"a.txt": "123"})
	gitInit(t, d("fe"), true)
	// target yang sudah ada dan milik proyek lain / tanpa pemilik
	mkdirs(t, filepath.Join(target, "conflict", "backend"))
	mkdirs(t, filepath.Join(target, "half", "frontend"))
	writeFiles(t, filepath.Join(target, "half", "frontend"), map[string]string{".vibeasik.json": `{"project_id": 99}`})
	mkdirs(t, filepath.Join(target, "half", "backend"))
	writeFiles(t, filepath.Join(target, "half", "backend"), map[string]string{"z": "1"})
	mkdirs(t, filepath.Join(target, "solo-conflict"))
	mkdirs(t, filepath.Join(target, "inplace", "backend"))

	comp := func(name, role, path string) models.Component {
		return models.Component{Name: name, Role: role, AbsolutePath: path}
	}
	projects := []models.Project{
		{Name: "Normal", Slug: "normal", Components: []models.Component{
			comp("api", models.RoleBackend, d("be")), comp("api-dup", models.RoleBackend, d("be")),
			comp("docs", models.RoleDocs, filepath.Join(d("be"), "docs")),
			comp("deep", models.RoleCLI, filepath.Join(d("be"), "docs", "deep")),
			comp("web", models.RoleFrontendClient, d("fe")),
			comp("web2", models.RoleFrontendClient, d("fe2")),
			comp("nopath", models.RoleWorker, "-"),
		}},
		{Name: "Conflict", Slug: "conflict", Components: []models.Component{
			comp("be", models.RoleBackend, d("be2")), comp("gone", models.RoleFrontendClient, d("gone")),
		}},
		{Name: "Half", Slug: "half", Components: []models.Component{
			comp("fe", models.RoleFrontendClient, filepath.Join(target, "half", "frontend")),
			comp("be", models.RoleBackend, filepath.Join(target, "half", "backend")),
		}},
		{Name: "Solo", Slug: "", RootPath: d("solo")},
		{Name: "SoloConflict", Slug: "solo-conflict", RootPath: d("solo-conf")},
		{Name: "NoRoot", Slug: "noroot", RootPath: "-"},
		{Name: "In Place", Slug: "inplace", Components: []models.Component{
			comp("be", models.RoleBackend, filepath.Join(target, "inplace", "backend")),
		}},
	}
	for i := range projects {
		if err := db.Create(&projects[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	var out []any
	for id := uint(1); id <= 7; id++ {
		pv, err := services.Organizer.PreviewConsolidate(ctx, id, target)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, pv)
	}
	pv, err := services.Organizer.PreviewConsolidate(ctx, 6, "")
	if err != nil {
		t.Fatal(err)
	}
	out = append(out, pv)
	_, err = services.Organizer.PreviewConsolidate(ctx, 999, target)
	out = append(out, err.Error())
	golden(t, "consolidate_preview", out, base)
}

func TestExecuteMovesGolden(t *testing.T) {
	services, db := setupTestServices(t)
	ctx := context.Background()
	base := t.TempDir()
	d := func(n string) string { return filepath.Join(base, n) }
	mkdirs(t, d("s1"), d("s2"), d("s3"), d("s4"), d("exists"), d("s5"))
	writeFiles(t, base, map[string]string{"blockfile": "x"})

	p := models.Project{Name: "P", Slug: "p", RootPath: d("s1"), Components: []models.Component{
		{Name: "c1", Role: models.RoleBackend, AbsolutePath: d("s1")},
		{Name: "c2", Role: models.RoleBackend, AbsolutePath: d("s2")},
	}}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	moves := []models.FolderMoveItem{
		{ID: "1", Status: "skipped", CurrentPath: d("s1"), TargetPath: d("t1")},
		{ID: "2", CurrentPath: d("nope"), TargetPath: d("t2"), Status: "pending"},
		{ID: "3", CurrentPath: d("s2"), TargetPath: d("exists"), Status: "pending"},
		{ID: "4", CurrentPath: d("s3"), TargetPath: filepath.Join(d("blockfile"), "x", "t"), Status: "pending"},
		{ID: "5", CurrentPath: d("s1"), TargetPath: d("out/t5"), Status: "pending", ProjectID: p.ID, ComponentID: p.Components[0].ID},
		{ID: "6", CurrentPath: d("s4"), TargetPath: d("out/t6"), Status: "pending", ProjectID: 999, ComponentID: 999},
		{ID: "7", CurrentPath: d("s5"), TargetPath: d("out/t7"), Status: "pending"},
	}
	res, err := services.Organizer.ExecuteMoves(ctx, models.ExecuteMovePayload{Moves: moves, UpdateDB: true})
	if err != nil {
		t.Fatal(err)
	}
	var gotP models.Project
	db.Preload("Components").First(&gotP, p.ID)
	golden(t, "execute_moves", []any{res, gotP.RootPath, gotP.Components[0].AbsolutePath, gotP.Components[0].RelativePath, gotP.Components[1].AbsolutePath}, base)

	// tanpa UpdateDB
	mkdirs(t, d("u1"))
	res2, _ := services.Organizer.ExecuteMoves(ctx, models.ExecuteMovePayload{Moves: []models.FolderMoveItem{
		{ID: "a", CurrentPath: d("u1"), TargetPath: d("o2/u1"), ProjectID: p.ID, ComponentID: p.Components[1].ID},
	}})
	golden(t, "execute_moves_nodb", res2, base)
}
