# Berkontribusi ke vibeasik

Terima kasih sudah mau membantu. Kontribusi yang paling berguna:

- **Perbaikan bug** dengan langkah reproduksi dan, bila bisa, tes.
- **Rule audit baru atau perbaikan rule** di `guidelines/` (berkas JSON per kategori).
- **Dokumentasi**: koreksi, contoh, terjemahan.
- **Dukungan agent** baru untuk `vibeasik mcp install`.

Untuk perubahan besar, buka issue lebih dulu agar arahnya cocok sebelum Anda mengerjakannya.

## Menyiapkan lingkungan

Prasyarat: Go 1.26+, [bun](https://bun.sh), `make`, git.

```bash
git clone https://github.com/vibeasik/vibeasik.git && cd vibeasik
make hooks      # WAJIB sekali setelah clone: mengaktifkan pemeriksaan commit
make studio     # membangun Studio (hanya perlu bila Anda mengubah/menjalankan Studio)
make build      # bin/vibeasik
make test
```

Jalankan hasil build tanpa mengganggu data Anda sendiri dengan database terpisah:

```bash
bin/vibeasik --db /tmp/vibeasik-dev.db serve --port 8799
```

## Alur kerja

1. Fork repo, buat cabang dari `main` (`fix/nama-singkat` atau `feat/nama-singkat`).
2. Buat perubahan kecil dan terfokus. Satu pull request = satu perubahan logis.
3. Jalankan `make fmt vet test` sebelum mengirim.
4. Tulis commit sesuai [COMMITS.md](COMMITS.md) (Conventional Commits, subjek ≤ 72 karakter). Hook akan menolak yang
   tidak sesuai.
5. Buka pull request: jelaskan *kenapa* perubahan ini perlu, bukan hanya apa yang berubah.

## Aturan yang ditegakkan hook

Commit ditolak bila memuat berkas `.env`, `*.db`, binary, log, berkas lebih dari 1 MB, atau pola kunci rahasia.
Bila tes Anda butuh token palsu, pecah stringnya (`"ghp" + "_..."`) agar tidak terbaca sebagai rahasia asli.

## Yang tidak diterima di repo ini

- Kode Pro, kunci, kredensial, data pribadi, atau path seperti `/home/<nama>`.
- Ketergantungan baru tanpa alasan kuat; engine sengaja berjalan tanpa CGO dan satu binary.
- Perubahan yang melemahkan model keamanan lokal (token, pemeriksaan Host, penjaga eksekusi) tanpa diskusi.

## Melaporkan celah keamanan

Jangan lewat issue publik. Ikuti [SECURITY.md](SECURITY.md).

## Lisensi

Dengan mengirim kontribusi, Anda setuju kontribusi itu dilisensikan di bawah [Apache-2.0](LICENSE), sama seperti repo ini.
