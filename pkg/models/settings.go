package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LLM Provider Constants
const (
	ProviderSystemOne   = "system_one"
	ProviderStandardLLM = "standard_llm"
)

// MaskKey masks sensitive API keys so they are never leaked in plaintext
func MaskKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "••••••••"
	}
	prefixLen := 4
	suffixLen := 4
	if len(key) < 12 {
		prefixLen = 2
		suffixLen = 2
	}
	return fmt.Sprintf("%s••••••••%s", key[:prefixLen], key[len(key)-suffixLen:])
}

// IsMaskedKey checks if the string contains mask characters (bullets or stars)
func IsMaskedKey(key string) bool {
	trimmed := strings.TrimSpace(key)
	return strings.Contains(trimmed, "•") || strings.Contains(trimmed, "****")
}

// LLMConfig stores an individual AI configuration endpoint and its usable models
type LLMConfig struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	Provider      string `gorm:"size:64;index;not null" json:"provider"` // "system_one" or "standard_llm"
	Name          string `gorm:"size:255;not null" json:"name"`
	URL           string `gorm:"size:512;not null" json:"url"`
	Key           string `gorm:"size:512" json:"-"`       // Stored in raw in DB, never serialized directly in JSON
	Models        string `gorm:"type:text" json:"models"` // JSON serialized array of strings
	SelectedModel string `gorm:"size:255" json:"selected_model"`
	IsActive      bool   `gorm:"default:false" json:"is_active"`
	VisionJSON    string `gorm:"type:text" json:"-"` // override dukungan gambar per model: {"model": true|false}
	// VisionAutoJSON: hasil uji gambar otomatis per model ({"model": true|false}); kalah dari override manual.
	VisionAutoJSON string    `gorm:"type:text" json:"-"`
	AliasJSON      string    `gorm:"type:text" json:"-"` // alias tampilan per model: {"model": "alias"}
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// LLMConfigDTO provides clean JSON serialization where keys are strictly masked and models are string slices
type LLMConfigDTO struct {
	ID            uint     `json:"id"`
	Provider      string   `json:"provider"`
	Name          string   `json:"name"`
	URL           string   `json:"url"`
	Key           string   `json:"key"`        // Always masked
	MaskedKey     string   `json:"masked_key"` // Always masked
	HasKey        bool     `json:"has_key"`
	Models        []string `json:"models"`
	SelectedModel string   `json:"selected_model"`
	IsActive      bool     `json:"is_active"`
	// Vision: dukungan gambar efektif per model; VisionManual: model yang nilainya diatur pengguna (bukan tebakan nama).
	Vision       map[string]bool `json:"vision"`
	VisionManual []string        `json:"vision_manual"`
	// VisionTested: model yang dukungan gambarnya ditetapkan dari uji otomatis (bukan tebakan nama).
	VisionTested []string `json:"vision_tested"`
	// Aliases: nama tampilan per model (pemilih model & Kantor Agent); request tetap memakai nama asli.
	Aliases   map[string]string `json:"aliases"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// ToDTO converts LLMConfig entity to DTO with masked key
func (c *LLMConfig) ToDTO() LLMConfigDTO {
	modelsList := []string{}
	if c.Models != "" {
		_ = json.Unmarshal([]byte(c.Models), &modelsList)
	}
	if len(modelsList) == 0 {
		if c.SelectedModel != "" {
			modelsList = []string{c.SelectedModel}
		} else if c.Provider == ProviderSystemOne {
			modelsList = []string{"convaiinnovations/laya-multilingual"}
		} else {
			modelsList = []string{"gpt-4o-mini", "gpt-4o"}
		}
	}

	selected := c.SelectedModel
	if selected == "" && len(modelsList) > 0 {
		selected = modelsList[0]
	}

	masked := MaskKey(c.Key)
	return LLMConfigDTO{
		ID:            c.ID,
		Provider:      c.Provider,
		Name:          c.Name,
		URL:           c.URL,
		Key:           masked,
		MaskedKey:     masked,
		HasKey:        c.Key != "",
		Models:        modelsList,
		SelectedModel: selected,
		IsActive:      c.IsActive,
		Vision:        c.visionMap(modelsList),
		VisionManual:  c.visionManual(),
		VisionTested:  c.visionTested(),
		Aliases:       c.aliasMap(),
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

// LLMSettings stores external LLM configurations (System One / Jev and Standard OpenAI-compatible LLM)
type LLMSettings struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	ActiveProvider string `gorm:"size:64;default:'system_one'" json:"active_provider"` // "system_one" or "standard_llm"

	// System One / Jev / TypeSafe
	SystemOneURL   string `gorm:"size:512;default:'http://localhost:8750/v1/systemone'" json:"system_one_url"`
	SystemOneModel string `gorm:"size:255;default:'convaiinnovations/laya-multilingual'" json:"system_one_model"`
	SystemOneKey   string `gorm:"size:512" json:"system_one_key"`

	// Standard External LLM (OpenAI-compatible)
	// 1 URL & Key can hold multiple models
	StandardLLMURL      string `gorm:"size:512;default:'https://api.openai.com/v1'" json:"standard_llm_url"`
	StandardLLMKey      string `gorm:"size:512" json:"standard_llm_key"`
	StandardLLMModels   string `gorm:"type:text" json:"standard_llm_models"` // JSON serialized array of strings
	StandardLLMSelected string `gorm:"size:255;default:'gpt-4o-mini'" json:"standard_llm_selected"`

	UpdatedAt time.Time `json:"updated_at"`
}

// LLMSettingsDTO provides clean JSON serialization for the REST API where models are string slices and keys are masked
type LLMSettingsDTO struct {
	ID                  uint      `json:"id"`
	ActiveProvider      string    `json:"active_provider"`
	SystemOneURL        string    `json:"system_one_url"`
	SystemOneModel      string    `json:"system_one_model"`
	SystemOneKey        string    `json:"system_one_key"`
	StandardLLMURL      string    `json:"standard_llm_url"`
	StandardLLMKey      string    `json:"standard_llm_key"`
	StandardLLMModels   []string  `json:"standard_llm_models"`
	StandardLLMSelected string    `json:"standard_llm_selected"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ToDTO converts LLMSettings entity to DTO with masked keys
func (s *LLMSettings) ToDTO() LLMSettingsDTO {
	modelsList := []string{}
	if s.StandardLLMModels != "" {
		_ = json.Unmarshal([]byte(s.StandardLLMModels), &modelsList)
	}
	if len(modelsList) == 0 {
		modelsList = []string{"gpt-4o-mini", "gpt-4o", "deepseek-chat", "claude-3-5-sonnet"}
	}

	selected := s.StandardLLMSelected
	if selected == "" && len(modelsList) > 0 {
		selected = modelsList[0]
	}

	return LLMSettingsDTO{
		ID:                  s.ID,
		ActiveProvider:      s.ActiveProvider,
		SystemOneURL:        s.SystemOneURL,
		SystemOneModel:      s.SystemOneModel,
		SystemOneKey:        MaskKey(s.SystemOneKey),
		StandardLLMURL:      s.StandardLLMURL,
		StandardLLMKey:      MaskKey(s.StandardLLMKey),
		StandardLLMModels:   modelsList,
		StandardLLMSelected: selected,
		UpdatedAt:           s.UpdatedAt,
	}
}

// FromDTO updates LLMSettings entity from DTO
func (s *LLMSettings) FromDTO(dto LLMSettingsDTO) {
	if dto.ActiveProvider != "" {
		s.ActiveProvider = dto.ActiveProvider
	}
	if dto.SystemOneURL != "" {
		s.SystemOneURL = dto.SystemOneURL
	}
	if dto.SystemOneModel != "" {
		s.SystemOneModel = dto.SystemOneModel
	}
	if dto.SystemOneKey != "" && !IsMaskedKey(dto.SystemOneKey) {
		s.SystemOneKey = dto.SystemOneKey
	}

	if dto.StandardLLMURL != "" {
		s.StandardLLMURL = dto.StandardLLMURL
	}
	if dto.StandardLLMKey != "" && !IsMaskedKey(dto.StandardLLMKey) {
		s.StandardLLMKey = dto.StandardLLMKey
	}

	if len(dto.StandardLLMModels) > 0 {
		bytes, err := json.Marshal(dto.StandardLLMModels)
		if err == nil {
			s.StandardLLMModels = string(bytes)
		}
	}

	if dto.StandardLLMSelected != "" {
		s.StandardLLMSelected = dto.StandardLLMSelected
	}
	s.UpdatedAt = time.Now()
}

// UserHome: home user yang menjalankan engine; tanpa HOME (service tanpa user) jatuh ke folder binary,
// tidak pernah ke path mesin developer.
func UserHome() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

// DefaultProjectsRoot: folder proyek bawaan (~/Documents/Projects).
func DefaultProjectsRoot() string {
	return filepath.Join(UserHome(), "Documents", "Projects")
}

// WorkspaceSettings stores user preferences for workspace directory layout
type WorkspaceSettings struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	DefaultProjectsRoot string    `gorm:"size:512" json:"default_projects_root"`
	UpdatedAt           time.Time `json:"updated_at"`
}
