package api

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/update"
)

// Pembaruan aplikasi: Studio menampilkan banner bila ada versi baru, memasangnya sekali klik,
// lalu memulai ulang engine.
func (s *Server) setupUpdateRoutes(api fiber.Router) {
	api.Get("/update", s.getUpdate)
	api.Post("/update/check", s.checkUpdate)
	api.Post("/update/apply", s.applyUpdate)
	api.Post("/update/restart", s.restartEngine)
	api.Put("/update/auto", s.setUpdateAuto)
}

// SetUpdater memasang updater (diisi perintah serve).
func (s *Server) SetUpdater(u *update.Updater) { s.updater = u }

func (s *Server) updateView(info update.Info) fiber.Map {
	return fiber.Map{"info": info, "auto": update.Auto(), "restart_supported": s.restart != nil}
}

func (s *Server) getUpdate(c *fiber.Ctx) error {
	if s.updater == nil {
		return c.JSON(fiber.Map{"info": update.Info{Reason: "pembaruan tidak tersedia"}, "auto": false})
	}
	info := s.updater.Last()
	if time.Since(info.CheckedAt) > time.Hour {
		info = s.updater.Check(c.Context())
	}
	return c.JSON(s.updateView(info))
}

func (s *Server) checkUpdate(c *fiber.Ctx) error {
	if s.updater == nil {
		return c.Status(501).JSON(fiber.Map{"error": "pembaruan tidak tersedia"})
	}
	return c.JSON(s.updateView(s.updater.Check(c.Context())))
}

func (s *Server) applyUpdate(c *fiber.Ctx) error {
	if s.updater == nil {
		return c.Status(501).JSON(fiber.Map{"error": "pembaruan tidak tersedia"})
	}
	// Unduhan bisa lebih lama dari WriteTimeout bawaan: jalankan dengan konteks sendiri.
	if _, err := s.updater.Apply(c.Context()); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(s.updateView(s.updater.Last()))
}

func (s *Server) setUpdateAuto(c *fiber.Ctx) error {
	var in struct {
		Auto bool `json:"auto"`
	}
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "body tidak valid"})
	}
	if err := update.SetAuto(in.Auto); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"auto": in.Auto})
}
