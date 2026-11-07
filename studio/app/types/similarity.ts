export interface SimilarityProjectBrief {
  id: number
  name: string
  slug: string
  root_path: string
  project_type: string
  status: string
  tags: string
  components: string[]
  languages: string[]
  frameworks: string[]
}

export interface SimilarityMatch {
  project_a: SimilarityProjectBrief
  project_b: SimilarityProjectBrief
  overall_score: number
  name_score: number
  path_score: number
  purpose_score: number
  exact_same_path: boolean
  path_overlap: boolean
  component_role_complementary: boolean
  reasons: string[]
  suggestion: 'merge_bundle' | 'duplicate_cleanup' | 'mark_deprecated' | 'review' | string
}

export interface SimilarityCluster {
  suggested_name: string
  reason: string
  confidence: number
  projects: SimilarityProjectBrief[]
  suggested_type: string
}

export interface SimilarityScanResult {
  total_projects: number
  matches_found: number
  high_matches: number
  scanned_at: string
  matches: SimilarityMatch[]
  suggested_groups: SimilarityCluster[]
}
