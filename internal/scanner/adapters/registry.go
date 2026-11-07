package adapters

import (
	"fmt"
	"strings"

	"github.com/vibeasik/vibeasik/internal/models"
)

// ScannerRegistry manages all registered backend and frontend scanner adapters
type ScannerRegistry struct {
	backendAdapters  []BackendAdapter
	frontendAdapters []FrontendAdapter
}

var DefaultRegistry = NewDefaultRegistry()

func NewDefaultRegistry() *ScannerRegistry {
	r := &ScannerRegistry{
		backendAdapters:  make([]BackendAdapter, 0),
		frontendAdapters: make([]FrontendAdapter, 0),
	}

	// Register Backend Adapters
	r.RegisterBackend(NewGoAdapter())
	r.RegisterBackend(NewCodeIgniterAdapter())
	r.RegisterBackend(NewLaravelAdapter())
	r.RegisterBackend(NewExpressNestAdapter())
	r.RegisterBackend(NewPythonAdapter())

	// Register Frontend Adapters (ordered by specificity)
	r.RegisterFrontend(NewNuxtAdapter())
	r.RegisterFrontend(NewNextJsAdapter())
	r.RegisterFrontend(NewSvelteKitAdapter())
	r.RegisterFrontend(NewVueRouterAdapter())
	r.RegisterFrontend(NewReactRouterAdapter())
	r.RegisterFrontend(NewGenericFrontendAdapter())

	return r
}

func (r *ScannerRegistry) RegisterBackend(adapter BackendAdapter) {
	r.backendAdapters = append(r.backendAdapters, adapter)
}

func (r *ScannerRegistry) RegisterFrontend(adapter FrontendAdapter) {
	r.frontendAdapters = append(r.frontendAdapters, adapter)
}

// DetectBackend returns the first matching backend adapter
func (r *ScannerRegistry) DetectBackend(dir string) BackendAdapter {
	for _, a := range r.backendAdapters {
		if a.Detect(dir) {
			return a
		}
	}
	return nil
}

// DetectFrontend returns the first matching frontend adapter (excluding generic fallback)
func (r *ScannerRegistry) DetectFrontend(dir string) FrontendAdapter {
	for _, a := range r.frontendAdapters {
		if a.Name() == "Generic Frontend (SPA / MPA / Custom UI)" {
			continue
		}
		if a.Detect(dir) {
			return a
		}
	}
	return nil
}

// ScanRoutes automatically executes matching adapters based on component role and directory inspection
func (r *ScannerRegistry) ScanRoutes(dir string, role string) []models.RouteInfo {
	var results []models.RouteInfo
	seen := make(map[string]int) // key -> index in results

	isBE := role == models.RoleBackend || role == "backend" || role == "api"
	isFE := role == models.RoleFrontendClient || role == models.RoleFrontendMobile || role == "frontend_client" || role == "frontend_mobile" || role == "frontend"

	// 1. Run Backend Adapters if BE or role unspecified
	if isBE || !isFE {
		for _, adapter := range r.backendAdapters {
			if adapter.Detect(dir) {
				routes, err := adapter.Scan(dir)
				if err == nil {
					for _, route := range routes {
						key := strings.ToUpper(route.Method) + " " + route.Path
						if _, exists := seen[key]; !exists {
							seen[key] = len(results)
							results = append(results, route)
						}
					}
				}
			}
		}
	}

	// 2. Run Frontend Adapters if FE or role unspecified
	if isFE || !isBE {
		// First try specific frontend adapters
		foundSpecific := false
		for _, adapter := range r.frontendAdapters {
			if adapter.Name() == "Generic Frontend (SPA / MPA / Custom UI)" {
				continue
			}
			if adapter.Detect(dir) {
				foundSpecific = true
				routes, err := adapter.Scan(dir)
				if err == nil {
					for _, route := range routes {
						key := strings.ToUpper(route.Method) + " " + route.Path
						if idx, exists := seen[key]; !exists {
							seen[key] = len(results)
							results = append(results, route)
						} else {
							// Merge target requests if existing didn't have any
							if len(results[idx].TargetRequests) == 0 && len(route.TargetRequests) > 0 {
								results[idx].TargetRequests = route.TargetRequests
							}
						}
					}
				}
			}
		}

		// If no specific adapter matched or 0 routes returned, try generic adapter
		if !foundSpecific && len(results) == 0 {
			for _, adapter := range r.frontendAdapters {
				if adapter.Name() == "Generic Frontend (SPA / MPA / Custom UI)" && adapter.Detect(dir) {
					routes, err := adapter.Scan(dir)
					if err == nil {
						for _, route := range routes {
							key := strings.ToUpper(route.Method) + " " + route.Path
							if _, exists := seen[key]; !exists {
								seen[key] = len(results)
								results = append(results, route)
							}
						}
					}
				}
			}
		}
	}

	return results
}

// Global convenience wrappers

func ScanRoutes(dir string, role string) []models.RouteInfo {
	return DefaultRegistry.ScanRoutes(dir, role)
}

func ScanBackend(dir string) []models.RouteInfo {
	return DefaultRegistry.ScanRoutes(dir, models.RoleBackend)
}

func ScanFrontend(dir string) []models.RouteInfo {
	return DefaultRegistry.ScanRoutes(dir, models.RoleFrontendClient)
}

func GetDetectedFrameworkSummary(dir string) string {
	var frameworks []string
	if be := DefaultRegistry.DetectBackend(dir); be != nil {
		frameworks = append(frameworks, fmt.Sprintf("Backend: %s", be.Name()))
	}
	if fe := DefaultRegistry.DetectFrontend(dir); fe != nil {
		frameworks = append(frameworks, fmt.Sprintf("Frontend: %s", fe.Name()))
	}
	if len(frameworks) == 0 {
		return "Framework: Generic / Auto-detect"
	}
	return strings.Join(frameworks, ", ")
}
