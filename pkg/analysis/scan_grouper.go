package analysis

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// scanGroupSystemPrompt: prompt sistem pengelompokan komponen hasil scan menjadi project.
const scanGroupSystemPrompt = `Anda adalah Principal Software Architect yang ahli dalam struktur repositori modern (Monorepo, Polyrepo, Microservices, Web + Mobile bundles).
Tugas Anda adalah mengenali dan mengelompokkan komponen-komponen software hasil scan filesystem lokal laptop ke dalam satu atau beberapa Project yang logis.

Kaidah Penting:
1. Sinergi Komponen (1 Project Bundle): Komponen Backend (Go/Python/Node/PHP), Frontend Web (Nuxt/Next/Vue/React), dan Mobile (Flutter/RN) yang berada di direktori yang sama atau saling bekerja sama dalam 1 sistem HARUS dikelompokkan ke dalam 1 project (tipe: "bundle" atau "multi_fe").
2. Sub-service Terpisah: Jika ada folder atau aplikasi yang jelas-jelas merupakan repositori atau produk mandiri yang terpisah (misalnya CLI tool independen atau proyek terpisah di direktori berbeda), buatkan project group baru terpisah.
3. Cocokkan dengan Project Eksisting: Periksa daftar project yang sudah ada di sistem. Jika suatu komponen cocok dengan project eksisting (berdasarkan kesamaan nama atau root path), tetapkan "matched_existing_project_id" dengan ID project tersebut. Jika membuat project baru, beri nilai 0.
4. Nama Project: Berikan nama project yang jelas, deskriptif, dan manusiawi (misal: "Vibeasik Platform", "Storefront Bundle", "Inventory Service").
5. Kelengkapan: Setiap komponen HARUS dimasukkan ke dalam tepat satu group. Pastikan seluruh indeks komponen (0 sampai N-1) terdistribusi.

Keluarkan respon HANYA dalam format JSON valid berikut (tanpa blok kode pembungkus):
{
  "summary": "Ringkasan penjelasan pengelompokan dalam Bahasa Indonesia (1-2 kalimat)",
  "groups": [
    {
      "id": "group-1",
      "name": "Nama Project yang Disarankan",
      "project_type": "bundle",
      "reason": "Alasan komponen-komponen ini disatukan dalam 1 project",
      "matched_existing_project_id": 0,
      "matched_existing_project_name": "",
      "component_indices": [0, 1]
    }
  ]
}`

// describeScanComponents menyusun ringkasan satu baris per komponen untuk prompt LLM.
func describeScanComponents(components []models.ComponentProposal) []string {
	var out []string
	for i, c := range components {
		desc := fmt.Sprintf("[%d] Nama: \"%s\" | Path: \"%s\" (Abs: \"%s\") | Role: %s | Stack: %s/%s",
			i, c.Name, c.RelativePath, c.AbsolutePath, c.Role, c.Language, c.Framework)
		if c.Port > 0 {
			desc += fmt.Sprintf(" | Port: :%d", c.Port)
		}
		out = append(out, desc)
	}
	return out
}

// describeExistingProjects menyusun ringkasan project eksisting untuk prompt LLM.
func describeExistingProjects(existingProjects []models.Project) string {
	var descs []string
	for _, p := range existingProjects {
		var compNames []string
		for _, c := range p.Components {
			compNames = append(compNames, c.Name)
		}
		descs = append(descs, fmt.Sprintf("- ID %d: \"%s\" (Tipe: %s, Root: \"%s\", Komponen: [%s])",
			p.ID, p.Name, p.ProjectType, p.RootPath, strings.Join(compNames, ", ")))
	}
	if len(descs) == 0 {
		return "Belum ada project yang terdaftar di database."
	}
	return strings.Join(descs, "\n")
}

// stripCodeFence membuang pagar kode ``` di awal/akhir respons LLM.
func stripCodeFence(raw string) string {
	if !strings.HasPrefix(raw, "```") {
		return raw
	}
	lines := strings.Split(raw, "\n")
	if len(lines) < 2 {
		return raw
	}
	if strings.HasPrefix(lines[0], "```") {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// normalizeLLMGroup melengkapi bidang kosong grup hasil LLM dan mencocokkan project eksisting.
func normalizeLLMGroup(g models.ScanGroupProposal, i int, existingProjects []models.Project) models.ScanGroupProposal {
	if g.ID == "" {
		g.ID = fmt.Sprintf("group-%d", i+1)
	}
	if g.Slug == "" && g.Name != "" {
		g.Slug = slugify(g.Name)
	}
	if g.ProjectType == "" {
		g.ProjectType = models.TypeBundle
	}
	if g.MatchedExistingProjectID > 0 {
		for _, ep := range existingProjects {
			if ep.ID == g.MatchedExistingProjectID {
				g.MatchedExistingProjectName = ep.Name
				break
			}
		}
	}
	return g
}

// parseLLMGrouping memvalidasi respons LLM dan memastikan semua komponen terkelompok; nil bila tak dapat dipakai.
func parseLLMGrouping(raw, rootPath string, components []models.ComponentProposal, existingProjects []models.Project) *models.ScanGroupingResult {
	var parsed struct {
		Summary string                     `json:"summary"`
		Groups  []models.ScanGroupProposal `json:"groups"`
	}
	if err := json.Unmarshal([]byte(stripCodeFence(raw)), &parsed); err != nil || len(parsed.Groups) == 0 {
		return nil
	}

	// Validate component indices & ensure complete coverage
	assignedMap := make(map[int]bool)
	validGroups := make([]models.ScanGroupProposal, 0, len(parsed.Groups))
	for i, g := range parsed.Groups {
		g = normalizeLLMGroup(g, i, existingProjects)
		g.ComponentIndices = extractValidIndices(g.ComponentIndices, len(components), assignedMap)
		if len(g.ComponentIndices) > 0 {
			validGroups = append(validGroups, g)
		}
	}
	validGroups = attachUnassigned(validGroups, assignedMap, len(components), rootPath)
	if len(validGroups) == 0 {
		return nil
	}

	summary := parsed.Summary
	if summary == "" {
		summary = fmt.Sprintf("LLM berhasil mengenali dan mengelompokkan %d komponen ke dalam %d project.", len(components), len(validGroups))
	}
	return &models.ScanGroupingResult{Groups: validGroups, LlmUsed: true, Summary: summary}
}

func extractValidIndices(indices []int, total int, assigned map[int]bool) []int {
	var valid []int
	for _, idx := range indices {
		if idx >= 0 && idx < total && !assigned[idx] {
			assigned[idx] = true
			valid = append(valid, idx)
		}
	}
	return valid
}

// attachUnassigned menempelkan komponen yang belum terkelompok ke grup pertama (atau grup baru bila belum ada grup).
func attachUnassigned(groups []models.ScanGroupProposal, assigned map[int]bool, total int, rootPath string) []models.ScanGroupProposal {
	var unassigned []int
	for idx := 0; idx < total; idx++ {
		if !assigned[idx] {
			unassigned = append(unassigned, idx)
		}
	}
	if len(unassigned) == 0 {
		return groups
	}
	if len(groups) > 0 {
		groups[0].ComponentIndices = append(groups[0].ComponentIndices, unassigned...)
		return groups
	}
	return append(groups, models.ScanGroupProposal{
		ID:               "group-unassigned",
		Name:             cleanName(filepath.Base(rootPath)),
		Slug:             slugify(filepath.Base(rootPath)),
		ProjectType:      models.TypeBundle,
		Reason:           "Komponen yang belum terkelompokkan",
		ComponentIndices: unassigned,
	})
}

// GroupScanComponents analyzes scanned components using active LLM (with heuristic fallback)
// to identify logical project boundaries and match against existing projects in the system.
func GroupScanComponents(db *gorm.DB, rootPath string, components []models.ComponentProposal, existingProjects []models.Project) (*models.ScanGroupingResult, error) {
	if len(components) == 0 {
		return &models.ScanGroupingResult{
			Groups:  []models.ScanGroupProposal{},
			LlmUsed: false,
			Summary: "Tidak ada komponen untuk dikelompokkan.",
		}, nil
	}

	// 1. If only 1 component, do quick single mapping
	if len(components) == 1 {
		res := heuristicGroupScanComponents(rootPath, components, existingProjects)
		res.Summary = "Ditemukan 1 komponen tunggal."
		return res, nil
	}

	// 2-3. Prepare concise briefs of components and existing projects for LLM
	userPrompt := fmt.Sprintf(`Direktori Root Scan: %s
Jumlah Komponen Ditemukan: %d

Daftar Komponen:
%s

Daftar Project Eksisting di Sistem:
%s

Silakan kelompokkan komponen-komponen di atas menjadi Project Groups dalam format JSON:`,
		rootPath, len(components), strings.Join(describeScanComponents(components), "\n"), describeExistingProjects(existingProjects))

	// 4. Query active LLM
	var rawResponse string
	if db != nil {
		rawResponse, _ = queryActiveLLM(db, scanGroupSystemPrompt, userPrompt)
	}

	if rawResponse = strings.TrimSpace(rawResponse); rawResponse != "" {
		if res := parseLLMGrouping(rawResponse, rootPath, components, existingProjects); res != nil {
			return res, nil
		}
	}

	// 5. Fallback Heuristic
	return heuristicGroupScanComponents(rootPath, components, existingProjects), nil
}

// matchExistingProjectGroups mengelompokkan komponen yang cocok dengan project eksisting (root path atau path komponen).
func matchExistingProjectGroups(components []models.ComponentProposal, existingProjects []models.Project, assigned map[int]bool) []models.ScanGroupProposal {
	var groups []models.ScanGroupProposal
	for _, ep := range existingProjects {
		var matchedIndices []int
		for i, c := range components {
			if !assigned[i] && componentBelongsTo(ep, c) {
				matchedIndices = append(matchedIndices, i)
				assigned[i] = true
			}
		}
		if len(matchedIndices) > 0 {
			groups = append(groups, models.ScanGroupProposal{
				ID:                         fmt.Sprintf("group-existing-%d", ep.ID),
				Name:                       ep.Name,
				Slug:                       ep.Slug,
				ProjectType:                ep.ProjectType,
				Description:                ep.Description,
				Reason:                     fmt.Sprintf("Cocok dengan project yang sudah ada di database (%s)", ep.Name),
				MatchedExistingProjectID:   ep.ID,
				MatchedExistingProjectName: ep.Name,
				ComponentIndices:           matchedIndices,
			})
		}
	}
	return groups
}

// componentBelongsTo: path absolut komponen berada di root project, atau sama dengan path komponen project.
func componentBelongsTo(ep models.Project, c models.ComponentProposal) bool {
	if ep.RootPath != "" && ep.RootPath != "-" {
		rel, err := filepath.Rel(ep.RootPath, c.AbsolutePath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	for _, ec := range ep.Components {
		if ec.AbsolutePath != "" && ec.AbsolutePath == c.AbsolutePath {
			return true
		}
	}
	return false
}

// folderGroupKey: folder teratas path relatif komponen ("root" bila tak ada).
func folderGroupKey(relPath string) string {
	parts := strings.Split(filepath.Clean(relPath), string(filepath.Separator))
	if len(parts) > 1 && parts[0] != "." && parts[0] != "" {
		// E.g. apps/web vs apps/api vs other-proj/service
		return parts[0]
	}
	return "root"
}

// bundleProjectType menentukan tipe project untuk satu bundle komponen tersisa.
func bundleProjectType(components []models.ComponentProposal, indices []int) string {
	hasMobile := false
	frontCount := 0
	for _, idx := range indices {
		c := components[idx]
		if strings.Contains(c.Role, "mobile") {
			hasMobile = true
		}
		if strings.Contains(c.Role, "front") {
			frontCount++
		}
	}
	if frontCount > 1 {
		return models.TypeMultiFE
	}
	if !hasMobile && len(indices) == 1 {
		return models.TypeSingleApp
	}
	return models.TypeBundle
}

// folderProposals membuat grup baru per folder independen.
func folderProposals(rootPath string, folderGroups map[string][]int, startCount int) []models.ScanGroupProposal {
	var groups []models.ScanGroupProposal
	for folderName, indices := range folderGroups {
		name := cleanName(folderName)
		if folderName == "root" {
			name = cleanName(filepath.Base(rootPath))
		}
		projType := models.TypeBundle
		if len(indices) == 1 {
			projType = models.TypeSingleApp
		}
		groups = append(groups, models.ScanGroupProposal{
			ID:               fmt.Sprintf("group-new-%d", startCount+len(groups)+1),
			Name:             name,
			Slug:             slugify(name),
			ProjectType:      projType,
			Reason:           fmt.Sprintf("Dikelompokkan dalam folder independen '%s'.", folderName),
			ComponentIndices: indices,
		})
	}
	return groups
}

// groupRemaining mengelompokkan komponen yang belum cocok dengan project eksisting.
func groupRemaining(rootPath string, components []models.ComponentProposal, remaining []int, existingCount int) []models.ScanGroupProposal {
	// Group by immediate subdirectory or unified root
	folderGroups := make(map[string][]int)
	for _, idx := range remaining {
		key := folderGroupKey(components[idx].RelativePath)
		folderGroups[key] = append(folderGroups[key], idx)
	}

	// Multiple distinct folder groups
	if len(folderGroups) > 1 && len(remaining) > 3 {
		return folderProposals(rootPath, folderGroups, existingCount)
	}

	// If all components are in "root" or only 1 folder key, keep as single project
	projName := cleanName(filepath.Base(rootPath))
	return []models.ScanGroupProposal{{
		ID:               fmt.Sprintf("group-new-%d", existingCount+1),
		Name:             projName,
		Slug:             slugify(projName),
		ProjectType:      bundleProjectType(components, remaining),
		Reason:           "Dikelompokkan otomatis berdasarkan kesamaan direktori root dan sinergi stack.",
		ComponentIndices: remaining,
	}}
}

// heuristicGroupScanComponents groups components based on directory hierarchy, tech stack synergy,
// and existing project matches when LLM is unavailable.
func heuristicGroupScanComponents(rootPath string, components []models.ComponentProposal, existingProjects []models.Project) *models.ScanGroupingResult {
	if len(components) == 0 {
		return &models.ScanGroupingResult{
			Groups:  []models.ScanGroupProposal{},
			LlmUsed: false,
			Summary: "Tidak ada komponen ditemukan.",
		}
	}

	assigned := make(map[int]bool)

	// 1. Try matching components to existing projects
	groups := matchExistingProjectGroups(components, existingProjects, assigned)

	// 2. For remaining unmatched components, check if they can be grouped by top-level folder
	// or form a cohesive bundle
	var remainingIndices []int
	for i := range components {
		if !assigned[i] {
			remainingIndices = append(remainingIndices, i)
		}
	}
	if len(remainingIndices) > 0 {
		groups = append(groups, groupRemaining(rootPath, components, remainingIndices, len(groups))...)
	}

	summary := fmt.Sprintf("Berhasil mengelompokkan %d komponen ke dalam %d project berdasarkan analisis struktur direktori.", len(components), len(groups))
	return &models.ScanGroupingResult{
		Groups:  groups,
		LlmUsed: false,
		Summary: summary,
	}
}

func cleanName(name string) string {
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	res := strings.Join(words, " ")
	if res == "" {
		return "Project"
	}
	return res
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteRune('-')
		}
	}
	res := b.String()
	for strings.Contains(res, "--") {
		res = strings.ReplaceAll(res, "--", "-")
	}
	res = strings.Trim(res, "-")
	if res == "" {
		res = "project"
	}
	return res
}
