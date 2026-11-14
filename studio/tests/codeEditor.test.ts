import { describe, expect, it } from 'bun:test'
import type { CostEvidence } from '../app/types'
import { editorSegments, evidenceMark, isHeavyQuery, tokenizeLine } from '../app/utils/codeEditor'

const ev = (p: Partial<CostEvidence>): CostEvidence => ({ kind: 'query', line: 1, code: '', depth: 0, note: '', ...p })

describe('pewarnaan baris kode', () => {
  it('memisah kata kunci, pemanggilan, string, angka dan komentar', () => {
    const t = tokenizeLine('for _, id := range ids { db.First(&x, "id = ?", 42) } // N+1')
    const kinds = Object.fromEntries(t.map(x => [x.text.trim(), x.kind]))
    expect(kinds.for).toBe('keyword')
    expect(kinds.range).toBe('keyword')
    expect(kinds.First).toBe('func')
    expect(kinds['"id = ?"']).toBe('string')
    expect(kinds['42']).toBe('number')
    expect(kinds['// N+1']).toBe('comment')
    expect(t.map(x => x.text).join('')).toBe('for _, id := range ids { db.First(&x, "id = ?", 42) } // N+1')
  })
  it('variabel PHP dan komentar pagar', () => {
    const t = tokenizeLine('foreach ($rows as $r) { # loop')
    expect(t.find(x => x.text === '$rows')?.kind).toBe('var')
    expect(t.at(-1)).toEqual({ text: '# loop', kind: 'comment' })
  })
})

describe('jendela editor', () => {
  it('baris berdekatan digabung, celah jauh dilipat', () => {
    expect(editorSegments([10, 14, 40])).toEqual([{ from: 7, to: 17 }, { from: 37, to: 43 }])
  })
  it('tidak keluar dari awal/akhir berkas', () => {
    expect(editorSegments([2, 2], 3, 4)).toEqual([{ from: 1, to: 4 }])
  })
})

describe('penanda query berat', () => {
  it('query di dalam loop atau Q(n) bertumbuh = berat', () => {
    expect(isHeavyQuery(ev({ depth: 1 }))).toBe(true)
    expect(isHeavyQuery(ev({ note: 'menjalankan query lewat `load`: Q = 10n' }))).toBe(true)
    expect(isHeavyQuery(ev({ note: 'menjalankan query: Q = 1' }))).toBe(false)
    expect(isHeavyQuery(ev({ kind: 'loop', depth: 2 }))).toBe(false)
  })
  it('label penanda', () => {
    expect(evidenceMark(ev({ depth: 1 })).label).toBe('N+1')
    expect(evidenceMark(ev({ kind: 'loop', depth: 2 })).label).toBe('loop ×2')
    expect(evidenceMark(ev({ kind: 'call' })).tone).toBe('call')
  })
})
