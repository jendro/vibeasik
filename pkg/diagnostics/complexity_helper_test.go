package diagnostics

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// scanComplexityForTest menelusuri dir seperti pemindai deep audit (Pro), tetapi hanya menjalankan
// pengukur kompleksitas fungsi milik Community. Dipakai test agar tidak bergantung pada kode Pro.
func scanComplexityForTest(dir string) []models.Finding {
	findings := make([]models.Finding, 0)
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if ignoredDirNames[name] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		lang, ok := auditFileExtensions[filepath.Ext(path)]
		if !ok || isTestOrGeneratedFile(info.Name()) {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		b, err := os.ReadFile(path)
		if err != nil || isGeneratedContent(string(b)) {
			return nil
		}
		if !isUISourceFile(rel, lang) {
			checkFunctionComplexity(string(b), rel, lang, &findings)
		}
		return nil
	})
	return findings
}
