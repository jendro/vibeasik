package api

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/codemap"
)

// Peta kode (gaya RAG) per proyek untuk AI agent; build tidak memakai LLM sehingga cukup cepat untuk sinkron.
// Ringkasan LLM per berkas lewat antrean: POST /api/queue/enqueue {type: "code_map_enrich", project_id}.

func (s *Server) getCodeMap(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	ov, err := codemap.BuildOverview(s.db, uint(id), uint(c.QueryInt("component_id")), c.Query("detail") == "files")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(ov)
}

func (s *Server) buildCodeMap(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	var body struct {
		Force bool `json:"force"`
	}
	_ = c.BodyParser(&body)
	res, err := codemap.Build(s.db, uint(id), body.Force)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

func (s *Server) searchCodeMap(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	q := c.Query("q")
	if q == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "parameter q wajib diisi"})
	}
	snippet, _ := strconv.ParseBool(c.Query("snippet", "false"))
	hits, err := codemap.Search(s.db, uint(id), codemap.SearchOptions{Query: q, Kind: c.Query("kind"),
		ComponentID: uint(c.QueryInt("component_id")), Limit: c.QueryInt("limit", 10), Snippet: snippet})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"query": q, "hits": hits})
}

func (s *Server) getCodeMapFile(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	fd, candidates, err := codemap.File(s.db, uint(id), c.Query("path"))
	switch {
	case err != nil:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	case len(candidates) > 1:
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "path ambigu, pilih salah satu", "candidates": candidates})
	case fd == nil:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "berkas tidak ada di peta kode"})
	}
	return c.JSON(fd)
}

func (s *Server) exportCodeMap(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	res, err := codemap.Export(s.db, uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}
