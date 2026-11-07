package diagnostics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestAnalyzeProject_CrossProjectPortIsWarning(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	projA := models.Project{
		ID:   1,
		Name: "Project Alpha",
		Components: []models.Component{
			{
				ID:           10,
				Name:         "Web Frontend",
				Role:         models.RoleFrontendClient,
				AbsolutePath: dirA,
				Port:         3000,
			},
		},
	}

	projB := models.Project{
		ID:   2,
		Name: "Project Beta",
		Components: []models.Component{
			{
				ID:           20,
				Name:         "Storefront Web",
				Role:         models.RoleFrontendClient,
				AbsolutePath: dirB,
				Port:         3000,
			},
		},
	}

	allProjects := []models.Project{projA, projB}

	diag, err := AnalyzeProject(&projA, allProjects)
	if err != nil {
		t.Fatalf("AnalyzeProject returned unexpected error: %v", err)
	}

	// Aturan 45: port dev lintas proyek wajib unik agar semua proyek bisa jalan serentak
	found := false
	for _, f := range getFindingsFromJSON(diag.FindingsJSON) {
		if f.Category == "port" {
			if f.Severity != "WARNING" || f.RuleID != "PORT-SHARED-3000" {
				t.Errorf("expected WARNING PORT-SHARED-3000, got %s %s", f.Severity, f.RuleID)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("expected a port finding for shared dev port 3000")
	}

	// run_port unik dari vibeasik menghapus bentrok walau port deklarasi sama
	projA.Components[0].RunPort = 4100
	projB.Components[0].RunPort = 4110
	diag, _ = AnalyzeProject(&projA, []models.Project{projA, projB})
	for _, f := range getFindingsFromJSON(diag.FindingsJSON) {
		if f.Category == "port" {
			t.Errorf("unexpected port finding with unique run_port: %s", f.Title)
		}
	}
}

func TestAnalyzeProject_InternalPortCollisionPenalized(t *testing.T) {
	dirClash := t.TempDir()

	projWithClash := models.Project{
		ID:   1,
		Name: "Project With Internal Clash",
		Components: []models.Component{
			{
				ID:           11,
				Name:         "Admin Web",
				Role:         models.RoleFrontendAdmin,
				AbsolutePath: dirClash,
				Port:         3000,
			},
			{
				ID:           12,
				Name:         "Client Web",
				Role:         models.RoleFrontendClient,
				AbsolutePath: dirClash,
				Port:         3000,
			},
		},
	}

	allProjects := []models.Project{projWithClash}

	diag, err := AnalyzeProject(&projWithClash, allProjects)
	if err != nil {
		t.Fatalf("AnalyzeProject returned unexpected error: %v", err)
	}

	// Internal port clash in same bundle must penalize health score
	if diag.Score >= 100 {
		t.Errorf("expected score penalty for internal port collision, got %d", diag.Score)
	}

	foundCritical := false
	for _, f := range getFindingsFromJSON(diag.FindingsJSON) {
		if f.Category == "port" && f.Severity == "CRITICAL" {
			foundCritical = true
		}
	}

	if !foundCritical {
		t.Errorf("expected CRITICAL finding for internal port collision")
	}
}

func getFindingsFromJSON(raw string) []models.Finding {
	var findings []models.Finding
	if raw == "" {
		return findings
	}
	_ = json.Unmarshal([]byte(raw), &findings)
	return findings
}

func TestCodeInspector_DetectsQueryRouterAndSmells(t *testing.T) {
	sampleGoCode := `package main

import (
	"fmt"
	"net/http"
)

func brokenHandler(w http.ResponseWriter, r *http.Request) {
	// N+1 Query in loop
	for i := 0; i < 10; i++ {
		db.Find(&users, "id = ?", i)
	}

	// SQL injection
	userInput := r.URL.Query().Get("id")
	db.Raw(fmt.Sprintf("SELECT * FROM secret_data WHERE id = %s", userInput))

	// Swallowed error
	_, err := doSomething()
	_ = err

	// Wildcard CORS
	_ = cors.Config{AllowOrigins: "*"}

	// Unguarded POST route
	api.Post("/api/delete-account", handleDelete)
}
`
	tmpDir := t.TempDir()
	filePath := tmpDir + "/test_handler.go"
	if err := os.WriteFile(filePath, []byte(sampleGoCode), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	findings := InspectFile(filePath, "test_handler.go", "go")
	if len(findings) == 0 {
		t.Fatalf("expected code inspector findings, got none")
	}

	foundNPlus1 := false
	foundSQLI := false
	foundSwallowedErr := false
	foundWildcardCORS := false
	foundUnguardedRoute := false

	for _, f := range findings {
		if f.RuleID == "QUERY-NPLUS1" {
			foundNPlus1 = true
			if f.Category != "query" || f.Severity != "CRITICAL" {
				t.Errorf("unexpected N+1 meta: %+v", f)
			}
			if f.Line == 0 {
				t.Errorf("expected line number to be set")
			}
			if f.TheoryBasis == "" {
				t.Errorf("expected theory basis to be set")
			}
		}
		if f.RuleID == "QUERY-SQLI" {
			foundSQLI = true
		}
		if f.RuleID == "SMELL-SWALLOWED-ERROR" {
			foundSwallowedErr = true
		}
		if f.RuleID == "ROUTER-WILDCARD-CORS" {
			foundWildcardCORS = true
		}
		if f.RuleID == "ROUTER-UNGUARDED-MUTATION" {
			foundUnguardedRoute = true
		}
	}

	if !foundNPlus1 {
		t.Errorf("expected QUERY-NPLUS1 finding")
	}
	if !foundSQLI {
		t.Errorf("expected QUERY-SQLI finding")
	}
	if !foundSwallowedErr {
		t.Errorf("expected SMELL-SWALLOWED-ERROR finding")
	}
	if !foundWildcardCORS {
		t.Errorf("expected ROUTER-WILDCARD-CORS finding")
	}
	if !foundUnguardedRoute {
		t.Errorf("expected ROUTER-UNGUARDED-MUTATION finding")
	}
}

func TestFinding_GenerateKeyAndStatusDefaults(t *testing.T) {
	f := models.Finding{
		RuleID:     "QUERY-NPLUS1",
		TargetFile: "app/services/user.go",
		Line:       42,
		Title:      "N+1 Query in loop",
	}

	key := models.GenerateFindingKey(f)
	expectedKey := "QUERY-NPLUS1_app/services/user.go_42"
	if key != expectedKey {
		t.Errorf("expected key %s, got %s", expectedKey, key)
	}

	f.ID = "custom-id"
	if models.GenerateFindingKey(f) != "custom-id" {
		t.Errorf("expected custom-id to be returned if present")
	}
}

func TestInspectFile_FrontendSmells(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Uncached Tab Reload Smell
	uncachedTabContent := `<script setup lang="ts">
import { onMounted } from 'vue'

onMounted(() => {
  loadData()
})

const loadData = async () => {
  const res = await $fetch('/api/heavy-data')
}
</script>`
	uncachedFile := filepath.Join(tempDir, "AnalyticsTab.vue")
	_ = os.WriteFile(uncachedFile, []byte(uncachedTabContent), 0644)

	findings1 := InspectFile(uncachedFile, "app/components/detail/AnalyticsTab.vue", "vue")
	foundReload := false
	for _, f := range findings1 {
		if f.RuleID == "FE-API-RELOAD-UNCACHED" {
			foundReload = true
		}
	}
	if !foundReload {
		t.Errorf("expected FE-API-RELOAD-UNCACHED finding in AnalyticsTab.vue")
	}

	// 2. Overfetching Unused API in non-dedicated tab
	overfetchingContent := `<script setup lang="ts">
import { onMounted } from 'vue'
const { getProjectBaseKnowledge } = useApi()

onMounted(async () => {
  const bk = await getProjectBaseKnowledge(1)
})
</script>`
	overfetchingFile := filepath.Join(tempDir, "ProjectChatTab.vue")
	_ = os.WriteFile(overfetchingFile, []byte(overfetchingContent), 0644)

	findings2 := InspectFile(overfetchingFile, "app/components/detail/ProjectChatTab.vue", "vue")
	foundOverfetch := false
	for _, f := range findings2 {
		if f.RuleID == "FE-OVERFETCHING-UNUSED-DATA" {
			foundOverfetch = true
		}
	}
	if !foundOverfetch {
		t.Errorf("expected FE-OVERFETCHING-UNUSED-DATA finding in ProjectChatTab.vue")
	}
}
