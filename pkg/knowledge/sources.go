package knowledge

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/guidelines"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// Source adalah penyedia butir yang dicerminkan ke pusat (sumber kebenarannya tetap di tempat asal).
// ok=false berarti sumbernya tidak terbaca saat ini — node lamanya dibiarkan, bukan dipensiunkan.
type Source struct {
	Name  string
	Notes func(db *gorm.DB) (notes []Note, ok bool)
}

var severityWeight = map[string]int{"critical": 95, "high": 85, "medium": 70, "low": 55}

// guidelineNotes: setiap butir guidelines/*.json menjadi node patokan (rule) berkunci guideline:<ID>.
func guidelineNotes(*gorm.DB) ([]Note, bool) {
	cats, err := guidelines.LoadAllGuidelines()
	if err != nil || len(cats) == 0 {
		return nil, false
	}
	var out []Note
	for _, c := range cats {
		domain := guidelineDomain(c.Category)
		for _, g := range c.Guidelines {
			var b strings.Builder
			b.WriteString(g.Rule)
			if g.Rationale != "" {
				b.WriteString(" Alasan: " + g.Rationale)
			}
			if g.InspectionCheck != "" {
				b.WriteString(" Cek: " + g.InspectionCheck)
			}
			if g.SuggestedAction != "" {
				b.WriteString(" Tindakan: " + g.SuggestedAction)
			}
			w := severityWeight[strings.ToLower(g.Severity)]
			if w == 0 {
				w = 70
			}
			out = append(out, Note{
				Key: "guideline:" + g.ID, Kind: models.KnowledgeKindRule, Domain: domain, Source: models.KnowledgeSrcGuideline,
				Title: g.ID + " " + g.Title, Content: b.String(), Weight: w, Tags: []string{c.Category, g.Level, g.Severity},
			})
		}
	}
	return out, true
}

var employeeKinds = map[string]string{"technical": models.KnowledgeKindProcedure, "process": models.KnowledgeKindProcedure}

// execDivision = staff.DivExecutive (paket staff mengimpor paket ini, jadi nilainya ditulis ulang di sini).
const execDivision = "executive"

// employeeNotes: pengetahuan aktif pegawai aktif (kecuali catatan leader tentang timnya) sebagai node pribadi.
// Pegawai dewan eksekutif dilewati: pengetahuannya cermin fakta proyek (ExecutiveKnowledge), bukan keahlian umum.
func employeeNotes(db *gorm.DB, empID uint) ([]Note, bool) {
	q := db.Where("status = ? AND division <> ?", models.EmployeeActive, execDivision)
	if empID > 0 {
		q = q.Where("id = ?", empID)
	}
	var emps []models.Employee
	if q.Select("id", "name").Find(&emps).Error != nil {
		return nil, false
	}
	names := make(map[uint]string, len(emps))
	ids := make([]uint, 0, len(emps))
	for _, e := range emps {
		names[e.ID] = e.Name
		ids = append(ids, e.ID)
	}
	if len(ids) == 0 {
		return nil, true
	}
	var ks []models.EmployeeKnowledge
	if db.Where("employee_id IN ? AND status = ? AND source <> ?", ids, models.KnowledgeActive, models.KnowledgeTeam).Find(&ks).Error != nil {
		return nil, false
	}
	out := make([]Note, 0, len(ks))
	for _, k := range ks {
		kind := employeeKinds[k.Kind]
		if kind == "" {
			kind = models.KnowledgeKindLesson
		}
		owner := k.EmployeeID
		out = append(out, Note{
			Key: fmt.Sprintf("employee:%d", k.ID), Kind: kind, Source: models.KnowledgeSrcEmployee, Title: k.Topic, Content: k.Content,
			Weight: k.Weight, Tags: []string{k.Source}, OwnerEmployeeID: &owner, OwnerName: names[k.EmployeeID],
		})
	}
	return out, true
}

func bySource(source string) func(*gorm.DB) *gorm.DB {
	return func(q *gorm.DB) *gorm.DB { return q.Where("source = ?", source) }
}

// SyncEmployee mencerminkan pengetahuan satu pegawai sebelum ia bekerja (agar butir dari review QA terakhir ikut).
func SyncEmployee(db *gorm.DB, empID uint) int {
	if db == nil || empID == 0 {
		return 0
	}
	notes, ok := employeeNotes(db, empID)
	if !ok {
		return 0
	}
	return syncSet(db, func(q *gorm.DB) *gorm.DB {
		return q.Where("source = ? AND owner_employee_id = ?", models.KnowledgeSrcEmployee, empID)
	}, notes)
}

// SyncSources mencerminkan semua sumber: guidelines, pengetahuan pegawai, dan sumber tambahan (checklist audit).
func SyncSources(db *gorm.DB, extra ...Source) int {
	all := append([]Source{
		{Name: models.KnowledgeSrcGuideline, Notes: guidelineNotes},
		{Name: models.KnowledgeSrcEmployee, Notes: func(db *gorm.DB) ([]Note, bool) { return employeeNotes(db, 0) }},
	}, extra...)
	n := 0
	for _, s := range all {
		if notes, ok := s.Notes(db); ok {
			n += syncSet(db, bySource(s.Name), notes)
		}
	}
	return n
}
