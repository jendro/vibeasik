package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func grouperComponents(rels ...string) []models.ComponentProposal {
	roles := []string{"backend", "frontend_client", "mobile", "frontend_admin", "other"}
	var cs []models.ComponentProposal
	for i, r := range rels {
		cs = append(cs, models.ComponentProposal{
			Name: "C" + r, Role: roles[i%len(roles)], RelativePath: r,
			AbsolutePath: "/w/acme/" + r, Language: "go", Framework: "fiber", Port: 8000 + i%2,
		})
	}
	return cs
}

func normalizeGrouping(r *models.ScanGroupingResult) string {
	for i := range r.Groups {
		r.Groups[i].ID = ""
	}
	sort.Slice(r.Groups, func(i, j int) bool { return r.Groups[i].Name < r.Groups[j].Name })
	b, _ := json.Marshal(r)
	return string(b)
}

func TestScanGrouperGolden(t *testing.T) {
	old := LLMQuery
	defer func() { LLMQuery = old }()

	existing := []models.Project{
		{ID: 7, Name: "Acme", Slug: "acme", RootPath: "/w/acme/apps", ProjectType: models.TypeBundle, Description: "d",
			Components: []models.Component{{Name: "x", AbsolutePath: "/w/acme/lone"}}},
		{ID: 9, Name: "Dash", RootPath: "-", Components: []models.Component{{AbsolutePath: ""}}},
		{ID: 10, Name: "NoRoot", RootPath: ""},
	}
	comps := map[string][]models.ComponentProposal{
		"none":       nil,
		"one":        grouperComponents("svc"),
		"two":        grouperComponents("be", "fe"),
		"three-flat": grouperComponents("a", "b", "c"),
		"four-flat":  grouperComponents("a", "b", "c", "d"),
		"nested":     grouperComponents("apps/web", "apps/api", "tools/cli", "tools/gen", "x"),
		"nested1":    grouperComponents("apps/web", "apps/api", "apps/m", "apps/z"),
		"dot":        grouperComponents(".", "./a", "b/c", "b/d"),
		"mixed":      grouperComponents("apps/web", "lone", "other/x", "other/y", "root1"),
		"fe2":        append(grouperComponents("a"), models.ComponentProposal{Name: "f1", Role: "frontend_client", RelativePath: "f1"}, models.ComponentProposal{Name: "f2", Role: "frontend_admin", RelativePath: "f2"}),
	}
	// satu komponen frontend & satu mobile
	comps["fe-mobile"] = []models.ComponentProposal{
		{Name: "m", Role: "mobile", RelativePath: "m"}, {Name: "f", Role: "frontend_client", RelativePath: "f"},
	}
	llm := map[string]string{
		"nil":    "",
		"spaces": "   ",
		"bad":    "not json",
		"empty":  `{"summary":"s","groups":[]}`,
		"ok": `{"summary":"ringkas","groups":[{"id":"g1","name":"Proj A","project_type":"bundle","reason":"r","component_indices":[0,1]},
			{"name":"Proj B","component_indices":[2,3,4]}]}`,
		"fence":      "```json\n" + `{"summary":"","groups":[{"name":"Proj A","component_indices":[0,1]}]}` + "\n```",
		"fence-bare": "```\n" + `{"groups":[{"name":"Proj A","component_indices":[0]}]}` + "\n```",
		"fence-one":  "```",
		"badidx":     `{"groups":[{"name":"A","component_indices":[-1,99,0,0,1]},{"name":"B","component_indices":[1,2]}]}`,
		"allbad":     `{"groups":[{"name":"A","component_indices":[99]}]}`,
		"match":      `{"groups":[{"name":"A","matched_existing_project_id":7,"component_indices":[0]},{"name":"B","matched_existing_project_id":123,"component_indices":[1]}]}`,
		"noname":     `{"groups":[{"component_indices":[0,1,2]}]}`,
		"unassigned": `{"groups":[{"name":"A b c","component_indices":[1]}]}`,
		"empty-grp":  `{"groups":[{"name":"E","component_indices":[]},{"name":"F","component_indices":[0]}]}`,
	}
	got := map[string]string{}
	for ck, cs := range comps {
		got["heur/"+ck] = normalizeGrouping(heuristicGroupScanComponents("/w/acme-root_dir", cs, nil))
		got["heur-ex/"+ck] = normalizeGrouping(heuristicGroupScanComponents("/w/acme", cs, existing))
		for lk, resp := range llm {
			r := resp
			LLMQuery = func(*gorm.DB, string, string) (string, error) { return r, nil }
			res, err := GroupScanComponents(new(gorm.DB), "/w/my_proj-root", cs, existing)
			if err != nil {
				t.Fatal(err)
			}
			got["llm/"+ck+"/"+lk] = normalizeGrouping(res)
			res, _ = GroupScanComponents(nil, "/w/my_proj-root", cs, nil)
			got["nodb/"+ck+"/"+lk] = normalizeGrouping(res)
		}
	}
	for k, v := range got {
		sum := sha256.Sum256([]byte(strings.ReplaceAll(v, "\n", "")))
		got[k] = hex.EncodeToString(sum[:8])
	}
	path := filepath.Join("testdata", "scan_grouper_golden.json")
	if *updateGolden {
		b, _ := json.MarshalIndent(got, "", " ")
		if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]string
	_ = json.Unmarshal(raw, &want)
	var bad []string
	for k, v := range got {
		if want[k] != v {
			bad = append(bad, k)
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 || len(want) != len(got) {
		t.Fatalf("%d skenario berubah (want %d got %d): %v", len(bad), len(want), len(got), bad[:min(len(bad), 10)])
	}
}
