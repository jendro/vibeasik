package knowledge

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/models"
)

// AuditContext: kunci pemakaian node pada review audit LLM satu berkas.
func AuditContext(projectID uint, relPath string) string {
	return fmt.Sprintf("audit:%d:%s", projectID, relPath)
}

// LearnFromVerdict menjadikan vonis audit (Aturan 20) bukti & pelajaran umum di pusat:
//   - node yang disisipkan ke review LLM berkas itu dinilai (temuan benar = membantu, audit keliru = merugikan);
//   - patokan yang menjadi dasar temuan (guideline ber-ID sama, atau checklist audit yang paling mirip) diberi bukti;
//   - alasan vonis menjadi pelajaran lintas proyek (tanpa path/baris) agar reviewer & agent lain tidak mengulang.
func LearnFromVerdict(db *gorm.DB, projectID uint, f models.Finding, verdict, reason, advice string) {
	if db == nil {
		return
	}
	confirmed := verdict == models.AuditVerdictConfirmed
	if f.IsLLM || strings.HasPrefix(f.RuleID, "LLM-REVIEW-") {
		score := 35
		if confirmed {
			score = 85
		}
		Reward(db, AuditContext(projectID, f.TargetFile), score)
	}
	domain := primaryDomain(f.TargetFile)
	if id := ruleNode(db, f, domain); id != 0 {
		_, _ = Feedback(db, id, confirmed)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return
	}
	title := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(f.Title), "[LLM]"))
	n := Note{Kind: models.KnowledgeKindLesson, Domain: domain, Source: models.KnowledgeSrcVerdict, Weight: 65, Tags: []string{"audit", f.Category}}
	if confirmed {
		n.Title = "Masalah nyata: " + title
		n.Content = clip(f.Description, 400) + " Terbukti benar: " + reason
	} else {
		n.Title = "Bukan masalah: " + title
		n.Content = "Temuan audit '" + title + "' (rule " + f.RuleID + ") terbukti keliru: " + reason
		if advice = strings.TrimSpace(advice); advice != "" {
			n.Content += " Saran rule: " + advice
		}
	}
	_, _, _ = Learn(db, n)
}

// ruleNode mencari patokan dasar sebuah temuan: guideline ber-ID sama dengan rule temuan, atau (untuk review
// LLM) checklist audit ekosistemnya yang isinya paling mirip judul & deskripsi temuan.
func ruleNode(db *gorm.DB, f models.Finding, domain string) uint {
	var n models.KnowledgeNode
	if f.RuleID != "" && db.Select("id").Where("source_key = ? AND status = ?", "guideline:"+f.RuleID, models.KnowledgeNodeActive).Limit(1).Find(&n).Error == nil && n.ID != 0 {
		return n.ID
	}
	ix := current(db)
	v := ix.queryVector(f.Title, f.Description, "", nil)
	nb := ix.nearest(v, 1, 0.2, func(e *entry) bool {
		return e.source != models.KnowledgeSrcChecklist || (domain != "" && e.domain != domain)
	})
	if len(nb) == 0 {
		return 0
	}
	return nb[0].e.id
}
