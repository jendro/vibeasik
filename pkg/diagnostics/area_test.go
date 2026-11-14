package diagnostics

import (
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestFindingAreaByRuleBeforeCategory(t *testing.T) {
	cases := []struct {
		rule, cat, want string
	}{
		{"QUERY-NPLUS1-CHAIN_x.go_10", "query", AreaSpeed},
		{"QUERY-SQLI_x.go_3", "query", AreaSecurity},
		{"QUERY-MISSING-TX-ROLLBACK", "query", AreaData},
		{"DEEP-FORM-MISSING-VALIDATION_POST_/api/x", "router", AreaData},
		{"ROUTER-UNGUARDED-MUTATION_a", "router", AreaSecurity},
		{"ROUTER-VERB-IN-PATH_a", "router", AreaCode},
		{"FE-IMG-NO-LAZY_a.vue_3", "frontend", AreaSpeed},
		{"FE-SMALL-FONT_a.vue_3", "frontend", AreaUI},
		{"RUNTIME-PAGE-BLANK", "runtime", AreaUI},
		{"RUNTIME-ROUTE-5XX", "runtime", AreaSpeed},
		{"RUNTIME-BOOT-TIMEOUT", "runtime", AreaOps},
		{"ENV-FILE-TRACKED", "security", AreaSecurity},
		{"ENV-EXAMPLE-DRIFT", "config", AreaOps},
		{"COST-POLYNOMIAL-GROWTH_x", "code_smell", AreaSpeed},
		{"DEEP-FUNC-HIGH-COMPLEXITY_x", "code_smell", AreaCode},
		{"", "risk", AreaOps},
		{"", "security", AreaSecurity},
		{"", "tak-dikenal", AreaCode},
	}
	for _, c := range cases {
		if got := FindingArea(models.Finding{RuleID: c.rule, Category: c.cat}); got != c.want {
			t.Errorf("%s/%s: %s, mau %s", c.rule, c.cat, got, c.want)
		}
	}
}
