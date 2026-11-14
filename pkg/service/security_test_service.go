package service

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/repository"
	"github.com/vibeasik/vibeasik/pkg/runner"
	"github.com/vibeasik/vibeasik/pkg/secaudit"
)

// SecurityTestRequest = permintaan uji keamanan proyek (router + server berjalan).
type SecurityTestRequest struct {
	ProjectID   uint   `json:"project_id"`
	ComponentID uint   `json:"component_id,omitempty"` // 0 = semua komponen backend proyek
	BaseURL     string `json:"base_url,omitempty"`     // URL server live; kosong = port lokal komponen
	Live        *bool  `json:"live,omitempty"`         // nil/true = jalankan probe bila ada sasaran
	SkipBurst   bool   `json:"skip_burst,omitempty"`   // lewati uji burst rate limit
	Burst       int    `json:"burst,omitempty"`        // jumlah request burst (maks 60)
	StartServer bool   `json:"start_server,omitempty"` // nyalakan server komponen bila belum berjalan
	KeepRunning bool   `json:"keep_running,omitempty"` // biarkan server yang dinyalakan uji tetap menyala
	BootTimeout int    `json:"boot_timeout_sec,omitempty"`
	SkipAttacks bool   `json:"skip_attacks,omitempty"` // lewati serangan SQLi/XSS/traversal/… ke semua endpoint
	AuthHeader  string `json:"auth_header,omitempty"`  // "Bearer …" / "Cookie: …" — tidak disimpan di hasil uji
}

// SecurityTestService menjalankan dan menyimpan hasil uji keamanan terakhir per proyek.
type SecurityTestService interface {
	Run(ctx context.Context, req SecurityTestRequest) (*secaudit.Report, error)
	Last(projectID uint) (*secaudit.Report, bool)
	// Start memulai uji di latar (nyalakan server → baca kode → uji server → matikan server).
	Start(req SecurityTestRequest) (*SecTestSession, error)
	// Session = snapshot sesi uji terakhir proyek (berjalan atau selesai) sejak engine menyala.
	Session(projectID uint) (*SecTestSession, bool)
	Cancel(projectID uint) error
}

type securityTestService struct {
	projects repository.ProjectRepository
	routes   repository.RouteRepository
	runner   *runner.Service
}

// Hasil terakhir dibagi semua instance layanan (API Studio dan MCP) dalam satu proses engine.
var (
	lastSecTestMu sync.RWMutex
	lastSecTest   = map[uint]*secaudit.Report{}
)

// NewSecurityTestService membuat layanan uji keamanan.
func NewSecurityTestService(projects repository.ProjectRepository, routes repository.RouteRepository, db *gorm.DB) SecurityTestService {
	svc := &securityTestService{projects: projects, routes: routes}
	if db != nil {
		svc.runner = runner.NewService(db)
	}
	return svc
}

var serverFrameworks = map[string]bool{"fiber": true, "gin": true, "echo": true, "chi": true, "express": true, "fastify": true,
	"nestjs": true, "hono": true, "koa": true, "fastapi": true, "flask": true, "django": true, "laravel": true, "slim": true}

// isBackendComponent: komponen yang punya router server.
func isBackendComponent(c models.Component) bool {
	switch c.Role {
	case models.RoleBackend, models.RoleWorker:
		return true
	case models.RoleFrontendMobile, models.RoleDocs, models.RoleCLI:
		return false
	}
	return serverFrameworks[strings.ToLower(c.Framework)]
}

func componentDir(p *models.Project, c models.Component) string {
	if c.AbsolutePath != "" {
		return c.AbsolutePath
	}
	if p.RootPath != "" {
		return filepath.Join(p.RootPath, c.RelativePath)
	}
	return ""
}

// localBaseURL memilih port lokal komponen yang benar-benar menjawab: port jalan dari vibeasik (run_port)
// lalu port yang dideklarasikan kode. Bila tidak ada yang menjawab, port pertama dipakai (probe melapor skipped)
// dan answering = false.
func localBaseURL(ports ...int) (base string, answering bool) {
	var list []int
	for _, p := range ports {
		if p > 0 && !slices.Contains(list, p) {
			list = append(list, p)
		}
	}
	if len(list) == 0 {
		return "", false
	}
	for _, p := range list {
		addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(p))
		if conn, err := net.DialTimeout("tcp", addr, 400*time.Millisecond); err == nil {
			conn.Close()
			return "http://" + addr, true
		}
	}
	return fmt.Sprintf("http://127.0.0.1:%d", list[0]), false
}

// secTestPlan = sasaran uji yang sudah divalidasi: input audit + komponen yang servernya diuji.
type secTestPlan struct {
	project  *models.Project
	in       secaudit.Input
	liveComp *models.Component
}

// prepare memvalidasi permintaan dan menyusun sasaran uji (folder backend, rute, komponen server).
func (s *securityTestService) prepare(ctx context.Context, req SecurityTestRequest) (*secTestPlan, error) {
	if req.ProjectID == 0 {
		return nil, fmt.Errorf("project_id wajib diisi")
	}
	if msg := secaudit.ValidateBaseURL(req.BaseURL); msg != "" {
		return nil, fmt.Errorf("%s", msg)
	}
	p, err := s.projects.FindByID(ctx, req.ProjectID)
	if err != nil || p == nil {
		return nil, fmt.Errorf("proyek %d tidak ditemukan", req.ProjectID)
	}
	var comps []models.Component
	for _, c := range p.Components {
		if req.ComponentID != 0 && c.ID != req.ComponentID {
			continue
		}
		comps = append(comps, c)
	}
	if req.ComponentID != 0 && len(comps) == 0 {
		return nil, fmt.Errorf("komponen %d bukan milik proyek %d", req.ComponentID, req.ProjectID)
	}

	in := secaudit.Input{ProjectID: p.ID, ComponentID: req.ComponentID, Target: p.Name, Root: p.RootPath}
	var liveComp *models.Component
	for i := range comps {
		c := comps[i]
		backend := isBackendComponent(c)
		if !backend && req.ComponentID == 0 {
			continue
		}
		if dir := componentDir(p, c); dir != "" {
			if st, statErr := os.Stat(dir); statErr == nil && st.IsDir() && backend {
				in.Dirs = append(in.Dirs, secaudit.SourceDir{Name: c.Name, Path: dir})
			}
		}
		if rs, rerr := s.routes.FindByComponentID(ctx, c.ID); rerr == nil {
			for _, r := range rs {
				in.Routes = append(in.Routes, secaudit.RouteRef{Method: r.Method, Path: r.Path})
			}
		}
		if liveComp == nil && (backend || req.ComponentID != 0) {
			liveComp = &comps[i]
		}
	}
	if req.ComponentID != 0 && len(comps) == 1 {
		in.Target = p.Name + " · " + comps[0].Name
	}
	if in.Root == "" && len(in.Dirs) == 1 {
		in.Root = in.Dirs[0].Path
	}
	return &secTestPlan{project: p, in: in, liveComp: liveComp}, nil
}

// Run menjalankan uji secara sinkron (MCP, POST lama). Server hanya dinyalakan bila StartServer diminta.
func (s *securityTestService) Run(ctx context.Context, req SecurityTestRequest) (*secaudit.Report, error) {
	plan, err := s.prepare(ctx, req)
	if err != nil {
		return nil, err
	}
	sess := newSecTestSession(plan.project, req)
	s.execute(ctx, sess, plan, req)
	if sess.Report == nil {
		return nil, fmt.Errorf("%s", sess.Error)
	}
	return sess.Report, nil
}

func (s *securityTestService) Last(projectID uint) (*secaudit.Report, bool) {
	lastSecTestMu.RLock()
	defer lastSecTestMu.RUnlock()
	r, ok := lastSecTest[projectID]
	return r, ok
}
