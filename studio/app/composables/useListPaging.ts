import type { MaybeRefOrGetter, WatchSource } from 'vue'

interface ListPagingOptions {
  /** Simpul DOM yang dirender satu baris/kartu; menentukan ukuran halaman (lihat utils/listPaging). */
  nodesPerRow: number
  /** Simpan nomor halaman di query string (?<queryKey>=3) agar bisa kembali ke halaman yang sama. */
  queryKey?: string
  /** Sumber yang mengembalikan ke halaman 1 bila berubah (filter, pencarian, tampilan). */
  resetOn?: WatchSource[]
}

// Paging sisi klien untuk daftar yang datanya sudah utuh di satu respons (peta, hasil scan).
// Daftar yang bisa tumbuh tanpa batas tetap dipaging di server (limit/offset), bukan di sini.
export function useListPaging<T>(items: MaybeRefOrGetter<readonly T[]>, opts: ListPagingOptions) {
  const route = useRoute()
  const router = useRouter()

  const defaultSize = pageSizeFor(opts.nodesPerRow)
  const perPageOptions = perPageOptionsFor(defaultSize)
  const perPage = ref(defaultSize)
  const fromQuery = opts.queryKey ? Number(route.query[opts.queryKey]) : 1
  const page = ref(Number.isFinite(fromQuery) && fromQuery > 1 ? Math.floor(fromQuery) : 1)

  const total = computed(() => toValue(items).length)
  const totalPages = computed(() => Math.max(1, Math.ceil(total.value / perPage.value)))
  // Kontrol paging muncul begitu daftar melebihi ukuran halaman terkecil.
  const enabled = computed(() => total.value > perPageOptions[0]!)
  const pagedItems = computed(() => pageSlice(toValue(items), page.value, perPage.value))
  const rangeStart = computed(() => (total.value ? (page.value - 1) * perPage.value + 1 : 0))
  const rangeEnd = computed(() => Math.min(page.value * perPage.value, total.value))

  // Data yang menyusut (filter/segarkan) tidak boleh meninggalkan halaman kosong.
  watch([total, perPage], () => {
    page.value = clampPage(page.value, total.value, perPage.value)
  })
  watch(perPage, () => { page.value = 1 })
  if (opts.resetOn?.length) watch(opts.resetOn, () => { page.value = 1 })

  if (opts.queryKey) {
    const key = opts.queryKey
    watch(page, (p) => {
      const cur = route.query[key]
      const next = p > 1 ? String(p) : undefined
      if (cur === next || (cur === undefined && next === undefined)) return
      const { [key]: _drop, ...rest } = route.query
      router.replace({ query: next ? { ...rest, [key]: next } : rest })
    })
  }

  return { page, perPage, perPageOptions, total, totalPages, enabled, pagedItems, rangeStart, rangeEnd }
}
