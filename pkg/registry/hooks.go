package registry

import (
	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Hooks adalah titik sambung tempat modul Pro memperkaya alur inti tanpa membuat inti bergantung
// pada Pro. Semua nil di build Community; pemanggil wajib memakai pembungkus di bawah.
type Hooks struct {
	// FilterFindings meredam temuan yang pernah divonis keliru (audit learning). Mengembalikan jumlah yang diredam.
	FilterFindings func(db *gorm.DB, projectID uint, findings []models.Finding) int
	// ProjectSummary membuat ringkasan proyek (LLM).
	ProjectSummary func(db *gorm.DB, projectID uint) (string, error)
	// ProjectChanged dipanggil setelah proyek/komponen berubah (mis. membuang cache pengetahuan dasar proyek).
	ProjectChanged func(projectID uint, rootPath string)
	// ProjectCreated dipanggil setelah proyek baru dibuat (mis. menyiapkan dewan eksekutif).
	ProjectCreated func(db *gorm.DB, project *models.Project)
	// TodoStarted dipanggil saat agent menandai todo in_progress. note ditambahkan ke jawaban agent;
	// stop=true berarti jawaban selesai di sini (mis. todo dipecah jadi sub todo).
	TodoStarted func(db *gorm.DB, todoID uint, agent string) (note string, stop bool)
	// TodoCompleted dipanggil saat todo selesai (mis. penilaian QA & pembaruan pengetahuan pegawai).
	TodoCompleted func(db *gorm.DB, todoID uint, agent string)
	// SuggestTodos menambah saran todo cerdas (rapat PM / LLM) di atas saran rule.
	SuggestTodos func(db *gorm.DB, project *models.Project, all []models.Project) []models.Todo
	// AutoAssignOn: keputusan saran todo langsung dikerjakan agent (mempengaruhi kebijakan eksekusi).
	AutoAssignOn func() bool
}

// H hook aktif. Diisi modul Pro di init().
var H Hooks

// FilterFindings menjalankan hook bila ada.
func FilterFindings(db *gorm.DB, projectID uint, findings []models.Finding) int {
	if H.FilterFindings == nil {
		return 0
	}
	return H.FilterFindings(db, projectID, findings)
}

// ProjectSummary menjalankan hook bila ada; "" bila tidak tersedia.
func ProjectSummary(db *gorm.DB, projectID uint) (string, error) {
	if H.ProjectSummary == nil {
		return "", nil
	}
	return H.ProjectSummary(db, projectID)
}

// ProjectChanged menjalankan hook bila ada.
func ProjectChanged(projectID uint, rootPath string) {
	if H.ProjectChanged != nil {
		H.ProjectChanged(projectID, rootPath)
	}
}

// ProjectCreated menjalankan hook bila ada.
func ProjectCreated(db *gorm.DB, project *models.Project) {
	if H.ProjectCreated != nil {
		H.ProjectCreated(db, project)
	}
}

// TodoStarted menjalankan hook bila ada.
func TodoStarted(db *gorm.DB, todoID uint, agent string) (string, bool) {
	if H.TodoStarted == nil {
		return "", false
	}
	return H.TodoStarted(db, todoID, agent)
}

// TodoCompleted menjalankan hook bila ada.
func TodoCompleted(db *gorm.DB, todoID uint, agent string) {
	if H.TodoCompleted != nil {
		H.TodoCompleted(db, todoID, agent)
	}
}

// SuggestTodos menjalankan hook bila ada.
func SuggestTodos(db *gorm.DB, project *models.Project, all []models.Project) []models.Todo {
	if H.SuggestTodos == nil {
		return nil
	}
	return H.SuggestTodos(db, project, all)
}

// AutoAssignOn melaporkan hook bila ada; false di Community.
func AutoAssignOn() bool {
	return H.AutoAssignOn != nil && H.AutoAssignOn()
}
