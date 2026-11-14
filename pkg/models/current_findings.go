package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// CurrentProjectFindings: temuan proyek yang berlaku sekarang — himpunan yang sama dengan halaman Audit & Diagnostik
// (GET /api/projects/:id/diagnostics/latest): audit proyek terbaru (tanpa komponen) digabung Deep Check terbaru, lalu
// dideduplikasi. Diagnosa per komponen tidak ikut: hasilnya sudah tercakup di audit proyek dan tidak tampil di halaman
// itu, sehingga menghitungnya membuat angka ringkasan berbeda dari daftar yang bisa diproses pengguna.
// lastAt = waktu pemeriksaan terakhir di antara keduanya (nil bila proyek belum pernah diperiksa). Hanya membaca.
func CurrentProjectFindings(db *gorm.DB, projectID uint) (findings []Finding, lastAt *time.Time, err error) {
	var diag Diagnostic
	res := db.Select("id", "findings_json", "created_at").
		Where("project_id = ? AND (component_id IS NULL OR component_id = 0)", projectID).
		Order("created_at DESC").Limit(1).Find(&diag)
	if res.Error != nil {
		return nil, nil, res.Error
	}
	if res.RowsAffected > 0 {
		at := diag.CreatedAt
		lastAt = &at
		if diag.FindingsJSON != "" {
			_ = json.Unmarshal([]byte(diag.FindingsJSON), &findings)
		}
	}

	var deep DeepCheckReport
	res = db.Select("id", "findings_json", "created_at").Where("project_id = ?", projectID).
		Order("created_at DESC").Limit(1).Find(&deep)
	if res.Error != nil {
		return nil, nil, res.Error
	}
	if res.RowsAffected > 0 {
		if lastAt == nil || deep.CreatedAt.After(*lastAt) {
			at := deep.CreatedAt
			lastAt = &at
		}
		var deepFindings []Finding
		if deep.FindingsJSON != "" && json.Unmarshal([]byte(deep.FindingsJSON), &deepFindings) == nil && len(deepFindings) > 0 {
			findings = MergeFindings(findings, deepFindings)
		}
	}
	return DeduplicateFindings(findings), lastAt, nil
}

// IsOpenFinding: temuan yang masih menunggu diproses (belum jadi todo/selesai dan tidak diabaikan) — sama dengan
// hitungan "bisa diselesaikan" di halaman Audit.
func IsOpenFinding(f Finding) bool {
	return f.Status != FindingStatusResolved && f.Status != FindingStatusIgnored
}
