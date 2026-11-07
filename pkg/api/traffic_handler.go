package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/traffic"
)

// Traffic Analyzer (Aturan 48): statistik request aplikasi live → saran rate limit, cache, load balancer,
// pemecahan service, dll. Lihat internal/traffic untuk alur data & aturan saran.

type trafficSourceView struct {
	models.TrafficSource
	ProjectName   string `json:"project_name,omitempty"`
	ComponentName string `json:"component_name,omitempty"`
}

func (s *Server) trafficSource(c *fiber.Ctx) (*models.TrafficSource, error) {
	id, bad := paramID(c, "id", "traffic source")
	if bad != "" {
		return nil, badPayload(c, bad)
	}
	var src models.TrafficSource
	if err := s.db.First(&src, id).Error; err != nil {
		return nil, c.Status(404).JSON(fiber.Map{"error": "Sumber traffic tidak ditemukan"})
	}
	return &src, nil
}

func (s *Server) getTrafficMeta(c *fiber.Ctx) error {
	windows := []string{"15m", "1h", "6h", "24h", "7d", "30d"}
	return c.JSON(fiber.Map{
		"sync_intervals":    traffic.SyncIntervals,
		"windows":           windows,
		"default_cloud_url": config.GetConfig().VibeasikCloudURL,
	})
}

func (s *Server) listTrafficSources(c *fiber.Ctx) error {
	q := s.db.Order("id ASC")
	if pid := c.QueryInt("project_id", 0); pid > 0 {
		q = q.Where("project_id = ?", pid)
	}
	var list []models.TrafficSource
	if err := q.Find(&list).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	out := make([]trafficSourceView, 0, len(list))
	for _, src := range list {
		v := trafficSourceView{TrafficSource: src}
		if src.ProjectID != nil {
			var p models.Project
			if s.db.Select("name").First(&p, *src.ProjectID).Error == nil {
				v.ProjectName = p.Name
			}
		}
		if src.ComponentID != nil {
			var comp models.Component
			if s.db.Select("name").First(&comp, *src.ComponentID).Error == nil {
				v.ComponentName = comp.Name
			}
		}
		out = append(out, v)
	}
	return c.JSON(out)
}

type trafficSourceRequest struct {
	Name            string `json:"name"`
	ProjectID       *uint  `json:"project_id"`
	ComponentID     *uint  `json:"component_id"`
	Mode            string `json:"mode"`
	CloudURL        string `json:"cloud_url"`
	SyncIntervalSec int    `json:"sync_interval_sec"`
	Enabled         *bool  `json:"enabled"`
}

// validateTrafficSourceRequest memeriksa bentuk payload sumber traffic (create & update).
func validateTrafficSourceRequest(r trafficSourceRequest) string {
	if r.SyncIntervalSec < 0 {
		return "field sync_interval_sec tidak boleh negatif"
	}
	return firstProblem(maxLen("name", r.Name, maxName), maxLen("cloud_url", r.CloudURL, maxURL),
		oneOf("mode", r.Mode, models.TrafficModeCloud, models.TrafficModeDirect))
}

func validInterval(sec int) int {
	for _, v := range traffic.SyncIntervals {
		if v == sec {
			return sec
		}
	}
	return 60
}

func (s *Server) createTrafficSource(c *fiber.Ctx) error {
	var req trafficSourceRequest
	if msg := firstProblem(bindJSON(c, &req), validateTrafficSourceRequest(req)); msg != "" {
		return badPayload(c, msg)
	}
	src := models.TrafficSource{
		Name: strings.TrimSpace(req.Name), ProjectID: req.ProjectID, ComponentID: req.ComponentID,
		Mode: req.Mode, CloudURL: strings.TrimRight(strings.TrimSpace(req.CloudURL), "/"),
		SyncIntervalSec: validInterval(req.SyncIntervalSec), Enabled: true,
	}
	if src.ComponentID != nil && src.ProjectID == nil {
		var comp models.Component
		if s.db.First(&comp, *src.ComponentID).Error == nil {
			src.ProjectID = &comp.ProjectID
		}
	}
	if src.Name == "" {
		src.Name = "Aplikasi live"
		if src.ComponentID != nil {
			var comp models.Component
			if s.db.Select("name").First(&comp, *src.ComponentID).Error == nil {
				src.Name = comp.Name
			}
		}
	}
	switch src.Mode {
	case models.TrafficModeDirect:
		b := make([]byte, 20)
		_, _ = rand.Read(b)
		src.IngestKey = "vtd_" + hex.EncodeToString(b)
		src.CloudURL = ""
	default:
		src.Mode = models.TrafficModeCloud
		if src.CloudURL == "" {
			src.CloudURL = strings.TrimRight(config.GetConfig().VibeasikCloudURL, "/")
		}
		if !strings.HasPrefix(src.CloudURL, "http://") && !strings.HasPrefix(src.CloudURL, "https://") {
			return c.Status(400).JSON(fiber.Map{"error": "URL vibeasik live harus diawali http:// atau https://"})
		}
		if err := traffic.RegisterCloud(&src); err != nil {
			return c.Status(502).JSON(fiber.Map{"error": err.Error()})
		}
	}
	if err := s.db.Create(&src).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(src)
}

func (s *Server) updateTrafficSource(c *fiber.Ctx) error {
	var req trafficSourceRequest
	if msg := firstProblem(bindJSONRequired(c, &req), validateTrafficSourceRequest(req)); msg != "" {
		return badPayload(c, msg)
	}
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	updates := map[string]any{}
	if n := strings.TrimSpace(req.Name); n != "" {
		updates["name"] = n
	}
	if req.SyncIntervalSec > 0 {
		updates["sync_interval_sec"] = validInterval(req.SyncIntervalSec)
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.ComponentID != nil {
		updates["component_id"] = *req.ComponentID
	}
	if len(updates) > 0 {
		if err := s.db.Model(src).Updates(updates).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}
	s.db.First(src, src.ID)
	return c.JSON(src)
}

func (s *Server) deleteTrafficSource(c *fiber.Ctx) error {
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	warning := ""
	if src.Mode == models.TrafficModeCloud && src.PullKey != "" {
		if err := traffic.DeleteCloud(src); err != nil {
			warning = "Data lokal dihapus, tetapi sumber di vibeasik live gagal dihapus: " + err.Error()
		}
	}
	for _, m := range []any{&models.TrafficRouteStat{}, &models.TrafficDimStat{}, &models.TrafficRecentEvent{}} {
		s.db.Where("source_id = ?", src.ID).Delete(m)
	}
	s.db.Delete(src)
	return c.JSON(fiber.Map{"deleted": true, "warning": warning})
}

func (s *Server) syncTrafficSource(c *fiber.Ctx) error {
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	n, syncErr := traffic.SyncNow(s.db, src.ID)
	s.db.First(src, src.ID)
	if syncErr != nil {
		return c.Status(502).JSON(fiber.Map{"error": syncErr.Error(), "source": src})
	}
	return c.JSON(fiber.Map{"pulled": n, "source": src})
}

func (s *Server) rotateTrafficKey(c *fiber.Ctx) error {
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	if src.Mode == models.TrafficModeCloud {
		if err := traffic.RotateCloudKey(src); err != nil {
			return c.Status(502).JSON(fiber.Map{"error": err.Error()})
		}
	} else {
		b := make([]byte, 20)
		_, _ = rand.Read(b)
		src.IngestKey = "vtd_" + hex.EncodeToString(b)
	}
	s.db.Model(src).Update("ingest_key", src.IngestKey)
	return c.JSON(fiber.Map{"rotated": true})
}

func (s *Server) getTrafficReport(c *fiber.Ctx) error {
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	rep, buildErr := traffic.BuildReport(s.db, *src, traffic.ParseWindow(c.Query("window", "24h")))
	if buildErr != nil {
		return c.Status(500).JSON(fiber.Map{"error": buildErr.Error()})
	}
	return c.JSON(rep)
}

func (s *Server) getTrafficRecent(c *fiber.Ctx) error {
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	limit := c.QueryInt("limit", 100)
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := s.db.Where("source_id = ?", src.ID).Order("id DESC").Limit(limit)
	if after := c.QueryInt("after_id", 0); after > 0 {
		q = q.Where("id > ?", after)
	}
	var events []models.TrafficRecentEvent
	q.Find(&events)
	return c.JSON(fiber.Map{"events": events, "source": src})
}

func (s *Server) trafficIngestURL(c *fiber.Ctx, src *models.TrafficSource) string {
	if src.Mode == models.TrafficModeCloud {
		return src.CloudURL + "/api/v1/traffic/ingest"
	}
	return c.BaseURL() + "/api/traffic/ingest"
}

func (s *Server) getTrafficSDK(c *fiber.Ctx) error {
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	var comp models.Component
	hasComp := src.ComponentID != nil && s.db.First(&comp, *src.ComponentID).Error == nil
	module, suggested := "", "go-fiber"
	if hasComp {
		module = traffic.GoModule(comp.AbsolutePath)
		suggested = traffic.SuggestKit(comp.Language, comp.Framework)
	}
	url := s.trafficIngestURL(c, src)
	env := fmt.Sprintf("VIBEASIK_TRAFFIC_URL=%s\nVIBEASIK_TRAFFIC_KEY=%s\n# interval kirim (detik): 1-2 = realtime, 60 = per menit\nVIBEASIK_TRAFFIC_FLUSH_SEC=10\n# opsional: VIBEASIK_TRAFFIC_INSTANCE=api-1  VIBEASIK_TRAFFIC_HASH_IP=1  VIBEASIK_TRAFFIC_SKIP=/health", url, src.IngestKey)
	return c.JSON(fiber.Map{
		"ingest_url":      url,
		"ingest_key":      src.IngestKey,
		"env":             env,
		"env_file":        map[bool]string{true: ".env.server", false: ".env"}[src.Mode == models.TrafficModeCloud],
		"suggested":       suggested,
		"kits":            traffic.Kits(module),
		"component_dir":   map[bool]string{true: comp.AbsolutePath, false: ""}[hasComp],
		"component_name":  map[bool]string{true: comp.Name, false: ""}[hasComp],
		"curl_test":       fmt.Sprintf("curl -X POST %s -H 'Content-Type: application/json' -H 'X-Vibeasik-Key: %s' -d '{\"instance\":\"uji\",\"events\":[{\"ts\":%d,\"method\":\"GET\",\"path\":\"/uji\",\"status\":200,\"dur_ms\":12}]}'", url, src.IngestKey, time.Now().UnixMilli()),
		"realtime_flush":  2,
		"recommend_flush": 10,
	})
}

func (s *Server) installTrafficSDK(c *fiber.Ctx) error {
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	var req struct {
		Kit       string `json:"kit"`
		Overwrite bool   `json:"overwrite"`
	}
	if msg := firstProblem(bindJSONRequired(c, &req), needText("kit", req.Kit, maxName)); msg != "" {
		return badPayload(c, msg)
	}
	var comp models.Component
	if src.ComponentID == nil || s.db.First(&comp, *src.ComponentID).Error != nil || comp.AbsolutePath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Sumber belum terhubung ke komponen yang punya folder di laptop ini"})
	}
	kit := traffic.KitByID(req.Kit, traffic.GoModule(comp.AbsolutePath))
	if kit == nil {
		return c.Status(400).JSON(fiber.Map{"error": "SDK tidak dikenal: " + req.Kit})
	}
	written, skipped, werr := traffic.InstallKit(kit, comp.AbsolutePath, req.Overwrite)
	if werr != nil {
		return c.Status(500).JSON(fiber.Map{"error": werr.Error(), "written": written})
	}
	return c.JSON(fiber.Map{"written": written, "skipped": skipped, "dir": comp.AbsolutePath, "usage": kit.Usage})
}

var trafficTodoCategory = map[string]string{
	"rate_limit": models.CategorySecurity, "security": models.CategorySecurity,
	"cache": models.CategoryPerformance, "performance": models.CategoryPerformance, "payload": models.CategoryPerformance,
	"load_balancer": models.CategoryDevOps, "capacity": models.CategoryDevOps,
	"split_service": models.CategoryArchitecture, "errors": models.CategoryBackend,
}

var trafficTodoPriority = map[string]string{
	traffic.SevCritical: models.TodoPriorityCritical, traffic.SevWarning: models.TodoPriorityHigh, traffic.SevInfo: models.TodoPriorityMedium,
}

// trafficRecommendationTodo menjadikan satu saran traffic sebagai todo saran (status suggested) di proyek sumber.
// Kunci audit "TRAFFIC|<sumber>|<id saran>" mencegah todo ganda untuk saran yang sama.
func (s *Server) trafficRecommendationTodo(c *fiber.Ctx) error {
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	var req struct {
		RecID  string `json:"rec_id"`
		Window string `json:"window"`
	}
	if msg := firstProblem(bindJSONRequired(c, &req), needText("rec_id", req.RecID, maxName), maxLen("window", req.Window, 16)); msg != "" {
		return badPayload(c, msg)
	}
	if src.ProjectID == nil {
		return c.Status(400).JSON(fiber.Map{"error": "Hubungkan sumber ke proyek dulu agar todo punya tempat"})
	}
	rep, buildErr := traffic.BuildReport(s.db, *src, traffic.ParseWindow(req.Window))
	if buildErr != nil {
		return c.Status(500).JSON(fiber.Map{"error": buildErr.Error()})
	}
	var rec *traffic.Recommendation
	for i := range rep.Recommendations {
		if rep.Recommendations[i].ID == req.RecID {
			rec = &rep.Recommendations[i]
		}
	}
	if rec == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Saran tidak ada lagi di jendela ini (data sudah berubah)"})
	}
	key := fmt.Sprintf("TRAFFIC|%d|%s", src.ID, rec.ID)
	var existing models.Todo
	if s.db.Where("audit_key = ? AND status NOT IN ?", key, []string{models.TodoStatusCompleted, models.TodoStatusRejected}).First(&existing).Error == nil {
		return c.JSON(fiber.Map{"todo": existing, "already_exists": true})
	}

	var project models.Project
	s.db.Select("id, name").First(&project, *src.ProjectID)
	var desc strings.Builder
	fmt.Fprintf(&desc, "%s\n\n**Langkah:** %s\n\n**Bukti traffic live (%s, sumber \"%s\"):**\n", rec.Detail, rec.Action, rep.Window, src.Name)
	for _, e := range rec.Evidence {
		fmt.Fprintf(&desc, "- %s\n", e)
	}
	if len(rec.Routes) > 0 {
		fmt.Fprintf(&desc, "\n**Rute:** %s\n", strings.Join(rec.Routes, ", "))
	}
	category := trafficTodoCategory[rec.Category]
	if category == "" {
		category = models.CategoryGeneral
	}
	todo := models.Todo{
		Level: models.TodoLevelProject, ProjectID: src.ProjectID, ProjectName: project.Name,
		Title: clipString(rec.Title, 500), Description: desc.String(), Status: models.TodoStatusSuggested,
		Priority: trafficTodoPriority[rec.Severity], Category: category,
		CreatorType: models.CreatorAI, CreatorName: "Traffic Analyzer", IsAISuggestion: true,
		GuidelineRef: rec.Rule, Source: "traffic_analyzer", AuditKey: key,
		Rationale: "Disimpulkan dari statistik request nyata aplikasi live, bukan tebakan dari kode.",
	}
	if src.ComponentID != nil {
		var comp models.Component
		if s.db.Select("id, name").First(&comp, *src.ComponentID).Error == nil {
			todo.Level = models.TodoLevelComponent
			todo.ComponentID = src.ComponentID
			todo.ComponentName = comp.Name
		}
	}
	if err := s.db.Create(&todo).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"todo": todo, "already_exists": false})
}

// ingestTrafficDirect: mode direct — BE (dev/staging) mengirim langsung ke engine ini tanpa lewat vibeasik live.
func (s *Server) ingestTrafficDirect(c *fiber.Ctx) error {
	key := strings.TrimSpace(c.Get("X-Vibeasik-Key"))
	var src models.TrafficSource
	if key == "" || s.db.Where("mode = ? AND ingest_key = ?", models.TrafficModeDirect, key).First(&src).Error != nil {
		return c.Status(401).JSON(fiber.Map{"error": "kunci traffic tidak valid"})
	}
	if !src.Enabled {
		return c.Status(202).JSON(fiber.Map{"accepted": 0, "paused": true})
	}
	var req struct {
		Instance string          `json:"instance"`
		Events   []traffic.Event `json:"events"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if len(req.Events) > traffic.MaxEventsPerIngest {
		return c.Status(413).JSON(fiber.Map{"error": "maksimal 5000 event per kiriman"})
	}
	n, err := traffic.Ingest(s.db, &src, req.Events, req.Instance)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	now := time.Now()
	s.db.Model(&src).Updates(map[string]any{"last_ingest_at": now, "last_sync_at": now})
	return c.Status(202).JSON(fiber.Map{"accepted": n})
}

func clipString(v string, n int) string {
	if len(v) > n {
		return v[:n]
	}
	return v
}
