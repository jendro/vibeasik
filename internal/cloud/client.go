package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vibeasik/vibeasik/internal/config"
)

// TutorialDTO represents a tutorial item from vibeasik cloud
type TutorialDTO struct {
	ID               uint      `json:"id"`
	Slug             string    `json:"slug"`
	Title            string    `json:"title"`
	Category         string    `json:"category"`
	Difficulty       string    `json:"difficulty"`
	EstimatedMinutes int       `json:"estimated_minutes"`
	Summary          string    `json:"summary"`
	ContentMarkdown  string    `json:"content_markdown"`
	CodeSnippets     string    `json:"code_snippets"`
	Tags             string    `json:"tags"`
	SourceURL        string    `json:"source_url"`
	CreatedAt        time.Time `json:"created_at"`
}

type DomainItemDTO struct {
	Domain       string  `json:"domain"`
	TLD          string  `json:"tld"`
	Available    bool    `json:"available"`
	PriceAnnual  float64 `json:"price_annual"`
	Currency     string  `json:"currency"`
	AutoDNSReady bool    `json:"auto_dns_ready"`
}

type DomainSearchResponseDTO struct {
	Query       string          `json:"query"`
	Suggestions []DomainItemDTO `json:"suggestions"`
}

type DeployFileDTO struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
	Purpose  string `json:"purpose"`
}

type DeployPlanResponseDTO struct {
	TargetPlatform string          `json:"target_platform"`
	Architecture   string          `json:"architecture"`
	GeneratedFiles []DeployFileDTO `json:"generated_files"`
	DeployCommands []string        `json:"deploy_commands"`
	EnvTemplate    string          `json:"env_template"`
	Notes          string          `json:"notes"`
}

type SecurityIssueDTO struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

type CloudAuditResponseDTO struct {
	SafetyScore int                `json:"safety_score"`
	Status      string             `json:"status"`
	Issues      []SecurityIssueDTO `json:"issues"`
	PassedRules []string           `json:"passed_rules"`
	Summary     string             `json:"summary"`
}

type AITemplateDTO struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Type            string          `json:"type"`
	TargetLanguage  string          `json:"target_language"`
	TargetFramework string          `json:"target_framework"`
	Description     string          `json:"description"`
	Benefits        []string        `json:"benefits"`
	Files           []DeployFileDTO `json:"files"`
	Dependencies    []string        `json:"dependencies"`
	InstallGuide    string          `json:"install_guide"`
}

// Client communicates with vibeasik.com cloud API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient() *Client {
	cfg := config.GetConfig()
	url := cfg.VibeasikCloudURL
	if url == "" {
		url = "http://localhost:8770"
	}
	return &Client{
		baseURL: url,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetTutorials retrieves interactive guides from vibeasik.com
func (c *Client) GetTutorials(category string) ([]TutorialDTO, error) {
	endpoint := fmt.Sprintf("%s/api/v1/tutorials", c.baseURL)
	if category != "" {
		endpoint += fmt.Sprintf("?category=%s", category)
	}

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		// Offline fallback
		return c.getFallbackTutorials(), nil
	}
	defer resp.Body.Close()

	var payload struct {
		Tutorials []TutorialDTO `json:"tutorials"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return c.getFallbackTutorials(), nil
	}

	return payload.Tutorials, nil
}

// SearchDomains queries domain availability and pricing
func (c *Client) SearchDomains(query string) (*DomainSearchResponseDTO, error) {
	endpoint := fmt.Sprintf("%s/api/v1/domains/search?query=%s", c.baseURL, query)
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("cloud server unreachable: %w", err)
	}
	defer resp.Body.Close()

	var result DomainSearchResponseDTO
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GenerateDeployPlan generates Dockerfile, docker-compose, and Caddyfile
func (c *Client) GenerateDeployPlan(req interface{}) (*DeployPlanResponseDTO, error) {
	endpoint := fmt.Sprintf("%s/api/v1/deploy/generate-plan", c.baseURL)
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("cloud deploy planner unreachable: %w", err)
	}
	defer resp.Body.Close()

	var plan DeployPlanResponseDTO
	if err := json.NewDecoder(resp.Body).Decode(&plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// AuditProject sends code snippets to cloud security scanner
func (c *Client) AuditProject(req interface{}) (*CloudAuditResponseDTO, error) {
	endpoint := fmt.Sprintf("%s/api/v1/security/audit", c.baseURL)
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("cloud security audit unreachable: %w", err)
	}
	defer resp.Body.Close()

	var result CloudAuditResponseDTO
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAITemplates retrieves available agent templates
func (c *Client) GetAITemplates() ([]AITemplateDTO, error) {
	endpoint := fmt.Sprintf("%s/api/v1/ai/templates", c.baseURL)
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("cloud ai service unreachable: %w", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Templates []AITemplateDTO `json:"templates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Templates, nil
}

func (c *Client) getFallbackTutorials() []TutorialDTO {
	return []TutorialDTO{
		{
			ID:               1,
			Slug:             "best-practice-go-fiber",
			Title:            "Best Practice Go dengan Fiber: Arsitektur, Kinerja Zero Allocation, & Keamanan",
			Category:         "dasar-pemrograman",
			Difficulty:       "pemula",
			EstimatedMinutes: 10,
			Summary:          "Panduan komprehensif best practice membangun REST API menggunakan Go Fiber berdasarkan dokumentasi resmi dan standar industri: arsitektur Clean Architecture, manajemen memori fasthttp, error handling terpusat, pengurutan middleware, hingga graceful shutdown.",
			Tags:             "go,fiber,best-practices,clean-architecture,rest-api,fasthttp,security",
			SourceURL:        "https://docs.gofiber.io",
			ContentMarkdown:  "Go Fiber adalah web framework berperforma tinggi di atas fasthttp. Terapkan Clean Architecture, perhatikan zero-allocation memory copying pada goroutine, gunakan centralized error handler, dan pasang graceful shutdown.",
		},
		{
			ID:               2,
			Slug:             "dasar-arsitektur-web-modern",
			Title:            "Dasar Arsitektur Web Modern untuk Vibecoder (Client, Server, & Database)",
			Category:         "dasar-pemrograman",
			Difficulty:       "pemula",
			EstimatedMinutes: 8,
			Summary:          "Pahami bagaimana Frontend (Nuxt), Backend (Go/Fiber), dan Database berkomunikasi agar tidak bingung saat menyatukan kode AI.",
			Tags:             "arsitektur,frontend,backend,rest-api",
			ContentMarkdown:  "Frontend menyajikan UI, Backend memvalidasi data dan mengakses database. Jangan simpan kredensial privat di Frontend!",
		},
		{
			ID:               3,
			Slug:             "5-celah-keamanan-fatal-vibecoder",
			Title:            "5 Celah Keamanan Fatal yang Sering Dibuat Vibecoder & Cara Mengatasinya",
			Category:         "security",
			Difficulty:       "menengah",
			EstimatedMinutes: 10,
			Summary:          "Ketahui bahaya file .env terekspos, SQL injection, CORS wildcard, dan hilangnya validasi input sebelum aplikasi Anda dideploy.",
			Tags:             "security,env,cors,sql-injection",
			ContentMarkdown:  "Pastikan file .env ada di .gitignore dan gunakan parameterized query saat berinteraksi dengan database.",
		},
	}
}
