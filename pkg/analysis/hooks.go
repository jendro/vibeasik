// Package analysis berisi analisis statis komponen tanpa LLM: fast scan (stack, arsitektur, env, rute),
// pengelompokan hasil scan menjadi proyek, dan sinkronisasi rute.
//
// Hasilnya bisa diperkaya LLM bila modul Pro memasang LLMQuery; tanpa itu dipakai heuristik.
package analysis

import "gorm.io/gorm"

// LLMQuery dipasang modul Pro (paket ai) untuk memperkaya fast scan & pengelompokan dengan LLM.
// Nil di build Community: semua fungsi memakai hasil heuristik saja.
var LLMQuery func(db *gorm.DB, systemPrompt, userPrompt string) (string, error)

func queryActiveLLM(db *gorm.DB, systemPrompt, userPrompt string) (string, error) {
	if LLMQuery == nil {
		return "", nil
	}
	return LLMQuery(db, systemPrompt, userPrompt)
}
