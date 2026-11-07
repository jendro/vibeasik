package models

import "time"

// SimilarityProjectBrief represents a compact summary of a project for comparison
type SimilarityProjectBrief struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	RootPath    string   `json:"root_path"`
	ProjectType string   `json:"project_type"`
	Status      string   `json:"status"`
	Tags        string   `json:"tags"`
	Components  []string `json:"components"` // names & roles
	Languages   []string `json:"languages"`
	Frameworks  []string `json:"frameworks"`
}

// SimilarityMatch represents a similarity comparison between two projects
type SimilarityMatch struct {
	ProjectA                   SimilarityProjectBrief `json:"project_a"`
	ProjectB                   SimilarityProjectBrief `json:"project_b"`
	OverallScore               int                    `json:"overall_score"` // 0-100
	NameScore                  int                    `json:"name_score"`    // 0-100
	PathScore                  int                    `json:"path_score"`    // 0-100
	PurposeScore               int                    `json:"purpose_score"` // 0-100
	ExactSamePath              bool                   `json:"exact_same_path"`
	PathOverlap                bool                   `json:"path_overlap"`
	ComponentRoleComplementary bool                   `json:"component_role_complementary"`
	Reasons                    []string               `json:"reasons"`
	Suggestion                 string                 `json:"suggestion"` // "merge_bundle", "duplicate_cleanup", "mark_deprecated", "review"
}

// SimilarityCluster represents a group of related projects that could be consolidated
type SimilarityCluster struct {
	SuggestedName string                   `json:"suggested_name"`
	Reason        string                   `json:"reason"`
	Confidence    int                      `json:"confidence"` // 0-100
	Projects      []SimilarityProjectBrief `json:"projects"`
	SuggestedType string                   `json:"suggested_type"` // bundle, multi_fe, monorepo
}

// SimilarityScanResult is returned after scanning all projects in the catalog
type SimilarityScanResult struct {
	TotalProjects   int                 `json:"total_projects"`
	MatchesFound    int                 `json:"matches_found"`
	HighMatches     int                 `json:"high_matches"` // score >= 70
	ScannedAt       time.Time           `json:"scanned_at"`
	Matches         []SimilarityMatch   `json:"matches"`
	SuggestedGroups []SimilarityCluster `json:"suggested_groups"`
}
