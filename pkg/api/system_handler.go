package api

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func (s *Server) checkPorts(c *fiber.Ctx) error {
	portsQuery := c.Query("ports") // e.g. "8080,3000,3001"
	if portsQuery == "" {
		return c.JSON(fiber.Map{"status": "ok", "ports": fiber.Map{}})
	}

	portList := strings.Split(portsQuery, ",")
	results := make(map[string]bool)

	for _, pStr := range portList {
		port, err := strconv.Atoi(strings.TrimSpace(pStr))
		if err != nil || port <= 0 {
			continue
		}

		// Try TCP connection to 127.0.0.1:port
		target := fmt.Sprintf("127.0.0.1:%d", port)
		conn, dialErr := net.DialTimeout("tcp", target, 150*time.Millisecond)
		if dialErr == nil {
			conn.Close()
			results[strconv.Itoa(port)] = true // Port is ACTIVE / LISTENING
		} else {
			results[strconv.Itoa(port)] = false // Port is INACTIVE / FREE
		}
	}

	return c.JSON(fiber.Map{"ports": results})
}

type DirectoryItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type BrowseDirectoryResponse struct {
	CurrentPath string          `json:"current_path"`
	ParentPath  string          `json:"parent_path"`
	Directories []DirectoryItem `json:"directories"`
	QuickPaths  []DirectoryItem `json:"quick_paths"`
}

func (s *Server) browseDirectories(c *fiber.Ctx) error {
	searchQuery := strings.TrimSpace(c.Query("search"))
	targetPath := strings.TrimSpace(c.Query("path"))

	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		homeDir = "."
	}
	documentsDir := filepath.Join(homeDir, "Documents")

	if targetPath == "" {
		if info, err := os.Stat(documentsDir); err == nil && info.IsDir() {
			targetPath = documentsDir
		} else {
			targetPath = homeDir
		}
	} else {
		if strings.HasPrefix(targetPath, "~") {
			targetPath = filepath.Join(homeDir, strings.TrimPrefix(targetPath, "~"))
		}
		targetPath = filepath.Clean(targetPath)
	}

	quickPaths := []DirectoryItem{
		{Name: "Documents", Path: documentsDir},
		{Name: "Home (~)", Path: homeDir},
	}
	if info, err := os.Stat(filepath.Join(documentsDir, "Nuxt")); err == nil && info.IsDir() {
		quickPaths = append(quickPaths, DirectoryItem{Name: "Nuxt", Path: filepath.Join(documentsDir, "Nuxt")})
	}
	if info, err := os.Stat(filepath.Join(documentsDir, "go")); err == nil && info.IsDir() {
		quickPaths = append(quickPaths, DirectoryItem{Name: "go", Path: filepath.Join(documentsDir, "go")})
	}

	// Search mode
	if searchQuery != "" {
		basePath := targetPath
		if info, err := os.Stat(basePath); err != nil || !info.IsDir() {
			basePath = documentsDir
		}

		var results []DirectoryItem
		lowerQuery := strings.ToLower(searchQuery)
		baseDepth := strings.Count(filepath.Clean(basePath), string(os.PathSeparator))
		maxDepth := 3

		_ = filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			if path == basePath {
				return nil
			}

			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "dist" || name == "build" || name == ".next" || name == ".nuxt" || name == ".git" {
				return filepath.SkipDir
			}

			curDepth := strings.Count(filepath.Clean(path), string(os.PathSeparator)) - baseDepth
			if curDepth > maxDepth {
				return filepath.SkipDir
			}

			if strings.Contains(strings.ToLower(name), lowerQuery) {
				results = append(results, DirectoryItem{
					Name: name,
					Path: path,
				})
				if len(results) >= 40 {
					return filepath.SkipAll
				}
			}
			return nil
		})

		return c.JSON(BrowseDirectoryResponse{
			CurrentPath: basePath,
			ParentPath:  filepath.Dir(basePath),
			Directories: results,
			QuickPaths:  quickPaths,
		})
	}

	// Browse mode
	info, err := os.Stat(targetPath)
	if err != nil || !info.IsDir() {
		targetPath = documentsDir
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	var dirs []DirectoryItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
			continue
		}
		dirs = append(dirs, DirectoryItem{
			Name: name,
			Path: filepath.Join(targetPath, name),
		})
	}

	parentPath := filepath.Dir(targetPath)
	if parentPath == targetPath {
		parentPath = ""
	}

	return c.JSON(BrowseDirectoryResponse{
		CurrentPath: targetPath,
		ParentPath:  parentPath,
		Directories: dirs,
		QuickPaths:  quickPaths,
	})
}
