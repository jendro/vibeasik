"""Aplikasi contoh yang sengaja error (WSGI nyata lewat wsgiref + SDK traffic). Env: VIBEASIK_SDK_DIR."""
import os
import sys
import threading
import urllib.error
import urllib.request
from wsgiref.simple_server import WSGIRequestHandler, make_server

sys.path.insert(0, os.path.join(os.environ["VIBEASIK_SDK_DIR"], "python"))
import vibeasik_traffic as vt


class DbError(Exception):
    pass


def load_user(uid, token):
    _ = token
    raise DbError("db down id=%d" % uid)


def app(environ, start_response):
    if environ["PATH_INFO"].startswith("/boom"):
        load_user(int(environ["PATH_INFO"].rsplit("/", 1)[1]), environ.get("HTTP_AUTHORIZATION"))
    start_response("200 OK", [("Content-Length", "2")])
    return [b"ok"]


class Quiet(WSGIRequestHandler):
    def log_message(self, *a):
        pass


def main():
    server = make_server("127.0.0.1", 0, vt.VibeasikTrafficWSGI(app), handler_class=Quiet)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    base = "http://127.0.0.1:%d" % server.server_port
    for p in ["/boom/11", "/boom/22", "/ok", "/boom/33", "/ok"]:
        req = urllib.request.Request(base + p + "?token=SECRETQ", data=b'{"password":"SECRETB"}',
                                     headers={"Authorization": "Bearer SECRETH", "Cookie": "sid=SECRETC"})
        try:
            urllib.request.urlopen(req, timeout=5).close()
        except urllib.error.HTTPError:
            pass
    vt.flush()
    print("selesai")


main()
