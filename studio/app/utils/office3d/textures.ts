import * as THREE from 'three'
import type { OfficePlanningState } from '~/types'
import { relativeTime } from '~/utils/agentOffice'

/** Tekstur kanvas untuk tulisan di dunia 3D (label ruang, papan nama, layar monitor). */
export function canvasTexture(w: number, h: number, draw: (ctx: CanvasRenderingContext2D) => void) {
  const canvas = document.createElement('canvas')
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext('2d')!
  draw(ctx)
  const texture = new THREE.CanvasTexture(canvas)
  texture.colorSpace = THREE.SRGBColorSpace
  texture.anisotropy = 4
  return { canvas, ctx, texture }
}

const FONT = 'ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif'

/** Tulisan satu baris, latar transparan (atau warna `bg`), lebar mengikuti panjang teks. */
export function textTexture(text: string, opts: { color?: string, bg?: string, size?: number, weight?: number, pad?: number } = {}) {
  const size = opts.size ?? 44
  const pad = opts.pad ?? 16
  const font = `${opts.weight ?? 800} ${size}px ${FONT}`
  const probe = document.createElement('canvas').getContext('2d')!
  probe.font = font
  const w = Math.ceil(probe.measureText(text).width + pad * 2)
  const h = Math.ceil(size * 1.5)
  const tex = canvasTexture(w, h, (ctx) => {
    if (opts.bg) {
      ctx.fillStyle = opts.bg
      ctx.beginPath()
      ctx.roundRect(0, 0, w, h, h / 4)
      ctx.fill()
    }
    ctx.font = font
    ctx.fillStyle = opts.color ?? '#475569'
    ctx.textBaseline = 'middle'
    ctx.textAlign = 'center'
    ctx.fillText(text, w / 2, h / 2 + 2)
  })
  return { ...tex, aspect: w / h }
}

/** Kisi ubin lantai (putih, diberi warna lewat material). */
export function tileTexture() {
  const { texture } = canvasTexture(64, 64, (ctx) => {
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(0, 0, 64, 64)
    ctx.strokeStyle = 'rgba(0,0,0,0.07)'
    ctx.lineWidth = 2
    ctx.strokeRect(0, 0, 64, 64)
  })
  texture.wrapS = texture.wrapT = THREE.RepeatWrapping
  return texture
}

export interface ScreenState { active: boolean, burning: boolean, project: string, seed: number }

/** Layar monitor: mati, kode berjalan + nama proyek, atau terbakar saat gagal. */
export function drawScreen(ctx: CanvasRenderingContext2D, s: ScreenState) {
  const { width: w, height: h } = ctx.canvas
  if (s.burning) {
    const g = ctx.createLinearGradient(0, h, 0, 0)
    g.addColorStop(0, '#f97316')
    g.addColorStop(0.6, '#dc2626')
    g.addColorStop(1, '#450a0a')
    ctx.fillStyle = g
    ctx.fillRect(0, 0, w, h)
    ctx.font = `${h * 0.5}px ${FONT}`
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText('🔥', w / 2, h / 2)
    return
  }
  ctx.fillStyle = s.active ? '#0b1220' : '#111827'
  ctx.fillRect(0, 0, w, h)
  if (!s.active) return
  const colors = ['#38bdf8', '#a78bfa', '#34d399', '#fbbf24', '#f472b6']
  let r = s.seed
  for (let y = 8; y < h - 22; y += 9) {
    r = (r * 1103515245 + 12345) >>> 0
    const indent = (r % 4) * 8
    const len = 20 + (r >>> 8) % (w - 40 - indent)
    ctx.fillStyle = colors[(r >>> 4) % colors.length]!
    ctx.fillRect(8 + indent, y, len, 4)
  }
  if (s.project) {
    ctx.fillStyle = 'rgba(56,189,248,0.25)'
    ctx.fillRect(0, h - 20, w, 20)
    ctx.fillStyle = '#e0f2fe'
    ctx.font = `700 13px ${FONT}`
    ctx.textBaseline = 'middle'
    ctx.textAlign = 'left'
    ctx.fillText(`📁 ${s.project}`.slice(0, 22), 6, h - 10)
  }
}

/** Pecah teks menjadi baris selebar `maxW`, paling banyak `maxLines` (baris terakhir diberi elipsis). */
function wrapLines(ctx: CanvasRenderingContext2D, text: string, maxW: number, maxLines: number) {
  const lines: string[] = []
  let line = ''
  for (const word of text.split(/\s+/).filter(Boolean)) {
    const next = line ? `${line} ${word}` : word
    if (ctx.measureText(next).width <= maxW || !line) { line = next; continue }
    lines.push(line)
    line = word
    if (lines.length === maxLines) break
  }
  if (lines.length < maxLines && line) lines.push(line)
  else if (line) {
    let last = lines[maxLines - 1]!
    while (last && ctx.measureText(`${last}…`).width > maxW) last = last.slice(0, -1)
    lines[maxLines - 1] = `${last}…`
  }
  return lines
}

const PRIO_COLORS: Record<string, string> = { critical: '#dc2626', high: '#f97316', medium: '#0ea5e9' }

type BoardPlanning = NonNullable<OfficePlanningState>
type BoardMeeting = NonNullable<BoardPlanning['latest']>
type BoardRunning = NonNullable<BoardPlanning['active']>[number]

/** Kursor gambar Papan Keputusan: kanvas, ukuran, dan posisi y yang bergeser seiring konten. */
interface Board {
  ctx: CanvasRenderingContext2D
  w: number
  h: number
  pad: number
  maxW: number
  y: number
}

function boardParagraph(b: Board, text: string, font: string, color: string, lines: number, lh: number) {
  b.ctx.font = font
  b.ctx.fillStyle = color
  for (const l of wrapLines(b.ctx, text, b.maxW, lines)) { b.ctx.fillText(l, b.pad, b.y); b.y += lh }
}

function drawBoardBackground(b: Board) {
  const bg = b.ctx.createLinearGradient(0, 0, 0, b.h)
  bg.addColorStop(0, '#0f172a')
  bg.addColorStop(1, '#1e1b4b')
  b.ctx.fillStyle = bg
  b.ctx.fillRect(0, 0, b.w, b.h)
  b.ctx.textBaseline = 'top'
  b.ctx.textAlign = 'left'
}

function drawHeader(b: Board, planning: OfficePlanningState | undefined) {
  const { ctx, w, pad } = b
  ctx.fillStyle = 'rgba(99,102,241,0.25)'
  ctx.fillRect(0, 0, w, 76)
  ctx.fillStyle = '#e0e7ff'
  ctx.font = `800 36px ${FONT}`
  ctx.fillText('📋 PAPAN KEPUTUSAN', pad, 20)
  const queued = (planning?.active ?? []).filter(s => s.status === 'queued').length
  const badge = planning && !planning.enabled ? ['NONAKTIF', '#64748b'] : queued ? [`${queued} antre`, '#a855f7'] : null
  if (!badge) return
  ctx.font = `800 26px ${FONT}`
  const bw = ctx.measureText(badge[0]!).width + 28
  ctx.fillStyle = badge[1]!
  ctx.beginPath()
  ctx.roundRect(w - pad - bw, 20, bw, 38, 8)
  ctx.fill()
  ctx.fillStyle = '#ffffff'
  ctx.fillText(badge[0]!, w - pad - bw + 14, 27)
}

function drawRunningMeeting(b: Board, running: BoardRunning) {
  const { ctx, pad } = b
  ctx.fillStyle = '#a855f7'
  ctx.beginPath()
  ctx.arc(pad + 10, b.y + 18, 10, 0, Math.PI * 2)
  ctx.fill()
  ctx.font = `800 34px ${FONT}`
  ctx.fillStyle = '#d8b4fe'
  ctx.fillText(`Rapat #${running.id} · ${running.project_name}`.slice(0, 44), pad + 32, b.y)
  b.y += 56
  boardParagraph(b, running.step || 'Menyiapkan rapat…', `700 30px ${FONT}`, '#f8fafc', 2, 40)
  b.y += 10
  if (running.situation) boardParagraph(b, running.situation, `500 26px ${FONT}`, '#cbd5e1', 6, 34)
}

function drawDecisionRow(b: Board, d: NonNullable<BoardMeeting['decisions']>[number]) {
  const { ctx, pad, y } = b
  ctx.font = `800 18px ${FONT}`
  const label = d.priority.toUpperCase()
  const pw = ctx.measureText(label).width + 16
  ctx.fillStyle = PRIO_COLORS[d.priority] ?? '#64748b'
  ctx.beginPath()
  ctx.roundRect(pad, y + 3, pw, 26, 5)
  ctx.fill()
  ctx.fillStyle = '#ffffff'
  ctx.fillText(label, pad + 8, y + 7)
  ctx.font = `600 25px ${FONT}`
  ctx.fillStyle = '#f1f5f9'
  ctx.fillText(wrapLines(ctx, d.title, b.maxW - pw - 12, 1)[0] ?? '', pad + pw + 12, y + 1)
  b.y += 38
}

function drawMeetingList(b: Board, latest: BoardMeeting) {
  for (const d of (latest.decisions ?? []).filter(d => d.outcome !== 'duplicate').slice(0, 3)) {
    if (b.y > b.h - 44) break
    drawDecisionRow(b, d)
  }
}

function drawSummary(b: Board, latest: BoardMeeting, nowMs: number) {
  const { ctx, w, pad } = b
  ctx.font = `800 32px ${FONT}`
  ctx.fillStyle = '#f8fafc'
  ctx.fillText(`Rapat #${latest.id} · ${latest.project_name}`.slice(0, 40), pad, b.y)
  ctx.font = `500 24px ${FONT}`
  ctx.fillStyle = '#94a3b8'
  ctx.textAlign = 'right'
  ctx.fillText(relativeTime(latest.finished_at, nowMs), w - pad, b.y + 6)
  ctx.textAlign = 'left'
  b.y += 54
  if (latest.status === 'failed') {
    boardParagraph(b, `✖ ${latest.error ?? 'Rapat gagal'}`, `600 28px ${FONT}`, '#fca5a5', 6, 38)
    return
  }
  boardParagraph(b, latest.add_features ? '✚ Perlu fitur baru' : '● Belum perlu fitur baru', `800 30px ${FONT}`, latest.add_features ? '#4ade80' : '#cbd5e1', 1, 44)
  if (latest.summary) boardParagraph(b, latest.summary, `500 25px ${FONT}`, '#e2e8f0', 4, 33)
  b.y += 8
  if (latest.todos_rejected) boardParagraph(b, `✕ ${latest.todos_rejected} saran lama ditolak`, `700 24px ${FONT}`, '#fca5a5', 1, 34)
  drawMeetingList(b, latest)
}

/**
 * Layar TV Papan Keputusan di Ruang Eksekutif: rapat yang sedang berjalan/antre, atau ringkasan keputusan rapat terakhir.
 * Hanya data rapat asli dari engine.
 */
export function drawDecisionBoard(ctx: CanvasRenderingContext2D, planning: OfficePlanningState | undefined, nowMs: number) {
  const { width: w, height: h } = ctx.canvas
  const pad = 36
  const b: Board = { ctx, w, h, pad, maxW: w - pad * 2, y: 100 }
  drawBoardBackground(b)
  drawHeader(b, planning)
  const running = planning?.active?.find(s => s.status === 'running')
  const latest = planning?.latest
  if (running) drawRunningMeeting(b, running)
  else if (latest) drawSummary(b, latest, nowMs)
  else boardParagraph(b, 'Belum ada rapat. PM akan rapat saat Saran AI diminta, agent coding meminta saran, atau todo selesai dikerjakan agent.', `500 28px ${FONT}`, '#94a3b8', 6, 38)
}
