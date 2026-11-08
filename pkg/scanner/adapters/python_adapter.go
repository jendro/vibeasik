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

var (
	// FastAPI: @app.get("/path") or @router.post("/path")
	pyFastApiRegex = regexp.MustCompile(`@(?:app|router|api_router)\.(get|post|put|delete|patch)\(\s*['"]([^'"]+)['"]`)
	// Flask: @app.route("/path", methods=["GET", "POST"])
	pyFlaskRegex = regexp.MustCompile(`@app\.route\(\s*['"]([^'"]+)['"](?:\s*,\s*methods\s*=\s*\[([^\]]+)\])?`)
	// Django: path('users/', views.user_list)
	pyDjangoRegex = regexp.MustCompile(`path\(\s*['"]([^'"]*)['"]\s*,\s*([a-zA-Z0-9_.]+)`)
	// Function def following decorator
	pyDefRegex        = regexp.MustCompile(`(?:async\s+)?def\s+([a-zA-Z0-9_]+)\s*\(`)
	pyMethodListRegex = regexp.MustCompile(`['"](GET|POST|PUT|DELETE|PATCH)['"]`)
)

// pyRouteParser mem-parse satu berkas Python; rute FastAPI menunggu nama fungsi di baris def berikutnya.
type pyRouteParser struct {
	routes  []models.RouteInfo
	pending *models.RouteInfo
}

func ensureLeadingSlash(path string) string {
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

// resolvePending memberi nama handler pada rute FastAPI yang menunggu; true bila baris adalah def-nya.
func (p *pyRouteParser) resolvePending(line string) bool {
	if p.pending == nil {
		return false
	}
	dm := pyDefRegex.FindStringSubmatch(line)
	if len(dm) < 2 {
		return false
	}
	p.pending.Handler = dm[1]
	p.routes = append(p.routes, *p.pending)
	p.pending = nil
	return true
}

func (p *pyRouteParser) tryFastAPI(line string) bool {
	fam := pyFastApiRegex.FindStringSubmatch(line)
	if len(fam) < 3 {
		return false
	}
	method := strings.ToUpper(fam[1])
	path := ensureLeadingSlash(fam[2])

	middleware := "Public"
	if strings.Contains(line, "Depends(") && (strings.Contains(line, "auth") || strings.Contains(line, "token")) {
		middleware = "Protected (FastAPI Depends)"
	}
	p.pending = &models.RouteInfo{
		Method:      method,
		Path:        path,
		Handler:     "endpoint",
		Description: inferEndpointDescription(method, path),
		Middleware:  middleware,
	}
	return true
}

func (p *pyRouteParser) tryFlask(line string) bool {
	flm := pyFlaskRegex.FindStringSubmatch(line)
	if len(flm) < 2 {
		return false
	}
	path := ensureLeadingSlash(flm[1])
	methodsStr := "GET"
	if len(flm) >= 3 && flm[2] != "" {
		methodsStr = flm[2]
	}
	methods := pyMethodListRegex.FindAllStringSubmatch(methodsStr, -1)
	if len(methods) == 0 {
		methods = [][]string{{"", "GET"}}
	}
	for _, m := range methods {
		p.routes = append(p.routes, models.RouteInfo{
			Method:      strings.ToUpper(m[1]),
			Path:        path,
			Handler:     "view_func",
			Description: inferEndpointDescription(strings.ToUpper(m[1]), path),
			Middleware:  "Public",
		})
	}
	return true
}

func (p *pyRouteParser) tryDjango(line string) bool {
	djm := pyDjangoRegex.FindStringSubmatch(line)
	if len(djm) < 3 {
		return false
	}
	path := "/" + strings.Trim(djm[1], "/")
	p.routes = append(p.routes, models.RouteInfo{
		Method:      "ALL",
		Path:        path,
		Handler:     djm[2],
		Description: inferEndpointDescription("ALL", path),
		Middleware:  "Public",
	})
	return true
}

func (p *pyRouteParser) handleLine(line string) {
	if p.resolvePending(line) {
		return
	}
	for _, try := range []func(string) bool{p.tryFastAPI, p.tryFlask, p.tryDjango} {
		if try(line) {
			return
		}
	}
}

func parsePythonRoutesFromFile(filePath string) []models.RouteInfo {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	p := &pyRouteParser{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}
		p.handleLine(line)
	}
	return p.routes
}
