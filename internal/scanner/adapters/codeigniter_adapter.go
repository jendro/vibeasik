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

	"github.com/vibeasik/vibeasik/internal/models"
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
	if fileExists(filepath.Join(dir, "spark")) {
		return true
	}
	// 2. Check for CI4 route file
	if fileExists(filepath.Join(dir, "app/Config/Routes.php")) {
		return true
	}
	// 3. Check for CI3 route file
	if fileExists(filepath.Join(dir, "application/config/routes.php")) {
		return true
	}
	// 4. Check composer.json for CodeIgniter framework
	composerPath := filepath.Join(dir, "composer.json")
	if fileExists(composerPath) {
		content := readFileString(composerPath)
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
	if fileExists(sparkPath) {
		if routes, err := a.scanSparkRoutes(dir); err == nil && len(routes) > 0 {
			return routes, nil
		}
	}

	// 2. Fallback to static code parsing of app/Config/Routes.php (CI4) or application/config/routes.php (CI3)
	var allRoutes []models.RouteInfo
	seen := make(map[string]bool)

	ci4RoutesPath := filepath.Join(dir, "app/Config/Routes.php")
	if fileExists(ci4RoutesPath) {
		for _, r := range ParseCodeIgniter4RoutesFromFile(ci4RoutesPath) {
			key := strings.ToUpper(r.Method) + " " + r.Path
			if !seen[key] {
				seen[key] = true
				allRoutes = append(allRoutes, r)
			}
		}
	}

	ci3RoutesPath := filepath.Join(dir, "application/config/routes.php")
	if fileExists(ci3RoutesPath) {
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

// ParseSparkRoutesOutput parses ASCII/markdown table output from `php spark routes`
func ParseSparkRoutesOutput(output string) []models.RouteInfo {
	var routes []models.RouteInfo
	seen := make(map[string]bool)

	lines := strings.Split(output, "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			continue
		}
		parts := strings.Split(line, "|")
		// parts[0] is before first |, parts[len-1] is after last |
		if len(parts) < 6 {
			continue
		}

		method := strings.ToUpper(strings.TrimSpace(parts[1]))
		rawPath := strings.TrimSpace(parts[2])
		rawHandler := strings.TrimSpace(parts[4])
		beforeFilters := ""
		if len(parts) >= 6 {
			beforeFilters = strings.TrimSpace(parts[5])
		}

		// Skip header rows or divider rows
		if method == "METHOD" || strings.Contains(method, "-") || strings.Contains(method, "+") {
			continue
		}

		validMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true, "DELETE": true,
			"PATCH": true, "OPTIONS": true, "HEAD": true, "ALL": true,
		}
		if !validMethods[method] {
			continue
		}

		// Skip CORS preflight catchall options route: OPTIONS (.*) or OPTIONS (:any)
		if method == "OPTIONS" && (rawPath == "(.*)" || rawPath == "(:any)" || rawPath == "(.*)?") {
			continue
		}

		// Normalize route regex placeholders to CodeIgniter convention
		cleanPath := rawPath
		cleanPath = strings.ReplaceAll(cleanPath, "([0-9]+)", "(:num)")
		cleanPath = strings.ReplaceAll(cleanPath, "(.*)", "(:any)")
		cleanPath = CleanURLPath(cleanPath)

		// Clean handler
		handler := strings.TrimPrefix(rawHandler, "\\")
		if handler == "(Closure)" {
			handler = "Closure / Inline Action"
		}

		// Clean middleware
		middleware := beforeFilters
		if middleware == "" {
			middleware = "Public"
		}

		key := method + " " + cleanPath
		if !seen[key] {
			seen[key] = true
			routes = append(routes, models.RouteInfo{
				Method:      method,
				Path:        cleanPath,
				Handler:     handler,
				Description: inferEndpointDescription(method, cleanPath),
				Middleware:  middleware,
			})
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

// ParseCodeIgniter4RoutesFromFile extracts routes from a CodeIgniter 4 Routes.php file
func ParseCodeIgniter4RoutesFromFile(filePath string) []models.RouteInfo {
	var routes []models.RouteInfo
	seen := make(map[string]bool)

	file, err := os.Open(filePath)
	if err != nil {
		return routes
	}
	defer file.Close()

	var groupStack []ciGroupContext
	currentDepth := 0

	// Regexes
	groupRegex := regexp.MustCompile(`(?i)\$routes->group\(\s*['"]([^'"]*)['"]\s*(?:,\s*(\[[^\]]*\]))?`)
	matchRouteRegex := regexp.MustCompile(`(?i)\$routes->match\(\s*\[([^\]]+)\]\s*,\s*['"]([^'"]*)['"]\s*,\s*([^,\)]+)(?:,\s*(\[[^\]]*\]))?`)
	methodRouteRegex := regexp.MustCompile(`(?i)\$routes->(get|post|put|delete|patch|options|head|add)\(\s*['"]([^'"]*)['"]\s*,\s*([^,\)]+)(?:,\s*(\[[^\]]*\]))?`)
	resourceRouteRegex := regexp.MustCompile(`(?i)\$routes->(resource|presenter)\(\s*['"]([^'"]+)['"](?:\s*,\s*(\[[^\]]*\]))?`)

	optNamespaceRegex := regexp.MustCompile(`['"]namespace['"]\s*=>\s*['"]([^'"]+)['"]`)
	optFilterRegex := regexp.MustCompile(`['"]filters?['"]\s*=>\s*['"]([^'"]+)['"]`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*") {
			continue
		}

		// 1. Check for group opening
		if gm := groupRegex.FindStringSubmatch(line); len(gm) >= 2 {
			grpPrefix := gm[1]
			grpOpts := ""
			if len(gm) >= 3 {
				grpOpts = gm[2]
			}

			parentPrefix := ""
			parentNamespace := ""
			parentFilter := ""
			if len(groupStack) > 0 {
				parent := groupStack[len(groupStack)-1]
				parentPrefix = parent.prefix
				parentNamespace = parent.namespace
				parentFilter = parent.filter
			}

			effPrefix := combinePrefixAndPath(parentPrefix, grpPrefix)
			effNamespace := parentNamespace
			if nsm := optNamespaceRegex.FindStringSubmatch(grpOpts); len(nsm) >= 2 {
				effNamespace = strings.Trim(nsm[1], "\\")
			}

			effFilter := parentFilter
			if fm := optFilterRegex.FindStringSubmatch(grpOpts); len(fm) >= 2 {
				effFilter = fm[1]
			}

			groupStack = append(groupStack, ciGroupContext{
				prefix:    effPrefix,
				namespace: effNamespace,
				filter:    effFilter,
				depth:     currentDepth,
			})
		}

		// Track current context
		curPrefix := ""
		curNamespace := ""
		curFilter := "Public"
		if len(groupStack) > 0 {
			cur := groupStack[len(groupStack)-1]
			curPrefix = cur.prefix
			curNamespace = cur.namespace
			if cur.filter != "" {
				curFilter = cur.filter
			}
		}

		// 2. Check $routes->match(['put', 'post'], 'path', 'Handler', [opts])
		if mm := matchRouteRegex.FindStringSubmatch(line); len(mm) >= 4 {
			methodsRaw := mm[1]
			pathRaw := mm[2]
			handlerRaw := mm[3]
			optsRaw := ""
			if len(mm) >= 5 {
				optsRaw = mm[4]
			}

			fullPath := combinePrefixAndPath(curPrefix, pathRaw)
			handler := cleanCIHandler(handlerRaw, curNamespace)
			routeFilter := curFilter
			if fm := optFilterRegex.FindStringSubmatch(optsRaw); len(fm) >= 2 {
				routeFilter = fm[1]
			}

			// Split methods e.g. 'put', 'post'
			rawMethodParts := strings.Split(methodsRaw, ",")
			for _, rmp := range rawMethodParts {
				method := strings.ToUpper(strings.Trim(strings.TrimSpace(rmp), `'" `))
				if method == "" {
					continue
				}
				key := method + " " + fullPath
				if !seen[key] {
					seen[key] = true
					routes = append(routes, models.RouteInfo{
						Method:      method,
						Path:        fullPath,
						Handler:     handler,
						Description: inferEndpointDescription(method, fullPath),
						Middleware:  routeFilter,
					})
				}
			}
			updateDepthAndPop(&groupStack, line, &currentDepth)
			continue
		}

		// 3. Check $routes->get/post/put/delete/patch/options/head/add
		if rm := methodRouteRegex.FindStringSubmatch(line); len(rm) >= 4 {
			method := strings.ToUpper(rm[1])
			if method == "ADD" {
				method = "ALL"
			}
			pathRaw := rm[2]
			handlerRaw := rm[3]
			optsRaw := ""
			if len(rm) >= 5 {
				optsRaw = rm[4]
			}

			fullPath := combinePrefixAndPath(curPrefix, pathRaw)
			handler := cleanCIHandler(handlerRaw, curNamespace)
			routeFilter := curFilter
			if fm := optFilterRegex.FindStringSubmatch(optsRaw); len(fm) >= 2 {
				routeFilter = fm[1]
			}

			// Skip OPTIONS (:any) CORS catchall
			if method == "OPTIONS" && (pathRaw == "(:any)" || pathRaw == "(.*)" || fullPath == "/(:any)") {
				updateDepthAndPop(&groupStack, line, &currentDepth)
				continue
			}

			key := method + " " + fullPath
			if !seen[key] {
				seen[key] = true
				routes = append(routes, models.RouteInfo{
					Method:      method,
					Path:        fullPath,
					Handler:     handler,
					Description: inferEndpointDescription(method, fullPath),
					Middleware:  routeFilter,
				})
			}
			updateDepthAndPop(&groupStack, line, &currentDepth)
			continue
		}

		// 4. Check $routes->resource / presenter
		if rm := resourceRouteRegex.FindStringSubmatch(line); len(rm) >= 3 {
			resType := strings.ToLower(rm[1])
			resName := strings.Trim(rm[2], "/")
			optsRaw := ""
			if len(rm) >= 4 {
				optsRaw = rm[3]
			}

			ctrl := resName
			if cm := regexp.MustCompile(`['"]controller['"]\s*=>\s*['"]([^'"]+)['"]`).FindStringSubmatch(optsRaw); len(cm) >= 2 {
				ctrl = cm[1]
			}
			ctrlHandler := cleanCIHandler(ctrl, curNamespace)
			baseResPath := combinePrefixAndPath(curPrefix, resName)

			resourceRoutes := []struct {
				method string
				suffix string
				action string
			}{
				{"GET", "", "index"},
				{"POST", "", "create"},
				{"GET", "/(:segment)", "show"},
				{"PUT", "/(:segment)", "update"},
				{"DELETE", "/(:segment)", "delete"},
			}
			if resType == "presenter" {
				resourceRoutes = append(resourceRoutes,
					struct {
						method string
						suffix string
						action string
					}{"GET", "/new", "new"},
					struct {
						method string
						suffix string
						action string
					}{"GET", "/(:segment)/edit", "edit"},
				)
			}

			for _, rr := range resourceRoutes {
				p := baseResPath + rr.suffix
				key := rr.method + " " + p
				if !seen[key] {
					seen[key] = true
					routes = append(routes, models.RouteInfo{
						Method:      rr.method,
						Path:        p,
						Handler:     fmt.Sprintf("%s::%s", ctrlHandler, rr.action),
						Description: inferEndpointDescription(rr.method, p),
						Middleware:  curFilter,
					})
				}
			}
			updateDepthAndPop(&groupStack, line, &currentDepth)
			continue
		}

		// Update bracket depth and pop groups if closed
		updateDepthAndPop(&groupStack, line, &currentDepth)
	}

	return routes
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
