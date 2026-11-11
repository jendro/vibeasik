import { describe, expect, it } from 'bun:test'
import { isProjectUnified } from '../app/utils/projectLayout'

describe('isProjectUnified', () => {
  it('menganggap proyek satu induk bila semua aplikasi di bawah root', () => {
    expect(isProjectUnified({ root_path: '/w/app', components: [{ absolute_path: '/w/app/api' }, { absolute_path: '/w/app/web/' }] })).toBe(true)
  })
  it('terpencar bila ada aplikasi di luar root', () => {
    expect(isProjectUnified({ root_path: '/w/app', components: [{ absolute_path: '/w/app/api' }, { absolute_path: '/other/web' }] })).toBe(false)
  })
  it('tidak tertipu awalan nama folder yang mirip', () => {
    expect(isProjectUnified({ root_path: '/w/app', components: [{ absolute_path: '/w/app-old/api' }] })).toBe(false)
  })
  it('mengikuti flag is_unified bila sudah disatukan', () => {
    expect(isProjectUnified({ is_unified: true })).toBe(true)
  })
})
