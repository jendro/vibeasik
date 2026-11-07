// Package ui menyajikan Studio (hasil build Nuxt) dari binary.
package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

//go:embed all:dist
var embeddedFS embed.FS

// override diisi build lain (mis. Official) yang menyematkan Studio versinya sendiri.
var override fs.FS

// SetFS mengganti Studio tersemat dengan root (folder berisi index.html). Panggil di init().
func SetFS(root fs.FS) { override = root }

// GetFileSystem mengembalikan berkas Studio yang disajikan engine. Urutan:
//  1. Folder di env VIBEASIK_UI_DIR (untuk pengembangan Studio), bila berisi index.html.
//  2. Studio yang dipasang lewat SetFS.
//  3. Studio yang tersemat di binary ini (pkg/ui/dist).
//
// Folder di sekitar direktori kerja sengaja tidak dicari, agar engine tidak menyajikan build lain secara tidak sengaja.
func GetFileSystem() (http.FileSystem, bool) {
	if dir := os.Getenv("VIBEASIK_UI_DIR"); dir != "" {
		if fi, err := os.Stat(filepath.Join(dir, "index.html")); err == nil && !fi.IsDir() {
			return http.Dir(dir), true
		}
	}
	if override != nil {
		if _, err := fs.Stat(override, "index.html"); err == nil {
			return http.FS(override), true
		}
	}
	if subFS, err := fs.Sub(embeddedFS, "dist"); err == nil {
		if _, err := fs.Stat(subFS, "index.html"); err == nil {
			return http.FS(subFS), true
		}
	}
	return nil, false
}
