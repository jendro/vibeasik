package analysis

import (
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestHeuristicGroupScanComponents_SingleBundle(t *testing.T) {
	rootPath := "/home/dev/workspace/acme"
	components := []models.ComponentProposal{
		{
			Name:         "Go Backend",
			Role:         models.RoleBackend,
			RelativePath: "local-engine",
			AbsolutePath: "/home/dev/workspace/acme/local-engine",
			Language:     "go",
			Framework:    "fiber",
			Port:         8765,
		},
		{
			Name:         "Nuxt Frontend",
			Role:         models.RoleFrontendClient,
			RelativePath: "studio-frontend",
			AbsolutePath: "/home/dev/workspace/acme/studio-frontend",
			Language:     "typescript",
			Framework:    "nuxt",
			Port:         3000,
		},
	}

	res := heuristicGroupScanComponents(rootPath, components, nil)
	if len(res.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(res.Groups))
	}
	g := res.Groups[0]
	if g.ProjectType != models.TypeBundle {
		t.Errorf("expected bundle project type, got %s", g.ProjectType)
	}
	if len(g.ComponentIndices) != 2 {
		t.Errorf("expected 2 components in group, got %d", len(g.ComponentIndices))
	}
}

func TestHeuristicGroupScanComponents_MatchExistingProject(t *testing.T) {
	rootPath := "/home/dev/workspace/acme"
	components := []models.ComponentProposal{
		{
			Name:         "Go Backend",
			Role:         models.RoleBackend,
			RelativePath: "local-engine",
			AbsolutePath: "/home/dev/workspace/acme/local-engine",
			Language:     "go",
			Framework:    "fiber",
		},
	}

	existingProjects := []models.Project{
		{
			ID:          42,
			Name:        "Existing Vibeasik",
			RootPath:    "/home/dev/workspace/acme",
			ProjectType: models.TypeBundle,
		},
	}

	res := heuristicGroupScanComponents(rootPath, components, existingProjects)
	if len(res.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(res.Groups))
	}
	if res.Groups[0].MatchedExistingProjectID != 42 {
		t.Errorf("expected matched existing project ID 42, got %d", res.Groups[0].MatchedExistingProjectID)
	}
}
