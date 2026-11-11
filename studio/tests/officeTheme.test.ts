import { describe, expect, test } from 'bun:test'
import { DEFAULT_OFFICE_THEME, applyOfficeTheme, dayPhase, nightFactor, sceneDate } from '../app/utils/office3d/theme'

const theme = () => structuredClone(DEFAULT_OFFICE_THEME)

describe('nightFactor', () => {
  test('siang 0, malam 1', () => {
    expect(nightFactor(12)).toBe(0)
    expect(nightFactor(8)).toBe(0)
    expect(nightFactor(21)).toBe(1)
    expect(nightFactor(2)).toBe(1)
  })

  test('fajar & senja berubah halus dan searah', () => {
    const dusk = [17.25, 17.6, 18, 18.4, 18.75].map(h => nightFactor(h))
    expect(dusk[0]).toBe(0)
    expect(dusk[4]).toBe(1)
    for (let i = 1; i < dusk.length; i++) expect(dusk[i]!).toBeGreaterThan(dusk[i - 1]!)
    const dawn = [5, 5.5, 6, 6.5].map(h => nightFactor(h))
    for (let i = 1; i < dawn.length; i++) expect(dawn[i]!).toBeLessThan(dawn[i - 1]!)
    expect(dawn[3]).toBe(0)
  })
})

describe('dayPhase', () => {
  test('fase mengikuti jam tema', () => {
    expect(dayPhase(3)).toBe('night')
    expect(dayPhase(5.5)).toBe('dawn')
    expect(dayPhase(12)).toBe('day')
    expect(dayPhase(18)).toBe('dusk')
    expect(dayPhase(19)).toBe('night')
  })
})

describe('applyOfficeTheme', () => {
  test('menimpa hanya kunci yang ditulis', () => {
    const t = applyOfficeTheme({ sky: { night: '#000000' }, lamp: { interiorNight: 22 }, hours: { dusk: [18, 19] } }, theme())
    expect(t.sky.night).toBe('#000000')
    expect(t.sky.day).toBe(DEFAULT_OFFICE_THEME.sky.day)
    expect(t.lamp.interiorNight).toBe(22)
    expect(t.hours.dusk).toEqual([18, 19])
    expect(t.hours.dawn).toEqual(DEFAULT_OFFICE_THEME.hours.dawn)
    expect(nightFactor(18, t)).toBe(0)
  })

  test('mengabaikan kunci asing dan tipe yang salah', () => {
    const t = applyOfficeTheme({ sky: 'biru', lamp: { interiorNight: 'terang', asing: 1 }, hours: { dawn: [5], dusk: ['a', 'b'] }, lain: {} }, theme())
    expect(t).toEqual(DEFAULT_OFFICE_THEME)
    expect(applyOfficeTheme(null, theme())).toEqual(DEFAULT_OFFICE_THEME)
    expect(applyOfficeTheme([1, 2], theme())).toEqual(DEFAULT_OFFICE_THEME)
  })
})

describe('sceneDate', () => {
  const now = new Date(2026, 10, 12, 9, 30).getTime()
  test('otomatis memakai jam sekarang; siang & malam dipaksa', () => {
    expect(sceneDate(now, 'auto').getHours()).toBe(9)
    expect(sceneDate(now, 'day').getHours()).toBe(12)
    expect(sceneDate(now, 'night').getHours()).toBe(21)
    expect(sceneDate(now, 'night').getDate()).toBe(12)
  })
})
