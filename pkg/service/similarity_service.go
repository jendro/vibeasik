package service

import (
	"context"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
)

// SimilarityService handles duplicate detection and similarity analysis across projects
type SimilarityService interface {
	ScanSimilarities(ctx context.Context) (*models.SimilarityScanResult, error)
	CompareTwoProjects(ctx context.Context, idA, idB uint) (*models.SimilarityMatch, error)
}

type similarityService struct {
	projectRepo repository.ProjectRepository
}

// NewSimilarityService instantiates a new SimilarityService
func NewSimilarityService(projectRepo repository.ProjectRepository) SimilarityService {
	return &similarityService{projectRepo: projectRepo}
}

// Suffixes and prefixes commonly used in vibecoding/polyglot repos
var techSuffixRegex = regexp.MustCompile(`(?i)(backend|frontend|mobile|admin|client|web|api|gui|server|service|fe|be|app|v\d+|\d+)$`)
var nonAlphaRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func normalizeProjectName(name string) string {
	cleaned := strings.ToLower(strings.TrimSpace(name))
	cleaned = nonAlphaRegex.ReplaceAllString(cleaned, "")
	// Iteratively strip tech suffixes like "be", "fe", "mobile", "erp2", etc.
	for {
		stripped := techSuffixRegex.ReplaceAllString(cleaned, "")
		if stripped == cleaned || len(stripped) < 3 {
			break
		}
		cleaned = stripped
	}
	return cleaned
}

// levenshteinDistance computes edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n, m := len(r1), len(r2)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}

	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
		dp[i][0] = i
	}
	for j := 0; j <= m; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			dp[i][j] = int(math.Min(
				float64(dp[i-1][j]+1),
				math.Min(
					float64(dp[i][j-1]+1),
					float64(dp[i-1][j-1]+cost),
				),
			))
		}
	}
	return dp[n][m]
}

func calculateStringSimilarity(s1, s2 string) int {
	if s1 == "" && s2 == "" {
		return 100
	}
	if s1 == "" || s2 == "" {
		return 0
	}
	if s1 == s2 {
		return 100
	}
	dist := levenshteinDistance(s1, s2)
	maxLen := math.Max(float64(len([]rune(s1))), float64(len([]rune(s2))))
	score := int((1.0 - (float64(dist) / maxLen)) * 100.0)
	if score < 0 {
		return 0
	}
	return score
}

func tokenizeText(text string) map[string]struct{} {
	words := strings.Fields(strings.ToLower(text))
	tokens := make(map[string]struct{})
	stopWords := map[string]struct{}{
		"the": {}, "and": {}, "with": {}, "for": {}, "app": {}, "project": {}, "dan": {}, "yang": {}, "untuk": {}, "aplikasi": {}, "proyek": {},
	}
	for _, w := range words {
		cleaned := nonAlphaRegex.ReplaceAllString(w, "")
		if len(cleaned) > 2 {
			if _, isStop := stopWords[cleaned]; !isStop {
				tokens[cleaned] = struct{}{}
			}
		}
	}
	return tokens
}

func calculateJaccardSimilarity(t1, t2 map[string]struct{}) int {
	if len(t1) == 0 && len(t2) == 0 {
		return 0
	}
	intersection := 0
	for k := range t1 {
		if _, exists := t2[k]; exists {
			intersection++
		}
	}
	union := len(t1) + len(t2) - intersection
	if union == 0 {
		return 0
	}
	return int((float64(intersection) / float64(union)) * 100.0)
}

func toProjectBrief(p models.Project) models.SimilarityProjectBrief {
	compNames := make([]string, 0, len(p.Components))
	langMap := make(map[string]struct{})
	fwMap := make(map[string]struct{})

	for _, c := range p.Components {
		compNames = append(compNames, c.Name+" ("+c.Role+")")
		if c.Language != "" && c.Language != "auto" {
			langMap[c.Language] = struct{}{}
		}
		if c.Framework != "" {
			fwMap[c.Framework] = struct{}{}
		}
	}

	langs := make([]string, 0, len(langMap))
	for l := range langMap {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	fws := make([]string, 0, len(fwMap))
	for f := range fwMap {
		fws = append(fws, f)
	}
	sort.Strings(fws)

	return models.SimilarityProjectBrief{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		RootPath:    p.RootPath,
		ProjectType: p.ProjectType,
		Status:      p.Status,
		Tags:        p.Tags,
		Components:  compNames,
		Languages:   langs,
		Frameworks:  fws,
	}
}

func (s *similarityService) comparePair(pA, pB models.Project) models.SimilarityMatch {
	match := models.SimilarityMatch{
		ProjectA: toProjectBrief(pA),
		ProjectB: toProjectBrief(pB),
		Reasons:  make([]string, 0),
	}

	scoreRootPaths(&match, pA, pB)
	scoreComponentPaths(&match, pA, pB)
	normA, normB := scoreNames(&match, pA, pB)
	scoreComplementaryRoles(&match, pA, pB)
	scorePurpose(&match, pA, pB)
	computeOverall(&match, pA, pB, normA == normB && normA != "")

	return match
}

func isRealPath(p string) bool { return p != "" && p != "-" }

func scoreRootPaths(match *models.SimilarityMatch, pA, pB models.Project) {
	pathA := filepath.Clean(strings.TrimSpace(pA.RootPath))
	pathB := filepath.Clean(strings.TrimSpace(pB.RootPath))

	if isRealPath(pathA) && pathA == pathB {
		match.ExactSamePath = true
		match.PathScore = 100
		match.Reasons = append(match.Reasons, "Root folder persis sama ("+pathA+")")
		return
	}
	if !isRealPath(pathA) || !isRealPath(pathB) {
		return
	}
	if strings.HasPrefix(pathA, pathB) || strings.HasPrefix(pathB, pathA) {
		match.PathOverlap = true
		match.PathScore = 80
		match.Reasons = append(match.Reasons, "Folder berada di hirarki sub-direktori yang sama")
	} else if filepath.Base(pathA) == filepath.Base(pathB) {
		match.PathScore = 70
		match.Reasons = append(match.Reasons, "Nama folder dasar (folder basename) sama persis: "+filepath.Base(pathA))
	}
}

// scoreComponentPaths memeriksa path komponen yang identik antar proyek.
func scoreComponentPaths(match *models.SimilarityMatch, pA, pB models.Project) {
	for _, cA := range pA.Components {
		cAPath := filepath.Clean(strings.TrimSpace(cA.AbsolutePath))
		for _, cB := range pB.Components {
			cBPath := filepath.Clean(strings.TrimSpace(cB.AbsolutePath))
			if cAPath != "" && cAPath == cBPath {
				match.ExactSamePath = true
				match.PathScore = 100
				match.Reasons = append(match.Reasons, "Memiliki komponen dengan path folder yang identik: "+cA.Name)
				return
			}
		}
	}
}

// scoreNames mengisi NameScore dan mengembalikan nama ternormalisasi kedua proyek.
func scoreNames(match *models.SimilarityMatch, pA, pB models.Project) (string, string) {
	normA := normalizeProjectName(pA.Name)
	normB := normalizeProjectName(pB.Name)
	rawScore := calculateStringSimilarity(strings.ToLower(pA.Name), strings.ToLower(pB.Name))
	normScore := calculateStringSimilarity(normA, normB)

	match.NameScore = int(math.Max(float64(rawScore), float64(normScore)))
	if normA != "" && normB != "" && normA == normB {
		match.NameScore = 100
		match.Reasons = append(match.Reasons, "Nama dasar produk identik ('"+normA+"')")
	} else if match.NameScore >= 70 {
		match.Reasons = append(match.Reasons, "Kemiripan nama proyek tinggi ("+pA.Name+" vs "+pB.Name+")")
	}
	return normA, normB
}

// projectHasRoles melaporkan apakah proyek punya komponen backend dan frontend/mobile.
func projectHasRoles(p models.Project) (hasBE, hasFE bool) {
	for _, c := range p.Components {
		if c.Role == models.RoleBackend {
			hasBE = true
		}
		if strings.Contains(c.Role, "front") || c.Role == models.RoleFrontendMobile {
			hasFE = true
		}
	}
	return hasBE, hasFE
}

func scoreComplementaryRoles(match *models.SimilarityMatch, pA, pB models.Project) {
	hasBEA, hasFEA := projectHasRoles(pA)
	hasBEB, hasFEB := projectHasRoles(pB)
	if (hasBEA && hasFEB && !hasFEA) || (hasFEA && hasBEB && !hasBEA) {
		match.ComponentRoleComplementary = true
		match.Reasons = append(match.Reasons, "Peran komponen saling melengkapi (satu sisi Backend, sisi lain Frontend/Mobile)")
	}
}

func scorePurpose(match *models.SimilarityMatch, pA, pB models.Project) {
	textA := pA.ShortDescription + " " + pA.Description + " " + pA.Tags
	textB := pB.ShortDescription + " " + pB.Description + " " + pB.Tags
	match.PurposeScore = calculateJaccardSimilarity(tokenizeText(textA), tokenizeText(textB))
	if match.PurposeScore >= 35 {
		match.Reasons = append(match.Reasons, "Topik dan kata kunci deskripsi memiliki kesamaan konteks")
	}
}

func computeOverall(match *models.SimilarityMatch, pA, pB models.Project, sameNormName bool) {
	if match.ExactSamePath {
		match.OverallScore = 100
		match.Suggestion = "duplicate_cleanup"
		return
	}
	const (
		weightName    = 0.50
		weightPath    = 0.20
		weightComp    = 0.20
		weightPurpose = 0.10
	)
	compScore := 0
	if match.ComponentRoleComplementary {
		compScore = 90
	}
	calculated := (float64(match.NameScore) * weightName) +
		(float64(match.PathScore) * weightPath) +
		(float64(compScore) * weightComp) +
		(float64(match.PurposeScore) * weightPurpose)

	// Boost if name is normalized identical and components are complementary
	if sameNormName && match.ComponentRoleComplementary {
		calculated = math.Max(calculated, 92.0)
	}
	match.OverallScore = int(math.Min(100.0, math.Round(calculated)))
	match.Suggestion = suggestionFor(match, pA, pB)
}

func suggestionFor(match *models.SimilarityMatch, pA, pB models.Project) string {
	switch {
	case match.OverallScore >= 80:
		if !match.ComponentRoleComplementary && (pA.Status == models.StatusDeprecated || pB.Status == models.StatusDeprecated) {
			return "mark_deprecated"
		}
		return "merge_bundle"
	case match.OverallScore >= 60:
		return "review"
	}
	return "low_similarity"
}

func (s *similarityService) ScanSimilarities(ctx context.Context) (*models.SimilarityScanResult, error) {
	projects, err := s.projectRepo.FindAll(ctx, "", "")
	if err != nil {
		return nil, err
	}

	result := &models.SimilarityScanResult{
		TotalProjects:   len(projects),
		ScannedAt:       time.Now(),
		Matches:         make([]models.SimilarityMatch, 0),
		SuggestedGroups: make([]models.SimilarityCluster, 0),
	}

	if len(projects) < 2 {
		return result, nil
	}

	// Pairwise compare
	for i := 0; i < len(projects); i++ {
		for j := i + 1; j < len(projects); j++ {
			match := s.comparePair(projects[i], projects[j])
			// Only include if meaningful similarity detected (>= 50% or exact path)
			if match.OverallScore >= 50 || match.ExactSamePath {
				result.Matches = append(result.Matches, match)
				if match.OverallScore >= 70 {
					result.HighMatches++
				}
			}
		}
	}

	// Sort matches by overall score descending
	sort.Slice(result.Matches, func(i, j int) bool {
		return result.Matches[i].OverallScore > result.Matches[j].OverallScore
	})

	result.MatchesFound = len(result.Matches)

	// Build cluster suggestions (Group projects sharing the same root name)
	clusterMap := make(map[string][]models.Project)
	for _, p := range projects {
		baseName := normalizeProjectName(p.Name)
		if len(baseName) >= 3 {
			clusterMap[baseName] = append(clusterMap[baseName], p)
		}
	}

	for baseName, group := range clusterMap {
		if len(group) >= 2 {
			briefs := make([]models.SimilarityProjectBrief, 0, len(group))
			for _, p := range group {
				briefs = append(briefs, toProjectBrief(p))
			}
			result.SuggestedGroups = append(result.SuggestedGroups, models.SimilarityCluster{
				SuggestedName: strings.Title(baseName),
				Reason:        "Ditemukan " + string(rune('0'+len(group))) + " proyek terpisah dengan basis nama '" + baseName + "' yang kemungkinan merupakan satu kesatuan bundle aplikasi.",
				Confidence:    88,
				Projects:      briefs,
				SuggestedType: models.TypeBundle,
			})
		}
	}

	return result, nil
}

func (s *similarityService) CompareTwoProjects(ctx context.Context, idA, idB uint) (*models.SimilarityMatch, error) {
	pA, err := s.projectRepo.FindByID(ctx, idA)
	if err != nil {
		return nil, err
	}
	pB, err := s.projectRepo.FindByID(ctx, idB)
	if err != nil {
		return nil, err
	}

	match := s.comparePair(*pA, *pB)
	return &match, nil
}
