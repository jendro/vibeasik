# Edisi dan paket

vibeasik dibuat dengan model **open-core**: inti-nya open source, sebagian fitur lanjutan berbayar.

## Dua build

| | Community | Official |
|---|---|---|
| Sumber | Repo ini, Apache-2.0 | Repo ini + modul Pro (tertutup) |
| Cara mendapatkan | `git clone`, lalu `make build` | Installer `https://vibeasik.com/install.sh` atau unduhan di `vibeasik.com/dl` |
| Fitur Community | Semua | Semua |
| Fitur Pro | Tidak ada kodenya; rute dan tool Pro menjawab `upgrade_required` | Ada, terkunci sampai paket Anda membukanya |
| Studio (antarmuka web) | Versi Community | Versi Official (halaman Pro ikut) |

Kedua build memakai engine, format data, dan protokol yang sama. Data Anda (`~/.vibeasik/`) bisa dipakai oleh
keduanya.

## Tiga paket edisi

vibeasik menawarkan tiga tingkat paket sesuai skala kebutuhan pengembangan:

| Fitur / Entitlement | Community (Free) | Pro | Team / Business |
|---|---|---|---|
| **Sasaran Pengguna** | Developer solo, hobiis, open source | Power developer, tech lead, konsultan | Tim rekayasa software (3–50 developer) |
| **Harga** | **Rp 0** (Gratis selamanya) | **Rp 299.000 / bulan** (~$19/bln) | **Rp 750.000 / seat / bulan** (~$49/seat/bln) |
| **Batas Perangkat** | Tanpa batas (tanpa akun) | Maks. 3 perangkat per akun | Hingga 5 perangkat per seat anggota |
| **Katalog MCP Tools** | 64 tool (inti lokal) | 123 tool (lengkap) | 123 tool + enterprise org API |
| **Fast Scan, Code Map, Port, Todo** | ✓ | ✓ | ✓ |
| **Deep Audit & AI Code Review** | Terkunci (`upgrade_required`) | ✓ | ✓ |
| **Pegawai AI & Orkestrator Kantor** | Terkunci (`upgrade_required`) | ✓ (1 workspace kantor) | ✓ (Multi-workspace kantor terdistribusi) |
| **Pusat Pengetahuan** | Paket dasar tersemat | Jaring lengkap L1 (kurasi lokal) | Jaring lengkap L2 (repositori patokan tim) |
| **AI Credits Bawaan** | 0 cloud credits | **1.000 Credits / bulan** | **3.000 Credits / seat / bulan** (pooled) |
| **Dukungan BYOK & Local LLM** | ✓ Bebas tanpa batas kuota | ✓ Bebas tanpa memotong kredit | ✓ Bebas + Private VPC LLM gateway |
| **Penyensoran Privasi (SEC-003)** | ✓ GlobalSanitizer | ✓ GlobalSanitizer | ✓ GlobalSanitizer + custom regex rahasia |
| **Dukungan & SLA** | Komunitas / GitHub | Email prioritas (< 24 jam) | Dedicated Slack, SLA < 4 jam, 2 jam workshop |

Community tidak butuh akun. Bagian mana pun dari fitur Pro yang belum terbuka akan menjawab `upgrade_required` (HTTP 402 untuk REST, kode yang sama untuk MCP), dan Studio menampilkan tautan peningkatan paket.

Rincian formula biaya, alokasi kredit per fitur, arsitektur database GORM, alur kuota habis, dan privasi kode: lihat [Spesifikasi AI Credits & Paket](AI_CREDITS.md).

Untuk tim yang memerlukan pendampingan langsung, audit kilat, atau implementasi orkestrasi AI terkelola, lihat [Paket Layanan Profesional](SERVICES.md), [Contoh Laporan Audit Anonim](SAMPLE_AUDIT_REPORT_ANONYMIZED.md), [Laporan Validasi Pasar Primer](MARKET_VALIDATION_REPORT.md), serta [Laporan Evaluasi Kelayakan VPS Hosting](VPS_HOSTING_FEASIBILITY.md).

## Cara fitur Pro dibuka

1. `vibeasik login` membuka browser untuk masuk (alur *device*, tanpa menyalin password ke terminal).
2. Server vibeasik menerbitkan **entitlement**: token yang ditandatangani dengan Ed25519 berisi paket dan daftar fitur.
3. Engine memverifikasinya dengan **kunci publik yang tertanam di binary**. Mengedit `~/.vibeasik/account.json` tidak
   membuka apa pun karena tanda tangannya akan gagal.
4. Entitlement berlaku 7 hari dan diperbarui otomatis di latar belakang. Bila Anda offline, fitur tetap jalan hingga
   **14 hari** masa tenggang. Jam komputer yang dimundurkan ditolak.

`vibeasik whoami` menampilkan akun, paket, dan fitur yang aktif; `vibeasik logout` mencabut perangkat ini dari akun.

## Apa yang dikirim ke server

Hanya yang perlu untuk login, entitlement, dan Pusat Pengetahuan. Kode proyek, fakta proyek, tangkapan desain, dan
kunci API provider AI Anda **tidak pernah dikirim**. Rinciannya: [KNOWLEDGE.md](KNOWLEDGE.md) dan
[SECURITY.md](../SECURITY.md).

## Pertanyaan umum

**Apakah Community dibatasi waktu atau jumlah proyek?** Tidak.

**Bisakah saya membangun sendiri lalu membuka fitur Pro?** Tidak. Kode Pro tidak ada di repo ini, dan fitur Pro di
build Official hanya terbuka oleh entitlement bertanda tangan server vibeasik.

**Bagaimana bila server vibeasik tidak bisa dijangkau?** Semua fitur lokal tetap berjalan. Fitur Pro tetap aktif
selama masa tenggang offline; sesudahnya kembali ke Free sampai engine bisa memperbarui entitlement.
