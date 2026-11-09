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

## Paket

| | Free | Pro |
|---|---|---|
| Scan proyek, 85 rule audit, port & runner, peta kode, todo | ✓ | ✓ |
| MCP | 64 tool | 123 tool |
| Pusat Pengetahuan | paket dasar; kirim otomatis terkunci | jaring lengkap; kirim otomatis dapat diubah |
| Deep audit, AI review, pegawai AI, orkestrator, Galeri Desain, … | terkunci | ✓ |

Free tidak butuh akun. Bagian mana pun dari fitur Pro yang belum terbuka akan menjawab `upgrade_required` (HTTP 402
untuk REST, kode yang sama untuk MCP), dan Studio menampilkan halaman *Upgrade* untuk fitur itu.

Untuk tim yang memerlukan pendampingan langsung, audit kilat, atau implementasi orkestrasi AI terkelola, lihat [Paket Layanan Profesional](SERVICES.md).

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
