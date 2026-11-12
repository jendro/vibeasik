import { describe, expect, test } from 'bun:test'
import { clampPage, needsPaging, pageSizeFor, pageSlice, perPageOptionsFor } from '../app/utils/listPaging'

describe('listPaging', () => {
  test('ukuran halaman mengikuti anggaran 800 simpul DOM', () => {
    expect(pageSizeFor(160)).toBe(5) // kartu alur data
    expect(pageSizeFor(40)).toBe(20) // baris katalog rute
    expect(pageSizeFor(35)).toBe(20) // baris matriks
    expect(pageSizeFor(25)).toBe(25)
    expect(pageSizeFor(14)).toBe(50)
    expect(pageSizeFor(4)).toBe(100) // dibatasi 100
    expect(pageSizeFor(1000)).toBe(5) // baris sangat berat tetap minimal 5
  })

  test('pilihan per halaman: bawaan, 2x, 4x, maks 100', () => {
    expect(perPageOptionsFor(5)).toEqual([5, 10, 20])
    expect(perPageOptionsFor(50)).toEqual([50, 100])
    expect(perPageOptionsFor(100)).toEqual([100])
  })

  test('wajib paging bila melebihi ukuran halaman', () => {
    expect(needsPaging(5, 160)).toBe(false)
    expect(needsPaging(6, 160)).toBe(true)
  })

  test('halaman dijepit ke rentang yang ada', () => {
    expect(clampPage(9, 12, 5)).toBe(3)
    expect(clampPage(0, 12, 5)).toBe(1)
    expect(clampPage(2, 0, 5)).toBe(1)
    expect(clampPage(Number.NaN, 12, 5)).toBe(1)
  })

  test('potongan halaman tanpa duplikasi atau baris terlewat', () => {
    const items = Array.from({ length: 23 }, (_, i) => i)
    const seen = [1, 2, 3, 4, 5].flatMap(p => pageSlice(items, p, 5))
    expect(seen).toEqual(items)
    expect(pageSlice(items, 99, 5)).toEqual([20, 21, 22])
  })
})
