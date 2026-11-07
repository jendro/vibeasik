package models

import "gorm.io/gorm"

// TodoIndex memuat todo yang judulnya sama dengan sekumpulan saran dalam
// satu query (menggantikan query per-saran / N+1), lalu mencocokkan di memori.
type TodoIndex struct {
	byTitle map[string][]Todo
}

// NewTodoIndex mengambil semua todo berjudul sama dengan salah satu saran.
func NewTodoIndex(db *gorm.DB, suggestions []Todo) *TodoIndex {
	idx := &TodoIndex{byTitle: map[string][]Todo{}}
	seen := map[string]bool{}
	var titles []string
	for _, s := range suggestions {
		if !seen[s.Title] {
			seen[s.Title] = true
			titles = append(titles, s.Title)
		}
	}
	const chunk = 500
	for i := 0; i < len(titles); i += chunk {
		end := i + chunk
		if end > len(titles) {
			end = len(titles)
		}
		var rows []Todo
		if err := db.Where("title IN ?", titles[i:end]).Order("id ASC").Find(&rows).Error; err != nil {
			continue
		}
		for _, r := range rows {
			idx.byTitle[r.Title] = append(idx.byTitle[r.Title], r)
		}
	}
	return idx
}

// Find mengembalikan todo tertua yang cocok (judul, serta project/component bila diisi pada saran).
func (x *TodoIndex) Find(s Todo) (Todo, bool) {
	for _, t := range x.byTitle[s.Title] {
		if s.ProjectID != nil && *s.ProjectID > 0 && (t.ProjectID == nil || *t.ProjectID != *s.ProjectID) {
			continue
		}
		if s.ComponentID != nil && *s.ComponentID > 0 && (t.ComponentID == nil || *t.ComponentID != *s.ComponentID) {
			continue
		}
		return t, true
	}
	return Todo{}, false
}

// Add mendaftarkan todo yang baru dibuat agar saran duplikat berikutnya terdeteksi.
func (x *TodoIndex) Add(t Todo) {
	x.byTitle[t.Title] = append(x.byTitle[t.Title], t)
}
