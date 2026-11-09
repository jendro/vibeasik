package api

import (
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/setup"
)

// DefaultDBName nama database bawaan untuk instalasi baru.
const DefaultDBName = "vibeasik_platform"

func currentDB() setup.DB {
	cfg := config.GetConfig()
	d := setup.DB{Driver: cfg.DBDriver, Name: cfg.DBName}
	if cfg.DBDriver == "sqlite" {
		// Nama dari berkas yang sedang dipakai (projects.db → projects).
		d.Name = strings.TrimSuffix(filepath.Base(cfg.DBPath), filepath.Ext(cfg.DBPath))
	} else {
		d.Host, d.Port, d.User = cfg.DBHost, cfg.DBPort, cfg.DBUser
	}
	return d
}

// validateSetupDB memeriksa bentuk payload database wizard (aturan koneksi sebenarnya ada di setup.Test).
func validateSetupDB(d setup.DB) string {
	return firstProblem(needOneOf("driver", strings.ToLower(strings.TrimSpace(d.Driver)), "sqlite", "mysql"), needText("name", d.Name, maxName),
		maxLen("host", d.Host, maxName), maxLen("port", d.Port, 5), maxLen("user", d.User, maxName), maxLen("password", d.Password, maxName*2))
}

func (s *Server) testSetupDatabase(c *fiber.Ctx) error {
	var in setup.DB
	if msg := firstProblem(bindJSONRequired(c, &in), validateSetupDB(in)); msg != "" {
		return badPayload(c, msg)
	}
	return c.JSON(setup.Test(in))
}

func (s *Server) saveSetupDatabase(c *fiber.Ctx) error {
	var in setup.DB
	if msg := firstProblem(bindJSONRequired(c, &in), validateSetupDB(in)); msg != "" {
		return badPayload(c, msg)
	}
	if rep := setup.Test(in); !rep.OK {
		return c.Status(400).JSON(fiber.Map{"error": rep.Message, "report": rep})
	}
	cur := currentDB()
	changed := in.Driver != cur.Driver || in.Name != cur.Name
	if in.Driver == "mysql" {
		changed = changed || in.Host != cur.Host || in.Port != cur.Port || in.User != cur.User || in.Password != ""
	}
	if changed {
		if err := setup.WriteEnv(in); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "gagal menulis " + setup.EnvPath() + ": " + err.Error()})
		}
	}
	if err := setup.MarkDatabase(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"restart_required": changed, "restart_supported": s.restart != nil, "env_file": setup.EnvPath()})
}
