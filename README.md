# vibeasik

> Lapisan standar untuk kode buatan AI. Jalan di laptop atau dev server Anda, dipanggil agent lewat MCP.

vibeasik mencatat proyek Anda, mengaudit kode hasil AI (85 rule bawaan), mengatur port & dev server,
membuat peta kode, dan menjadi server MCP untuk agent coding (Claude Code, Cursor, Antigravity, Aider).
Semuanya berjalan **lokal**: kode Anda tidak pernah diunggah.

## Pasang

```bash
curl -fsSL https://vibeasik.com/install.sh | sh
vibeasik serve        # jalankan engine
vibeasik open         # buka Studio di browser (masuk otomatis)
```

Windows: unduh `.zip` dari halaman Releases.

## Gratis & Pro

| | Community (gratis) | Pro (langganan) |
|---|---|---|
| Scan, audit dasar + 85 rule, port & runner, peta kode, todo | ✓ | ✓ |
| Pusat Pengetahuan (paket dasar, diperbarui bersama pengguna) | ✓ | ✓ jaring lengkap |
| MCP dasar (±64 tool) | ✓ | ✓ + ±60 tool Pro |
| Deep audit, AI review, pegawai AI, orkestrator, Galeri Desain, … | terkunci | ✓ |

`vibeasik login` untuk masuk akun. Fitur Pro terbuka sesuai entitlement bertanda tangan dari server vibeasik.

## Pusat Pengetahuan

```bash
vibeasik knowledge pull                 # tarik paket terbaru (diverifikasi tanda tangan)
vibeasik knowledge contribute           # tinjau usulan Anda (sudah disensor)
vibeasik knowledge contribute --send 12 # kirim yang Anda setujui
```

Pencarian pengetahuan berjalan di laptop Anda. Yang dikirim ke server hanya angka (helpful/harmful) dan usulan
yang Anda setujui.

## Keamanan

Engine bind ke `127.0.0.1`, mewajibkan token lokal, dan memeriksa Host. Lihat [SECURITY.md](SECURITY.md).

## Pengembangan

```bash
make hooks     # aktifkan aturan commit (wajib sekali setelah clone)
make studio    # build Studio ke pkg/ui/dist
make build && make test
scripts/release.sh v0.1.0
```

Aturan commit: [COMMITS.md](COMMITS.md). Lisensi: [Apache-2.0](LICENSE).
