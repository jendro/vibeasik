package paging

import (
	"strings"
	"testing"
)

func TestPageSizeScalesWithRowWeight(t *testing.T) {
	cases := []struct{ row, web, mobile int }{
		{100, 200, 100}, {500, 200, 100}, {1000, 100, 50}, {2000, 50, 25}, {4000, 25, 10}, {8000, 10, 5}, {16000, 5, 3}, {32000, 3, 1}, {120000, 1, 1},
	}
	for _, c := range cases {
		if got := PageSize(c.row, Options{}); got != c.web {
			t.Errorf("web row=%d: %d, mau %d", c.row, got, c.web)
		}
		if got := PageSize(c.row, Options{Mobile: true}); got != c.mobile {
			t.Errorf("mobile row=%d: %d, mau %d", c.row, got, c.mobile)
		}
	}
	prev := 1 << 30
	for row := 64; row <= 200000; row += 37 {
		p := PageSize(row, Options{})
		if p > prev {
			t.Fatalf("ukuran halaman harus tak naik saat baris membesar: row=%d %d>%d", row, p, prev)
		}
		if p*row > TargetPayload && p > 1 {
			t.Fatalf("halaman melewati anggaran: row=%d page=%d", row, p)
		}
		prev = p
	}
}

func TestClampLimit(t *testing.T) {
	if got := ClampLimit(0, 2000, Options{}); got != 50 {
		t.Errorf("default = %d", got)
	}
	if got := ClampLimit(10_000, 0, Options{}); got != MaxPageSize {
		t.Errorf("tanpa bobot baris dibatasi %d, dapat %d", MaxPageSize, got)
	}
	if got := ClampLimit(200, 20_000, Options{}); got*20_000 > HardPayload {
		t.Errorf("tak boleh melewati batas keras 512 KB: %d baris", got)
	}
	if got := ClampLimit(20, 20_000, Options{}); got != 20 {
		t.Errorf("permintaan kecil dihormati: %d", got)
	}
	if got := ClampLimit(5, 10_000_000, Options{}); got < 1 {
		t.Errorf("minimal 1 baris")
	}
}

func TestEstimateRowFlagsHeavyColumns(t *testing.T) {
	bytes, heavy := EstimateRow([]Column{{Name: "id", Type: "bigint"}, {Name: "title", Type: "varchar", Size: 200}, {Name: "created_at", Type: "timestamp"},
		{Name: "analysis_json", Type: "longtext"}})
	if len(heavy) != 1 || heavy[0] != "analysis_json" {
		t.Errorf("kolom berat = %v", heavy)
	}
	if bytes < 4000 || bytes > 4500 {
		t.Errorf("perkiraan bobot baris = %d", bytes)
	}
	slim, h := EstimateRow([]Column{{Name: "id", Type: "int"}, {Name: "title", Type: "varchar", Size: 200}, {Name: "status", Type: "enum"}})
	if len(h) != 0 || slim > 250 {
		t.Errorf("baris tipis keliru: %d %v", slim, h)
	}
}

func TestRecommendStrategies(t *testing.T) {
	small := Recommend(Input{RowBytes: 900, TableRows: 300, NodesPerRow: 10})
	if small.Strategy != "offset" || small.Count != "exact" || small.PageSize != 100 {
		t.Errorf("tabel kecil: %+v", small)
	}
	big := Recommend(Input{RowBytes: 900, TableRows: 2_000_000})
	if big.Strategy != "keyset" || big.Count != "has_more" {
		t.Errorf("tabel besar harus keyset + has_more: %+v", big)
	}
	fat := Recommend(Input{Columns: []Column{{Name: "id", Type: "int"}, {Name: "body", Type: "longtext"}}})
	if !fat.SlimList || len(fat.HeavyColumns) != 1 {
		t.Errorf("kolom longtext harus memaksa proyeksi tipis: %+v", fat)
	}
	dom := Recommend(Input{RowBytes: 400, NodesPerRow: 40, TableRows: 5000})
	if dom.UIMode != "virtual_list" || dom.UIRows != 20 {
		t.Errorf("200 baris × 40 simpul harus virtual list: %+v", dom)
	}
	if !strings.Contains(dom.Markdown(), "baris per permintaan") {
		t.Errorf("markdown rencana kosong")
	}
	unknown := Recommend(Input{})
	if unknown.RowBytes != 1024 || len(unknown.Notes) == 0 {
		t.Errorf("bobot tak diketahui harus diberi catatan: %+v", unknown)
	}
}

func TestGuidanceOnlyForListTasks(t *testing.T) {
	if g := AgentGuidance("Ubah warna tombol simpan"); g != "" {
		t.Errorf("tugas non-data tidak diberi panduan paging: %q", g)
	}
	for _, task := range []string{"Buat endpoint daftar pesanan", "Tampilkan tabel riwayat transaksi", "Tambahkan infinite scroll di feed", "SELECT * FROM orders tanpa limit"} {
		g := AgentGuidance(task)
		if !strings.Contains(g, "Batas data per load") || !strings.Contains(g, "vibeasik_paging_plan") {
			t.Errorf("%q harus mendapat panduan paging:\n%s", task, g)
		}
	}
}
