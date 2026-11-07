package adapters

// AdapterMetadata describes the capabilities and detection criteria of a scanner adapter
type AdapterMetadata struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"` // "backend" or "frontend"
	Language      string   `json:"language"`
	Framework     string   `json:"framework"`
	Icon          string   `json:"icon"`
	Description   string   `json:"description"`
	DetectRules   []string `json:"detect_rules"`
	ScanMethods   []string `json:"scan_methods"`
	Features      []string `json:"features"`
	ExampleFile   string   `json:"example_file"`
	Status        string   `json:"status"`
	LatestVersion string   `json:"latest_version,omitempty"`
	StatusBadge   string   `json:"status_badge,omitempty"`
}

// AdaptersCatalogSummary encapsulates the entire adapter ecosystem
type AdaptersCatalogSummary struct {
	Total     int               `json:"total"`
	Backend   int               `json:"backend"`
	Frontend  int               `json:"frontend"`
	Languages []string          `json:"languages"`
	Adapters  []AdapterMetadata `json:"adapters"`
}

// GetRegisteredAdaptersMetadata returns metadata for all available adapters in the system
func GetRegisteredAdaptersMetadata() AdaptersCatalogSummary {
	items := []AdapterMetadata{
		// ==================== BACKEND ADAPTERS ====================
		{
			ID:          "codeigniter",
			Name:        "PHP / CodeIgniter",
			Type:        "backend",
			Language:    "php",
			Framework:   "codeigniter",
			Icon:        "i-simple-icons-codeigniter",
			Description: "Ekstraksi rute resmi CodeIgniter 4 & 3 melalui Spark CLI runner serta static AST parser untuk nested groups, filters/middleware, resource, dan parameter pattern.",
			DetectRules: []string{
				"File CLI spark di root direktori",
				"app/Config/Routes.php (CodeIgniter 4)",
				"application/config/routes.php (CodeIgniter 3)",
				"composer.json (codeigniter4/framework atau codeigniter4/appstarter)",
			},
			ScanMethods: []string{
				"php spark routes (eksekusi CLI kernel CI4)",
				"Static parser app/Config/Routes.php (nested groups, filters, match)",
				"Static parser application/config/routes.php (CI3 verb routing)",
			},
			Features: []string{
				"Dual-Engine: CLI Kernel + Static AST Parser",
				"Nested Route Groups ($routes->group)",
				"Route Options (namespace, filter:auth)",
				"HTTP Verb Arrays ($routes->match(['put', 'post']))",
				"RESTful Resource ($routes->resource / presenter)",
				"Dynamic Wildcard Normalization ((:num), (:any))",
			},
			ExampleFile:   "app/Config/Routes.php",
			Status:        "active",
			LatestVersion: "v4.5.8 (PHP 8.2+)",
			StatusBadge:   "Stable",
		},
		{
			ID:          "laravel",
			Name:        "PHP / Laravel",
			Type:        "backend",
			Language:    "php",
			Framework:   "laravel",
			Icon:        "i-simple-icons-laravel",
			Description: "Pemindaian rute REST API dan Web Laravel menggunakan analisis statis file definisi rute, prefix groups, resource controller, dan guard autentikasi.",
			DetectRules: []string{
				"File artisan di root direktori",
				"composer.json memuat laravel/framework atau illuminate/*",
				"routes/api.php atau routes/web.php",
			},
			ScanMethods: []string{
				"Static regex parsing routes/api.php",
				"Static regex parsing routes/web.php",
			},
			Features: []string{
				"Route::get / post / put / delete / patch / options",
				"Route::prefix grouping & middleware detection",
				"Route::apiResource & Route::resource auto-expansion",
				"Array handler [Controller::class, 'method'] & Controller@method",
				"Sanctum & Auth Guard detection",
			},
			ExampleFile:   "routes/api.php",
			Status:        "active",
			LatestVersion: "v11.43.0 (PHP 8.2+)",
			StatusBadge:   "LTS / Active",
		},
		{
			ID:          "go",
			Name:        "Go Web Service",
			Type:        "backend",
			Language:    "go",
			Framework:   "fiber",
			Icon:        "i-simple-icons-go",
			Description: "Analisis sintaksis kode Go multi-framework berkinerja tinggi untuk memetakan endpoint HTTP, routing groups, serta middleware otentikasi JWT.",
			DetectRules: []string{
				"go.mod di root direktori",
				"main.go atau server.go dengan HTTP multiplexer",
			},
			ScanMethods: []string{
				"Static AST inspection seluruh file .go dalam modul",
				"Group prefix resolution",
			},
			Features: []string{
				"Mendukung Fiber v2/v3 (app.Get, app.Post, group.Group)",
				"Mendukung Gin (r.GET, r.POST, r.Group)",
				"Mendukung Echo (e.GET, e.POST, g.Group)",
				"Mendukung Chi Router & net/http standar (http.HandleFunc)",
				"Otomatis mendeteksi middleware JWT & CORS",
			},
			ExampleFile:   "main.go",
			Status:        "active",
			LatestVersion: "Go 1.24.0 (Fiber v2.52/v3, Gin v1.10)",
			StatusBadge:   "Latest Release",
		},
		{
			ID:          "express_nest",
			Name:        "Node.js (Express & NestJS)",
			Type:        "backend",
			Language:    "typescript",
			Framework:   "nestjs",
			Icon:        "i-simple-icons-nodedotjs",
			Description: "Deteksi rute backend Node.js baik berbasis TypeScript decorator NestJS maupun routing modular Express & Fastify.",
			DetectRules: []string{
				"package.json memuat @nestjs/core, express, atau fastify",
				"File controller *.controller.ts (NestJS)",
				"Direktori routes/ atau handlers/ (Express)",
			},
			ScanMethods: []string{
				"TypeScript Decorator Parser (@Controller, @Get, @Post, @Put, @Delete)",
				"Express router chaining regex inspection (router.get, app.post)",
			},
			Features: []string{
				"NestJS class & method routing decorators",
				"Express Router & sub-router mounting",
				"Fastify route declarations",
				"TypeScript & JavaScript support",
			},
			ExampleFile:   "src/**/*.controller.ts",
			Status:        "active",
			LatestVersion: "NestJS v11.0 / Express v5.0 (Node 22 LTS)",
			StatusBadge:   "Active LTS",
		},
		{
			ID:          "python",
			Name:        "Python Service (FastAPI / Flask / Django)",
			Type:        "backend",
			Language:    "python",
			Framework:   "fastapi",
			Icon:        "i-simple-icons-python",
			Description: "Deteksi endpoint API Python dari decorator FastAPI APIRouter, Flask route handler, serta Django url patterns.",
			DetectRules: []string{
				"requirements.txt atau pyproject.toml memuat fastapi, flask, atau django",
				"File entrypoint main.py, app.py, atau urls.py",
			},
			ScanMethods: []string{
				"Python AST decorator scan (@app.get, @router.post, @blueprint.route)",
				"Django urlpatterns regex resolution",
			},
			Features: []string{
				"FastAPI typed decorators (@app.get, @router.post)",
				"Flask & Blueprints routing",
				"Django path() & re_path() mapping",
				"Async def handler extraction",
			},
			ExampleFile:   "app/main.py",
			Status:        "active",
			LatestVersion: "FastAPI v0.115.8 (Python 3.13)",
			StatusBadge:   "Production Ready",
		},

		// ==================== FRONTEND ADAPTERS ====================
		{
			ID:          "nuxt",
			Name:        "Nuxt (Vue 3 / Nitro)",
			Type:        "frontend",
			Language:    "typescript",
			Framework:   "nuxt",
			Icon:        "i-simple-icons-nuxtdotjs",
			Description: "Pemindaian katalog halaman Nuxt 3/4 berbasis file-system routing serta pemetaan seluruh pemanggilan REST API di tiap komponen via ofetch & useApi.",
			DetectRules: []string{
				"nuxt.config.ts atau nuxt.config.js",
				"package.json memuat nuxt atau nuxt3",
				"Direktori pages/ atau app/pages/",
			},
			ScanMethods: []string{
				"File-system walk direktori pages/ atau app/pages/",
				"Static AST inspection pemanggilan API (useFetch, $fetch, useApi, axios)",
			},
			Features: []string{
				"Nuxt 3 & Nuxt 4 directory structure",
				"Dynamic routes [id].vue & catch-all [...slug].vue",
				"Otomatis ekstrak request API yang dipanggil per halaman",
				"Nitro server route mapping (server/api/**/*.ts)",
			},
			ExampleFile:   "pages/**/*.vue",
			Status:        "active",
			LatestVersion: "v3.16.0 / v4.0 (Vue 3.5+, Nitro 2.10)",
			StatusBadge:   "Nuxt 4 Ready",
		},
		{
			ID:          "nextjs",
			Name:        "Next.js (React 18/19)",
			Type:        "frontend",
			Language:    "typescript",
			Framework:   "next",
			Icon:        "i-simple-icons-nextdotjs",
			Description: "Pemindaian dual-routing Next.js (modern App Router app/**/page.tsx dan legacy Pages Router pages/**/*.tsx) serta route handlers server-side.",
			DetectRules: []string{
				"next.config.js, next.config.mjs, atau next.config.ts",
				"package.json memuat next",
				"Direktori app/ atau pages/",
			},
			ScanMethods: []string{
				"File-system walk app/ (App Router)",
				"File-system walk pages/ (Pages Router)",
			},
			Features: []string{
				"App Router layout, page, route handler",
				"Route segments, dynamic parameters ([id], [...slug])",
				"Pages Router legacy page mapping",
				"Target API calls detection via fetch & axios",
			},
			ExampleFile:   "app/**/page.tsx",
			Status:        "active",
			LatestVersion: "v15.2.0 (React 19 / Turbopack)",
			StatusBadge:   "Latest Release",
		},
		{
			ID:          "flutter",
			Name:        "Flutter Mobile (Dart)",
			Type:        "frontend",
			Language:    "dart",
			Framework:   "flutter",
			Icon:        "i-simple-icons-flutter",
			Description: "Ekstraksi rute navigasi dan konfigurasi HTTP client (Dio / http) untuk aplikasi mobile Android & iOS berbasis Flutter dan Dart.",
			DetectRules: []string{
				"pubspec.yaml memuat flutter",
				"Direktori lib/ dengan main.dart",
				"lib/routes.dart atau lib/router/ (go_router, auto_route)",
			},
			ScanMethods: []string{
				"Static AST inspection lib/main.dart & lib/routes.dart",
				"Ekstraksi GoRoute deklarasi dan Dio API service base URLs",
			},
			Features: []string{
				"Mendukung Flutter 3.29 & Dart 3.7",
				"Declarative routing (go_router, auto_route, GetX)",
				"Ekstraksi endpoint REST API dari Dio / package:http",
				"Material Design 3 & adaptive layout checks",
			},
			ExampleFile:   "lib/main.dart",
			Status:        "active",
			LatestVersion: "v3.29.0 (Dart 3.7.0)",
			StatusBadge:   "Latest Stable",
		},
		{
			ID:          "sveltekit",
			Name:        "SvelteKit",
			Type:        "frontend",
			Language:    "typescript",
			Framework:   "svelte",
			Icon:        "i-simple-icons-svelte",
			Description: "Pemindaian struktur file-based routing SvelteKit (+page.svelte dan +server.ts) dengan parameter dinamis [slug].",
			DetectRules: []string{
				"svelte.config.js di root direktori",
				"package.json memuat @sveltejs/kit",
				"Direktori src/routes/",
			},
			ScanMethods: []string{
				"File-system walk src/routes/ untuk file +page.svelte dan +server.ts",
			},
			Features: []string{
				"SvelteKit directory conventions",
				"Page UI routing (+page.svelte)",
				"Server endpoint routing (+server.ts)",
				"Dynamic params normalization",
			},
			ExampleFile:   "src/routes/**/+page.svelte",
			Status:        "active",
			LatestVersion: "v2.17.0 (Svelte 5 Runes)",
			StatusBadge:   "Svelte 5 Modern",
		},
		{
			ID:          "vue_router",
			Name:        "Vue Router (Vite SPA)",
			Type:        "frontend",
			Language:    "typescript",
			Framework:   "vue",
			Icon:        "i-simple-icons-vuedotjs",
			Description: "Ekstraksi deklarasi rute Vue Router dari konfigurasi array JavaScript/TypeScript dalam aplikasi Vite / Vue 3 SPA.",
			DetectRules: []string{
				"package.json memuat vue-router",
				"src/router/index.ts, src/router.ts, atau router/index.js",
			},
			ScanMethods: []string{
				"Static regex parsing createRouter() & routes array",
			},
			Features: []string{
				"Declarative routes config parsing (path, name, component)",
				"Nested child routes & layout wrappers",
				"Route meta & navigation guards context",
			},
			ExampleFile:   "src/router/index.ts",
			Status:        "active",
			LatestVersion: "Vue Router v4.5.0 (Vue 3.5.13)",
			StatusBadge:   "Stable",
		},
		{
			ID:          "react_router",
			Name:        "React Router (SPA)",
			Type:        "frontend",
			Language:    "typescript",
			Framework:   "react",
			Icon:        "i-simple-icons-react",
			Description: "Ekstraksi deklarasi JSX <Route path=...> maupun createBrowserRouter object structure dalam aplikasi React SPA.",
			DetectRules: []string{
				"package.json memuat react-router atau react-router-dom",
				"src/App.tsx, src/routes.tsx, atau src/Router.tsx",
			},
			ScanMethods: []string{
				"JSX <Route> tag inspection",
				"createBrowserRouter() object definition parser",
			},
			Features: []string{
				"React Router v6 data routers & element tags",
				"Dynamic segments (:id, :slug)",
				"Component target mapping",
			},
			ExampleFile:   "src/App.tsx",
			Status:        "active",
			LatestVersion: "React Router v7.2.0 (React 19)",
			StatusBadge:   "v7 Framework Ready",
		},
		{
			ID:          "generic_frontend",
			Name:        "Generic Frontend (SPA / MPA / HTML)",
			Type:        "frontend",
			Language:    "all",
			Framework:   "generic",
			Icon:        "i-lucide-globe",
			Description: "Fallback discovery cerdas untuk halaman statis HTML, template MPA, atau framework frontend kustom yang tidak memiliki adapter khusus.",
			DetectRules: []string{
				"File *.html di root, public/, atau views/",
				"Komponen *.vue atau *.jsx tanpa konfigurasi router baku",
			},
			ScanMethods: []string{
				"File discovery walk & view layout inspection",
			},
			Features: []string{
				"Universal fallback scanner",
				"Deteksi halaman HTML publik & templates",
				"Ekstraksi referensi aset & endpoint",
			},
			ExampleFile:   "public/index.html",
			Status:        "active",
			LatestVersion: "HTML5 / ES2024 Standards",
			StatusBadge:   "Universal",
		},
	}

	backendCount := 0
	frontendCount := 0
	langMap := make(map[string]bool)

	for _, item := range items {
		if item.Type == "backend" {
			backendCount++
		} else {
			frontendCount++
		}
		if item.Language != "all" {
			langMap[item.Language] = true
		}
	}

	var languages []string
	for l := range langMap {
		languages = append(languages, l)
	}

	return AdaptersCatalogSummary{
		Total:     len(items),
		Backend:   backendCount,
		Frontend:  frontendCount,
		Languages: languages,
		Adapters:  items,
	}
}
