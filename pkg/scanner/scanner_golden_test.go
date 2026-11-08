package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, c := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

var scanFixtures = map[string]map[string]string{
	"mono": {
		"go-api/go.mod":                             "module x\nrequire github.com/gin-gonic/gin v1\n",
		"go-api/.env":                               "PORT=9090\nSECRET_A=1\n# c\n\nB=2\n",
		"admin-panel/package.json":                  `{"name":"admin_panel","dependencies":{"nuxt":"3"},"scripts":{"dev":"x","build":"y"}}`,
		"admin-panel/tsconfig.json":                 "{}",
		"shop-web/package.json":                     `{"dependencies":{"react":"18"},"scripts":{"start":"x"}}`,
		"shop-web/pnpm-lock.yaml":                   "",
		"api-node/package.json":                     `{"devDependencies":{"express":"4"}}`,
		"api-node/yarn.lock":                        "",
		"api-node/.env.example":                     "PORT = 7000\nK=v\n",
		"nest-svc/package.json":                     `{"dependencies":{"@nestjs/core":"1"}}`,
		"nest-svc/bun.lockb":                        "",
		"misc/package.json":                         `{"dependencies":{"vue":"3","left":"1"}}`,
		"nx/package.json":                           `{"dependencies":{"next":"14"}}`,
		"badjson/package.json":                      `{not json`,
		"apps/mobile/pubspec.yaml":                  "name: m",
		"apps/cms/composer.json":                    `{"require":{"codeigniter4/framework":"4"},"scripts":{"serve":"php spark serve --port 8123"}}`,
		"apps/cms/spark":                            "",
		"apps/legacy/application/config/routes.php": "<?php",
		"apps/lara/artisan":                         "",
		"apps/plain/composer.json":                  `{}`,
		"apps/routes-only/routes/web.php":           "<?php",
		"apps/ci4b/composer.json":                   `{"scripts":{"serve":"php spark serve"}}`,
		"apps/ci4b/app/Config/Routes.php":           "<?php",
		"apps/ci4b/.env":                            "PORT=8555\n",
		"py-svc/requirements.txt":                   "Flask==2\n",
		"py2/pyproject.toml":                        "x",
		"py3/requirements.txt":                      "Django\n",
		"py4/requirements.txt":                      "fastapi\n",
		"rs/Cargo.toml":                             "x",
		"node_modules/x/package.json":               `{}`,
		"empty/readme.md":                           "x",
	},
	"single":   {"go.mod": "module x\n", "main.go": "package main\n", "sub/readme": "x"},
	"fe2":      {"alpha/package.json": `{"dependencies":{"react":"18"}}`, "beta/package.json": `{"dependencies":{"vue":"3"}}`, "srv/go.mod": "module s\nrequire github.com/labstack/echo v1\n"},
	"fe2b":     {"one/package.json": `{"name":"backoffice","dependencies":{"react":"18"}}`, "two/package.json": `{"dependencies":{"vue":"3"}}`, "chi/go.mod": "require github.com/go-chi/chi v1\n", "fib/go.mod": "require github.com/gofiber/fiber v2\n"},
	"fe2c":     {"web-a/package.json": `{"dependencies":{"react":"18"}}`, "web-b/package.json": `{"dependencies":{"vue":"3"}}`},
	"fe1":      {"web/package.json": `{"dependencies":{"react":"18"}}`, "api/package.json": `{"dependencies":{"express":"4"}}`},
	"rootnode": {"package.json": `{"name":"root-app","dependencies":{"next":"1"}}`, "inner/package.json": `{"dependencies":{"vue":"3"}}`, "inner/deep/package.json": `{}`},
}

func TestScanDirectoryGolden(t *testing.T) {
	names := make([]string, 0)
	for n := range scanFixtures {
		names = append(names, n)
	}
	sort.Strings(names)
	out := map[string]any{}
	for _, n := range names {
		root := filepath.Join(t.TempDir(), "my-proj_"+n)
		writeFiles(t, root, scanFixtures[n])
		res, err := ScanDirectory(root)
		if err != nil {
			t.Fatal(err)
		}
		res.GitBranch, res.GitRemote = "", ""
		sort.Strings(res.DetectedStack)
		b, _ := json.MarshalIndent(res, "", " ")
		out[n] = json.RawMessage(strings.ReplaceAll(string(b), root, "<ROOT>"))
	}
	if _, err := ScanDirectory(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("harus error")
	}
	got, _ := json.MarshalIndent(out, "", " ")
	path := filepath.Join("testdata", "scan.golden.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(got) {
		t.Error("output ScanDirectory berubah")
	}
}
