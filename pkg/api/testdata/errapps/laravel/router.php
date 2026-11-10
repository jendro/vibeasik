<?php
// Aplikasi contoh yang sengaja error untuk middleware Laravel (php -S, satu proses PHP per request seperti PHP-FPM).
// Laravel tidak dipasang: kelas Illuminate\Http\Request/Response minimal ditiru; alur handle -> report -> terminate sama.
namespace Illuminate\Http {
    class Attrs { private array $d = []; public function set($k, $v) { $this->d[$k] = $v; } public function get($k) { return $this->d[$k] ?? null; } }
    class Route { public function __construct(private string $u) {} public function uri() { return $this->u; } }
    class Request {
        public Attrs $attributes;
        public function __construct(private array $s) { $this->attributes = new Attrs(); }
        public function path() { return trim(parse_url($this->s['REQUEST_URI'], PHP_URL_PATH), '/'); }
        public function method() { return $this->s['REQUEST_METHOD']; }
        public function header($n, $d = null) { return $this->s['HTTP_' . strtoupper(str_replace('-', '_', $n))] ?? $d; }
        public function ip() { return '203.0.113.9'; }
        public function userAgent() { return $this->header('User-Agent', ''); }
        public function route() { return str_starts_with($this->path(), 'boom') ? new Route('boom/{id}') : new Route($this->path()); }
    }
    class Response {
        public function __construct(private string $c, private int $st) {}
        public function getStatusCode() { return $this->st; }
        public function getContent() { return $this->c; }
    }
}

namespace {
    use App\Http\Middleware\VibeasikTraffic;
    use Illuminate\Http\Request;
    use Illuminate\Http\Response;

    require getenv('VIBEASIK_SDK_DIR') . '/php/VibeasikTraffic.php';

    class DbException extends RuntimeException {}

    function loadUser(int $id, string $token)
    {
        throw new DbException('db down id=' . $id);
    }

    $request = new Request($_SERVER);
    $mw = new VibeasikTraffic();
    try {
        $response = $mw->handle($request, function (Request $r) {
            if (str_starts_with($r->path(), 'boom')) {
                loadUser((int) substr($r->path(), 5), (string) $r->header('Authorization'));
            }
            return new Response('ok', 200);
        });
    } catch (\Throwable $e) {
        VibeasikTraffic::captureError($e, $request); // = report() Laravel
        $response = new Response('error', 500);
    }
    http_response_code($response->getStatusCode());
    echo $response->getContent();
    $mw->terminate($request, $response);
}
