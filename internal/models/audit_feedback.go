package models

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"
)

// Vonis umpan balik audit: bahan pembelajaran agar audit & hasil koding makin tepat.
const (
	AuditVerdictFalsePositive = "false_positive" // audit keliru: kode sudah benar, rule/prompt audit perlu diperbarui
	AuditVerdictConfirmed     = "confirmed"      // temuan benar: pola yang wajib dihindari agent coding
)

// Sumber vonis
const (
	AuditFeedbackSourceManual = "manual" // ditandai manusia di halaman audit
	AuditFeedbackSourceLLM    = "llm"    // hasil "Cek ulang via LLM" yang diterima manusia
	AuditFeedbackSourceAgent  = "agent"  // agent coding (MCP) yang mengecek ulang kode
)

// AuditFeedback adalah satu vonis atas temuan audit. Satu baris per (proyek, kunci temuan); vonis terakhir menang.
type AuditFeedback struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	ProjectID   uint   `gorm:"index;not null" json:"project_id"`
	FindingKey  string `gorm:"size:255;index" json:"finding_key"`
	RuleID      string `gorm:"size:120;index" json:"rule_id"`
	Category    string `gorm:"size:60" json:"category"`
	Severity    string `gorm:"size:20" json:"severity"`
	Title       string `json:"title"`
	Description string `gorm:"type:text" json:"description"`
	TargetFile  string `gorm:"size:500;index" json:"target_file"`
	Line        int    `json:"line"`
	Snippet     string `gorm:"type:text" json:"snippet"`
	// SnippetHash: hash potongan kode yang dinormalisasi; dipakai mengenali temuan yang sama walau barisnya bergeser
	SnippetHash string `gorm:"size:40;index" json:"snippet_hash"`
	IsLLM       bool   `json:"is_llm"`

	Verdict string `gorm:"size:20;index" json:"verdict"`
	Reason  string `gorm:"type:text" json:"reason"`
	// RuleAdvice: saran perbaikan rule/prompt audit agar kekeliruan serupa tidak terulang
	RuleAdvice string `gorm:"type:text" json:"rule_advice,omitempty"`
	FixCode    string `gorm:"type:text" json:"fix_code,omitempty"` // contoh perbaikan untuk vonis confirmed
	Source     string `gorm:"size:20" json:"source"`
	Model      string `gorm:"size:120" json:"model,omitempty"`
	HitCount   int    `json:"hit_count"` // berapa kali temuan serupa otomatis diredam berkat vonis ini

	// RuleUpdated: rule/prompt audit sudah diperbarui untuk vonis false_positive ini (flag "audit perlu update" selesai)
	RuleUpdated   bool       `gorm:"index" json:"rule_updated"`
	RuleUpdatedAt *time.Time `json:"rule_updated_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuditSnippetHash menormalkan potongan kode (spasi & huruf) lalu meng-hash-nya; "" bila potongan kosong.
func AuditSnippetHash(snippet string) string {
	norm := strings.ToLower(strings.Join(strings.Fields(snippet), " "))
	if norm == "" {
		return ""
	}
	sum := sha1.Sum([]byte(norm))
	return hex.EncodeToString(sum[:])
}
