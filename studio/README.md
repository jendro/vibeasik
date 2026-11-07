# vibeasik Studio

Antarmuka web vibeasik (Nuxt 4 + Nuxt UI 4). Hasil build disematkan ke binary `vibeasik` (lihat `pkg/ui`).

```bash
bun install
NUXT_PUBLIC_API_BASE=http://localhost:8765 bun run dev   # dev server :8760, engine di :8765
make -C .. studio                                          # build & salin ke pkg/ui/dist
```

Menu fitur berbayar memakai `useEdition()` + `utils/proFeatures.ts`: halaman Pro yang belum terbuka diarahkan
ke `/upgrade`. Komponen Pro di build ini berupa stub; build Official menimpanya dengan komponen asli.

Berasal dari Nuxt UI Starter Template (MIT, lihat `LICENSE`).
