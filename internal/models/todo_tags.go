package models

import (
	"sort"
	"strings"

	"gorm.io/gorm"
)

// Tag fokus siklus pengembangan (disimpan CSV di Todo.Tags, mis. "fitur,keamanan-dasar").
// Empat tag pertama = fokus utama yang dikerjakan duluan; "belajar" = topik lain
// (staging, cutover, observability, aset media, dsb.) yang disimpan sebagai bahan
// pembelajaran dan tidak didahulukan untuk dikirim ke agent.
const (
	TagFitur         = "fitur"
	TagKeamananDasar = "keamanan-dasar"
	TagOptimasiQuery = "optimasi-query"
	TagRawatAI       = "rawat-ai" // maintainability kode untuk AI agent (file kecil, SRP, kompleksitas rendah)
	TagBelajar       = "belajar"
)

// FocusTags = tag fokus utama, urut sesuai prioritas pengerjaan.
var FocusTags = []string{TagFitur, TagKeamananDasar, TagOptimasiQuery, TagRawatAI}

// NormalizeTags menerima CSV string atau array JSON, lalu mengembalikan CSV
// huruf kecil, tanpa spasi/duplikat, terurut.
func NormalizeTags(raw interface{}) string {
	var parts []string
	switch v := raw.(type) {
	case string:
		parts = strings.Split(v, ",")
	case []string:
		parts = v
	case []interface{}:
		for _, x := range v {
			if s, ok := x.(string); ok {
				parts = append(parts, s)
			}
		}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.ToLower(strings.TrimSpace(p))
		t = strings.ReplaceAll(t, " ", "-")
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// FocusRank = urutan kirim todo ke agent menurut tag (kecil = duluan):
// indeks tag fokus pertama yang dimiliki, len(FocusTags) bila tanpa tag fokus,
// dan paling akhir bila hanya bertanda belajar.
func FocusRank(tags string) int {
	set := map[string]bool{}
	for _, t := range strings.Split(tags, ",") {
		set[strings.TrimSpace(t)] = true
	}
	for i, t := range FocusTags {
		if set[t] {
			return i
		}
	}
	if set[TagBelajar] {
		return len(FocusTags) + 1
	}
	return len(FocusTags)
}

// WhereTag menyaring todo yang memiliki tag tertentu (cocok persis per elemen CSV).
func WhereTag(q *gorm.DB, tag string) *gorm.DB {
	tag = NormalizeTags(tag)
	if tag == "" {
		return q
	}
	expr := "(',' || COALESCE(tags,'') || ',') LIKE ?"
	if q.Dialector != nil && q.Dialector.Name() == "mysql" {
		expr = "CONCAT(',', COALESCE(tags,''), ',') LIKE ?" // di MySQL `||` = OR logika
	}
	return q.Where(expr, "%,"+tag+",%")
}
