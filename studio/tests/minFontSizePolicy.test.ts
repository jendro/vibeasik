import { describe, expect, it } from 'bun:test'
import fs from 'node:fs'
import path from 'node:path'

describe('Minimum Font Size Policy (FE-SMALL-FONT / todo #3474)', () => {
  const targetFiles = [
    'studio/app/components/AppVersionBadge.vue',
    'studio/app/components/knowledge/KnowledgeNodeDetail.vue',
    'studio/app/components/knowledge/KnowledgeDonut.vue',
    'studio/app/components/knowledge/KnowledgeImpact.vue',
  ]

  it('seluruh target berkas mematuhi batas ukuran teks minimum (>= 12px / text-xs)', () => {
    for (const relPath of targetFiles) {
      const fullPath = path.resolve(process.cwd(), relPath)
      expect(fs.existsSync(fullPath)).toBe(true)
      const content = fs.readFileSync(fullPath, 'utf-8')
      expect(content).not.toMatch(/\btext-\[(?:[1-9]|1[01])px\]/)
      expect(content).not.toMatch(/font-size:\s*(?:[1-9]|1[01])px/)
    }
  })

  it('seluruh berkas Vue di studio/app tidak menggunakan font size di bawah 12px', () => {
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
      expect(content).not.toMatch(/\btext-\[(?:[1-9]|1[01])px\]/)
      expect(content).not.toMatch(/font-size:\s*(?:[1-9]|1[01])px/)
    }
  })
})
