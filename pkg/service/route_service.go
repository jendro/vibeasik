package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/analysis"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/repository"
)

// RouteService defines business logic for component routes and endpoints
type RouteService interface {
	GetRoutes(ctx context.Context, componentID uint) ([]models.ComponentRoute, error)
	SyncRoutes(ctx context.Context, componentID uint) (*models.RouteSyncResult, error)
	UpdateRoute(ctx context.Context, routeID uint, updates *models.ComponentRoute) (*models.ComponentRoute, error)
	DeleteRoute(ctx context.Context, routeID uint) error
}

type routeService struct {
	routeRepo     repository.RouteRepository
	componentRepo repository.ComponentRepository
	projectRepo   repository.ProjectRepository
	db            *gorm.DB
}

// NewRouteService instantiates a new RouteService
func NewRouteService(routeRepo repository.RouteRepository, compRepo repository.ComponentRepository, projRepo repository.ProjectRepository, db *gorm.DB) RouteService {
	return &routeService{
		routeRepo:     routeRepo,
		componentRepo: compRepo,
		projectRepo:   projRepo,
		db:            db,
	}
}

func (s *routeService) GetRoutes(ctx context.Context, componentID uint) ([]models.ComponentRoute, error) {
	comp, err := s.componentRepo.FindByID(ctx, componentID)
	if err != nil {
		return nil, fmt.Errorf("component not found: %w", err)
	}

	routes, err := s.routeRepo.FindByComponentID(ctx, componentID)
	if err != nil {
		return nil, err
	}

	// If no routes recorded yet in DB, perform auto-sync on first access
	if len(routes) == 0 {
		proj, _ := s.projectRepo.FindByID(ctx, comp.ProjectID)
		if syncRes, err := analysis.SyncComponentRoutes(s.db, comp, proj); err == nil && syncRes != nil {
			return syncRes.Routes, nil
		}
	}

	return routes, nil
}

func (s *routeService) SyncRoutes(ctx context.Context, componentID uint) (*models.RouteSyncResult, error) {
	comp, err := s.componentRepo.FindByID(ctx, componentID)
	if err != nil {
		return nil, fmt.Errorf("component not found: %w", err)
	}

	proj, err := s.projectRepo.FindByID(ctx, comp.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	return analysis.SyncComponentRoutes(s.db, comp, proj)
}

func (s *routeService) UpdateRoute(ctx context.Context, routeID uint, updates *models.ComponentRoute) (*models.ComponentRoute, error) {
	route, err := s.routeRepo.FindByID(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("route not found: %w", err)
	}

	if updates.Method != "" {
		route.Method = updates.Method
	}
	if updates.Path != "" {
		route.Path = updates.Path
	}
	if updates.Handler != "" {
		route.Handler = updates.Handler
	}
	if updates.Middleware != "" {
		route.Middleware = updates.Middleware
	}
	if updates.Description != "" {
		route.Description = updates.Description
	}
	if updates.Domain != "" {
		route.Domain = updates.Domain
	}
	if updates.FunctionGroup != "" {
		route.FunctionGroup = updates.FunctionGroup
	}
	if updates.TargetRequests != "" {
		route.TargetRequests = updates.TargetRequests
	}
	if updates.Tags != "" {
		route.Tags = updates.Tags
	}

	if err := s.routeRepo.Update(ctx, route); err != nil {
		return nil, err
	}
	return route, nil
}

func (s *routeService) DeleteRoute(ctx context.Context, routeID uint) error {
	return s.routeRepo.Delete(ctx, routeID)
}
