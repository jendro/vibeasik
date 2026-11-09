package diagnostics

import (
	"os"
	"testing"
)

func hasRuleID(fs []string, id string) bool {
	for _, f := range fs {
		if f == id {
			return true
		}
	}
	return false
}

func inspectRuleIDs(path, rel string) []string {
	var ids []string
	for _, f := range InspectFile(path, rel, "go") {
		ids = append(ids, f.RuleID)
	}
	return ids
}

func TestQuerySQLiNoSelfTrigger(t *testing.T) {
	for i, f := range []string{"code_inspector.go"} {
		_ = i
		if hasRuleID(inspectRuleIDs(f, "pkg/diagnostics/"+f), "QUERY-SQLI") {
			t.Errorf("%s memicu QUERY-SQLI pada dirinya sendiri", f)
		}
	}
}

func TestCommandInjectionNoSelfTrigger(t *testing.T) {
	for _, f := range []string{"code_inspector.go", "code_line_rules.go"} {
		if hasRuleID(inspectRuleIDs(f, "pkg/diagnostics/"+f), "SEC-COMMAND-INJECTION") {
			t.Errorf("%s memicu SEC-COMMAND-INJECTION pada dirinya sendiri", f)
		}
	}
}

func TestQuerySQLiStillDetected(t *testing.T) {
	p := t.TempDir() + "/repo.go"
	src := "package x\n\nfunc f(db *DB, id string) {\n\tdb.Raw(fmt.Sprintf(\"SELECT * FROM users WHERE id = %s\", id))\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasRuleID(inspectRuleIDs(p, "internal/repo.go"), "QUERY-SQLI") {
		t.Error("SQL concatenation harus tetap terdeteksi")
	}
}

func TestCommandInjection_MetadataAndDynamic(t *testing.T) {
	// Guard metadata check
	if !isCommandInjectionRuleMetadata("// CWE-78: exec.Command(\"sh \" + input)") {
		t.Error("expected comment/metadata to be recognized by guard")
	}
	if isCommandInjectionRuleMetadata("exec.Command(\"sh\", \"-c\", \"echo \" + input)") {
		t.Error("actual command invocation should not be recognized as rule metadata")
	}

	// Real dynamic command invocation must still be detected
	p := t.TempDir() + "/runner.go"
	src := "package runner\n\nimport \"os/exec\"\n\nfunc run(input string) {\n\texec.Command(\"sh\", \"-c\", \"echo \" + input).Run()\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasRuleID(inspectRuleIDs(p, "pkg/runner/runner.go"), "SEC-COMMAND-INJECTION") {
		t.Error("dynamic command execution harus terdeteksi SEC-COMMAND-INJECTION")
	}
}

func TestWildcardCORS_AllowlistAndSelfTrigger(t *testing.T) {
	// Self-trigger check: code_line_rules.go must not trigger ROUTER-WILDCARD-CORS
	if hasRuleID(inspectRuleIDs("code_line_rules.go", "pkg/diagnostics/code_line_rules.go"), "ROUTER-WILDCARD-CORS") {
		t.Error("code_line_rules.go tidak boleh memicu ROUTER-WILDCARD-CORS pada dirinya sendiri")
	}

	// Real server wildcard CORS must be detected
	p := t.TempDir() + "/server.go"
	src := "package api\n\nfunc setup(app *App) {\n\tapp.Use(cors.New(cors.Config{\n\t\tAllowOrigins: \"*\",\n\t}))\n}\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasRuleID(inspectRuleIDs(p, "internal/api/server.go"), "ROUTER-WILDCARD-CORS") {
		t.Error("wildcard CORS pada server harus terdeteksi ROUTER-WILDCARD-CORS")
	}
}
