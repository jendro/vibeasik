package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Guard todo sejenis yang pernah ditolak: dipanggil dari Todo.BeforeCreate sehingga berlaku untuk SEMUA jalur
// pembuatan todo (audit, pegawai, saran AI, rapat PM, dewan eksekutif, MCP). Todo baru yang sejenis dengan todo
// ditolak di proyek yang sama tetap disimpan (agar pemanggil tidak mengulang/menganggap gagal) tetapi langsung
// berstatus rejected dengan rujukan ke todo lama, jadi tidak pernah masuk daftar aktif atau dikirim ke agent.
// Todo buatan manusia (creator_type human) tidak disaring: manusia boleh membuka ulang topik secara sadar.

// RejectSimilarityThreshold = ambang Jaccard token judul untuk dianggap sejenis.
const RejectSimilarityThreshold = 0.6

var (
	reTargetAppsFile = regexp.MustCompile(`\(([^():]+):L\d+\)`)
	reTitleToken     = regexp.MustCompile(`[a-z0-9_.]+`)
	// kata kerja & kata sambung umum di judul todo — tidak membedakan topik
	titleStopwords = map[string]bool{
		"dan": true, "atau": true, "pada": true, "untuk": true, "yang": true, "di": true, "ke": true, "dari": true,
		"dengan": true, "seluruh": true, "semua": true, "secara": true, "menjadi": true, "agar": true, "tanpa": true,
		"amankan": true, "tambahkan": true, "perkuat": true, "perbaiki": true, "refactor": true, "turunkan": true,
		"optimalkan": true, "implementasikan": true, "implementasi": true, "pecah": true, "buat": true, "bangun": true,
		"kurangi": true, "terapkan": true, "perketat": true, "the": true, "a": true, "of": true, "to": true, "and": true,
	}
)

// rejectedRef = ringkasan todo ditolak yang dipakai pembanding.
type rejectedRef struct {
	ID           uint
	Title        string
	AuditKey     string
	GuidelineRef string
	TargetApps   string
}

func titleTokens(title string) map[string]bool {
	out := map[string]bool{}
	for _, tok := range reTitleToken.FindAllString(strings.ToLower(title), -1) {
		if len(tok) > 1 && !titleStopwords[tok] {
			out[tok] = true
		}
	}
	return out
}

// TitleSimilarity = Jaccard token judul (tanpa stopword), 0..1.
func TitleSimilarity(a, b string) float64 {
	ta, tb := titleTokens(a), titleTokens(b)
	if len(ta) < 2 || len(tb) < 2 {
		return 0
	}
	inter := 0
	for k := range ta {
		if tb[k] {
			inter++
		}
	}
	return float64(inter) / float64(len(ta)+len(tb)-inter)
}

// todoTargetFile: berkas target todo dari audit_key, target_apps "(berkas:L12)", atau judul/deskripsi.
func todoTargetFile(auditKey, targetApps string) string {
	if i := strings.Index(auditKey, "|"); i != -1 {
		return strings.TrimSpace(auditKey[i+1:])
	}
	if m := reTargetAppsFile.FindStringSubmatch(targetApps); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// sameTarget: dua berkas dianggap sama bila salah satunya akhiran path yang lain (routes.go ~ routes/routes.go).
func sameTarget(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return a == b || strings.HasSuffix(a, "/"+b) || strings.HasSuffix(b, "/"+a)
}

// similarRejected mengembalikan todo ditolak yang sejenis dengan t beserta alasannya.
func similarRejected(t *Todo, refs []rejectedRef) (*rejectedRef, string) {
	file := todoTargetFile(t.AuditKey, t.TargetApps)
	for i := range refs {
		r := &refs[i]
		if t.AuditKey != "" && t.AuditKey == r.AuditKey {
			return r, "kunci audit sama (" + r.AuditKey + ")"
		}
		if t.GuidelineRef != "" && t.GuidelineRef == r.GuidelineRef && sameTarget(file, todoTargetFile(r.AuditKey, r.TargetApps)) {
			return r, "rule " + r.GuidelineRef + " pada berkas yang sama"
		}
	}
	for i := range refs {
		if sim := TitleSimilarity(t.Title, refs[i].Title); sim >= RejectSimilarityThreshold {
			return &refs[i], fmt.Sprintf("judul mirip %.0f%%", sim*100)
		}
	}
	return nil, ""
}

// FindSimilarRejected mencari todo ditolak di proyek t yang sejenis dengan t (kunci audit, rule+berkas, atau
// judul mirip). Dipakai juga sebelum memanggil LLM (resolver audit) agar temuan sejenis tidak diproses ulang.
func FindSimilarRejected(db *gorm.DB, t *Todo) (id uint, title, why string) {
	if db == nil || t.ProjectID == nil || *t.ProjectID == 0 {
		return 0, "", ""
	}
	var refs []rejectedRef
	if db.Session(&gorm.Session{NewDB: true}).Model(&Todo{}).
		Select("id, title, audit_key, guideline_ref, target_apps").
		Where("project_id = ? AND status = ?", *t.ProjectID, TodoStatusRejected).
		Scan(&refs).Error != nil {
		return 0, "", ""
	}
	if ref, reason := similarRejected(t, refs); ref != nil {
		return ref.ID, ref.Title, reason
	}
	return 0, "", ""
}

// applyRejectGuard menandai t ditolak bila sejenis dengan todo yang pernah ditolak di proyek yang sama.
func applyRejectGuard(tx *gorm.DB, t *Todo) {
	// Todo gabungan tidak disaring: setiap sub todonya sudah melewati penjaga ini saat dibuat.
	if t.CreatorType == "human" || t.Source == SourceTodoGroup || t.Status == TodoStatusRejected || t.Status == TodoStatusCompleted {
		return
	}
	refID, refTitle, why := FindSimilarRejected(tx, t)
	if refID == 0 {
		return
	}
	t.Status = TodoStatusRejected
	t.IsAISuggestion = false
	t.Rationale = strings.TrimSpace(t.Rationale + fmt.Sprintf(
		"\n\nDitolak otomatis (%s, kind=duplicate): sejenis dengan todo #%d \"%s\" yang sudah ditolak — %s. Buka ulang todo itu bila memang perlu dikerjakan.",
		time.Now().Format("2006-01-02"), refID, refTitle, why))
}
