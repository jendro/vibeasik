package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoAdapter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_go_adapter_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testgo\ngo 1.22\n"), 0644)
	mainCode := `package main
import "github.com/gofiber/fiber/v2"

func main() {
	app := fiber.New()
	app.Get("/health", health)
	api := app.Group("/api/v1")
	api.Post("/auth/login", login)
	api.Get("/users/:id", getUser)
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainCode), 0644)

	adapter := NewGoAdapter()
	if !adapter.Detect(tmpDir) {
		t.Errorf("Expected GoAdapter to detect Go project")
	}

	routes, err := adapter.Scan(tmpDir)
	if err != nil {
		t.Fatalf("GoAdapter Scan failed: %v", err)
	}
	if len(routes) != 3 {
		t.Fatalf("Expected 3 routes, got %d: %+v", len(routes), routes)
	}
}

func TestLaravelAdapter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_laravel_adapter_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "artisan"), []byte("#!/usr/bin/env php\n"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "routes"), 0755)

	apiPhp := `<?php
use Illuminate\Support\Facades\Route;
use App\Http\Controllers\AuthController;
use App\Http\Controllers\UserController;

Route::post('/login', [AuthController::class, 'login']);
Route::apiResource('users', UserController::class);
`
	_ = os.WriteFile(filepath.Join(tmpDir, "routes", "api.php"), []byte(apiPhp), 0644)

	adapter := NewLaravelAdapter()
	if !adapter.Detect(tmpDir) {
		t.Errorf("Expected LaravelAdapter to detect project")
	}

	routes, err := adapter.Scan(tmpDir)
	if err != nil {
		t.Fatalf("LaravelAdapter Scan failed: %v", err)
	}

	// 1 route for login + 5 routes for apiResource = 6 routes
	if len(routes) < 6 {
		t.Fatalf("Expected at least 6 routes for Laravel, got %d: %+v", len(routes), routes)
	}

	foundIndex := false
	foundLogin := false
	for _, r := range routes {
		if r.Path == "/api/login" && r.Method == "POST" {
			foundLogin = true
		}
		if r.Path == "/api/users" && r.Method == "GET" && r.Handler == "UserController@index" {
			foundIndex = true
		}
	}
	if !foundLogin {
		t.Errorf("Expected /api/login route")
	}
	if !foundIndex {
		t.Errorf("Expected /api/users index route from apiResource")
	}
}

func TestExpressNestAdapter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_nest_adapter_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies": {"@nestjs/core": "^10.0.0"}}`), 0644)
	_ = os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)

	ctrlContent := `
import { Controller, Get, Post, Param } from '@nestjs/common';

@Controller('products')
export class ProductsController {
  @Get()
  findAll() {}

  @Get(':id')
  findOne(@Param('id') id: string) {}

  @Post()
  create() {}
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "src", "products.controller.ts"), []byte(ctrlContent), 0644)

	adapter := NewExpressNestAdapter()
	if !adapter.Detect(tmpDir) {
		t.Errorf("Expected ExpressNestAdapter to detect NestJS project")
	}

	routes, err := adapter.Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(routes) != 3 {
		t.Fatalf("Expected 3 routes, got %d: %+v", len(routes), routes)
	}
}

func TestPythonFastApiAdapter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_fastapi_adapter_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte("fastapi==0.110.0\nuvicorn\n"), 0644)
	mainPy := `
from fastapi import FastAPI

app = FastAPI()

@app.get("/items")
async def read_items():
    return []

@app.post("/items")
async def create_item():
    return {}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte(mainPy), 0644)

	adapter := NewPythonAdapter()
	if !adapter.Detect(tmpDir) {
		t.Errorf("Expected PythonAdapter to detect project")
	}

	routes, err := adapter.Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("Expected 2 routes, got %d: %+v", len(routes), routes)
	}
}

func TestNuxtAdapter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_nuxt_adapter_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "nuxt.config.ts"), []byte("export default defineNuxtConfig({})"), 0644)
	_ = os.MkdirAll(filepath.Join(tmpDir, "pages", "assets"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "composables"), 0755)

	useApiTs := `
export const useApi = () => {
  const getAssets = () => $fetch('/api/assets', { method: 'GET' })
  const createAsset = () => $fetch('/api/assets', { method: 'POST' })
  return { getAssets, createAsset }
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "composables", "useApi.ts"), []byte(useApiTs), 0644)

	indexVue := `
<script setup>
const { getAssets } = useApi()
const assets = await getAssets()
</script>
<template><div>Home</div></template>
`
	_ = os.WriteFile(filepath.Join(tmpDir, "pages", "index.vue"), []byte(indexVue), 0644)

	detailVue := `
<script setup>
const res = await $fetch('/api/assets/' + id)
</script>
<template><div>Detail</div></template>
`
	_ = os.WriteFile(filepath.Join(tmpDir, "pages", "assets", "[id].vue"), []byte(detailVue), 0644)

	adapter := NewNuxtAdapter()
	if !adapter.Detect(tmpDir) {
		t.Errorf("Expected NuxtAdapter to detect Nuxt project")
	}

	routes, err := adapter.Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("Expected 2 routes, got %d: %+v", len(routes), routes)
	}

	foundHome := false
	foundDetail := false
	for _, r := range routes {
		if r.Path == "/" {
			foundHome = true
			if len(r.TargetRequests) == 0 {
				t.Errorf("Expected TargetRequests on '/', got none")
			}
			if r.TargetRequests[0] != "GET /api/assets" {
				t.Errorf("Expected GET /api/assets in TargetRequests, got %v", r.TargetRequests)
			}
		}
		if r.Path == "/assets/:id" {
			foundDetail = true
		}
	}
	if !foundHome || !foundDetail {
		t.Errorf("Missing expected routes (home: %v, detail: %v)", foundHome, foundDetail)
	}
}

func TestNextJsAdapter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_next_adapter_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "next.config.js"), []byte("module.exports = {}"), 0644)
	_ = os.MkdirAll(filepath.Join(tmpDir, "app", "dashboard"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "app", "api", "users"), 0755)

	pageCode := `
export default async function DashboardPage() {
  const data = await fetch('/api/dashboard/stats')
  return <div>Dashboard</div>
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "app", "dashboard", "page.tsx"), []byte(pageCode), 0644)

	routeCode := `
import { NextResponse } from 'next/server'
export async function GET() { return NextResponse.json([]) }
export async function POST() { return NextResponse.json({}) }
`
	_ = os.WriteFile(filepath.Join(tmpDir, "app", "api", "users", "route.ts"), []byte(routeCode), 0644)

	adapter := NewNextJsAdapter()
	if !adapter.Detect(tmpDir) {
		t.Errorf("Expected NextJsAdapter to detect Next project")
	}

	routes, err := adapter.Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// 1 page (/dashboard) + 2 API route methods (GET & POST on /api/users) = 3 routes
	if len(routes) != 3 {
		t.Fatalf("Expected 3 routes, got %d: %+v", len(routes), routes)
	}
}

func TestReactRouterAdapter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_react_adapter_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies": {"react-router-dom": "^6.20.0"}}`), 0644)
	_ = os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)

	appCode := `
import { Routes, Route } from 'react-router-dom';
import Home from './pages/Home';
import Login from './pages/Login';

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/login" element={<Login />} />
    </Routes>
  );
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "src", "App.tsx"), []byte(appCode), 0644)

	adapter := NewReactRouterAdapter()
	if !adapter.Detect(tmpDir) {
		t.Errorf("Expected ReactRouterAdapter to detect React project")
	}

	routes, err := adapter.Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("Expected 2 routes, got %d: %+v", len(routes), routes)
	}
}

func TestSvelteKitAdapter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_svelte_adapter_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "svelte.config.js"), []byte("export default {}"), 0644)
	_ = os.MkdirAll(filepath.Join(tmpDir, "src", "routes", "about"), 0755)

	_ = os.WriteFile(filepath.Join(tmpDir, "src", "routes", "+page.svelte"), []byte("<main>Home</main>"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "src", "routes", "about", "+page.svelte"), []byte("<main>About</main>"), 0644)

	adapter := NewSvelteKitAdapter()
	if !adapter.Detect(tmpDir) {
		t.Errorf("Expected SvelteKitAdapter to detect project")
	}

	routes, err := adapter.Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("Expected 2 routes, got %d: %+v", len(routes), routes)
	}
}

func TestRegistryMultiFramework(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_registry_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "nuxt.config.ts"), []byte("export default defineNuxtConfig({})"), 0644)
	_ = os.MkdirAll(filepath.Join(tmpDir, "pages"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "pages", "index.vue"), []byte("<template><div>Home</div></template>"), 0644)

	routes := ScanRoutes(tmpDir, "frontend_client")
	if len(routes) != 1 {
		t.Fatalf("Expected 1 route from registry, got %d", len(routes))
	}
	if routes[0].Path != "/" || routes[0].Method != "PAGE (GET)" {
		t.Errorf("Unexpected route: %+v", routes[0])
	}

	summary := GetDetectedFrameworkSummary(tmpDir)
	if !strings.Contains(summary, "Nuxt") {
		t.Errorf("Expected summary to mention Nuxt, got %s", summary)
	}
}

func TestCodeIgniterAdapter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_ci_adapter_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "spark"), []byte("#!/usr/bin/env php\n"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "app", "Config"), 0755)

	routesPhp := `<?php
use CodeIgniter\Router\RouteCollection;
$routes->get('/', 'Home::index');
$routes->options('(:any)', static function () { return service('response')->setStatusCode(200); });

$routes->group('api', ['namespace' => 'App\Controllers\Api'], static function ($routes) {
    $routes->post('auth/login', 'AuthController::login');
    $routes->get('auth/me', 'AuthController::me', ['filter' => 'auth']);
    $routes->match(['put', 'post'], 'auth/profile', 'AuthController::updateProfile', ['filter' => 'auth']);

    $routes->group('kontingen', ['filter' => 'auth:kontingen,admin'], static function ($routes) {
        $routes->get('profile', 'KontingenController::profile');
        $routes->put('officials/(:num)', 'KontingenController::officialsUpdate/$1');
        $routes->delete('officials/(:num)', 'KontingenController::officialsDelete/$1');
    });

    $routes->group('admin', ['filter' => 'auth:admin'], static function ($routes) {
        $routes->get('dashboard', 'AdminController::dashboard');
        $routes->post('users/(:num)/toggle', 'AdminController::toggleUserStatus/$1');
    });
});
`
	_ = os.WriteFile(filepath.Join(tmpDir, "app", "Config", "Routes.php"), []byte(routesPhp), 0644)

	adapter := NewCodeIgniterAdapter()
	if !adapter.Detect(tmpDir) {
		t.Errorf("Expected CodeIgniterAdapter to detect project")
	}

	routes, err := adapter.Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Routes expected:
	// GET /
	// POST /api/auth/login
	// GET /api/auth/me
	// PUT /api/auth/profile
	// POST /api/auth/profile
	// GET /api/kontingen/profile
	// PUT /api/kontingen/officials/(:num)
	// DELETE /api/kontingen/officials/(:num)
	// GET /api/admin/dashboard
	// POST /api/admin/users/(:num)/toggle
	// Total: 10 HTTP routes (OPTIONS (:any) is skipped)
	if len(routes) < 10 {
		t.Fatalf("Expected at least 10 routes, got %d: %+v", len(routes), routes)
	}

	foundRoot := false
	foundLogin := false
	foundMatchPut := false
	foundMatchPost := false
	foundOfficialsPut := false
	foundAdminDashboard := false

	for _, r := range routes {
		if r.Path == "/" && r.Method == "GET" {
			foundRoot = true
		}
		if r.Path == "/api/auth/login" && r.Method == "POST" {
			foundLogin = true
			if !strings.Contains(r.Handler, "AuthController::login") {
				t.Errorf("Unexpected login handler: %s", r.Handler)
			}
		}
		if r.Path == "/api/auth/profile" && r.Method == "PUT" {
			foundMatchPut = true
			if r.Middleware != "auth" {
				t.Errorf("Expected middleware auth, got %s", r.Middleware)
			}
		}
		if r.Path == "/api/auth/profile" && r.Method == "POST" {
			foundMatchPost = true
		}
		if r.Path == "/api/kontingen/officials/(:num)" && r.Method == "PUT" {
			foundOfficialsPut = true
			if r.Middleware != "auth:kontingen,admin" {
				t.Errorf("Expected inherited middleware auth:kontingen,admin, got %s", r.Middleware)
			}
		}
		if r.Path == "/api/admin/dashboard" && r.Method == "GET" {
			foundAdminDashboard = true
			if r.Middleware != "auth:admin" {
				t.Errorf("Expected inherited middleware auth:admin, got %s", r.Middleware)
			}
		}
	}

	if !foundRoot {
		t.Errorf("Expected root route GET /")
	}
	if !foundLogin {
		t.Errorf("Expected POST /api/auth/login")
	}
	if !foundMatchPut || !foundMatchPost {
		t.Errorf("Expected match routes for /api/auth/profile (PUT & POST)")
	}
	if !foundOfficialsPut {
		t.Errorf("Expected PUT /api/kontingen/officials/(:num)")
	}
	if !foundAdminDashboard {
		t.Errorf("Expected GET /api/admin/dashboard")
	}
}

func TestCodeIgniterOnSilatsoloBE(t *testing.T) {
	// Uji pada proyek CodeIgniter nyata: set VIBEASIK_TEST_CODEIGNITER_PATH ke folder proyeknya.
	silatsoloPath := os.Getenv("VIBEASIK_TEST_CODEIGNITER_PATH")
	if silatsoloPath == "" {
		t.Skip("VIBEASIK_TEST_CODEIGNITER_PATH not set, skipping real project test")
	}
	if _, err := os.Stat(silatsoloPath); err != nil {
		t.Skip("CodeIgniter project not found on disk, skipping real project test")
	}

	adapter := NewCodeIgniterAdapter()
	if !adapter.Detect(silatsoloPath) {
		t.Fatalf("Expected adapter to detect silatsoloBE")
	}

	routes, err := adapter.Scan(silatsoloPath)
	if err != nil {
		t.Fatalf("Scan on silatsoloBE failed: %v", err)
	}

	if len(routes) < 100 {
		t.Fatalf("Expected over 100 routes in silatsoloBE, got %d", len(routes))
	}
}
