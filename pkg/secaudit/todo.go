package secaudit

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// TodoBatch = hasil "Jadikan semua todo" di halaman Uji Keamanan.
type TodoBatch struct {
	Created int           `json:"created"`
	Existed int           `json:"existed"` // sudah punya todo terbuka
	Todos   []models.Todo `json:"todos"`
}

// NeedsTodo: pemeriksaan yang gagal atau lemah perlu diperbaiki; pass/declared/skipped tidak.
func NeedsTodo(c Check) bool { return c.Status == StatusFail || c.Status == StatusWarn }

var sevPriority = map[string]string{SevCritical: models.TodoPriorityHigh, SevWarning: models.TodoPriorityMedium, SevInfo: models.TodoPriorityLow}

// CreateTodos menjadikan setiap pemeriksaan gagal/lemah di laporan satu todo keamanan. Kunci audit
// "SECTEST|<proyek>|<id pemeriksaan>" mencegah todo ganda: pemeriksaan yang masih punya todo terbuka dilewati.
// Tanpa LLM — isi todo diambil dari bukti dan saran pemeriksaan itu sendiri.
func CreateTodos(db *gorm.DB, rep *Report) (*TodoBatch, error) {
	var project models.Project
	if err := db.Select("id", "name").First(&project, rep.ProjectID).Error; err != nil {
		return nil, fmt.Errorf("proyek %d tidak ditemukan", rep.ProjectID)
	}
	out := &TodoBatch{}
	for _, c := range rep.Checks {
		if !NeedsTodo(c) {
			continue
		}
		key := fmt.Sprintf("SECTEST|%d|%s", rep.ProjectID, c.ID)
		var existing models.Todo
		if db.Where("audit_key = ? AND status NOT IN ?", key, []string{models.TodoStatusCompleted, models.TodoStatusRejected}).
			First(&existing).Error == nil {
			out.Existed++
			continue
		}
		priority := sevPriority[c.Severity]
		if priority == "" {
			priority = models.TodoPriorityMedium
		}
		if c.Status == StatusFail && priority == models.TodoPriorityMedium {
			priority = models.TodoPriorityHigh
		}
		pid := project.ID
		todo := models.Todo{
			Level: models.TodoLevelProject, ProjectID: &pid, ProjectName: project.Name,
			Title: "Keamanan: " + c.Title, Description: todoDescription(c, rep.Target),
			Status: models.TodoStatusPending, Priority: priority, Category: models.CategorySecurity,
			CreatorType: models.CreatorHuman, CreatorName: "Uji Keamanan",
			GuidelineRef: c.GuidelineID, Source: "security_test", AuditKey: key, Tags: models.TagKeamananDasar,
			Rationale: "Dari Uji Keamanan: pemeriksaan kode router dan respons server yang berjalan.",
		}
		if err := db.Create(&todo).Error; err != nil {
			return out, err
		}
		out.Created++
		out.Todos = append(out.Todos, todo)
	}
	return out, nil
}

func todoDescription(c Check, target string) string {
	var b strings.Builder
	status := "gagal"
	if c.Status == StatusWarn {
		status = "lemah / sebagian"
	}
	fmt.Fprintf(&b, "Hasil Uji Keamanan: **%s** (%s, sumber %s).\n", status, c.Category, c.Source)
	if target != "" {
		fmt.Fprintf(&b, "Sasaran uji: %s\n", target)
	}
	if c.Suggestion != "" {
		fmt.Fprintf(&b, "\n**Langkah:** %s\n", c.Suggestion)
	}
	if len(c.Evidence) > 0 {
		b.WriteString("\n**Bukti:**\n")
		for _, e := range c.Evidence {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}
	if len(c.Locations) > 0 {
		b.WriteString("\n**Lokasi kode:**\n")
		for _, l := range c.Locations {
			fmt.Fprintf(&b, "- `%s:%d`", l.File, l.Line)
			if l.Snippet != "" {
				fmt.Fprintf(&b, " — `%s`", strings.TrimSpace(l.Snippet))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\nSetelah diperbaiki, jalankan ulang Uji Keamanan untuk membuktikan pemeriksaan ini lulus.")
	return b.String()
}
