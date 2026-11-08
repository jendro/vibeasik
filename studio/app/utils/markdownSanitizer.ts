import { Marked } from 'marked'
import DOMPurify, { type DOMPurifyI, type Config } from 'dompurify'

export const escapeHtml = (text: string): string =>
  text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')

// Allowlist ketat untuk tag yang dihasilkan markdown; selain ini dibuang DOMPurify (K5079).
export const SANITIZE_CONFIG: Config = {
  ALLOWED_TAGS: [
    'p', 'strong', 'em', 'b', 'i', 'del', 'code', 'pre', 'ul', 'ol', 'li', 'span', 'a', 'br', 'hr',
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'blockquote', 'table', 'thead', 'tbody', 'tr', 'th', 'td'
  ],
  ALLOWED_ATTR: ['class', 'href', 'target', 'rel'],
  ALLOW_DATA_ATTR: false
}

export type PurifyInstance = typeof DOMPurify | DOMPurifyI

const installedHooks = new WeakSet<object>()

export const ensureLinkHook = (purify: PurifyInstance = DOMPurify): void => {
  if (typeof purify.addHook !== 'function') return
  if (installedHooks.has(purify as object)) return
  installedHooks.add(purify as object)

  purify.addHook('afterSanitizeAttributes', (node: Element) => {
    if (node.tagName === 'A') {
      node.setAttribute('target', '_blank')
      node.setAttribute('rel', 'noopener noreferrer')
    }
  })
}

export const createMarkedInstance = (): Marked =>
  new Marked({
    gfm: true,
    breaks: true,
    renderer: {
      html({ text }) {
        return escapeHtml(text)
      }
    }
  })

const defaultMd = createMarkedInstance()

// DOMPurify butuh DOM (browser). Tanpa DOM (prerender/SSR) fallback ke teks ter-escape, tidak pernah HTML mentah.
export const sanitizeHtml = (html: string, purify: PurifyInstance = DOMPurify): string => {
  if (!purify.isSupported) return escapeHtml(html)
  ensureLinkHook(purify)
  return purify.sanitize(html, SANITIZE_CONFIG)
}

export interface RenderMarkdownOptions {
  purify?: PurifyInstance
  marked?: Marked
}

export const renderMarkdown = (
  content?: string | null,
  options?: RenderMarkdownOptions
): string => {
  const text = (content || '').trim()
  if (!text) return ''

  const md = options?.marked || defaultMd
  const purify = options?.purify || DOMPurify

  try {
    const raw = md.parse(text) as string
    return sanitizeHtml(raw, purify)
  } catch (err) {
    console.warn('Failed to parse markdown:', err)
    return escapeHtml(text)
  }
}
