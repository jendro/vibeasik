package diagnostics

import "testing"

func TestInspectSource_WhenContentHasConcatenatedQuery_ReportsRule(t *testing.T) {
	src := "package h\n\nimport \"database/sql\"\n\nfunc L(db *sql.DB, n string) error {\n\t_, err := db.Query(\"SELECT id FROM t WHERE n = '\" + n + \"'\")\n\treturn err\n}\n"
	found := false
	for _, f := range InspectSource("internal/handlers/list_handler.go", src) {
		found = found || f.RuleID == "QUERY-SQLI"
	}
	if !found {
		t.Fatal("isi berkas dengan query dirangkai harus dilaporkan")
	}
}

func TestInspectSource_WhenFileIsNotAudited_ReturnsNothing(t *testing.T) {
	for _, p := range []string{"notes.txt", "internal/handlers/list_handler_test.go"} {
		if got := InspectSource(p, "db.Query(\"SELECT \" + x)"); len(got) != 0 {
			t.Fatalf("%s tidak boleh diperiksa: %+v", p, got)
		}
	}
}
