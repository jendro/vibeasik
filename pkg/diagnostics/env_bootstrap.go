package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Berkas env yang tidak masuk git dibuat engine langsung (Aturan 45), bukan agent: agent bekerja di worktree
// dan hasilnya masuk lewat git, sehingga berkas yang di-ignore tidak pernah sampai ke folder proyek.
//   .env.local  = salinan env dev sekarang (.env), atau .env.example bila .env belum ada
//   .env.server = dari .env.example; nilai rahasia & URL localhost dikosongkan untuk diisi pengguna
// Berkas hanya dibuat bila git memastikan namanya ter-ignore (atau folder di luar git), dan tidak pernah ditimpa.
// Port di .env / .env.local (yang tidak di-track git) diselaraskan dengan run_port blok PORTS.md.

var (
	envSecretKey  = regexp.MustCompile(`(?i)(PASS|SECRET|TOKEN|KEY|PRIVATE|CREDENTIAL|SALT|DSN)`)
	envLineRe     = regexp.MustCompile(`^(\s*(?:export\s+)?)([A-Za-z_][A-Za-z0-9_.]*)(\s*=\s*)(.*)$`)
	envLocalURLRe = regexp.MustCompile(`(https?://(?:localhost|127\.0\.0\.1|0\.0\.0\.0))(?::(\d+))?`)
)

const (
	envLocalHeader  = "# Env dev laptop ini (dibuat vibeasik dari %s). Tidak di-commit; deploy.sh memulihkan .env dari berkas ini.\n"
	envServerHeader = "# Env server untuk ./deploy.sh (dibuat vibeasik dari %s). Isi nilai yang kosong; berkas ini tidak di-commit.\n"
)

// EnsureEnvFiles membuat .env.local & .env.server yang belum ada dan menyelaraskan port env dengan run_port.
// Mengembalikan daftar tindakan (untuk log/todo).
func EnsureEnvFiles(project *models.Project) []string {
	var actions []string
	// Port hanya diselaraskan untuk proyek yang punya blok slot (bukan vibeasik sendiri / belum dialokasikan)
	syncPorts := project.PortSlot != nil && *project.PortSlot >= 0
	backendPort := singleBackendPort(project)
	seen := map[string]bool{}
	if project.RootPath != "" && project.RootPath != "-" {
		seen[project.RootPath] = true
		actions = append(actions, ensureEnvDir(project.RootPath, "", nil, 0)...)
	}
	for i := range project.Components {
		c := &project.Components[i]
		if c.AbsolutePath == "" || seen[c.AbsolutePath] {
			continue
		}
		seen[c.AbsolutePath] = true
		bePort := 0
		if c.Role != models.RoleBackend {
			bePort = backendPort
		}
		portComp := c
		if !syncPorts {
			portComp = nil
		}
		actions = append(actions, ensureEnvDir(c.AbsolutePath, c.RelativePath, portComp, bePort)...)
	}
	return actions
}

// singleBackendPort = run_port backend bila proyek punya tepat satu backend (lebih dari satu → ambigu, tidak disentuh).
func singleBackendPort(p *models.Project) int {
	port := 0
	for _, c := range p.Components {
		if c.Role == models.RoleBackend && c.RunPort > 0 {
			if port != 0 {
				return 0
			}
			port = c.RunPort
		}
	}
	return port
}

// envBootstrap memuat konteks pembuatan berkas env satu folder.
type envBootstrap struct {
	dir, rel string
	inGit    bool
	ignored  map[string]bool
	tracked  map[string]bool
}

func (b *envBootstrap) has(name string) bool { return util.FileExists(filepath.Join(b.dir, name)) }

func (b *envBootstrap) where(name string) string { return filepath.ToSlash(filepath.Join(b.rel, name)) }

func (b *envBootstrap) canCreate(name string) bool { return !b.inGit || b.ignored[name] }

func newEnvBootstrap(dir, rel string) *envBootstrap {
	b := &envBootstrap{dir: dir, rel: rel, inGit: isGitWorkTree(dir), tracked: map[string]bool{}}
	if b.inGit {
		b.ignored = gitIgnored(dir, []string{".env.local", ".env.server"})
		for _, f := range trackedEnvFiles(dir) {
			b.tracked[f] = true
		}
	}
	return b
}

// createEnvLocal membuat .env.local dari .env (atau .env.example); mengembalikan tindakan atau "".
func (b *envBootstrap) createEnvLocal() string {
	if b.has(".env.local") || !b.canCreate(".env.local") {
		return ""
	}
	src := ".env"
	if !b.has(src) {
		src = ".env.example"
	}
	raw, err := os.ReadFile(filepath.Join(b.dir, src))
	if err != nil {
		return ""
	}
	content := string(raw)
	if src != ".env" {
		content = fmt.Sprintf(envLocalHeader, src) + content
	}
	if os.WriteFile(filepath.Join(b.dir, ".env.local"), []byte(content), 0o600) != nil {
		return ""
	}
	return fmt.Sprintf("membuat %s dari %s", b.where(".env.local"), src)
}

// createEnvServer membuat .env.server dari .env.example (atau .env); mengembalikan tindakan atau "".
func (b *envBootstrap) createEnvServer() string {
	if b.has(".env.server") || !b.canCreate(".env.server") {
		return ""
	}
	src := ".env.example"
	if !b.has(src) {
		src = ".env"
	}
	raw, err := os.ReadFile(filepath.Join(b.dir, src))
	if err != nil {
		return ""
	}
	content := fmt.Sprintf(envServerHeader, src) + serverEnvFrom(string(raw), src == ".env")
	if os.WriteFile(filepath.Join(b.dir, ".env.server"), []byte(content), 0o600) != nil {
		return ""
	}
	return fmt.Sprintf("membuat %s dari %s (isi nilainya untuk server)", b.where(".env.server"), src)
}

// syncPorts menyelaraskan port di .env/.env.local yang tidak di-track git.
func (b *envBootstrap) syncPorts(comp *models.Component, bePort int) []string {
	var actions []string
	for _, name := range []string{".env", ".env.local"} {
		if b.has(name) && !b.tracked[name] && syncEnvPorts(filepath.Join(b.dir, name), *comp, bePort) {
			actions = append(actions, fmt.Sprintf("menyelaraskan port di %s dengan blok PORTS.md", b.where(name)))
		}
	}
	return actions
}

func ensureEnvDir(dir, rel string, comp *models.Component, bePort int) []string {
	usesEnv := false
	for _, name := range []string{".env", ".env.example", ".env.local"} {
		usesEnv = usesEnv || util.FileExists(filepath.Join(dir, name))
	}
	if !usesEnv {
		return nil // folder tidak memakai env
	}
	b := newEnvBootstrap(dir, rel)
	var actions []string
	if a := b.createEnvLocal(); a != "" {
		actions = append(actions, a)
	}
	if a := b.createEnvServer(); a != "" {
		actions = append(actions, a)
	}
	if comp != nil {
		actions = append(actions, b.syncPorts(comp, bePort)...)
	}
	return actions
}

// serverEnvFrom: kunci rahasia & URL localhost dikosongkan; dari .env (bukan template) semua nilai dikosongkan
// agar nilai dev tidak pernah ikut ke server.
func serverEnvFrom(content string, blankAll bool) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		m := envLineRe.FindStringSubmatch(line)
		if m == nil || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if blankAll || envSecretKey.MatchString(m[2]) || envLocalURLRe.MatchString(m[4]) {
			lines[i] = m[1] + m[2] + m[3]
		}
	}
	return strings.Join(lines, "\n")
}

// syncEnvPorts menyetel PORT/NUXT_PORT ke run_port komponen dan port URL API localhost FE/mobile ke port BE.
func syncEnvPorts(path string, comp models.Component, bePort int) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	lines := strings.Split(string(raw), "\n")
	changed := false
	for i, line := range lines {
		m := envLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, val := m[2], m[4]
		newVal := val
		switch {
		case (key == "PORT" || key == "NUXT_PORT") && comp.RunPort > 0:
			quote := ""
			if strings.HasPrefix(val, `"`) || strings.HasPrefix(val, "'") {
				quote = val[:1]
			}
			newVal = quote + strconv.Itoa(comp.RunPort) + quote
		case bePort > 0 && apiKeyHint.MatchString(key):
			newVal = envLocalURLRe.ReplaceAllString(val, "${1}:"+strconv.Itoa(bePort))
		}
		if newVal != val {
			lines[i] = m[1] + key + m[3] + newVal
			changed = true
		}
	}
	if !changed {
		return false
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600) == nil
}
