package paging

import (
	"fmt"
	"regexp"
	"strings"
)

var reDataTask = regexp.MustCompile(`(?i)\b(list|daftar|tabel|table|datagrid|data grid|endpoint|paginat\w*|paging|infinite|scroll|load more|muat lagi|koleksi|collection|riwayat|history|histori|feed|export|ekspor|findall|getall|get all|katalog|inbox|timeline|repository)\b|select\s+\*`)

// IsDataTask menebak apakah tugas menyentuh pengambilan/penampilan banyak data (butuh patokan paging).
func IsDataTask(text string) bool { return reDataTask.MatchString(text) }

// AgentGuidance: patokan "batas data per load" untuk prompt agent (kosong bila tugas bukan soal data/list).
// Dipakai orkestrator, divisi, dan prompt pegawai agar SEMUA agent coding menerapkannya tanpa perlu diingatkan.
func AgentGuidance(taskText string) string {
	if !IsDataTask(taskText) {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n### Batas data per load (patokan paging vibeasik — Aturan 37)\n")
	b.WriteString("Tugas ini menyentuh list/query/tampilan banyak data. Jumlah baris per permintaan TIDAK tetap: turunkan dari bobot baris (byte JSON satu baris list; ukur p90 dari ±20 baris contoh), bukan angka asal.\n")
	fmt.Fprintf(&b, "- Rumus: `limit = floor(%d KB ÷ bobot_baris)` dibulatkan ke angka bulat, 1..%d (mobile %d KB, maks 100). Batas keras respons %d KB — server MEMOTONG `limit` dari klien (clamp), jangan percaya input.\n", TargetPayload/1024, MaxPageSize, MobilePayload/1024, HardPayload/1024)
	b.WriteString("- Tabel acuan (web): baris ≤500 B → 200 · ≤1 KB → 100 · ≤2 KB → 50 · ≤4 KB → 25 · ≤8 KB → 10 · ≤16 KB → 5 · ≤32 KB → 3 · lebih berat → proyeksi tipis wajib (1–2 baris tak masuk akal untuk list).\n")
	b.WriteString("- List hanya memilih kolom yang tampil; kolom >1 KB (text/longtext/JSON/blob/base64) diambil lewat endpoint detail. Tanpa `SELECT *`.\n")
	b.WriteString("- Selalu `ORDER BY` kolom stabil + `id`. OFFSET untuk tabel kecil; tabel ≥10 rb baris atau halaman dalam (offset >1000) → keyset/cursor `(sort_key,id) > (:last)` + indeks. Jangan `COUNT(*)` per request di tabel besar: ambil `limit+1` untuk `has_more`/`next_cursor`.\n")
	b.WriteString("- UI: simpul DOM yang dirender ≤ 800 (baris × simpul per baris); lebih dari itu pakai virtual list atau 'muat lagi'. Tampilkan skeleton saat memuat, jangan menahan seluruh halaman menunggu semua data.\n")
	b.WriteString("- Hitung rencana tepatnya: MCP `vibeasik_paging_plan {columns|row_bytes, table_rows, mobile, nodes_per_row}`; patokan audit: NET-007, NET-008, NET-009, FE-014, MOB-007.\n")
	return b.String()
}
