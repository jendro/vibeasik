// Package util menampung fungsi pembantu kecil yang dipakai bersama banyak paket
// (string, berkas) agar satu aturan hanya hidup di satu tempat (DRY). Hanya bergantung
// pada stdlib, sehingga aman diimpor dari paket mana pun, termasuk vibeasik-pro.
package util

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

// ContainsAny melaporkan apakah string s memuat salah satu dari substring subs.
// Bila subs kosong, hasilnya false.
func ContainsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// Uniq mengembalikan irisan string tanpa duplikat, mempertahankan urutan kemunculan pertama.
// Elemen kosong ("") diabaikan.
func Uniq(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// ShouldSkipFolderTreeName melaporkan apakah sebuah berkas atau direktori diabaikan dalam pohon direktori.
func ShouldSkipFolderTreeName(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "node_modules", "vendor", "dist", "build", ".output", "__pycache__", ".venv", "env", "coverage":
		return true
	}
	return false
}

// GenerateFolderTree menghasilkan representasi teks pohon direktori berformat ASCII/Unicode
// hingga batas maxDepth. Mengabaikan entri tersembunyi (awalan titik) dan direktori umum artefak build.
func GenerateFolderTree(root string, maxDepth int) string {
	var sb strings.Builder
	var walk func(path string, depth int, prefix string)

	walk = func(current string, depth int, prefix string) {
		if depth > maxDepth {
			return
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			return
		}
		for i, entry := range entries {
			name := entry.Name()
			if ShouldSkipFolderTreeName(name) {
				continue
			}
			isLast := i == len(entries)-1
			connector := "├── "
			if isLast {
				connector = "└── "
			}

			if entry.IsDir() {
				sb.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, connector, name))
				newPrefix := prefix + "│   "
				if isLast {
					newPrefix = prefix + "    "
				}
				walk(filepath.Join(current, name), depth+1, newPrefix)
			} else {
				sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, name))
			}
		}
	}

	sb.WriteString(fmt.Sprintf("%s/\n", filepath.Base(root)))
	walk(root, 1, "")
	return sb.String()
}

