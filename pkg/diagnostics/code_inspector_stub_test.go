package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestStubHandlerRegexPrecision(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{`// TODO: implement this`, true},
		{`// TODO: fix this bug`, true},
		{`// TODO: handle this error`, true},
		{`c.SendString("TODO")`, true},
		{`panic("not implemented")`, true},
		{`return c.Status(501).SendString("status: not implemented")`, true},
		{`return c.SendStatus(http.StatusNotImplemented)`, true},
		{`w.WriteHeader(http.StatusNotImplemented)`, true},
		{`throw new NotImplementedError();`, true},
		{`throw NotImplementedException(`, true},
		{`throw new UnsupportedOperationException("stub")`, true},
		{`TODO()`, true},
		{`throwMyNotImplementedError`, false},
		{`throw new NotImplementedErrorX`, false},
		{`// handle the request`, false},
		{`handleFoo()`, false},
		{`return ` + strings.Repeat("x", 100) + ` not implemented`, false},
		// Pemisahan token tidak boleh cocok dengan pola stub
		{"\t\t\t`|\\breturn\\b[^\\n]{0,80}\\bnot` + ` implemented\\b` +", false},
		{"\t\t\t`|\\bhttp\\.StatusNot` + `Implemented\\b` +", false},
		{"\t\t\t`|\\bthrow\\s+(?:new\\s+)?(?:Not` + `ImplementedError`", false},
	}
	for _, c := range cases {
		if got := stubHandlerRegex.MatchString(c.line); got != c.want {
			t.Errorf("%q: got %v want %v", c.line, got, c.want)
		}
	}
}

func isSelfTargetFile(target string) bool {
	return strings.Contains(target, "code_inspector.go") || strings.Contains(target, "code_line_rules.go")
}

func assertNoStubSelfMatch(t *testing.T, findings []models.Finding, contextName string) {
	t.Helper()
	for _, f := range findings {
		if f.RuleID != "ROUTER-STUB-HANDLER" {
			continue
		}
		if isSelfTargetFile(f.TargetFile) {
			t.Errorf("[%s] self-match ROUTER-STUB-HANDLER pada %s:L%d: %s", contextName, f.TargetFile, f.Line, f.Snippet)
		}
	}
}

func TestStubHandlerNoSelfTrigger(t *testing.T) {
	// 1. Verifikasi langsung InspectFile pada code_inspector.go dan code_line_rules.go
	files := []string{"code_inspector.go", "code_line_rules.go"}
	for _, f := range files {
		findings := InspectFile(f, "pkg/diagnostics/"+f, "go")
		assertNoStubSelfMatch(t, findings, "InspectFile:"+f)
	}

	// 2. Verifikasi melalui InspectComponentCode
	absPath, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("gagal membaca path absolut: %v", err)
	}
	comp := &models.Component{
		AbsolutePath: absPath,
		RelativePath: "pkg/diagnostics",
	}
	assertNoStubSelfMatch(t, InspectComponentCode(comp), "InspectComponentCode")

	// 3. Verifikasi melalui InspectProjectCode
	project := &models.Project{
		RootPath: filepath.Dir(filepath.Dir(absPath)),
		Components: []models.Component{
			{
				AbsolutePath: absPath,
				RelativePath: "pkg/diagnostics",
			},
		},
	}
	assertNoStubSelfMatch(t, InspectProjectCode(project), "InspectProjectCode")
}

func TestStubHandlerStillDetected(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		filename string
		lang     string
		content  string
		desc     string
	}{
		{
			filename: "handler_http_status.go",
			lang:     "go",
			content:  "package api\n\nimport \"net/http\"\n\nfunc HandleUser(w http.ResponseWriter, r *http.Request) {\n\tw.WriteHeader(http.StatusNotImplemented)\n}\n",
			desc:     "http.StatusNotImplemented pada Go handler",
		},
		{
			filename: "handler_panic.go",
			lang:     "go",
			content:  "package api\n\nfunc HandleOrder() {\n\tpanic(\"not implemented\")\n}\n",
			desc:     "panic(\"not implemented\") pada Go handler",
		},
		{
			filename: "handler_return_stub.go",
			lang:     "go",
			content:  "package api\n\nfunc HandlePayment() string {\n\treturn \"not implemented\"\n}\n",
			desc:     "return not implemented pada Go handler",
		},
		{
			filename: "controller_stub.ts",
			lang:     "typescript",
			content:  "export class UserController {\n  async getUser() {\n    throw new NotImplementedError();\n  }\n}\n",
			desc:     "throw new NotImplementedError() pada TypeScript handler",
		},
		{
			filename: "view_stub.py",
			lang:     "python",
			content:  "def get_user_profile(request):\n    raise NotImplementedError\n",
			desc:     "raise NotImplementedError pada Python view",
		},
	}

	for _, tc := range testCases {
		filePath := filepath.Join(tempDir, tc.filename)
		if err := os.WriteFile(filePath, []byte(tc.content), 0o644); err != nil {
			t.Fatalf("gagal menulis berkas uji %s: %v", tc.filename, err)
		}

		findings := InspectFile(filePath, tc.filename, tc.lang)
		detected := false
		for _, f := range findings {
			if f.RuleID == "ROUTER-STUB-HANDLER" {
				detected = true
				break
			}
		}

		if !detected {
			t.Errorf("Deteksi stub gagal untuk skenario nyata '%s' (%s)", tc.desc, tc.filename)
		}
	}
}
