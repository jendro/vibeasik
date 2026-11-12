// Aturan paging daftar di Studio: berapa baris ditampilkan sekaligus ditentukan oleh bobot DOM
// satu baris, bukan angka tetap. Anggaran 800 simpul DOM di layar (Lighthouse memperingatkan > 800).

export const DOM_NODE_BUDGET = 800
export const PAGE_SIZE_STEPS = [5, 10, 20, 25, 50, 100] as const

/** Ukuran halaman: floor(800 / simpul per baris), dibulatkan turun ke langkah baku 5…100. */
export function pageSizeFor(nodesPerRow: number): number {
  const raw = Math.floor(DOM_NODE_BUDGET / Math.max(1, Math.round(nodesPerRow)))
  let size: number = PAGE_SIZE_STEPS[0]
  for (const s of PAGE_SIZE_STEPS) if (s <= raw) size = s
  return size
}

/** Pilihan "Per hal": ukuran bawaan lalu kelipatannya (2× dan 4×), paling besar 100. */
export function perPageOptionsFor(size: number): number[] {
  const opts = [size, size * 2, size * 4].filter(n => n <= 100)
  return [...new Set(opts.length ? opts : [size])]
}

/** Daftar wajib dipaging bila jumlahnya melebihi ukuran halaman bawaannya. */
export function needsPaging(total: number, nodesPerRow: number): boolean {
  return total > pageSizeFor(nodesPerRow)
}

export function clampPage(page: number, total: number, perPage: number): number {
  const pages = Math.max(1, Math.ceil(total / Math.max(1, perPage)))
  if (!Number.isFinite(page) || page < 1) return 1
  return Math.min(Math.floor(page), pages)
}

export function pageSlice<T>(items: readonly T[], page: number, perPage: number): T[] {
  const p = clampPage(page, items.length, perPage)
  return items.slice((p - 1) * perPage, p * perPage)
}
