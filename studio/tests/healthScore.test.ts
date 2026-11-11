import { describe, expect, it } from 'bun:test'
import { healthLabel, healthTone } from '../app/utils/healthScore'

describe('healthScore', () => {
  it('membedakan belum diaudit dari skor 100', () => {
    expect(healthLabel(null)).toBe('Belum diaudit')
    expect(healthLabel(100)).not.toBe('Belum diaudit')
  })
  it('memakai ambang 90/70', () => {
    expect(healthTone(90)).toContain('emerald')
    expect(healthTone(89)).toContain('amber')
    expect(healthTone(69)).toContain('rose')
  })
})
