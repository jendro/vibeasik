package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/codemap"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// IdentifierFileName is the standard file name for workspace identifiers
const IdentifierFileName = ".vibeasik.json"

// SiblingComponentInfo holds details of peer components in a bundle
type SiblingComponentInfo struct {
	ComponentID  uint   `json:"component_id"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	Language     string `json:"language"`
	Framework    string `json:"framework,omitempty"`
	Port         int    `json:"port,omitempty"`
	AbsolutePath string `json:"absolute_path"`
}

// VibeasikEngineInfo holds local engine connection details
type VibeasikEngineInfo struct {
	APIBase       string `json:"api_base"`
	MCPServer     string `json:"mcp_server"`
	TodosEndpoint string `json:"todos_endpoint"`
}

// CodeMapInfo menunjukkan peta kode proyek (gaya RAG) agar agent menemukan lokasi kode tanpa menjelajah repo.
type CodeMapInfo struct {
	Markdown string   `json:"markdown"`
	JSONL    string   `json:"jsonl"`
	MCPTools []string `json:"mcp_tools"`
	AutoSync string   `json:"auto_sync"`
}

// ProjectIdentifier is the metadata schema written into .vibeasik.json
type ProjectIdentifier struct {
	SchemaVersion       string                 `json:"$schema_version"`
	ProjectID           uint                   `json:"project_id"`
	ProjectName         string                 `json:"project_name"`
	ProjectSlug         string                 `json:"project_slug"`
	ProjectType         string                 `json:"project_type"`
	HealthScore         int                    `json:"health_score"`
	ComponentID         uint                   `json:"component_id,omitempty"`
	ComponentName       string                 `json:"component_name,omitempty"`
	Role                string                 `json:"role,omitempty"`
	Language            string                 `json:"language,omitempty"`
	Framework           string                 `json:"framework,omitempty"`
	Port                int                    `json:"port,omitempty"`
	AbsolutePath        string                 `json:"absolute_path,omitempty"`
	SiblingComponents   []SiblingComponentInfo `json:"sibling_components,omitempty"`
	Vibeasik            VibeasikEngineInfo     `json:"vibeasik"`
	CodeMap             *CodeMapInfo           `json:"code_map,omitempty"`
	AIAgentInstructions string                 `json:"ai_agent_instructions"`
	LinkedAt            string                 `json:"linked_at"`
}

// WorkspaceMatchResult holds identification result including active todos
type WorkspaceMatchResult struct {
	Matched             bool               `json:"matched"`
	MatchSource         string             `json:"match_source"` // "file" or "database"
	IdentifierFilePath  string             `json:"identifier_file_path,omitempty"`
	Project             *models.Project    `json:"project,omitempty"`
	CurrentComponent    *models.Component  `json:"current_component,omitempty"`
	Identifier          *ProjectIdentifier `json:"identifier,omitempty"`
	ActiveTodos         []models.Todo      `json:"active_todos,omitempty"`
	TodosCount          int                `json:"todos_count"`
	PendingTodosCount   int                `json:"pending_todos_count"`
	SuggestedTodosCount int                `json:"suggested_todos_count"`
	AIAgentInstructions string             `json:"ai_agent_instructions"`
}

// BuildIdentifier builds a ProjectIdentifier struct for a given project and component
func BuildIdentifier(p *models.Project, comp *models.Component, apiBase string) ProjectIdentifier {
	if apiBase == "" {
		apiBase = "http://127.0.0.1:8765"
	}
	apiBase = strings.TrimRight(apiBase, "/")

	var siblings []SiblingComponentInfo
	for _, c := range p.Components {
		if comp != nil && c.ID == comp.ID {
			continue
		}
		siblings = append(siblings, SiblingComponentInfo{
			ComponentID:  c.ID,
			Name:         c.Name,
			Role:         c.Role,
			Language:     c.Language,
			Framework:    c.Framework,
			Port:         runPort(c),
			AbsolutePath: c.AbsolutePath,
		})
	}

	compName := ""
	var compID uint
	compRole := ""
	compLang := ""
	compFmt := ""
	compPort := 0
	compPath := ""

	if comp != nil {
		compName = comp.Name
		compID = comp.ID
		compRole = comp.Role
		compLang = comp.Language
		compFmt = comp.Framework
		compPort = runPort(*comp)
		compPath = comp.AbsolutePath
	}

	instructions := fmt.Sprintf(
		"Proyek ini tersambung ke Vibeasik Local Engine (Project ID: %d, Nama: '%s'). "+
			"Gunakan MCP tool 'list_todos' dengan parameter 'project_id': %d untuk melihat daftar tugas. "+
			"Ikuti alur kerja todo di bawah dan perbarui status lewat MCP tool 'update_todo'.",
		p.ID, p.Name, p.ID,
	)
	if comp != nil {
		instructions = fmt.Sprintf(
			"Proyek ini tersambung ke Vibeasik Local Engine (Project ID: %d, Nama: '%s', Komponen ID: %d, Komponen: '%s' [%s]). "+
				"Gunakan MCP tool 'list_todos' dengan parameter 'project_id': %d dan 'component_id': %d untuk melihat daftar tugas komponen ini. "+
				"Ikuti alur kerja todo di bawah dan perbarui status lewat MCP tool 'update_todo'.",
			p.ID, p.Name, comp.ID, comp.Name, comp.Role, p.ID, comp.ID,
		)
	}
	instructions += " " + KnowledgeFirst + " " + TodoWorkflow

	var codeMap *CodeMapInfo
	if root := codemap.EffectiveRoot(*p); root != "" {
		md := codemap.MarkdownPath(root)
		codeMap = &CodeMapInfo{
			Markdown: md, JSONL: filepath.Join(filepath.Dir(md), "codemap.jsonl"),
			MCPTools: []string{"vibeasik_code_map_search", "vibeasik_code_map_file", "vibeasik_code_map_overview", "vibeasik_code_map_routes"},
			AutoSync: "Diperbarui otomatis oleh vibeasik serve tiap 2 menit saat ada berkas berubah",
		}
		instructions += fmt.Sprintf(" Sebelum mencari atau mengubah kode, pakai peta kode: MCP 'vibeasik_code_map_search' / 'vibeasik_code_map_file' (project_id: %d; cek 'used_by' sebelum mengedit berkas) atau baca %s. Untuk menilai/mengubah rute API (auth, validasi, response), cek rantai middleware efektif dari main lewat 'vibeasik_code_map_routes' atau .vibeasik/routes.jsonl — middleware global/group tidak terlihat di baris rute; jangan memasang ulang middleware yang sudah aktif.", p.ID, md)
	}

	return ProjectIdentifier{
		SchemaVersion:     "1.0",
		ProjectID:         p.ID,
		ProjectName:       p.Name,
		ProjectSlug:       p.Slug,
		ProjectType:       p.ProjectType,
		HealthScore:       p.HealthScore,
		ComponentID:       compID,
		ComponentName:     compName,
		Role:              compRole,
		Language:          compLang,
		Framework:         compFmt,
		Port:              compPort,
		AbsolutePath:      compPath,
		SiblingComponents: siblings,
		Vibeasik: VibeasikEngineInfo{
			APIBase:       apiBase,
			MCPServer:     "vibeasik",
			TodosEndpoint: fmt.Sprintf("%s/api/projects/%d/todos", apiBase, p.ID),
		},
		CodeMap:             codeMap,
		AIAgentInstructions: instructions,
		LinkedAt:            time.Now().Format(time.RFC3339),
	}
}

// WriteIdentifierFile writes the .vibeasik.json file to targetDir
func WriteIdentifierFile(targetDir string, idf ProjectIdentifier) (string, error) {
	if targetDir == "" {
		return "", fmt.Errorf("target directory is empty")
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	data, err := json.MarshalIndent(idf, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode identifier json: %w", err)
	}

	filePath := filepath.Join(targetDir, IdentifierFileName)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write identifier file %s: %w", filePath, err)
	}

	return filePath, nil
}

// LinkProjectWorkspaces creates .vibeasik.json in component directories and root path
func LinkProjectWorkspaces(db *gorm.DB, projectID uint, targetComponentID uint, apiBase string) ([]string, *ProjectIdentifier, error) {
	var p models.Project
	if err := db.Preload("Components").First(&p, projectID).Error; err != nil {
		return nil, nil, fmt.Errorf("project not found (ID: %d): %w", projectID, err)
	}

	var createdFiles []string
	var primaryIdentifier *ProjectIdentifier

	if targetComponentID > 0 {
		var targetComp *models.Component
		for i := range p.Components {
			if p.Components[i].ID == targetComponentID {
				targetComp = &p.Components[i]
				break
			}
		}
		if targetComp == nil {
			return nil, nil, fmt.Errorf("component ID %d not found in project %d", targetComponentID, projectID)
		}
		if targetComp.AbsolutePath != "" {
			idf := BuildIdentifier(&p, targetComp, apiBase)
			path, err := WriteIdentifierFile(targetComp.AbsolutePath, idf)
			if err != nil {
				return nil, nil, err
			}
			createdFiles = append(createdFiles, path)
			primaryIdentifier = &idf
		}
		return createdFiles, primaryIdentifier, nil
	}

	// Link all components (komponen ganda yang menunjuk folder sama hanya ditulis sekali, entri pertama menang)
	written := map[string]bool{}
	for i := range p.Components {
		comp := &p.Components[i]
		dir := filepath.Clean(comp.AbsolutePath)
		if comp.AbsolutePath != "" && !written[dir] {
			written[dir] = true
			idf := BuildIdentifier(&p, comp, apiBase)
			path, err := WriteIdentifierFile(comp.AbsolutePath, idf)
			if err == nil {
				createdFiles = append(createdFiles, path)
				if primaryIdentifier == nil {
					primaryIdentifier = &idf
				}
			}
		}
	}

	// Also link the project root when distinct. Root efektif dipakai agar proyek bundel ber-root folder umum
	// (mis. ~/Documents) tidak menulis .vibeasik.json ke folder itu.
	if root := codemap.EffectiveRoot(p); root != "" {
		if !written[root] {
			rootIdf := BuildIdentifier(&p, nil, apiBase)
			path, err := WriteIdentifierFile(root, rootIdf)
			if err == nil {
				createdFiles = append(createdFiles, path)
				if primaryIdentifier == nil {
					primaryIdentifier = &rootIdf
				}
			}
		}
	}

	if primaryIdentifier == nil {
		rootIdf := BuildIdentifier(&p, nil, apiBase)
		primaryIdentifier = &rootIdf
	}

	return createdFiles, primaryIdentifier, nil
}

// FindIdentifierInPath searches for .vibeasik.json in dirPath and its parents (up to 4 levels)
func FindIdentifierInPath(dirPath string) (*ProjectIdentifier, string, error) {
	if dirPath == "" {
		var err error
		dirPath, err = os.Getwd()
		if err != nil {
			return nil, "", err
		}
	}

	current := filepath.Clean(dirPath)
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(current, IdentifierFileName)
		if fileInfo, err := os.Stat(candidate); err == nil && !fileInfo.IsDir() {
			data, err := os.ReadFile(candidate)
			if err == nil {
				var idf ProjectIdentifier
				if err := json.Unmarshal(data, &idf); err == nil && idf.ProjectID > 0 {
					return &idf, candidate, nil
				}
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return nil, "", fmt.Errorf("no %s found in %s or its parent directories", IdentifierFileName, dirPath)
}

// IdentifyWorkspace resolves project, component, and active todos from path or database
func IdentifyWorkspace(db *gorm.DB, dirPath string, apiBase string) (*WorkspaceMatchResult, error) {
	if dirPath == "" {
		var err error
		dirPath, err = os.Getwd()
		if err != nil {
			dirPath = "."
		}
	}
	cleanPath := filepath.Clean(dirPath)

	result := &WorkspaceMatchResult{
		Matched: false,
	}

	// 1. Try finding .vibeasik.json
	idf, idfPath, err := FindIdentifierInPath(cleanPath)
	if err == nil && idf != nil {
		result.Matched = true
		result.MatchSource = "file"
		result.IdentifierFilePath = idfPath
		result.Identifier = idf

		var p models.Project
		if err := db.Preload("Components").First(&p, idf.ProjectID).Error; err == nil {
			result.Project = &p
			if idf.ComponentID > 0 {
				for i := range p.Components {
					if p.Components[i].ID == idf.ComponentID {
						result.CurrentComponent = &p.Components[i]
						break
					}
				}
			}
		}
	}

	// 2. If not found by file, search by matching absolute path in database
	if !result.Matched {
		var allProjects []models.Project
		if err := db.Preload("Components").Find(&allProjects).Error; err == nil {
			for i := range allProjects {
				p := &allProjects[i]
				// Check components
				for j := range p.Components {
					comp := &p.Components[j]
					if comp.AbsolutePath != "" {
						compClean := filepath.Clean(comp.AbsolutePath)
						if compClean == cleanPath || strings.HasPrefix(cleanPath, compClean+string(filepath.Separator)) {
							result.Matched = true
							result.MatchSource = "database_component_match"
							result.Project = p
							result.CurrentComponent = comp
							builtIdf := BuildIdentifier(p, comp, apiBase)
							result.Identifier = &builtIdf
							break
						}
					}
				}
				if result.Matched {
					break
				}
				// Check project root path
				if p.RootPath != "" && p.RootPath != "-" {
					rootClean := filepath.Clean(p.RootPath)
					if rootClean == cleanPath || strings.HasPrefix(cleanPath, rootClean+string(filepath.Separator)) {
						result.Matched = true
						result.MatchSource = "database_root_match"
						result.Project = p
						builtIdf := BuildIdentifier(p, nil, apiBase)
						result.Identifier = &builtIdf
						break
					}
				}
			}
		}
	}

	if !result.Matched {
		return result, fmt.Errorf("no vibeasik project or .vibeasik.json matched path '%s'", cleanPath)
	}

	// 3. Fetch active todos
	if result.Project != nil {
		query := db.Model(&models.Todo{}).Where("project_id = ?", result.Project.ID)
		var todos []models.Todo
		_ = query.Where("status IN ?", []string{models.TodoStatusPending, models.TodoStatusInProgress, models.TodoStatusSuggested}).
			Order("priority DESC, id DESC").
			Find(&todos).Error

		result.ActiveTodos = todos
		result.TodosCount = len(todos)
		for _, t := range todos {
			if t.Status == models.TodoStatusPending || t.Status == models.TodoStatusInProgress {
				result.PendingTodosCount++
			} else if t.Status == models.TodoStatusSuggested {
				result.SuggestedTodosCount++
			}
		}
	}

	if result.Identifier != nil {
		result.AIAgentInstructions = result.Identifier.AIAgentInstructions
	}

	return result, nil
}

// runPort: port yang benar-benar dipakai saat komponen dijalankan — port blok proyek yang dialokasikan engine
// (RunPort), atau port terdeteksi bila belum dialokasikan. Port terdeteksi lama (mis. 8091/3020) membuat agent dan
// audit runtime memakai alamat yang tidak lagi dipakai dev.sh/.env.
func runPort(c models.Component) int {
	if c.RunPort > 0 {
		return c.RunPort
	}
	return c.Port
}
