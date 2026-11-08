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

type PythonAdapter struct{}

func NewPythonAdapter() *PythonAdapter {
	return &PythonAdapter{}
}

func (a *PythonAdapter) Name() string {
	return "Python (FastAPI / Flask / Django)"
}

func (a *PythonAdapter) Type() string {
	return "backend"
}

func (a *PythonAdapter) Detect(dir string) bool {
	for _, f := range []string{"requirements.txt", "pyproject.toml", "Pipfile", "setup.py", "manage.py"} {
		if util.FileExists(filepath.Join(dir, f)) {
			return true
		}
	}
	return false
}

func (a *PythonAdapter) Scan(dir string) ([]models.RouteInfo, error) {
	var routes []models.RouteInfo

	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldIgnoreName(d.Name()) || d.Name() == "venv" || d.Name() == ".venv" || d.Name() == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".py") && !strings.HasSuffix(d.Name(), "_test.py") {
			r := parsePythonRoutesFromFile(path)
			routes = append(routes, r...)
		}
		return nil
	})

	return routes, nil
}

func parsePythonRoutesFromFile(filePath string) []models.RouteInfo {
	var routes []models.RouteInfo
	file, err := os.Open(filePath)
	if err != nil {
		return routes
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// FastAPI: @app.get("/path") or @router.post("/path")
	fastApiRegex := regexp.MustCompile(`@(?:app|router|api_router)\.(get|post|put|delete|patch)\(\s*['"]([^'"]+)['"]`)
	// Flask: @app.route("/path", methods=["GET", "POST"])
	flaskRegex := regexp.MustCompile(`@app\.route\(\s*['"]([^'"]+)['"](?:\s*,\s*methods\s*=\s*\[([^\]]+)\])?`)
	// Django: path('users/', views.user_list)
	djangoRegex := regexp.MustCompile(`path\(\s*['"]([^'"]*)['"]\s*,\s*([a-zA-Z0-9_.]+)`)
	// Function def following decorator
	defRegex := regexp.MustCompile(`(?:async\s+)?def\s+([a-zA-Z0-9_]+)\s*\(`)

	var pendingRoute *models.RouteInfo

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Check if pending route gets its handler name from `def func_name(`
		if pendingRoute != nil {
			if dm := defRegex.FindStringSubmatch(line); len(dm) >= 2 {
				pendingRoute.Handler = dm[1]
				routes = append(routes, *pendingRoute)
				pendingRoute = nil
				continue
			}
		}

		// 1. FastAPI
		if fam := fastApiRegex.FindStringSubmatch(line); len(fam) >= 3 {
			method := strings.ToUpper(fam[1])
			path := fam[2]
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}

			middleware := "Public"
			if strings.Contains(line, "Depends(") && (strings.Contains(line, "auth") || strings.Contains(line, "token")) {
				middleware = "Protected (FastAPI Depends)"
			}

			pendingRoute = &models.RouteInfo{
				Method:      method,
				Path:        path,
				Handler:     "endpoint",
				Description: inferEndpointDescription(method, path),
				Middleware:  middleware,
			}
			continue
		}

		// 2. Flask
		if flm := flaskRegex.FindStringSubmatch(line); len(flm) >= 2 {
			path := flm[1]
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			methodsStr := "GET"
			if len(flm) >= 3 && flm[2] != "" {
				methodsStr = flm[2]
			}

			methodListRegex := regexp.MustCompile(`['"](GET|POST|PUT|DELETE|PATCH)['"]`)
			methods := methodListRegex.FindAllStringSubmatch(methodsStr, -1)
			if len(methods) == 0 {
				methods = [][]string{{"", "GET"}}
			}

			for _, m := range methods {
				routes = append(routes, models.RouteInfo{
					Method:      strings.ToUpper(m[1]),
					Path:        path,
					Handler:     "view_func",
					Description: inferEndpointDescription(strings.ToUpper(m[1]), path),
					Middleware:  "Public",
				})
			}
			continue
		}

		// 3. Django
		if djm := djangoRegex.FindStringSubmatch(line); len(djm) >= 3 {
			path := "/" + strings.Trim(djm[1], "/")
			handler := djm[2]
			routes = append(routes, models.RouteInfo{
				Method:      "ALL",
				Path:        path,
				Handler:     handler,
				Description: inferEndpointDescription("ALL", path),
				Middleware:  "Public",
			})
		}
	}

	return routes
}
