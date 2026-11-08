package guidelines

import (
	"io/fs"

	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	vibeasik "github.com/vibeasik/vibeasik"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// GuidelineItem represents a single actionable rule/rubric
type GuidelineItem struct {
	ID              string `json:"id"`
	Level           string `json:"level"` // global, project, component
	Title           string `json:"title"`
	Rule            string `json:"rule"`
	Rationale       string `json:"rationale"`
	InspectionCheck string `json:"inspection_check"`
	SuggestedAction string `json:"suggested_action"`
	Severity        string `json:"severity"` // critical, high, medium, low
}

// GuidelineCategory represents a grouped collection of guidelines (e.g. 01_perencanaan, 02_arsitektur)
type GuidelineCategory struct {
	Category    string          `json:"category"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	FilePath    string          `json:"file_path,omitempty"`
	Guidelines  []GuidelineItem `json:"guidelines"`
}

// FindGuidelinesDir locates the guidelines folder dynamically
func FindGuidelinesDir() string {
	candidates := []string{
		"guidelines",
		"../guidelines",
		"../../guidelines",
	}

	// Override pengguna: rule kustom di ~/.vibeasik/guidelines menggantikan rule bawaan.
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".vibeasik", "guidelines"))
	}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "guidelines"),
			filepath.Join(cwd, "..", "guidelines"),
		)
	}

	for _, c := range candidates {
		if stat, err := os.Stat(c); err == nil && stat.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}

	return ""
}

// LoadAllGuidelines reads and parses all JSON files from the guidelines folder
func LoadAllGuidelines() ([]GuidelineCategory, error) {
	dir := FindGuidelinesDir()
	if dir == "" {
		return loadEmbeddedGuidelines()
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read guidelines dir: %w", err)
	}

	var categories []GuidelineCategory
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			fullPath := filepath.Join(dir, f.Name())
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}

			var cat GuidelineCategory
			if err := json.Unmarshal(content, &cat); err == nil {
				cat.FilePath = fullPath
				categories = append(categories, cat)
			}
		}
	}

	return categories, nil
}

// loadEmbeddedGuidelines memuat rule bawaan yang disematkan di binary (vibeasik.Guidelines).
func loadEmbeddedGuidelines() ([]GuidelineCategory, error) {
	files, err := fs.ReadDir(vibeasik.Guidelines, "guidelines")
	if err != nil {
		return nil, fmt.Errorf("embedded guidelines unavailable: %w", err)
	}
	var categories []GuidelineCategory
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		content, err := fs.ReadFile(vibeasik.Guidelines, "guidelines/"+f.Name())
		if err != nil {
			continue
		}
		var cat GuidelineCategory
		if err := json.Unmarshal(content, &cat); err == nil {
			cat.FilePath = "embedded:guidelines/" + f.Name()
			categories = append(categories, cat)
		}
	}
	return categories, nil
}

// GenerateGlobalSuggestions generates workspace/national level todo suggestions
func GenerateGlobalSuggestions(allProjects []models.Project) []models.Todo {
	var suggestions []models.Todo

	// PLAN-001: Global MCP Observability across workspace - Category: Feature
	suggestions = append(suggestions, models.Todo{
		Level:          models.TodoLevelGlobal,
		Category:       models.CategoryFeature,
		TargetApps:     "Global Platform (Semua Proyek)",
		Title:          "Standarisasi Model Context Protocol (MCP) di Seluruh Proyek Lokal",
		Description:    "Pastikan seluruh proyek lokal mendaftarkan berkas .vscode/mcp.json agar AI coding agent dapat langsung menginspeksi topologi kode.",
		Status:         models.TodoStatusSuggested,
		Priority:       models.TodoPriorityHigh,
		CreatorType:    models.CreatorAI,
		CreatorName:    "vibeasik-ai-advisor",
		IsAISuggestion: true,
		GuidelineRef:   "AI-001",
		Rationale:      "Patokan: [AI-001] Integrasi MCP menyeluruh meningkatkan akurasi konteks AI agent dan mengeliminasi tebak-tebakan rute pada level perencanaan workspace.",
		Source:         "ai_agent",
	})

	// SEC-001: Zero-Credential Leak Audit - Category: Security
	suggestions = append(suggestions, models.Todo{
		Level:          models.TodoLevelGlobal,
		Category:       models.CategorySecurity,
		TargetApps:     "Global Platform (Semua Proyek)",
		Title:          "Audit Rutin Sanitasi Kredensial & Secrets Lintas-Proyek",
		Description:    "Jalankan scanner keamanan berkala untuk memastikan tidak ada database password atau AI API key yang tertrack ke dalam git repositori.",
		Status:         models.TodoStatusSuggested,
		Priority:       models.TodoPriorityHigh,
		CreatorType:    models.CreatorAI,
		CreatorName:    "vibeasik-ai-advisor",
		IsAISuggestion: true,
		GuidelineRef:   "SEC-001",
		Rationale:      "Patokan: [SEC-001] Menjaga integritas keamanan seluruh workspace dari kebocoran token rahasia secara otomatis.",
		Source:         "ai_agent",
	})

	// NET-002: Local Subdomain / Traefik Reverse Proxy - Category: Performance
	suggestions = append(suggestions, models.Todo{
		Level:          models.TodoLevelGlobal,
		Category:       models.CategoryPerformance,
		TargetApps:     "Global Network (Reverse Proxy Traefik/Caddy)",
		Title:          "Setup Reverse Proxy Lokal untuk DNS Subdomain Proyek (*.localhost)",
		Description:    "Konfigurasikan Traefik atau Caddy lokal untuk memetakan domain lokal (misal: app.localhost, api.localhost) ke port internal masing-masing servis.",
		Status:         models.TodoStatusSuggested,
		Priority:       models.TodoPriorityMedium,
		CreatorType:    models.CreatorAI,
		CreatorName:    "vibeasik-ai-advisor",
		IsAISuggestion: true,
		GuidelineRef:   "NET-002",
		Rationale:      "Patokan: [NET-002] Menghilangkan konflik nomor port statis secara permanen di tingkat jaringan lokal dan menyederhanakan akses multi-bundle.",
		Source:         "ai_agent",
	})

	return suggestions
}
