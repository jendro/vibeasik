import type { StaffDivision } from '~/types'

/**
 * Denah Kampus Kantor Agent (unit logis = 1 cm; scene 3D memetakan x → X, y → Z, f → lantai).
 * Gedung kantor vibeasik (0..BUILDING_W × 0..BUILDING_D) bertingkat tiga dengan koridor tengah dan lift di ujung barat:
 *   Lantai 1 (f=0): Bisnis & Operasional, Pantry, Lobi + Loket LLM, Perpustakaan, Lounge pintu taman.
 *   Lantai 2 (f=1): Produk & Desain, Quality & Security, Engineering, Platform & Data.
 *   Lantai 3 (f=2): Ruang Eksekutif, Ruang Rapat, Ruang PM, Teras atap.
 * Di luar gedung (f=0): Taman di selatan (rehat pegawai) dan Peta Luar di timur (Kafe Lepas agent luar,
 * gedung Model LLM tujuan kurir, jalan). Perjalanan antar-ruang dirutekan lewat pintu, koridor, dan lift (`waypoints`).
 */
export const BUILDING_W = 1200
export const BUILDING_D = 820
export const FLOORS = 3
/** Tinggi satu lantai (unit logis). */
export const FLOOR_H = 320
/** Batas peta (luar + taman). */
export const MAP_W = 2300
export const MAP_D = 1500
/** Garis tengah koridor tiap lantai; ruang utara di atasnya, ruang selatan di bawahnya. */
export const CORRIDOR_Y = 420
export const CORRIDOR_HALF = 40
/** Lebar inti gedung (tangga & lift) di sisi barat. */
export const CORE_W = 140
/** Rasio tampilan scene 3D. */
export const SCENE_RATIO = 16 / 9

export interface Spot {
  x: number
  y: number
  /** Lantai (0 = lantai dasar & luar gedung). */
  f?: number
  /** Arah hadap saat diam di titik ini (radian, 0 = menghadap selatan/kamera, π = utara). */
  face?: number
}
export interface Rect { x: number, y: number, w: number, h: number }

export const FACE_S = 0
export const FACE_N = Math.PI
export const FACE_E = Math.PI / 2
export const FACE_W = -Math.PI / 2

const fl = (s: Spot) => s.f ?? 0
export const sameSpot = (a: Spot, b: Spot) => a.x === b.x && a.y === b.y && fl(a) === fl(b)
/** Jarak tempuh (lift dihitung per lantai). */
export const spotDist = (a: Spot, b: Spot) => Math.hypot(a.x - b.x, a.y - b.y) + Math.abs(fl(a) - fl(b)) * FLOOR_H

export type OfficeAreaKey = 'outside' | 'garden' | 'floor1' | 'floor2' | 'floor3'
export interface OfficeArea { key: OfficeAreaKey, label: string, short: string, icon: string, floor?: number }
/** Area yang bisa dipilih di scene; hanya area terpilih yang digambar penuh. */
export const AREAS: OfficeArea[] = [
  { key: 'outside', label: 'Peta Luar', short: 'Luar', icon: 'i-lucide-map' },
  { key: 'garden', label: 'Taman', short: 'Taman', icon: 'i-lucide-trees' },
  { key: 'floor1', label: 'Lantai 1 · Lobi & Layanan', short: 'Lt 1', icon: 'i-lucide-building', floor: 0 },
  { key: 'floor2', label: 'Lantai 2 · Ruang Kerja', short: 'Lt 2', icon: 'i-lucide-building-2', floor: 1 },
  { key: 'floor3', label: 'Lantai 3 · Eksekutif & Rapat', short: 'Lt 3', icon: 'i-lucide-crown', floor: 2 }
]

export const inBuilding = (p: Spot) => p.x >= 0 && p.x <= BUILDING_W && p.y >= 0 && p.y <= BUILDING_D
/** Area tempat sebuah titik berada (untuk badge jumlah orang & fokus kamera). */
export function areaOf(p: Spot): OfficeAreaKey {
  if (inBuilding(p) && !(fl(p) === 0 && p.x === BUILDING_W)) return (['floor1', 'floor2', 'floor3'] as const)[Math.min(FLOORS - 1, Math.max(0, Math.round(fl(p))))]!
  return p.x < BUILDING_W ? 'garden' : 'outside'
}

export type RoomKind = 'division' | 'pantry' | 'lobby' | 'library' | 'lounge' | 'meeting' | 'pm' | 'terrace' | 'garden' | 'outside'
export interface Room {
  key: string
  label: string
  kind: RoomKind
  f: number
  rect: Rect
  /** Pintu ruang (di tepi koridor untuk ruang dalam gedung; titik masuk untuk area luar). */
  door: Spot
  floor: string
}

const N_Y = 0
const N_H = CORRIDOR_Y - CORRIDOR_HALF
const S_Y = CORRIDOR_Y + CORRIDOR_HALF
const S_H = BUILDING_D - S_Y
const north = (x: number, w: number) => ({ x, y: N_Y, w, h: N_H })
const south = (x: number, w: number) => ({ x, y: S_Y, w, h: S_H })
const nDoor = (x: number, f: number): Spot => ({ x, y: N_H, f })
const sDoor = (x: number, f: number): Spot => ({ x, y: S_Y, f })

/** Pintu utama gedung (dinding timur lantai 1) dan anak tangga di luarnya. */
export const DOOR: Spot = { x: BUILDING_W, y: CORRIDOR_Y, f: 0 }
export const DOOR_OUT: Spot = { x: BUILDING_W + 70, y: CORRIDOR_Y, f: 0 }
/** Pintu taman di Lounge lantai 1 (dinding selatan). */
export const GARDEN_DOOR: Spot = { x: 920, y: BUILDING_D, f: 0 }
export const GARDEN_OUT: Spot = { x: 920, y: BUILDING_D + 70, f: 0 }
/** Sudut tenggara gedung: jalan setapak taman ↔ halaman depan. */
const OUT_CORNER: Spot = { x: BUILDING_W + 70, y: BUILDING_D + 70, f: 0 }
/** Jalan raya: pegawai baru & agent luar datang/pulang lewat sini. */
export const STREET: Spot = { x: MAP_W - 20, y: 640, f: 0 }
/** Lift di ujung barat koridor (titik yang sama di setiap lantai). */
export const liftSpot = (f: number): Spot => ({ x: 70, y: CORRIDOR_Y, f })

export const ROOMS: Room[] = [
  // Lantai 1
  { key: 'business', label: 'BISNIS & OPERASIONAL', kind: 'division', f: 0, rect: north(CORE_W, 500), door: nDoor(390, 0), floor: '#e7efe9' },
  { key: 'pantry', label: 'PANTRY', kind: 'pantry', f: 0, rect: north(640, 260), door: nDoor(770, 0), floor: '#f3e3cf' },
  { key: 'lobby', label: 'LOBI · LOKET LLM', kind: 'lobby', f: 0, rect: north(900, 300), door: nDoor(1050, 0), floor: '#e5e7f5' },
  { key: 'library', label: 'PERPUSTAKAAN', kind: 'library', f: 0, rect: south(CORE_W, 500), door: sDoor(390, 0), floor: '#efe6da' },
  { key: 'lounge', label: 'LOUNGE · PINTU TAMAN', kind: 'lounge', f: 0, rect: south(640, 560), door: sDoor(920, 0), floor: '#e8eef2' },
  // Lantai 2
  { key: 'product', label: 'PRODUK & DESAIN', kind: 'division', f: 1, rect: north(CORE_W, 520), door: nDoor(400, 1), floor: '#f6e6ee' },
  { key: 'quality', label: 'QUALITY & SECURITY', kind: 'division', f: 1, rect: north(660, 540), door: nDoor(930, 1), floor: '#ece6f6' },
  { key: 'engineering', label: 'ENGINEERING', kind: 'division', f: 1, rect: south(CORE_W, 660), door: sDoor(470, 1), floor: '#e3ebf7' },
  { key: 'platform', label: 'PLATFORM & DATA', kind: 'division', f: 1, rect: south(800, 400), door: sDoor(960, 1), floor: '#f6eadf' },
  // Lantai 3
  { key: 'executive', label: 'RUANG EKSEKUTIF', kind: 'division', f: 2, rect: north(CORE_W, 500), door: nDoor(390, 2), floor: '#f5ecd9' },
  { key: 'meeting', label: 'RUANG RAPAT', kind: 'meeting', f: 2, rect: north(640, 560), door: nDoor(700, 2), floor: '#ede5f5' },
  { key: 'pm', label: 'RUANG PROJECT MANAGER', kind: 'pm', f: 2, rect: south(CORE_W, 500), door: sDoor(390, 2), floor: '#e9edf2' },
  { key: 'terrace', label: 'TERAS ATAP', kind: 'terrace', f: 2, rect: south(640, 560), door: sDoor(920, 2), floor: '#d9e8d2' },
  // Luar gedung
  { key: 'garden', label: 'TAMAN', kind: 'garden', f: 0, rect: { x: 0, y: BUILDING_D, w: BUILDING_W, h: MAP_D - BUILDING_D }, door: GARDEN_OUT, floor: '' },
  { key: 'outside', label: 'HALAMAN', kind: 'outside', f: 0, rect: { x: BUILDING_W, y: 0, w: MAP_W - BUILDING_W, h: MAP_D }, door: DOOR_OUT, floor: '' }
]

const inRect = (p: Spot, r: Rect) => p.x >= r.x && p.x <= r.x + r.w && p.y >= r.y && p.y <= r.y + r.h
export const roomAt = (p: Spot) => ROOMS.find(r => r.f === fl(p) && inRect(p, r.rect))
export const roomByKey = (key: string) => ROOMS.find(r => r.key === key)

export interface DivisionZone {
  key: StaffDivision
  label: string
  room: Room
  /** Titik kaki pegawai yang duduk menghadap meja di selatannya (wajah ke kamera). Urutan = nomor kursi (`desk`). */
  desks: Spot[]
}

const deskRow = (xs: number[], y: number, f: number) => xs.map(x => ({ x, y, f, face: FACE_S }))
const zone = (key: StaffDivision, desks: Spot[]): DivisionZone => {
  const room = roomByKey(key)!
  return { key, label: room.label, room, desks }
}

export const ZONES: DivisionZone[] = [
  zone('business', [...deskRow([250, 390, 530], 110, 0), ...deskRow([250, 390, 530], 250, 0)]),
  zone('product', [...deskRow([230, 350, 470, 590], 110, 1), ...deskRow([290, 410, 530], 250, 1)]),
  zone('quality', [...deskRow([740, 860, 980, 1100], 110, 1), ...deskRow([800, 920, 1040], 250, 1)]),
  zone('engineering', [...deskRow([220, 340, 460, 580, 700], 540, 1), ...deskRow([220, 340, 460, 580, 700], 680, 1)]),
  zone('platform', [...deskRow([880, 1000], 540, 1), ...deskRow([880, 1000], 680, 1)]),
  zone('executive', deskRow([230, 350, 470, 590], 140, 2))
]

export const zoneOf = (division: string) => ZONES.find(z => z.key === division)

/** Kursi pegawai; null bila kursi divisinya sudah penuh (pegawai menumpang di pantry). */
export function deskSpot(division: string, desk: number): Spot | null {
  return zoneOf(division)?.desks[desk] ?? null
}

/** Pantry lantai 1: obrolan santai, kurir gagal menunggu diperiksa, pegawai yang kehabisan kursi. */
export const LOUNGE_SPOTS: Spot[] = [
  ...[700, 770, 840].map(x => ({ x, y: 230, f: 0, face: FACE_S })),
  ...[700, 770, 840].map(x => ({ x, y: 320, f: 0, face: FACE_N }))
]

/** Loket LLM di lobi: kurir menerima surat menghadap meja loket. */
export const COUNTER_SLOTS: Spot[] = [960, 1020, 1080, 1140].map(x => ({ x, y: 200, f: 0, face: FACE_N }))
/** Menunggu di ujung loket bila semua slot terisi. */
export const COUNTER_WAIT: Spot = { x: 1130, y: 310, f: 0, face: FACE_N }

/** Perpustakaan: pegawai yang mengerjakan agenda pribadi (status `studying`) membaca di sini. */
export const LIBRARY_SEATS: Spot[] = [230, 290, 350, 430, 490, 550].map(x => ({ x, y: 600, f: 0, face: FACE_S }))

/** Taman: sebagian pegawai yang rehat memilih duduk di bangku taman. */
export const GARDEN_SPOTS: Spot[] = [
  { x: 260, y: 1080, f: 0, face: FACE_S }, { x: 330, y: 1080, f: 0, face: FACE_S },
  { x: 560, y: 1290, f: 0, face: FACE_N }, { x: 630, y: 1290, f: 0, face: FACE_N },
  { x: 860, y: 1110, f: 0, face: FACE_W }, { x: 860, y: 1180, f: 0, face: FACE_W }
]
/** Pegawai rehat bergantian antara pantry dan taman (stabil per pegawai). */
export const breaksInGarden = (employeeId: number) => employeeId % 2 === 1

/** Ruang Rapat lantai 3. Peserta duduk menghadap meja di selatannya (wajah terlihat kamera). */
export const MEETING_TABLE: Spot = { x: 920, y: 210, f: 2 }
export const MEETING_SEATS: Spot[] = [790, 855, 920, 985, 1050].map(x => ({ x, y: 150, f: 2, face: FACE_S }))
export const MEETING_EXTRA: Spot = { x: 1130, y: 300, f: 2, face: FACE_W }
export const PM_PODIUM: Spot = { x: 700, y: 210, f: 2, face: FACE_E }
/** Meja PM di Ruang Project Manager. */
export const PM_DESK: Spot = { x: 390, y: 600, f: 2, face: FACE_S }
/** Pintu Ruang Rapat: tim perencana tanpa pegawai muncul/pulang di sini. */
export const PM_DOOR: Spot = roomByKey('meeting')!.door

/** Kafe pekerja lepas (halaman tenggara): meja bundar, dua kursi per meja. */
export const CAFE_RECT: Rect = { x: 1340, y: 760, w: 520, h: 330 }
export const CAFE_TABLES: Spot[] = [{ x: 1480, y: 910 }, { x: 1720, y: 910 }, { x: 1480, y: 1030 }, { x: 1720, y: 1030 }]
export const CAFE_SEATS: Spot[] = CAFE_TABLES.flatMap(t => [{ x: t.x - 42, y: t.y, f: 0, face: FACE_E }, { x: t.x + 42, y: t.y, f: 0, face: FACE_W }])
/** Antre di depan barista bila semua kursi kafe terisi. */
export const CAFE_EXTRA: Spot[] = [1540, 1600, 1660].map(x => ({ x, y: 830, f: 0, face: FACE_N }))

/** Gedung Model LLM (utara halaman, di belakang agar tidak menutupi kamera); kurir menunggu jawaban di plaza depannya. */
export const LLM_TOWER: Rect = { x: 1440, y: 30, w: 380, h: 290 }
export const OUTSIDE_SPOTS: Spot[] = [
  ...[1530, 1630, 1730].map(x => ({ x, y: 400, f: 0, face: FACE_N })),
  ...[1580, 1680, 1780].map(x => ({ x, y: 460, f: 0, face: FACE_N }))
]
/** Jalan raya di tepi timur peta. */
export const ROAD_X = 2080
export const ROAD_W = 170

// ---------- Rute: graf pintu, koridor, lift ----------

interface Node { id: string, spot: Spot, next: Set<string> }
const nodes = new Map<string, Node>()
const node = (id: string, spot: Spot) => {
  if (!nodes.has(id)) nodes.set(id, { id, spot, next: new Set() })
  return id
}
const link = (a: string, b: string) => { nodes.get(a)!.next.add(b); nodes.get(b)!.next.add(a) }
const roomNode = new Map<string, string>()

for (let f = 0; f < FLOORS; f++) {
  const xs = new Set<number>([liftSpot(f).x])
  const doors = ROOMS.filter(r => r.f === f && inBuilding(r.door) && r.kind !== 'garden' && r.kind !== 'outside')
  doors.forEach(r => xs.add(r.door.x))
  if (f === 0) xs.add(BUILDING_W - 20)
  const corridor = [...xs].sort((a, b) => a - b).map(x => node(`c${f}:${x}`, { x, y: CORRIDOR_Y, f }))
  corridor.slice(1).forEach((id, i) => link(corridor[i]!, id))
  for (const r of doors) {
    roomNode.set(r.key, node(`d:${r.key}`, r.door))
    link(`d:${r.key}`, `c${f}:${r.door.x}`)
  }
  node(`lift${f}`, liftSpot(f))
  link(`lift${f}`, `c${f}:${liftSpot(f).x}`)
  if (f > 0) link(`lift${f - 1}`, `lift${f}`)
}
link(`c0:${BUILDING_W - 20}`, node('door', DOOR))
link('door', node('door-out', DOOR_OUT))
link('d:lounge', node('garden-door', GARDEN_DOOR))
link('garden-door', node('garden-out', GARDEN_OUT))
link('garden-out', node('corner', OUT_CORNER))
link('corner', 'door-out')
roomNode.set('garden', 'garden-out')
roomNode.set('outside', 'door-out')
// Lift berdiri di koridor; titik lain di koridor masuk lewat titik koridor terdekat.
const nodeAt = (p: Spot) => [...nodes.values()].find(n => sameSpot(n.spot, p))?.id

function entryNode(p: Spot): string | undefined {
  const exact = nodeAt(p)
  if (exact) return exact
  const room = roomAt(p)
  if (room) return roomNode.get(room.key)
  // Di koridor/inti gedung: titik koridor terdekat di lantainya.
  let best: Node | undefined
  for (const n of nodes.values()) {
    if (fl(n.spot) !== fl(p) || !inBuilding(n.spot) || n.spot.y !== CORRIDOR_Y) continue
    if (!best || spotDist(n.spot, p) < spotDist(best.spot, p)) best = n
  }
  return best?.id
}

function shortest(from: string, to: string): string[] {
  const dist = new Map<string, number>([[from, 0]])
  const prev = new Map<string, string>()
  const open = new Set([from])
  while (open.size) {
    let cur = ''
    for (const id of open) if (!cur || dist.get(id)! < dist.get(cur)!) cur = id
    open.delete(cur)
    if (cur === to) break
    for (const nb of nodes.get(cur)!.next) {
      const d = dist.get(cur)! + spotDist(nodes.get(cur)!.spot, nodes.get(nb)!.spot)
      if (d < (dist.get(nb) ?? Infinity)) { dist.set(nb, d); prev.set(nb, cur); open.add(nb) }
    }
  }
  if (!dist.has(to)) return []
  const path = [to]
  while (path[0] !== from) path.unshift(prev.get(path[0]!)!)
  return path
}

/** Titik antara agar perjalanan tidak menembus dinding/lantai: lewat pintu ruang, koridor, lift, dan pintu gedung. */
export function waypoints(from: Spot, to: Spot): Spot[] {
  const rf = roomAt(from)
  const rt = roomAt(to)
  if (rf && rf === rt && !nodeAt(from) && !nodeAt(to)) return []
  const a = entryNode(from)
  const b = entryNode(to)
  if (!a || !b || a === b) return []
  return shortest(a, b)
    .map(id => nodes.get(id)!.spot)
    .filter(s => !sameSpot(s, from) && !sameSpot(s, to))
}
