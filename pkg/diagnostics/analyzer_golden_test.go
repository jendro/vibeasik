package diagnostics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/util"
)

// normTmp mengubah v menjadi JSON dengan path folder sementara diganti <TMP> agar golden stabil.
func normTmp(t *testing.T, v any, dirs ...string) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, d := range dirs {
		s = strings.ReplaceAll(s, strings.ReplaceAll(d, `/`, `\/`), "<TMP>")
		s = strings.ReplaceAll(s, d, "<TMP>")
	}
	return json.RawMessage(s)
}

func goldenProject(t *testing.T) (*models.Project, *models.Project, string, string) {
	root := t.TempDir()
	other := t.TempDir()
	writeFiles(t, root, map[string]string{
		"dev.sh":                          "#!/bin/sh\ncd backend\n",
		"backend/main.go":                 "package main\nfunc main() {}\n",
		"backend/main_test.go":            "package main\n",
		"backend/.env.local":              "PORT=4101\n",
		"backend/.env.example":            "PORT=\n",
		"web/nuxt.config.ts":              "export default {}\n",
		"web/.env.local":                  "NUXT_PUBLIC_API_BASE=http://localhost:9999\nOTHER_URL=http://localhost:1\n",
		"web/composables/useA.ts":         "export const useThing = () => 1\nexport function ref() {}\n",
		"web/utils/useThing.ts":           "export const useThing = () => 2\n",
		"web/utils/other.ts":              "export default function () {}\n",
		"web/pages/index.vue":             "<template><div/></template>\n",
		"mobile/pubspec.yaml":             "name: m\ndependencies:\n  flutter:\n    sdk: flutter\n",
		"mobile/.env":                     "API_BASE_URL=http://127.0.0.1:4101/api\n",
		"mobile/lib/main.dart":            "// http://localhost:1\nconst u = 'http://localhost:4101';\nconst v = 'http://10.0.0.1';\n",
		"mobile/lib/sub/other.dart":       "var a = \"http://0.0.0.0:80\";\n",
		"mobile/test/widget_test.dart":    "void main() {}\n",
		"backend/handler/user_handler.go": "package h\nimport \"net/http\"\nfunc F() { http.Get(\"x\") }\n",
		"README_IGNORED_NOT_ROOT/.keep":   "",
		".hidden/skip.txt":                "",
		"node_modules/x/y.js":             "",
	})
	git(t, root, "init", "-q")
	p := &models.Project{
		ID: 1, Name: "Proj One", ProjectType: "bundle", RootPath: root,
		Components: []models.Component{
			{ID: 1, Name: "API", Role: models.RoleBackend, RelativePath: "backend", AbsolutePath: filepath.Join(root, "backend"), Language: "go", Port: 4101},
			{ID: 2, Name: "Web", Role: models.RoleFrontendClient, RelativePath: "web", AbsolutePath: filepath.Join(root, "web"), Language: "typescript", Framework: "nuxt", EnvSummary: "PORT only", RunPort: 4102, Port: 3000},
			{ID: 3, Name: "App", Role: models.RoleFrontendMobile, RelativePath: "mobile", AbsolutePath: filepath.Join(root, "mobile"), Language: "dart", Framework: "flutter", Port: 4102},
			{ID: 4, Name: "Gone", Role: models.RoleFrontendAdmin, RelativePath: "gone", AbsolutePath: filepath.Join(root, "gone"), Port: 4101},
		},
	}
	o := &models.Project{
		ID: 2, Name: "Proj Two", RootPath: "-",
		Components: []models.Component{{ID: 9, Name: "Other", Role: models.RoleBackend, AbsolutePath: other, Port: 4101}},
	}
	return p, o, root, other
}

func TestAnalyzerGolden(t *testing.T) {
	p, o, root, other := goldenProject(t)
	all := []models.Project{*p, *o}
	out := map[string]any{}

	diag, err := AnalyzeProject(p, all)
	if err != nil {
		t.Fatal(err)
	}
	out["project"] = normTmp(t, diag, root, other)
	diag2, _ := AnalyzeProject(o, all)
	out["project_other"] = normTmp(t, diag2, root, other)

	comps := map[string]any{}
	for i := range p.Components {
		c := &p.Components[i]
		d, err := AnalyzeComponent(c, p, all)
		if err != nil {
			t.Fatal(err)
		}
		comps[c.Name] = normTmp(t, d, root, other)
	}
	noPath := &models.Component{ID: 50, Name: "NoPath", Role: models.RoleFrontendClient, EnvSummary: "x"}
	d, _ := AnalyzeComponent(noPath, p, all)
	comps["nopath"] = normTmp(t, d, root, other)
	out["components"] = comps

	pen := map[string]int{}
	for _, f := range []models.Finding{
		{Category: "port", Title: "Internal Port Collision: 1"}, {Category: "port", Title: "Cross-Project x"},
		{Category: "port", Title: "Shared Dev Port"}, {Title: "Missing Application Absolute Path"},
		{Title: "Directory Missing on Disk: x"}, {Title: "Component Directory Missing: x"},
		{Title: ".env Not Excluded"}, {RuleID: "ENV-FILE-TRACKED"}, {RuleID: "ENV-NOT-IGNORED"},
		{Title: "Missing .gitignore at Project Root"}, {Title: "Missing .env.example"}, {RuleID: "ENV-EXAMPLE-MISSING"},
		{Title: "Missing README.md"}, {Title: "No Automated Tests Detected"}, {Category: "git", Severity: "CRITICAL"},
		{Severity: "critical"}, {Severity: "WARNING"}, {Severity: "INFO"}, {Severity: "weird"}, {},
	} {
		pen[f.Category+"|"+f.RuleID+"|"+f.Title+"|"+f.Severity] = CalculateFindingPenalty(f)
	}
	out["penalty"] = pen
	out["score"] = []int{
		CalculateHealthScore(nil),
		CalculateHealthScore([]models.Finding{{Severity: "CRITICAL"}, {Severity: "WARNING", Status: models.FindingStatusResolved}, {Severity: "INFO", Status: models.FindingStatusIgnored}}),
		CalculateHealthScore(make([]models.Finding, 20)),
	}
	for i := 0; i < 20; i++ {
		_ = i
	}
	out["tree"] = strings.ReplaceAll(util.GenerateFolderTree(root, 2), filepath.Base(root), "<ROOT>")
	out["tree_depth1"] = strings.ReplaceAll(util.GenerateFolderTree(root, 1), filepath.Base(root), "<ROOT>")
	out["has_tests"] = map[string]bool{
		"backend": hasTests(filepath.Join(root, "backend"), "go"),
		"web":     hasTests(filepath.Join(root, "web"), "typescript"),
		"mobile":  hasTests(filepath.Join(root, "mobile"), "dart"),
		"none":    hasTests(filepath.Join(root, "nope"), "go"),
	}

	// dev_tooling
	dt := map[string]any{}
	dt["link_all"] = normTmp(t, inspectFrontendBackendLink(p, 0), root)
	dt["link_web"] = normTmp(t, inspectFrontendBackendLink(p, 2), root)
	dt["link_app"] = normTmp(t, inspectFrontendBackendLink(p, 3), root)
	dt["nuxt_dup"] = normTmp(t, inspectNuxtDuplicateAutoImports(p.Components[1]), root)
	dt["nuxt_non"] = normTmp(t, inspectNuxtDuplicateAutoImports(p.Components[0]), root)
	dt["mobile_src"] = normTmp(t, mobileSourceLocalhost(p.Components[2]), root)
	dt["tooling_dev_present"] = normTmp(t, inspectProjectTooling(p), root)
	_ = os.Remove(filepath.Join(root, "dev.sh"))
	dt["tooling_nodev"] = normTmp(t, inspectProjectTooling(p), root)
	writeFiles(t, root, map[string]string{"deploy.sh": "#!/bin/sh\n", "dev-mobile.sh": "echo hi\n"})
	dt["tooling_deploy_bad"] = normTmp(t, inspectProjectTooling(p), root)
	writeFiles(t, root, map[string]string{"deploy.sh": "trap x EXIT\ncp .env.local .env\ncp .env.server .env\n"})
	_ = os.Chmod(filepath.Join(root, "dev-mobile.sh"), 0o755)
	dt["tooling_deploy_ok"] = normTmp(t, inspectProjectTooling(p), root)
	dt["tooling_noroot"] = inspectProjectTooling(o)
	dt["cross"] = normTmp(t, crossProjectPortFindings(p, all), root)
	out["dev_tooling"] = dt
	assertGolden(t, "analyzer", out)
}
