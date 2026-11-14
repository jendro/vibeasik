package models

import (
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ProjectType defines the architectural topology of the project
const (
	TypeBundle       = "bundle"       // Backend + Frontend
	TypeMultiFE      = "multi_fe"     // Backend + Primary/Admin FE + Client FE
	TypeMonorepo     = "monorepo"     // Monorepo with multiple packages
	TypeSingleApp    = "single_app"   // Single service or standalone frontend
	TypeMicroservice = "microservice" // Distributed microservices
)

// ComponentRole defines the functional role of a component
const (
	RoleBackend        = "backend"
	RoleFrontendAdmin  = "frontend_admin"
	RoleFrontendClient = "frontend_client"
	RoleFrontendMobile = "frontend_mobile"
	RoleCLI            = "cli"
	RoleWorker         = "worker"
	RoleDocs           = "docs"
	RoleOther          = "other"
)

// Project Status constants
const (
	StatusActive     = "active"
	StatusPaused     = "paused"
	StatusArchived   = "archived"
	StatusDeprecated = "deprecated"
	StatusHold       = "hold"
	StatusDone       = "done"
)

// Project represents a managed software project/bundle
type Project struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	Name                  string     `gorm:"size:255;not null" json:"name"`
	Slug                  string     `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	ShortDescription      string     `gorm:"type:text" json:"short_description"`
	Description           string     `gorm:"type:text" json:"description"`
	RootPath              string     `gorm:"size:1024;default:''" json:"root_path"`
	PortSlot              *int       `json:"port_slot,omitempty"`                          // blok port Aturan 35: slot N = 4100+N×10..+9; -1 = vibeasik sendiri (87xx); nil = belum dialokasikan
	ProjectType           string     `gorm:"size:64;default:'bundle'" json:"project_type"` // bundle, multi_fe, monorepo, single_app
	Status                string     `gorm:"size:64;default:'active'" json:"status"`       // active, paused, archived, deprecated
	GitBranch             string     `gorm:"size:128" json:"git_branch"`
	GitRemote             string     `gorm:"size:512" json:"git_remote"`
	Tags                  string     `gorm:"size:512" json:"tags"` // comma-separated or json string
	HealthScore           int        `json:"health_score"`
	IsUnified             bool       `gorm:"default:false" json:"is_unified"`
	Autonomous            bool       `gorm:"default:false" json:"autonomous"`
	DeprecatedAt          *time.Time `json:"deprecated_at,omitempty"`
	DeprecatedReason      string     `gorm:"type:text" json:"deprecated_reason,omitempty"`
	LastDeepCheckAt       *time.Time `json:"last_deep_check_at,omitempty"`
	LastDeepCheckScore    int        `json:"last_deep_check_score,omitempty"`
	LastDeepCheckCommit   string     `gorm:"size:64" json:"last_deep_check_commit,omitempty"`
	LastDeepCheckChecksum string     `gorm:"size:64" json:"last_deep_check_checksum,omitempty"`
	SynergyMapData        string     `gorm:"type:longtext" json:"synergy_map_data,omitempty"`
	ProfileData           string     `gorm:"type:longtext" json:"-"` // JSON ProjectProfile, served via /projects/:id/profile
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`

	Components []Component `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"components"`
	// Diagnostics tidak ikut respons detail project (bisa puluhan MB); gunakan GET /projects/:id/diagnostics.
	Diagnostics []Diagnostic      `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"-"`
	DeepChecks  []DeepCheckReport `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"deep_checks,omitempty"`
}

// ComponentKey: identitas komponen pada temuan — RelativePath bila tercatat, path folder komponen relatif
// terhadap akar proyek ("admin"), atau nama komponen bila foldernya di luar akar.
func (p Project) ComponentKey(c Component) string {
	if c.RelativePath != "" {
		if rel := strings.Trim(filepath.ToSlash(filepath.Clean(c.RelativePath)), "/"); rel != "." {
			return rel
		}
	}
	if p.RootPath != "" && p.RootPath != "-" && c.AbsolutePath != "" {
		if rel, err := filepath.Rel(p.RootPath, c.AbsolutePath); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return c.Name
}

// FindingPaths: kandidat path absolut berkas sebuah temuan, urut dari yang paling mungkin — folder komponen asal
// temuan (component = ComponentKey), akar proyek, folder komponen lain, nama berkas di komponen lain, lalu
// targetFile apa adanya. Tanpa component, komponen pertama yang memuat berkas itu yang menang (bisa keliru bila
// dua komponen punya path relatif yang sama).
func (p Project) FindingPaths(component, targetFile string) []string {
	rel := filepath.FromSlash(targetFile)
	paths := make([]string, 0, 3+len(p.Components)*2)
	if component != "" {
		for _, c := range p.Components {
			if c.AbsolutePath != "" && p.ComponentKey(c) == component {
				paths = append(paths, filepath.Join(c.AbsolutePath, rel))
			}
		}
	}
	if p.RootPath != "" && p.RootPath != "-" {
		paths = append(paths, filepath.Join(p.RootPath, rel))
	}
	for _, c := range p.Components {
		if c.AbsolutePath != "" {
			paths = append(paths, filepath.Join(c.AbsolutePath, rel), filepath.Join(c.AbsolutePath, filepath.Base(rel)))
		}
	}
	return append(paths, targetFile)
}

// Component represents an individual app/service inside a project bundle
type Component struct {
	ID             uint             `gorm:"primaryKey" json:"id"`
	ProjectID      uint             `gorm:"index;not null" json:"project_id"`
	Name           string           `gorm:"size:255;not null" json:"name"`
	Role           string           `gorm:"size:64;not null;default:'other'" json:"role"` // backend, frontend_admin, frontend_client, frontend_mobile, etc.
	RelativePath   string           `gorm:"size:512;default:''" json:"relative_path"`
	AbsolutePath   string           `gorm:"size:1024;default:''" json:"absolute_path"`
	Language       string           `gorm:"size:64;default:'auto'" json:"language"`    // go, typescript, python, dart, php, etc.
	Framework      string           `gorm:"size:64;default:''" json:"framework"`       // fiber, nuxt, next, flutter, fastapi, etc.
	PackageManager string           `gorm:"size:64;default:''" json:"package_manager"` // bun, npm, pnpm, go, pub, pip, etc.
	DevCommand     string           `gorm:"size:255;default:''" json:"dev_command"`
	BuildCommand   string           `gorm:"size:255;default:''" json:"build_command"`
	Port           int              `gorm:"default:0" json:"port"`     // port yang dideklarasikan kode/proyek (hasil scan)
	RunPort        int              `gorm:"default:0" json:"run_port"` // port unik yang dialokasikan vibeasik saat menjalankan komponen
	HealthURL      string           `gorm:"size:255;default:''" json:"health_url"`
	HealthScore    int              `json:"health_score"`
	EnvFile        string           `gorm:"size:255;default:''" json:"env_file"`
	EnvSummary     string           `gorm:"type:longtext" json:"env_summary"`
	FastScanData   string           `gorm:"type:longtext" json:"fast_scan_data,omitempty"`
	DesignTags     string           `gorm:"type:text" json:"-"` // JSON AppDesignTags: tag desain yang dikunci (FE/mobile), via designlib.AppProfile
	Routes         []ComponentRoute `gorm:"foreignKey:ComponentID;constraint:OnDelete:CASCADE" json:"routes,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// ProjectChanged dipanggil setelah proyek atau komponen dibuat (diisi paket standard saat `vibeasik serve`):
// menyiapkan blok port, env lokal/server, dan todo standar Aturan 45. Nil = tidak melakukan apa pun.
var ProjectChanged func(projectID uint)

func (p *Project) AfterCreate(tx *gorm.DB) error {
	if ProjectChanged != nil {
		ProjectChanged(p.ID)
	}
	return nil
}

func (c *Component) AfterCreate(tx *gorm.DB) error {
	if ProjectChanged != nil {
		ProjectChanged(c.ProjectID)
	}
	return nil
}
