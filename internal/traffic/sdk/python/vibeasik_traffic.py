"""vibeasik Traffic Analyzer — collector Python tanpa dependensi (FastAPI/Starlette via ASGI, Flask/Django via WSGI).

Mengirim rute, status, lama proses, IP, negara, UA & host perujuk. Query string, header lain & body tidak dikirim.
Env: VIBEASIK_TRAFFIC_URL, VIBEASIK_TRAFFIC_KEY (wajib) · VIBEASIK_TRAFFIC_FLUSH_SEC (default 10; 1-2 = realtime)
     VIBEASIK_TRAFFIC_INSTANCE · VIBEASIK_TRAFFIC_HASH_IP=1 · VIBEASIK_TRAFFIC_SKIP (default /health)

FastAPI/Starlette:  app.add_middleware(VibeasikTrafficASGI)
Flask:              app.wsgi_app = VibeasikTrafficWSGI(app.wsgi_app)
Django (wsgi.py):   application = VibeasikTrafficWSGI(get_wsgi_application())
"""
import atexit
import hashlib
import json
import os
import socket
import threading
import time
import urllib.error
import urllib.request
from urllib.parse import urlparse

ENDPOINT = os.environ.get("VIBEASIK_TRAFFIC_URL", "")
KEY = os.environ.get("VIBEASIK_TRAFFIC_KEY", "")
FLUSH_SEC = max(1.0, float(os.environ.get("VIBEASIK_TRAFFIC_FLUSH_SEC") or 10))
INSTANCE = os.environ.get("VIBEASIK_TRAFFIC_INSTANCE") or socket.gethostname()
HASH_IP = os.environ.get("VIBEASIK_TRAFFIC_HASH_IP") == "1"
SKIP = [p.strip() for p in (os.environ.get("VIBEASIK_TRAFFIC_SKIP") or "/health").split(",") if p.strip()]
MAX_BUFFER = 20000
ENABLED = bool(ENDPOINT and KEY)

_lock = threading.Lock()
_buffer = []
_started = False


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
    with _lock:
        if len(_buffer) < MAX_BUFFER:
            _buffer.append(ev)
    _start()


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
