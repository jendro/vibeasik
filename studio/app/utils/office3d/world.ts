import * as THREE from 'three'
import {
  BUILDING_D, BUILDING_W, CAFE_RECT, CAFE_SEATS, CAFE_TABLES, CORE_W, CORRIDOR_HALF, CORRIDOR_Y, COUNTER_SLOTS, DOOR, FACE_E, FACE_N,
  FACE_S, FACE_W, FLOORS, FLOOR_H, GARDEN_DOOR, GARDEN_SPOTS, LIBRARY_SEATS, LLM_TOWER, LOUNGE_SPOTS, MAP_D, MAP_W, MEETING_SEATS, MEETING_TABLE,
  PM_DESK, PM_PODIUM, ROAD_W, ROAD_X, ROOMS, ZONES, type Room, type Spot
} from '~/utils/officeLayout'
import type { PropName } from './assets'
import { canvasTexture, drawScreen, textTexture, tileTexture } from './textures'

/** 1 unit logis denah = 1 cm. */
export const S = 0.01
export const FH = FLOOR_H * S
const WALL_H = 2.8
const LOW_WALL = 0.5
const PART_H = 1.15
/** Jendela kaca: ambang bawah, tinggi (meter) & lebar (unit denah), sama untuk dinding dalam & fasad. */
const WIN_Y0 = 0.85
const WIN_H = 1.3
const WIN_W = 90
/** Titik tengah jendela berjarak 120 sepanjang dinding `len`, rata tengah. */
const winCenters = (len: number) => {
  const n = Math.floor((len - 60) / 120)
  const start = (len - (n - 1) * 120) / 2
  return Array.from({ length: n }, (_, i) => start + i * 120)
}

export const toWorld = (x: number, y: number, f = 0, h = 0) => new THREE.Vector3(x * S, f * FH + h, y * S)

export interface DeskView {
  key: string
  division: string
  index: number
  spot: Spot
  leader: boolean
  chairMat: THREE.MeshStandardMaterial
  screen: { ctx: CanvasRenderingContext2D, texture: THREE.CanvasTexture, mat: THREE.MeshBasicMaterial }
  plate: { ctx: CanvasRenderingContext2D, texture: THREE.CanvasTexture, mesh: THREE.Mesh }
  sig: string
}

export interface World {
  root: THREE.Group
  ground: THREE.Group
  floors: THREE.Group[]
  shells: THREE.Group[]
  roof: THREE.Group
  desks: DeskView[]
  cafeLaptops: THREE.Group[]
  meetingGlow: THREE.MeshStandardMaterial
  counterGlow: THREE.MeshStandardMaterial
  sun: THREE.DirectionalLight
  hemi: THREE.HemisphereLight
  /** Layar TV Papan Keputusan di Ruang Eksekutif (digambar ulang dari data rapat). */
  decisionBoard: { ctx: CanvasRenderingContext2D, texture: THREE.CanvasTexture }
  setTime: (date: Date, busyFloors: boolean[]) => void
  update: (t: number) => void
  dispose: () => void
}

/** Penyimpan material/geometri bersama + pembuat bentuk dasar (satuan meter). */
class Kit {
  private mats = new Map<string, THREE.MeshStandardMaterial>()
  private geos = new Map<string, THREE.BufferGeometry>()
  owned: Array<{ dispose: () => void }> = []

  m(color: string, opts: THREE.MeshStandardMaterialParameters = {}) {
    const key = color + JSON.stringify(opts)
    let mat = this.mats.get(key)
    if (!mat) {
      mat = new THREE.MeshStandardMaterial({ color: new THREE.Color().setStyle(color, THREE.SRGBColorSpace), roughness: 0.8, ...opts })
      this.mats.set(key, mat)
    }
    return mat
  }

  /** Material unik (tidak dibagi) — untuk warna/kilau yang berubah per objek. */
  own<T extends { dispose: () => void }>(o: T): T {
    this.owned.push(o)
    return o
  }

  private geo(key: string, make: () => THREE.BufferGeometry) {
    let g = this.geos.get(key)
    if (!g) { g = make(); this.geos.set(key, g) }
    return g
  }

  box(parent: THREE.Object3D, w: number, h: number, d: number, mat: THREE.Material, x = 0, y = 0, z = 0, shadow = true) {
    const mesh = new THREE.Mesh(this.geo(`b${w}|${h}|${d}`, () => new THREE.BoxGeometry(w, h, d)), mat)
    mesh.position.set(x, y, z)
    mesh.castShadow = shadow
    mesh.receiveShadow = true
    parent.add(mesh)
    return mesh
  }

  cyl(parent: THREE.Object3D, rt: number, rb: number, h: number, mat: THREE.Material, x = 0, y = 0, z = 0, seg = 16) {
    const mesh = new THREE.Mesh(this.geo(`c${rt}|${rb}|${h}|${seg}`, () => new THREE.CylinderGeometry(rt, rb, h, seg)), mat)
    mesh.position.set(x, y, z)
    mesh.castShadow = true
    mesh.receiveShadow = true
    parent.add(mesh)
    return mesh
  }

  ball(parent: THREE.Object3D, r: number, mat: THREE.Material, x = 0, y = 0, z = 0, detail = 1) {
    const mesh = new THREE.Mesh(this.geo(`i${r}|${detail}`, () => new THREE.IcosahedronGeometry(r, detail)), mat)
    mesh.position.set(x, y, z)
    mesh.castShadow = true
    parent.add(mesh)
    return mesh
  }

  /** Bidang datar di lantai (tidak memberi bayangan). */
  plane(parent: THREE.Object3D, w: number, d: number, mat: THREE.Material, x: number, y: number, z: number) {
    const mesh = new THREE.Mesh(this.geo(`p${w}|${d}`, () => new THREE.PlaneGeometry(w, d)), mat)
    mesh.rotation.x = -Math.PI / 2
    mesh.position.set(x, y, z)
    mesh.receiveShadow = true
    parent.add(mesh)
    return mesh
  }

  dispose() {
    this.mats.forEach(m => m.dispose())
    this.geos.forEach(g => g.dispose())
    this.owned.forEach(o => o.dispose())
  }
}

/** Grup perabot bertanda `prop` (bisa diganti aset .glb), diletakkan di titik denah dengan arah hadap. */
function prop(name: PropName, parent: THREE.Object3D, x: number, y: number, yaw = 0, h = 0) {
  const g = new THREE.Group()
  g.userData.prop = name
  g.position.set(x * S, h, y * S)
  g.rotation.y = yaw
  parent.add(g)
  return g
}

const WOOD = '#b07a4f'
const WOOD_DARK = '#7a5232'

/** Material & daftar yang dipakai bersama oleh pembangun bagian dunia (dispose lewat `Kit`). */
interface WorldState {
  lampMat: THREE.MeshStandardMaterial
  windowMats: THREE.MeshStandardMaterial[]
  ledMats: THREE.MeshStandardMaterial[]
  cars: THREE.Group[]
  desks: DeskView[]
  cafeLaptops: THREE.Group[]
  meetingGlow: THREE.MeshStandardMaterial
  counterGlow: THREE.MeshStandardMaterial
  tileRepeat: (rx: number, ry: number) => THREE.Texture
  decisionBoard?: World['decisionBoard']
}

const W = MAP_W * S
const D = MAP_D * S
const WEST_WINS = winCenters(BUILDING_D).filter(c => Math.abs(c - CORRIDOR_Y) > 110)

function createWorldState(k: Kit): WorldState {
  const lampMat = k.own(new THREE.MeshStandardMaterial({ color: '#fef3c7', emissive: '#fde68a', emissiveIntensity: 0.2 }))
  const tiles = k.own(tileTexture())
  const tileRepeat = (rx: number, ry: number) => {
    const t = k.own(tiles.clone())
    t.repeat.set(rx, ry)
    t.needsUpdate = true
    return t
  }
  const glowStrip = (color: string) => k.own(new THREE.MeshStandardMaterial({ color, emissive: color, emissiveIntensity: 0.15 }))
  const meetingGlow = glowStrip('#a855f7')
  const counterGlow = glowStrip('#6366f1')
  return { lampMat, windowMats: [], ledMats: [], cars: [], desks: [], cafeLaptops: [], meetingGlow, counterGlow, tileRepeat }
}

function createLabel(k: Kit) {
  const label = (parent: THREE.Object3D, text: string, x: number, y: number, h: number, height = 0.32, opts: Parameters<typeof textTexture>[1] = {}, flat = true) => {
    const t = textTexture(text, opts)
    k.own(t.texture)
    const mat = k.own(new THREE.MeshBasicMaterial({ map: t.texture, transparent: true, depthWrite: false }))
    const mesh = new THREE.Mesh(k.own(new THREE.PlaneGeometry(height * t.aspect, height)), mat)
    if (flat) mesh.rotation.x = -Math.PI / 2
    mesh.position.set(x * S, h, y * S)
    parent.add(mesh)
    return mesh
  }
  return label
}

/** Meja kerja, kursi, monitor, rak, papan tulis, TV: perabot ruang kerja. */
function createOfficeProps(k: Kit, st: WorldState) {
  const { ledMats } = st
  const desk = (parent: THREE.Object3D, x: number, y: number, yaw: number, w = 1.0, leader = false) => {
    const g = prop('desk', parent, x, y, yaw)
    const top = k.m(leader ? '#8b5a2b' : WOOD)
    k.box(g, w, 0.05, 0.6, top, 0, 0.72, 0)
    for (const sx of [-1, 1]) k.box(g, 0.05, 0.7, 0.55, k.m(leader ? '#5b3a1e' : WOOD_DARK), sx * (w / 2 - 0.04), 0.35, 0)
    k.box(g, w - 0.1, 0.35, 0.03, k.m(leader ? '#5b3a1e' : WOOD_DARK), 0, 0.5, -0.27)
    if (leader) k.box(g, w + 0.02, 0.02, 0.62, k.m('#f59e0b', { metalness: 0.6, roughness: 0.3 }), 0, 0.745, 0)
    return g
  }
  const chair = (parent: THREE.Object3D, x: number, y: number, yaw: number, mat?: THREE.Material, tall = false) => {
    const g = prop('chair', parent, x, y, yaw)
    const seat = mat ?? k.m('#475569')
    k.box(g, 0.44, 0.08, 0.44, seat, 0, 0.45, 0)
    k.box(g, 0.44, tall ? 0.75 : 0.5, 0.07, seat, 0, tall ? 0.86 : 0.73, -0.21)
    k.cyl(g, 0.03, 0.03, 0.4, k.m('#334155'), 0, 0.22, 0)
    k.box(g, 0.5, 0.04, 0.08, k.m('#1f2937'), 0, 0.03, 0)
    k.box(g, 0.08, 0.04, 0.5, k.m('#1f2937'), 0, 0.03, 0)
    return g
  }
  const monitor = (parent: THREE.Object3D, x: number, y: number, yaw: number, h = 0.745) => {
    const g = new THREE.Group()
    g.position.set(x * S, h, y * S)
    g.rotation.y = yaw
    parent.add(g)
    k.box(g, 0.2, 0.02, 0.14, k.m('#334155'), 0, 0.01, 0)
    k.box(g, 0.04, 0.16, 0.04, k.m('#334155'), 0, 0.09, -0.02)
    k.box(g, 0.64, 0.4, 0.03, k.m('#1e293b', { roughness: 0.4 }), 0, 0.36, 0)
    const sc = canvasTexture(160, 100, ctx => drawScreen(ctx, { active: false, burning: false, project: '', seed: 1 }))
    k.own(sc.texture)
    const mat = k.own(new THREE.MeshBasicMaterial({ map: sc.texture }))
    const screen = new THREE.Mesh(k.own(new THREE.PlaneGeometry(0.6, 0.36)), mat)
    screen.position.set(0, 0.36, 0.017)
    g.add(screen)
    return { g, ctx: sc.ctx, texture: sc.texture, mat }
  }
  const bookshelf = (parent: THREE.Object3D, x: number, y: number, yaw: number, w = 1.2, h = 1.9) => {
    const g = prop('bookshelf', parent, x, y, yaw)
    const wood = k.m(WOOD_DARK)
    k.box(g, w, h, 0.32, wood, 0, h / 2, 0)
    const colors = ['#ef4444', '#f59e0b', '#22c55e', '#3b82f6', '#a855f7', '#14b8a6', '#ec4899', '#64748b']
    const rows = Math.floor(h / 0.4)
    for (let r = 0; r < rows; r++) {
      let bx = -w / 2 + 0.08
      let i = r * 3
      while (bx < w / 2 - 0.12) {
        const bw = 0.05 + ((i * 7) % 4) * 0.015
        k.box(g, bw, 0.28, 0.24, k.m(colors[i % colors.length]!), bx + bw / 2, 0.2 + r * 0.4, 0.06, false)
        bx += bw + 0.01
        i++
      }
    }
    return g
  }
  const serverRack = (parent: THREE.Object3D, x: number, y: number, yaw: number) => {
    const g = prop('server_rack', parent, x, y, yaw)
    k.box(g, 0.6, 1.7, 0.8, k.m('#1f2937', { roughness: 0.5, metalness: 0.3 }), 0, 0.85, 0)
    const led = k.own(new THREE.MeshStandardMaterial({ color: '#22c55e', emissive: '#22c55e', emissiveIntensity: 1 }))
    ledMats.push(led)
    for (let i = 0; i < 7; i++) k.box(g, 0.45, 0.02, 0.01, i % 3 ? led : k.m('#38bdf8', { emissive: '#38bdf8', emissiveIntensity: 0.8 }), 0, 0.3 + i * 0.2, 0.405, false)
    return g
  }
  const tv = (parent: THREE.Object3D, x: number, y: number, yaw: number, h = 1.3, w = 1.4) => {
    const g = prop('tv', parent, x, y, yaw, h)
    k.box(g, w, w * 0.58, 0.05, k.m('#0f172a', { roughness: 0.2 }), 0, 0, 0)
    k.box(g, w - 0.08, w * 0.58 - 0.08, 0.01, k.m('#1d4ed8', { emissive: '#1e40af', emissiveIntensity: 0.6 }), 0, 0, 0.03, false)
    return g
  }
  const whiteboard = (parent: THREE.Object3D, x: number, y: number, yaw: number, w = 1.6, h = 1.3) => {
    const g = new THREE.Group()
    g.position.set(x * S, h, y * S)
    g.rotation.y = yaw
    parent.add(g)
    k.box(g, w, 0.9, 0.04, k.m('#f8fafc'), 0, 0, 0)
    const ink = ['#2563eb', '#dc2626', '#16a34a']
    for (let i = 0; i < 4; i++) k.box(g, w * (0.3 + (i % 3) * 0.15), 0.03, 0.01, k.m(ink[i % 3]!), -w * 0.2 + (i % 2) * 0.1, 0.25 - i * 0.16, 0.025, false)
    for (let i = 0; i < 3; i++) k.box(g, 0.14, 0.14, 0.01, k.m(['#fde047', '#f9a8d4', '#86efac'][i]!), w * 0.3, 0.25 - i * 0.2, 0.025, false)
    return g
  }
  return { desk, chair, monitor, bookshelf, serverRack, tv, whiteboard }
}

/** Perabot ruang santai: tanaman, sofa, bangku, meja bundar, kursi kafe, konter, mesin kopi, kulkas. */
function createLoungeProps(k: Kit) {
  const plant = (parent: THREE.Object3D, x: number, y: number, s = 1) => {
    const g = prop('plant', parent, x, y)
    k.cyl(g, 0.16 * s, 0.12 * s, 0.32 * s, k.m('#e7e5e4'), 0, 0.16 * s, 0)
    const leaf = k.m('#3f9d4a')
    k.ball(g, 0.24 * s, leaf, 0, 0.52 * s, 0)
    k.ball(g, 0.17 * s, k.m('#58b45e'), 0.1 * s, 0.72 * s, 0.05 * s)
    return g
  }
  const sofa = (parent: THREE.Object3D, x: number, y: number, yaw: number, w = 1.6, color = '#f97316') => {
    const g = prop('sofa', parent, x, y, yaw)
    const c = k.m(color)
    k.box(g, w, 0.4, 0.75, c, 0, 0.2, 0)
    k.box(g, w, 0.45, 0.18, c, 0, 0.6, -0.3)
    for (const sx of [-1, 1]) k.box(g, 0.16, 0.25, 0.75, c, sx * (w / 2 - 0.08), 0.52, 0)
    return g
  }
  const bench = (parent: THREE.Object3D, x: number, y: number, yaw: number, w = 1.5) => {
    const g = prop('bench', parent, x, y, yaw)
    const wood = k.m('#a0693d')
    for (let i = 0; i < 3; i++) k.box(g, w, 0.04, 0.12, wood, 0, 0.44, -0.15 + i * 0.14)
    k.box(g, w, 0.3, 0.04, wood, 0, 0.72, -0.24)
    for (const sx of [-1, 1]) k.box(g, 0.06, 0.44, 0.42, k.m('#374151'), sx * (w / 2 - 0.1), 0.22, -0.02)
    return g
  }
  const roundTable = (parent: THREE.Object3D, x: number, y: number, r = 0.45, h = 0.74) => {
    const g = prop('round_table', parent, x, y)
    k.cyl(g, r, r, 0.05, k.m('#f5f5f4'), 0, h, 0, 24)
    k.cyl(g, 0.05, 0.05, h, k.m('#57534e'), 0, h / 2, 0, 8)
    k.cyl(g, 0.25, 0.25, 0.03, k.m('#57534e'), 0, 0.015, 0, 16)
    return g
  }
  const cafeChair = (parent: THREE.Object3D, x: number, y: number, yaw: number) => {
    const g = prop('cafe_chair', parent, x, y, yaw)
    const c = k.m('#15803d')
    k.box(g, 0.4, 0.05, 0.4, c, 0, 0.45, 0)
    k.box(g, 0.4, 0.4, 0.04, c, 0, 0.68, -0.19)
    for (const [sx, sz] of [[-1, -1], [1, -1], [-1, 1], [1, 1]]) k.box(g, 0.03, 0.45, 0.03, k.m('#1f2937'), sx! * 0.17, 0.22, sz! * 0.17)
    return g
  }
  const counter = (parent: THREE.Object3D, x: number, y: number, yaw: number, w: number, glow: THREE.Material) => {
    const g = prop('counter', parent, x, y, yaw)
    k.box(g, w, 1.0, 0.55, k.m('#e2e8f0'), 0, 0.5, 0)
    k.box(g, w + 0.06, 0.05, 0.62, k.m('#475569'), 0, 1.02, 0)
    k.box(g, w - 0.1, 0.06, 0.01, glow, 0, 0.75, 0.28, false)
    return g
  }
  const coffeeMachine = (parent: THREE.Object3D, x: number, y: number, yaw: number, h = 1.0) => {
    const g = prop('coffee_machine', parent, x, y, yaw, 0)
    k.box(g, 0.4, 0.5, 0.35, k.m('#18181b', { metalness: 0.5, roughness: 0.3 }), 0, h + 0.25, 0)
    k.box(g, 0.12, 0.08, 0.02, k.m('#ef4444', { emissive: '#ef4444', emissiveIntensity: 0.6 }), 0, h + 0.38, 0.18, false)
    return g
  }
  const fridge = (parent: THREE.Object3D, x: number, y: number, yaw: number) => {
    const g = prop('fridge', parent, x, y, yaw)
    k.box(g, 0.7, 1.85, 0.65, k.m('#e5e7eb', { metalness: 0.3, roughness: 0.4 }), 0, 0.925, 0)
    k.box(g, 0.03, 0.6, 0.03, k.m('#64748b'), 0.28, 1.1, 0.34)
    return g
  }
  return { plant, sofa, bench, roundTable, cafeChair, counter, coffeeMachine, fridge }
}

/** Perabot luar ruangan: pohon, semak, tiang lampu, mobil. */
function createOutdoorProps(k: Kit, st: WorldState) {
  const { lampMat, cars } = st
  const tree = (parent: THREE.Object3D, x: number, y: number, s = 1) => {
    const g = prop('tree', parent, x, y)
    k.cyl(g, 0.1 * s, 0.14 * s, 1.2 * s, k.m('#7c4a2d'), 0, 0.6 * s, 0, 8)
    k.ball(g, 0.75 * s, k.m('#3d8b3d'), 0, 1.6 * s, 0)
    k.ball(g, 0.55 * s, k.m('#4fa34f'), 0.3 * s, 2.0 * s, 0.15 * s)
    k.ball(g, 0.45 * s, k.m('#2f7a35'), -0.35 * s, 1.85 * s, -0.1 * s)
    return g
  }
  const bush = (parent: THREE.Object3D, x: number, y: number, s = 1, flower?: string) => {
    const g = prop('bush', parent, x, y)
    k.ball(g, 0.35 * s, k.m('#4a9a44'), 0, 0.25 * s, 0)
    k.ball(g, 0.28 * s, k.m('#5aae52'), 0.25 * s, 0.22 * s, 0.1 * s)
    if (flower) for (let i = 0; i < 4; i++) k.ball(g, 0.06, k.m(flower), Math.cos(i * 1.6) * 0.3 * s, 0.45 * s, Math.sin(i * 1.6) * 0.25 * s, 0)
    return g
  }
  const lampPost = (parent: THREE.Object3D, x: number, y: number) => {
    const g = prop('lamp_post', parent, x, y)
    k.cyl(g, 0.05, 0.07, 3.2, k.m('#334155', { metalness: 0.5 }), 0, 1.6, 0, 8)
    k.ball(g, 0.18, lampMat, 0, 3.3, 0)
    return g
  }
  const car = (parent: THREE.Object3D, color: string) => {
    const g = new THREE.Group()
    g.userData.prop = 'car'
    parent.add(g)
    const body = k.m(color, { metalness: 0.4, roughness: 0.4 })
    k.box(g, 0.9, 0.45, 2.0, body, 0, 0.4, 0)
    k.box(g, 0.8, 0.35, 1.1, k.m('#bae6fd', { metalness: 0.3, roughness: 0.1 }), 0, 0.8, -0.1)
    for (const [sx, sz] of [[-1, -1], [1, -1], [-1, 1], [1, 1]]) {
      const w = k.cyl(g, 0.17, 0.17, 0.12, k.m('#111827'), sx! * 0.44, 0.17, sz! * 0.65, 12)
      w.rotation.z = Math.PI / 2
    }
    k.box(g, 0.2, 0.08, 0.02, lampMat, -0.28, 0.45, 1.0, false)
    k.box(g, 0.2, 0.08, 0.02, lampMat, 0.28, 0.45, 1.0, false)
    cars.push(g)
    return g
  }

  return { tree, bush, lampPost, car }
}

/** Dinding, partisi kaca, dinding berjendela. */
function createWalls(k: Kit) {
  const wall = (parent: THREE.Object3D, axis: 'x' | 'z', at: number, from: number, to: number, h: number, mat: THREE.Material, gaps: Array<[number, number]> = [], t = 0.1) => {
    const cuts = gaps.map(([c, w]) => [c - w / 2, c + w / 2] as const).sort((a, b) => a[0] - b[0])
    let start = from
    const segs: Array<[number, number]> = []
    for (const [a, b] of cuts) {
      if (a > start) segs.push([start, Math.min(a, to)])
      start = Math.max(start, b)
    }
    if (start < to) segs.push([start, to])
    for (const [a, b] of segs) {
      const len = (b - a) * S
      if (len <= 0.01) continue
      const mid = ((a + b) / 2) * S
      if (axis === 'x') k.box(parent, len, h, t, mat, mid, h / 2, at * S)
      else k.box(parent, t, h, len, mat, at * S, h / 2, mid)
    }
  }
  const glass = k.m('#bfdbfe', { transparent: true, opacity: 0.28, roughness: 0.1, metalness: 0.1, depthWrite: false })
  const partition = (parent: THREE.Object3D, axis: 'x' | 'z', at: number, from: number, to: number, gaps: Array<[number, number]> = []) => {
    const g = new THREE.Group()
    parent.add(g)
    wall(g, axis, at, from, to, 0.12, k.m('#94a3b8'), gaps, 0.08)
    const glassG = new THREE.Group()
    glassG.position.y = 0.12
    g.add(glassG)
    wall(glassG, axis, at, from, to, PART_H - 0.16, glass, gaps, 0.03)
    glassG.children.forEach((c) => { c.castShadow = false })
    const top = new THREE.Group()
    top.position.y = PART_H - 0.04
    g.add(top)
    wall(top, axis, at, from, to, 0.04, k.m('#64748b'), gaps, 0.06)
  }

  const windowWall = (parent: THREE.Object3D, axis: 'x' | 'z', at: number, from: number, to: number, h: number, mat: THREE.Material, glassMat: THREE.Material, wins: number[], doors: Array<[number, number]> = [], t = 0.1) => {
    const band = (y: number, bh: number, gaps: Array<[number, number]>) => {
      const b = new THREE.Group()
      b.position.y = y
      parent.add(b)
      wall(b, axis, at, from, to, bh, mat, gaps, t)
    }
    band(0, WIN_Y0, doors)
    band(WIN_Y0, WIN_H, [...doors, ...wins.map(c => [c, WIN_W] as [number, number])])
    band(WIN_Y0 + WIN_H, h - WIN_Y0 - WIN_H, [])
    for (const c of wins) {
      const pane = axis === 'x'
        ? k.box(parent, WIN_W * S, WIN_H, 0.02, glassMat, c * S, WIN_Y0 + WIN_H / 2, at * S, false)
        : k.box(parent, 0.02, WIN_H, WIN_W * S, glassMat, at * S, WIN_Y0 + WIN_H / 2, c * S, false)
      pane.receiveShadow = false
      pane.renderOrder = 1
    }
  }
  const windowGlass = () => k.own(new THREE.MeshStandardMaterial({
    color: '#bae6fd', emissive: '#fde68a', emissiveIntensity: 0, roughness: 0.05, metalness: 0.1, transparent: true, opacity: 0.22, depthWrite: false
  }))
  return { wall, partition, windowWall, windowGlass }
}

type Props = ReturnType<typeof createOfficeProps> & ReturnType<typeof createLoungeProps> & ReturnType<typeof createOutdoorProps> & ReturnType<typeof createWalls>
type Ctx = Props & { k: Kit, st: WorldState, label: ReturnType<typeof createLabel> }

// ---------- Tanah, jalan, taman, halaman ----------

function buildRoads(c: Ctx, ground: THREE.Group) {
  const { k, tree, lampPost, car, st } = c
  const { cars } = st
  const grass = k.m('#7cc36a')
  k.plane(ground, W + 14, D + 14, grass, W / 2, 0, D / 2)
  const pave = k.m('#d6d3d1')
  const paver = k.m('#cbbfae')
  const asphalt = k.m('#3f3f46')
  const T0 = LLM_TOWER
  const flat = (w: number, d: number, mat: THREE.Material, x: number, y: number, h = 0.01) => k.plane(ground, w * S, d * S, mat, (x + w / 2) * S, h, (y + d / 2) * S)
  flat(BUILDING_W + 120, BUILDING_D + 120, pave, -60, -60, 0.005)
  flat(220, 220, paver, BUILDING_W, CORRIDOR_Y - 110, 0.012)
  flat(ROAD_X - BUILDING_W - 160, 80, paver, BUILDING_W + 160, 600, 0.012)
  flat(T0.w + 120, 160, paver, T0.x - 60, T0.y + T0.h + 20, 0.012)
  flat(80, 120, paver, 1560, 660, 0.012)
  flat(BUILDING_W + 40, 80, paver, 1220 - 40, 860, 0.011)
  flat(70, 600, k.m('#d9c9a8'), 885, BUILDING_D + 60, 0.012)
  flat(700, 60, k.m('#d9c9a8'), 200, 1160, 0.013)
  // Jalan raya
  k.plane(ground, ROAD_W * S, D + 14, asphalt, (ROAD_X + ROAD_W / 2) * S, 0.015, D / 2)
  for (let z = -6; z < D + 6; z += 1.6) k.plane(ground, 0.12, 0.8, k.m('#fafafa'), (ROAD_X + ROAD_W / 2) * S, 0.02, z)
  k.plane(ground, 0.4, D + 14, pave, (ROAD_X - 20) * S, 0.02, D / 2)
  k.plane(ground, 0.4, D + 14, pave, (ROAD_X + ROAD_W + 20) * S, 0.02, D / 2)
  car(ground, '#ef4444').position.set((ROAD_X + 45) * S, 0, 2)
  car(ground, '#3b82f6').position.set((ROAD_X + 125) * S, 0, 9)
  car(ground, '#facc15').position.set((ROAD_X + 45) * S, 0, 14)
  cars[1]!.rotation.y = Math.PI
  for (let y = 60; y < MAP_D; y += 280) {
    lampPost(ground, ROAD_X - 30, y)
    tree(ground, ROAD_X + ROAD_W + 70, y + 140, 0.9)
  }
}

function buildGarden(c: Ctx, ground: THREE.Group) {
  const { k, label, bench, tree, bush, lampPost } = c
  // Taman
  k.cyl(ground, 1.75, 1.75, 0.06, k.m('#a8a29e'), 4.2, 0.02, 12.4, 32).castShadow = false
  const pond = k.cyl(ground, 1.6, 1.6, 0.06, k.m('#38bdf8', { roughness: 0.15, metalness: 0.2 }), 4.2, 0.035, 12.4, 32)
  pond.castShadow = false
  k.cyl(ground, 0.12, 0.2, 0.7, k.m('#e7e5e4'), 4.2, 0.35, 12.4, 12)
  const water = k.ball(ground, 0.14, k.m('#bae6fd', { transparent: true, opacity: 0.8 }), 4.2, 0.8, 12.4)
  for (const [x, y, s] of [[90, 920, 1.1], [40, 1400, 1.2], [700, 960, 0.9], [1040, 1000, 1], [1150, 1420, 1.1], [40, 1150, 1]] as const) tree(ground, x, y, s)
  for (const [x, y, c] of [[230, 980, '#f472b6'], [380, 980, '#facc15'], [600, 1080, '#f87171'], [980, 1250, '#c084fc'], [300, 1360, '#fb923c'], [720, 1220, '#f472b6']] as const) bush(ground, x, y, 1, c)
  const benchAt = (a: Spot, b: Spot) => bench(ground, (a.x + b.x) / 2, (a.y + b.y) / 2, a.face ?? 0, Math.max(1.3, Math.hypot(a.x - b.x, a.y - b.y) * S + 0.6))
  benchAt(GARDEN_SPOTS[0]!, GARDEN_SPOTS[1]!)
  benchAt(GARDEN_SPOTS[2]!, GARDEN_SPOTS[3]!)
  benchAt(GARDEN_SPOTS[4]!, GARDEN_SPOTS[5]!)
  lampPost(ground, 960, 1000)
  lampPost(ground, 180, 1210)
  // Gazebo kecil
  const gazebo = new THREE.Group()
  gazebo.position.set(9.0, 0, 13.6)
  ground.add(gazebo)
  for (const [sx, sz] of [[-1, -1], [1, -1], [-1, 1], [1, 1]]) k.cyl(gazebo, 0.06, 0.06, 2.3, k.m('#f5f5f4'), sx! * 0.9, 1.15, sz! * 0.9, 8)
  const cone = new THREE.Mesh(k.own(new THREE.ConeGeometry(1.6, 0.8, 4)), k.m('#b45309'))
  cone.position.y = 2.7
  cone.rotation.y = Math.PI / 4
  cone.castShadow = true
  gazebo.add(cone)
  k.cyl(gazebo, 1.2, 1.2, 0.1, k.m('#d6d3d1'), 0, 0.05, 0, 6)
  label(ground, '🌳 TAMAN', 560, 900, 0.03, 0.5, { color: '#14532d' })
  return water
}

function buildCafe(c: Ctx, ground: THREE.Group) {
  const { k, label, plant, roundTable, cafeChair, counter, coffeeMachine, st } = c
  const { cafeLaptops } = st
  // Kafe Lepas
  const cafe = new THREE.Group()
  ground.add(cafe)
  const C = CAFE_RECT
  k.box(cafe, C.w * S, 0.1, C.h * S, k.m('#c8a27a'), (C.x + C.w / 2) * S, 0.05, (C.y + C.h / 2) * S, false)
  k.box(cafe, (C.w - 120) * S, 2.8, 0.25, k.m('#fef3c7'), (C.x + C.w / 2) * S, 1.4, (C.y + 5) * S)
  counter(cafe, C.x + C.w / 2, C.y + 50, FACE_S, 3.4, k.m('#92400e'))
  coffeeMachine(cafe, C.x + C.w / 2 - 120, C.y + 45, FACE_S, 1.0)
  coffeeMachine(cafe, C.x + C.w / 2 + 120, C.y + 45, FACE_S, 1.0)
  for (let i = 0; i < 8; i++) k.box(cafe, (C.w / 8) * S - 0.02, 0.06, 1.0, k.m(i % 2 ? '#fafafa' : '#16a34a'), (C.x + (i + 0.5) * (C.w / 8)) * S, 2.65, (C.y + 60) * S)
  label(cafe, '☕ KAFE LEPAS', C.x + C.w / 2, C.y + 8, 2.3, 0.42, { color: '#ffffff', bg: '#166534' }, false).position.z += 0.14
  CAFE_TABLES.forEach(t => roundTable(cafe, t.x, t.y, 0.42))
  CAFE_SEATS.forEach((s) => {
    const back = s.face! + Math.PI
    cafeChair(cafe, s.x + Math.sin(back) * 8, s.y + Math.cos(back) * 8, s.face!)
    const lap = new THREE.Group()
    lap.position.set((s.x + Math.sin(s.face!) * 26) * S, 0.77, s.y * S)
    lap.rotation.y = s.face! + Math.PI
    k.box(lap, 0.32, 0.02, 0.22, k.m('#94a3b8', { metalness: 0.5 }), 0, 0, 0)
    const scr = k.box(lap, 0.32, 0.2, 0.015, k.m('#38bdf8', { emissive: '#38bdf8', emissiveIntensity: 0.9 }), 0, 0.1, -0.1)
    scr.rotation.x = -0.25
    lap.visible = false
    cafe.add(lap)
    cafeLaptops.push(lap)
  })
  for (const x of [C.x + 20, C.x + C.w - 20]) plant(cafe, x, C.y + C.h - 20, 1.1)
}

function buildLlmTower(c: Ctx, ground: THREE.Group) {
  const { k, label, plant, st } = c
  const { windowMats, ledMats } = st
  // Gedung Model LLM
  const T = LLM_TOWER
  const tower = new THREE.Group()
  tower.position.set((T.x + T.w / 2) * S, 0, (T.y + T.h / 2) * S)
  ground.add(tower)
  const towerH = 8
  k.box(tower, T.w * S, towerH, T.h * S, k.m('#334155', { metalness: 0.3, roughness: 0.5 }), 0, towerH / 2, 0)
  const towerGlass = k.own(new THREE.MeshStandardMaterial({ color: '#1e3a8a', emissive: '#6366f1', emissiveIntensity: 0.25, metalness: 0.6, roughness: 0.2 }))
  windowMats.push(towerGlass)
  for (let i = 0; i < 7; i++) {
    const y = 1.6 + i * 1.05
    k.box(tower, T.w * S + 0.04, 0.6, T.h * S + 0.04, towerGlass, 0, y, 0, false)
  }
  k.box(tower, 1.6, 2.2, 0.1, k.m('#0f172a'), 0, 1.1, (T.h / 2) * S + 0.05)
  k.box(tower, 2.4, 0.12, 1.2, k.m('#e2e8f0'), 0, 2.4, (T.h / 2) * S + 0.55)
  label(tower, '✨ MODEL LLM', 0, T.h / 2 + 6, 3.0, 0.55, { color: '#e0e7ff', bg: '#4338ca' }, false)
  k.cyl(tower, 0.04, 0.06, 2.2, k.m('#94a3b8'), 0, towerH + 1.1, 0, 6)
  const beacon = k.own(new THREE.MeshStandardMaterial({ color: '#ef4444', emissive: '#ef4444', emissiveIntensity: 1 }))
  ledMats.push(beacon)
  k.ball(tower, 0.12, beacon, 0, towerH + 2.25, 0)
  for (const sx of [-1, 1]) plant(ground, T.x + T.w / 2 + sx * 150, T.y + T.h + 40, 1.2)
}

function buildFrontYard(c: Ctx, ground: THREE.Group) {
  const { label, plant, tree, lampPost } = c
  // Halaman depan
  for (const y of [CORRIDOR_Y - 130, CORRIDOR_Y + 130]) plant(ground, BUILDING_W + 40, y, 1.2)
  lampPost(ground, BUILDING_W + 200, 260)
  lampPost(ground, BUILDING_W + 200, 580)
  for (const [x, y] of [[1300, 1250], [1950, 700], [1950, 380], [1960, 1150], [1900, 1400], [1300, 120]] as const) tree(ground, x, y, 1)
  label(ground, 'HALAMAN', 1420, 560, 0.03, 0.4, { color: '#57534e' })
}

// ---------- Lantai dalam gedung ----------

function buildRoomFloors(c: Ctx, g: THREE.Group, f: number) {
  const { k, label, st } = c
  const { tileRepeat } = st
  for (const r of ROOMS.filter(r => r.f === f && r.floor)) {
    const mat = k.own(new THREE.MeshStandardMaterial({ color: new THREE.Color().setStyle(r.floor, THREE.SRGBColorSpace), roughness: 0.85, map: tileRepeat(r.rect.w / 50, r.rect.h / 50) }))
    k.plane(g, r.rect.w * S, r.rect.h * S, mat, (r.rect.x + r.rect.w / 2) * S, 0.004, (r.rect.y + r.rect.h / 2) * S)
    const isNorth = r.rect.y < CORRIDOR_Y
    label(g, r.label, r.rect.x + r.rect.w / 2, isNorth ? r.rect.y + r.rect.h - 22 : r.rect.y + 26, 0.008, 0.26, { color: 'rgba(51,65,85,0.75)' })
  }
}

function buildExteriorWalls(c: Ctx, g: THREE.Group, f: number) {
  const { k, wall, windowWall, windowGlass, st } = c
  const { windowMats } = st
  // Dinding luar: utara & barat tinggi (berjendela kaca), selatan & timur rendah (potongan agar isi terlihat)
  const extWall = k.m('#e2e8f0')
  const win = windowGlass()
  windowMats.push(win)
  windowWall(g, 'x', 0, 0, BUILDING_W, WALL_H, extWall, win, winCenters(BUILDING_W))
  windowWall(g, 'z', 0, 0, BUILDING_D, WALL_H, extWall, win, WEST_WINS, [[CORRIDOR_Y, 110]])
  wall(g, 'x', BUILDING_D, 0, BUILDING_W, LOW_WALL, extWall, f === 0 ? [[GARDEN_DOOR.x, 130]] : [])
  wall(g, 'z', BUILDING_W, 0, BUILDING_D, LOW_WALL, extWall, f === 0 ? [[DOOR.y, 150]] : [])
}

function buildCore(c: Ctx, g: THREE.Group, f: number) {
  const { k, label } = c
  // Lift di ujung barat koridor
  k.box(g, 0.08, 2.2, 1.0, k.m('#cbd5e1', { metalness: 0.7, roughness: 0.25 }), 0.06, 1.1, CORRIDOR_Y * S)
  k.box(g, 0.09, 0.02, 1.0, k.m('#94a3b8'), 0.065, 1.1, CORRIDOR_Y * S)
  k.box(g, 0.1, 0.12, 0.3, k.m('#22c55e', { emissive: '#22c55e', emissiveIntensity: 0.8 }), 0.07, 2.35, CORRIDOR_Y * S)
  label(g, `LIFT · LANTAI ${f + 1}`, 70, CORRIDOR_Y, 0.009, 0.2, { color: '#334155', bg: 'rgba(255,255,255,0.75)' })
  // Inti gedung: tangga (utara) & toilet (selatan)
  const stairs = new THREE.Group()
  g.add(stairs)
  for (let i = 0; i < 10; i++) k.box(stairs, 1.1, (i + 1) * (FH / 10), 0.3, k.m('#d6d3d1'), 0.7, ((i + 1) * (FH / 10)) / 2, (340 - i * 30) * S)
  label(g, 'TANGGA', 70, 360, 0.01, 0.2, { color: '#475569' })
  k.box(g, 1.2, 2.2, 0.08, k.m('#e2e8f0'), 0.7, 1.1, 6.4)
  k.box(g, 0.7, 1.9, 0.04, k.m('#64748b'), 0.7, 0.95, 6.36)
  label(g, '🚻 WC', 70, 700, 0.01, 0.24, { color: '#475569' })
}

function buildPartitions(c: Ctx, g: THREE.Group, f: number) {
  const { partition } = c
  // Partisi koridor & antar-ruang
  const rooms = ROOMS.filter(r => r.f === f && r.kind !== 'garden' && r.kind !== 'outside')
  const doorsOn = (y: number) => rooms.filter(r => r.door.y === y).map(r => [r.door.x, 100] as [number, number])
  partition(g, 'x', CORRIDOR_Y - CORRIDOR_HALF, CORE_W, BUILDING_W, doorsOn(CORRIDOR_Y - CORRIDOR_HALF))
  partition(g, 'x', CORRIDOR_Y + CORRIDOR_HALF, CORE_W, BUILDING_W, doorsOn(CORRIDOR_Y + CORRIDOR_HALF))
  partition(g, 'z', CORE_W, 0, CORRIDOR_Y - CORRIDOR_HALF)
  partition(g, 'z', CORE_W, CORRIDOR_Y + CORRIDOR_HALF, BUILDING_D)
  for (const r of rooms) {
    const right = r.rect.x + r.rect.w
    if (right < BUILDING_W - 1) partition(g, 'z', right, r.rect.y, r.rect.y + r.rect.h)
  }
  return rooms
}

// ---------- Perabot per zona ----------

/** Papan nama kecil di meja (digambar ulang saat pegawai berganti). */
function makeNamePlate(c: Ctx, g: THREE.Group, s: Spot, dx: number, fwd: number) {
  const { k } = c
  const dy = Math.cos(fwd)
  const pl = canvasTexture(192, 48, () => {})
  k.own(pl.texture)
  const plateMat = k.own(new THREE.MeshBasicMaterial({ map: pl.texture, transparent: true }))
  const plate = new THREE.Mesh(k.own(new THREE.PlaneGeometry(0.44, 0.11)), plateMat)
  plate.position.set((s.x + dx * 29) * S + Math.cos(fwd) * 0.32, 0.8, (s.y + dy * 29) * S - Math.sin(fwd) * 0.32)
  plate.rotation.y = fwd + Math.PI
  plate.rotation.x = 0
  plate.visible = false
  g.add(plate)
  return { ctx: pl.ctx, texture: pl.texture, mesh: plate }
}

function furnishDesk(c: Ctx, g: THREE.Group, z: (typeof ZONES)[number], s: Spot, i: number) {
  const { k, chair, desk, monitor, st } = c
  const leader = z.key === 'executive'
  const fwd = s.face ?? FACE_S
  const dx = Math.sin(fwd)
  const dy = Math.cos(fwd)
  const chairMat = k.own(new THREE.MeshStandardMaterial({ color: '#475569', roughness: 0.7 }))
  chair(g, s.x - dx * 4, s.y - dy * 4, fwd, chairMat, leader)
  desk(g, s.x + dx * 58, s.y + dy * 58, fwd, leader ? 1.25 : 1.0, leader)
  const mon = monitor(g, s.x + dx * 70, s.y + dy * 70, fwd + Math.PI)
  const plate = makeNamePlate(c, g, s, dx, fwd)
  st.desks.push({
    key: `${z.key}-${i}`, division: z.key, index: i, spot: s, leader, chairMat,
    screen: { ctx: mon.ctx, texture: mon.texture, mat: mon.mat },
    plate, sig: ''
  })
}

function furnishDesks(c: Ctx, g: THREE.Group, f: number) {
  for (const z of ZONES.filter(z => z.room.f === f)) {
    z.desks.forEach((s, i) => furnishDesk(c, g, z, s, i))
  }
}

type RoomFurnisher = (c: Ctx, g: THREE.Group, r: Room) => void

const furnishBusiness: RoomFurnisher = ({ bookshelf, whiteboard, plant }, g, { rect: { x, y, w, h } }) => {
  bookshelf(g, x + 120, y + 25, FACE_S, 1.4, 1.6)
  whiteboard(g, x + 380, y + 8, FACE_S)
  plant(g, x + w - 30, y + 30)
  plant(g, x + 30, y + h - 30, 0.9)
}

const furnishPantry: RoomFurnisher = ({ k, counter, coffeeMachine, fridge, plant }, g, { rect: { x, y, w, h } }) => {
  counter(g, x + w / 2 - 20, y + 40, FACE_S, 1.8, k.m('#f59e0b'))
  coffeeMachine(g, x + w / 2 - 70, y + 38, FACE_S)
  fridge(g, x + w - 40, y + 45, FACE_S)
  const bar = new THREE.Group()
  bar.position.set(770 * S, 0, ((LOUNGE_SPOTS[0]!.y + LOUNGE_SPOTS[3]!.y) / 2) * S)
  g.add(bar)
  k.box(bar, 2.0, 0.05, 0.5, k.m(WOOD), 0, 1.02, 0)
  for (const sx of [-0.85, 0.85]) k.box(bar, 0.06, 1.0, 0.4, k.m('#334155'), sx, 0.5, 0)
  plant(g, x + 25, y + h - 25, 0.8)
}

const furnishLobby: RoomFurnisher = ({ k, st, label, counter, bookshelf, sofa, plant }, g, { rect: { x, y, w, h } }) => {
  const cx = (COUNTER_SLOTS[0]!.x + COUNTER_SLOTS[COUNTER_SLOTS.length - 1]!.x) / 2
  counter(g, cx, COUNTER_SLOTS[0]!.y - 55, FACE_S, 2.6, st.counterGlow)
  label(g, '📨 LOKET LLM', cx, COUNTER_SLOTS[0]!.y - 82, 1.25, 0.24, { color: '#ffffff', bg: '#4f46e5' }, false)
  bookshelf(g, cx, y + 18, FACE_S, 2.4, 1.5)
  sofa(g, x + 70, y + h - 70, FACE_E, 1.4, '#6366f1')
  plant(g, x + w - 25, y + h - 25, 1.1)
  k.plane(g, 2.2, 1.2, k.m('#4f46e5', { roughness: 1 }), (x + w / 2) * S, 0.006, (y + h - 75) * S)
}

/** Dua meja baca panjang di perpustakaan, masing-masing di antara dua kursi. */
function libraryTables(k: Kit, g: THREE.Group) {
  for (const [a, b] of [[0, 2], [3, 5]] as const) {
    const s0 = LIBRARY_SEATS[a]!
    const s1 = LIBRARY_SEATS[b]!
    const t = new THREE.Group()
    t.userData.prop = 'meeting_table'
    t.position.set(((s0.x + s1.x) / 2) * S, 0, (s0.y + 55) * S)
    g.add(t)
    k.box(t, (s1.x - s0.x) * S + 0.7, 0.05, 0.7, k.m(WOOD), 0, 0.74, 0)
    for (const sx of [-1, 1]) k.box(t, 0.06, 0.72, 0.6, k.m(WOOD_DARK), sx * ((s1.x - s0.x) * S / 2 + 0.25), 0.36, 0)
    k.cyl(t, 0.05, 0.08, 0.35, k.m('#16a34a'), 0, 0.93, 0, 10)
  }
}

const furnishLibrary: RoomFurnisher = ({ k, bookshelf, chair, plant }, g, { rect: { x, y, w, h } }) => {
  bookshelf(g, x + 110, y + h - 20, FACE_N, 1.6)
  bookshelf(g, x + 290, y + h - 20, FACE_N, 1.6)
  bookshelf(g, x + 470, y + h - 20, FACE_N, 1.6)
  libraryTables(k, g)
  LIBRARY_SEATS.forEach(s => chair(g, s.x, s.y - 4, s.face ?? FACE_S, k.m('#7c3aed')))
  plant(g, x + w - 30, y + 30)
}

const furnishLounge: RoomFurnisher = ({ k, label, sofa, tv, plant }, g, { rect: { x, y, w, h } }) => {
  sofa(g, x + 160, y + 140, FACE_S, 1.8, '#0ea5e9')
  sofa(g, x + 160, y + 300, FACE_N, 1.8, '#0ea5e9')
  k.box(g, 1.0, 0.4, 0.6, k.m(WOOD), (x + 160) * S, 0.2, (y + 220) * S)
  tv(g, x + 12, y + 220, FACE_E, 1.3)
  plant(g, x + w - 30, y + 30, 1.2)
  plant(g, GARDEN_DOOR.x - 110, y + h - 30)
  plant(g, GARDEN_DOOR.x + 110, y + h - 30)
  label(g, '🌳 PINTU TAMAN', GARDEN_DOOR.x, y + h - 40, 0.01, 0.2, { color: '#166534' })
}

const furnishProduct: RoomFurnisher = ({ whiteboard, plant }, g, { rect: { x, y, w, h } }) => {
  whiteboard(g, x + w / 2, y + 8, FACE_S, 2.2)
  plant(g, x + 30, y + 30)
  plant(g, x + w - 30, y + h - 30, 0.9)
}

const furnishQuality: RoomFurnisher = ({ whiteboard, plant }, g, { rect: { x, y, w } }) => {
  whiteboard(g, x + w / 2, y + 8, FACE_S, 2.2)
  plant(g, x + w - 30, y + 30)
}

const furnishEngineering: RoomFurnisher = ({ whiteboard, plant }, g, { rect: { x, y, w, h } }) => {
  whiteboard(g, x + 12, y + h / 2, FACE_E, 2.2)
  plant(g, x + w - 30, y + h - 30)
  plant(g, x + w - 30, y + 30, 0.8)
}

const furnishPlatform: RoomFurnisher = ({ serverRack, plant }, g, { rect: { x, y, w, h } }) => {
  for (const yy of [y + 70, y + 170, y + 270]) serverRack(g, x + w - 45, yy, FACE_W)
  plant(g, x + 30, y + h - 30)
}

/** TV Papan Keputusan di dinding utara (menghadap kamera); layarnya di luar grup prop agar tetap ada bila TV diganti aset .glb. */
function buildDecisionBoardScreen(c: Ctx, g: THREE.Group, r: Room) {
  const { k, tv, st } = c
  const { x, y, w } = r.rect
  const tw = 2.0
  const th = tw * 0.58
  const ty = 1.55
  tv(g, x + w / 2, y + 8, FACE_S, ty, tw)
  const board = canvasTexture(1024, 594, () => {})
  k.own(board.texture)
  const screen = new THREE.Mesh(k.own(new THREE.PlaneGeometry(tw - 0.08, th - 0.08)), k.own(new THREE.MeshBasicMaterial({ map: board.texture, toneMapped: false })))
  screen.position.set((x + w / 2) * S, ty, (y + 8) * S + 0.04)
  g.add(screen)
  st.decisionBoard = { ctx: board.ctx, texture: board.texture }
}

const furnishExecutive: RoomFurnisher = (c, g, r) => {
  const { bookshelf, plant, sofa } = c
  const { x, y, w, h } = r.rect
  bookshelf(g, x + 75, y + 22, FACE_S, 1.0, 1.9)
  bookshelf(g, x + w - 75, y + 22, FACE_S, 1.0, 1.9)
  buildDecisionBoardScreen(c, g, r)
  plant(g, x + 30, y + h - 30, 1.2)
  plant(g, x + w - 30, y + h - 30, 1.2)
  sofa(g, x + w / 2, y + h - 60, FACE_N, 2.0, '#b45309')
}

/** Meja rapat panjang dengan strip cahaya dan kaki di kedua ujung. */
function meetingTable(c: Ctx, g: THREE.Group) {
  const { k, st } = c
  const t = new THREE.Group()
  t.userData.prop = 'meeting_table'
  t.position.set(MEETING_TABLE.x * S, 0, MEETING_TABLE.y * S)
  g.add(t)
  const len = (MEETING_SEATS[MEETING_SEATS.length - 1]!.x - MEETING_SEATS[0]!.x) * S + 0.9
  k.box(t, len, 0.06, 0.95, k.m('#5b3a1e', { roughness: 0.4 }), 0, 0.74, 0)
  k.box(t, len - 0.2, 0.02, 0.1, st.meetingGlow, 0, 0.775, 0, false)
  for (const sx of [-1, 1]) k.box(t, 0.1, 0.72, 0.7, k.m('#3f2a17'), sx * (len / 2 - 0.3), 0.36, 0)
}

const furnishMeeting: RoomFurnisher = (c, g, { rect: { x, y, w, h } }) => {
  const { k, chair, tv, plant } = c
  meetingTable(c, g)
  MEETING_SEATS.forEach(s => chair(g, s.x, s.y - 4, FACE_S, k.m('#7e22ce'), true))
  const fc = prop('flipchart', g, PM_PODIUM.x - 30, PM_PODIUM.y - 70, FACE_S + 0.4)
  k.box(fc, 0.8, 1.0, 0.03, k.m('#f8fafc'), 0, 1.3, 0)
  for (const sx of [-0.3, 0.3]) k.box(fc, 0.04, 1.6, 0.04, k.m('#334155'), sx, 0.8, -0.05)
  tv(g, x + w - 12, MEETING_TABLE.y, FACE_W, 1.4, 1.6)
  plant(g, x + w - 30, y + h - 30)
}

const furnishPm: RoomFurnisher = ({ k, chair, desk, monitor, whiteboard, bookshelf, plant }, g, { rect: { x, y, w, h } }) => {
  chair(g, PM_DESK.x, PM_DESK.y - 4, FACE_S, k.m('#1e293b'), true)
  desk(g, PM_DESK.x, PM_DESK.y + 58, FACE_S, 1.4, true)
  drawScreen(monitor(g, PM_DESK.x, PM_DESK.y + 70, FACE_N).ctx, { active: true, burning: false, project: 'Roadmap', seed: 7 })
  whiteboard(g, x + 12, y + h / 2, FACE_E, 2.4)
  bookshelf(g, x + w - 100, y + h - 20, FACE_N, 1.4)
  plant(g, x + 30, y + h - 30, 1.1)
}

const furnishTerrace: RoomFurnisher = ({ k, bush, roundTable, bench }, g, { rect: { x, y, w, h } }) => {
  k.plane(g, w * S, h * S, k.m('#a16207', { roughness: 0.9 }), (x + w / 2) * S, 0.006, (y + h / 2) * S)
  for (const [px, py] of [[x + 40, y + 40], [x + w - 40, y + 40], [x + 40, y + h - 40], [x + w - 40, y + h - 40], [x + w / 2, y + h - 40]] as const) bush(g, px, py, 1.1, '#f472b6')
  roundTable(g, x + 200, y + 200, 0.5)
  bench(g, x + 380, y + 200, FACE_W, 1.4)
}

const ROOM_FURNISHERS: Record<string, RoomFurnisher> = {
  business: furnishBusiness,
  pantry: furnishPantry,
  lobby: furnishLobby,
  library: furnishLibrary,
  lounge: furnishLounge,
  product: furnishProduct,
  quality: furnishQuality,
  engineering: furnishEngineering,
  platform: furnishPlatform,
  executive: furnishExecutive,
  meeting: furnishMeeting,
  pm: furnishPm,
  terrace: furnishTerrace
}

function furnish(c: Ctx, g: THREE.Group, f: number, rooms: Room[]) {
  furnishDesks(c, g, f)
  for (const r of rooms) ROOM_FURNISHERS[r.key]?.(c, g, r)
}

function buildFloor(c: Ctx, root: THREE.Group, f: number, corridorFloor: THREE.Material) {
  const { k } = c
  const g = new THREE.Group()
  g.position.y = f * FH
  root.add(g)
  // Pelat & lantai koridor/inti
  k.box(g, BUILDING_W * S, 0.16, BUILDING_D * S, k.m('#a8a29e'), (BUILDING_W / 2) * S, -0.08, (BUILDING_D / 2) * S, false)
  k.plane(g, BUILDING_W * S, BUILDING_D * S, corridorFloor, (BUILDING_W / 2) * S, 0.002, (BUILDING_D / 2) * S)
  buildRoomFloors(c, g, f)
  buildExteriorWalls(c, g, f)
  buildCore(c, g, f)
  furnish(c, g, f, buildPartitions(c, g, f))
  return g
}

// ---------- Fasad luar & atap ----------

/** Pintu depan, papan nama, dan pintu taman di fasad lantai dasar. */
function buildGroundFloorEntrances(c: Ctx, g: THREE.Group, O: number) {
  const { k, label } = c
  k.box(g, 0.06, 2.3, 1.4, k.m('#0f172a', { roughness: 0.2, metalness: 0.4 }), (BUILDING_W + O) * S + 0.04, 1.15, DOOR.y * S)
  k.box(g, 0.9, 0.1, 2.2, k.m('#64748b'), BUILDING_W * S + 0.45, 2.55, DOOR.y * S)
  label(g, 'vibeasik', BUILDING_W + O + 6, DOOR.y, 2.95, 0.5, { color: '#ffffff', bg: '#0ea5e9' }, false).rotation.y = Math.PI / 2
  k.box(g, 1.2, 2.2, 0.06, k.m('#14532d', { roughness: 0.3 }), GARDEN_DOOR.x * S, 1.1, (BUILDING_D + O) * S + 0.04)
}

/** Fasad berjendela kaca tepat di luar dinding dalam, sehingga isi lantai (dan pegawainya) terlihat dari luar. */
function buildFacade(c: Ctx, shells: THREE.Group[], root: THREE.Group, f: number) {
  const { k, windowWall, wall, st } = c
  const g = new THREE.Group()
  g.position.y = f * FH
  root.add(g)
  shells.push(g)
  const extWall = k.m('#e2e8f0')
  const hgt = FH - 0.18
  const win = st.windowMats[f + 1]!
  const O = 8
  const xWins = winCenters(BUILDING_W)
  const southWins = f === 0 ? xWins.filter(c => Math.abs(c - GARDEN_DOOR.x) > 110) : xWins
  const eastWins = f === 0 ? winCenters(BUILDING_D).filter(c => Math.abs(c - DOOR.y) > 110) : winCenters(BUILDING_D)
  windowWall(g, 'x', -O, -O - 3, BUILDING_W + O + 3, hgt, extWall, win, xWins, [], 0.06)
  windowWall(g, 'x', BUILDING_D + O, -O - 3, BUILDING_W + O + 3, hgt, extWall, win, southWins, [], 0.06)
  windowWall(g, 'z', -O, -O, BUILDING_D + O, hgt, extWall, win, WEST_WINS, [], 0.06)
  windowWall(g, 'z', BUILDING_W + O, -O, BUILDING_D + O, hgt, extWall, win, eastWins, [], 0.06)
  const cornice = new THREE.Group()
  cornice.position.y = hgt
  g.add(cornice)
  for (const at of [-O, BUILDING_D + O]) wall(cornice, 'x', at, -O - 6, BUILDING_W + O + 6, 0.18, k.m('#94a3b8'), [], 0.12)
  for (const at of [-O, BUILDING_W + O]) wall(cornice, 'z', at, -O, BUILDING_D + O, 0.18, k.m('#94a3b8'), [], 0.12)
  if (f === 0) buildGroundFloorEntrances(c, g, O)
}

function buildRoof(c: Ctx, roof: THREE.Group) {
  const { k, label, wall } = c
  roof.position.y = FLOORS * FH
  k.box(roof, BUILDING_W * S + 0.2, 0.2, BUILDING_D * S + 0.2, k.m('#64748b'), (BUILDING_W / 2) * S, 0.1, (BUILDING_D / 2) * S)
  wall(roof, 'x', -5, -5, BUILDING_W + 5, 0.6, k.m('#94a3b8'), [], 0.15)
  wall(roof, 'x', BUILDING_D + 5, -5, BUILDING_W + 5, 0.6, k.m('#94a3b8'), [], 0.15)
  wall(roof, 'z', -5, -5, BUILDING_D + 5, 0.6, k.m('#94a3b8'), [], 0.15)
  wall(roof, 'z', BUILDING_W + 5, -5, BUILDING_D + 5, 0.6, k.m('#94a3b8'), [], 0.15)
  for (let i = 0; i < 6; i++) {
    const p = k.box(roof, 1.4, 0.05, 0.9, k.m('#1e3a8a', { metalness: 0.6, roughness: 0.25 }), 2 + (i % 3) * 1.7, 0.5, 1.5 + Math.floor(i / 3) * 1.3)
    p.rotation.x = -0.35
  }
  k.box(roof, 1.2, 0.8, 1.0, k.m('#cbd5e1'), 9.5, 0.6, 2)
  k.box(roof, 1.2, 0.8, 1.0, k.m('#cbd5e1'), 9.5, 0.6, 3.4)
  k.box(roof, 1.4, 1.2, 1.4, k.m('#94a3b8'), 0.7, 0.8, 4.2)
  label(roof, 'VIBEASIK HQ', 600, 620, 0.22, 0.9, { color: '#ffffff' })
}

// ---------- Cahaya & animasi ----------

function createLights(root: THREE.Group) {
  const hemi = new THREE.HemisphereLight('#dbeafe', '#6b8f5e', 1.4)
  const sun = new THREE.DirectionalLight('#fff7e6', 2.4)
  sun.castShadow = true
  sun.shadow.mapSize.set(2048, 2048)
  const sc = sun.shadow.camera
  sc.left = -15; sc.right = 15; sc.top = 12; sc.bottom = -12; sc.near = 1; sc.far = 60
  sun.shadow.bias = -0.0005
  sun.shadow.normalBias = 0.02
  sun.target.position.set(W / 2, 0, D / 2)
  root.add(hemi, sun, sun.target)
  return { hemi, sun }
}

function createSetTime(st: WorldState, sun: THREE.DirectionalLight, hemi: THREE.HemisphereLight) {
  return (date: Date, busyFloors: boolean[]) => {
    const hour = date.getHours() + date.getMinutes() / 60
    const day = Math.max(0, Math.sin(((hour - 6) / 12) * Math.PI))
    const night = hour < 6 || hour >= 18
    const az = ((hour - 6) / 12) * Math.PI
    sun.position.set(W / 2 + Math.cos(az) * 18, 6 + day * 16, D / 2 - 8 + Math.sin(az) * 2)
    sun.intensity = night ? 0.5 : 0.9 + day * 1.8
    sun.color.set(night ? '#a5b4fc' : day < 0.35 ? '#fdba74' : '#fff7e6')
    hemi.intensity = night ? 0.9 : 1.1 + day * 0.5
    hemi.color.set(night ? '#64748b' : '#dbeafe')
    st.lampMat.emissiveIntensity = night ? 1.6 : 0.1
    st.windowMats.forEach((m, i) => {
      if (i === 0) { m.emissiveIntensity = night ? 0.9 : 0.25; return } // gedung LLM
      const busy = busyFloors[i - 1] ?? false
      m.emissiveIntensity = night ? (busy ? 1.1 : 0.35) : busy ? 0.25 : 0
    })
  }
}

function createUpdate(st: WorldState, water: THREE.Mesh) {
  return (t: number) => {
    st.cars.forEach((c, i) => {
      const dir = c.rotation.y ? -1 : 1
      const speed = 2.2 + i * 0.6
      const span = D + 12
      c.position.z = (((c.userData.z0 ??= c.position.z) + 6 + dir * t * speed) % span + span) % span - 6
    })
    st.ledMats.forEach((m, i) => { m.emissiveIntensity = 0.4 + (Math.sin(t * (3 + i) + i) > 0.3 ? 1 : 0) })
    water.position.y = 0.8 + Math.sin(t * 3) * 0.05
  }
}

export function buildWorld(): World {
  const k = new Kit()
  const root = new THREE.Group()
  const ground = new THREE.Group()
  const floors: THREE.Group[] = []
  const shells: THREE.Group[] = []
  const roof = new THREE.Group()
  root.add(ground, roof)
  const st = createWorldState(k)
  const c: Ctx = {
    k, st, label: createLabel(k),
    ...createOfficeProps(k, st), ...createLoungeProps(k), ...createOutdoorProps(k, st), ...createWalls(k)
  }

  buildRoads(c, ground)
  const water = buildGarden(c, ground)
  buildCafe(c, ground)
  buildLlmTower(c, ground)
  buildFrontYard(c, ground)

  const corridorFloor = k.m('#d4d4d8', { map: st.tileRepeat(BUILDING_W / 60, BUILDING_D / 60) })
  for (let f = 0; f < FLOORS; f++) floors.push(buildFloor(c, root, f, corridorFloor))
  for (let f = 0; f < FLOORS; f++) buildFacade(c, shells, root, f)
  buildRoof(c, roof)

  const { hemi, sun } = createLights(root)
  return {
    root, ground, floors, shells, roof, desks: st.desks, cafeLaptops: st.cafeLaptops,
    meetingGlow: st.meetingGlow, counterGlow: st.counterGlow, sun, hemi,
    decisionBoard: st.decisionBoard!,
    setTime: createSetTime(st, sun, hemi), update: createUpdate(st, water),
    dispose: () => k.dispose()
  }
}
