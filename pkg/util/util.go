// Package util menampung fungsi pembantu kecil yang dipakai bersama banyak paket
// (string, berkas) agar satu aturan hanya hidup di satu tempat (DRY). Hanya bergantung
// pada stdlib, sehingga aman diimpor dari paket mana pun, termasuk vibeasik-pro.
package util

import (
	"log"
	"os"
	"strings"
)

// FirstNonEmpty mengembalikan argumen pertama yang tidak kosong setelah di-trim, dalam bentuk ter-trim.
// Bila semuanya kosong, hasilnya "".
func FirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

// FirstNonBlank seperti FirstNonEmpty, tetapi mengembalikan nilai ASLI (tidak di-trim).
func FirstNonBlank(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// OneLine memadatkan semua whitespace menjadi satu spasi lalu memotong ke n rune (ditambah "…" bila terpotong).
func OneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// ClipText men-trim s lalu memotongnya ke n rune (ditambah "…" bila terpotong); baris baru dipertahankan.
func ClipText(s string, n int) string {
	if r := []rune(strings.TrimSpace(s)); len(r) > n {
		return string(r[:n]) + "…"
	}
	return strings.TrimSpace(s)
}

// ReadFileString membaca seluruh berkas sebagai string; "" bila gagal dibaca.
func ReadFileString(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return string(b)
}

// FileExists true bila p ada DAN bukan direktori.
func FileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// PathExists true bila p ada (berkas ATAU direktori). Error sistem selain "tidak ada" (izin, I/O) dicatat ke log.
func PathExists(p string) bool {
	_, err := os.Stat(p)
	if err != nil && !os.IsNotExist(err) {
		// Berkas tidak ada adalah jalur normal; error sistem (izin, I/O) dicatat agar terlihat.
		log.Printf("[WARN] PathExists: os.Stat(%q): %v", p, err)
	}
	return err == nil
}
