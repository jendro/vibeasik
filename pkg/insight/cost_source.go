package insight

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/codemap"
	"github.com/vibeasik/vibeasik/pkg/models"
)

// SourceWindow = potongan berkas untuk tampilan editor bukti biaya (nomor baris asli + isi baris apa adanya).
type SourceWindow struct {
	File  string   `json:"file"`
	From  int      `json:"from"` // baris pertama (1-based)
	Lines []string `json:"lines"`
	Total int      `json:"total"` // jumlah baris berkas
}

const maxSourceWindow = 400

// ErrSourceOutside: path keluar dari folder proyek atau berkasnya tidak ada.
var ErrSourceOutside = errors.New("berkas tidak ada di folder proyek")

// CostSourceLines membaca baris from..to berkas relatif root proyek (paling banyak 400 baris). Hanya berkas
// di dalam folder proyek (root atau folder aplikasinya) yang bisa dibaca.
func CostSourceLines(db *gorm.DB, projectID uint, file string, from, to int) (*SourceWindow, error) {
	var project models.Project
	if err := db.Preload("Components").First(&project, projectID).Error; err != nil {
		return nil, fmt.Errorf("proyek %d tidak ditemukan", projectID)
	}
	rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(file)))
	if rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, ErrSourceOutside
	}
	roots := []string{codemap.EffectiveRoot(project)}
	for _, c := range project.Components {
		roots = append(roots, c.AbsolutePath)
	}
	var data []byte
	for _, root := range roots {
		if root == "" {
			continue
		}
		p := filepath.Join(root, rel)
		if !strings.HasPrefix(p, filepath.Clean(root)+string(filepath.Separator)) {
			continue
		}
		if b, err := os.ReadFile(p); err == nil {
			data = b
			break
		}
	}
	if data == nil {
		return nil, ErrSourceOutside
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if from < 1 {
		from = 1
	}
	if to < from {
		to = from
	}
	if to-from+1 > maxSourceWindow {
		to = from + maxSourceWindow - 1
	}
	if to > len(lines) {
		to = len(lines)
	}
	out := &SourceWindow{File: filepath.ToSlash(rel), From: from, Total: len(lines)}
	if from <= to {
		out.Lines = lines[from-1 : to]
	}
	return out, nil
}
