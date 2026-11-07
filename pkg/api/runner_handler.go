package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/runner"
)

// runError maps runner errors to HTTP responses; a busy port returns 409 so the UI can offer reassign.
func runError(c *fiber.Ctx, err error) error {
	var busy *runner.PortBusyError
	if errors.As(err, &busy) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error(), "code": "port_busy", "port": busy.Port})
	}
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
}

// getPortRegistry lists declared vs allocated ports of every component with live state.
func (s *Server) getPortRegistry(c *fiber.Ctx) error {
	entries, err := s.runner.Registry()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"entries": entries,
		"range":   fiber.Map{"start": runner.RangeStart, "end": runner.RangeEnd},
	})
}

// assignComponentPort sets a manual port ({"port": 4123}) or lets vibeasik pick ({"reassign": true}).
func (s *Server) assignComponentPort(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}
	var req struct {
		Port     int  `json:"port"`
		Reassign bool `json:"reassign"`
	}
	if msg := firstProblem(bindJSON(c, &req)); msg != "" {
		return badPayload(c, msg)
	}
	if req.Port < 0 || req.Port > 65535 {
		return badPayload(c, "field port harus 0..65535")
	}
	comp, err := s.runner.AssignPort(uint(id), req.Port, req.Reassign)
	if err != nil {
		return runError(c, err)
	}
	return c.JSON(fiber.Map{"component_id": comp.ID, "run_port": comp.RunPort})
}

func (s *Server) runComponent(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}
	var req struct {
		Reassign bool `json:"reassign"`
	}
	if msg := bindJSON(c, &req); msg != "" {
		return badPayload(c, msg)
	}
	st, err := s.runner.Run(uint(id), req.Reassign)
	if err != nil {
		return runError(c, err)
	}
	return c.JSON(st)
}

func (s *Server) stopComponent(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}
	st, err := s.runner.Stop(uint(id))
	if err != nil {
		return runError(c, err)
	}
	return c.JSON(st)
}

// getComponentRunStatus returns status plus the last log lines (?logs=0 to omit them).
func (s *Server) getComponentRunStatus(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid component id"})
	}
	st, err := s.runner.Status(uint(id), c.Query("logs", "1") != "0")
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(st)
}

func (s *Server) listRuns(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"runs": s.runner.List()})
}

func (s *Server) runProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	var req struct {
		Reassign bool `json:"reassign"`
	}
	if msg := bindJSON(c, &req); msg != "" {
		return badPayload(c, msg)
	}
	items, err := s.runner.RunProject(uint(id), req.Reassign)
	if err != nil {
		return runError(c, err)
	}
	return c.JSON(fiber.Map{"results": items})
}

func (s *Server) stopProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	items, err := s.runner.StopProject(uint(id))
	if err != nil {
		return runError(c, err)
	}
	return c.JSON(fiber.Map{"results": items})
}

// ensureUniquePorts mengalokasikan port unik untuk seluruh komponen proyek atau seluruh sistem bila id <= 0
func (s *Server) ensureUniquePorts(c *fiber.Ctx) error {
	// /ports/ensure-unique tanpa :id berarti semua proyek (id 0); :id yang ada harus valid.
	var id uint
	if c.Params("id") != "" {
		var bad string
		if id, bad = paramID(c, "id", "project"); bad != "" {
			return badPayload(c, bad)
		}
	}
	updated, err := s.runner.EnsureUniquePorts(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"message":       "Port unik lokal berhasil dialokasikan",
		"updated_count": len(updated),
		"components":    updated,
	})
}
