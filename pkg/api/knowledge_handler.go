package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/knowledge"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// Pusat Pengetahuan (satu pintu): ringkasan, daftar & detail node, uji temu-kembali, umpan balik, kurasi, celah.

func (s *Server) getKnowledgeOverview(c *fiber.Ctx) error {
	return c.JSON(knowledge.GetOverview(s.db))
}

func (s *Server) getKnowledgeGraph(c *fiber.Ctx) error {
	return c.JSON(knowledge.GetGraph(s.db, c.QueryInt("limit")))
}

func (s *Server) listKnowledgeNodes(c *fiber.Ctx) error {
	return c.JSON(knowledge.List(s.db, knowledge.Filter{
		Text: c.Query("q"), Domain: c.Query("domain"), Kind: c.Query("kind"), Source: c.Query("source"),
		ClusterID: uint(c.QueryInt("cluster_id")), Status: c.Query("status"), Limit: c.QueryInt("limit"),
	}))
}

func (s *Server) getKnowledgeNode(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid node id"})
	}
	d, err := knowledge.GetDetail(s.db, uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "node tidak ditemukan"})
	}
	return c.JSON(d)
}

// teachKnowledge menambah pengetahuan ke pustaka bersama kantor (diajarkan manusia).
func (s *Server) teachKnowledge(c *fiber.Ctx) error {
	var body knowledge.Note
	if msg := firstProblem(bindJSONRequired(c, &body)); msg != "" {
		return badPayload(c, msg)
	}
	if msg := firstProblem(needText("title", body.Title, maxLabel), needText("content", body.Content, maxText),
		maxLen("domain", body.Domain, maxName), maxLen("kind", body.Kind, maxName), maxLen("owner_name", body.OwnerName, maxName)); msg != "" {
		return badPayload(c, msg)
	}
	body.Source, body.OwnerEmployeeID = models.KnowledgeSrcManual, nil
	node, merged, err := knowledge.Learn(s.db, body)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"node": node, "merged": merged})
}

func (s *Server) knowledgeFeedback(c *fiber.Ctx) error {
	id, bad := paramID(c, "id", "node")
	if bad != "" {
		return badPayload(c, bad)
	}
	var body struct {
		Helpful bool `json:"helpful"`
	}
	if msg := bindJSONRequired(c, &body); msg != "" {
		return badPayload(c, msg)
	}
	node, err := knowledge.Feedback(s.db, id, body.Helpful)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(node)
}

func (s *Server) retireKnowledgeNode(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid node id"})
	}
	if err := knowledge.Retire(s.db, uint(id), c.Query("reason")); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// recallKnowledge menampilkan apa yang akan diterima agent untuk sebuah teks pekerjaan (tanpa mencatat pemakaian).
func (s *Server) recallKnowledge(c *fiber.Ctx) error {
	var q knowledge.Query
	if msg := firstProblem(bindJSONRequired(c, &q)); msg != "" {
		return badPayload(c, msg)
	}
	if msg := firstProblem(needText("text", q.Text, maxText)); msg != "" {
		return badPayload(c, msg)
	}
	if q.Limit < 0 || q.Limit > 200 {
		return badPayload(c, "field limit harus 0..200")
	}
	q.Context = ""
	return c.JSON(knowledge.Recall(s.db, q))
}

func (s *Server) curateKnowledge(c *fiber.Ctx) error {
	return c.JSON(knowledge.Curate(s.db))
}

func (s *Server) listKnowledgeGaps(c *fiber.Ctx) error {
	return c.JSON(knowledge.ListGaps(s.db, c.Query("status"), c.QueryInt("limit")))
}

func (s *Server) reportKnowledgeGap(c *fiber.Ctx) error {
	var body struct {
		Domain string `json:"domain"`
		Topic  string `json:"topic"`
		Sample string `json:"sample"`
	}
	if msg := firstProblem(bindJSONRequired(c, &body), needText("topic", body.Topic, maxLabel),
		maxLen("domain", body.Domain, maxName), maxLen("sample", body.Sample, maxText)); msg != "" {
		return badPayload(c, msg)
	}
	g, err := knowledge.ReportGap(s.db, body.Domain, body.Topic, body.Sample)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(g)
}

func (s *Server) dismissKnowledgeGap(c *fiber.Ctx) error {
	id, bad := paramID(c, "gid", "gap")
	if bad != "" {
		return badPayload(c, bad)
	}
	if msg := maxLen("note", c.Query("note"), maxLabel); msg != "" {
		return badPayload(c, msg)
	}
	if err := knowledge.DismissGap(s.db, id, c.Query("note")); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}
