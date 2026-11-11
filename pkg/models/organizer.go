package models

import "time"

const (
	StrategyProductCentric = "product_centric"
	StrategyGroupedByRole  = "grouped_by_role"
)

// FolderMoveItem represents a single folder move operation
type FolderMoveItem struct {
	ID                    string `json:"id"`
	ProjectID             uint   `json:"project_id"`
	ProjectName           string `json:"project_name"`
	ComponentID           uint   `json:"component_id"`
	ComponentIDs          []uint `json:"component_ids,omitempty"` // semua komponen yang menunjuk folder ini (komponen ganda)
	ComponentName         string `json:"component_name"`
	Role                  string `json:"role"`
	CurrentPath           string `json:"current_path"`
	TargetPath            string `json:"target_path"`
	RelativeTargetPath    string `json:"relative_target_path"`
	IsGitRepo             bool   `json:"is_git_repo"`
	HasUncommittedChanges bool   `json:"has_uncommitted_changes"`
	Status                string `json:"status"` // pending, success, skipped, conflict, failed
	Error                 string `json:"error,omitempty"`
	Warning               string `json:"warning,omitempty"`
	SizeBytes             int64  `json:"size_bytes"`
}

// ReorgPlanRequest is the input to generate a reorganization plan
type ReorgPlanRequest struct {
	TargetDirectory string `json:"target_directory"`
	Strategy        string `json:"strategy"`
	ProjectIDs      []uint `json:"project_ids,omitempty"`
}

// ReorgPlan represents a full workspace reorganization proposal
type ReorgPlan struct {
	TargetDirectory string           `json:"target_directory"`
	Strategy        string           `json:"strategy"`
	TotalItems      int              `json:"total_items"`
	TotalSize       int64            `json:"total_size"`
	PlannedMoves    []FolderMoveItem `json:"planned_moves"`
	GeneratedScript string           `json:"generated_script"`
	CreatedAt       time.Time        `json:"created_at"`
}

// ExecuteMovePayload is the input to execute one or all move items
type ExecuteMovePayload struct {
	Moves    []FolderMoveItem `json:"moves"`
	UpdateDB bool             `json:"update_db"`
}

// ExecuteMoveResult represents the outcome of moving folders
type ExecuteMoveResult struct {
	SuccessCount int              `json:"success_count"`
	FailedCount  int              `json:"failed_count"`
	SkippedCount int              `json:"skipped_count"`
	Moves        []FolderMoveItem `json:"moves"`
	Message      string           `json:"message"`
}

// ConsolidatePreview contains the dry-run analysis before moving project folders
type ConsolidatePreview struct {
	ProjectID        uint             `json:"project_id"`
	ProjectName      string           `json:"project_name"`
	ProjectSlug      string           `json:"project_slug"`
	IsUnified        bool             `json:"is_unified"`
	TargetParentDir  string           `json:"target_parent_dir"`
	ProjectTargetDir string           `json:"project_target_dir"`
	Items            []FolderMoveItem `json:"items"`
	HasConflict      bool             `json:"has_conflict"`
	CanConsolidate   bool             `json:"can_consolidate"`
	TotalSizeBytes   int64            `json:"total_size_bytes"`
	// WholeRoot: proyek sudah berada di satu folder, sehingga folder root dipindahkan utuh lewat satu item.
	WholeRoot bool `json:"whole_root,omitempty"`
}

// ConsolidatePayload contains options for consolidating a project
type ConsolidatePayload struct {
	TargetParentDir string `json:"target_parent_dir,omitempty"`
}

// ConsolidateResult contains the execution outcome of project consolidation
type ConsolidateResult struct {
	Success     bool             `json:"success"`
	Message     string           `json:"message"`
	ProjectID   uint             `json:"project_id"`
	ProjectName string           `json:"project_name"`
	ProjectRoot string           `json:"project_root"`
	IsUnified   bool             `json:"is_unified"`
	MovedCount  int              `json:"moved_count"`
	Items       []FolderMoveItem `json:"items"`
	LinkedFiles []string         `json:"linked_files"`
	// CodeMapMarkdown menunjuk CODEMAP.md di root folder induk setelah Init vibeasik dijalankan ulang.
	CodeMapMarkdown string   `json:"code_map_markdown,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}
