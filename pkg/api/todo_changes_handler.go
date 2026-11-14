package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/todochange"
)

// getTodoChanges: commit hasil todo dibaca ulang dari git — berkas, fungsi yang berubah, dan diff per bagian —
// agar pengguna bisa memeriksa apa yang benar-benar dikirim. Todo selesai yang belum punya commit tercatat dicari
// commit squash-nya saat itu juga (mengisi todo lama); sub todo gabungan memakai commit induknya.
func (s *Server) getTodoChanges(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid todo id"})
	}
	var todo models.Todo
	if err := s.db.First(&todo, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "todo not found"})
	}
	src := s.commitSource(&todo)
	if src.CommitSHA == "" {
		reason := "Todo ini belum selesai, jadi belum ada commit hasilnya."
		if todo.Status == models.TodoStatusCompleted {
			reason = "Commit hasil todo ini tidak ditemukan: belum ada commit squash bertrailer `" + todochange.SquashTrailer +
				todochange.BranchPrefix(todo.ID) + "…` di repo proyek, dan agent tidak melaporkan commit_sha."
		}
		return c.JSON(fiber.Map{"commit": nil, "reason": reason})
	}
	ch, err := todochange.Read(src.CommitRepo, src.CommitSHA)
	if err != nil {
		return c.JSON(fiber.Map{"commit": nil, "sha": src.CommitSHA,
			"reason": "Commit " + src.CommitSHA + " tercatat tetapi tidak bisa dibaca dari repo-nya (repo dipindah atau riwayat ditulis ulang)."})
	}
	out := fiber.Map{"commit": ch, "functions": ch.SortedFunctions()}
	if src.ID != todo.ID {
		out["via_todo_id"] = src.ID
	}
	return c.JSON(out)
}

// commitSource: todo yang commit-nya dipakai — todo itu sendiri (dicari bila belum tercatat), atau induk gabungannya.
func (s *Server) commitSource(todo *models.Todo) *models.Todo {
	cur := todo
	for hop := 0; hop < 3; hop++ {
		if cur.CommitSHA == "" && cur.Status == models.TodoStatusCompleted {
			_, _ = todochange.Attach(s.db, cur, "")
		}
		if cur.CommitSHA != "" || cur.ParentID == nil {
			return cur
		}
		var parent models.Todo
		if s.db.First(&parent, *cur.ParentID).Error != nil {
			return cur
		}
		cur = &parent
	}
	return cur
}
