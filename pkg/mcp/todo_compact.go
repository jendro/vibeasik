package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/todorag"
)

// Bentuk ringkas list_todos / list_projects: daftar lengkap bisa berukuran MB (keterangan berisi contoh kode,
// rationale berisi semua catatan penyelesaian, fast_scan_data komponen) sehingga agent lambat atau terpotong.
// Versi ringkas memuat memori todo sebagai pengganti riwayat; detail utuh tetap lewat detail: true.

const (
	listTodosDefaultLimit = 50
	listTodosMaxLimit     = 200
	todoPreviewMax        = 280
	listMemoryMax         = 320
)

type compactTodo struct {
	ID               uint       `json:"id"`
	Title            string     `json:"title"`
	Status           string     `json:"status"`
	Priority         string     `json:"priority,omitempty"`
	Category         string     `json:"category,omitempty"`
	Level            string     `json:"level,omitempty"`
	ProjectID        *uint      `json:"project_id,omitempty"`
	ComponentID      *uint      `json:"component_id,omitempty"`
	ComponentName    string     `json:"component_name,omitempty"`
	TargetApps       string     `json:"target_apps,omitempty"`
	Tags             string     `json:"tags,omitempty"`
	Source           string     `json:"source,omitempty"`
	ComplexityLevel  string     `json:"complexity_level,omitempty"`
	ImpactScore      int        `json:"impact_score,omitempty"`
	Score            *int       `json:"score,omitempty"`
	EmployeeName     string     `json:"employee_name,omitempty"`
	CompletedByAgent string     `json:"completed_by_agent,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	Preview          string     `json:"description_preview,omitempty"`
	Memory           string     `json:"memory,omitempty"`
	Similarity       float64    `json:"similarity,omitempty"`
}

func toCompactTodo(t models.Todo) compactTodo {
	return compactTodo{
		ID: t.ID, Title: t.Title, Status: t.Status, Priority: t.Priority, Category: t.Category, Level: t.Level,
		ProjectID: t.ProjectID, ComponentID: t.ComponentID, ComponentName: t.ComponentName, TargetApps: t.TargetApps,
		Tags: t.Tags, Source: t.Source, ComplexityLevel: t.ComplexityLevel, ImpactScore: t.ImpactScore, Score: t.Score,
		EmployeeName: t.EmployeeName, CompletedByAgent: t.CompletedByAgent, CompletedAt: t.CompletedAt, CreatedAt: t.CreatedAt,
		Preview: util.OneLine(todorag.IndexText(t.Description, ""), todoPreviewMax), Memory: t.Memory,
	}
}

type compactComponent struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Role         string `json:"role,omitempty"`
	RelativePath string `json:"relative_path,omitempty"`
	Language     string `json:"language,omitempty"`
	Framework    string `json:"framework,omitempty"`
	Port         int    `json:"port,omitempty"`
}

type compactProject struct {
	ID          uint               `json:"id"`
	Name        string             `json:"name"`
	Slug        string             `json:"slug"`
	Status      string             `json:"status,omitempty"`
	ProjectType string             `json:"project_type,omitempty"`
	RootPath    string             `json:"root_path,omitempty"`
	Summary     string             `json:"short_description,omitempty"`
	Components  []compactComponent `json:"components,omitempty"`
}

func toCompactProject(p models.Project) compactProject {
	out := compactProject{ID: p.ID, Name: p.Name, Slug: p.Slug, Status: p.Status, ProjectType: p.ProjectType, RootPath: p.RootPath,
		Summary: util.OneLine(p.ShortDescription, 240)}
	for _, c := range p.Components {
		out.Components = append(out.Components, compactComponent{ID: c.ID, Name: c.Name, Role: c.Role, RelativePath: c.RelativePath,
			Language: c.Language, Framework: c.Framework, Port: c.Port})
	}
	return out
}

func intArg(args map[string]interface{}, key string) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// duplicateNotice: keterangan todo serupa untuk respons create_todo.
func duplicateNotice(hits []todorag.Hit) string {
	var b strings.Builder
	b.WriteString("Todo serupa sudah ada di proyek ini — pastikan ini bukan pekerjaan ganda:\n")
	for _, h := range hits {
		fmt.Fprintf(&b, "- #%d [%s] %s (kemiripan %.2f)\n", h.ID, h.Status, h.Title, h.Sim)
		if m := util.OneLine(h.Memory, 200); m != "" {
			b.WriteString("  " + m + "\n")
		}
	}
	return b.String()
}
