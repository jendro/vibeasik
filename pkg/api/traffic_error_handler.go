package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/traffic"
)

// Grup error & status proses (heartbeat) per sumber traffic. Akses dijaga localAccess global.

func (s *Server) listTrafficErrors(c *fiber.Ctx) error {
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	rows, more, qErr := traffic.ErrorGroups(s.db, src.ID, c.Query("status"), c.QueryInt("limit", 50), c.QueryInt("offset", 0))
	if qErr != nil {
		return c.Status(500).JSON(fiber.Map{"error": qErr.Error()})
	}
	return c.JSON(fiber.Map{"groups": rows, "has_more": more})
}

// getTrafficError = detail satu grup, termasuk stack contoh (kolom berat tidak ikut di daftar).
func (s *Server) getTrafficError(c *fiber.Ctx) error {
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	gid, bad := paramID(c, "gid", "grup error")
	if bad != "" {
		return badPayload(c, bad)
	}
	var g models.TrafficErrorGroup
	if s.db.Where("source_id = ? AND id = ?", src.ID, gid).First(&g).Error != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Grup error tidak ditemukan"})
	}
	return c.JSON(g)
}

func (s *Server) listTrafficProcesses(c *fiber.Ctx) error {
	src, err := s.trafficSource(c)
	if src == nil {
		return err
	}
	rows, qErr := traffic.Processes(s.db, src.ID)
	if qErr != nil {
		return c.Status(500).JSON(fiber.Map{"error": qErr.Error()})
	}
	return c.JSON(fiber.Map{"processes": rows})
}
