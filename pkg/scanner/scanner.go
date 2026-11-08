package scanner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// ScanDirectory analyzes a local filesystem path and detects its architecture
func ScanDirectory(rootPath string) (*models.ScanResult, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("directory not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("specified path is not a directory")
	}

	result := &models.ScanResult{
		RootPath:       absPath,
		SuggestedName:  cleanProjectName(filepath.Base(absPath)),
		SuggestedSlug:  slugify(filepath.Base(absPath)),
		DetectedStack:  make([]string, 0),
		Components:     make([]models.ComponentProposal, 0),
		IssuesDetected: make([]models.Finding, 0),
	}

	// 1. Git info
	branch, remote := getGitInfo(absPath)
	result.GitBranch = branch
	result.GitRemote = remote

	// 2. Discover components
	// First check if root itself is a single project
	rootComp := inspectDirectory(absPath, absPath, "")
	if rootComp != nil {
		// Root is a component. Still inspect subfolders if it's a monorepo or bundle
		result.Components = append(result.Components, *rootComp)
	}

	// Now inspect 1st and 2nd level subdirectories
	entries, _ := os.ReadDir(absPath)
	for _, entry := range entries {
		if !entry.IsDir() || shouldSkipDir(entry.Name()) {
			continue
		}
		subPath := filepath.Join(absPath, entry.Name())
		relPath := entry.Name()

		comp := inspectDirectory(absPath, subPath, relPath)
		if comp != nil && !isDuplicateComponent(result.Components, comp.AbsolutePath) {
			result.Components = append(result.Components, *comp)
		} else {
			// Check 2nd level subdirectories (e.g. apps/web, apps/api, packages/...)
			subEntries, _ := os.ReadDir(subPath)
			for _, subEntry := range subEntries {
				if !subEntry.IsDir() || shouldSkipDir(subEntry.Name()) {
					continue
				}
				deepPath := filepath.Join(subPath, subEntry.Name())
				deepRelPath := filepath.Join(relPath, subEntry.Name())

				deepComp := inspectDirectory(absPath, deepPath, deepRelPath)
				if deepComp != nil && !isDuplicateComponent(result.Components, deepComp.AbsolutePath) {
					result.Components = append(result.Components, *deepComp)
				}
			}
		}
	}

	// 3. Classify roles and project type
	refineComponentRoles(result)

	// 4. Collect detected stack summary
	stackMap := make(map[string]bool)
	for _, c := range result.Components {
		if c.Language != "" {
			stackMap[c.Language] = true
		}
		if c.Framework != "" {
			stackMap[c.Framework] = true
		}
	}
	for s := range stackMap {
		result.DetectedStack = append(result.DetectedStack, s)
	}

	// 5. Generate Directory Tree summary
	result.DirectoryTree = generateFolderTree(absPath, 2)

	return result, nil
}

func inspectDirectory(rootPath, dirPath, relPath string) *models.ComponentProposal {
	if relPath == "" {
		relPath = "."
	}
	dirName := filepath.Base(dirPath)

	// Check for Go
	if util.FileExists(filepath.Join(dirPath, "go.mod")) {
		comp := &models.ComponentProposal{
			Name:           formatComponentName(dirName, "Go Backend"),
			Role:           models.RoleBackend,
			RelativePath:   relPath,
			AbsolutePath:   dirPath,
			Language:       "go",
			PackageManager: "go",
			DevCommand:     "go run main.go",
			BuildCommand:   "go build -o server .",
			Port:           findPortInDir(dirPath, 8080),
		}
		comp.Framework = detectGoFramework(filepath.Join(dirPath, "go.mod"))
		inspectEnv(dirPath, comp)
		return comp
	}

	// Check for Node / Bun / TypeScript (package.json)
	if util.FileExists(filepath.Join(dirPath, "package.json")) {
		comp := inspectPackageJSON(dirPath, relPath, dirName)
		inspectEnv(dirPath, comp)
		return comp
	}

	// Check for Flutter / Dart (pubspec.yaml)
	if util.FileExists(filepath.Join(dirPath, "pubspec.yaml")) {
		comp := &models.ComponentProposal{
			Name:           formatComponentName(dirName, "Flutter App"),
			Role:           models.RoleFrontendMobile,
			RelativePath:   relPath,
			AbsolutePath:   dirPath,
			Language:       "dart",
			Framework:      "flutter",
			PackageManager: "pub",
			DevCommand:     "flutter run",
			BuildCommand:   "flutter build appbundle",
			Port:           0,
		}
		inspectEnv(dirPath, comp)
		return comp
	}

	// Check for Python (requirements.txt, pyproject.toml)
	if util.FileExists(filepath.Join(dirPath, "requirements.txt")) || util.FileExists(filepath.Join(dirPath, "pyproject.toml")) {
		comp := &models.ComponentProposal{
			Name:           formatComponentName(dirName, "Python Service"),
			Role:           models.RoleBackend,
			RelativePath:   relPath,
			AbsolutePath:   dirPath,
			Language:       "python",
			PackageManager: "pip",
			DevCommand:     "python main.py",
			BuildCommand:   "",
			Port:           findPortInDir(dirPath, 8000),
		}
		comp.Framework = detectPythonFramework(dirPath)
		inspectEnv(dirPath, comp)
		return comp
	}

	// Check for PHP (composer.json, spark, artisan, or routes)
	if util.FileExists(filepath.Join(dirPath, "composer.json")) ||
		util.FileExists(filepath.Join(dirPath, "spark")) ||
		util.FileExists(filepath.Join(dirPath, "artisan")) ||
		util.FileExists(filepath.Join(dirPath, "app/Config/Routes.php")) ||
		util.FileExists(filepath.Join(dirPath, "application/config/routes.php")) {
		comp := inspectPhpDirectory(dirPath, relPath, dirName)
		inspectEnv(dirPath, comp)
		return comp
	}

	// Check for Rust (Cargo.toml)
	if util.FileExists(filepath.Join(dirPath, "Cargo.toml")) {
		comp := &models.ComponentProposal{
			Name:           formatComponentName(dirName, "Rust Service"),
			Role:           models.RoleBackend,
			RelativePath:   relPath,
			AbsolutePath:   dirPath,
			Language:       "rust",
			PackageManager: "cargo",
			DevCommand:     "cargo run",
			BuildCommand:   "cargo build --release",
			Port:           findPortInDir(dirPath, 8080),
		}
		inspectEnv(dirPath, comp)
		return comp
	}

	return nil
}

func inspectPhpDirectory(dirPath, relPath, dirName string) *models.ComponentProposal {
	composerPath := filepath.Join(dirPath, "composer.json")
	composerContent := ""
	if util.FileExists(composerPath) {
		if b, err := os.ReadFile(composerPath); err == nil {
			composerContent = string(b)
		}
	}

	// 1. CodeIgniter 4
	isCI4 := util.FileExists(filepath.Join(dirPath, "spark")) ||
		util.FileExists(filepath.Join(dirPath, "app/Config/Routes.php")) ||
		strings.Contains(composerContent, "codeigniter4/framework") ||
		strings.Contains(composerContent, "codeigniter4/appstarter")

	if isCI4 {
		devCmd := "php spark serve"
		port := 8080
		if strings.Contains(composerContent, `"serve":`) {
			devCmd = "composer serve"
			if strings.Contains(composerContent, "--port") {
				rePort := regexp.MustCompile(`--port\s+([0-9]+)`)
				if m := rePort.FindStringSubmatch(composerContent); len(m) > 1 {
					port, _ = strconv.Atoi(m[1])
				}
			}
		}
		if p := findPortInDir(dirPath, 0); p > 0 {
			port = p
		}

		return &models.ComponentProposal{
			Name:           formatComponentName(dirName, "CodeIgniter Backend"),
			Role:           models.RoleBackend,
			RelativePath:   relPath,
			AbsolutePath:   dirPath,
			Language:       "php",
			PackageManager: "composer",
			DevCommand:     devCmd,
			BuildCommand:   "",
			Port:           port,
			Framework:      "codeigniter",
		}
	}

	// 2. CodeIgniter 3
	isCI3 := util.FileExists(filepath.Join(dirPath, "application/config/routes.php")) ||
		util.FileExists(filepath.Join(dirPath, "system/core/CodeIgniter.php"))

	if isCI3 {
		return &models.ComponentProposal{
			Name:           formatComponentName(dirName, "CodeIgniter 3 App"),
			Role:           models.RoleBackend,
			RelativePath:   relPath,
			AbsolutePath:   dirPath,
			Language:       "php",
			PackageManager: "composer",
			DevCommand:     "php -S localhost:8000",
			BuildCommand:   "",
			Port:           findPortInDir(dirPath, 8000),
			Framework:      "codeigniter3",
		}
	}

	// 3. Laravel
	isLaravel := util.FileExists(filepath.Join(dirPath, "artisan")) ||
		strings.Contains(composerContent, "laravel/framework") ||
		strings.Contains(composerContent, "illuminate/") ||
		util.FileExists(filepath.Join(dirPath, "routes/web.php"))

	if isLaravel {
		return &models.ComponentProposal{
			Name:           formatComponentName(dirName, "Laravel Backend"),
			Role:           models.RoleBackend,
			RelativePath:   relPath,
			AbsolutePath:   dirPath,
			Language:       "php",
			PackageManager: "composer",
			DevCommand:     "php artisan serve",
			BuildCommand:   "",
			Port:           findPortInDir(dirPath, 8000),
			Framework:      "laravel",
		}
	}

	// 4. Generic PHP
	return &models.ComponentProposal{
		Name:           formatComponentName(dirName, "PHP App"),
		Role:           models.RoleBackend,
		RelativePath:   relPath,
		AbsolutePath:   dirPath,
		Language:       "php",
		PackageManager: "composer",
		DevCommand:     "php -S localhost:8000",
		BuildCommand:   "",
		Port:           findPortInDir(dirPath, 8000),
		Framework:      "php",
	}
}

func inspectPackageJSON(dirPath, relPath, dirName string) *models.ComponentProposal {
	pkgPath := filepath.Join(dirPath, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil
	}

	var pkg struct {
		Name         string            `json:"name"`
		Scripts      map[string]string `json:"scripts"`
		Dependencies map[string]string `json:"dependencies"`
		DevDeps      map[string]string `json:"devDependencies"`
	}
	_ = json.Unmarshal(data, &pkg)

	// Detect package manager
	pm := "npm"
	if util.FileExists(filepath.Join(dirPath, "bun.lockb")) || util.FileExists(filepath.Join(dirPath, "bun.lock")) {
		pm = "bun"
	} else if util.FileExists(filepath.Join(dirPath, "pnpm-lock.yaml")) {
		pm = "pnpm"
	} else if util.FileExists(filepath.Join(dirPath, "yarn.lock")) {
		pm = "yarn"
	}

	allDeps := make(map[string]bool)
	for k := range pkg.Dependencies {
		allDeps[k] = true
	}
	for k := range pkg.DevDeps {
		allDeps[k] = true
	}

	framework := "node"
	lang := "javascript"
	role := models.RoleFrontendClient
	defaultPort := 3000

	if util.FileExists(filepath.Join(dirPath, "tsconfig.json")) {
		lang = "typescript"
	}

	if allDeps["nuxt"] || allDeps["nuxt3"] || allDeps["@nuxt/ui"] {
		framework = "nuxt"
		defaultPort = 3000
	} else if allDeps["next"] {
		framework = "next"
		defaultPort = 3000
	} else if allDeps["vue"] {
		framework = "vue"
		defaultPort = 5173
	} else if allDeps["react"] {
		framework = "react"
		defaultPort = 5173
	} else if allDeps["@nestjs/core"] {
		framework = "nestjs"
		role = models.RoleBackend
		defaultPort = 3000
	} else if allDeps["express"] || allDeps["fastify"] || allDeps["koa"] {
		framework = "express"
		role = models.RoleBackend
		defaultPort = 8000
	}

	devCmd := fmt.Sprintf("%s run dev", pm)
	if _, ok := pkg.Scripts["dev"]; !ok {
		if _, ok := pkg.Scripts["start"]; ok {
			devCmd = fmt.Sprintf("%s start", pm)
		}
	}

	buildCmd := ""
	if _, ok := pkg.Scripts["build"]; ok {
		buildCmd = fmt.Sprintf("%s run build", pm)
	}

	// Guess role by folder name
	lowerDir := strings.ToLower(dirName)
	if strings.Contains(lowerDir, "admin") || strings.Contains(lowerDir, "backoffice") || strings.Contains(lowerDir, "dash") {
		role = models.RoleFrontendAdmin
		if defaultPort == 3000 {
			defaultPort = 3001
		}
	} else if strings.Contains(lowerDir, "client") || strings.Contains(lowerDir, "store") || strings.Contains(lowerDir, "public") || strings.Contains(lowerDir, "web") {
		role = models.RoleFrontendClient
	} else if strings.Contains(lowerDir, "api") || strings.Contains(lowerDir, "server") || strings.Contains(lowerDir, "backend") {
		role = models.RoleBackend
		if defaultPort == 3000 {
			defaultPort = 8080
		}
	}

	compName := pkg.Name
	if compName == "" {
		compName = dirName
	}

	return &models.ComponentProposal{
		Name:           cleanComponentName(compName),
		Role:           role,
		RelativePath:   relPath,
		AbsolutePath:   dirPath,
		Language:       lang,
		Framework:      framework,
		PackageManager: pm,
		DevCommand:     devCmd,
		BuildCommand:   buildCmd,
		Port:           findPortInDir(dirPath, defaultPort),
	}
}

func detectGoFramework(goModPath string) string {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "standard"
	}
	content := string(data)
	if strings.Contains(content, "github.com/gofiber/fiber") {
		return "fiber"
	}
	if strings.Contains(content, "github.com/gin-gonic/gin") {
		return "gin"
	}
	if strings.Contains(content, "github.com/labstack/echo") {
		return "echo"
	}
	if strings.Contains(content, "github.com/go-chi/chi") {
		return "chi"
	}
	return "standard"
}

func detectPythonFramework(dirPath string) string {
	reqPath := filepath.Join(dirPath, "requirements.txt")
	if util.FileExists(reqPath) {
		data, _ := os.ReadFile(reqPath)
		content := strings.ToLower(string(data))
		if strings.Contains(content, "fastapi") {
			return "fastapi"
		}
		if strings.Contains(content, "flask") {
			return "flask"
		}
		if strings.Contains(content, "django") {
			return "django"
		}
	}
	return "standard"
}

func inspectEnv(dirPath string, comp *models.ComponentProposal) {
	for _, envCandidate := range []string{".env.example", ".env.sample", ".env.local", ".env"} {
		target := filepath.Join(dirPath, envCandidate)
		if util.FileExists(target) {
			comp.EnvFile = envCandidate
			keys := extractEnvKeys(target)
			if len(keys) > 0 {
				bytes, _ := json.Marshal(keys)
				comp.EnvSummary = string(bytes)
			}
			return
		}
	}
}

func extractEnvKeys(filePath string) []string {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	keys := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) > 0 {
			k := strings.TrimSpace(parts[0])
			if k != "" {
				keys = append(keys, k)
			}
		}
	}
	return keys
}

func findPortInDir(dirPath string, fallback int) int {
	// Look inside .env files for PORT
	for _, envCandidate := range []string{".env", ".env.local", ".env.development", ".env.example"} {
		fPath := filepath.Join(dirPath, envCandidate)
		if util.FileExists(fPath) {
			content, _ := os.ReadFile(fPath)
			re := regexp.MustCompile(`(?m)^PORT\s*=\s*(\d+)`)
			matches := re.FindStringSubmatch(string(content))
			if len(matches) > 1 {
				var port int
				fmt.Sscanf(matches[1], "%d", &port)
				if port > 0 {
					return port
				}
			}
		}
	}
	return fallback
}

func refineComponentRoles(result *models.ScanResult) {
	hasBackend := false
	frontends := make([]int, 0)

	for i, c := range result.Components {
		if c.Role == models.RoleBackend {
			hasBackend = true
		} else if strings.HasPrefix(c.Role, "frontend") {
			frontends = append(frontends, i)
		}
	}

	// If there are multiple frontends and none marked as admin yet, disambiguate
	if len(frontends) >= 2 {
		hasAdmin := false
		for _, idx := range frontends {
			if result.Components[idx].Role == models.RoleFrontendAdmin {
				hasAdmin = true
				break
			}
		}
		if !hasAdmin {
			// Mark first as Admin and second as Client (or by name hint)
			for _, idx := range frontends {
				name := strings.ToLower(result.Components[idx].Name)
				if strings.Contains(name, "admin") || strings.Contains(name, "manage") || strings.Contains(name, "back") {
					result.Components[idx].Role = models.RoleFrontendAdmin
					hasAdmin = true
					break
				}
			}
			if !hasAdmin && len(frontends) > 0 {
				result.Components[frontends[0]].Role = models.RoleFrontendAdmin
				if len(frontends) > 1 {
					result.Components[frontends[1]].Role = models.RoleFrontendClient
				}
			}
		}
	}

	// Determine overall project type
	if hasBackend && len(frontends) >= 2 {
		result.ProjectType = models.TypeMultiFE
	} else if hasBackend && len(frontends) >= 1 {
		result.ProjectType = models.TypeBundle
	} else if len(result.Components) > 1 {
		result.ProjectType = models.TypeMonorepo
	} else {
		result.ProjectType = models.TypeSingleApp
	}
}

func getGitInfo(dir string) (branch, remote string) {
	cmdBranch := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmdBranch.Dir = dir
	if out, err := cmdBranch.Output(); err == nil {
		branch = strings.TrimSpace(string(out))
	}

	cmdRemote := exec.Command("git", "config", "--get", "remote.origin.url")
	cmdRemote.Dir = dir
	if out, err := cmdRemote.Output(); err == nil {
		remote = strings.TrimSpace(string(out))
	}
	return
}

func generateFolderTree(root string, maxDepth int) string {
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
			if shouldSkipDir(entry.Name()) {
				continue
			}
			isLast := i == len(entries)-1
			connector := "├── "
			if isLast {
				connector = "└── "
			}

			if entry.IsDir() {
				sb.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, connector, entry.Name()))
				newPrefix := prefix + "│   "
				if isLast {
					newPrefix = prefix + "    "
				}
				walk(filepath.Join(current, entry.Name()), depth+1, newPrefix)
			} else {
				sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, entry.Name()))
			}
		}
	}

	sb.WriteString(fmt.Sprintf("%s/\n", filepath.Base(root)))
	walk(root, 1, "")
	return sb.String()
}

func shouldSkipDir(name string) bool {
	skip := []string{
		".git", "node_modules", "vendor", ".nuxt", ".output", "dist",
		"build", ".next", ".turbo", ".dart_tool", ".idea", ".vscode",
		"__pycache__", ".venv", "env", "coverage",
	}
	for _, s := range skip {
		if name == s {
			return true
		}
	}
	return false
}

func isDuplicateComponent(list []models.ComponentProposal, absPath string) bool {
	for _, c := range list {
		if c.AbsolutePath == absPath {
			return true
		}
	}
	return false
}

func cleanProjectName(name string) string {
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	return strings.Title(name)
}

func cleanComponentName(name string) string {
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	return strings.Title(name)
}

func formatComponentName(dirName, fallback string) string {
	if dirName == "." || dirName == "" {
		return fallback
	}
	return cleanComponentName(dirName)
}

func slugify(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile("[^a-z0-9]+")
	return strings.Trim(reg.ReplaceAllString(s, "-"), "-")
}
