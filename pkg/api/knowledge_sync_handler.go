package api

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/account"
	"github.com/vibeasik/vibeasik/pkg/knowledgesync"
)

// Pusat Pengetahuan: pertukaran dengan server vibeasik untuk Studio (persetujuan S&K, mode kirim, pratinjau, riwayat,
// pencabutan). Logika ada di pkg/knowledgesync; handler ini hanya membungkusnya.
func (s *Server) setupKnowledgeSyncRoutes(api fiber.Router) {
	api.Get("/knowledge-sync", s.getKnowledgeSync)
	api.Post("/knowledge-sync/consent", s.consentKnowledgeSync)
	api.Post("/knowledge-sync/mode", s.setKnowledgeSyncMode)
	api.Get("/knowledge-sync/preview", s.previewKnowledgeSync)
	api.Post("/knowledge-sync/ledger/sync", s.syncKnowledgeLedger)
	api.Post("/knowledge-sync/revoke", s.revokeKnowledgeSync)
	api.Post("/knowledge-sync/pull", s.pullKnowledgeSync)
}

func knowledgeSyncView() fiber.Map {
	st := knowledgesync.LoadState()
	l := knowledgesync.LoadLedger()
	sent, skipped := l.Sent, l.Skipped
	if sent == nil {
		sent = []knowledgesync.LedgerSent{}
	}
	if skipped == nil {
		skipped = []knowledgesync.LedgerSkip{}
	}
	return fiber.Map{
		"mode":        knowledgesync.EffectiveMode(),
		"can_control": knowledgesync.CanControl(),
		"consent": fiber.Map{
			"accepted": knowledgesync.HasConsent(), "version": knowledgesync.ConsentVersion,
			"accepted_version": st.ConsentVersion, "accepted_at": st.ConsentAt,
		},
		"consent_text": knowledgesync.ConsentText,
		"logged_in":    account.DeviceToken() != "",
		"pack":         fiber.Map{"version": st.Version, "tier": st.Tier, "nodes": st.Nodes, "pulled_at": st.PulledAt, "license": st.License},
		"limits":       fiber.Map{"max_per_day": knowledgesync.MaxPerDay, "min_helpful": knowledgesync.MinHelpful},
		"ledger":       fiber.Map{"sent": sent, "skipped": skipped, "sent_today": l.SentOn(time.Now())},
	}
}

func (s *Server) getKnowledgeSync(c *fiber.Ctx) error {
	account.Activate() // login/logout lewat CLI mengubah berkas akun saat engine berjalan
	return c.JSON(knowledgeSyncView())
}

func (s *Server) consentKnowledgeSync(c *fiber.Ctx) error {
	var in struct {
		Accept  bool   `json:"accept"`
		Version string `json:"version"`
	}
	if p := bindJSONRequired(c, &in); p != "" {
		return badPayload(c, p)
	}
	if !in.Accept {
		return badPayload(c, "persetujuan harus eksplisit (accept: true)")
	}
	if p := needText("version", in.Version, maxName); p != "" {
		return badPayload(c, p)
	}
	// Pengguna menyetujui teks versi tertentu; tolak bila tampilan Studio sudah usang.
	if in.Version != knowledgesync.ConsentVersion {
		return c.Status(409).JSON(fiber.Map{"error": "versi ketentuan sudah berubah; muat ulang halaman", "version": knowledgesync.ConsentVersion})
	}
	if err := knowledgesync.Consent(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(knowledgeSyncView())
}

func (s *Server) setKnowledgeSyncMode(c *fiber.Ctx) error {
	var in struct {
		Mode string `json:"mode"`
	}
	if p := firstProblem(bindJSONRequired(c, &in), needOneOf("mode", in.Mode, "auto", "ask", "off")); p != "" {
		return badPayload(c, p)
	}
	if err := knowledgesync.SetMode(knowledgesync.Mode(in.Mode)); err != nil {
		if errors.Is(err, knowledgesync.ErrLocked) {
			return c.Status(403).JSON(fiber.Map{"error": err.Error(), "code": "locked"})
		}
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(knowledgeSyncView())
}

func (s *Server) previewKnowledgeSync(c *fiber.Ctx) error {
	el := knowledgesync.AutoEligible(s.db, time.Now())
	ok, skipped := el.OK, el.Skipped
	if ok == nil {
		ok = []knowledgesync.AutoCandidate{}
	}
	if skipped == nil {
		skipped = []knowledgesync.LedgerSkip{}
	}
	return c.JSON(fiber.Map{"will_send": ok, "skipped": skipped, "max_per_day": knowledgesync.MaxPerDay})
}

func (s *Server) syncKnowledgeLedger(c *fiber.Ctx) error {
	n, err := knowledgesync.SyncLedger()
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	v := knowledgeSyncView()
	v["updated"] = n
	return c.JSON(v)
}

func (s *Server) revokeKnowledgeSync(c *fiber.Ctx) error {
	var in struct {
		ServerID uint `json:"server_id"`
	}
	if p := firstProblem(bindJSONRequired(c, &in), needID("server_id", in.ServerID)); p != "" {
		return badPayload(c, p)
	}
	if err := knowledgesync.Revoke(in.ServerID); err != nil {
		return c.Status(502).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(knowledgeSyncView())
}

func (s *Server) pullKnowledgeSync(c *fiber.Ctx) error {
	res, err := knowledgesync.Pull(s.db, false)
	if err != nil {
		code := 502
		if errors.Is(err, knowledgesync.ErrConsentRequired) {
			code = 403
		}
		return c.Status(code).JSON(fiber.Map{"error": err.Error()})
	}
	v := knowledgeSyncView()
	v["pull"] = res
	return c.JSON(v)
}
