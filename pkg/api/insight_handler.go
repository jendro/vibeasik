package api

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/insight"
)

// getProcessCost: rumus tick T(n), Big-O, dan jumlah query Q(n) tiap rute/fungsi backend Go proyek.
func (s *Server) getProcessCost(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	page, err := insight.ProcessCost(s.db, uint(id), insight.CostQuery{
		Kind: c.Query("kind"), Search: c.Query("q"), Class: c.Query("class"),
		NPlusOne: c.Query("n_plus_one") == "1", Limit: c.QueryInt("limit"), Offset: c.QueryInt("offset"),
	}, time.Now())
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(page)
}

// getOwnerRecap: rekap kemajuan proyek untuk pemilik (todo, bug ditutup, perbaikan algoritma, kesehatan kode).
func (s *Server) getOwnerRecap(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	recap, err := insight.BuildOwnerRecap(s.db, uint(id), c.QueryInt("days"), time.Now())
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(recap)
}

// createProcessCostTodo: satu sumber biaya tinggi di tab Biaya Proses dijadikan todo (kunci audit sama dengan
// temuan audit, jadi audit berikutnya menggabung ke todo ini). Todo terbuka yang sudah ada dikembalikan apa adanya.
func (s *Server) createProcessCostTodo(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	var req struct {
		File string `json:"file"`
		Line int    `json:"line"`
		Kind string `json:"kind"`
	}
	if msg := firstProblem(bindJSONRequired(c, &req), needText("file", req.File, maxPath), oneOf("kind", req.Kind, "nplus1", "growth")); msg != "" {
		return badPayload(c, msg)
	}
	todo, existed, err := insight.CreateCostTodo(s.db, uint(id), req.File, req.Line, req.Kind)
	switch {
	case errors.Is(err, insight.ErrCostSourceGone):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case err != nil:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	case existed:
		return c.JSON(fiber.Map{"todo": todo, "already_exists": true})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"todo": todo, "already_exists": false})
}

// createAllProcessCostTodos: "Jadikan semua todo" di halaman Biaya Proses — semua sumber biaya yang belum punya
// todo terbuka, satu todo per berkas (tanpa LLM).
func (s *Server) createAllProcessCostTodos(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	res, err := insight.CreateAllCostTodos(s.db, uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error(), "partial": res})
	}
	return c.JSON(res)
}

// getProcessCostSource: potongan berkas (baris from..to) untuk tampilan editor bukti biaya.
func (s *Server) getProcessCostSource(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	file := c.Query("file")
	if msg := needText("file", file, maxPath); msg != "" {
		return badPayload(c, msg)
	}
	win, err := insight.CostSourceLines(s.db, uint(id), file, c.QueryInt("from", 1), c.QueryInt("to", 1))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(win)
}

// getFindingsTodoPrompt: prompt siap salin untuk agent coding — menjadikan temuan satu sumber (audit, cost,
// runtime) todo lewat MCP vibeasik tanpa LLM luar. Teksnya sama dengan tool MCP vibeasik_findings_todo_prompt.
func (s *Server) getFindingsTodoPrompt(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	res, err := insight.BuildFindingsTodoPrompt(s.db, uint(id), c.Query("scope"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}
