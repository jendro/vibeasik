<?php
// vibeasik Traffic Analyzer — middleware Laravel (app/Http/Middleware/VibeasikTraffic.php).
// PHP-FPM tidak punya proses panjang, jadi event ditulis ke berkas antrean lokal lalu dikirim per batch
// paling cepat tiap VIBEASIK_TRAFFIC_FLUSH_SEC detik (default 10), setelah respons terkirim (terminate).
// Env: VIBEASIK_TRAFFIC_URL, VIBEASIK_TRAFFIC_KEY (wajib) · VIBEASIK_TRAFFIC_INSTANCE · VIBEASIK_TRAFFIC_HASH_IP=1
//      VIBEASIK_TRAFFIC_SKIP (default /health) · VIBEASIK_RELEASE (versi/commit rilis, ikut di semua event).
//      Query string, header lain & body request tidak dikirim.
// Error: daftarkan ke report() Laravel agar exception terkirim sebagai event error (tipe, pesan, frame fungsi + berkas:baris;
//      argumen fungsi & nilai variabel TIDAK ikut):
//        Laravel 11+ bootstrap/app.php → ->withExceptions(fn ($e) => $e->report(fn (\Throwable $t) => \App\Http\Middleware\VibeasikTraffic::captureError($t)))
//        Laravel ≤10 app/Exceptions/Handler.php register() → $this->reportable(fn (\Throwable $t) => \App\Http\Middleware\VibeasikTraffic::captureError($t));
// Heartbeat: PHP-FPM tidak punya proses panjang untuk timer 30 detik, jadi heartbeat dikirim oportunistik: paling banyak
//      sekali per 30 detik, saat ada request (termasuk /health). Tanpa request sama sekali tidak ada heartbeat.
// Daftarkan: Laravel 11+ bootstrap/app.php → ->withMiddleware(fn ($m) => $m->append(\App\Http\Middleware\VibeasikTraffic::class))
//            Laravel ≤10 app/Http/Kernel.php → $middleware[] = \App\Http\Middleware\VibeasikTraffic::class;

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;

class VibeasikTraffic
{
    private const MAX_QUEUE_BYTES = 8 * 1024 * 1024;
    private const HEARTBEAT_SEC = 30;

    /** Waktu worker PHP pertama kali memuat kelas ini (pendekatan uptime proses untuk heartbeat). */
    private static ?float $bootedAt = null;

    private static function cfg(string $name, $default = null)
    {
        $v = function_exists('env') ? env($name) : null;
        if ($v === null) {
            $v = getenv($name);
        }
        return ($v === null || $v === false || $v === '') ? $default : $v;
    }

    private static function queuePath(string $key): string
    {
        return sys_get_temp_dir() . '/vibeasik-traffic-' . substr(md5($key), 0, 8) . '.ndjson';
    }

    private static function enqueue(array $event, string $key): void
    {
        $event['release'] = $event['release'] ?? substr((string) self::cfg('VIBEASIK_RELEASE', ''), 0, 64);
        $queue = self::queuePath($key);
        if (@filesize($queue) < self::MAX_QUEUE_BYTES) {
            @file_put_contents($queue, json_encode($event) . "\n", FILE_APPEND | LOCK_EX);
        }
    }

    /**
     * Laporkan exception sebagai event error. Dipanggil dari report() Laravel. Tidak pernah melempar.
     * Hanya frame (kelas::fungsi + berkas:baris) yang dikirim: argumen di getTrace() sengaja diabaikan.
     */
    public static function captureError(\Throwable $e, ?Request $request = null): void
    {
        try {
            $url = self::cfg('VIBEASIK_TRAFFIC_URL');
            $key = self::cfg('VIBEASIK_TRAFFIC_KEY');
            $status = method_exists($e, 'getStatusCode') ? (int) $e->getStatusCode() : 500;
            if (!$url || !$key || $status < 500) {
                return;
            }
            $request = $request ?? (function_exists('request') ? request() : null);
            $route = $request && method_exists($request, 'route') ? $request->route() : null;
            self::enqueue([
                'kind' => 'error', 'ts' => (int) round(microtime(true) * 1000),
                'method' => $request ? $request->method() : '',
                'route' => $route && is_object($route) ? '/' . ltrim($route->uri(), '/') : '',
                'path' => $request ? '/' . ltrim($request->path(), '/') : '',
                'status' => $status, 'err_type' => substr(get_class($e), 0, 128), 'err_msg' => substr($e->getMessage(), 0, 500),
                'stack' => self::frames($e), 'side' => 'server',
            ], (string) $key);
        } catch (\Throwable $ignored) {
            // SDK tidak boleh menjatuhkan aplikasi inang
        }
    }

    private static function frames(\Throwable $e): string
    {
        $trace = $e->getTrace();
        $first = $trace[0] ?? [];
        $lines = [self::frameLine($first['class'] ?? '', $first['type'] ?? '', $first['function'] ?? '{main}', $e->getFile(), $e->getLine())];
        foreach ($trace as $t) {
            $lines[] = self::frameLine($t['class'] ?? '', $t['type'] ?? '', $t['function'] ?? '', $t['file'] ?? '', $t['line'] ?? 0);
        }
        return substr(implode("\n", array_slice($lines, 0, 40)), 0, 8000);
    }

    private static function frameLine(string $class, string $type, string $fn, string $file, int $line): string
    {
        return $class . $type . $fn . ' (' . $file . ':' . $line . ')';
    }

    private static function heartbeatDue(string $queue): bool
    {
        $stamp = $queue . '.hb';
        if (time() - (int) @filemtime($stamp) < self::HEARTBEAT_SEC) {
            return false;
        }
        @touch($stamp);
        return true;
    }

    public function handle(Request $request, Closure $next)
    {
        $request->attributes->set('vibeasik_start', microtime(true));
        return $next($request);
    }

    public function terminate(Request $request, $response): void
    {
        $url = self::cfg('VIBEASIK_TRAFFIC_URL');
        $key = self::cfg('VIBEASIK_TRAFFIC_KEY');
        $start = $request->attributes->get('vibeasik_start');
        if (!$url || !$key || !$start) {
            return;
        }
        self::$bootedAt = self::$bootedAt ?? microtime(true);
        if (self::heartbeatDue(self::queuePath((string) $key))) {
            self::enqueue([
                'kind' => 'heartbeat', 'ts' => (int) round(microtime(true) * 1000), 'instance' => self::cfg('VIBEASIK_TRAFFIC_INSTANCE', gethostname()),
                'mem_bytes' => memory_get_usage(true), 'uptime_s' => (int) (microtime(true) - self::$bootedAt),
            ], (string) $key);
        }
        $path = '/' . ltrim($request->path(), '/');
        foreach (array_filter(array_map('trim', explode(',', self::cfg('VIBEASIK_TRAFFIC_SKIP', '/health')))) as $skip) {
            if (str_starts_with($path, $skip)) {
                return;
            }
        }
        $ip = $request->header('CF-Connecting-IP') ?: $request->ip();
        if (self::cfg('VIBEASIK_TRAFFIC_HASH_IP') == '1' && $ip) {
            $ip = 'h:' . substr(hash('sha256', $ip . $key), 0, 16);
        }
        $route = $request->route();
        $event = [
            'ts' => (int) round($start * 1000),
            'method' => $request->method(),
            'route' => $route ? '/' . ltrim($route->uri(), '/') : '',
            'path' => $path,
            'status' => method_exists($response, 'getStatusCode') ? $response->getStatusCode() : 200,
            'dur_ms' => (microtime(true) - $start) * 1000,
            'bytes_in' => (int) $request->header('Content-Length', 0),
            'bytes_out' => method_exists($response, 'getContent') ? strlen((string) $response->getContent()) : 0,
            'ip' => $ip,
            'country' => $request->header('CF-IPCountry', ''),
            'ua' => (string) $request->userAgent(),
            'referer' => (string) parse_url((string) $request->header('referer', ''), PHP_URL_HOST),
        ];

        self::enqueue($event, (string) $key);
        $queue = self::queuePath((string) $key);
        $flushSec = max(1, (int) self::cfg('VIBEASIK_TRAFFIC_FLUSH_SEC', 10));
        $stamp = $queue . '.sent';
        if (time() - (int) @filemtime($stamp) < $flushSec) {
            return;
        }
        $this->flush($queue, $stamp, $url, $key);
    }

    private function flush(string $queue, string $stamp, string $url, string $key): void
    {
        $lock = @fopen($queue . '.lock', 'c');
        if (!$lock || !flock($lock, LOCK_EX | LOCK_NB)) {
            return; // proses lain sedang mengirim
        }
        @touch($stamp);
        $sending = $queue . '.sending';
        if (!file_exists($sending)) {
            @rename($queue, $sending);
        }
        $lines = @file($sending, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES) ?: [];
        $events = array_values(array_filter(array_map(fn ($l) => json_decode($l, true), $lines)));
        $ok = true;
        foreach (array_chunk($events, 5000) as $chunk) {
            $ch = curl_init($url);
            curl_setopt_array($ch, [
                CURLOPT_POST => true,
                CURLOPT_RETURNTRANSFER => true,
                CURLOPT_TIMEOUT => 5,
                CURLOPT_HTTPHEADER => ['Content-Type: application/json', 'X-Vibeasik-Key: ' . $key],
                CURLOPT_POSTFIELDS => json_encode(['instance' => self::cfg('VIBEASIK_TRAFFIC_INSTANCE', gethostname()), 'events' => $chunk]),
            ]);
            curl_exec($ch);
            $code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
            curl_close($ch);
            if ($code === 0 || $code >= 500 || $code === 429) {
                $ok = false; // vibeasik tidak terjangkau: berkas .sending dicoba lagi pada flush berikutnya
                break;
            }
        }
        if ($ok) {
            @unlink($sending);
        }
        flock($lock, LOCK_UN);
        fclose($lock);
    }
}
