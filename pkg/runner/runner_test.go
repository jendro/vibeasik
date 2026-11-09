package runner

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	orig := RegistryPath
	RegistryPath = func() string { return "" } // jangan pernah menulis PORTS.md asli
	t.Cleanup(func() { RegistryPath = orig })
	return db
}

func TestRewritePorts(t *testing.T) {
	cases := map[string]string{
		"go run main.go serve --port 8765":        "go run main.go serve --port 4100",
		"nuxt dev --port=3000":                    "nuxt dev --port=4100",
		"next dev -p 3001":                        "next dev -p 4100",
		"PORT=3000 node server.js":                "PORT=4100 node server.js",
		"python manage.py runserver 0.0.0.0:8000": "python manage.py runserver 0.0.0.0:4100",
		"docker compose up -p 8080 --port 9000":   "docker compose up -p 8080 --port 4100",
	}
	for in, want := range cases {
		got, changed := rewritePorts(in, "4100")
		if !changed || got != want {
			t.Errorf("rewritePorts(%q) = %q, want %q", in, got, want)
		}
	}
	if _, changed := rewritePorts("go run .", "4100"); changed {
		t.Error("command without port must stay unchanged")
	}
}

func TestBuildLaunchPackageScripts(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"scripts":{"dev":"vite","dev:fixed":"nuxt dev --port 3000"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}

	spec := BuildLaunch(models.Component{AbsolutePath: dir, DevCommand: "npm run dev"}, 4101, nil)
	if spec.Command != "npm run dev -- --port 4101 --strictPort" {
		t.Errorf("npm vite: got %q", spec.Command)
	}
	spec = BuildLaunch(models.Component{AbsolutePath: dir, DevCommand: "bun run dev"}, 4101, nil)
	if spec.Command != "bun run dev --port 4101 --strictPort" {
		t.Errorf("bun vite: got %q", spec.Command)
	}
	spec = BuildLaunch(models.Component{AbsolutePath: dir, DevCommand: "bun run dev:fixed"}, 4102, nil)
	if spec.Command != "nuxt dev --port 4102" {
		t.Errorf("hardcoded script should be inlined, got %q", spec.Command)
	}

	be := models.Component{ID: 2, Name: "Core API", Role: models.RoleBackend, RunPort: 4200}
	spec = BuildLaunch(models.Component{ID: 1, AbsolutePath: dir, DevCommand: "go run ."}, 4103, []models.Component{be})
	env := strings.Join(spec.Env, " ")
	for _, want := range []string{"PORT=4103", "VIBEASIK_PORT_CORE_API=4200", "VIBEASIK_BACKEND_URL=http://localhost:4200"} {
		if !strings.Contains(env, want) {
			t.Errorf("env missing %s: %s", want, env)
		}
	}
}

func TestAllocateUsesProjectSlot(t *testing.T) {
	db := setupDB(t)
	db.Create(&models.Project{ID: 1, Name: "A", Slug: "a"})
	db.Create(&models.Project{ID: 2, Name: "B", Slug: "b"})
	a := models.Component{ProjectID: 1, Name: "a", Role: models.RoleFrontendClient, Port: 3000}
	b := models.Component{ProjectID: 2, Name: "b", Role: models.RoleFrontendClient, Port: 3000}
	db.Create(&a)
	db.Create(&b)

	pa, err := Allocate(db, &a, false)
	if err != nil || pa != slotBase(0) {
		t.Fatalf("first project must get slot 0 port %d, got %d (%v)", slotBase(0), pa, err)
	}
	pb, err := Allocate(db, &b, false)
	if err != nil || pb != slotBase(1) {
		t.Fatalf("second project must get slot 1 port %d, got %d (%v)", slotBase(1), pb, err)
	}
	again, _ := Allocate(db, &b, false)
	if again != pb {
		t.Fatalf("allocation must be stable: %d then %d", pb, again)
	}
	if err := Assign(db, &b, pa); err == nil {
		t.Fatal("assigning a port claimed by another component must fail")
	}
	if err := Assign(db, &b, 8765); err == nil {
		t.Fatal("assigning a vibeasik-reserved port must fail")
	}
}

// freeSlot mencari slot yang seluruh portnya bebas di mesin ini.
func freeSlot(t *testing.T) int {
	t.Helper()
	for s := MaxSlot; s >= 0; s-- {
		ok := !SlotForbidden(s)
		for p := slotBase(s); ok && p < slotBase(s)+SlotSize; p++ {
			ok = IsPortFree(p)
		}
		if ok {
			return s
		}
	}
	t.Skip("no free slot")
	return 0
}

func TestRunStopLifecycle(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	db := setupDB(t)
	slot := freeSlot(t)
	busyPort := slotBase(slot)
	busy, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", busyPort))
	if err != nil {
		t.Skip("cannot listen on slot port")
	}
	defer busy.Close()
	db.Create(&models.Project{ID: 1, Name: "Web", Slug: "web", PortSlot: &slot})

	comp := models.Component{ProjectID: 1, Role: models.RoleFrontendClient, Name: "web", AbsolutePath: t.TempDir(), Port: busyPort, RunPort: busyPort,
		DevCommand: `python3 -m http.server "$PORT" --bind 127.0.0.1`}
	db.Create(&comp)
	svc := NewService(db)

	var busyErr *PortBusyError
	if _, err := svc.Run(comp.ID, false); !errors.As(err, &busyErr) {
		t.Fatalf("expected PortBusyError, got %v", err)
	}
	st, err := svc.Run(comp.ID, true)
	if err != nil {
		t.Fatalf("run with reassign: %v", err)
	}
	if st.Port == busyPort || st.Port < busyPort || st.Port >= busyPort+SlotSize {
		t.Fatalf("reassign must move to another port inside the project block, got %d", st.Port)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		cur, _ := svc.Status(comp.ID, true)
		if cur.State == StateRunning && cur.Listening {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("component never became running: %+v", cur)
		}
		time.Sleep(200 * time.Millisecond)
	}

	stopped, err := svc.Stop(comp.ID)
	if err != nil || stopped.State != StateStopped {
		t.Fatalf("stop: %+v %v", stopped, err)
	}
	if IsListening(st.Port) {
		t.Fatal("port still listening after stop")
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	for p := 20000; p < 21000; p++ {
		if IsPortFree(p) {
			return p
		}
	}
	t.Fatal("no free port")
	return 0
}

func TestEnsureUniquePorts(t *testing.T) {
	db := setupDB(t)
	db.Create(&models.Project{ID: 10, Name: "P", Slug: "p"})
	// Dua komponen dalam proyek yang sama memiliki port yang sama (3000)
	c1 := models.Component{ProjectID: 10, Name: "Frontend 1", Role: models.RoleFrontendClient, Port: 3000, RunPort: 0}
	c2 := models.Component{ProjectID: 10, Name: "Frontend 2", Role: models.RoleFrontendClient, Port: 3000, RunPort: 0}
	db.Create(&c1)
	db.Create(&c2)

	updated, err := EnsureUniquePorts(db, 10)
	if err != nil {
		t.Fatalf("EnsureUniquePorts error: %v", err)
	}
	if len(updated) != 2 {
		t.Fatalf("expected 2 updated components, got %d", len(updated))
	}

	var res1, res2 models.Component
	db.First(&res1, c1.ID)
	db.First(&res2, c2.ID)

	if res1.RunPort == 0 || res2.RunPort == 0 {
		t.Fatalf("run_port should be > 0: res1=%d, res2=%d", res1.RunPort, res2.RunPort)
	}
	if res1.RunPort == res2.RunPort {
		t.Fatalf("run_ports must be unique: res1=%d, res2=%d", res1.RunPort, res2.RunPort)
	}
	if res1.RunPort != slotBase(0) || res2.RunPort != slotBase(0)+1 {
		t.Fatalf("frontends must take +0/+1 of slot 0: %d %d", res1.RunPort, res2.RunPort)
	}
}

func TestPlanSlotPorts(t *testing.T) {
	comps := []models.Component{
		{ID: 5, Role: models.RoleBackend},
		{ID: 3, Role: models.RoleFrontendAdmin},
		{ID: 4, Role: models.RoleFrontendClient},
		{ID: 6, Role: models.RoleWorker},
	}
	plan := PlanSlotPorts(2, comps, map[uint]int{6: 4127, 3: 3000})
	want := map[uint]int{4: 4120, 3: 4121, 5: 4125, 6: 4127}
	for id, p := range want {
		if plan[id] != p {
			t.Errorf("component %d: want %d got %d (%v)", id, p, plan[id], plan)
		}
	}
	if !SlotForbidden(7) || SlotForbidden(8) {
		t.Error("slot 7 (4173) forbidden, slot 8 allowed")
	}
}

func TestEnsureAllMovesStalePortHolder(t *testing.T) {
	db := setupDB(t)
	db.Create(&models.Project{ID: 1, Name: "A", Slug: "a"})
	db.Create(&models.Project{ID: 2, Name: "B", Slug: "b"})
	// Komponen proyek B memegang 4100 (sisa alokasi lama) padahal 4100 milik blok proyek A
	b := models.Component{ProjectID: 2, Name: "b", Role: models.RoleBackend, RunPort: 4100}
	a := models.Component{ProjectID: 1, Name: "a", Role: models.RoleFrontendClient}
	db.Create(&b)
	db.Create(&a)
	if _, err := EnsureAllProjectPorts(db); err != nil {
		t.Fatal(err)
	}
	db.First(&a, a.ID)
	db.First(&b, b.ID)
	if a.RunPort != 4100 || b.RunPort != 4115 {
		t.Fatalf("want a=4100 b=4115, got a=%d b=%d", a.RunPort, b.RunPort)
	}
}

func TestLoadComponentAndStatus_OptimizedProjection(t *testing.T) {
	db := setupDB(t)
	p := models.Project{ID: 10, Name: "Proj10", Slug: "proj10"}
	db.Create(&p)

	c1 := models.Component{
		ID:           1001,
		ProjectID:    10,
		Name:         "frontend",
		Role:         models.RoleFrontendClient,
		DevCommand:   "npm run dev",
		AbsolutePath: "/tmp/frontend",
		Port:         3000,
		RunPort:      4120,
	}
	c2 := models.Component{
		ID:           1002,
		ProjectID:    10,
		Name:         "backend",
		Role:         models.RoleBackend,
		DevCommand:   "go run .",
		AbsolutePath: "/tmp/backend",
		Port:         8080,
		RunPort:      4125,
	}
	db.Create(&c1)
	db.Create(&c2)

	svc := NewService(db)

	// 1. Test loadComponentByID
	comp, err := svc.loadComponentByID(c1.ID)
	if err != nil {
		t.Fatalf("loadComponentByID failed: %v", err)
	}
	if comp.ID != c1.ID || comp.Name != "frontend" || comp.DevCommand != "npm run dev" || comp.AbsolutePath != "/tmp/frontend" || comp.RunPort != 4120 {
		t.Errorf("loadComponentByID fields incomplete: %+v", comp)
	}

	// 2. Test loadComponent (with siblings)
	compLoaded, siblings, err := svc.loadComponent(c1.ID)
	if err != nil {
		t.Fatalf("loadComponent failed: %v", err)
	}
	if compLoaded.ID != c1.ID {
		t.Errorf("loadComponent primary mismatch: %+v", compLoaded)
	}
	if len(siblings) != 2 {
		t.Fatalf("expected 2 siblings for project 10, got %d", len(siblings))
	}
	for _, s := range siblings {
		if s.RunPort == 0 || s.DevCommand == "" || s.Role == "" {
			t.Errorf("sibling projection missing essential fields: %+v", s)
		}
	}

	// 3. Test Status uses loadComponentByID (idle status)
	st, err := svc.Status(c1.ID, false)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if st.ComponentID != c1.ID || st.State != StateIdle || st.Port != 4120 {
		t.Errorf("Status mismatch: %+v", st)
	}

	// 4. Test nonexistent component returns error
	_, err = svc.loadComponentByID(99999)
	if err == nil {
		t.Errorf("expected error for nonexistent component")
	}
}

func BenchmarkLoadComponent_Projected(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	_ = db.AutoMigrate(&models.Project{}, &models.Component{})
	c := models.Component{
		ID:           1,
		ProjectID:    1,
		Name:         "frontend",
		Role:         models.RoleFrontendClient,
		DevCommand:   "npm run dev",
		AbsolutePath: "/tmp/fe",
		Port:         3000,
		RunPort:      4120,
	}
	db.Create(&c)
	for i := 2; i <= 10; i++ {
		db.Create(&models.Component{
			ID:         uint(i),
			ProjectID:  1,
			Name:       fmt.Sprintf("comp-%d", i),
			Role:       models.RoleBackend,
			DevCommand: "go run .",
			Port:       8000 + i,
			RunPort:    4200 + i,
		})
	}
	svc := NewService(db)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, err := svc.loadComponent(1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

