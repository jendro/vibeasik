package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
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
}

// SecurityTestService menjalankan dan menyimpan hasil uji keamanan terakhir per proyek.
type SecurityTestService interface {
	Run(ctx context.Context, req SecurityTestRequest) (*secaudit.Report, error)
	Last(projectID uint) (*secaudit.Report, bool)
}

type securityTestService struct {
	projects repository.ProjectRepository
	routes   repository.RouteRepository
}

// Hasil terakhir dibagi semua instance layanan (API Studio dan MCP) dalam satu proses engine.
var (
	lastSecTestMu sync.RWMutex
	lastSecTest   = map[uint]*secaudit.Report{}
)

// NewSecurityTestService membuat layanan uji keamanan.
func NewSecurityTestService(projects repository.ProjectRepository, routes repository.RouteRepository) SecurityTestService {
	return &securityTestService{projects: projects, routes: routes}
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

func localBaseURL(c models.Component) string {
	port := c.RunPort
	if port == 0 {
		port = c.Port
	}
	if port == 0 {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

func (s *securityTestService) Run(ctx context.Context, req SecurityTestRequest) (*secaudit.Report, error) {
	if req.ProjectID == 0 {
		return nil, fmt.Errorf("project_id wajib diisi")
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

	wantLive := req.Live == nil || *req.Live
	base := strings.TrimSpace(req.BaseURL)
	if base == "" && liveComp != nil {
		base = localBaseURL(*liveComp)
	}
	if wantLive && base != "" {
		in.Live = &secaudit.LiveOptions{BaseURL: base, Burst: req.Burst, SkipBurst: req.SkipBurst, Timeout: 5 * time.Second}
	}

	rep := secaudit.Run(ctx, in)
	if wantLive && base == "" {
		rep.Notes = append(rep.Notes, "Tidak ada port lokal maupun URL live untuk diuji; isi base_url untuk membuktikan kontrol di server.")
	}
	lastSecTestMu.Lock()
	lastSecTest[p.ID] = rep
	lastSecTestMu.Unlock()
	return rep, nil
}

func (s *securityTestService) Last(projectID uint) (*secaudit.Report, bool) {
	lastSecTestMu.RLock()
	defer lastSecTestMu.RUnlock()
	r, ok := lastSecTest[projectID]
	return r, ok
}
