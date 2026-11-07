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
