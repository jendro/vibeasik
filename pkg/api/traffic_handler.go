package api

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

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
