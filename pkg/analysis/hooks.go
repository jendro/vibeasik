// Package analysis berisi analisis statis komponen tanpa LLM: fast scan (stack, arsitektur, env, rute),
// pengelompokan hasil scan menjadi proyek, dan sinkronisasi rute.
//
// Hasilnya bisa diperkaya LLM bila modul Pro memasang LLMQuery; tanpa itu dipakai heuristik.
package analysis

import (
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/security"
	"gorm.io/gorm"
)

// LLMQuery dipasang modul Pro (paket ai) untuk memperkaya fast scan & pengelompokan dengan LLM.
// Nil di build Community: semua fungsi memakai hasil heuristik saja.
var LLMQuery func(db *gorm.DB, systemPrompt, userPrompt string) (string, error)

func queryActiveLLM(db *gorm.DB, systemPrompt, userPrompt string) (string, error) {
	if LLMQuery == nil {
		return "", nil
	}
	cleanSys := security.SanitizeText(systemPrompt)
	cleanUser := security.SanitizeText(userPrompt)
	return LLMQuery(db, cleanSys, cleanUser)
}

// ExtractRoutes mengekstrak rute secara statis dari dir (dipakai modul Pro saat memperkaya rute).
func ExtractRoutes(dir string, role ...string) []models.RouteInfo { return extractRoutes(dir, role...) }
