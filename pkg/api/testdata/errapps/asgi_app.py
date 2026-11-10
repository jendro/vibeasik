"""Aplikasi contoh yang sengaja error (ASGI mentah + SDK traffic). Env: VIBEASIK_SDK_DIR.
ASGI digerakkan langsung (uvicorn/starlette tidak dipasang): app dipanggil seperti server ASGI memanggilnya."""
import asyncio
import os
import sys

sys.path.insert(0, os.path.join(os.environ["VIBEASIK_SDK_DIR"], "python"))
import vibeasik_traffic as vt


class DbError(Exception):
    pass


def load_user(uid, token):
    _ = token  # nilai argumen tidak boleh muncul di stack
    raise DbError("db down id=%d" % uid)


async def app(scope, receive, send):
    if scope["path"].startswith("/boom"):
        load_user(int(scope["path"].rsplit("/", 1)[1]), dict(scope["headers"]).get(b"authorization"))
    await send({"type": "http.response.start", "status": 200, "headers": []})
    await send({"type": "http.response.body", "body": b"ok"})


async def call(wrapped, path):
    scope = {"type": "http", "method": "GET", "path": path, "query_string": b"token=SECRETQ",
             "headers": [(b"authorization", b"Bearer SECRETH"), (b"cookie", b"sid=SECRETC"), (b"user-agent", b"harness")],
             "client": ("203.0.113.9", 1234)}

    async def receive():
        return {"type": "http.request", "body": b'{"password":"SECRETB"}'}

    async def send(_msg):
        return None

    try:
        await wrapped(scope, receive, send)
    except DbError:
        pass


async def main():
    wrapped = vt.VibeasikTrafficASGI(app)
    for p in ["/boom/11", "/boom/22", "/ok", "/boom/33", "/ok"]:
        await call(wrapped, p)
    vt.flush()
    print("selesai")


asyncio.run(main())
