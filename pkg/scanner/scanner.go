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

func validateScanDirectory(rootPath string) (string, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("directory not found: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("specified path is not a directory")
	}
	return absPath, nil
}

func discoverSubComponents(absPath, relPath, subPath string, components *[]models.ComponentProposal) {
	comp := inspectDirectory(absPath, subPath, relPath)
	if comp != nil && !isDuplicateComponent(*components, comp.AbsolutePath) {
		*components = append(*components, *comp)
		return
	}
	subEntries, _ := os.ReadDir(subPath)
	for _, subEntry := range subEntries {
		if !subEntry.IsDir() || shouldSkipDir(subEntry.Name()) {
			continue
		}
		deepPath := filepath.Join(subPath, subEntry.Name())
		deepRelPath := filepath.Join(relPath, subEntry.Name())
		deepComp := inspectDirectory(absPath, deepPath, deepRelPath)
		if deepComp != nil && !isDuplicateComponent(*components, deepComp.AbsolutePath) {
			*components = append(*components, *deepComp)
		}
	}
}

func discoverAllComponents(absPath string) []models.ComponentProposal {
	var components []models.ComponentProposal
	if rootComp := inspectDirectory(absPath, absPath, ""); rootComp != nil {
		components = append(components, *rootComp)
	}
	entries, _ := os.ReadDir(absPath)
	for _, entry := range entries {
		if !entry.IsDir() || shouldSkipDir(entry.Name()) {
			continue
		}
		subPath := filepath.Join(absPath, entry.Name())
		discoverSubComponents(absPath, entry.Name(), subPath, &components)
	}
	return components
}

func collectDetectedStack(components []models.ComponentProposal) []string {
	stackMap := make(map[string]bool)
	for _, c := range components {
		if c.Language != "" {
			stackMap[c.Language] = true
		}
		if c.Framework != "" {
			stackMap[c.Framework] = true
		}
	}
	var stack []string
	for s := range stackMap {
		stack = append(stack, s)
	}
	return stack
}

// ScanDirectory analyzes a local filesystem path and detects its architecture
func ScanDirectory(rootPath string) (*models.ScanResult, error) {
	absPath, err := validateScanDirectory(rootPath)
	if err != nil {
		return nil, err
	}

	branch, remote := getGitInfo(absPath)
	result := &models.ScanResult{
		RootPath:       absPath,
		SuggestedName:  cleanProjectName(filepath.Base(absPath)),
		SuggestedSlug:  slugify(filepath.Base(absPath)),
		GitBranch:      branch,
		GitRemote:      remote,
		DetectedStack:  make([]string, 0),
		Components:     discoverAllComponents(absPath),
		IssuesDetected: make([]models.Finding, 0),
	}

	refineComponentRoles(result)
	result.DetectedStack = collectDetectedStack(result.Components)
	result.DirectoryTree = util.GenerateFolderTree(absPath, 2)

	return result, nil
}

func inspectDirectory(rootPath, dirPath, relPath string) *models.ComponentProposal {
	if relPath == "" {
		relPath = "."
	}
	dirName := filepath.Base(dirPath)
	var comp *models.ComponentProposal
	switch {
	case util.FileExists(filepath.Join(dirPath, "go.mod")):
		comp = inspectGoDirectory(dirPath, relPath, dirName)
	case util.FileExists(filepath.Join(dirPath, "package.json")):
		comp = inspectPackageJSON(dirPath, relPath, dirName)
	case util.FileExists(filepath.Join(dirPath, "pubspec.yaml")):
		comp = inspectFlutterDirectory(dirPath, relPath, dirName)
	case util.FileExists(filepath.Join(dirPath, "requirements.txt")) || util.FileExists(filepath.Join(dirPath, "pyproject.toml")):
		comp = inspectPythonDirectory(dirPath, relPath, dirName)
	case isPhpDirectory(dirPath):
		comp = inspectPhpDirectory(dirPath, relPath, dirName)
	case util.FileExists(filepath.Join(dirPath, "Cargo.toml")):
		comp = inspectRustDirectory(dirPath, relPath, dirName)
	default:
		return nil
	}
	if comp != nil {
		inspectEnv(dirPath, comp)
	}
	return comp
}

func inspectGoDirectory(dirPath, relPath, dirName string) *models.ComponentProposal {
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
	return comp
}

func inspectFlutterDirectory(dirPath, relPath, dirName string) *models.ComponentProposal {
	return &models.ComponentProposal{
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
}

func inspectPythonDirectory(dirPath, relPath, dirName string) *models.ComponentProposal {
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
	return comp
}

func inspectRustDirectory(dirPath, relPath, dirName string) *models.ComponentProposal {
	return &models.ComponentProposal{
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
}

func isPhpDirectory(dirPath string) bool {
	return util.FileExists(filepath.Join(dirPath, "composer.json")) ||
		util.FileExists(filepath.Join(dirPath, "spark")) ||
		util.FileExists(filepath.Join(dirPath, "artisan")) ||
		util.FileExists(filepath.Join(dirPath, "app/Config/Routes.php")) ||
		util.FileExists(filepath.Join(dirPath, "application/config/routes.php"))
}

func detectCI4(dirPath, relPath, dirName, composerContent string) *models.ComponentProposal {
	isCI4 := util.FileExists(filepath.Join(dirPath, "spark")) ||
		util.FileExists(filepath.Join(dirPath, "app/Config/Routes.php")) ||
		strings.Contains(composerContent, "codeigniter4/framework") ||
		strings.Contains(composerContent, "codeigniter4/appstarter")

	if !isCI4 {
		return nil
	}

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

func detectCI3(dirPath, relPath, dirName string) *models.ComponentProposal {
	isCI3 := util.FileExists(filepath.Join(dirPath, "application/config/routes.php")) ||
		util.FileExists(filepath.Join(dirPath, "system/core/CodeIgniter.php"))

	if !isCI3 {
		return nil
	}
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

func detectLaravel(dirPath, relPath, dirName, composerContent string) *models.ComponentProposal {
	isLaravel := util.FileExists(filepath.Join(dirPath, "artisan")) ||
		strings.Contains(composerContent, "laravel/framework") ||
		strings.Contains(composerContent, "illuminate/") ||
		util.FileExists(filepath.Join(dirPath, "routes/web.php"))

	if !isLaravel {
		return nil
	}
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

func detectGenericPHP(dirPath, relPath, dirName string) *models.ComponentProposal {
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

func inspectPhpDirectory(dirPath, relPath, dirName string) *models.ComponentProposal {
	composerPath := filepath.Join(dirPath, "composer.json")
	composerContent := ""
	if util.FileExists(composerPath) {
		if b, err := os.ReadFile(composerPath); err == nil {
			composerContent = string(b)
		}
	}

	if comp := detectCI4(dirPath, relPath, dirName, composerContent); comp != nil {
		return comp
	}
	if comp := detectCI3(dirPath, relPath, dirName); comp != nil {
		return comp
	}
	if comp := detectLaravel(dirPath, relPath, dirName, composerContent); comp != nil {
		return comp
	}
	return detectGenericPHP(dirPath, relPath, dirName)
}

func detectPackageManager(dirPath string) string {
	if util.FileExists(filepath.Join(dirPath, "bun.lockb")) || util.FileExists(filepath.Join(dirPath, "bun.lock")) {
		return "bun"
	}
	if util.FileExists(filepath.Join(dirPath, "pnpm-lock.yaml")) {
		return "pnpm"
	}
	if util.FileExists(filepath.Join(dirPath, "yarn.lock")) {
		return "yarn"
	}
	return "npm"
}

func detectJSFramework(allDeps map[string]bool) (string, string, int) {
	if hasAnyDep(allDeps, "nuxt", "nuxt3", "@nuxt/ui") {
		return "nuxt", models.RoleFrontendClient, 3000
	}
	if allDeps["next"] {
		return "next", models.RoleFrontendClient, 3000
	}
	if allDeps["react"] {
		return "react", models.RoleFrontendClient, 5173
	}
	if allDeps["vue"] {
		return "vue", models.RoleFrontendClient, 5173
	}
	if allDeps["@nestjs/core"] {
		return "nestjs", models.RoleBackend, 3000
	}
	if hasAnyDep(allDeps, "express", "fastify", "koa") {
		return "express", models.RoleBackend, 8000
	}
	return "node", models.RoleFrontendClient, 3000
}

func hasAnyDep(allDeps map[string]bool, deps ...string) bool {
	for _, d := range deps {
		if allDeps[d] {
			return true
		}
	}
	return false
}

func detectJSScripts(scripts map[string]string, pm string) (string, string) {
	devCmd := fmt.Sprintf("%s run dev", pm)
	if _, ok := scripts["dev"]; !ok {
		if _, ok := scripts["start"]; ok {
			devCmd = fmt.Sprintf("%s start", pm)
		}
	}
	buildCmd := ""
	if _, ok := scripts["build"]; ok {
		buildCmd = fmt.Sprintf("%s run build", pm)
	}
	return devCmd, buildCmd
}

func hasAnyKeyword(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func guessJSRole(dirName string, role string, defaultPort int) (string, int) {
	lowerDir := strings.ToLower(dirName)
	if hasAnyKeyword(lowerDir, "admin", "backoffice", "dash") {
		role = models.RoleFrontendAdmin
		if defaultPort == 3000 {
			defaultPort = 3001
		}
	} else if hasAnyKeyword(lowerDir, "client", "store", "public", "web") {
		role = models.RoleFrontendClient
	} else if hasAnyKeyword(lowerDir, "api", "server", "backend") {
		role = models.RoleBackend
		if defaultPort == 3000 {
			defaultPort = 8080
		}
	}
	return role, defaultPort
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

	pm := detectPackageManager(dirPath)
	allDeps := make(map[string]bool)
	for k := range pkg.Dependencies {
		allDeps[k] = true
	}
	for k := range pkg.DevDeps {
		allDeps[k] = true
	}

	lang := "javascript"
	if util.FileExists(filepath.Join(dirPath, "tsconfig.json")) {
		lang = "typescript"
	}

	framework, role, defaultPort := detectJSFramework(allDeps)
	devCmd, buildCmd := detectJSScripts(pkg.Scripts, pm)
	role, defaultPort = guessJSRole(dirName, role, defaultPort)

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

func categorizeComponents(components []models.ComponentProposal) (bool, []int) {
	hasBackend := false
	var frontends []int
	for i, c := range components {
		if c.Role == models.RoleBackend {
			hasBackend = true
		} else if strings.HasPrefix(c.Role, "frontend") {
			frontends = append(frontends, i)
		}
	}
	return hasBackend, frontends
}

func disambiguateFrontends(components []models.ComponentProposal, frontends []int) {
	if len(frontends) < 2 {
		return
	}
	for _, idx := range frontends {
		if components[idx].Role == models.RoleFrontendAdmin {
			return
		}
	}
	for _, idx := range frontends {
		name := strings.ToLower(components[idx].Name)
		if strings.Contains(name, "admin") || strings.Contains(name, "manage") || strings.Contains(name, "back") {
			components[idx].Role = models.RoleFrontendAdmin
			return
		}
	}
	components[frontends[0]].Role = models.RoleFrontendAdmin
	if len(frontends) > 1 {
		components[frontends[1]].Role = models.RoleFrontendClient
	}
}

func determineScanProjectType(hasBackend bool, frontendsCount, compCount int) string {
	if hasBackend && frontendsCount >= 2 {
		return models.TypeMultiFE
	}
	if hasBackend && frontendsCount >= 1 {
		return models.TypeBundle
	}
	if compCount > 1 {
		return models.TypeMonorepo
	}
	return models.TypeSingleApp
}

func refineComponentRoles(result *models.ScanResult) {
	hasBackend, frontends := categorizeComponents(result.Components)
	disambiguateFrontends(result.Components, frontends)
	result.ProjectType = determineScanProjectType(hasBackend, len(frontends), len(result.Components))
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
