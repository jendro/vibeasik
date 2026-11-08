// Package todorag mencari todo serupa dalam satu proyek (RAG ringan tanpa LLM/embedding): indeks TF-IDF dari
// judul + keterangan (tanpa blok kode) + berkas target + memori todo. Dipakai untuk (1) mencegah todo duplikat
// saat dibuat, (2) memberi agent contoh todo serupa yang pernah dikerjakan/ditolak, dan (3) list_todos dengan
// kueri — sehingga yang dikirim ke LLM hanya potongan yang relevan, bukan seluruh daftar todo.
package todorag

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vibeasik/vibeasik/pkg/util"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/knowledge"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// Ambang kosinus, dikalibrasi pada todo proyek nyata (Bimtek APSI, 438 todo). Kemiripan isi saja tidak cukup
// untuk menyebut duplikat — todo fitur yang keterangannya memakai templat sama bisa ≥ 0,65 padahal topiknya
// beda — jadi duplikat juga wajib mirip judulnya (indeks judul terpisah). ≥ RelatedSim masih satu topik.
const (
	DuplicateSim      = 0.5
	DuplicateTitleSim = 0.5
	BlockSim          = 0.75 // duplikat sekuat ini tidak dibuat ulang (pembuat otomatis dilewati, MCP perlu force)
	RelatedSim        = 0.15
	QuerySim          = 0.05 // kueri bebas yang pendek: kosinusnya kecil terhadap todo yang panjang
)

// Hit adalah satu todo serupa.
type Hit struct {
	ID          uint       `json:"id"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	Category    string     `json:"category,omitempty"`
	Score       *int       `json:"score,omitempty"`
	Memory      string     `json:"memory,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Sim         float64    `json:"similarity"`
}

type doc struct {
	id          uint
	title       string
	status      string
	category    string
	score       *int
	memory      string
	completedAt *time.Time
	targetApps  string
	componentID uint
}

type projectIndex struct {
	sig    string
	ix     *knowledge.TextIndex
	titles *knowledge.TextIndex
	docs   map[uint]doc
}

var (
	mu    sync.Mutex
	cache = map[uint]*projectIndex{}
)

func scope(db *gorm.DB, projectID uint) *gorm.DB {
	if projectID == 0 {
		return db.Model(&models.Todo{}).Where("project_id IS NULL")
	}
	return db.Model(&models.Todo{}).Where("project_id = ?", projectID)
}

// signature berubah bila todo proyek bertambah/berkurang atau ada yang diperbarui.
func signature(db *gorm.DB, projectID uint) string {
	var row struct {
		N int64
		M string
	}
	scope(db, projectID).Select("COUNT(*) AS n, COALESCE(MAX(updated_at), '') AS m").Scan(&row)
	return strconv.FormatInt(row.N, 10) + "|" + row.M
}

func indexFor(db *gorm.DB, projectID uint) *projectIndex {
	sig := signature(db, projectID)
	mu.Lock()
	defer mu.Unlock()
	if p := cache[projectID]; p != nil && p.sig == sig {
		return p
	}
	var rows []models.Todo
	scope(db, projectID).Select("id", "title", "description", "target_apps", "memory", "status", "category", "score", "completed_at", "component_id").
		Order("id").Find(&rows)
	p := &projectIndex{sig: sig, docs: make(map[uint]doc, len(rows))}
	docs := make([]knowledge.TextDoc, len(rows))
	titles := make([]knowledge.TextDoc, len(rows))
	for i, t := range rows {
		docs[i] = knowledge.TextDoc{ID: t.ID, Title: t.Title, Content: IndexText(t.Description, t.Memory), Tags: t.TargetApps}
		titles[i] = knowledge.TextDoc{ID: t.ID, Title: t.Title}
		p.docs[t.ID] = doc{id: t.ID, title: t.Title, status: t.Status, category: t.Category, score: t.Score, memory: t.Memory, completedAt: t.CompletedAt,
			targetApps: strings.ToLower(strings.TrimSpace(t.TargetApps)), componentID: deref(t.ComponentID)}
	}
	p.ix, p.titles = knowledge.NewTextIndex(docs), knowledge.NewTextIndex(titles)
	cache[projectID] = p
	return p
}

var fencedCode = regexp.MustCompile("(?s)```.*?```")

// IndexText: teks isi yang diindeks — keterangan tanpa blok kode (contoh kode menenggelamkan topik), dipotong,
// ditambah memori pengerjaan.
func IndexText(description, memory string) string {
	d := fencedCode.ReplaceAllString(description, " ")
	if r := []rune(d); len(r) > 1500 {
		d = string(r[:1500])
	}
	return d + "\n" + memory
}

// Similar mengembalikan k todo proyek paling mirip (kosinus ≥ min). exclude: id todo yang dilewati.
func Similar(db *gorm.DB, projectID uint, title, text string, k int, min float64, exclude ...uint) []Hit {
	skip := map[uint]bool{}
	for _, id := range exclude {
		skip[id] = true
	}
	return similar(indexFor(db, projectID), title, text, k, min, func(d doc) bool { return !skip[d.id] })
}

func similar(p *projectIndex, title, text string, k int, min float64, keep func(doc) bool) []Hit {
	if p.ix.Len() == 0 {
		return nil
	}
	hits := p.ix.Similar(title, IndexText(text, ""), k, min, func(id uint) bool { return !keep(p.docs[id]) })
	out := make([]Hit, 0, len(hits))
	for _, h := range hits {
		d := p.docs[h.ID]
		out = append(out, Hit{ID: d.id, Title: d.title, Status: d.status, Category: d.category, Score: d.score, Memory: d.memory, CompletedAt: d.completedAt, Sim: round(h.Sim)})
	}
	return out
}

// Duplicates: todo yang hampir pasti sama dengan calon todo c (masih terbuka, sudah selesai, atau ditolak): isi
// mirip ≥ DuplicateSim, judul mirip ≥ DuplicateTitleSim, dan sasarannya sama — todo serupa untuk komponen lain
// (Web vs Admin) atau berkas lain (admin.go vs admin_user.go) adalah pekerjaan berbeda, bukan duplikat.
func Duplicates(db *gorm.DB, c models.Todo, exclude ...uint) []Hit {
	p := indexFor(db, deref(c.ProjectID))
	if p.titles.Len() == 0 {
		return nil
	}
	titleSim := map[uint]float64{}
	for _, h := range p.titles.Similar(c.Title, "", 20, DuplicateTitleSim, nil) {
		titleSim[h.ID] = h.Sim
	}
	skip := map[uint]bool{c.ID: c.ID != 0}
	for _, id := range exclude {
		skip[id] = true
	}
	target, comp, files := strings.ToLower(strings.TrimSpace(c.TargetApps)), deref(c.ComponentID), titleFiles(c.Title)
	return similar(p, c.Title, c.Description, 3, DuplicateSim, func(d doc) bool {
		switch {
		case skip[d.id] || titleSim[d.id] == 0:
			return false
		case comp != 0 && d.componentID != 0 && comp != d.componentID:
			return false
		case target != "" && d.targetApps != "" && target != d.targetApps:
			return false
		}
		other := titleFiles(d.title)
		return len(files) == 0 || len(other) == 0 || files == other
	})
}

var titleFileRe = regexp.MustCompile(`[A-Za-z0-9_.-]+\.[A-Za-z]{1,5}\b`)

// titleFiles: nama berkas yang disebut judul, terurut & digabung (pembanding sasaran todo audit).
func titleFiles(title string) string {
	fs := titleFileRe.FindAllString(strings.ToLower(title), -1)
	sort.Strings(fs)
	return strings.Join(fs, ",")
}

func deref(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}

// StrongDuplicate: duplikat terkuat bila kemiripannya ≥ BlockSim (nil = boleh dibuat).
func StrongDuplicate(db *gorm.DB, c models.Todo) *Hit {
	if d := Duplicates(db, c); len(d) > 0 && d[0].Sim >= BlockSim {
		return &d[0]
	}
	return nil
}

// PromptSection: bagian prompt "todo serupa" untuk agent yang mengerjakan t — paling banyak k todo selesai /
// ditolak yang satu topik, beserta memorinya, agar agent memakai cara yang terbukti berhasil dan menghindari
// yang ditolak. Todo yang masih terbuka tidak disertakan (belum ada pelajaran darinya).
func PromptSection(db *gorm.DB, t models.Todo, k int, exclude ...uint) string {
	pid := uint(0)
	if t.ProjectID != nil {
		pid = *t.ProjectID
	}
	skip := map[uint]bool{t.ID: true}
	for _, id := range exclude {
		skip[id] = true
	}
	hits := similar(indexFor(db, pid), t.Title, t.Description, k, RelatedSim, func(d doc) bool {
		return !skip[d.id] && (d.status == models.TodoStatusCompleted || d.status == models.TodoStatusRejected)
	})
	var b strings.Builder
	for i, h := range hits {
		if i == 0 {
			b.WriteString("\n### Todo serupa di proyek ini (rujukan, bukan tugas)\n")
		}
		state := "selesai"
		if h.Status == models.TodoStatusRejected {
			state = "ditolak — jangan dikerjakan ulang dengan pendekatan yang sama"
		}
		if h.Score != nil {
			state += fmt.Sprintf(", nilai QA %d", *h.Score)
		}
		fmt.Fprintf(&b, "- #%d %s (%s, kemiripan %.2f)\n", h.ID, h.Title, state, h.Sim)
		if m := util.OneLine(h.Memory, 320); m != "" {
			b.WriteString("  " + m + "\n")
		}
	}
	return b.String()
}

func round(f float64) float64 { return float64(int(f*1000+0.5)) / 1000 }

// Neighbors: todo proyek yang satu konteks dengan t (bahan peleburan todo serupa, Aturan 43) — isi mirip
// ≥ contentMin dan judul mirip ≥ titleMin. Hasil: id → kemiripan isi.
func Neighbors(db *gorm.DB, t models.Todo, contentMin, titleMin float64) map[uint]float64 {
	p := indexFor(db, deref(t.ProjectID))
	if p.titles.Len() == 0 {
		return nil
	}
	titleOK := map[uint]bool{}
	for _, h := range p.titles.Similar(t.Title, "", 50, titleMin, nil) {
		titleOK[h.ID] = true
	}
	out := map[uint]float64{}
	for _, h := range p.ix.Similar(t.Title, IndexText(t.Description, ""), 50, contentMin, func(id uint) bool { return id == t.ID || !titleOK[id] }) {
		out[h.ID] = h.Sim
	}
	return out
}
