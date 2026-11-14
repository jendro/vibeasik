import { describe, expect, it } from 'bun:test'
import { buildProjectNavItems, isProjectNavActive, normalizeProjectTab, pageTitleFor, PROJECT_TAB_PAGES, PROJECT_TABS } from '../app/utils/sidebarMenuItems'

describe('menu proyek aktif', () => {
  const items = buildProjectNavItems(7, { name: 'Demo' })
  const groupOf = (id: string) => items.find(i => i.id === id)?.group

  it('setiap tab detail proyek punya item sidebar sendiri', () => {
    for (const t of PROJECT_TABS) {
      const it = items.find(i => i.id === `tab:${t.id}`)
      expect(it).toBeDefined()
      expect(it!.to).toBe(t.id === 'overview' ? '/projects/7' : `/projects/7?tab=${t.id}`)
    }
  })

  it('Biaya Proses ada di kelompok pemeriksaan bersama audit, uji runtime & traffic', () => {
    for (const id of ['audit-report', 'process-cost', 'runtime-test', 'traffic', 'security-test', 'screenshots']) {
      expect(groupOf(id)).toBe('Periksa & tingkatkan')
    }
    expect(groupOf('tab:overview')).toBe('Ringkasan')
    expect(groupOf('tab:recap')).toBe('Ringkasan')
    expect(groupOf('todos')).toBe('Kerjakan perbaikan')
  })

  it('kelompok tidak terpecah: item sekelompok berurutan', () => {
    const seen: string[] = []
    for (const it of items) {
      if (seen[seen.length - 1] !== it.group) {
        expect(seen).not.toContain(it.group)
        seen.push(it.group!)
      }
    }
  })

  it('tanpa proyek aktif hanya satu item detail yang menuju pemilihan proyek', () => {
    const none = buildProjectNavItems(null, null).filter(i => i.isDetail)
    expect(none.map(i => i.to)).toEqual(['/cockpit'])
  })

  it('item aktif mengikuti ?tab, dan detail aplikasi menandai Aplikasi', () => {
    const byId = (id: string) => items.find(i => i.id === id)!
    expect(isProjectNavActive(byId('tab:recap'), '/projects/7', 'recap')).toBe(true)
    expect(isProjectNavActive(byId('tab:overview'), '/projects/7', 'recap')).toBe(false)
    expect(isProjectNavActive(byId('process-cost'), '/process-cost', undefined)).toBe(true)
    expect(isProjectNavActive(byId('tab:overview'), '/projects/7', undefined)).toBe(true)
    expect(isProjectNavActive(byId('tab:apps'), '/projects/7/apps/3', undefined)).toBe(true)
    expect(isProjectNavActive(byId('tab:apps'), '/projects/7', 'topology')).toBe(true)
    expect(isProjectNavActive(byId('audit-report'), '/audit-report', undefined)).toBe(true)
  })
})

describe('tab & judul halaman', () => {
  it('id tab lama dan id asing dinormalkan', () => {
    expect(normalizeProjectTab('base_knowledge')).toBe('technical')
    expect(normalizeProjectTab('visual_audit')).toBe('overview')
    expect(normalizeProjectTab('ngawur')).toBe('overview')
    expect(PROJECT_TAB_PAGES.cost).toBe('/process-cost')
  })
  it('judul navbar memakai nama tab', () => {
    expect(pageTitleFor('/projects/7', 'recap').title).toBe('Rekap Pemilik')
    expect(pageTitleFor('/process-cost').title).toBe('Biaya Proses')
    expect(pageTitleFor('/projects/7').title).toBe('Ringkasan Proyek')
    expect(pageTitleFor('/projects/7/apps/2').title).toBe('Detail Aplikasi')
  })
})
