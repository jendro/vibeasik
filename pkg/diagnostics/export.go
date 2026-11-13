package diagnostics

import "github.com/vibeasik/vibeasik/pkg/models"

// Ekspor untuk modul Pro (deep audit, runtime simulation, UI perf deep) yang tinggal di paket lain
// tetapi memakai aturan penelusuran & pengukuran yang sama dengan audit dasar.

// IgnoredDirNames folder yang tidak pernah diaudit (node_modules, vendor, build, …).
var IgnoredDirNames = ignoredDirNames

// AuditFileExtensions ekstensi berkas sumber → bahasa.
var AuditFileExtensions = auditFileExtensions

// AuditAssetExtensions ekstensi aset (gambar, font, …) → jenis.
var AuditAssetExtensions = auditAssetExtensions

// FeRouterStaticImport mendeteksi import halaman statis di router frontend.
var FeRouterStaticImport = feRouterStaticImport

// FeRouterLazyRegex mendeteksi import halaman lazy di router frontend.
var FeRouterLazyRegex = feRouterLazyRegex

// HumanBytes memformat ukuran berkas (mis. "1.2 MB").
func HumanBytes(n int64) string { return humanBytes(n) }

// IsUISourceFile melaporkan apakah berkas adalah kode UI frontend.
func IsUISourceFile(relPath, lang string) bool { return isUISourceFile(relPath, lang) }

// IsTestOrGeneratedFile melaporkan berkas test / hasil generator menurut namanya.
func IsTestOrGeneratedFile(base string) bool { return isTestOrGeneratedFile(base) }

// IsGeneratedContent melaporkan isi berkas hasil generator.
func IsGeneratedContent(content string) bool { return isGeneratedContent(content) }

// GitStatusShort ringkasan `git status --short` folder dir.
func GitStatusShort(dir string) string { return getGitStatusShort(dir) }

// CheckFunctionComplexity mengukur kompleksitas fungsi di content dan menambah temuan.
func CheckFunctionComplexity(content, targetFile, lang string, findings *[]models.Finding) {
	checkFunctionComplexity(content, targetFile, lang, findings)
}

// FillFindingEvidence mengisi potongan kode temuan yang menunjuk berkas & baris tetapi belum membawa buktinya
// (dipakai pemeriksaan di luar audit statis, mis. Deep Check).
func FillFindingEvidence(project *models.Project, findings []models.Finding) {
	fillFindingEvidence(project, findings)
}
