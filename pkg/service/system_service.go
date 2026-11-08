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
	targetPath = resolveBrowseTarget(targetPath, homeDir, documentsDir)
	quickPaths := browseQuickPaths(homeDir, documentsDir)

	if searchQuery != "" {
		return searchDirectories(targetPath, documentsDir, searchQuery, quickPaths), nil
	}
	return listDirectories(targetPath, documentsDir, quickPaths)
}

func resolveBrowseTarget(targetPath, homeDir, documentsDir string) string {
	if targetPath == "" {
		if isDirPath(documentsDir) {
			return documentsDir
		}
		return homeDir
	}
	if strings.HasPrefix(targetPath, "~") {
		targetPath = filepath.Join(homeDir, strings.TrimPrefix(targetPath, "~"))
	}
	return filepath.Clean(targetPath)
}

func browseQuickPaths(homeDir, documentsDir string) []DirectoryItem {
	quickPaths := []DirectoryItem{
		{Name: "Documents", Path: documentsDir},
		{Name: "Home (~)", Path: homeDir},
	}
	for _, name := range []string{"Nuxt", "go"} {
		if p := filepath.Join(documentsDir, name); isDirPath(p) {
			quickPaths = append(quickPaths, DirectoryItem{Name: name, Path: p})
		}
	}
	return quickPaths
}

var searchSkipDirs = map[string]bool{"node_modules": true, "vendor": true, "dist": true, "build": true, ".next": true, ".nuxt": true, ".git": true}

func searchDirectories(targetPath, documentsDir, searchQuery string, quickPaths []DirectoryItem) *BrowseDirectoryResponse {
	basePath := targetPath
	if !isDirPath(basePath) {
		basePath = documentsDir
	}

	var results []DirectoryItem
	lowerQuery := strings.ToLower(searchQuery)
	baseDepth := strings.Count(filepath.Clean(basePath), string(os.PathSeparator))
	const maxDepth = 3

	_ = filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == basePath {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") || searchSkipDirs[name] {
			return filepath.SkipDir
		}
		if strings.Count(filepath.Clean(path), string(os.PathSeparator))-baseDepth > maxDepth {
			return filepath.SkipDir
		}
		if strings.Contains(strings.ToLower(name), lowerQuery) {
			results = append(results, DirectoryItem{Name: name, Path: path})
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
	}
}

func listDirectories(targetPath, documentsDir string, quickPaths []DirectoryItem) (*BrowseDirectoryResponse, error) {
	if !isDirPath(targetPath) {
		targetPath = documentsDir
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var dirs []DirectoryItem
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
			continue
		}
		dirs = append(dirs, DirectoryItem{Name: name, Path: filepath.Join(targetPath, name)})
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
