# Mulai cepat

Dari nol sampai agent coding Anda memakai vibeasik, sekitar lima menit.

## 1. Pasang

**Linux / macOS**

```bash
curl -fsSL https://vibeasik.com/install.sh | sh
```

Installer mengunduh binary sesuai OS dan arsitektur Anda (`amd64` atau `arm64`), memeriksa checksum SHA-256, lalu
memasangnya di `~/.local/bin/vibeasik`. Bila `~/.local/bin` belum ada di `PATH`, installer memberi tahu perintahnya.

Variabel opsional: `VIBEASIK_VERSION` (mis. `v0.1.0`, bawaan versi terbaru), `INSTALL_DIR` (bawaan `~/.local/bin`),
`VIBEASIK_SITE` (alamat unduhan lain).

**Windows**

Unduh `vibeasik-<versi>-windows-amd64.zip` dari `https://vibeasik.com/dl/`, ekstrak, dan jalankan `vibeasik.exe`
dari terminal.

**Dari kode sumber** (build Community): lihat [bagian Bangun dari kode di README](../README.md#bangun-dari-kode).

Periksa:

```bash
vibeasik version
```

## 2. Jalankan engine

```bash
vibeasik serve
```

Engine berjalan di `http://127.0.0.1:8765` dan mencetak tautan masuk ke Studio. Biarkan terminal ini terbuka, atau
jalankan sebagai layanan di komputer Anda. Port diganti dengan `--port` atau env `VIBEASIK_PORT`.

Buka Studio, yaitu antarmuka web vibeasik:

```bash
vibeasik open
```

`open` memakai token lokal di `~/.vibeasik/token`, jadi Anda tidak perlu memasukkan apa pun.

## 3. Kenalkan proyek pertama

```bash
vibeasik scan ~/proyek/toko-saya        # tampilkan komponen yang terdeteksi: backend, frontend, stack
vibeasik list                           # daftar proyek yang sudah tercatat
vibeasik diagnose ~/proyek/toko-saya    # pemeriksaan kesehatan dan arsitektur
vibeasik rules                          # 85 rule audit bawaan per kategori
```

Menambahkan proyek ke daftar, melihat temuan audit, peta kode, dan todo paling nyaman lewat Studio atau lewat agent
Anda (langkah 4).

### Port tanpa bentrok

Bila beberapa proyek memakai port yang sama (3000, 8080, …):

```bash
vibeasik ports                   # lihat port registry
vibeasik ports --ensure-unique   # alokasikan port unik bebas bentrok untuk semua komponen
```

## 4. Sambungkan agent coding

```bash
vibeasik mcp install --dry-run   # lihat dulu apa yang akan diubah
vibeasik mcp install             # daftarkan ke agent yang terpasang
```

Agent yang didukung: Claude Code, Claude Desktop, Cursor, Windsurf, Antigravity, Cline, dan Roo. Mulai ulang agent,
lalu minta, misalnya, "tampilkan proyek saya lewat vibeasik" atau "audit proyek ini". Detail, termasuk
konfigurasi manual dan koneksi jarak jauh: [MCP.md](MCP.md).

## 5. Pusat Pengetahuan (opsional)

```bash
vibeasik knowledge pull
```

Menarik paket panduan dan checklist terbaru. Tidak perlu login untuk paket dasar. Lihat [KNOWLEDGE.md](KNOWLEDGE.md).

## 6. Akun dan fitur Pro (opsional)

```bash
vibeasik login      # membuka browser untuk masuk atau mendaftar
vibeasik whoami     # akun, paket, dan fitur yang aktif
vibeasik logout
```

Tanpa akun, semua fitur Community berjalan penuh. Lihat [EDITIONS.md](EDITIONS.md).

## Masalah umum

| Gejala | Penyebab dan solusi |
|---|---|
| `vibeasik: command not found` | `~/.local/bin` belum ada di `PATH`. Tambahkan `export PATH="$HOME/.local/bin:$PATH"` ke profil shell Anda |
| Studio menolak akses atau meminta token | Buka lewat `vibeasik open`. Token ada di `~/.vibeasik/token`; `vibeasik token --rotate` membuat yang baru |
| Port 8765 sudah dipakai | `vibeasik serve --port 8766` atau set `VIBEASIK_PORT` |
| `knowledge pull` gagal menghubungi server | Periksa koneksi internet. Server bawaan `https://enjoy.vibeasik.com`; ganti dengan `VIBEASIK_CLOUD_URL` |
| Agent tidak melihat vibeasik setelah `mcp install` | Mulai ulang agent. Jalankan `vibeasik mcp list` untuk memastikan server MCP berfungsi |
