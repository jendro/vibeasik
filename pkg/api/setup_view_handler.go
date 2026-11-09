package api

import (
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/account"
	"github.com/vibeasik/vibeasik/pkg/knowledgesync"
	"github.com/vibeasik/vibeasik/pkg/setup"
)

func setupView() fiber.Map {
	st := setup.Load()
	cur := currentDB()
	acc := account.Current()
	return fiber.Map{
		"complete": setup.Complete(),
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
