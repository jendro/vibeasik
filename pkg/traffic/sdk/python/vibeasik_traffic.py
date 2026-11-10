"""vibeasik Traffic Analyzer — collector Python tanpa dependensi (FastAPI/Starlette via ASGI, Flask/Django via WSGI).

Mengirim rute, status, lama proses, IP, negara, UA & host perujuk. Query string, header lain & body tidak dikirim.
Env: VIBEASIK_TRAFFIC_URL, VIBEASIK_TRAFFIC_KEY (wajib) · VIBEASIK_TRAFFIC_FLUSH_SEC (default 10; 1-2 = realtime)
     VIBEASIK_TRAFFIC_INSTANCE · VIBEASIK_TRAFFIC_HASH_IP=1 · VIBEASIK_TRAFFIC_SKIP (default /health)
     VIBEASIK_RELEASE (versi/commit rilis, ikut di semua event)

Exception yang lolos dari aplikasi dilaporkan sebagai event error (tipe, pesan, frame fungsi + berkas:baris; tanpa nilai
variabel/body/header/query) lalu di-raise ulang. Heartbeat tiap 30 detik (rilis, memori, uptime).
Flask/Django menangkap exception sendiri dan membalas 500: panggil capture_exception(exc) dari handler/sinyal errornya
(Flask: got_request_exception, Django: got_request_exception).

FastAPI/Starlette:  app.add_middleware(VibeasikTrafficASGI)
Flask:              app.wsgi_app = VibeasikTrafficWSGI(app.wsgi_app)
Django (wsgi.py):   application = VibeasikTrafficWSGI(get_wsgi_application())
"""
import atexit
import hashlib
import json
import os
import socket
import sys
import threading
import time
import traceback
import urllib.error
import urllib.request
from urllib.parse import urlparse

ENDPOINT = os.environ.get("VIBEASIK_TRAFFIC_URL", "")
KEY = os.environ.get("VIBEASIK_TRAFFIC_KEY", "")
FLUSH_SEC = max(1.0, float(os.environ.get("VIBEASIK_TRAFFIC_FLUSH_SEC") or 10))
INSTANCE = os.environ.get("VIBEASIK_TRAFFIC_INSTANCE") or socket.gethostname()
HASH_IP = os.environ.get("VIBEASIK_TRAFFIC_HASH_IP") == "1"
SKIP = [p.strip() for p in (os.environ.get("VIBEASIK_TRAFFIC_SKIP") or "/health").split(",") if p.strip()]
RELEASE = (os.environ.get("VIBEASIK_RELEASE") or "")[:64]
HEARTBEAT_SEC = 30
MAX_BUFFER = 20000
_BOOT = time.time()
ENABLED = bool(ENDPOINT and KEY)

_lock = threading.Lock()
_buffer = []
_started = False
_hb_started = False


def _start():
    global _started
    if _started:
        return
    _started = True

    def loop():
        while True:
            time.sleep(FLUSH_SEC)
            flush()

    threading.Thread(target=loop, name="vibeasik-traffic", daemon=True).start()
    atexit.register(flush)


def record(ev):
    if not ENABLED or any(ev["path"].startswith(p) for p in SKIP):
        return
    if HASH_IP and ev.get("ip"):
        ev["ip"] = "h:" + hashlib.sha256((ev["ip"] + KEY).encode()).hexdigest()[:16]
    _push(ev)


def _push(ev):
    """Masukkan event ke buffer tanpa filter path (error & heartbeat juga lewat sini)."""
    if RELEASE and "release" not in ev:
        ev["release"] = RELEASE
    with _lock:
        if len(_buffer) < MAX_BUFFER:
            _buffer.append(ev)
    _start()


def _rss_bytes():
    try:
        with open("/proc/self/statm") as f:
            return int(f.read().split()[1]) * os.sysconf("SC_PAGE_SIZE")
    except (OSError, ValueError, IndexError):
        pass
    try:
        import resource
        peak = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
        return peak if sys.platform == "darwin" else peak * 1024
    except ImportError:
        return 0


def _heartbeat():
    _push({"kind": "heartbeat", "ts": int(time.time() * 1000), "instance": INSTANCE, "path": "",
           "mem_bytes": _rss_bytes(), "uptime_s": int(time.time() - _BOOT)})


def _heartbeat_loop():
    while True:
        _heartbeat()
        time.sleep(HEARTBEAT_SEC)


def _start_heartbeat():
    global _hb_started
    if _hb_started or not ENABLED:
        return
    _hb_started = True
    threading.Thread(target=_heartbeat_loop, name="vibeasik-heartbeat", daemon=True).start()


def _frames(exc):
    """Frame dari yang terdalam: 'fungsi (berkas:baris)'. Baris kode sumber & nilai variabel tidak ikut."""
    out = []
    for fr in reversed(traceback.extract_tb(exc.__traceback__)):
        if os.path.basename(fr.filename).startswith("vibeasik_traffic"):
            continue
        out.append("%s (%s:%d)" % (fr.name, fr.filename, fr.lineno))
    return "\n".join(out[:40])[:8000]


def capture_exception(exc, method="", route="", path="", status=500):
    """Catat exception sebagai event 'error'. Tidak pernah melempar."""
    if not ENABLED:
        return
    try:
        t = type(exc)
        name = t.__name__ if t.__module__ == "builtins" else t.__module__ + "." + t.__qualname__
        _push({"kind": "error", "ts": int(time.time() * 1000), "method": method, "route": route, "path": path or "",
               "status": status, "err_type": name[:128], "err_msg": str(exc)[:500], "stack": _frames(exc), "side": "server"})
    except Exception:  # SDK tidak boleh menjatuhkan aplikasi inang
        pass


def flush():
    global _buffer
    with _lock:
        batch, _buffer = _buffer, []
    while batch:
        chunk = batch[:5000]
        body = json.dumps({"instance": INSTANCE, "events": chunk}).encode()
        req = urllib.request.Request(ENDPOINT, data=body, method="POST",
                                     headers={"Content-Type": "application/json", "X-Vibeasik-Key": KEY})
        try:
            urllib.request.urlopen(req, timeout=10).close()
        except urllib.error.HTTPError as e:
            if e.code >= 500 or e.code == 429:
                _requeue(batch)
                return
        except Exception:
            _requeue(batch)  # vibeasik sedang tidak terjangkau: coba lagi pada flush berikutnya
            return
        batch = batch[5000:]


def _requeue(batch):
    global _buffer
    with _lock:
        _buffer = (batch + _buffer)[-MAX_BUFFER:]


def _referer_host(ref):
    try:
        return urlparse(ref).netloc if ref else ""
    except ValueError:
        return ""


def _client_ip(get, remote):
    for k in ("cf-connecting-ip", "true-client-ip", "x-real-ip"):
        v = get(k)
        if v:
            return v.strip()
    xff = get("x-forwarded-for")
    if xff:
        return xff.split(",")[0].strip()
    return remote or ""


class VibeasikTrafficASGI:
    def __init__(self, app):
        self.app = app
        _start_heartbeat()

    async def __call__(self, scope, receive, send):
        if scope["type"] != "http" or not ENABLED:
            return await self.app(scope, receive, send)
        start = time.perf_counter()
        ts = int(time.time() * 1000)
        state = {"status": 500, "out": 0}

        async def send_wrapper(message):
            if message["type"] == "http.response.start":
                state["status"] = message["status"]
            elif message["type"] == "http.response.body":
                state["out"] += len(message.get("body", b""))
            await send(message)

        try:
            await self.app(scope, receive, send_wrapper)
        except Exception as exc:
            capture_exception(exc, scope.get("method", "GET"), getattr(scope.get("route"), "path", "") or "", scope.get("path", "/"))
            raise
        finally:
            headers = {k.decode("latin-1").lower(): v.decode("latin-1") for k, v in scope.get("headers", [])}
            route = scope.get("route")
            client = scope.get("client") or ("", 0)
            record({
                "ts": ts, "method": scope.get("method", "GET"),
                "route": getattr(route, "path", "") or "", "path": scope.get("path", "/"),
                "status": state["status"], "dur_ms": (time.perf_counter() - start) * 1000,
                "bytes_in": int(headers.get("content-length") or 0), "bytes_out": state["out"],
                "ip": _client_ip(headers.get, client[0]),
                "country": headers.get("cf-ipcountry") or headers.get("x-country-code") or "",
                "ua": headers.get("user-agent", ""), "referer": _referer_host(headers.get("referer", "")),
            })


class VibeasikTrafficWSGI:
    def __init__(self, app):
        self.app = app
        _start_heartbeat()

    def __call__(self, environ, start_response):
        if not ENABLED:
            return self.app(environ, start_response)
        start = time.perf_counter()
        ts = int(time.time() * 1000)
        state = {"status": 500, "out": 0}

        def sr(status, headers, exc_info=None):
            state["status"] = int(status.split(" ", 1)[0])
            for k, v in headers:
                if k.lower() == "content-length":
                    state["out"] = int(v or 0)
            return start_response(status, headers, exc_info)

        try:
            return self.app(environ, sr)
        except Exception as exc:
            capture_exception(exc, environ.get("REQUEST_METHOD", "GET"), "", environ.get("PATH_INFO", "/") or "/")
            raise
        finally:
            get = lambda k: environ.get("HTTP_" + k.upper().replace("-", "_"), "")
            record({
                "ts": ts, "method": environ.get("REQUEST_METHOD", "GET"), "route": "",
                "path": environ.get("PATH_INFO", "/") or "/", "status": state["status"],
                "dur_ms": (time.perf_counter() - start) * 1000,
                "bytes_in": int(environ.get("CONTENT_LENGTH") or 0), "bytes_out": state["out"],
                "ip": _client_ip(get, environ.get("REMOTE_ADDR", "")),
                "country": get("cf-ipcountry") or get("x-country-code"),
                "ua": get("user-agent"), "referer": _referer_host(get("referer")),
            })
