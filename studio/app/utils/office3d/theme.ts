/**
 * Suasana Kantor Agent: warna langit, matahari/bulan, cahaya ruangan, dan jam fajar/senja di satu tempat.
 * Nilai bawaan bisa ditimpa lewat kunci `theme` di `public/office3d/assets.json` (lihat `applyOfficeTheme`),
 * jadi tampilan kantor bisa diperbarui tanpa menyentuh kode scene.
 */
export interface OfficeTheme {
  /** Warna langit & kabut per fase hari. */
  sky: { day: string, dusk: string, night: string }
  /** Cahaya arah: matahari siang, matahari rendah (fajar/senja), dan bulan. */
  sun: { day: string, low: string, moon: string, dayIntensity: number, moonIntensity: number }
  /** Cahaya langit merata (hemisphere). */
  ambient: { day: string, night: string, ground: string, dayIntensity: number, nightIntensity: number }
  /** Lampu ruangan & lampu luar: warna dan kekuatan siang/malam. */
  lamp: { color: string, interiorDay: number, interiorNight: number, outdoorNight: number, glowNight: number }
  /** Jendela gedung yang menyala saat malam (lantai sibuk lebih terang). */
  window: { nightBusy: number, nightIdle: number, dayBusy: number }
  /** Jam (0–24) awal & akhir peralihan fajar dan senja. */
  hours: { dawn: [number, number], dusk: [number, number] }
}

export const DEFAULT_OFFICE_THEME: OfficeTheme = {
  sky: { day: '#bfe3ff', dusk: '#f6b98a', night: '#131c3f' },
  sun: { day: '#fff7e6', low: '#fdba74', moon: '#b9c6ff', dayIntensity: 2.6, moonIntensity: 0.75 },
  ambient: { day: '#dbeafe', night: '#8794c9', ground: '#6b8f5e', dayIntensity: 1.45, nightIntensity: 0.95 },
  lamp: { color: '#ffdfae', interiorDay: 2.5, interiorNight: 14, outdoorNight: 13, glowNight: 0.6 },
  window: { nightBusy: 1.5, nightIdle: 0.85, dayBusy: 0.2 },
  hours: { dawn: [5, 6.5], dusk: [17.25, 18.75] }
}

/** Tema yang sedang dipakai scene (salinan bawaan; diubah lewat `applyOfficeTheme`). */
export const OFFICE_THEME: OfficeTheme = structuredClone(DEFAULT_OFFICE_THEME)

type DeepPartial<T> = { [K in keyof T]?: T[K] extends unknown[] ? T[K] : T[K] extends object ? DeepPartial<T[K]> : T[K] }
export type OfficeThemePatch = DeepPartial<OfficeTheme>

const isRecord = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null && !Array.isArray(v)

/** Timpa sebagian tema. Kunci yang tidak dikenal atau bertipe lain dari bawaannya diabaikan (manifest ditulis tangan). */
export function applyOfficeTheme(patch: unknown, target: OfficeTheme = OFFICE_THEME): OfficeTheme {
  if (!isRecord(patch)) return target
  const merge = (dst: Record<string, unknown>, src: Record<string, unknown>) => {
    for (const [key, cur] of Object.entries(dst)) {
      const next = src[key]
      if (next === undefined) continue
      if (isRecord(cur)) { if (isRecord(next)) merge(cur, next) } else if (Array.isArray(cur)) {
        if (Array.isArray(next) && next.length === cur.length && next.every(n => typeof n === 'number' && Number.isFinite(n))) dst[key] = [...next]
      } else if (typeof next === typeof cur && (typeof next !== 'number' || Number.isFinite(next))) dst[key] = next
    }
  }
  merge(target as unknown as Record<string, unknown>, patch)
  return target
}

const smooth = (k: number) => {
  const c = Math.min(1, Math.max(0, k))
  return c * c * (3 - 2 * c)
}

/** Seberapa malam (0 = siang penuh, 1 = malam penuh); berubah halus selama fajar & senja. */
export function nightFactor(hour: number, theme: OfficeTheme = OFFICE_THEME): number {
  const [d0, d1] = theme.hours.dawn
  const [k0, k1] = theme.hours.dusk
  if (hour < d0 || hour >= k1) return 1
  if (hour < d1) return 1 - smooth((hour - d0) / (d1 - d0))
  if (hour < k0) return 0
  return smooth((hour - k0) / (k1 - k0))
}

export type DayPhase = 'day' | 'dawn' | 'dusk' | 'night'

/** Fase hari untuk label HUD. */
export function dayPhase(hour: number, theme: OfficeTheme = OFFICE_THEME): DayPhase {
  const [d0, d1] = theme.hours.dawn
  const [k0, k1] = theme.hours.dusk
  if (hour < d0 || hour >= k1) return 'night'
  if (hour < d1) return 'dawn'
  return hour < k0 ? 'day' : 'dusk'
}

export const DAY_PHASE_LABEL: Record<DayPhase, string> = { day: 'Siang', dawn: 'Fajar', dusk: 'Senja', night: 'Malam' }

export const hourOf = (date: Date) => date.getHours() + date.getMinutes() / 60

/** Mode waktu scene: mengikuti jam nyata, atau dipaksa siang/malam untuk melihat suasananya. */
export type TimeMode = 'auto' | 'day' | 'night'
export const TIME_MODES: Array<{ key: TimeMode, label: string, icon: string, title: string }> = [
  { key: 'auto', label: 'Otomatis', icon: 'i-lucide-clock', title: 'Suasana mengikuti jam sekarang' },
  { key: 'day', label: 'Siang', icon: 'i-lucide-sun', title: 'Tampilkan kantor pada siang hari' },
  { key: 'night', label: 'Malam', icon: 'i-lucide-moon', title: 'Tampilkan kantor pada malam hari (lampu menyala)' }
]

/** Waktu yang dipakai scene untuk mode tertentu (siang = 12.00, malam = 21.00 pada tanggal yang sama). */
export function sceneDate(nowMs: number, mode: TimeMode): Date {
  const d = new Date(nowMs)
  if (mode === 'day') d.setHours(12, 0, 0, 0)
  else if (mode === 'night') d.setHours(21, 0, 0, 0)
  return d
}
