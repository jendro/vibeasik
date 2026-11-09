import { describe, it, expect } from 'bun:test'
import { AUDIT_CRITERIA } from '../app/utils/auditCriteria'

describe('auditCriteria catalog metadata validation (#3374)', () => {
  it('harus memuat seluruh aturan esensial (SEC-PATH-TRAVERSAL, BE-UNGUARDED-MUTATION, BE-WILDCARD-CORS, BE-SWALLOWED-ERROR)', () => {
    const ids = AUDIT_CRITERIA.map(c => c.id)
    expect(ids).toContain('SEC-PATH-TRAVERSAL')
    expect(ids).toContain('BE-UNGUARDED-MUTATION')
    expect(ids).toContain('BE-WILDCARD-CORS')
    expect(ids).toContain('BE-SWALLOWED-ERROR')
  })

  it('BE-WILDCARD-CORS tidak boleh memuat literal wildcard origin "*" yang memicu self-match', () => {
    const corsRule = AUDIT_CRITERIA.find(c => c.id === 'BE-WILDCARD-CORS')
    expect(corsRule).toBeDefined()
    expect(corsRule?.inspection_check).not.toContain('"*"')
    expect(corsRule?.example_code?.bad).not.toContain('"*"')
    expect(corsRule?.example_code?.bad).toContain('allowAllOrigins')
  })

  it('BE-UNGUARDED-MUTATION suggested_action harus normatif tanpa sintaks registrasi rute statis', () => {
    const mutRule = AUDIT_CRITERIA.find(c => c.id === 'BE-UNGUARDED-MUTATION')
    expect(mutRule).toBeDefined()
    expect(mutRule?.suggested_action).not.toContain('api.Post')
    expect(mutRule?.suggested_action).toContain('Lindungi endpoint mutasi')
  })

  it('SEC-PATH-TRAVERSAL inspection_check mengecualikan properti example_code', () => {
    const ptRule = AUDIT_CRITERIA.find(c => c.id === 'SEC-PATH-TRAVERSAL')
    expect(ptRule).toBeDefined()
    expect(ptRule?.inspection_check).toContain('Kecualikan string di dalam properti example_code')
  })
})
