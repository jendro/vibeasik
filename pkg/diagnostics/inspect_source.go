package diagnostics

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// InspectSource memeriksa isi satu berkas yang tidak ada di disk (mis. versi berkas di branch lain) dengan
// aturan yang sama seperti InspectFile. Berkas test, hasil generator, dan ekstensi yang tidak diaudit dilewati.
func InspectSource(relPath, content string) []models.Finding {
	lang, ok := auditFileExtensions[strings.ToLower(filepath.Ext(relPath))]
	if !ok || lang == "xml" || isTestOrGeneratedFile(filepath.Base(relPath)) {
		return nil
	}
	dir, err := os.MkdirTemp("", "inspect-source-")
	if err != nil {
		return nil
	}
	defer os.RemoveAll(dir)
	full := filepath.Join(dir, filepath.Base(relPath))
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		return nil
	}
	return InspectFile(full, relPath, lang)
}
