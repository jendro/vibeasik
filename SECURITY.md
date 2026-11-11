# Keamanan vibeasik

vibeasik berjalan di laptop atau dev server Anda dan **bisa menjalankan perintah** (dev server, agent coding)
serta **membaca kode** proyek Anda. Karena itu engine dikunci secara bawaan.

## Model keamanan engine lokal

| Lapisan | Perilaku bawaan |
|---|---|
| Bind jaringan | `127.0.0.1` saja. Akses LAN harus dinyalakan sadar: `vibeasik serve --host 0.0.0.0` |
| Pemeriksaan Host | Hanya `localhost` / IP loopback (dan IP LAN privat bila mode LAN). Menolak serangan DNS rebinding |
| Token akses lokal | Wajib untuk semua API, MCP HTTP, dan halaman Studio — **termasuk dari localhost**, karena website mana pun yang Anda buka di browser bisa mengirim request ke localhost. Disimpan di `~/.vibeasik/token` (izin `0600`) |
| Studio | Masuk lewat `vibeasik open` atau tautan `/auth?t=…` yang dicetak `vibeasik serve`; token disimpan sebagai cookie `HttpOnly`, `SameSite=Strict` |
| Penjaga eksekusi | Aksi yang menjalankan atau menyimpan perintah shell hanya dari loopback dengan origin localhost, walau pemanggil punya token. LAN: `VIBEASIK_ALLOW_LAN_EXEC=1` atau `VIBEASIK_EXEC_TOKEN` + header `X-Vibeasik-Exec-Token` |
| Gerbang LAN | `vibeasik lan`: proxy ber-PIN 6 digit di port 8766; engine tetap bind `127.0.0.1` dan token tidak keluar dari laptop. Hanya IP privat, PIN berganti setelah 5 kali salah, Origin selain halaman gerbang ditolak. Komputer yang lolos PIN setara dengan `vibeasik open` (termasuk aksi eksekusi) selama 30 hari; PIN & sesi disimpan di `~/.vibeasik/lan-gate.json` (0600). HTTP biasa: hanya untuk jaringan tepercaya |
| CORS | Hanya dev server Studio (`http://localhost:8760`); ubah lewat `VIBEASIK_CORS_ORIGINS` |
| Header | `X-Content-Type-Options: nosniff`, `X-Frame-Options: SAMEORIGIN`, `Referrer-Policy: no-referrer` |

Pengecualian token: `GET /health`, aset statis Studio, dan `POST /api/traffic/ingest` (berkunci sendiri per sumber traffic).

### Memakai token

```bash
vibeasik token            # tampilkan token
vibeasik token --rotate   # buat token baru (restart serve; sesi Studio & konfigurasi MCP HTTP lama tidak berlaku)
vibeasik open             # buka Studio dan masuk otomatis
```

Agent yang memakai **MCP stdio** (`vibeasik mcp`) tidak perlu dikonfigurasi: token dibaca dari berkas.
Agent yang memakai **MCP HTTP** (`http://localhost:8765/mcp`) harus mengirim header `X-Vibeasik-Token`.

Token bisa diganti dengan env `VIBEASIK_TOKEN` (mis. engine di dev server jarak jauh; jembatan:
`vibeasik mcp --remote http://host:8765 --token <token>`).

### Data

Kode, fakta proyek, tangkapan desain, dan kredensial provider AI **tidak pernah dikirim** ke server vibeasik.
Yang dikirim ke Pusat Pengetahuan hanya sinyal & usulan yang Anda setujui.

## Melaporkan celah keamanan

Jangan membuka issue publik untuk celah keamanan. Laporkan secara privat lewat salah satu:

1. **GitHub**: tab *Security* repo ini, lalu *Report a vulnerability* (pelaporan privat).
2. **Email**: jendro@tenan.id (Jendro, CEO PT Pancen Apik Tenan)

Sertakan langkah reproduksi, dampak, dan versi (`vibeasik version`). Kami berusaha membalas dalam 3 hari kerja dan
memberi kabar perbaikan sebelum pengungkapan publik.
