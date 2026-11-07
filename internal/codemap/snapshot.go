package codemap

import (
	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/models"
)

// FileSymbols adalah satu berkas peta kode beserta simbol, impor, dan layernya — bahan analisis lintas berkas
// (konteks audit, pengecekan fungsi global) di paket lain tanpa membuka indeks internal.
type FileSymbols struct {
	Path      string              `json:"path"`
	AbsPath   string              `json:"abs_path"`
	Component string              `json:"component,omitempty"`
	Lang      string              `json:"lang"`
	Kind      string              `json:"kind"`
	Layer     string              `json:"layer,omitempty"`
	Lines     int                 `json:"lines"`
	Symbols   []models.CodeSymbol `json:"symbols"`
	Imports   []string            `json:"imports,omitempty"`
	Routes    []string            `json:"routes,omitempty"`
}

// Snapshot mengembalikan semua berkas peta kode proyek (dibangun/diperbarui dulu bila basi).
func Snapshot(db *gorm.DB, projectID uint) ([]FileSymbols, error) {
	if err := EnsureFresh(db, projectID, StaleAfter); err != nil {
		return nil, err
	}
	ix, err := loadIndex(db, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]FileSymbols, 0, len(ix.docs))
	for _, d := range ix.docs {
		out = append(out, FileSymbols{
			Path: d.entry.Path, AbsPath: d.entry.AbsPath, Component: d.entry.ComponentName, Lang: d.entry.Lang, Kind: d.entry.Kind,
			Layer: LayerOf(d.entry), Lines: d.entry.Lines, Symbols: d.symbols, Imports: d.imports, Routes: d.routes,
		})
	}
	return out, nil
}

// IsAppCode: berkas kode aplikasi (bukan test/dokumen/migrasi/konfigurasi/skrip/style).
func (f FileSymbols) IsAppCode() bool { return !skipKinds[f.Kind] }

// IsTest: berkas test.
func (f FileSymbols) IsTest() bool { return f.Kind == "test" }
