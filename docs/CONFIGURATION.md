# Konfigurasi

Bawaan vibeasik sengaja aman dan tanpa konfigurasi. Halaman ini untuk yang ingin mengubahnya.

## Berkas di komputer Anda

Semua data ada di `~/.vibeasik/`:

| Berkas | Isi |
|---|---|
| `projects.db` | Database SQLite: proyek, komponen, temuan audit, todo, pengetahuan |
| `token` | Token akses lokal (izin `0600`). Jangan dibagikan |
| `account.json` | Status login dan entitlement bertanda tangan |
| `.env` | Opsional: variabel lingkungan di bawah. Dibaca dari folder tempat engine dijalankan, lalu dari `~/.vibeasik/.env` |

Mencadangkan atau memindahkan vibeasik ke komputer lain cukup menyalin folder ini.

## Opsi baris perintah

| Opsi | Perintah | Fungsi |
|---|---|---|
| `--db <berkas>` | semua | Pakai berkas database lain (bawaan `~/.vibeasik/projects.db`) |
| `-p, --port` | `serve` | Port engine (bawaan 8765) |
| `-H, --host` | `serve` | Alamat bind (bawaan `127.0.0.1`; `0.0.0.0` membuka ke LAN, baca [SECURITY.md](../SECURITY.md)) |
| `-o, --open` | `serve` | Buka Studio setelah start |
| `--remote`, `--token`, `--exec-token` | `mcp` | Jembatan ke engine jarak jauh, lihat [MCP.md](MCP.md) |

## Variabel lingkungan

| Variabel | Bawaan | Fungsi |
|---|---|---|
| `VIBEASIK_PORT` | `8765` | Port engine (lebih diutamakan daripada `PORT` karena `PORT` sering dipakai proyek lain) |
| `HOST` | `127.0.0.1` | Alamat bind engine |
| `VIBEASIK_TOKEN` | berkas `token` | Token akses; untuk klien dan engine jarak jauh |
| `VIBEASIK_EXEC_TOKEN` | kosong | Mengizinkan tool eksekusi dari LAN dengan header `X-Vibeasik-Exec-Token` |
| `VIBEASIK_ALLOW_LAN_EXEC` | kosong | `1` mengizinkan eksekusi perintah dari LAN (berisiko) |
| `VIBEASIK_CORS_ORIGINS` | dev Studio saja | Origin tambahan yang boleh memanggil API |
| `VIBEASIK_CLOUD_URL` | `https://enjoy.vibeasik.com` | Server akun dan Pusat Pengetahuan |
| `VIBEASIK_KNOWLEDGE_SYNC` | menyala | `off` mematikan penarikan paket otomatis |
| `VIBEASIK_REMOTE_URL` | kosong | Alamat engine jarak jauh untuk `mcp` dan `hook` |
| `VIBEASIK_UI_DIR` | Studio tersemat | Folder Studio hasil build, untuk pengembangan Studio |
| `DB_DRIVER` | `sqlite` | `sqlite` atau `mysql`. Dengan `mysql`, atur `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` |
| `DB_PATH` | `~/.vibeasik/projects.db` | Lokasi database SQLite |

## Port

Engine memakai satu port (8765). Port untuk dev server proyek Anda dikelola otomatis oleh port registry
(`vibeasik ports`): tiap proyek mendapat blok port sendiri sehingga tidak bentrok. Jangan memilih port manual untuk
proyek yang dikelola vibeasik.
