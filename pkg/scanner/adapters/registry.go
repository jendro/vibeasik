package adapters

import (
	"fmt"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
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
		if a.Name() == genericFrontendName {
			continue
		}
		if a.Detect(dir) {
			return a
		}
	}
	return nil
}

const genericFrontendName = "Generic Frontend (SPA / MPA / Custom UI)"

// routeCollector menampung rute tanpa duplikat (kunci: METHOD + path).
type routeCollector struct {
	results []models.RouteInfo
	seen    map[string]int // key -> index in results
}

func routeKeyOf(route models.RouteInfo) string {
	return strings.ToUpper(route.Method) + " " + route.Path
}

// add menambahkan rute yang belum ada; bila mergeTargets, rute yang sudah ada tanpa TargetRequests mewarisinya.
func (c *routeCollector) add(routes []models.RouteInfo, mergeTargets bool) {
	for _, route := range routes {
		key := routeKeyOf(route)
		idx, exists := c.seen[key]
		if !exists {
			c.seen[key] = len(c.results)
			c.results = append(c.results, route)
			continue
		}
		// Merge target requests if existing didn't have any
		if mergeTargets && len(c.results[idx].TargetRequests) == 0 && len(route.TargetRequests) > 0 {
			c.results[idx].TargetRequests = route.TargetRequests
		}
	}
}

// scanWith menjalankan adapter bila terdeteksi; mengembalikan (terdeteksi, hasil kosong bila gagal).
func scanWith(adapter RouterScanner, dir string) (detected bool, routes []models.RouteInfo) {
	if !adapter.Detect(dir) {
		return false, nil
	}
	routes, err := adapter.Scan(dir)
	if err != nil {
		return true, nil
	}
	return true, routes
}

func (r *ScannerRegistry) scanBackends(dir string, c *routeCollector) {
	for _, adapter := range r.backendAdapters {
		_, routes := scanWith(adapter, dir)
		c.add(routes, false)
	}
}

func (r *ScannerRegistry) scanFrontends(dir string, c *routeCollector) {
	// First try specific frontend adapters
	foundSpecific := false
	for _, adapter := range r.frontendAdapters {
		if adapter.Name() == genericFrontendName {
			continue
		}
		detected, routes := scanWith(adapter, dir)
		foundSpecific = foundSpecific || detected
		c.add(routes, true)
	}

	// If no specific adapter matched or 0 routes returned, try generic adapter
	if foundSpecific || len(c.results) != 0 {
		return
	}
	for _, adapter := range r.frontendAdapters {
		if adapter.Name() == genericFrontendName {
			_, routes := scanWith(adapter, dir)
			c.add(routes, false)
		}
	}
}

// roleKinds menerjemahkan peran komponen menjadi (backend, frontend).
func roleKinds(role string) (isBE, isFE bool) {
	isBE = role == models.RoleBackend || role == "backend" || role == "api"
	isFE = role == models.RoleFrontendClient || role == models.RoleFrontendMobile || role == "frontend_client" || role == "frontend_mobile" || role == "frontend"
	return
}

// ScanRoutes automatically executes matching adapters based on component role and directory inspection
func (r *ScannerRegistry) ScanRoutes(dir string, role string) []models.RouteInfo {
	c := &routeCollector{seen: make(map[string]int)}

	isBE, isFE := roleKinds(role)

	// 1. Run Backend Adapters if BE or role unspecified
	if isBE || !isFE {
		r.scanBackends(dir, c)
	}
	// 2. Run Frontend Adapters if FE or role unspecified
	if isFE || !isBE {
		r.scanFrontends(dir, c)
	}
	return c.results
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
