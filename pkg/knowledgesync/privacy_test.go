package knowledgesync

import (
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestStaticCheckRejects(t *testing.T) {
	cases := map[string]string{
		"path unix":         "Simpan konfigurasi di /home/budi/app/config.yml sebelum deploy",
		"path tilde":        "Jalankan skrip di ~/proyek/rahasia.sh",
		"path windows":      `Buka C:\Users\Budi\Documents\app`,
		"ip":                "Hubungkan ke database di 192.168.1.20 lewat port biasa",
		"localhost":         "Buka http://localhost:3000 untuk melihat hasilnya",
		"host internal":     "Panggil layanan billing.internal untuk menagih",
		"url kredensial":    "Gunakan postgres://admin:hunter2@db.example.com/app",
		"url token":         "Unduh dari https://cdn.example.com/f.zip?token=abc123",
		"string acak":       "Kunci cadangan: 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		"uuid":              "Tenant 123e4567-e89b-12d3-a456-426614174000 harus dimigrasi",
		"email":             "Hubungi budi@perusahaan.co.id untuk akses",
		"telepon":           "Telepon 0812-3456-7890 bila gagal",
		"kartu":             "Nomor kartu 4111 1111 1111 1111 tidak boleh dicatat",
		"rahasia":           "Dokumen ini confidential, jangan dibagikan",
		"kunci privat":      "-----BEGIN RSA " + "PRIVATE KEY-----",
		"kode panjang":      strings.Repeat("    x := doSomething()\n", 17),
		"blok kode panjang": "```go\n" + strings.Repeat("line\n", 17) + "```",
	}
	for name, text := range cases {
		if r := staticCheck(text); r == "" {
			t.Errorf("%s: seharusnya ditolak: %q", name, text)
		}
	}
}

func TestStaticCheckAllowsGeneralKnowledge(t *testing.T) {
	ok := []string{
		"Gunakan parameter binding pada query SQL untuk mencegah injeksi.",
		"Selalu batasi ukuran body request dan atur timeout server HTTP.",
		"Pada Nuxt 4, pisahkan runtimeConfig publik dan privat; jangan menaruh rahasia di bagian public.",
		"Endpoint /api/users sebaiknya memakai paginasi dengan batas maksimum 100.",
		"Contoh: GET /health harus mengembalikan 200 tanpa autentikasi.",
		"Versi Go 1.26 mendukung rentang atas integer.\n\nLangkah:\n1. Jalankan tes\n2. Periksa hasil",
	}
	for _, text := range ok {
		if r := staticCheck(text); r != "" {
			t.Errorf("seharusnya lolos (%s): %q", r, text)
		}
	}
}

func TestPrivacyCheckerUsesLocalProjectsAndKeywords(t *testing.T) {
	db := testDB(t)
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}, &models.CensoredKeyword{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&models.Project{Name: "Toko Sinar Jaya", Slug: "toko-sinar-jaya", RootPath: "/work/sinarjaya-shop"})
	db.Create(&models.Component{ProjectID: 1, Name: "backend", RelativePath: "be", AbsolutePath: "/work/sinarjaya-shop/be"})
	db.Create(&models.CensoredKeyword{Keyword: "PROYEK-X", IsActive: true})
	db.Create(&models.CensoredKeyword{Keyword: `klien\d+`, IsRegex: true, IsActive: true})
	pc := NewPrivacyChecker(db)

	reject := []string{
		"Pada Toko Sinar Jaya, cache keranjang perlu TTL",
		"Repo sinarjaya-shop memakai pola ini",
		"catatan untuk proyek-x ada di wiki",
		"khusus klien42 gunakan batas lebih tinggi",
	}
	for _, text := range reject {
		if pc.Check(text) == "" {
			t.Errorf("seharusnya ditolak: %q", text)
		}
	}
	allow := []string{
		"Komponen backend sebaiknya memisahkan lapisan service dan repository", // kata umum tidak dianggap identitas
		"Gunakan cache dengan TTL pendek untuk data keranjang",
	}
	for _, text := range allow {
		if r := pc.Check(text); r != "" {
			t.Errorf("seharusnya lolos (%s): %q", r, text)
		}
	}
}
