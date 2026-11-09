import { describe, expect, it } from 'bun:test'
import fs from 'node:fs'
import path from 'node:path'

describe('Zero Alert Policy & Non-blocking Modal Audit (FE-004 / todo #3472)', () => {
  const targetFiles = [
    'studio/app/components/diagnostics/AuditMechanismView.vue',
    'studio/app/components/projects/detail/RouteCatalogSection.vue',
    'studio/app/pages/cockpit.vue',
    'studio/app/components/settings/SettingsKnowledgeTab.vue',
  ]

  it('semua target file harus bebas dari pemanggilan native dialog (alert, confirm, prompt)', () => {
    for (const relPath of targetFiles) {
      const fullPath = path.resolve(process.cwd(), relPath)
      expect(fs.existsSync(fullPath)).toBe(true)
      const content = fs.readFileSync(fullPath, 'utf-8')
      expect(content).not.toMatch(/\b(window\.)?(alert|confirm|prompt)\s*\(/)
    }
  })

  it('seluruh komponen Vue di studio/app tidak memanggil native dialog', () => {
    function scanDir(dir: string): string[] {
      let results: string[] = []
      for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
        const full = path.join(dir, entry.name)
        if (entry.isDirectory()) {
          results = results.concat(scanDir(full))
        } else if (entry.name.endsWith('.vue')) {
          results.push(full)
        }
      }
      return results
    }

    const vueFiles = scanDir(path.resolve(process.cwd(), 'studio/app'))
    expect(vueFiles.length).toBeGreaterThan(10)
    for (const file of vueFiles) {
      const content = fs.readFileSync(file, 'utf-8')
      expect(content).not.toMatch(/\b(window\.)?(alert|confirm|prompt)\s*\(/)
    }
  })

  it('komponen konfirmasi menggunakan UModal non-blocking', () => {
    for (const relPath of [
      'studio/app/components/projects/detail/RouteCatalogSection.vue',
      'studio/app/pages/cockpit.vue',
      'studio/app/components/settings/SettingsKnowledgeTab.vue',
    ]) {
      const content = fs.readFileSync(path.resolve(process.cwd(), relPath), 'utf-8')
      expect(content).toContain('<UModal')
    }
  })
})
