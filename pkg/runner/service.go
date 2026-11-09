package runner

import (
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// PortBusyError means the component's stable run port is held by a process vibeasik does not own.
type PortBusyError struct {
	Port      int
	Component string
}

func (e *PortBusyError) Error() string {
	return fmt.Sprintf("port :%d untuk '%s' sedang dipakai proses lain di luar vibeasik", e.Port, e.Component)
}

// RegistryEntry is one row of the local port registry (declared vs allocated port, live state).
type RegistryEntry struct {
	ComponentID   uint   `json:"component_id"`
	ProjectID     uint   `json:"project_id"`
	ProjectName   string `json:"project_name"`
	ComponentName string `json:"component_name"`
	Role          string `json:"role"`
	DeclaredPort  int    `json:"declared_port"`
	RunPort       int    `json:"run_port"`
	State         State  `json:"state"`
	Listening     bool   `json:"listening"`
	HasDevCommand bool   `json:"has_dev_command"`
}

// ProjectRunItem reports the outcome for one component of a project-wide run/stop.
type ProjectRunItem struct {
	ComponentID uint    `json:"component_id"`
	Name        string  `json:"name"`
	Status      *Status `json:"status,omitempty"`
	Error       string  `json:"error,omitempty"`
	Skipped     string  `json:"skipped,omitempty"`
}

// Service binds the port registry (DB) to the process manager.
type Service struct {
	db  *gorm.DB
	mgr *Manager
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db, mgr: Default()}
}

// nonRunnableRoles are skipped by project-wide runs (need a device, or are not servers).
var nonRunnableRoles = map[string]bool{"frontend_mobile": true, "docs": true, "cli": true}

func (s *Service) loadComponentByID(id uint) (models.Component, error) {
	var comp models.Component
	if err := s.db.
		Select("id, project_id, name, dev_command, role, absolute_path, port, run_port").
		First(&comp, id).Error; err != nil {
		return comp, fmt.Errorf("komponen tidak ditemukan")
	}
	return comp, nil
}

func (s *Service) loadComponent(id uint) (models.Component, []models.Component, error) {
	comp, err := s.loadComponentByID(id)
	if err != nil {
		return comp, nil, err
	}

	var siblings []models.Component
	if err := s.db.
		Select("id, project_id, name, dev_command, role, port, run_port").
		Where("project_id = ?", comp.ProjectID).
		Find(&siblings).Error; err != nil {
		return comp, nil, fmt.Errorf("gagal memuat komponen terkait: %w", err)
	}
	return comp, siblings, nil
}

// Run allocates (or reuses) the component's port and launches its dev command. With reassign=true a
// busy port is swapped for a free one instead of returning *PortBusyError.
func (s *Service) Run(id uint, reassign bool) (*Status, error) {
	comp, siblings, err := s.loadComponent(id)
	if err != nil {
		return nil, err
	}
	if st, ok := s.mgr.Status(id, false); ok && (st.State == StateStarting || st.State == StateRunning) {
		return &st, nil
	}

	port, err := Allocate(s.db, &comp, false)
	if err != nil {
		return nil, err
	}
	if !IsPortFree(port) {
		if !reassign {
			return nil, &PortBusyError{Port: port, Component: comp.Name}
		}
		if port, err = Allocate(s.db, &comp, true); err != nil {
			return nil, err
		}
	}

	// Siapkan port saudara agar frontend langsung tahu alamat backend-nya (VIBEASIK_BACKEND_URL).
	for i := range siblings {
		sib := &siblings[i]
		if sib.ID == comp.ID {
			sib.RunPort = comp.RunPort
			continue
		}
		if sib.RunPort == 0 && sib.DevCommand != "" && !nonRunnableRoles[sib.Role] {
			_, _ = Allocate(s.db, sib, false)
		}
	}

	return s.mgr.Start(comp, port, BuildLaunch(comp, port, siblings))
}

// Stop terminates a running component.
func (s *Service) Stop(id uint) (*Status, error) {
	return s.mgr.Stop(id)
}

// Status returns the live process status, or an idle status carrying the allocated port.
func (s *Service) Status(id uint, withLogs bool) (Status, error) {
	if st, ok := s.mgr.Status(id, withLogs); ok {
		return st, nil
	}
	comp, err := s.loadComponentByID(id)
	if err != nil {
		return Status{}, err
	}
	st := Status{ComponentID: comp.ID, ProjectID: comp.ProjectID, Name: comp.Name, State: StateIdle, Port: comp.RunPort}
	if comp.RunPort > 0 {
		st.URL = fmt.Sprintf("http://localhost:%d", comp.RunPort)
		st.Listening = IsListening(comp.RunPort)
	}
	return st, nil
}

// List returns every process vibeasik launched in this engine instance.
func (s *Service) List() []Status {
	return s.mgr.List()
}

// AssignPort sets port manually (port>0) or lets the allocator pick (reassign forces a new one).
func (s *Service) AssignPort(id uint, port int, reassign bool) (*models.Component, error) {
	comp, err := s.loadComponentByID(id)
	if err != nil {
		return nil, err
	}
	if st, ok := s.mgr.Status(id, false); ok && (st.State == StateStarting || st.State == StateRunning) {
		return nil, errors.New("hentikan komponen dulu sebelum mengganti port")
	}
	if port > 0 {
		err = Assign(s.db, &comp, port)
	} else {
		_, err = Allocate(s.db, &comp, reassign)
	}
	if err != nil {
		return nil, err
	}
	return &comp, nil
}

// RunProject starts every runnable component of a project, backends first.
func (s *Service) RunProject(projectID uint, reassign bool) ([]ProjectRunItem, error) {
	comps, err := s.projectComponents(projectID)
	if err != nil {
		return nil, err
	}
	items := make([]ProjectRunItem, 0, len(comps))
	for _, c := range comps {
		item := ProjectRunItem{ComponentID: c.ID, Name: c.Name}
		switch {
		case c.DevCommand == "":
			item.Skipped = "tanpa dev command"
		case nonRunnableRoles[c.Role]:
			item.Skipped = "role " + c.Role + " tidak dijalankan otomatis"
		default:
			st, err := s.Run(c.ID, reassign)
			item.Status = st
			if err != nil {
				item.Error = err.Error()
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// StopProject stops every running component of a project.
func (s *Service) StopProject(projectID uint) ([]ProjectRunItem, error) {
	comps, err := s.projectComponents(projectID)
	if err != nil {
		return nil, err
	}
	items := make([]ProjectRunItem, 0)
	for _, c := range comps {
		st, ok := s.mgr.Status(c.ID, false)
		if !ok || (st.State != StateStarting && st.State != StateRunning) {
			continue
		}
		item := ProjectRunItem{ComponentID: c.ID, Name: c.Name}
		stopped, err := s.mgr.Stop(c.ID)
		item.Status = stopped
		if err != nil {
			item.Error = err.Error()
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) projectComponents(projectID uint) ([]models.Component, error) {
	var comps []models.Component
	if err := s.db.Where("project_id = ?", projectID).Find(&comps).Error; err != nil {
		return nil, err
	}
	if len(comps) == 0 {
		return nil, fmt.Errorf("proyek tidak ditemukan atau belum punya komponen")
	}
	sort.SliceStable(comps, func(i, j int) bool {
		return comps[i].Role == models.RoleBackend && comps[j].Role != models.RoleBackend
	})
	return comps, nil
}

// Registry lists every component with a declared or allocated port, plus its live state.
func (s *Service) Registry() ([]RegistryEntry, error) {
	var projects []models.Project
	if err := s.db.Preload("Components").Find(&projects).Error; err != nil {
		return nil, err
	}
	entries := make([]RegistryEntry, 0)
	for _, p := range projects {
		for _, c := range p.Components {
			if c.Port <= 0 && c.RunPort <= 0 {
				continue
			}
			e := RegistryEntry{
				ComponentID:   c.ID,
				ProjectID:     p.ID,
				ProjectName:   p.Name,
				ComponentName: c.Name,
				Role:          c.Role,
				DeclaredPort:  c.Port,
				RunPort:       c.RunPort,
				State:         StateIdle,
				HasDevCommand: c.DevCommand != "",
			}
			if st, ok := s.mgr.Status(c.ID, false); ok {
				e.State = st.State
			}
			if c.RunPort > 0 {
				e.Listening = IsListening(c.RunPort)
			}
			entries = append(entries, e)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		pi, pj := entries[i].RunPort, entries[j].RunPort
		if pi == 0 {
			pi = entries[i].DeclaredPort
		}
		if pj == 0 {
			pj = entries[j].DeclaredPort
		}
		return pi < pj
	})
	return entries, nil
}

// EnsureUniquePorts mengalokasikan port unik untuk seluruh komponen proyek (atau seluruh sistem bila projectID == 0).
func (s *Service) EnsureUniquePorts(projectID uint) ([]models.Component, error) {
	return EnsureUniquePorts(s.db, projectID)
}
