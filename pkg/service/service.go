package service

import (
	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/repository"
)

// Services encapsulates all business services in one clean container
type Services struct {
	Projects    ProjectService
	Components  ComponentService
	Routes      RouteService
	Diagnostics DiagnosticService
	Settings    SettingsService
	Security    SecurityService
	System      SystemService
	ScaleUp     ScaleUpService
	Similarity  SimilarityService
	Organizer   OrganizerService
	SecTest     SecurityTestService
}

// NewServices initializes all domain services using the provided repositories and db
func NewServices(repos *repository.Repositories, db *gorm.DB) *Services {
	return &Services{
		Projects:    NewProjectService(repos.Projects),
		Components:  NewComponentService(repos.Components, repos.Projects),
		Routes:      NewRouteService(repos.Routes, repos.Components, repos.Projects, db),
		Diagnostics: NewDiagnosticService(repos.Diagnostics, repos.Projects, repos.Components, db),
		Settings:    NewSettingsService(repos.Settings),
		Security:    NewSecurityService(repos.Security, db),
		System:      NewSystemService(),
		ScaleUp:     NewScaleUpService(repos.Components, repos.Projects),
		Similarity:  NewSimilarityService(repos.Projects),
		Organizer:   NewOrganizerService(repos.Projects, repos.Components, db),
		SecTest:     NewSecurityTestService(repos.Projects, repos.Routes, db),
	}
}
