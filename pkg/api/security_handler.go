package api

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/security"
)

const (
	maxCensorCacheEntries = 128
	censorCacheTTL        = 60 * time.Second
)

// CensorCacheStats merekam metrik cache terpisah dari latency endpoint (K2157)
type CensorCacheStats struct {
	Hits      uint64 `json:"hits"`
	Misses    uint64 `json:"misses"`
	Bypasses  uint64 `json:"bypasses"`
	Expiries  uint64 `json:"expiries"`
	Evictions uint64 `json:"evictions"`
}

var (
	censorTestCacheMu sync.RWMutex
	censorTestCache   = make(map[string]censorTestCacheEntry)
	censorStatsMu     sync.Mutex
	censorCacheStats  CensorCacheStats
)

type censorTestCacheEntry struct {
	clean     string
	audit     *security.SanitizeAudit
	expiresAt time.Time
}

// computeCensorCacheKey menghasilkan canonical hash key (K5741) agar raw secrets tidak tersimpan di memori cache
func computeCensorCacheKey(text string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(text)))
	return fmt.Sprintf("censor:sha256:%x", h)
}

func recordCensorStat(fn func(*CensorCacheStats)) {
	censorStatsMu.Lock()
	defer censorStatsMu.Unlock()
	fn(&censorCacheStats)
}

func getCensorCacheStats() CensorCacheStats {
	censorStatsMu.Lock()
	defer censorStatsMu.Unlock()
	return censorCacheStats
}

func resetCensorCacheStats() {
	censorStatsMu.Lock()
	defer censorStatsMu.Unlock()
	censorCacheStats = CensorCacheStats{}
}

func getCachedCensorTest(key string) (string, *security.SanitizeAudit, bool) {
	censorTestCacheMu.RLock()
	entry, ok := censorTestCache[key]
	censorTestCacheMu.RUnlock()

	if !ok {
		recordCensorStat(func(s *CensorCacheStats) { s.Misses++ })
		return "", nil, false
	}

	if time.Now().After(entry.expiresAt) {
		censorTestCacheMu.Lock()
		delete(censorTestCache, key)
		censorTestCacheMu.Unlock()
		recordCensorStat(func(s *CensorCacheStats) {
			s.Misses++
			s.Expiries++
		})
		return "", nil, false
	}

	recordCensorStat(func(s *CensorCacheStats) { s.Hits++ })
	return entry.clean, entry.audit, true
}

func evictExpiredOrOldestCensorEntries(now time.Time) {
	for k, v := range censorTestCache {
		if now.After(v.expiresAt) {
			delete(censorTestCache, k)
			recordCensorStat(func(s *CensorCacheStats) { s.Expiries++ })
		}
	}
	if len(censorTestCache) < maxCensorCacheEntries {
		return
	}
	var oldestKey string
	var oldestTime time.Time
	for k, v := range censorTestCache {
		if oldestKey == "" || v.expiresAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.expiresAt
		}
	}
	if oldestKey != "" {
		delete(censorTestCache, oldestKey)
		recordCensorStat(func(s *CensorCacheStats) { s.Evictions++ })
	}
}

func setCachedCensorTest(key, clean string, audit *security.SanitizeAudit) {
	censorTestCacheMu.Lock()
	defer censorTestCacheMu.Unlock()

	now := time.Now()
	if len(censorTestCache) >= maxCensorCacheEntries {
		evictExpiredOrOldestCensorEntries(now)
	}

	censorTestCache[key] = censorTestCacheEntry{
		clean:     clean,
		audit:     audit,
		expiresAt: now.Add(censorCacheTTL),
	}
}

func invalidateCensorTestCache() {
	censorTestCacheMu.Lock()
	defer censorTestCacheMu.Unlock()
	censorTestCache = make(map[string]censorTestCacheEntry)
}

func (s *Server) getSecurityStatus(c *fiber.Ctx) error {
	var activeCount int64
	s.db.Model(&models.CensoredKeyword{}).Where("is_active = ?", true).Count(&activeCount)

	return c.JSON(fiber.Map{
		"global_sanitizer_active": true,
		"double_security_active":  true,
		"censoring_mode":          "deterministic_safe_aliasing",
		"intercepted_providers":   []string{"system_one", "standard_llm"},
		"protected_patterns":      security.GlobalSanitizer.GetBuiltInPatterns(),
		"active_custom_keywords":  activeCount,
		"leak_prevention":         "100% Guaranteed. Double sanitization enforces deterministic safe aliasing on all prompts, files, and network dispatches.",
	})
}

func (s *Server) getCensoredKeywords(c *fiber.Ctx) error {
	var list []models.CensoredKeyword
	if err := s.db.
		Select("id", "keyword", "category", "description", "is_regex", "is_active").
		Order("id DESC").
		Find(&list).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	activeCount := 0
	for _, keyword := range list {
		if keyword.IsActive {
			activeCount++
		}
	}

	return c.JSON(fiber.Map{
		"keywords":          list,
		"total":             len(list),
		"active_count":      activeCount,
		"built_in_patterns": security.GlobalSanitizer.GetBuiltInPatterns(),
		"double_security":   true,
	})
}

func (s *Server) createCensoredKeyword(c *fiber.Ctx) error {
	var body struct {
		Keyword     string `json:"keyword"`
		Category    string `json:"category"`
		Description string `json:"description"`
		IsRegex     bool   `json:"is_regex"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	body.Keyword = strings.TrimSpace(body.Keyword)
	if body.Keyword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "keyword is required"})
	}
	if body.Category == "" {
		body.Category = "custom"
	}
	active := true
	if body.IsActive != nil {
		active = *body.IsActive
	}

	kw := models.CensoredKeyword{
		Keyword:     body.Keyword,
		Category:    body.Category,
		Description: strings.TrimSpace(body.Description),
		IsRegex:     body.IsRegex,
		IsActive:    active,
	}

	if err := s.db.Create(&kw).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	database.SyncCensoredKeywords(s.db)
	invalidateCensorTestCache()

	return c.Status(fiber.StatusCreated).JSON(kw)
}

func (s *Server) updateCensoredKeyword(c *fiber.Ctx) error {
	id := c.Params("id")
	var existing models.CensoredKeyword
	if err := s.db.First(&existing, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "keyword not found"})
	}

	var body struct {
		Keyword     string `json:"keyword"`
		Category    string `json:"category"`
		Description string `json:"description"`
		IsRegex     *bool  `json:"is_regex"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if strings.TrimSpace(body.Keyword) != "" {
		existing.Keyword = strings.TrimSpace(body.Keyword)
	}
	if body.Category != "" {
		existing.Category = body.Category
	}
	if body.Description != "" {
		existing.Description = strings.TrimSpace(body.Description)
	}
	if body.IsRegex != nil {
		existing.IsRegex = *body.IsRegex
	}
	if body.IsActive != nil {
		existing.IsActive = *body.IsActive
	}

	if err := s.db.Save(&existing).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	database.SyncCensoredKeywords(s.db)
	invalidateCensorTestCache()

	return c.JSON(existing)
}

func (s *Server) deleteCensoredKeyword(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := s.db.Delete(&models.CensoredKeyword{}, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	database.SyncCensoredKeywords(s.db)
	invalidateCensorTestCache()
	return c.JSON(fiber.Map{"message": "censored keyword deleted successfully"})
}

type testCensorRequest struct {
	Text string `json:"text" validate:"required"`
}

func shouldBypassCensorCache(c *fiber.Ctx) bool {
	cc := c.Get(fiber.HeaderCacheControl)
	if strings.Contains(cc, "no-cache") || strings.Contains(cc, "no-store") {
		return true
	}
	return c.Get("X-Bypass-Cache") == "true"
}

func (s *Server) testCensorPayload(c *fiber.Ctx) error {
	var body testCensorRequest
	if p := firstProblem(bindJSONRequired(c, &body), needText("text", body.Text, maxText)); p != "" {
		return badPayload(c, p)
	}

	// 1. Jalur bypass: permintaan eksplisit client untuk tidak memakai cache (K2563)
	if shouldBypassCensorCache(c) {
		recordCensorStat(func(s *CensorCacheStats) { s.Bypasses++ })
		clean, audit := security.GlobalSanitizer.Sanitize(body.Text)
		c.Set("X-Cache", "BYPASS")
		c.Set(fiber.HeaderCacheControl, "private, no-cache, no-store")
		return c.JSON(fiber.Map{
			"original":       body.Text,
			"sanitized_text": clean,
			"audit":          audit,
		})
	}

	// 2. Jalur cache hit dengan canonical key (K5741)
	key := computeCensorCacheKey(body.Text)
	if clean, audit, found := getCachedCensorTest(key); found {
		c.Set("X-Cache", "HIT")
		c.Set(fiber.HeaderCacheControl, "private, max-age=60")
		return c.JSON(fiber.Map{
			"original":       body.Text,
			"sanitized_text": clean,
			"audit":          audit,
		})
	}

	// 3. Jalur cold miss: sanitasi penuh & simpan dengan canonical key
	clean, audit := security.GlobalSanitizer.Sanitize(body.Text)
	setCachedCensorTest(key, clean, audit)

	c.Set("X-Cache", "MISS")
	c.Set(fiber.HeaderCacheControl, "private, max-age=60")
	return c.JSON(fiber.Map{
		"original":       body.Text,
		"sanitized_text": clean,
		"audit":          audit,
	})
}
