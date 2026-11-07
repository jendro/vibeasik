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

// GetFileSystem returns an http.FileSystem to serve static frontend files.
// Priority:
// 1. Local disk path studio-frontend/.output/public (if running from repo/dev)
// 2. Embedded filesystem compiled into binary
func GetFileSystem() (http.FileSystem, bool) {
	candidates := []string{
		"../studio-frontend/.output/public",
		"studio-frontend/.output/public",
		"internal/ui/dist",
	}

	for _, dir := range candidates {
		idxPath := filepath.Join(dir, "index.html")
		if fi, err := os.Stat(idxPath); err == nil && !fi.IsDir() {
			return http.Dir(dir), true
		}
	}

	subFS, err := fs.Sub(embeddedFS, "dist")
	if err == nil {
		if _, err := fs.Stat(subFS, "index.html"); err == nil {
			return http.FS(subFS), true
		}
	}

	return nil, false
}
