package codemap

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/internal/models"
	"github.com/vibeasik/vibeasik/internal/routemap"
)

// RouteGraphs menelusuri rute HTTP Go proyek (router → middleware → handler → response) dari root efektif peta.
func RouteGraphs(db *gorm.DB, projectID uint) ([]*routemap.Graph, string, error) {
	var proj models.Project
	if err := db.Preload("Components").First(&proj, projectID).Error; err != nil {
		return nil, "", fmt.Errorf("proyek %d tidak ditemukan: %w", projectID, err)
	}
	root := EffectiveRoot(proj)
	if root == "" || !isDir(root) {
		return nil, "", fmt.Errorf("root proyek %s tidak ditemukan di disk", proj.Name)
	}
	return routemap.Analyze(root), root, nil
}
