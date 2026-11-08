package diagnostics

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestServerRulesSkipFrontendCode(t *testing.T) {
	composable := `export const useProjects = () => {
  const remove = (id: number) => api.post('/projects/delete-item', { id })
  const openDoc = () => window.open('/docs/' + params.id)
  return { remove, openDoc }
}
`
	rel := "app/composables/useProjects.ts"
	got := ruleSet(InspectFile(writeFixture(t, t.TempDir(), rel, composable), rel, "typescript"))
	for _, id := range []string{"ROUTER-UNGUARDED-MUTATION", "ROUTER-VERB-IN-PATH", "SEC-PATH-TRAVERSAL"} {
		if _, ok := got[id]; ok {
			t.Errorf("%s must not run on frontend composables", id)
		}
	}

	server := `router.post('/orders', createOrder)
const data = fs.readFileSync('./uploads/' + req.params.name)
`
	relSrv := "server/routes/orders.ts"
	gotSrv := ruleSet(InspectFile(writeFixture(t, t.TempDir(), relSrv, server), relSrv, "typescript"))
	if _, ok := gotSrv["ROUTER-UNGUARDED-MUTATION"]; !ok {
		t.Errorf("expected ROUTER-UNGUARDED-MUTATION on server route")
	}
}

func TestNativeDialogOnlyOnUIGlobalCalls(t *testing.T) {
	cli := `const answer = await inquirer.prompt([{ name: 'x' }])
`
	relCli := "scripts/setup.js"
	if _, ok := ruleSet(InspectFile(writeFixture(t, t.TempDir(), relCli, cli), relCli, "javascript"))["DESIGN-NATIVE-DIALOG"]; ok {
		t.Errorf("CLI inquirer.prompt must not be flagged")
	}

	vue := `<script setup lang="ts">
const del = () => { if (confirm('Hapus?')) remove() }
const ok = dialog.confirm('x')
</script>
`
	relVue := "app/pages/items.vue"
	f, ok := ruleSet(InspectFile(writeFixture(t, t.TempDir(), relVue, vue), relVue, "vue"))["DESIGN-NATIVE-DIALOG"]
	if !ok || f.Line != 2 {
		t.Errorf("expected global confirm() on line 2 to be flagged, got %+v", f)
	}
}

func TestAndroidRulesSkipJVMServer(t *testing.T) {
	spring := `package com.acme.api;

import org.springframework.web.bind.annotation.*;

public class UserActivityController {
    static Context ctx;
    public void refresh() { adapter.notifyDataSetChanged(); }
}
`
	rel := "src/main/java/com/acme/api/UserActivityController.java"
	for id := range ruleSet(InspectFile(writeFixture(t, t.TempDir(), rel, spring), rel, "java")) {
		if strings.HasPrefix(id, "AND-") {
			t.Errorf("Android rule %s must not run on Spring server code", id)
		}
	}
}

func TestPythonStubOnlyForPython(t *testing.T) {
	py := `def handler(request):
    pass
`
	if _, ok := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "api/views.py", py), "api/views.py", "python"))["ROUTER-STUB-HANDLER"]; !ok {
		t.Errorf("expected python pass stub to be flagged")
	}
	goCode := "package main\n\nfunc main() {\n\tvar pass int\n\t_ = pass\n}\n"
	if _, ok := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "main.go", goCode), "main.go", "go"))["ROUTER-STUB-HANDLER"]; ok {
		t.Errorf("go identifier 'pass' must not be flagged as stub")
	}
}

func TestComplexityPerLanguage(t *testing.T) {
	branches := strings.Repeat("  if (x > 1 && y) { y++ }\n", 10)
	cases := map[string]struct{ rel, code string }{
		"typescript": {"src/util.ts", "export function decide(x: number) {\n" + branches + "}\n"},
		"kotlin":     {"src/Util.kt", "fun decide(x: Int) {\n" + branches + "}\n"},
		"python":     {"util.py", "def decide(x):\n" + strings.Repeat("    if x > 1 and y:\n        y += 1\n", 10)},
		"php":        {"Util.php", "<?php\nfunction decide($x) {\n" + branches + "}\n"},
		"dart":       {"lib/util.dart", "int decide(int x) {\n" + branches + "}\n"},
	}
	for lang, c := range cases {
		dir := t.TempDir()
		writeFixture(t, dir, c.rel, c.code)
		findings := scanComplexityForTest(dir)
		found := false
		for _, f := range findings {
			if f.RuleID == "DEEP-FUNC-HIGH-COMPLEXITY" && strings.Contains(f.Title, "decide") {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: expected DEEP-FUNC-HIGH-COMPLEXITY for decide()", lang)
		}
	}
}

func TestFrameworkConfigIsWebSource(t *testing.T) {
	for _, rel := range []string{"web/nuxt.config.ts", "vite.config.js", "app/next.config.mjs"} {
		lang := "typescript"
		if strings.HasSuffix(rel, "js") {
			lang = "javascript"
		}
		if !isUISourceFile(rel, lang) {
			t.Errorf("%s must be treated as frontend web source", rel)
		}
	}
	if isUISourceFile("api/server.ts", "typescript") {
		t.Errorf("plain server file must not be web source")
	}
}

func complexityFindings(t *testing.T, files map[string]string) []models.Finding {
	t.Helper()
	dir := t.TempDir()
	for rel, code := range files {
		writeFixture(t, dir, rel, code)
	}
	findings := scanComplexityForTest(dir)
	var out []models.Finding
	for _, f := range findings {
		if f.RuleID == "DEEP-FUNC-HIGH-COMPLEXITY" {
			out = append(out, f)
		}
	}
	return out
}

func TestComplexityIgnoresStringsAndComments(t *testing.T) {
	noisy := strings.Repeat("  const s = 'a && b || if (c) case'\n  // if (x && y) for while\n", 20)
	ts := "export const pick = (m: string) => {\n" + noisy + "  if (m) return 1\n  return 0\n}\n"
	if got := complexityFindings(t, map[string]string{"src/pick.ts": ts}); len(got) != 0 {
		t.Errorf("strings and comments must not count as branches, got %v", got[0].Title)
	}
}

func TestComplexitySkipsFrontendUI(t *testing.T) {
	// Kode UI berjalan di browser satu pengguna: cukup diaudit ukuran komponennya, bukan CC
	fn := "export function decide(x: number) {\n" + strings.Repeat("  if (x > 1 && y) { y++ }\n", 10) + "}\n"
	vue := "<template><div /></template>\n<script setup lang=\"ts\">\n" + fn + "</script>\n"
	got := complexityFindings(t, map[string]string{
		"app/pages/x.vue":          vue,
		"src/components/Panel.tsx": fn,
		"app/composables/useX.ts":  fn,
		"server/api/handler.ts":    fn,
	})
	if len(got) != 1 || got[0].TargetFile != filepath.FromSlash("server/api/handler.ts") {
		t.Errorf("expected only the server file to be flagged, got %+v", got)
	}
}

func TestComplexityStopsAtFunctionEnd(t *testing.T) {
	// Fungsi kecil diikuti method kelas yang tidak dikenali sebagai fungsi baru oleh pola lama:
	// cabang setelah '}' penutup tidak boleh ditimpakan ke fungsi sebelumnya.
	ts := "export function small(x: number) {\n  return x\n}\n\nexport default {\n  computed: {\n" + strings.Repeat("    a: () => x && y || z,\n", 12) + "  }\n}\n"
	if got := complexityFindings(t, map[string]string{"src/a.ts": ts}); len(got) != 0 {
		t.Errorf("branches after the function body must not be attributed to it, got %v", got[0].Title)
	}

	cls := "class A {\n  load() {\n" + strings.Repeat("    if (a && b) { c++ }\n", 9) + "  }\n  tiny() {\n    return 1\n  }\n}\n"
	got := complexityFindings(t, map[string]string{"src/b.ts": cls})
	if len(got) != 1 || !strings.Contains(got[0].Title, "'load'") || !strings.Contains(got[0].Title, "CC=19") {
		t.Errorf("expected only load() with CC=19, got %+v", got)
	}
}

func TestComplexityUniquePerFile(t *testing.T) {
	body := "Widget build(BuildContext context) {\n" + strings.Repeat("  if (a && b) x();\n", 9) + "  return Container();\n}\n"
	got := models.DeduplicateFindings(complexityFindings(t, map[string]string{"lib/a.dart": body, "lib/b.dart": body}))
	if len(got) != 2 {
		t.Errorf("same function name & line in different files must stay separate findings, got %d", len(got))
	}
}

func TestComplexityPythonIndentEnd(t *testing.T) {
	py := "def small(x):\n    return x\n\n" + strings.Repeat("if a and b:\n    pass\n", 12) + "\ndef big(x):\n" + strings.Repeat("    if x and y:\n        pass\n", 9)
	got := complexityFindings(t, map[string]string{"tool.py": py})
	if len(got) != 1 || !strings.Contains(got[0].Title, "'big'") {
		t.Errorf("expected only big(), got %+v", got)
	}
}

func TestAuditCriteriaNoFalsePositivePathTraversal(t *testing.T) {
	fullPath := filepath.Join("..", "..", "studio", "app", "utils", "auditCriteria.ts")
	findings := InspectFile(fullPath, "studio/app/utils/auditCriteria.ts", "typescript")
	for _, f := range findings {
		if f.RuleID == "SEC-PATH-TRAVERSAL" {
			t.Errorf("found false positive SEC-PATH-TRAVERSAL in auditCriteria.ts at line %d: %s", f.Line, f.Snippet)
		}
	}
}

func TestAuditCriteriaNoFalsePositiveSwallowedError(t *testing.T) {
	fullPath := filepath.Join("..", "..", "studio", "app", "utils", "auditCriteria.ts")
	findings := InspectFile(fullPath, "studio/app/utils/auditCriteria.ts", "typescript")
	for _, f := range findings {
		if f.RuleID == "SMELL-SWALLOWED-ERROR" {
			t.Errorf("found false positive SMELL-SWALLOWED-ERROR in auditCriteria.ts at line %d: %s", f.Line, f.Snippet)
		}
	}
}

