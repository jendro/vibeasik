package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// SetRestart memasang fungsi mulai-ulang engine (diisi perintah serve). Tanpa ini wizard meminta
// pengguna menjalankan ulang engine sendiri.
func (s *Server) SetRestart(fn func() error) { s.restart = fn }

func (s *Server) viewWithRestart() fiber.Map {
	v := setupView()
	v["restart_supported"] = s.restart != nil
	// Instalasi baru (belum ada proyek) disarankan memakai nama bawaan; yang sudah berisi data tetap memakai nama sekarang.
	var n int64
	s.db.Table("projects").Count(&n)
	db := v["database"].(fiber.Map)
	db["has_data"] = n > 0
	db["suggested_name"] = DefaultDBName
	return v
}

func (s *Server) restartEngine(c *fiber.Ctx) error {
	if s.restart == nil {
		return c.Status(501).JSON(fiber.Map{"error": "mulai ulang otomatis tidak tersedia di sistem ini; hentikan lalu jalankan `vibeasik serve` lagi"})
	}
	go func() {
		time.Sleep(400 * time.Millisecond) // beri waktu jawaban terkirim
		_ = s.restart()
	}()
	return c.JSON(fiber.Map{"restarting": true})
}
