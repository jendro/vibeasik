package diagnostics

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Test karakterisasi env: inspectEnvHygiene (berbagai skenario git), ensureEnvDir, syncEnvPorts, serverEnvFrom.
func TestEnvGolden(t *testing.T) {
	out := map[string]any{}

	// 1. repo git: tracked + tidak ter-ignore + template ter-ignore + drift
	d1 := t.TempDir()
	git(t, d1, "init", "-q")
	writeFiles(t, d1, map[string]string{
		".gitignore":      ".env\n",
		".env.server.jkt": "DB_URL=postgres://jkt\n",
	})
	git(t, d1, "add", ".")
	git(t, d1, "commit", "-qm", "init")
	writeFiles(t, d1, map[string]string{
		".env.local":   "DB_URL=postgres://local\nexport JWT_SECRET=rahasia\nA.B=1\n",
		".env.example": "DB_URL=\n",
	})
	out["leaks"] = inspectEnvHygiene(d1, "be")

	// 2. template ter-ignore, root
	d2 := t.TempDir()
	git(t, d2, "init", "-q")
	writeFiles(t, d2, map[string]string{".gitignore": ".env*\n", ".env.example": "A=\n", ".env": "A=1\n"})
	out["template_ignored"] = inspectEnvHygiene(d2, "")

	// 3. di luar git, tanpa template
	d3 := t.TempDir()
	writeFiles(t, d3, map[string]string{".env.server": "A=1\n", ".env.local": "B=2\n"})
	out["no_template"] = inspectEnvHygiene(d3, ".")

	// 4. bersih
	d4 := t.TempDir()
	git(t, d4, "init", "-q")
	writeFiles(t, d4, map[string]string{
		".gitignore": envGitignoreBlock + "\n", ".env.example": "PORT=\nDB_URL=\n",
		".env.local": "PORT=8080\nDB_URL=x\n", ".env.server.a": "PORT=80\n",
	})
	git(t, d4, "add", ".")
	git(t, d4, "commit", "-qm", "init")
	out["clean"] = inspectEnvHygiene(d4, "")

	// 5. folder tanpa env / tidak ada
	out["empty"] = inspectEnvHygiene(t.TempDir(), "x")
	out["missing"] = inspectEnvHygiene(filepath.Join(d1, "nope"), "x")

	// 6. tracked tanpa ignore tambahan, template .env.dist
	d6 := t.TempDir()
	git(t, d6, "init", "-q")
	writeFiles(t, d6, map[string]string{".env": "K=1\n", ".env.dist": "K=\n"})
	git(t, d6, "add", ".")
	git(t, d6, "commit", "-qm", "init")
	out["tracked_dist"] = inspectEnvHygiene(d6, "api")

	// ensureEnvDir
	envActions := map[string]any{}
	comp := &models.Component{RunPort: 4101}
	read := func(dir string) map[string]string {
		m := map[string]string{}
		for _, n := range []string{".env", ".env.local", ".env.server"} {
			if b, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
				m[n] = string(b)
			}
		}
		return m
	}
	e1 := t.TempDir() // non-git
	writeFiles(t, e1, map[string]string{".env": "PORT=3000\nAPI_URL=http://localhost:3000/api\nJWT_SECRET=abc\n# c=1\nNAME=x\n", ".env.example": "PORT=\nSECRET_KEY=zz\nURL=http://127.0.0.1:80\nNAME=app\n"})
	envActions["nongit_actions"] = ensureEnvDir(e1, "web", comp, 4102)
	envActions["nongit_files"] = read(e1)
	e2 := t.TempDir() // git, ignore menutup .env.local & .env.server
	git(t, e2, "init", "-q")
	writeFiles(t, e2, map[string]string{".gitignore": ".env\n.env.*\n!.env.example\n", ".env.example": "PORT=1\nTOKEN=a\n"})
	envActions["git_actions"] = ensureEnvDir(e2, "", nil, 0)
	envActions["git_files"] = read(e2)
	e3 := t.TempDir() // git tanpa ignore
	git(t, e3, "init", "-q")
	writeFiles(t, e3, map[string]string{".env.example": "PORT=1\n"})
	envActions["git_noignore"] = ensureEnvDir(e3, "", nil, 0)
	e4 := t.TempDir()
	envActions["no_env"] = ensureEnvDir(e4, "", nil, 0)
	e5 := t.TempDir() // .env saja, tracked
	git(t, e5, "init", "-q")
	writeFiles(t, e5, map[string]string{".env": "PORT=9\n", ".gitignore": ".env.local\n.env.server\n"})
	git(t, e5, "add", "-f", ".env")
	envActions["tracked_actions"] = ensureEnvDir(e5, "svc", comp, 0)
	envActions["tracked_files"] = read(e5)
	out["ensure"] = envActions

	// syncEnvPorts
	sync := map[string]any{}
	p := filepath.Join(t.TempDir(), "e")
	for name, c := range map[string]struct {
		body string
		port int
		be   int
	}{
		"port_quote":  {"PORT=\"1\"\nNUXT_PORT='2'\nOTHER=3\n", 4101, 0},
		"api":         {"API_URL=http://localhost:1/x\nBACKEND_API=https://127.0.0.1\nNOPE=http://localhost\n", 0, 4102},
		"nochange":    {"PORT=4101\n", 4101, 0},
		"no_port":     {"PORT=1\n", 0, 0},
		"comment_key": {"# PORT=1\n  export PORT = 7\n", 4101, 0},
	} {
		_ = os.WriteFile(p, []byte(c.body), 0o600)
		changed := syncEnvPorts(p, models.Component{RunPort: c.port}, c.be)
		b, _ := os.ReadFile(p)
		sync[name] = []any{changed, string(b)}
	}
	sync["missing"] = syncEnvPorts(filepath.Join(t.TempDir(), "none"), models.Component{RunPort: 1}, 1)
	out["sync"] = sync

	srv := map[string]string{}
	in := "A=1\n# TOKEN=x\nexport DB_PASS=pw\nURL=http://localhost:3000\nNAME=app\nbad line\n"
	srv["blank_all"] = serverEnvFrom(in, true)
	srv["secrets"] = serverEnvFrom(in, false)
	out["server_env"] = srv

	assertGolden(t, "env", out)
}
