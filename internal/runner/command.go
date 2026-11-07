package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/vibeasik/vibeasik/internal/models"
)

// LaunchSpec is a component dev command translated so the app listens on the port vibeasik picked.
type LaunchSpec struct {
	Command string   `json:"command"`
	Env     []string `json:"-"`
	Notes   []string `json:"notes,omitempty"`
}

var (
	// --port 3000, --port=3000, --web-port 3000, -p 3000 (yang terakhir dilewati untuk docker)
	portFlagRe     = regexp.MustCompile(`(--(?:web-)?port[= ]|(?:^|\s)-p\s+)(\d{2,5})\b`)
	longPortFlagRe = regexp.MustCompile(`(--(?:web-)?port[= ])(\d{2,5})\b`)
	inlineEnvRe    = regexp.MustCompile(`\b(PORT|NUXT_PORT|NITRO_PORT)=\d{2,5}\b`)
	runserverRe    = regexp.MustCompile(`(runserver\s+(?:[\w.\-\[\]:]+:)?)(\d{2,5})\b`)
	pkgScriptRe    = regexp.MustCompile(`^\s*(npm|pnpm|yarn|bun)\s+(?:run\s+)?([\w:.\-]+)\s*(.*)$`)
	slugRe         = regexp.MustCompile(`[^A-Z0-9]+`)
)

// pkgBuiltins are package-manager subcommands that are not package.json scripts.
var pkgBuiltins = map[string]bool{"install": true, "i": true, "add": true, "exec": true, "x": true, "dlx": true, "create": true}

// BuildLaunch rewrites comp.DevCommand for port and returns the env to inject. Siblings are the
// other components of the same project, exposed as VIBEASIK_PORT_<NAME> so a frontend can find its backend.
func BuildLaunch(comp models.Component, port int, siblings []models.Component) LaunchSpec {
	spec := LaunchSpec{Command: strings.TrimSpace(comp.DevCommand)}
	p := strconv.Itoa(port)

	spec.Env = []string{
		"PORT=" + p,
		"NUXT_PORT=" + p,
		"NITRO_PORT=" + p,
		"BROWSER=none",
		"VIBEASIK_RUN=1",
	}
	backendURL := ""
	for _, s := range siblings {
		if s.ID == comp.ID || s.RunPort <= 0 {
			continue
		}
		spec.Env = append(spec.Env, fmt.Sprintf("VIBEASIK_PORT_%s=%d", envSlug(s.Name), s.RunPort))
		if backendURL == "" && s.Role == models.RoleBackend {
			backendURL = fmt.Sprintf("http://localhost:%d", s.RunPort)
		}
	}
	if backendURL != "" {
		spec.Env = append(spec.Env, "VIBEASIK_BACKEND_URL="+backendURL)
	}

	if rewritten, changed := rewritePorts(spec.Command, p); changed {
		spec.Command = rewritten
		spec.Notes = append(spec.Notes, fmt.Sprintf("Port hardcoded di dev command diganti ke :%d", port))
		return spec
	}

	// Script package.json: lihat isi script-nya untuk tahu tool apa yang dijalankan.
	if m := pkgScriptRe.FindStringSubmatch(spec.Command); m != nil && !pkgBuiltins[m[2]] {
		pm, script, extra := m[1], m[2], m[3]
		body := readPackageScript(comp.AbsolutePath, script)
		if body == "" {
			return spec
		}
		if rewritten, changed := rewritePorts(body, p); changed {
			// Script meng-hardcode port: jalankan isinya langsung (seperti `npm run`) dengan port baru.
			binDir := filepath.Join(comp.AbsolutePath, "node_modules", ".bin")
			spec.Env = append(spec.Env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			spec.Command = strings.TrimSpace(rewritten + " " + extra)
			spec.Notes = append(spec.Notes, fmt.Sprintf("Script '%s' di package.json meng-hardcode port; dijalankan langsung dengan :%d", script, port))
			return spec
		}
		if flag := portFlagFor(detectTool(body)); flag != "" {
			sep := " "
			if pm == "npm" {
				sep = " -- "
			}
			spec.Command = spec.Command + sep + fmt.Sprintf(flag, port)
			spec.Notes = append(spec.Notes, fmt.Sprintf("Flag port ditambahkan untuk %s", detectTool(body)))
		}
		return spec
	}

	if flag := portFlagFor(detectTool(spec.Command)); flag != "" {
		spec.Command = spec.Command + " " + fmt.Sprintf(flag, port)
		spec.Notes = append(spec.Notes, fmt.Sprintf("Flag port ditambahkan untuk %s", detectTool(spec.Command)))
	}
	return spec
}

// rewritePorts replaces explicit port arguments and inline PORT=… assignments with port.
func rewritePorts(cmd, port string) (string, bool) {
	out := inlineEnvRe.ReplaceAllString(cmd, "${1}="+port)
	out = runserverRe.ReplaceAllString(out, "${1}"+port)
	if strings.Contains(out, "docker") {
		out = longPortFlagRe.ReplaceAllString(out, "${1}"+port) // -p milik docker berarti publish, bukan port dev
	} else {
		out = portFlagRe.ReplaceAllString(out, "${1}"+port)
	}
	return out, out != cmd
}

// detectTool names the dev server binary a command (or package.json script body) launches.
func detectTool(cmd string) string {
	c := " " + strings.ToLower(cmd) + " "
	switch {
	case strings.Contains(c, " nuxt ") || strings.Contains(c, " nuxi ") || strings.Contains(c, "nuxt dev"):
		return "nuxt"
	case strings.Contains(c, " next ") || strings.Contains(c, "next dev"):
		return "next"
	case strings.Contains(c, " astro "):
		return "astro"
	case strings.Contains(c, " vite ") || strings.Contains(c, "vite dev"):
		return "vite"
	case strings.Contains(c, "ng serve"):
		return "angular"
	case strings.Contains(c, "artisan serve"):
		return "laravel"
	case strings.Contains(c, "runserver"):
		return "django"
	case strings.Contains(c, "uvicorn "):
		return "uvicorn"
	case strings.Contains(c, "flask run"):
		return "flask"
	}
	return ""
}

// portFlagFor returns a printf template that makes the tool listen on a port; empty means PORT env suffices.
func portFlagFor(tool string) string {
	switch tool {
	case "vite":
		return "--port %d --strictPort"
	case "nuxt", "next", "astro", "angular", "uvicorn", "flask":
		return "--port %d"
	case "laravel":
		return "--port=%d"
	case "django":
		return "%d"
	}
	return ""
}

func readPackageScript(dir, script string) string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return ""
	}
	return pkg.Scripts[script]
}

func envSlug(name string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToUpper(name), "_"), "_")
}
