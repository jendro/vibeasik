# vibeasik

> Lapisan standar untuk kode buatan AI. Berjalan di laptop atau dev server Anda, dipanggil agent coding lewat MCP.

[English summary](README.en.md) · [Mulai cepat](docs/GETTING_STARTED.md) · [MCP](docs/MCP.md) · [Edisi & paket](docs/EDITIONS.md) · [Arsitektur](docs/ARCHITECTURE.md)

vibeasik adalah program lokal untuk **vibecoder**, yaitu orang yang membangun software bersama agent AI.
Ia mencatat semua proyek Anda, mengaudit kode hasil AI dengan **85 rule bawaan**, mengatur port & dev server agar tidak
bentrok, membuat peta kode, dan menjadi **server MCP** sehingga Claude Code, Cursor, Antigravity, dan agent lain
bisa memakai semua itu.

**Semuanya berjalan lokal.** Kode, fakta proyek, dan kunci API Anda tidak pernah diunggah ke server mana pun.

## Pasang

Linux dan macOS:

```bash
curl -fsSL https://vibeasik.com/install.sh | sh
```

Windows: unduh `.zip` dari `https://vibeasik.com/dl/` (file `vibeasik-<versi>-windows-amd64.zip`), ekstrak, lalu
jalankan `vibeasik.exe`.

Installer memeriksa checksum SHA-256 sebelum memasang dan menaruh binary di `~/.local/bin/vibeasik`.
Setelah itu:

```bash
vibeasik serve          # jalankan engine di http://127.0.0.1:8765
vibeasik open           # buka Studio (antarmuka web) di browser, masuk otomatis
vibeasik mcp install    # daftarkan ke agent coding yang terpasang
```

Panduan lengkap, termasuk memindai proyek pertama: [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md).

> Dua cara mendapatkan vibeasik, dan bedanya: **installer** memasang build *Official* (fitur Pro terkunci sampai Anda
> login dengan paket Pro), sedangkan **build dari kode ini** adalah build *Community* (sepenuhnya open source, tanpa
> modul Pro). Lihat [docs/EDITIONS.md](docs/EDITIONS.md).

## Apa yang bisa dilakukan

| Fitur | Perintah / tempat |
|---|---|
| Mengenali proyek dan komponennya (backend, frontend, stack, port, perintah dev) | `vibeasik scan <folder>`, `vibeasik list`, Studio |
| Audit kode dengan 85 rule dalam 12 kategori | `vibeasik rules`, `vibeasik diagnose`, Studio |
| Port registry tanpa bentrok antarproyek | `vibeasik ports` |
| Peta kode dan rute | Studio |
| Daftar todo per proyek | MCP, Studio |
| Server MCP untuk agent (64 tool di Community) | `vibeasik mcp`, `vibeasik mcp list` |
| Pusat Pengetahuan: panduan & checklist yang diperbarui bersama pengguna | `vibeasik knowledge pull` |

## Community dan Pro

| | Community (gratis) | Pro (langganan) |
|---|---|---|
| Scan proyek, 85 rule audit, port & runner, peta kode, todo | ✓ | ✓ |
| Pusat Pengetahuan (paket dasar; kirim otomatis terkunci) | ✓ | ✓ jaring lengkap; kirim otomatis dapat diubah |
| MCP | 64 tool | 123 tool |
| Deep audit, AI review, pegawai AI, orkestrator, Galeri Desain, … | terkunci | ✓ |

Fitur Pro dibuka oleh *entitlement* bertanda tangan dari server vibeasik (`vibeasik login`). Tanpa akun, semua fitur
Community tetap jalan penuh. Rinciannya: [docs/EDITIONS.md](docs/EDITIONS.md).

## Pusat Pengetahuan

```bash
vibeasik knowledge consent               # baca dan setujui Syarat & Ketentuan (sekali)
vibeasik knowledge pull                  # tarik paket terbaru (tanda tangan & checksum diverifikasi)
vibeasik knowledge auto preview          # lihat node mana yang akan dikirim, tanpa mengirim
vibeasik knowledge ledger                # riwayat kiriman
```

Pencarian pengetahuan selalu berjalan **di laptop Anda**. Pusat Pengetahuan bekerja dua arah: pengetahuan di komputer
Anda yang terbukti berguna dikirim otomatis (setelah disensor dan disaring ketat; maksimal 5 per hari) untuk memperkaya
paket. Bawaan menyala; **pengguna Pro dapat mengubahnya, pengguna Free tidak**. Fitur lokal lain tidak terkait.
Kode, nama proyek, path, dan kunci tidak pernah dikirim. Detail dan syaratnya: [docs/KNOWLEDGE.md](docs/KNOWLEDGE.md).

## Keamanan

Engine hanya mendengarkan `127.0.0.1`, mewajibkan token lokal untuk semua API, dan memeriksa header Host. Ia bisa
menjalankan perintah di komputer Anda, jadi jangan membukanya ke jaringan tanpa memahami
[SECURITY.md](SECURITY.md). Celah keamanan dilaporkan secara privat, bukan lewat issue publik.

## Bangun dari kode

Prasyarat: Go 1.26+, [bun](https://bun.sh) (untuk Studio), `make`.

```bash
git clone https://github.com/vibeasik/vibeasik.git && cd vibeasik
make hooks      # aktifkan aturan commit (sekali setelah clone)
make studio     # bangun Studio ke pkg/ui/dist (tersemat di binary)
make build      # hasil: bin/vibeasik
make test
```

Module Go-nya `github.com/vibeasik/vibeasik`, sama dengan alamat repo ini, sehingga bisa diimpor dari proyek Go lain.
Untuk memakai programnya, pasang lewat installer atau `make build` seperti di atas.

## Berkontribusi

Kontribusi diterima: perbaikan bug, rule audit baru (`guidelines/`), dan dokumentasi. Mulai dari
[CONTRIBUTING.md](CONTRIBUTING.md). Aturan commit: [COMMITS.md](COMMITS.md).

## Lisensi

Kode: [Apache-2.0](LICENSE). Isi Pusat Pengetahuan: paket gratis CC BY 4.0, paket penuh berlisensi terbatas
(lihat [docs/KNOWLEDGE.md](docs/KNOWLEDGE.md#lisensi)). Modul Pro dan server vibeasik adalah proyek terpisah dan tidak termasuk repo ini.
