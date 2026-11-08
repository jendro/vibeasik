package adapters

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
)

type CodeIgniterAdapter struct{}

func NewCodeIgniterAdapter() *CodeIgniterAdapter {
	return &CodeIgniterAdapter{}
}

func (a *CodeIgniterAdapter) Name() string {
	return "PHP / CodeIgniter"
}

func (a *CodeIgniterAdapter) Type() string {
	return "backend"
}

func (a *CodeIgniterAdapter) Detect(dir string) bool {
	// 1. Check for spark (CodeIgniter 4 CLI)
	if util.FileExists(filepath.Join(dir, "spark")) {
		return true
	}
	// 2. Check for CI4 route file
	if util.FileExists(filepath.Join(dir, "app/Config/Routes.php")) {
		return true
	}
	// 3. Check for CI3 route file
	if util.FileExists(filepath.Join(dir, "application/config/routes.php")) {
		return true
	}
	// 4. Check composer.json for CodeIgniter framework
	composerPath := filepath.Join(dir, "composer.json")
	if util.FileExists(composerPath) {
		content := util.ReadFileString(composerPath)
		if strings.Contains(content, "codeigniter4/framework") ||
			strings.Contains(content, "codeigniter4/appstarter") ||
			strings.Contains(content, "codeigniter") {
			return true
		}
	}
	return false
}

func (a *CodeIgniterAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	// 1. If spark exists, try executing `php spark routes`
	sparkPath := filepath.Join(dir, "spark")
	if util.FileExists(sparkPath) {
		if routes, err := a.scanSparkRoutes(dir); err == nil && len(routes) > 0 {
			return routes, nil
		}
	}

	// 2. Fallback to static code parsing of app/Config/Routes.php (CI4) or application/config/routes.php (CI3)
	var allRoutes []models.RouteInfo
	seen := make(map[string]bool)

	ci4RoutesPath := filepath.Join(dir, "app/Config/Routes.php")
	if util.FileExists(ci4RoutesPath) {
		for _, r := range ParseCodeIgniter4RoutesFromFile(ci4RoutesPath) {
			key := strings.ToUpper(r.Method) + " " + r.Path
			if !seen[key] {
				seen[key] = true
				allRoutes = append(allRoutes, r)
			}
		}
	}

	ci3RoutesPath := filepath.Join(dir, "application/config/routes.php")
	if util.FileExists(ci3RoutesPath) {
		for _, r := range ParseCodeIgniter3RoutesFromFile(ci3RoutesPath) {
			key := strings.ToUpper(r.Method) + " " + r.Path
			if !seen[key] {
				seen[key] = true
				allRoutes = append(allRoutes, r)
			}
		}
	}

	return allRoutes, nil
}

// scanSparkRoutes executes `php spark routes` and parses its output table
func (a *CodeIgniterAdapter) scanSparkRoutes(dir string) ([]models.RouteInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "php", "spark", "routes")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return ParseSparkRoutesOutput(string(out)), nil
}

var sparkValidMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"PATCH": true, "OPTIONS": true, "HEAD": true, "ALL": true,
}

// isSparkSkippableRow: baris header/pembatas, metode tak dikenal, atau catchall OPTIONS (CORS preflight).
func isSparkSkippableRow(method, rawPath string) bool {
	if method == "METHOD" || strings.Contains(method, "-") || strings.Contains(method, "+") {
		return true
	}
	if !sparkValidMethods[method] {
		return true
	}
	return method == "OPTIONS" && (rawPath == "(.*)" || rawPath == "(:any)" || rawPath == "(.*)?")
}

// parseSparkRow mengubah satu baris tabel spark menjadi rute; ok=false bila baris dilewati.
func parseSparkRow(line string) (models.RouteInfo, bool) {
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return models.RouteInfo{}, false
	}
	parts := strings.Split(line, "|")
	// parts[0] is before first |, parts[len-1] is after last |
	if len(parts) < 6 {
		return models.RouteInfo{}, false
	}
	method := strings.ToUpper(strings.TrimSpace(parts[1]))
	rawPath := strings.TrimSpace(parts[2])
	rawHandler := strings.TrimSpace(parts[4])
	beforeFilters := strings.TrimSpace(parts[5])
	if isSparkSkippableRow(method, rawPath) {
		return models.RouteInfo{}, false
	}

	// Normalize route regex placeholders to CodeIgniter convention
	cleanPath := strings.ReplaceAll(rawPath, "([0-9]+)", "(:num)")
	cleanPath = strings.ReplaceAll(cleanPath, "(.*)", "(:any)")
	cleanPath = CleanURLPath(cleanPath)

	handler := strings.TrimPrefix(rawHandler, "\\")
	if handler == "(Closure)" {
		handler = "Closure / Inline Action"
	}
	middleware := beforeFilters
	if middleware == "" {
		middleware = "Public"
	}
	return models.RouteInfo{
		Method:      method,
		Path:        cleanPath,
		Handler:     handler,
		Description: inferEndpointDescription(method, cleanPath),
		Middleware:  middleware,
	}, true
}

// ParseSparkRoutesOutput parses ASCII/markdown table output from `php spark routes`
func ParseSparkRoutesOutput(output string) []models.RouteInfo {
	var routes []models.RouteInfo
	seen := make(map[string]bool)

	for _, rawLine := range strings.Split(output, "\n") {
		route, ok := parseSparkRow(strings.TrimSpace(rawLine))
		if !ok {
			continue
		}
		key := route.Method + " " + route.Path
		if !seen[key] {
			seen[key] = true
			routes = append(routes, route)
		}
	}
	return routes
}

type ciGroupContext struct {
	prefix    string
	namespace string
	filter    string
	depth     int
}

var (
	ciGroupRegex         = regexp.MustCompile(`(?i)\$routes->group\(\s*['"]([^'"]*)['"]\s*(?:,\s*(\[[^\]]*\]))?`)
	ciMatchRouteRegex    = regexp.MustCompile(`(?i)\$routes->match\(\s*\[([^\]]+)\]\s*,\s*['"]([^'"]*)['"]\s*,\s*([^,\)]+)(?:,\s*(\[[^\]]*\]))?`)
	ciMethodRouteRegex   = regexp.MustCompile(`(?i)\$routes->(get|post|put|delete|patch|options|head|add)\(\s*['"]([^'"]*)['"]\s*,\s*([^,\)]+)(?:,\s*(\[[^\]]*\]))?`)
	ciResourceRouteRegex = regexp.MustCompile(`(?i)\$routes->(resource|presenter)\(\s*['"]([^'"]+)['"](?:\s*,\s*(\[[^\]]*\]))?`)
	ciOptNamespaceRegex  = regexp.MustCompile(`['"]namespace['"]\s*=>\s*['"]([^'"]+)['"]`)
	ciOptFilterRegex     = regexp.MustCompile(`['"]filters?['"]\s*=>\s*['"]([^'"]+)['"]`)
	ciOptControllerRegex = regexp.MustCompile(`['"]controller['"]\s*=>\s*['"]([^'"]+)['"]`)
)

type ciResourceRoute struct {
	method string
	suffix string
	action string
}

var ciResourceRoutes = []ciResourceRoute{
	{"GET", "", "index"},
	{"POST", "", "create"},
	{"GET", "/(:segment)", "show"},
	{"PUT", "/(:segment)", "update"},
	{"DELETE", "/(:segment)", "delete"},
}

var ciPresenterExtraRoutes = []ciResourceRoute{
	{"GET", "/new", "new"},
	{"GET", "/(:segment)/edit", "edit"},
}

// ciParser = keadaan satu kali parse Routes.php CodeIgniter 4.
type ciParser struct {
	routes       []models.RouteInfo
	seen         map[string]bool
	groupStack   []ciGroupContext
	currentDepth int
}

// add menambahkan rute bila kombinasi metode+path belum ada.
func (p *ciParser) add(method, path, handler, middleware string) {
	key := method + " " + path
	if p.seen[key] {
		return
	}
	p.seen[key] = true
	p.routes = append(p.routes, models.RouteInfo{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Description: inferEndpointDescription(method, path),
		Middleware:  middleware,
	})
}

// openGroup mendorong konteks grup baru bila baris adalah pembuka $routes->group.
func (p *ciParser) openGroup(line string) {
	gm := ciGroupRegex.FindStringSubmatch(line)
	if len(gm) < 2 {
		return
	}
	grpOpts := ""
	if len(gm) >= 3 {
		grpOpts = gm[2]
	}

	var parent ciGroupContext
	if len(p.groupStack) > 0 {
		parent = p.groupStack[len(p.groupStack)-1]
	}
	effNamespace := parent.namespace
	if nsm := ciOptNamespaceRegex.FindStringSubmatch(grpOpts); len(nsm) >= 2 {
		effNamespace = strings.Trim(nsm[1], "\\")
	}
	effFilter := parent.filter
	if fm := ciOptFilterRegex.FindStringSubmatch(grpOpts); len(fm) >= 2 {
		effFilter = fm[1]
	}
	p.groupStack = append(p.groupStack, ciGroupContext{
		prefix:    combinePrefixAndPath(parent.prefix, gm[1]),
		namespace: effNamespace,
		filter:    effFilter,
		depth:     p.currentDepth,
	})
}

// context mengembalikan prefix, namespace, dan filter aktif (default filter "Public").
func (p *ciParser) context() (prefix, namespace, filter string) {
	filter = "Public"
	if len(p.groupStack) > 0 {
		cur := p.groupStack[len(p.groupStack)-1]
		prefix, namespace = cur.prefix, cur.namespace
		if cur.filter != "" {
			filter = cur.filter
		}
	}
	return
}

// routeFilterOf mengambil filter dari opsi rute, atau fallback ke filter grup.
func routeFilterOf(optsRaw, fallback string) string {
	if fm := ciOptFilterRegex.FindStringSubmatch(optsRaw); len(fm) >= 2 {
		return fm[1]
	}
	return fallback
}

// optsAt mengambil grup regex ke-i bila ada, atau "".
func optsAt(m []string, i int) string {
	if len(m) > i {
		return m[i]
	}
	return ""
}

// tryMatch: $routes->match(['put', 'post'], 'path', 'Handler', [opts])
func (p *ciParser) tryMatch(line string) bool {
	mm := ciMatchRouteRegex.FindStringSubmatch(line)
	if len(mm) < 4 {
		return false
	}
	curPrefix, curNamespace, curFilter := p.context()
	fullPath := combinePrefixAndPath(curPrefix, mm[2])
	handler := cleanCIHandler(mm[3], curNamespace)
	routeFilter := routeFilterOf(optsAt(mm, 4), curFilter)

	// Split methods e.g. 'put', 'post'
	for _, rmp := range strings.Split(mm[1], ",") {
		method := strings.ToUpper(strings.Trim(strings.TrimSpace(rmp), `'" `))
		if method == "" {
			continue
		}
		p.add(method, fullPath, handler, routeFilter)
	}
	return true
}

// tryMethod: $routes->get/post/put/delete/patch/options/head/add
func (p *ciParser) tryMethod(line string) bool {
	rm := ciMethodRouteRegex.FindStringSubmatch(line)
	if len(rm) < 4 {
		return false
	}
	method := strings.ToUpper(rm[1])
	if method == "ADD" {
		method = "ALL"
	}
	pathRaw := rm[2]
	curPrefix, curNamespace, curFilter := p.context()
	fullPath := combinePrefixAndPath(curPrefix, pathRaw)

	// Skip OPTIONS (:any) CORS catchall
	if method == "OPTIONS" && (pathRaw == "(:any)" || pathRaw == "(.*)" || fullPath == "/(:any)") {
		return true
	}
	p.add(method, fullPath, cleanCIHandler(rm[3], curNamespace), routeFilterOf(optsAt(rm, 4), curFilter))
	return true
}

// tryResource: $routes->resource / presenter
func (p *ciParser) tryResource(line string) bool {
	rm := ciResourceRouteRegex.FindStringSubmatch(line)
	if len(rm) < 3 {
		return false
	}
	resType := strings.ToLower(rm[1])
	resName := strings.Trim(rm[2], "/")
	optsRaw := optsAt(rm, 3)
	curPrefix, curNamespace, curFilter := p.context()

	ctrl := resName
	if cm := ciOptControllerRegex.FindStringSubmatch(optsRaw); len(cm) >= 2 {
		ctrl = cm[1]
	}
	ctrlHandler := cleanCIHandler(ctrl, curNamespace)
	baseResPath := combinePrefixAndPath(curPrefix, resName)

	resourceRoutes := ciResourceRoutes
	if resType == "presenter" {
		resourceRoutes = append(append([]ciResourceRoute(nil), ciResourceRoutes...), ciPresenterExtraRoutes...)
	}
	for _, rr := range resourceRoutes {
		p.add(rr.method, baseResPath+rr.suffix, fmt.Sprintf("%s::%s", ctrlHandler, rr.action), curFilter)
	}
	return true
}

// handleLine memproses satu baris; pola pertama yang cocok menang. Kedalaman kurung diperbarui sekali per baris.
func (p *ciParser) handleLine(line string) {
	p.openGroup(line)
	for _, try := range []func(string) bool{p.tryMatch, p.tryMethod, p.tryResource} {
		if try(line) {
			break
		}
	}
	updateDepthAndPop(&p.groupStack, line, &p.currentDepth)
}

func isCIComment(line string) bool {
	return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*")
}

// ParseCodeIgniter4RoutesFromFile extracts routes from a CodeIgniter 4 Routes.php file
func ParseCodeIgniter4RoutesFromFile(filePath string) []models.RouteInfo {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	p := &ciParser{seen: make(map[string]bool)}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if isCIComment(line) {
			continue
		}
		p.handleLine(line)
	}
	return p.routes
}

// ParseCodeIgniter3RoutesFromFile extracts routes from a CodeIgniter 3 application/config/routes.php file
func ParseCodeIgniter3RoutesFromFile(filePath string) []models.RouteInfo {
	var routes []models.RouteInfo
	seen := make(map[string]bool)

	file, err := os.Open(filePath)
	if err != nil {
		return routes
	}
	defer file.Close()

	routeVerbRegex := regexp.MustCompile(`(?i)\$route\['([^']+)'\]\['(get|post|put|delete|patch|head|options)'\]\s*=\s*['"]([^'"]+)['"]`)
	routeStandardRegex := regexp.MustCompile(`(?i)\$route\['([^']+)'\]\s*=\s*['"]([^'"]+)['"]`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "/*") {
			continue
		}

		// Verb specific: $route['user']['post'] = 'user/create';
		if vm := routeVerbRegex.FindStringSubmatch(line); len(vm) >= 4 {
			p := CleanURLPath(vm[1])
			method := strings.ToUpper(vm[2])
			handler := vm[3]
			key := method + " " + p
			if !seen[key] {
				seen[key] = true
				routes = append(routes, models.RouteInfo{
					Method:      method,
					Path:        p,
					Handler:     handler,
					Description: inferEndpointDescription(method, p),
					Middleware:  "Public",
				})
			}
			continue
		}

		// Standard: $route['api/users'] = 'api/users/index';
		if sm := routeStandardRegex.FindStringSubmatch(line); len(sm) >= 3 {
			routeKey := sm[1]
			handler := sm[2]

			// Ignore CI3 system configuration keys
			if routeKey == "default_controller" || routeKey == "404_override" || routeKey == "translate_uri_dashes" {
				continue
			}

			p := CleanURLPath(routeKey)
			key := "GET " + p
			if !seen[key] {
				seen[key] = true
				routes = append(routes, models.RouteInfo{
					Method:      "GET",
					Path:        p,
					Handler:     handler,
					Description: inferEndpointDescription("GET", p),
					Middleware:  "Public",
				})
			}
		}
	}

	return routes
}

func updateDepthAndPop(groupStack *[]ciGroupContext, line string, currentDepth *int) {
	openCount := strings.Count(line, "{")
	closeCount := strings.Count(line, "}")
	*currentDepth += (openCount - closeCount)

	for len(*groupStack) > 0 && *currentDepth <= (*groupStack)[len(*groupStack)-1].depth {
		*groupStack = (*groupStack)[:len(*groupStack)-1]
	}
}

func cleanCIHandler(raw, namespace string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ")")
	raw = strings.TrimSuffix(raw, ";")
	raw = strings.Trim(raw, `'" `)

	if strings.Contains(raw, "function") || strings.Contains(raw, "fn(") || strings.Contains(raw, "Closure") {
		return "Closure / Inline Action"
	}

	// Remove trailing param placeholders like /$1, /$2
	if idx := strings.Index(raw, "/$"); idx != -1 {
		raw = raw[:idx]
	}

	raw = strings.TrimPrefix(raw, "\\")

	// If relative handler and namespace provided, prepend namespace
	if namespace != "" && !strings.Contains(raw, "\\") && !strings.HasPrefix(raw, namespace) {
		raw = namespace + "\\" + raw
	}

	return raw
}
