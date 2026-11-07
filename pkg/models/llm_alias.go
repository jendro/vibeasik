package models

import (
	"encoding/json"
	"strings"

	"gorm.io/gorm"
)

// Alias tampilan per model LLM. Nama model asli bisa berbeda antar penyedia pihak ketiga (mis. "anthropic/claude-x"
// vs "claude-x-20250101"), jadi pengguna memberi alias yang tampil di pemilih model & Kantor Agent.
// Disimpan di LLMConfig.AliasJSON; request ke LLM tetap memakai nama model asli.

func (c *LLMConfig) aliasMap() map[string]string {
	out := map[string]string{}
	if c.AliasJSON != "" {
		_ = json.Unmarshal([]byte(c.AliasJSON), &out)
	}
	return out
}

// ModelAlias mengembalikan alias model, atau "" bila belum diberi alias.
func (c *LLMConfig) ModelAlias(model string) string {
	return c.aliasMap()[model]
}

// SetModelAlias menyimpan alias satu model; alias kosong menghapusnya.
func (c *LLMConfig) SetModelAlias(model, alias string) {
	a := c.aliasMap()
	if alias = strings.TrimSpace(alias); alias == "" {
		delete(a, model)
	} else {
		a[model] = alias
	}
	b, _ := json.Marshal(a)
	c.AliasJSON = string(b)
}

// ResolveModelAlias mencari alias model dari konfigurasi LLM, mengutamakan konfigurasi dengan endpoint yang sama.
// Tanpa alias, nama model asli dikembalikan.
func ResolveModelAlias(db *gorm.DB, endpoint, model string) string {
	if db == nil || model == "" {
		return model
	}
	var cfgs []LLMConfig
	if err := db.Where("alias_json <> ''").Find(&cfgs).Error; err != nil {
		return model
	}
	fallback := ""
	for i := range cfgs {
		alias := cfgs[i].ModelAlias(model)
		if alias == "" {
			continue
		}
		if endpoint != "" && strings.HasPrefix(endpoint, strings.TrimRight(cfgs[i].URL, "/")) {
			return alias
		}
		if fallback == "" {
			fallback = alias
		}
	}
	if fallback != "" {
		return fallback
	}
	return model
}
