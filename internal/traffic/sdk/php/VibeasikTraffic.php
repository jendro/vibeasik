<?php
// vibeasik Traffic Analyzer — middleware Laravel (app/Http/Middleware/VibeasikTraffic.php).
// PHP-FPM tidak punya proses panjang, jadi event ditulis ke berkas antrean lokal lalu dikirim per batch
// paling cepat tiap VIBEASIK_TRAFFIC_FLUSH_SEC detik (default 10), setelah respons terkirim (terminate).
// Env: VIBEASIK_TRAFFIC_URL, VIBEASIK_TRAFFIC_KEY (wajib) · VIBEASIK_TRAFFIC_INSTANCE · VIBEASIK_TRAFFIC_HASH_IP=1
//      VIBEASIK_TRAFFIC_SKIP (default /health). Query string, header lain & body request tidak dikirim.
// Daftarkan: Laravel 11+ bootstrap/app.php → ->withMiddleware(fn ($m) => $m->append(\App\Http\Middleware\VibeasikTraffic::class))
//            Laravel ≤10 app/Http/Kernel.php → $middleware[] = \App\Http\Middleware\VibeasikTraffic::class;

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;

class VibeasikTraffic
{
    private const MAX_QUEUE_BYTES = 8 * 1024 * 1024;

    public function handle(Request $request, Closure $next)
    {
        $request->attributes->set('vibeasik_start', microtime(true));
        return $next($request);
    }

    public function terminate(Request $request, $response): void
    {
        $url = env('VIBEASIK_TRAFFIC_URL');
        $key = env('VIBEASIK_TRAFFIC_KEY');
        $start = $request->attributes->get('vibeasik_start');
        if (!$url || !$key || !$start) {
            return;
        }
        $path = '/' . ltrim($request->path(), '/');
        foreach (array_filter(array_map('trim', explode(',', env('VIBEASIK_TRAFFIC_SKIP', '/health')))) as $skip) {
            if (str_starts_with($path, $skip)) {
                return;
            }
        }
        $ip = $request->header('CF-Connecting-IP') ?: $request->ip();
        if (env('VIBEASIK_TRAFFIC_HASH_IP') == '1' && $ip) {
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

        $queue = sys_get_temp_dir() . '/vibeasik-traffic-' . substr(md5($key), 0, 8) . '.ndjson';
        if (@filesize($queue) < self::MAX_QUEUE_BYTES) {
            @file_put_contents($queue, json_encode($event) . "\n", FILE_APPEND | LOCK_EX);
        }
        $flushSec = max(1, (int) env('VIBEASIK_TRAFFIC_FLUSH_SEC', 10));
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
                CURLOPT_POSTFIELDS => json_encode(['instance' => env('VIBEASIK_TRAFFIC_INSTANCE', gethostname()), 'events' => $chunk]),
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
