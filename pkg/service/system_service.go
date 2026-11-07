package service

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// DirectoryItem represents a browsable filesystem folder
type DirectoryItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// BrowseDirectoryResponse represents the folder browsing payload
type BrowseDirectoryResponse struct {
	CurrentPath string          `json:"current_path"`
	ParentPath  string          `json:"parent_path"`
	Directories []DirectoryItem `json:"directories"`
	QuickPaths  []DirectoryItem `json:"quick_paths"`
}

// SystemService defines utility operations for host system inspection
type SystemService interface {
	CheckPorts(ctx context.Context, ports []int) map[string]bool
	BrowseDirectories(ctx context.Context, targetPath, searchQuery string) (*BrowseDirectoryResponse, error)
}

type systemService struct{}

// NewSystemService instantiates a new SystemService
func NewSystemService() SystemService {
	return &systemService{}
}

func (s *systemService) CheckPorts(ctx context.Context, ports []int) map[string]bool {
	results := make(map[string]bool)
	for _, port := range ports {
		if port <= 0 {
			continue
		}
		target := fmt.Sprintf("127.0.0.1:%d", port)
		conn, dialErr := net.DialTimeout("tcp", target, 150*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			results[strconv.Itoa(port)] = true // Active / Listening
		} else {
			results[strconv.Itoa(port)] = false // Free / Inactive
		}
	}
	return results
}

func (s *systemService) BrowseDirectories(ctx context.Context, targetPath, searchQuery string) (*BrowseDirectoryResponse, error) {
	searchQuery = strings.TrimSpace(searchQuery)
	targetPath = strings.TrimSpace(targetPath)

	homeDir := models.UserHome()
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

		return &BrowseDirectoryResponse{
			CurrentPath: basePath,
			ParentPath:  filepath.Dir(basePath),
			Directories: results,
			QuickPaths:  quickPaths,
		}, nil
	}

	// Browse mode
	info, err := os.Stat(targetPath)
	if err != nil || !info.IsDir() {
		targetPath = documentsDir
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
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

	return &BrowseDirectoryResponse{
		CurrentPath: targetPath,
		ParentPath:  parentPath,
		Directories: dirs,
		QuickPaths:  quickPaths,
	}, nil
}
