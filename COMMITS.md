# Aturan Commit

Repo ini memakai **Conventional Commits** dan menolak commit yang tidak rapi lewat hook di `.githooks/`.
Aktifkan sekali setelah clone:

```bash
make hooks   # = git config core.hooksPath .githooks
```

## Format
```
<type>(<scope>): <subjek, maks 72 karakter>

<isi opsional: kenapa, bukan apa>
```

| type | Untuk |
|---|---|
| `feat` | Fitur baru |
| `fix` | Perbaikan bug |
| `refactor` | Ubah struktur tanpa ubah perilaku |
| `perf` | Kinerja |
| `test` | Tes |
| `docs` | Dokumentasi |
| `build` / `ci` | Build, dependensi, pipeline |
| `chore` | Pekerjaan rutin lain |

Contoh: `feat(scanner): deteksi stack Bun`, `fix(runner): port bentrok saat dua komponen start`.

## Aturan
- Satu commit = satu perubahan logis. Tidak ada `wip`, `update`, `misc`.
- Hook `pre-commit` menolak `.env`, database, binary, log, `node_modules`, berkas > 1 MB, dan pola kunci rahasia.
- Jangan force-push ke `main`.
