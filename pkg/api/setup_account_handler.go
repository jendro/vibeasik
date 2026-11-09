package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/knowledgesync"
	"github.com/vibeasik/vibeasik/pkg/setup"
)

func (s *Server) setSetupAccount(c *fiber.Ctx) error {
	var in struct {
		Choice string `json:"choice"`
	}
	if msg := firstProblem(bindJSONRequired(c, &in), needOneOf("choice", in.Choice, "register", "login", "skip")); msg != "" {
		return badPayload(c, msg)
	}
	if err := setup.SetAccountChoice(in.Choice); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(s.viewWithRestart())
}

func (s *Server) acceptSetupTerms(c *fiber.Ctx) error {
	var in struct {
		Version string `json:"version"`
		Accept  bool   `json:"accept"`
	}
	if msg := bindJSONRequired(c, &in); msg != "" {
		return badPayload(c, msg)
	}
	if !in.Accept {
		return badPayload(c, "persetujuan wajib dicentang")
	}
	if msg := needText("version", in.Version, maxName); msg != "" {
		return badPayload(c, msg)
	}
	if in.Version != knowledgesync.ConsentVersion {
		return c.Status(409).JSON(fiber.Map{"error": "versi ketentuan sudah berubah; muat ulang halaman", "version": knowledgesync.ConsentVersion})
	}
	// Satu persetujuan mencakup S&K dan ketentuan Pusat Pengetahuan lokal.
	if err := knowledgesync.Consent(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if err := setup.AcceptTerms(in.Version); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(s.viewWithRestart())
}
