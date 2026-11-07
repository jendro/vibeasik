package api

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/account"
	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/knowledgesync"
	"github.com/vibeasik/vibeasik/pkg/setup"
)

// Penyiapan awal: akun (opsional), Syarat & Ketentuan (wajib), dan database. Studio mengarahkan
// pengguna ke /setup sampai `complete` bernilai true.
func (s *Server) setupSetupRoutes(api fiber.Router) {
	api.Get("/setup", s.getSetup)
	api.Post("/setup/account", s.setSetupAccount)
	api.Post("/setup/terms", s.acceptSetupTerms)
	api.Post("/setup/database/test", s.testSetupDatabase)
	api.Post("/setup/database", s.saveSetupDatabase)
	api.Post("/setup/complete", s.completeSetup)
	api.Post("/setup/restart", s.restartEngine)
}

// SetRestart memasang fungsi mulai-ulang engine (diisi perintah serve). Tanpa ini wizard meminta
// pengguna menjalankan ulang engine sendiri.
func (s *Server) SetRestart(fn func() error) { s.restart = fn }

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

func setupView() fiber.Map {
	st := setup.Load()
	cur := currentDB()
	acc := account.Current()
	return fiber.Map{
		"complete": setup.Complete(knowledgesync.ConsentVersion),
		"terms": fiber.Map{
			"version":     knowledgesync.ConsentVersion,
			"accepted":    setup.TermsAccepted(knowledgesync.ConsentVersion),
			"accepted_at": st.TermsAt,
			"text":        knowledgesync.ConsentText,
		},
		"account": fiber.Map{"choice": st.AccountChoice, "logged_in": acc.LoggedIn, "email": acc.Email},
		"database": fiber.Map{
			"current":  fiber.Map{"driver": cur.Driver, "name": cur.Name, "host": cur.Host, "port": cur.Port, "user": cur.User},
			"data_dir": filepath.Dir(setup.SQLitePath("x")),
			"chosen":   !st.DatabaseAt.IsZero(),
			"requirements": fiber.Map{
				"sqlite": []string{
					"Tanpa instalasi tambahan: satu berkas di komputer ini.",
					"Cocok untuk satu pengguna; pilihan bawaan.",
					"Cadangkan dengan menyalin berkasnya.",
				},
				"mysql": []string{
					"Server MySQL 8.0+ atau MariaDB 10.5+ sudah berjalan dan bisa dijangkau dari komputer ini.",
					"User dengan hak SELECT, INSERT, UPDATE, DELETE, CREATE, ALTER, INDEX, DROP pada database tersebut.",
					"Untuk membuat database otomatis, user juga butuh hak CREATE pada server.",
					"Charset utf8mb4 (dibuat otomatis bila vibeasik yang membuat database).",
				},
			},
		},
	}
}

func (s *Server) getSetup(c *fiber.Ctx) error {
	account.Activate()
	return c.JSON(s.viewWithRestart())
}

// DefaultDBName nama database bawaan untuk instalasi baru.
const DefaultDBName = "vibeasik_platform"

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

func (s *Server) completeSetup(c *fiber.Ctx) error {
	if err := setup.Finish(knowledgesync.ConsentVersion); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(s.viewWithRestart())
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

// validateSetupDB memeriksa bentuk payload database wizard (aturan koneksi sebenarnya ada di setup.Test).
func validateSetupDB(d setup.DB) string {
	return firstProblem(needOneOf("driver", strings.ToLower(strings.TrimSpace(d.Driver)), "sqlite", "mysql"), needText("name", d.Name, maxName),
		maxLen("host", d.Host, maxName), maxLen("port", d.Port, 5), maxLen("user", d.User, maxName), maxLen("password", d.Password, maxName*2))
}
