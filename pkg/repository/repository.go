package repository

import (
	"gorm.io/gorm"
)

// Repositories encapsulates all domain repository interfaces in one clean container
type Repositories struct {
	Projects    ProjectRepository
	Components  ComponentRepository
	Routes      RouteRepository
	Diagnostics DiagnosticRepository
	Settings    SettingsRepository
	Security    SecurityRepository
}

// NewRepositories initializes and returns all repositories backed by the provided GORM DB
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Projects:    NewProjectRepository(db),
		Components:  NewComponentRepository(db),
		Routes:      NewRouteRepository(db),
		Diagnostics: NewDiagnosticRepository(db),
		Settings:    NewSettingsRepository(db),
		Security:    NewSecurityRepository(db),
	}
}
