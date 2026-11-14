package api

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/secaudit"
	"github.com/vibeasik/vibeasik/pkg/service"
)

// parseSecurityTestRequest membaca & memvalidasi payload uji keamanan; pesan kosong = sah.
func parseSecurityTestRequest(c *fiber.Ctx, id int) (service.SecurityTestRequest, string) {
	var req service.SecurityTestRequest
	if msg := bindJSON(c, &req); msg != "" {
		return req, msg
	}
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	req.AuthHeader = strings.TrimSpace(req.AuthHeader)
	bootProblem := ""
	if strings.ContainsAny(req.AuthHeader, "\r\n") {
		bootProblem = "auth_header tidak boleh berisi baris baru"
	}
	if req.BootTimeout < 0 || req.BootTimeout > 300 {
		bootProblem = "boot_timeout_sec harus 0–300"
	}
	if msg := firstProblem(
		maxLen("base_url", req.BaseURL, maxURL),
		maxLen("auth_header", req.AuthHeader, 4096),
		secaudit.ValidateBaseURL(req.BaseURL),
		burstProblem(req.Burst),
		bootProblem,
	); msg != "" {
		return req, msg
	}
	req.ProjectID = uint(id)
	return req, ""
}

// startProjectSecurityTest: POST /api/projects/:id/security-test/start — uji di latar seperti Uji Runtime:
// server komponen dinyalakan bila perlu, kode dibaca, server diuji, lalu dimatikan. Studio mem-poll GET.
func (s *Server) startProjectSecurityTest(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	req, msg := parseSecurityTestRequest(c, id)
	if msg != "" {
		return badPayload(c, msg)
	}
	sess, err := s.services.SecTest.Start(req)
	if errors.Is(err, service.ErrSecTestBusy) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error(), "code": "security_test_busy", "session": sess})
	}
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"session": sess})
}

// cancelProjectSecurityTest: POST /api/projects/:id/security-test/cancel — hentikan uji yang berjalan.
func (s *Server) cancelProjectSecurityTest(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	if err := s.services.SecTest.Cancel(uint(id)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// runProjectSecurityTest: POST /api/projects/:id/security-test — audit router/sumber backend proyek dan
// probe server berjalan (port lokal komponen atau base_url live) untuk kontrol keamanan standar.
func (s *Server) runProjectSecurityTest(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	req, msg := parseSecurityTestRequest(c, id)
	if msg != "" {
		return badPayload(c, msg)
	}
	rep, err := s.services.SecTest.Run(c.UserContext(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rep)
}

// getProjectSecurityTest: GET /api/projects/:id/security-test — hasil uji terakhir dan sesi uji (berjalan atau
// selesai) sejak engine menyala.
func (s *Server) getProjectSecurityTest(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	out := fiber.Map{"report": nil, "session": nil}
	if rep, ok := s.services.SecTest.Last(uint(id)); ok {
		out["report"] = rep
	}
	if sess, ok := s.services.SecTest.Session(uint(id)); ok {
		out["session"] = sess
	}
	return c.JSON(out)
}

// createSecurityTestTodos: "Jadikan semua todo" di halaman Uji Keamanan — tiap pemeriksaan gagal/lemah pada hasil
// uji terakhir menjadi satu todo keamanan (tanpa LLM); yang sudah punya todo terbuka dilewati.
func (s *Server) createSecurityTestTodos(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	rep, ok := s.services.SecTest.Last(uint(id))
	if !ok || rep == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Belum ada hasil uji keamanan; jalankan uji dulu"})
	}
	res, err := secaudit.CreateTodos(s.db, rep)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error(), "partial": res})
	}
	return c.JSON(res)
}

func burstProblem(n int) string {
	if n < 0 || n > 60 {
		return "burst harus 0–60"
	}
	return ""
}
