# Pusat Pengetahuan

Pusat Pengetahuan adalah kumpulan panduan, checklist audit, dan teori desain yang dipakai agent saat bekerja.
Isinya tumbuh bersama pengguna, tetapi **pencarian selalu berjalan di laptop Anda**, sehingga server tetap ringan
dan konteks proyek Anda tidak keluar dari komputer.

## Cara kerjanya

```
   server vibeasik                         laptop Anda
 ┌─────────────────────┐   paket bertanda   ┌──────────────────────────┐
 │ paket pengetahuan   │ ─────────────────▶ │ verifikasi → impor lokal │
 │ (berkas statis)     │   tangan Ed25519   │ pencarian & rekomendasi  │
 │                     │ ◀───────────────── │ (tanpa jaringan)         │
 │ antrean kurasi      │   angka & usulan   └──────────────────────────┘
 └─────────────────────┘   yang Anda setujui
```

1. **Paket**: server menerbitkan berkas `.json.gz` berversi dan bertanda tangan. Ada dua tingkat: *free* (panduan,
   checklist audit, teori desain) dan *full* (jaring pengetahuan lengkap dengan tautan antarnode, untuk Pro).
2. **Tarik**: `vibeasik knowledge pull`. Engine memeriksa checksum SHA-256 dan tanda tangan Ed25519; paket yang
   diubah atau rusak ditolak. Paket free tidak butuh login.
3. **Impor**: node masuk ke database lokal sebagai sumber `hub`, terpisah dari pengetahuan yang Anda buat sendiri.
4. **Pakai**: agent mencari pengetahuan lewat MCP; semuanya dari database lokal.

Engine juga menarik paket baru secara berkala di latar belakang. Matikan dengan `VIBEASIK_KNOWLEDGE_SYNC=off`.

## Perintah

```bash
vibeasik knowledge pull               # tarik paket terbaru
vibeasik knowledge status             # versi paket terpasang dan status sinkronisasi
vibeasik knowledge signals            # kirim angka helpful/harmful (anonim, wajib login)
vibeasik knowledge contribute         # tinjau usulan pengetahuan Anda
vibeasik knowledge contribute --send 12,15   # kirim usulan yang Anda setujui
```

## Privasi: apa yang dikirim, apa yang tidak

| Data | Dikirim? |
|---|---|
| Kode proyek, nama file, fakta proyek, tangkapan layar desain | Tidak pernah |
| Kunci API dan kredensial provider AI | Tidak pernah |
| Pengetahuan pribadi milik "pegawai AI" Anda | Tidak pernah |
| Angka *helpful* / *harmful* per node paket | Ya, anonim, tanpa teks, wajib login (`knowledge signals`) |
| Usulan pengetahuan baru | Hanya yang Anda setujui satu per satu lewat `contribute --send` |

Setiap usulan **disensor lebih dulu** (token, kunci, path pribadi, dan pola rahasia dibuang) sebelum Anda lihat, dan
Anda tetap menyetujui isinya sebelum dikirim. Usulan masuk antrean kurasi; hanya yang disetujui kurator yang masuk
paket berikutnya.

## Server sendiri

Alamat server bawaan `https://enjoy.vibeasik.com`. Ganti dengan env `VIBEASIK_CLOUD_URL` bila Anda menjalankan server
yang kompatibel.
