import { describe, expect, test } from 'bun:test'
import { JSDOM } from 'jsdom'
import DOMPurify from 'dompurify'
import {
  escapeHtml,
  SANITIZE_CONFIG,
  sanitizeHtml,
  renderMarkdown,
  createMarkedInstance,
  ensureLinkHook
} from '../app/utils/markdownSanitizer'

// Setup JSDOM instance to test DOMPurify in browser-like environment
const dom = new JSDOM('<!DOCTYPE html><html><body></body></html>')
const jsdomPurify = DOMPurify(dom.window)

describe('markdownSanitizer - escapeHtml', () => {
  test('meng-escape karakter khusus HTML', () => {
    expect(escapeHtml('<script>alert("xss") & \'foo\'</script>')).toBe(
      '&lt;script&gt;alert(&quot;xss&quot;) &amp; &#39;foo&#39;&lt;/script&gt;'
    )
  })
})

describe('markdownSanitizer - sanitizeHtml & SANITIZE_CONFIG', () => {
  test('memiliki allowlist tag dan atribut eksplisit (K5079)', () => {
    expect(SANITIZE_CONFIG.ALLOWED_TAGS).toBeDefined()
    expect(SANITIZE_CONFIG.ALLOWED_ATTR).toBeDefined()
    expect(SANITIZE_CONFIG.ALLOW_DATA_ATTR).toBe(false)
    expect(SANITIZE_CONFIG.ALLOWED_TAGS).not.toContain('script')
    expect(SANITIZE_CONFIG.ALLOWED_TAGS).not.toContain('iframe')
    expect(SANITIZE_CONFIG.ALLOWED_TAGS).not.toContain('img')
    expect(SANITIZE_CONFIG.ALLOWED_TAGS).not.toContain('svg')
    expect(SANITIZE_CONFIG.ALLOWED_ATTR).not.toContain('onerror')
    expect(SANITIZE_CONFIG.ALLOWED_ATTR).not.toContain('onclick')
  })

  test('fallback ke escapeHtml bila purify.isSupported bernilai false (SSR safe)', () => {
    const mockSSRPurify = {
      isSupported: false,
      sanitize: () => '<raw-unsanitized-html>'
    }
    const result = sanitizeHtml('<script>alert(1)</script><p>Aman</p>', mockSSRPurify)
    expect(result).not.toContain('<script>')
    expect(result).toContain('&lt;script&gt;alert(1)&lt;/script&gt;')
  })

  test('menghilangkan tag berbahaya dan event handler saat DOMPurify aktif', () => {
    const dirty = '<p>Halo</p><script>alert("xss")</script><img src="x" onerror="alert(1)">'
    const clean = sanitizeHtml(dirty, jsdomPurify)
    expect(clean).not.toContain('<script>')
    expect(clean).not.toContain('onerror')
    expect(clean).not.toContain('<img')
    expect(clean).toContain('<p>Halo</p>')
  })
})

describe('markdownSanitizer - renderMarkdown XSS Neutralization', () => {
  test('mengembalikan string kosong untuk konten kosong/null/undefined', () => {
    expect(renderMarkdown('')).toBe('')
    expect(renderMarkdown(null)).toBe('')
    expect(renderMarkdown(undefined)).toBe('')
    expect(renderMarkdown('   ')).toBe('')
  })

  test('menetralisir raw HTML script injection (<script>)', () => {
    const payload = '# Judul\n<script>alert("pwned")</script>\nKonten normal'
    const result = renderMarkdown(payload, { purify: jsdomPurify })
    expect(result).not.toContain('<script>')
    expect(result).not.toContain('alert("pwned")</script>')
  })

  test('menetralisir payload gambar dengan event handler (<img onerror> dan ![alt](url))', () => {
    // 1. Raw HTML img disanitasi & di-escape menjadi entitas teks aman
    const rawPayload = 'Coba injeksi: <img src="invalid.jpg" onerror="alert(document.cookie)">'
    const rawResult = renderMarkdown(rawPayload, { purify: jsdomPurify })
    expect(rawResult).not.toContain('<img')
    expect(rawResult).toContain('&lt;img')

    // 2. Markdown image syntax ![alt](url) dibuang oleh DOMPurify karena tag img tidak ada di ALLOWED_TAGS
    const mdImagePayload = 'Gambar markdown: ![foto bahaya](https://evil.example.com/bad.png)'
    const mdResult = renderMarkdown(mdImagePayload, { purify: jsdomPurify })
    expect(mdResult).not.toContain('<img')
  })

  test('menetralisir tautan markdown dengan skema javascript: ([click](javascript:...))', () => {
    const payload = '[Klik Hadiah](javascript:alert("xss"))'
    const result = renderMarkdown(payload, { purify: jsdomPurify })
    expect(result).not.toContain('href="javascript:')
    expect(result).not.toContain('javascript:alert')
  })

  test('menetralisir inline tag iframe, svg, dan objek asing lainnya', () => {
    const payload = '<iframe src="https://evil.example.com"></iframe><svg onload="alert(1)">'
    const result = renderMarkdown(payload, { purify: jsdomPurify })
    expect(result).not.toContain('<iframe')
    expect(result).not.toContain('<svg')
    expect(result).toContain('&lt;iframe')
    expect(result).toContain('&lt;svg')
  })
})

describe('markdownSanitizer - Safe Markdown Rendering', () => {
  test('merender markdown standar (formatting, code, list, heading)', () => {
    const markdown = [
      '# Heading 1',
      '## Heading 2',
      'Teks dengan **tebal**, *miring*, dan ~~coret~~.',
      '- Butir 1',
      '- Butir 2',
      '`inline code`',
      '```ts',
      'const x = 42',
      '```'
    ].join('\n\n')

    const result = renderMarkdown(markdown, { purify: jsdomPurify })
    expect(result).toContain('<h1>Heading 1</h1>')
    expect(result).toContain('<h2>Heading 2</h2>')
    expect(result).toContain('<strong>tebal</strong>')
    expect(result).toContain('<em>miring</em>')
    expect(result).toContain('<del>coret</del>')
    expect(result).toContain('<ul>')
    expect(result).toContain('Butir 1')
    expect(result).toContain('<code>inline code</code>')
    expect(result).toContain('<pre><code')
  })

  test('merender tautan aman dengan target="_blank" dan rel="noopener noreferrer"', () => {
    const markdown = '[vibeasik](https://vibeasik.dev)'
    const result = renderMarkdown(markdown, { purify: jsdomPurify })
    expect(result).toContain('href="https://vibeasik.dev"')
    expect(result).toContain('target="_blank"')
    expect(result).toContain('rel="noopener noreferrer"')
  })

  test('merender tabel markdown dengan tag aman (table, tr, th, td)', () => {
    const tableMd = '| Kolom A | Kolom B |\n|---|---|\n| Nilai 1 | Nilai 2 |'
    const result = renderMarkdown(tableMd, { purify: jsdomPurify })
    expect(result).toContain('<table>')
    expect(result).toContain('<th>Kolom A</th>')
    expect(result).toContain('<td>Nilai 1</td>')
  })
})
