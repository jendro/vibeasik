package workspace

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/codemap"
)

// InitResult merangkum "Init vibeasik" sebuah proyek: tautan AI (.vibeasik.json) + peta kode di repo.
type InitResult struct {
	ProjectID        uint                `json:"project_id"`
	Root             string              `json:"root"`
	FilesCreated     []string            `json:"files_created"`
	Identifier       *ProjectIdentifier  `json:"identifier,omitempty"`
	CodeMap          *codemap.SyncResult `json:"code_map"`
	CodeMapMarkdown  string              `json:"code_map_markdown"`
	GitignoreUpdated bool                `json:"gitignore_updated"`
	AgentRules       []string            `json:"agent_rules,omitempty"` // GEMINI.md/AGENTS.md/CLAUDE.md berblok "pengetahuan vibeasik dulu"
	AutoSyncMinutes  int                 `json:"auto_sync_minutes"`
}

// InitProject menyiapkan proyek untuk dirawat AI agent: membangun & menulis peta kode ke
// <root>/.vibeasik/ (lalu dijaga mutakhir oleh auto-sync), memastikan .vibeasik/ di-gitignore, dan
// menulis .vibeasik.json (identitas proyek + petunjuk peta) ke root dan setiap komponen, lalu blok protokol
// "pengetahuan vibeasik dulu" di GEMINI.md/AGENTS.md/CLAUDE.md root (Aturan 47). Aman diulang.
func InitProject(db *gorm.DB, projectID uint, apiBase string) (*InitResult, error) {
	var res InitResult
	res.ProjectID = projectID
	sync, err := codemap.Sync(db, projectID, true)
	if err != nil {
		return nil, fmt.Errorf("gagal membangun peta kode: %w", err)
	}
	res.CodeMap = sync
	res.CodeMapMarkdown = codemap.ExportedMarkdown(db, projectID)
	res.Root = codemap.RootOf(db, projectID)
	if res.Root != "" {
		res.GitignoreUpdated, _ = codemap.EnsureGitignore(res.Root)
	}
	files, idf, err := LinkProjectWorkspaces(db, projectID, 0, apiBase)
	if err != nil {
		return nil, err
	}
	res.FilesCreated, res.Identifier = files, idf
	if res.Root != "" && idf != nil {
		if res.AgentRules, err = WriteAgentRules(res.Root, projectID, idf.ProjectName); err != nil {
			return nil, err
		}
	}
	res.AutoSyncMinutes = int(codemap.AutoSyncInterval.Minutes())
	return &res, nil
}
