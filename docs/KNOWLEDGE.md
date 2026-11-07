# Pusat Pengetahuan

Pusat Pengetahuan adalah kumpulan panduan, checklist audit, dan teori desain yang dipakai agent saat bekerja.
Isinya tumbuh bersama pengguna, tetapi **pencarian selalu berjalan di laptop Anda**, sehingga server tetap ringan
dan konteks proyek Anda tidak keluar dari komputer.

> **Berlaku mulai rilis setelah v0.1.0.** Versi v0.1.0 masih memakai usulan yang disetujui satu per satu.
>
> **Timbal balik.** Pusat Pengetahuan bekerja dua arah: Anda menerima paket, dan pengetahuan di komputer Anda yang
> terbukti berguna ikut dikirim (setelah disensor dan disaring) untuk memperkaya paket. Ini bawaan menyala; pengguna
> **Pro dapat mengubah atau mematikannya**, pengguna **Free tidak**. Fitur lokal vibeasik lainnya tidak terkait dan
> berjalan penuh tanpa Pusat Pengetahuan. Anda harus menyetujui Syarat & Ketentuan lebih dulu (`vibeasik knowledge
> consent`). Rinciannya di bagian *Kirim otomatis* di bawah.

## Cara kerjanya

```
   server vibeasik                         laptop Anda
 ┌─────────────────────┐   paket bertanda   ┌──────────────────────────┐
 │ paket pengetahuan   │ ─────────────────▶ │ verifikasi → impor lokal │
 │ (berkas statis)     │   tangan Ed25519   │ pencarian & rekomendasi  │
 │                     │ ◀───────────────── │ (tanpa jaringan)         │
 │ antrean kurasi      │   angka & usulan   └──────────────────────────┘
 └─────────────────────┘   yang lolos penyaringan
```

1. **Paket**: server menerbitkan berkas `.json.gz` berversi dan bertanda tangan. Ada dua tingkat: *free* (panduan,
   checklist audit, teori desain) dan *full* (jaring pengetahuan lengkap dengan tautan antarnode, untuk Pro).
2. **Tarik**: `vibeasik knowledge pull` (setelah menyetujui S&K). Engine memeriksa checksum SHA-256 dan tanda tangan
   Ed25519; paket yang diubah atau rusak ditolak.
3. **Impor**: node masuk ke database lokal sebagai sumber `hub`, terpisah dari pengetahuan yang Anda buat sendiri.
4. **Pakai**: agent mencari pengetahuan lewat MCP; semuanya dari database lokal.

Engine juga menarik paket baru secara berkala di latar belakang. Matikan dengan `VIBEASIK_KNOWLEDGE_SYNC=off`.

## Perintah

```bash
vibeasik knowledge consent            # baca dan setujui Syarat & Ketentuan Pusat Pengetahuan
vibeasik knowledge pull               # tarik paket terbaru
vibeasik knowledge status             # versi paket terpasang dan status sinkronisasi
vibeasik knowledge auto status        # mode kirim, persetujuan, jumlah kiriman
vibeasik knowledge auto preview       # node yang AKAN dikirim dan yang dilewati (tidak mengirim apa pun)
vibeasik knowledge auto on|ask|off    # mengubah mode: hanya paket Pro
vibeasik knowledge ledger             # riwayat kiriman dan alasan node dilewati
vibeasik knowledge signals            # kirim angka helpful/harmful (anonim, wajib login)
vibeasik knowledge contribute         # tinjau usulan; kirim manual: contribute --send 12,15
```

## Kirim otomatis

Node pengetahuan buatan Anda (hasil promosi, review QA, riset celah) dikirim ke server untuk dikurasi bila **semua**
syarat ini terpenuhi:

| Syarat | Nilai |
|---|---|
| Node bersama, aktif, bukan milik pegawai AI, bukan dari paket publik, belum pernah dikirim | wajib |
| Terbukti berguna | helpful ≥ 3 dan helpful ≥ 3 × harmful |
| Umur node | ≥ 7 hari |
| Penyensor rahasia tidak perlu mengubah apa pun | wajib (bila perlu disunting, node **tidak dikirim**) |
| Lolos pemeriksaan privasi: tidak ada path, IP, host internal, URL ber-token, string acak, UUID, email, telepon, nomor kartu, kata rahasia/kepemilikan, potongan kode panjang, nama proyek/komponen/repo Anda, atau kata sensor Anda | wajib |

Paling banyak **5 node per hari** per perangkat, dengan jeda acak. Hanya berjalan bila Anda sudah menyetujui S&K dan
login. Setiap kiriman dicatat di `~/.vibeasik/contributions.json` (hash isi, judul, waktu; bukan salinan isi), dan
`vibeasik knowledge ledger` menampilkan juga mengapa suatu node dilewati.

Di server, kiriman masuk **karantina**: ditinjau sebelum dipakai pengguna lain, dan tidak langsung menjadi isi paket.

| Paket | Mode |
|---|---|
| Free / Community | `auto`, terkunci (bagian dari Pusat Pengetahuan) |
| Pro | `auto` bawaan; dapat diubah ke `ask` (setujui satu per satu) atau `off` |

## Privasi: apa yang dikirim, apa yang tidak

| Data | Dikirim? |
|---|---|
| Kode proyek, nama file, fakta proyek, tangkapan layar desain | Tidak pernah |
| Nama proyek, komponen, atau repo Anda | Tidak pernah (node yang memuatnya tidak dikirim) |
| Kunci API dan kredensial provider AI | Tidak pernah |
| Pengetahuan pribadi milik "pegawai AI" Anda | Tidak pernah |
| Angka *helpful* / *harmful* per node paket | Ya, anonim, tanpa teks, wajib login (`knowledge signals`) |
| Judul dan isi node yang lolos semua syarat di atas | Ya, otomatis (Free) atau sesuai mode Anda (Pro) |

Kiriman ditinjau sebelum dipakai pengguna lain. Anda dapat melihat pratinjau dan riwayat kapan saja. Perintah
`vibeasik knowledge revoke` untuk meminta penghapusan kiriman yang belum terbit menyusul pada pembaruan berikutnya.
Data terbit tidak memuat identitas akun.

## Lisensi

| Bagian | Lisensi | Artinya |
|---|---|---|
| **Paket gratis** | [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/) | Boleh disalin, dibagikan, dan diubah, termasuk untuk komersial. Wajib menyebut "vibeasik Pusat Pengetahuan", menautkan lisensi, dan menyatakan perubahan |
| **Paket penuh** (Pro) | [Lisensi Pengetahuan vibeasik 1.0](https://vibeasik.com/legal/knowledge-license) | Hak cipta vibeasik. Boleh dipakai di komputer Anda untuk pekerjaan Anda; dilarang menerbitkan ulang, menjual, menyalin massal, atau memakainya untuk melatih model AI atau layanan yang bersaing |
| **Kode vibeasik dan 85 rule audit** | Apache-2.0 | Terbuka penuh, seperti repositori ini |
| **Isi yang Anda kirim** | Hak cipta tetap pada Anda | Anda memberi vibeasik lisensi luas (selamanya untuk bagian yang sudah terbit, boleh disublisensikan) untuk menerbitkannya dalam paket gratis maupun penuh |

Lisensi tertulis di dalam berkas paket yang ditandatangani server, jadi tidak dapat diubah tanpa membatalkan tanda
tangan, dan `vibeasik knowledge pull` serta `knowledge status` menampilkannya.

Alasan modelnya: paket gratis dibuat terbuka agar pengetahuan menyebar cepat dan kewajiban berkontribusi terasa adil
(yang Anda kirim ikut memperkaya milik bersama); nilai tertutup vibeasik ada di paket penuh (graf, tautan, skor dari
pemakaian) dan di mesin pembaruan serta kurasinya.

## Server sendiri

Alamat server bawaan `https://enjoy.vibeasik.com`. Ganti dengan env `VIBEASIK_CLOUD_URL` bila Anda menjalankan server
yang kompatibel.
