package api

import (
	"errors"
	"os"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/account"
	"github.com/vibeasik/vibeasik/pkg/registry"
)

// Akun vibeasik untuk Studio: status, login perangkat (kode ditampilkan di Studio), keluar, perbarui.
func (s *Server) setupAccountRoutes(api fiber.Router) {
	api.Get("/account", s.getAccount)
	api.Post("/account/login", s.startAccountLogin)
	api.Post("/account/login/poll", s.pollAccountLogin)
	api.Post("/account/refresh", s.refreshAccount)
	api.Post("/account/logout", s.logoutAccount)
}

func accountView() fiber.Map {
	return fiber.Map{"account": account.Current(), "edition": registry.Edition, "features": registry.Features()}
}

func (s *Server) getAccount(c *fiber.Ctx) error { return c.JSON(accountView()) }

func (s *Server) startAccountLogin(c *fiber.Ctx) error {
	host, _ := os.Hostname()
	dc, err := account.StartLogin("vibeasik " + registry.Edition + " · " + host + " (Studio)")
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(dc)
}

func (s *Server) pollAccountLogin(c *fiber.Ctx) error {
	var in struct {
		DeviceCode string `json:"device_code"`
	}
	if err := c.BodyParser(&in); err != nil || in.DeviceCode == "" {
		return c.Status(400).JSON(fiber.Map{"error": "device_code wajib"})
	}
	if _, err := account.PollLogin(in.DeviceCode); err != nil {
		if errors.Is(err, account.ErrPending) {
			return c.JSON(fiber.Map{"status": "pending"})
		}
		return c.Status(400).JSON(fiber.Map{"status": "failed", "error": err.Error()})
	}
	v := accountView()
	v["status"] = "done"
	return c.JSON(v)
}

func (s *Server) refreshAccount(c *fiber.Ctx) error {
	if err := account.Refresh(); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(accountView())
}

func (s *Server) logoutAccount(c *fiber.Ctx) error {
	if err := account.Logout(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(accountView())
}
