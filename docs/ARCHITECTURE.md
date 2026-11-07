# Arsitektur

Untuk kontributor dan siapa pun yang ingin memahami isi repo.

```
 agent coding (Claude Code, Cursor, …)           browser (Studio)
        │ MCP stdio / HTTP                              │ HTTP
        ▼                                               ▼
 ┌──────────────────────────  vibeasik (satu binary Go, tanpa CGO) ─────────────────────────┐
 │  pkg/cli  ──  pkg/mcp  ──  pkg/api (Fiber)  ──  pkg/ui (Studio tersemat)                 │
 │                 │              │                                                          │
 │            pkg/registry  (modul, gate fitur, hook)   pkg/localauth (token, Host check)    │
 │                 │                                                                         │
 │  scanner · analysis · guidelines(85 rule) · runner · codemap · routemap · todorag          │
 │  knowledge · knowledgesync · account · security(sanitizer) · database (SQLite/MySQL)      │
 └───────────────────────────────────────────────────────────────────────────────────────────┘
                    │ HTTPS (opsional): login, entitlement, paket pengetahuan
                    ▼
              server vibeasik (repo terpisah)
```

## Peta paket (`pkg/`)

| Paket | Tugas |
|---|---|
| `cli` | Perintah `vibeasik` (cobra): serve, open, scan, mcp, knowledge, login, … |
| `api` | REST API (Fiber), akses lokal (token, Host, CORS), menyajikan Studio |
| `mcp` | Server MCP stdio dan HTTP dan daftar tool |
| `registry` | Titik sambung modul: pendaftaran modul, gate fitur (HTTP 402 `upgrade_required`), hook, katalog Pro |
| `localauth` | Token lokal `~/.vibeasik/token`, cookie sesi Studio |
| `account` | Login device flow, entitlement bertanda Ed25519, pembaruan latar belakang |
| `scanner`, `analysis`, `diagnostics` | Mengenali stack proyek dan memeriksanya |
| `guidelines` | 85 rule audit (`guidelines/*.json` di-embed) |
| `runner`, `workspace` | Dev server proyek, port registry |
| `codemap`, `routemap`, `codectx` | Peta kode dan rute untuk konteks agent |
| `knowledge`, `knowledgesync` | Basis pengetahuan lokal; tarik paket, sinyal, dan usulan |
| `security` | Penyensor rahasia pada teks yang akan dikirim atau disimpan |
| `database`, `models`, `repository`, `service` | Penyimpanan (GORM, SQLite atau MySQL) |
| `ui` | Studio hasil build yang tersemat di binary (`go:embed`) |
| `cloud` | Klien server vibeasik |

Kode inti ada di `pkg/` (bukan `internal/`) supaya build Official dapat mengimpornya.

## Open-core: bagaimana Pro tersambung

Repo ini lengkap dan bisa dibangun sendiri (Community). Build Official adalah proyek lain yang **mengimpor repo ini**
dan mendaftarkan modul tambahan lewat `pkg/registry`; arah ketergantungan selalu dari Pro ke sini, tidak pernah
sebaliknya.

- **Modul**: Pro mendaftarkan modul bernama dengan satu fitur. Rute REST dan tool MCP-nya dilindungi *gate*.
- **Gate**: `registry.Has(feature)` menanyakan entitlement. Bila belum terbuka, rute menjawab HTTP 402
  `upgrade_required` dan tool MCP menjawab kode yang sama.
- **Katalog Pro**: di Community, daftar rute Pro (`registry.ProCatalog`) tetap menjawab 402 agar Studio dan agent tahu
  fitur itu ada dan bisa menawarkan upgrade.
- **Hook**: titik sambung kecil (mis. menyaring temuan audit, aksi saat proyek berubah atau todo selesai) yang boleh
  diisi modul Pro tanpa mengubah inti.

Kode yang menyentuh rahasia, data pribadi, atau modul Pro tidak boleh masuk repo ini.

## Studio

Antarmuka web di `studio/` (Nuxt 4, bun). `make studio` membangunnya ke `pkg/ui/dist`, yang tersemat di binary.
Saat mengembangkan Studio: jalankan `vibeasik serve` dan dev server Nuxt (`cd studio && bun run dev`, port 8760);
atau arahkan `VIBEASIK_UI_DIR` ke folder hasil build.

## Menguji

```bash
make test          # go test ./...
make vet           # go vet ./...
make fmt           # gofmt
```

## Catatan API: diagnostik project

`GET /api/projects/:id` tidak lagi memuat field `diagnostics` (sebelumnya ~8,7 MB). Riwayat diagnostik
diambil lewat `GET /api/projects/:id/diagnostics?limit=&offset=&component_id=&full=1` dengan respons
`{items,total,limit,offset}` (default limit 100, maks 500, terbaru dulu; tanpa `full=1` item tidak membawa
`findings_json`/`optimizations`/`structure_tree`). Laporan terbaru tetap di `/diagnostics/latest`.
Tool MCP `get_project_detail` juga tidak lagi menyertakan `diagnostics`.
