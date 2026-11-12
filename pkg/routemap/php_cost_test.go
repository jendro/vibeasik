package routemap

import (
	"strings"
	"testing"
)

var laravelFixture = map[string]string{
	"composer.json": `{"name": "demo/shop", "require": {"laravel/framework": "^11.0"}}`,
	"artisan":       "#!/usr/bin/env php\n<?php\n",
	"routes/web.php": `<?php

use App\Http\Controllers\UserController;
use App\Http\Controllers\PhotoController;
use Illuminate\Support\Facades\Route;

Route::get('/one', [UserController::class, 'one']);

Route::middleware(['auth'])->prefix('admin')->group(function () {
    Route::get('/list', [UserController::class, 'list']);
    Route::get('pairs', 'UserController@pairs')->name('pairs');
});

Route::controller(UserController::class)->group(function () {
    Route::get('/nplus', 'nplus');
    Route::post('/sorted', 'sorted');
});

Route::get('/fixed', [UserController::class, 'fixed']);
Route::get('/each', [UserController::class, 'each']);
Route::get('/users/{user}', [UserController::class, 'show']);
Route::get('/closure', function () {
    return 'ok';
});
Route::resource('photos', PhotoController::class)->only(['index', 'show']);
require __DIR__.'/auth.php';
`,
	"routes/auth.php": `<?php
use Illuminate\Support\Facades\Route;
Route::get('/login', fn () => view('login'));
`,
	"routes/api.php": `<?php
use Illuminate\Support\Facades\Route;
Route::get('/ping', fn () => ['ok' => true]);
`,
	"app/Http/Kernel.php": `<?php
namespace App\Http;
class Kernel extends HttpKernel {
    protected $middlewareAliases = [
        'auth' => \App\Http\Middleware\Authenticate::class,
    ];
}
`,
	"app/Http/Middleware/Authenticate.php": `<?php
namespace App\Http\Middleware;
class Authenticate {
    public function handle($request, $next) {
        if (!$request->user()) {
            abort(401);
        }
        return $next($request);
    }
}
`,
	"app/Models/User.php": `<?php
namespace App\Models;
use Illuminate\Database\Eloquent\Model;
class User extends Model {
    public function posts() {
        return $this->hasMany(Post::class);
    }
}
`,
	"app/Models/Order.php":  "<?php\nnamespace App\\Models;\nuse Illuminate\\Database\\Eloquent\\Model;\nclass Order extends Model {}\n",
	"app/Models/Photo.php":  "<?php\nnamespace App\\Models;\nuse Illuminate\\Database\\Eloquent\\Model;\nclass Photo extends Model {}\n",
	"resources/views/x.php": "<?php echo 'view';",
	"app/Http/Controllers/PhotoController.php": `<?php
namespace App\Http\Controllers;
use App\Models\Photo;
class PhotoController extends Controller {
    public function index() {
        return Photo::latest()->paginate(20);
    }
    public function show(Photo $photo) {
        return view('photo', compact('photo'));
    }
}
`,
	"app/Http/Controllers/UserController.php": `<?php
namespace App\Http\Controllers;

use App\Models\User;
use App\Models\Order;

class UserController extends Controller
{
    public function one()
    {
        $u = User::find(1);
        return response()->json($u);
    }

    public function list()
    {
        $us = User::all();
        $total = 0;
        foreach ($us as $u) {
            $total += $u->id;
        }
        return $total;
    }

    public function pairs()
    {
        $us = User::all();
        $hits = 0;
        foreach ($us as $a) {
            foreach ($us as $b) {
                if ($a->id === $b->id) {
                    $hits++;
                }
            }
        }
        return $hits;
    }

    public function nplus()
    {
        $us = User::all();
        foreach ($us as $u) {
            $u->orders = Order::where('user_id', $u->id)->get();
        }
        return $us;
    }

    public function sorted()
    {
        $us = User::all()->sortBy(fn ($u) => $u->id);
        return $us;
    }

    public function fixed()
    {
        $sum = 0;
        for ($i = 0; $i < 3; $i++) {
            $sum += $this->helper($i);
        }
        return $sum;
    }

    private function helper($i)
    {
        return $i * 2;
    }

    public function each()
    {
        User::all()->each(function (User $u) {
            $u->posts()->count();
        });
    }

    public function show(User $user)
    {
        return view('user', compact('user'));
    }
}
`,
}

func phpRouteCost(t *testing.T, rep *CostReport, method, path string) RouteCost {
	t.Helper()
	for _, r := range rep.Routes {
		if r.Path == path && r.Method == method {
			return r
		}
	}
	var got []string
	for _, r := range rep.Routes {
		got = append(got, r.Method+" "+r.Path)
	}
	t.Fatalf("rute %s %s tidak ditemukan; ada %v", method, path, got)
	return RouteCost{}
}

func TestLaravelRouteCosts(t *testing.T) {
	rep, mods := PHPCosts(writeTree(t, laravelFixture))
	if len(mods) != 1 || mods[0].Framework != "laravel" || mods[0].Name != "demo/shop" {
		t.Fatalf("modul: %+v", mods)
	}
	cases := []struct {
		method, path, formula, bigO, queries string
		nPlusOne                             bool
	}{
		{"GET", "/one", "2", "O(1)", "1", false},
		// 2n + 4 dari handler + 3 tick middleware auth lokal (Authenticate::handle)
		{"GET", "/admin/list", "2n + 7", "O(n)", "1", false},
		{"GET", "/admin/pairs", "3n² + 2n + 7", "O(n²)", "1", false},
		{"GET", "/nplus", "2n + 3", "O(n)", "n + 1", true},
		{"POST", "/sorted", "2n log n + 2", "O(n log n)", "1", false},
		{"GET", "/fixed", "16", "O(1)", "0", false},
		{"GET", "/each", "2n + 1", "O(n)", "n + 1", true}, // relasi lokal User::posts() ikut 1 tick
		{"GET", "/users/{user}", "1", "O(1)", "1", false}, // route model binding = 1 query
		{"GET", "/closure", "1", "O(1)", "0", false},
		{"GET", "/photos", "1", "O(1)", "1", false},
		{"GET", "/photos/{photo}", "1", "O(1)", "1", false},
		{"GET", "/api/ping", "1", "O(1)", "0", false},
		{"GET", "/login", "1", "O(1)", "0", false},
	}
	for _, tc := range cases {
		got := phpRouteCost(t, rep, tc.method, tc.path)
		if got.Formula != tc.formula || got.BigO != tc.bigO || got.QueryFormula != tc.queries || got.NPlusOne != tc.nPlusOne {
			t.Errorf("%s %s: T(n)=%q %s Q(n)=%q n+1=%v; mau T(n)=%q %s Q(n)=%q n+1=%v", tc.method, tc.path,
				got.Formula, got.BigO, got.QueryFormula, got.NPlusOne, tc.formula, tc.bigO, tc.queries, tc.nPlusOne)
		}
		if got.Lang != "php" || got.Framework != "Laravel" {
			t.Errorf("%s: lang=%q framework=%q", tc.path, got.Lang, got.Framework)
		}
	}
	if len(rep.Routes) != len(cases) {
		t.Errorf("%d rute, mau %d (resource only index/show)", len(rep.Routes), len(cases))
	}
	list := phpRouteCost(t, rep, "GET", "/admin/list")
	if !list.Auth || list.Middlewares != 1 || list.Handler != "UserController::list" || list.HandlerDecl == "" {
		t.Errorf("/admin/list: auth=%v mws=%d handler=%q decl=%q", list.Auth, list.Middlewares, list.Handler, list.HandlerDecl)
	}
	if phpRouteCost(t, rep, "GET", "/one").Auth {
		t.Error("/one tidak memakai middleware auth")
	}
	if rep.Routes[0].Path != "/admin/pairs" {
		t.Errorf("rute termahal = %s, mau /admin/pairs", rep.Routes[0].Path)
	}
	var helper *FuncCost
	for i := range rep.Funcs {
		if rep.Funcs[i].Name == "UserController::helper" {
			helper = &rep.Funcs[i]
		}
	}
	if helper == nil || helper.Routes != 1 || helper.Formula != "1" || helper.Lang != "php" {
		t.Errorf("fungsi helper: %+v", helper)
	}
}

func TestPlainPHPPages(t *testing.T) {
	root := writeTree(t, map[string]string{
		"db.php": "<?php\n$pdo = new PDO('sqlite::memory:');\n",
		"index.php": `<?php
require 'db.php';
$rows = $pdo->query('SELECT * FROM users')->fetchAll();
foreach ($rows as $r) {
    $stmt = $pdo->prepare('SELECT * FROM orders WHERE user_id = ?');
    $stmt->execute([$r['id']]);
}
?>
<html><body><?= count($rows) ?></body></html>
`,
		"about.php": `<?php
function page_title(): string { return 'About'; }
$items = ['a', 'b', 'c'];
foreach ($items as $it) {
    echo $it;
}
switch ($_GET['tab'] ?? '') {
    case 'x':
        echo page_title();
        break;
    default:
        echo 'y';
}
while (true) {
    if (rand(0, 1)) { break; }
}
`,
		"lib/helpers.php": "<?php\nfunction slug($s) { return strtolower($s); }\n",
	})
	rep, mods := PHPCosts(root)
	if len(mods) != 1 || mods[0].Framework != "php" {
		t.Fatalf("modul: %+v", mods)
	}
	idx := phpRouteCost(t, rep, "ANY", "/")
	// require (1 + 1 db.php) + query (1) + foreach (3n + 1) + echo (1)
	if idx.Formula != "3n + 5" || idx.QueryFormula != "n + 1" || !idx.NPlusOne {
		t.Errorf("index.php: T(n)=%q Q(n)=%q n+1=%v", idx.Formula, idx.QueryFormula, idx.NPlusOne)
	}
	about := phpRouteCost(t, rep, "ANY", "/about.php")
	// fungsi 0 + assign 1 + foreach tetap 3×2+1 = 7 + switch 1 + case (echo 1 + page_title 1 + break 1) + while tanpa batas
	if !about.Unbounded || about.BigO != "O(n)" {
		t.Errorf("about.php: %s %q unbounded=%v", about.BigO, about.Formula, about.Unbounded)
	}
	for _, r := range rep.Routes {
		if r.Path == "/db.php" || r.Path == "/lib/helpers.php" {
			t.Errorf("%s di-include / hanya deklarasi, bukan halaman", r.Path)
		}
	}
}

func TestCodeIgniterRoutes(t *testing.T) {
	root := writeTree(t, map[string]string{
		"composer.json": `{"require": {"codeigniter4/framework": "^4.4"}}`,
		"app/Config/Routes.php": `<?php
use CodeIgniter\Router\RouteCollection;
/** @var RouteCollection $routes */
$routes->get('/', 'Home::index');
$routes->group('admin', ['filter' => 'auth'], static function ($routes) {
    $routes->get('users', 'Admin\Users::list');
});
$routes->get('stats', 'Home::stats');
`,
		"app/Controllers/Home.php": `<?php
namespace App\Controllers;
class Home extends BaseController {
    public function index(): string { return view('welcome_message'); }
    public function stats() {
        $db = \Config\Database::connect();
        $builder = $db->table('atlet')->where('aktif', 1);
        if ($this->request->getGet('q')) {
            $builder->like('nama', $this->request->getGet('q'));
        }
        return $this->response->setJSON($builder->get()->getResultArray());
    }
}
`,
		"app/Controllers/Admin/Users.php": `<?php
namespace App\Controllers\Admin;
use App\Models\UserModel;
class Users {
    public function list() {
        $m = new UserModel();
        $users = $m->findAll();
        foreach ($users as $u) {
            $u->roles = $this->db->table('roles')->where('user_id', $u->id)->get()->getResult();
        }
        return $this->response->setJSON($users);
    }
}
`,
		"app/Models/UserModel.php": "<?php\nnamespace App\\Models;\nuse CodeIgniter\\Model;\nclass UserModel extends Model {}\n",
	})
	rep, mods := PHPCosts(root)
	if len(mods) != 1 || mods[0].Framework != "codeigniter4" {
		t.Fatalf("modul: %+v", mods)
	}
	if home := phpRouteCost(t, rep, "GET", "/"); home.Formula != "1" || home.Handler != "Home::index" {
		t.Errorf("/: %q %q", home.Formula, home.Handler)
	}
	// builder disimpan di variabel lalu dieksekusi belakangan tetap 1 query
	if st := phpRouteCost(t, rep, "GET", "/stats"); st.QueryFormula != "1" {
		t.Errorf("/stats: Q(n)=%q, mau 1", st.QueryFormula)
	}
	users := phpRouteCost(t, rep, "GET", "/admin/users")
	if !users.Auth || users.QueryFormula != "n + 1" || users.Framework != "CodeIgniter 4" {
		t.Errorf("/admin/users: auth=%v Q(n)=%q fw=%q", users.Auth, users.QueryFormula, users.Framework)
	}
}

func TestMergeCostsSortsAcrossLanguages(t *testing.T) {
	goRep := Costs(Analyze(writeTree(t, map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": costFixture})))
	phpRep, _ := PHPCosts(writeTree(t, laravelFixture))
	all := MergeCosts(goRep, phpRep)
	if len(all.Routes) != len(goRep.Routes)+len(phpRep.Routes) {
		t.Fatalf("gabungan %d rute", len(all.Routes))
	}
	for i := 1; i < len(all.Routes); i++ {
		if all.Routes[i-1].rank < all.Routes[i].rank {
			t.Fatalf("urutan gabungan salah di %d", i)
		}
	}
	if goRep.Routes[0].Lang != "go" || goRep.Funcs[0].Lang != "go" {
		t.Error("rute/fungsi Go harus berlabel lang=go")
	}
}

func TestPHPHotspotsPointAtSource(t *testing.T) {
	hots := PHPHotspots(writeTree(t, laravelFixture))
	find := func(kind, name string) *Hotspot {
		for i := range hots {
			if hots[i].Kind == kind && strings.HasSuffix(hots[i].Func.Name, name) {
				return &hots[i]
			}
		}
		return nil
	}
	n := find("nplus1", "nplus")
	if n == nil || n.Via != "" || len(n.AllRoutes) != 1 || n.AllRoutes[0] != "GET /nplus" {
		t.Fatalf("hotspot N+1 nplus(): %+v (semua: %+v)", n, hots)
	}
	if !strings.HasSuffix(n.Func.File, ".php") || n.Func.Line == 0 {
		t.Errorf("lokasi hotspot: %s:%d", n.Func.File, n.Func.Line)
	}
	if g := find("growth", "pairs"); g == nil || g.Func.BigO != "O(n²)" {
		t.Errorf("hotspot O(n²) pairs(): %+v", g)
	}
	if find("growth", "list") != nil || find("nplus1", "list") != nil {
		t.Error("list() O(n) tanpa N+1 bukan sumber biaya")
	}
}

func TestPHPHotspotsOnPlainPage(t *testing.T) {
	hots := PHPHotspots(writeTree(t, map[string]string{
		"index.php": "<?php\n$pdo = new PDO('x');\n$rows = $pdo->query('SELECT 1')->fetchAll();\nforeach ($rows as $r) {\n    $pdo->query('SELECT 2');\n}\n",
	}))
	if len(hots) != 1 || hots[0].Kind != "nplus1" || hots[0].Func.File != "index.php" {
		t.Fatalf("halaman PHP biasa: %+v", hots)
	}
}
