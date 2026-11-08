package adapters

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
)

type ExpressNestAdapter struct{}

func NewExpressNestAdapter() *ExpressNestAdapter {
	return &ExpressNestAdapter{}
}

func (a *ExpressNestAdapter) Name() string {
	return "Node.js (Express / NestJS / Fastify)"
}

func (a *ExpressNestAdapter) Type() string {
	return "backend"
}

func (a *ExpressNestAdapter) Detect(dir string) bool {
	pkgPath := filepath.Join(dir, "package.json")
	if !util.FileExists(pkgPath) {
		return false
	}
	content := util.ReadFileString(pkgPath)
	return strings.Contains(content, `"express"`) ||
		strings.Contains(content, `"@nestjs/core"`) ||
		strings.Contains(content, `"fastify"`)
}

func (a *ExpressNestAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	var routes []models.RouteInfo
	scannedFiles := make(map[string]bool)

	// 1. Scan NestJS controllers (*.controller.ts)
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldIgnoreName(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".controller.ts") || strings.HasSuffix(d.Name(), ".controller.js") {
			scannedFiles[path] = true
			r := parseNestJsController(path)
			routes = append(routes, r...)
		}
		return nil
	})

	// 2. Scan Express / Fastify files
	candidateFiles := []string{
		"src/app.ts", "src/app.js", "app.ts", "app.js",
		"src/server.ts", "src/server.js", "server.ts", "server.js",
		"src/index.ts", "src/index.js", "index.ts", "index.js",
	}
	for _, cf := range candidateFiles {
		p := filepath.Join(dir, cf)
		if util.FileExists(p) && !scannedFiles[p] {
			scannedFiles[p] = true
			r := parseExpressRoutes(p)
			routes = append(routes, r...)
		}
	}

	// Scan routes/ and src/routes/
	for _, sub := range []string{"routes", "src/routes", "api", "src/api"} {
		subDir := filepath.Join(dir, sub)
		if dirExists(subDir) {
			_ = filepath.WalkDir(subDir, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() || scannedFiles[path] {
					return nil
				}
				ext := filepath.Ext(path)
				if ext == ".ts" || ext == ".js" {
					scannedFiles[path] = true
					r := parseExpressRoutes(path)
					routes = append(routes, r...)
				}
				return nil
			})
		}
	}

	return routes, nil
}

func parseNestJsController(filePath string) []models.RouteInfo {
	var routes []models.RouteInfo
	content := util.ReadFileString(filePath)

	ctrlRegex := regexp.MustCompile(`@Controller\(\s*['"]?([^'")]*)['"]?\s*\)`)
	ctrlPrefix := ""
	if m := ctrlRegex.FindStringSubmatch(content); len(m) >= 2 {
		ctrlPrefix = strings.Trim(m[1], "/")
		if ctrlPrefix != "" {
			ctrlPrefix = "/" + ctrlPrefix
		}
	}

	// Methods: @Get, @Post, @Put, @Delete, @Patch
	methodRegex := regexp.MustCompile(`(?m)@(Get|Post|Put|Delete|Patch)\(\s*['"]?([^'")]*)['"]?\s*\)\s*(?:@[a-zA-Z0-9_()]+\s*)*\s*(?:async\s+)?([a-zA-Z0-9_]+)\s*\(`)
	matches := methodRegex.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		httpMethod := strings.ToUpper(m[1])
		subPath := strings.Trim(m[2], "/")
		handlerName := m[3]

		fullPath := ctrlPrefix
		if subPath != "" {
			fullPath = strings.TrimRight(ctrlPrefix, "/") + "/" + subPath
		}
		if fullPath == "" {
			fullPath = "/"
		}

		middleware := "Public"
		if strings.Contains(content, "@UseGuards") {
			middleware = "Protected (Nest Guard)"
		}

		routes = append(routes, models.RouteInfo{
			Method:      httpMethod,
			Path:        fullPath,
			Handler:     handlerName,
			Description: inferEndpointDescription(httpMethod, fullPath),
			Middleware:  middleware,
		})
	}

	return routes
}

func parseExpressRoutes(filePath string) []models.RouteInfo {
	var routes []models.RouteInfo
	file, err := os.Open(filePath)
	if err != nil {
		return routes
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	routeRegex := regexp.MustCompile(`(?:app|router|api)\.(get|post|put|delete|patch)\(\s*['"]([^'"]+)['"]\s*,\s*([^)]+)\)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}

		if m := routeRegex.FindStringSubmatch(line); len(m) >= 4 {
			method := strings.ToUpper(m[1])
			path := m[2]
			handler := strings.TrimSpace(m[3])
			handler = strings.TrimSuffix(handler, ");")
			handler = strings.TrimSuffix(handler, ")")

			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}

			middleware := "Public"
			if strings.Contains(line, "auth") || strings.Contains(line, "verify") || strings.Contains(line, "protect") {
				middleware = "Protected (Auth Middleware)"
			}

			routes = append(routes, models.RouteInfo{
				Method:      method,
				Path:        path,
				Handler:     handler,
				Description: inferEndpointDescription(method, path),
				Middleware:  middleware,
			})
		}
	}

	return routes
}
