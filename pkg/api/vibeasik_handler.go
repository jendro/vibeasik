package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/cloud"
)

var (
	cloudAuditCacheMu sync.RWMutex
	cloudAuditCache   = make(map[string]cloudAuditCacheEntry)
)

type cloudAuditCacheEntry struct {
	result    *cloud.CloudAuditResponseDTO
	expiresAt time.Time
}

func getCachedCloudAudit(key string) (*cloud.CloudAuditResponseDTO, bool) {
	cloudAuditCacheMu.RLock()
	defer cloudAuditCacheMu.RUnlock()
	entry, ok := cloudAuditCache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.result, true
}

func setCachedCloudAudit(key string, result *cloud.CloudAuditResponseDTO) {
	cloudAuditCacheMu.Lock()
	defer cloudAuditCacheMu.Unlock()
	if len(cloudAuditCache) > 100 {
		cloudAuditCache = make(map[string]cloudAuditCacheEntry)
	}
	cloudAuditCache[key] = cloudAuditCacheEntry{
		result:    result,
		expiresAt: time.Now().Add(60 * time.Second),
	}
}

func computeBodyHash(body interface{}) string {
	b, err := json.Marshal(body)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (s *Server) getVibeTutorials(c *fiber.Ctx) error {
	category := c.Query("category")
	cli := cloud.NewClient()
	list, err := cli.GetTutorials(category)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"tutorials": list})
}

func (s *Server) searchVibeDomains(c *fiber.Ctx) error {
	query := c.Query("query")
	if query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Query domain harus diisi"})
	}
	cli := cloud.NewClient()
	resp, err := cli.SearchDomains(query)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(resp)
}

func (s *Server) generateVibeDeployPlan(c *fiber.Ctx) error {
	var body map[string]interface{}
	if msg := bindJSONRequired(c, &body); msg != "" {
		return badPayload(c, msg)
	}
	cli := cloud.NewClient()
	plan, err := cli.GenerateDeployPlan(body)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(plan)
}

func (s *Server) runCloudSecurityAudit(c *fiber.Ctx) error {
	var body map[string]interface{}
	if msg := bindJSONRequired(c, &body); msg != "" {
		return badPayload(c, msg)
	}

	c.Set(fiber.HeaderCacheControl, "private, max-age=60")

	cacheKey := computeBodyHash(body)
	if cacheKey != "" {
		if cached, found := getCachedCloudAudit(cacheKey); found {
			return c.JSON(cached)
		}
	}

	cli := cloud.NewClient()
	result, err := cli.AuditProject(body)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}

	if cacheKey != "" && result != nil {
		setCachedCloudAudit(cacheKey, result)
	}

	return c.JSON(result)
}

func (s *Server) getVibeAITemplates(c *fiber.Ctx) error {
	cli := cloud.NewClient()
	templates, err := cli.GetAITemplates()
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"templates": templates})
}

type InjectAITemplateRequest struct {
	ComponentID uint   `json:"component_id"`
	TemplateID  string `json:"template_id"`
	TargetPath  string `json:"target_path"` // absolute path to component directory
}

func (s *Server) injectVibeAITemplate(c *fiber.Ctx) error {
	var req InjectAITemplateRequest
	if msg := bindJSONRequired(c, &req); msg != "" {
		return badPayload(c, msg)
	}

	if req.TargetPath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Target path komponen harus diisi"})
	}
	if msg := firstProblem(absPath("target_path", req.TargetPath), needText("template_id", req.TemplateID, maxName)); msg != "" {
		return badPayload(c, msg)
	}

	cli := cloud.NewClient()
	templates, err := cli.GetAITemplates()
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}

	var matched *cloud.AITemplateDTO
	for _, t := range templates {
		if t.ID == req.TemplateID {
			matched = &t
			break
		}
	}

	if matched == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Template AI tidak ditemukan"})
	}

	var createdFiles []string
	for _, f := range matched.Files {
		fullPath := filepath.Join(req.TargetPath, filepath.FromSlash(f.Filename))
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal membuat direktori: " + err.Error()})
		}
		if err := os.WriteFile(fullPath, []byte(f.Content), 0644); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal menulis file: " + err.Error()})
		}
		createdFiles = append(createdFiles, strings.TrimPrefix(fullPath, req.TargetPath))
	}

	return c.JSON(fiber.Map{
		"success":       true,
		"template_name": matched.Name,
		"created_files": createdFiles,
		"install_guide": matched.InstallGuide,
		"message":       "Template AI berhasil disuntikkan ke proyek lokal!",
	})
}
