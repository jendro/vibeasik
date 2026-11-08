package adapters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func checkGoldenJSON(t *testing.T, name string, v any) {
	t.Helper()
	got, _ := json.MarshalIndent(v, "", " ")
	path := filepath.Join("testdata", name+".golden.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(got) {
		t.Errorf("%s: keluaran berubah", name)
	}
}

var goldenFixtures = map[string]map[string]string{
	"go1": {
		"go.mod": "module x\n",
		"main.go": `package main
// app.Get("/commented", c)
func main() {
	app := fiber.New()
	app.Get("/health", health)
	app.Get("/inline", func(c *fiber.Ctx) error { return nil })
	api := app.Group("/api/v1")
	v2 := api.Group("/v2")
	api.Post("/auth/login", controllers.Login)
	api.Get("/users/:id", authMiddleware, controllers.GetCurrentUser)
	v2.Get("/assets", jwtGuard, getAssets)
	v2.Delete("/assets/:id", func(c *fiber.Ctx) error { return nil })
	r.HandleFunc("/mux/items", itemsHandler).Methods("GET", "POST")
	r.HandleFunc("/mux/any", anyHandler).Methods(verbs)
	http.HandleFunc("GET /std/ping", PingHandler)
	http.HandleFunc("/std/all", AllHandler)
	e.POST("/gin/projects", createProject)
}
`,
		"routes/extra.go":        "package routes\nfunc R(){ g.Put(\"/extra/:id\", Protect(h)) ; g.Patch(\"/patch\", h)}\n",
		"routes/extra_test.go":   "package routes\nfunc T(){ g.Get(\"/nope\", h) }\n",
		"internal/api/server.go": "package api\nfunc S(){ s.Get(\"/srv/family\", h)\n s.Options(\"/srv/opt\", h) }\n",
	},
	"go2": {
		"pkg/svc/handlers.go":    "package svc\nfunc H(){\n x := app.Group(\"/svc\")\n x.Get(\"/a\", ha)\n x.Post(\"/b\", hb) }\n",
		"pkg/svc/none.go":        "package svc\nfunc N(){}\n",
		"vendor/lib/lib.go":      "package lib\nfunc L(){ a.Get(\"/vendor\", h) }\n",
		"node_modules/x/main.go": "package x\n",
	},
	"ci4": {
		"app/Config/Routes.php": `<?php
// $routes->get('commented', 'X::y');
$routes->get('/', 'Home::index');
$routes->add('legacy', 'Legacy::run');
$routes->options('(:any)', 'Cors::options');
$routes->group('api', ['namespace' => 'App\Controllers\Api', 'filter' => 'auth'], function ($routes) {
    $routes->get('users', 'Users::index');
    $routes->post('users', 'Users::create', ['filter' => 'admin']);
    $routes->match(['put', 'patch'], 'users/(:num)', 'Users::update/$1');
    $routes->resource('photos');
    $routes->resource('items', ['controller' => 'ItemsCtl']);
    $routes->group('v2', function ($routes) {
        $routes->get('ping', function () { return 1; });
        $routes->presenter('docs');
    });
});
$routes->get('after', 'After::index');
`,
		"application/config/routes.php": `<?php
$route['default_controller'] = 'welcome';
$route['404_override'] = '';
// $route['x'] = 'y';
$route['api/users'] = 'api/users/index';
$route['api/users']['post'] = 'api/users/create';
$route['api/users']['DELETE'] = 'api/users/remove';
$route['dup'] = 'a';
$route['dup'] = 'b';
`,
	},
	"laravel": {
		"artisan": "x",
		"routes/api.php": `<?php
Route::post('/login', [AuthController::class, 'login']);
Route::prefix('v1')->group(function () {
  Route::get('/me', 'UserController@me')->middleware('auth:sanctum');
  Route::apiResource('users', UserController::class);
  Route::any('/all', function () { return 1; });
});
Route::resource('posts', PostController::class);
`,
		"routes/web.php": "<?php\nRoute::get('/', [HomeController::class, 'index']);\n",
	},
	"nest": {
		"package.json": `{"dependencies": {"@nestjs/core": "1", "express": "4"}}`,
		"src/app.controller.ts": `
@Controller('cats')
@UseGuards(AuthGuard)
export class CatsController {
  @Get()
  findAll() {}
  @Get(':id')
  async findOne() {}
  @Post('new')
  create() {}
}`,
		"src/other.controller.js": "@Controller()\nclass A { @Delete('x') rm(){} }",
		"src/app.ts":              "app.get('/express/a', handler)\n// app.get('/no', h)\napp.post('b', verifyToken, handler);\nrouter.put('/c', protectX)\n",
		"routes/users.js":         "router.delete('/users/:id', authMw, rm)\n",
		"src/routes/dup.ts":       "api.patch('/p', h)\n",
		"node_modules/x/app.js":   "app.get('/nm', h)",
	},
	"py": {
		"requirements.txt": "fastapi\n",
		"main.py": `# @app.get("/c")
@app.get("/items")
async def list_items():
    pass
@router.post("items/new", dependencies=[Depends(auth_token)])
def create():
    pass
@app.get("/nodef")
x = 1
@app.route("/flask", methods=["GET", "POST"])
def f(): pass
@app.route("/flask2")
def g(): pass
@app.route("/flask3", methods=['X'])
urlpatterns = [path('users/', views.user_list), path('', views.home)]
`,
		"venv/lib.py":     "@app.get('/venv')\ndef v(): pass\n",
		"tests/test_x.py": "@app.get('/t')\ndef v(): pass\n",
		"x_test.py":       "@app.get('/tt')\n",
	},
	"nuxt": {
		"nuxt.config.ts": "export default {}",
		"app/pages/index.vue": `<script setup>
import Card from '@/components/Card.vue'
import Rel from '../components/Rel.vue'
import Ext from 'vue'
const { data } = await useFetch('/api/dashboard')
</script>`,
		"app/pages/login.vue":             "<script>definePageMeta({ middleware: 'auth' })</script>",
		"app/pages/projects/index.vue":    "useAuth()",
		"app/pages/projects/[id].vue":     "$fetch(`/api/projects/${id}`, { method: 'POST' })\nfetchThing()",
		"app/pages/blog/[...slug].vue":    "x",
		"app/pages/_legacy.vue":           "requireAuth",
		"app/pages/settings/page.vue":     "x",
		"app/components/Card.vue":         "axios.post('/api/card')\n",
		"app/components/Rel.vue":          "fetch('/api/rel')",
		"app/composables/useThing.ts":     "export const fetchThing = async () => { return $fetch('/api/thing', { method: 'DELETE' }) }\nexport function loadList() { return fetch(`${base}/list`) }\nconst o = { getX: () => axios.get('/api/x') }\nexport const useApi = () => '/api/skip'\n",
		"server/api/users.get.ts":         "x",
		"server/api/users.post.ts":        "x",
		"server/api/users/[id].put.ts":    "x",
		"server/api/users/[id].delete.ts": "x",
		"server/api/health.ts":            "x",
		"server/api/readme.txt":           "x",
	},
	"nuxt2": {
		"package.json": `{"dependencies":{"nuxt":"2"}}`,
		"app.vue":      "<template/>fetch('/api/app')",
	},
	"next": {
		"package.json":                   `{"dependencies":{"next":"14","react":"18"}}`,
		"app/page.tsx":                   "export default function P(){ fetch('/api/home') }",
		"app/(marketing)/about/page.jsx": "x",
		"app/dash/[id]/page.tsx":         "const s = await getServerSession()",
		"app/docs/[...slug]/page.js":     "auth()",
		"app/shop/[[...opt]]/page.tsx":   "x",
		"app/api/items/route.ts":         "export async function GET() {}\nexport function POST() {}\nexport async function GET2() {}",
		"app/other/route.js":             "export function DELETE() {}",
		"app/notes.md":                   "x",
		"src/app/page.tsx":               "x",
		"pages/index.tsx":                "fetch('/api/pi')",
		"pages/about.jsx":                "x",
		"pages/blog/[slug].js":           "x",
		"pages/[...rest].tsx":            "x",
		"pages/_app.tsx":                 "x",
		"pages/api/ping.ts":              "x",
		"pages/api/hello.js":             "x",
		"pages/readme.md":                "x",
		"pages/nested/index.tsx":         "x",
		"src/pages/s.tsx":                "x",
	},
	"svelte": {
		"svelte.config.js":                   "x",
		"src/routes/+page.svelte":            "fetch('/api/s')",
		"src/routes/about/+page.svelte":      "x",
		"src/routes/items/[id]/+page.svelte": "x",
		"src/routes/api/x/+server.ts":        "export async function GET() {}\nexport function POST() {}",
		"src/routes/other/+server.js":        "export function PATCH() {}",
		"src/routes/lib.ts":                  "x",
	},
	"svelte2": {
		"package.json":        `{"devDependencies":{"@sveltejs/kit":"1"}}`,
		"routes/+page.svelte": "x",
	},
	"vue": {
		"package.json": `{"dependencies":{"vue":"3","vue-router":"4"}}`,
		"src/router/index.ts": `const routes = [
  { path: '/', name: 'home', component: Home },
  { path: '/about', component: About },
  { path: '/x', name: 'x' },
  { path: '/y' },
]`,
		"src/views/Home.vue":       "x",
		"src/views/About.vue":      "fetch('/api/about')",
		"src/views/index.vue":      "fetch('/api/vi')",
		"src/views/deep/index.vue": "x",
		"src/pages/p.vue":          "x",
		"src/views/readme.txt":     "x",
	},
	"react": {
		"package.json": `{"dependencies":{"react":"18","react-router-dom":"6"}}`,
		"src/App.tsx": `import Home from './pages/Home'
<Route path="/" element={<Home/>} />
<Route path="/about" element={<About />} />
const router = createBrowserRouter([{ path: '/dash', element: <Dash/> }, { path: '/', element: <Dup/> }])
fetch('/api/app')
`,
		"src/pages/Home.tsx":      "export const x = 1",
		"src/pages/index.jsx":     "x",
		"src/pages/deep/index.js": "x",
		"src/pages/util.ts":       "x",
		"src/screens/Scr.tsx":     "x",
	},
	"generic": {
		"index.html":          "x",
		"pages/index.html":    "x",
		"pages/a.svelte":      "x",
		"pages/b.vue":         "fetch('/api/b')",
		"pages/sub/index.tsx": "x",
		"pages/skip.txt":      "x",
		"src/views/v.vue":     "x",
		"app/z.jsx":           "x",
		"src/app/q.html":      "x",
	},
}

func TestAdaptersGolden(t *testing.T) {
	names := make([]string, 0, len(goldenFixtures))
	for n := range goldenFixtures {
		names = append(names, n)
	}
	sort.Strings(names)
	type adRow struct {
		Adapter string
		Detect  bool
		Routes  []models.RouteInfo
		Err     string
	}
	type fxRow struct {
		Fixture  string
		Adapters []adRow
		Summary  string
		BE       []models.RouteInfo
		FE       []models.RouteInfo
		Auto     []models.RouteInfo
		API      []models.RouteInfo
	}
	var out []fxRow
	for _, n := range names {
		root := t.TempDir()
		writeTree(t, root, goldenFixtures[n])
		row := fxRow{Fixture: n}
		var all []RouterScanner
		for _, a := range DefaultRegistry.backendAdapters {
			all = append(all, a)
		}
		for _, a := range DefaultRegistry.frontendAdapters {
			all = append(all, a)
		}
		for _, a := range all {
			ar := adRow{Adapter: a.Name(), Detect: a.Detect(root)}
			if ar.Detect {
				r, err := a.Scan(root)
				ar.Routes = r
				if err != nil {
					ar.Err = err.Error()
				}
			}
			row.Adapters = append(row.Adapters, ar)
		}
		row.Summary = GetDetectedFrameworkSummary(root)
		row.BE = ScanBackend(root)
		row.FE = ScanFrontend(root)
		row.Auto = ScanRoutes(root, "")
		row.API = ScanRoutes(root, "api")
		out = append(out, row)
	}
	checkGoldenJSON(t, "adapters", out)
}

func TestAdapterHelpersGolden(t *testing.T) {
	paths := []string{"/", "/health", "/api/ping", "/x/login", "/auth/login", "/register", "/auth/register", "/auth/refresh", "/auth/logout", "/users", "/assets", "/family/x", "/projects", "/other",
		"/projects/:id", "/projects/[id]", "/ports", "/settings", "/scan", "/mcp", "/auth", "/profile", "/user/1", "/asset", "/aset", "/finance", "/keuangan", "/sync", "/categories", "/zzz"}
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "ALL", "HEAD"}
	type d struct{ M, P, Ep string }
	var descs []d
	for _, m := range methods {
		for _, p := range paths {
			descs = append(descs, d{m, p, inferEndpointDescription(m, p)})
		}
	}
	pages := map[string]string{}
	for _, p := range paths {
		pages[p] = InferFrontendPageDescription(p)
	}
	handlers := map[string]string{}
	for _, h := range []string{"controllers.Login", "controllers.GetAsset", "controllers.Nope", "inline"} {
		handlers[h] = describeGoHandler(h, "GET", "/zzz")
	}
	fn := map[string]string{}
	ParseApiFunctionsFromSource(`
export const getUsers = async () => { return $fetch('/api/users') }
export const createUser = (d: any) => axios.post(`+"`${apiBase}/users/${id}`"+`, d)
export function removeUser(id) { return fetch('/api/users/' + id, { method: 'DELETE' }) }
const obj = { listX: () => fetch('/x?y=1') }
export const noUrl = () => 1
export const useApi = () => '/api/skip'
function plain() { return http.put('https://h.example.com/api/p') }
`, fn)
	clean := map[string]string{}
	for _, u := range []string{"'/api/x'", "`${base}/y//z/`", "${apiUrl}/q?x=1", "/a/${id}/b/${a.b}", "", "/", "http://h/x"} {
		clean[u] = CleanEndpointURL(u)
	}
	urls := map[string]string{}
	for _, u := range []string{"/a//b/", "a/b", "", " /x / y ", "/"} {
		urls[u] = CleanURLPath(u)
	}
	sparkOut := `+--------+----------+------+---------+----------------+---------------+
| Method | Route    | Name | Handler | Before Filters | After Filters |
+--------+----------+------+---------+----------------+---------------+
| GET    | /        | »    | \App\Controllers\Home::index |  | toolbar |
| POST   | users/([0-9]+) | » | (Closure) | auth | |
| OPTIONS | (.*) | » | x | | |
| GET    | files/(.*) | » | x | | |
| FOO    | bad      | » | x | | |
| GET    | dup      | » | a | | |
| GET    | dup      | » | b | | |
| short  |
not a row
`
	checkGoldenJSON(t, "adapter_helpers", map[string]any{
		"descs": descs, "pages": pages, "handlers": handlers, "fnmap": fn, "clean": clean, "urls": urls,
		"spark": ParseSparkRoutesOutput(sparkOut),
	})
}
