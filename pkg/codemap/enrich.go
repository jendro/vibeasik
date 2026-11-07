package codemap

import (
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// EnrichCandidate adalah berkas yang belum (atau sudah usang) ringkasan LLM-nya, beserta bahan prompt.
type EnrichCandidate struct {
	Path        string
	AbsPath     string
	Kind        string
	Lang        string
	Summary     string
	Symbols     []string
	Routes      []string
	UsedBy      int
	ContentHash string
	Head        string // awal berkas (dipotong) sebagai konteks
}

// EnrichCandidates memilih berkas tanpa ringkasan LLM terbaru; modul yang paling banyak dipakai didahulukan,
// test & berkas konfigurasi kecil dilewati.
func EnrichCandidates(db *gorm.DB, projectID uint, limit int) ([]EnrichCandidate, int, error) {
	if err := EnsureFresh(db, projectID, StaleAfter); err != nil {
		return nil, 0, err
	}
	ix, err := loadIndex(db, projectID)
	if err != nil {
		return nil, 0, err
	}
	var out []EnrichCandidate
	for _, d := range ix.docs {
		e := d.entry
		if e.AIHash == e.ContentHash && e.AISummary != "" || e.Kind == "test" || e.Kind == "style" || e.Lines < 5 {
			continue
		}
		c := EnrichCandidate{Path: e.Path, AbsPath: e.AbsPath, Kind: e.Kind, Lang: e.Lang, Summary: e.Summary, Routes: d.routes,
			UsedBy: len(ix.dependents[e.Path]), ContentHash: e.ContentHash}
		for _, s := range d.symbols {
			if len(c.Symbols) < 15 {
				c.Symbols = append(c.Symbols, s.Kind+" "+s.Name)
			}
		}
		if len(c.Routes) > 6 {
			c.Routes = c.Routes[:6]
		}
		out = append(out, c)
	}
	total := len(out)
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].Summary == "") != (out[j].Summary == "") {
			return out[i].Summary == ""
		}
		return out[i].UsedBy > out[j].UsedBy
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	for i := range out {
		head, _ := readLines(out[i].AbsPath, 1, 40)
		if len(head) > 1500 {
			head = head[:1500]
		}
		out[i].Head = head
	}
	return out, total, nil
}

// SaveEnrichment menyimpan ringkasan & kata kunci LLM untuk satu berkas (hanya bila isinya belum berubah).
func SaveEnrichment(db *gorm.DB, projectID uint, path, contentHash, summary string, keywords []string) bool {
	summary = truncate(strings.TrimSpace(summary), 300)
	if summary == "" {
		return false
	}
	var kw []string
	for _, k := range keywords {
		if k = strings.TrimSpace(k); k != "" && len(kw) < 12 {
			kw = append(kw, k)
		}
	}
	res := db.Model(&models.CodeMapEntry{}).Where("project_id = ? AND path = ? AND content_hash = ?", projectID, path, contentHash).
		Updates(map[string]interface{}{"ai_summary": summary, "keywords": strings.Join(kw, ", "), "ai_hash": contentHash})
	return res.Error == nil && res.RowsAffected > 0
}

// RefreshEnrichedCount memperbarui jumlah berkas berringkasan LLM dan membuang indeks memori agar ringkasan baru ikut dicari.
func RefreshEnrichedCount(db *gorm.DB, projectID uint) {
	var n int64
	db.Model(&models.CodeMapEntry{}).Where("project_id = ? AND ai_summary <> '' AND ai_hash = content_hash", projectID).Count(&n)
	db.Model(&models.CodeMap{}).Where("project_id = ?", projectID).Update("enriched_count", n)
	invalidate(projectID)
}
