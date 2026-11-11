import * as THREE from 'three'
import type { ActorMood, AgentWorker } from '~/types'
import type { AgentLook } from '~/utils/agentOffice'
import { bakeVertexColors } from './bake'

/**
 * Karakter 3D chibi prosedural (satuan meter, menghadap +Z, titik asal di kaki): badan membulat dari kapsul & bola,
 * wajah dengan mata berkedip dan mulut, siku & lutut yang menekuk. Warna kulit/rambut/baju/celana, gaya rambut,
 * rambut wajah, dan aksesori dari `AgentLook` yang sama dengan avatar 2D.
 *
 * `Character3D` adalah kontrak rig-nya: scene hanya memakai sendi-sendi di sini lewat `poseCharacter`, jadi model
 * karakter bisa diganti kelak selama menyediakan sendi yang sama.
 */
export interface Character3D {
  root: THREE.Group
  /** Seluruh badan (naik-turun saat jalan, duduk, melompat). */
  body: THREE.Group
  /** Badan atas: poros di pinggul (membungkuk/bersandar tanpa menggerakkan kaki). */
  torso: THREE.Group
  head: THREE.Group
  legL: THREE.Group
  legR: THREE.Group
  /** Betis: poros di lutut. */
  shinL: THREE.Group
  shinR: THREE.Group
  armL: THREE.Group
  armR: THREE.Group
  /** Lengan bawah: poros di siku. */
  foreL: THREE.Group
  foreR: THREE.Group
  /** Mata (skala vertikal = kedipan) dan mulut (skala = ekspresi): grup berisi satu mesh. */
  eyes: THREE.Object3D
  mouth: THREE.Object3D
  /** Kuncir yang ikut berayun (hanya gaya rambut `ponytail`). */
  tail?: THREE.Object3D
  mug: THREE.Group
  book: THREE.Group
  stars: THREE.Group
  zzz: THREE.Group
  confetti: THREE.Group
  envelope?: THREE.Object3D
  materials: THREE.Material[]
  geometries: THREE.BufferGeometry[]
  /** Nilai sendi yang sudah dihaluskan antar-frame (lihat `poseCharacter`). */
  smooth: Float32Array
  posed: boolean
}

export const HIP_Y = 0.42
export const HEAD_TOP = 1.32

/** Titik tengah tengkorak & jari-jarinya, relatif terhadap leher (grup `head`). */
const SKULL_Y = 0.22
const SKULL_R = 0.25
const TAU = Math.PI * 2

// Geometri dipakai bersama semua karakter (bentuknya sama, hanya warna yang berbeda).
const sharedGeos = new Map<string, THREE.BufferGeometry>()
const geo = (key: string, make: () => THREE.BufferGeometry) => {
  let g = sharedGeos.get(key)
  if (!g) { g = make(); sharedGeos.set(key, g) }
  return g
}
const G = {
  sphere: (r: number) => geo(`s${r}`, () => new THREE.SphereGeometry(r, 12, 8)),
  /** Bola kecil bersegmen sedikit untuk detail (mata, pipi, telinga, ikal rambut). */
  dot: (r: number) => geo(`e${r}`, () => new THREE.SphereGeometry(r, 8, 5)),
  /** Potongan bola: `phi` mengelilingi sumbu Y (π/2 = depan, 3π/2 = belakang), `theta` dari puncak ke bawah. */
  dome: (r: number, phi0: number, phiLen: number, theta0: number, thetaLen: number) =>
    geo(`d${r}|${phi0}|${phiLen}|${theta0}|${thetaLen}`, () => new THREE.SphereGeometry(r, 14, 7, phi0, phiLen, theta0, thetaLen)),
  capsule: (r: number, len: number) => geo(`c${r}|${len}`, () => new THREE.CapsuleGeometry(r, len, 3, 9)),
  torus: (r: number, tube: number, arc = TAU) => geo(`t${r}|${tube}|${arc}`, () => new THREE.TorusGeometry(r, tube, 5, 16, arc)),
  cyl: (rt: number, rb: number, h: number) => geo(`y${rt}|${rb}|${h}`, () => new THREE.CylinderGeometry(rt, rb, h, 10)),
  cone: (r: number, h: number) => geo(`k${r}|${h}`, () => new THREE.ConeGeometry(r, h, 8)),
  box: (w: number, h: number, d: number) => geo(`b${w}|${h}|${d}`, () => new THREE.BoxGeometry(w, h, d)),
  octa: (r: number) => geo(`o${r}`, () => new THREE.OctahedronGeometry(r))
}

let zTexture: THREE.CanvasTexture | null = null
/** Huruf "Z" untuk karakter tertidur (sprite selalu menghadap kamera). */
function zzzTexture() {
  if (zTexture) return zTexture
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = 64
  const c = canvas.getContext('2d')!
  c.font = '900 54px ui-sans-serif, system-ui, sans-serif'
  c.textAlign = 'center'
  c.textBaseline = 'middle'
  c.lineWidth = 8
  c.strokeStyle = '#1e3a8a'
  c.strokeText('Z', 32, 34)
  c.fillStyle = '#bfdbfe'
  c.fillText('Z', 32, 34)
  zTexture = new THREE.CanvasTexture(canvas)
  zTexture.colorSpace = THREE.SRGBColorSpace
  return zTexture
}

let skinMaterial: THREE.MeshStandardMaterial | null = null
/** Satu material berwarna per-vertex untuk badan semua karakter (lihat `bakeVertexColors`). */
const bodyMaterial = () => skinMaterial ??= new THREE.MeshStandardMaterial({ vertexColors: true, roughness: 0.75 })

/** Lepas geometri, material & tekstur bersama (dipanggil saat scene ditutup). */
export function disposeCharacterAssets() {
  sharedGeos.forEach(g => g.dispose())
  sharedGeos.clear()
  skinMaterial?.dispose()
  skinMaterial = null
  zTexture?.dispose()
  zTexture = null
}

interface BuildContext {
  materials: THREE.Material[]
  mat: (color: string, extra?: THREE.MeshStandardMaterialParameters) => THREE.MeshStandardMaterial
  /** Mesh di `parent` pada posisi (x, y, z); skala opsional. Tanpa bayangan — bagian besar memakai `solid`. */
  add: (parent: THREE.Object3D, g: THREE.BufferGeometry, m: THREE.Material, x?: number, y?: number, z?: number, scale?: [number, number, number]) => THREE.Mesh
  /** Seperti `add`, tetapi ikut memberi bayangan (dada, kepala, paha: bagian lain dilewati agar ringan). */
  solid: BuildContext['add']
  group: (parent: THREE.Object3D, x?: number, y?: number, z?: number) => THREE.Group
}

function createBuildContext(): BuildContext {
  const materials: THREE.Material[] = []
  const cache = new Map<string, THREE.MeshStandardMaterial>()
  const mat = (color: string, extra: THREE.MeshStandardMaterialParameters = {}) => {
    const key = color + JSON.stringify(extra)
    let m = cache.get(key)
    if (!m) {
      m = new THREE.MeshStandardMaterial({ color: new THREE.Color().setStyle(color, THREE.SRGBColorSpace), roughness: 0.72, ...extra })
      cache.set(key, m)
      materials.push(m)
    }
    return m
  }
  const add: BuildContext['add'] = (parent, g, m, x = 0, y = 0, z = 0, scale) => {
    const mesh = new THREE.Mesh(g, m)
    mesh.position.set(x, y, z)
    if (scale) mesh.scale.set(...scale)
    parent.add(mesh)
    return mesh
  }
  const solid: BuildContext['add'] = (...args) => {
    const mesh = add(...args)
    mesh.castShadow = true
    return mesh
  }
  const group: BuildContext['group'] = (parent, x = 0, y = 0, z = 0) => {
    const g = new THREE.Group()
    g.position.set(x, y, z)
    parent.add(g)
    return g
  }
  return { materials, mat, add, solid, group }
}

interface Palette {
  skin: THREE.Material
  hair: THREE.Material
  shirt: THREE.Material
  pants: THREE.Material
  dark: THREE.Material
  shoe: THREE.Material
  white: THREE.Material
}

/** Kaki: paha berporos di pinggul, betis + sepatu berporos di lutut. */
function buildLeg(body: THREE.Group, x: number, p: Palette, ctx: BuildContext) {
  const leg = ctx.group(body, x, HIP_Y, 0)
  ctx.solid(leg, G.capsule(0.07, 0.07), p.pants, 0, -0.1, 0)
  const shin = ctx.group(leg, 0, -0.2, 0)
  ctx.add(shin, G.capsule(0.06, 0.06), p.pants, 0, -0.08, 0)
  ctx.add(shin, G.dot(0.075), p.shoe, 0, -0.17, 0.035, [1, 0.62, 1.55])
  return { leg, shin }
}

/** Lengan: lengan atas berporos di bahu, lengan bawah + tangan berporos di siku. */
function buildArm(torso: THREE.Group, x: number, p: Palette, ctx: BuildContext) {
  const arm = ctx.group(torso, x, 0.35, 0)
  ctx.add(arm, G.dot(0.062), p.shirt, 0, 0, 0)
  ctx.add(arm, G.capsule(0.052, 0.07), p.shirt, 0, -0.08, 0)
  const fore = ctx.group(arm, 0, -0.16, 0)
  ctx.add(fore, G.capsule(0.046, 0.06), p.shirt, 0, -0.07, 0)
  ctx.add(fore, G.dot(0.055), p.skin, 0, -0.165, 0)
  return { arm, fore }
}

/** Badan atas: pinggul, dada membulat, leher, dan aksesori badan (dasi, syal) — satu mesh setelah digabung. */
function buildTorso(body: THREE.Group, look: AgentLook, p: Palette, ctx: BuildContext) {
  const torso = ctx.group(body, 0, HIP_Y, 0)
  ctx.add(torso, G.sphere(0.15), p.pants, 0, 0.02, 0, [1.16, 0.62, 0.9])
  ctx.solid(torso, G.capsule(0.15, 0.14), p.shirt, 0, 0.2, 0, [1.18, 1, 0.88])
  ctx.add(torso, G.cyl(0.06, 0.065, 0.07), p.skin, 0, 0.42, 0)
  if (look.accessory === 'tie') {
    ctx.add(torso, G.torus(0.085, 0.022), p.white, 0, 0.405, 0).rotation.x = Math.PI / 2
    const tie = ctx.mat('#b91c1c')
    ctx.add(torso, G.dot(0.026), tie, 0, 0.375, 0.118)
    ctx.add(torso, G.box(0.05, 0.2, 0.012), tie, 0, 0.27, 0.136).rotation.x = -0.06
  }
  if (look.accessory === 'scarf') {
    const scarf = ctx.mat('#dc2626')
    ctx.add(torso, G.torus(0.11, 0.045), scarf, 0, 0.41, 0).rotation.x = Math.PI / 2
    ctx.add(torso, G.box(0.07, 0.17, 0.03), scarf, 0.07, 0.3, 0.13).rotation.z = -0.12
  }
  return torso
}

/** Kepala bulat + wajah: mata (berkedip lewat skala), pipi merona, mulut, telinga. */
function buildHead(torso: THREE.Group, look: AgentLook, p: Palette, ctx: BuildContext) {
  const head = ctx.group(torso, 0, 0.42, 0)
  ctx.solid(head, G.sphere(SKULL_R), p.skin, 0, SKULL_Y, 0, [1, 0.94, 0.95])
  for (const sx of [-1, 1]) ctx.add(head, G.dot(0.045), p.skin, sx * 0.243, SKULL_Y - 0.02, 0, [0.6, 1, 0.9])
  // Kedua mata satu grup: kedipan = skala vertikal grup setinggi mata.
  const eyes = ctx.group(head, 0, SKULL_Y - 0.005, 0.212)
  for (const sx of [-1, 1]) {
    ctx.add(eyes, G.dot(0.034), p.dark, sx * 0.085, 0, 0, [1, 1.1, 0.5])
    ctx.add(eyes, G.dot(0.011), p.white, sx * 0.085 + 0.012, 0.014, 0.014)
  }
  // Rona pipi: warna kulit dicampur merah muda (tanpa transparansi agar bisa digabung dengan kepala).
  const blush = ctx.mat(`#${new THREE.Color().setStyle(look.skin).lerp(new THREE.Color('#fb7185'), 0.42).getHexString()}`)
  for (const sx of [-1, 1]) ctx.add(head, G.dot(0.036), blush, sx * 0.14, SKULL_Y - 0.07, 0.178, [1, 0.7, 0.3])
  const mouth = ctx.group(head, 0, SKULL_Y - 0.09, 0.224)
  ctx.add(mouth, G.dot(0.03), ctx.mat('#8f3a35'), 0, 0, 0, [1.1, 0.35, 0.4])
  return { head, eyes, mouth }
}

type HairStyle = NonNullable<AgentLook['hairStyle']>
type HairBuilder = (head: THREE.Group, hair: THREE.Material, ctx: BuildContext) => THREE.Object3D | undefined

const HAIR_EXTRAS: Partial<Record<HairStyle, HairBuilder>> = {
  long: (head, hair, ctx) => { ctx.solid(head, G.capsule(0.2, 0.14), hair, 0, SKULL_Y - 0.13, -0.11, [1.12, 1, 0.5]); return undefined },
  bun: (head, hair, ctx) => { ctx.add(head, G.dot(0.1), hair, 0, SKULL_Y + 0.25, -0.1); return undefined },
  ponytail: (head, hair, ctx) => {
    const tail = ctx.group(head, 0, SKULL_Y + 0.08, -0.24)
    ctx.add(tail, G.dot(0.065), hair, 0, 0, 0)
    ctx.add(tail, G.capsule(0.05, 0.16), hair, 0, -0.14, -0.03)
    tail.rotation.x = 0.25
    return tail
  },
  curly: (head, hair, ctx) => {
    for (let i = 0; i < 9; i++) {
      const a = (i / 9) * TAU
      ctx.add(head, G.dot(0.085), hair, Math.cos(a) * 0.17, SKULL_Y + 0.17, Math.sin(a) * 0.16 - 0.02)
    }
    ctx.add(head, G.dot(0.09), hair, 0, SKULL_Y + 0.23, -0.01)
    return undefined
  },
  spiky: (head, hair, ctx) => {
    for (const [x, z, tilt] of [[-0.13, 0.02, 0.5], [0, 0.07, 0], [0.13, 0.02, -0.5], [-0.07, -0.1, 0.25], [0.07, -0.1, -0.25]] as const) {
      const spike = ctx.add(head, G.cone(0.07, 0.2), hair, x, SKULL_Y + 0.26 - Math.abs(tilt) * 0.06, z)
      spike.rotation.z = tilt
    }
    return undefined
  }
}

/** Hijab: kerudung menutup kepala kecuali wajah, dengan lipatan di leher & bahu. */
function buildHijab(head: THREE.Group, hair: THREE.Material, ctx: BuildContext) {
  const open = 0.8
  ctx.add(head, G.dome(0.272, Math.PI / 2 + open, TAU - open * 2, 0, Math.PI * 0.9), hair, 0, SKULL_Y, 0, [1, 0.96, 0.97])
  ctx.add(head, G.dome(0.274, 0, TAU, 0, Math.PI * 0.36), hair, 0, SKULL_Y, 0, [1, 0.96, 0.97]).rotation.x = -0.12
  ctx.add(head, G.sphere(0.21), hair, 0, -0.03, 0, [1.15, 0.45, 1])
}

/** Gaya rambut: tudung rambut miring ke belakang + poni + tambahan per gaya. Topi/beanie menutup poni. */
function buildHair(head: THREE.Group, look: AgentLook, hair: THREE.Material, ctx: BuildContext) {
  const cut = look.hairStyle ?? 'short'
  if (cut === 'hijab') { buildHijab(head, hair, ctx); return undefined }
  if (cut === 'bald') return undefined
  ctx.add(head, G.dome(0.262, 0, TAU, 0, Math.PI * 0.5), hair, 0, SKULL_Y, 0, [1, 0.96, 0.97]).rotation.x = -0.38
  ctx.add(head, G.dome(0.26, Math.PI, Math.PI, 0, Math.PI * 0.7), hair, 0, SKULL_Y, 0, [1, 0.96, 0.97])
  if (look.accessory !== 'cap' && look.accessory !== 'beanie') {
    ctx.add(head, G.dot(0.1), hair, -0.075, SKULL_Y + 0.175, 0.15, [1.25, 0.5, 0.62]).rotation.z = 0.32
    ctx.add(head, G.dot(0.09), hair, 0.09, SKULL_Y + 0.185, 0.14, [1.2, 0.5, 0.62]).rotation.z = -0.3
  }
  return HAIR_EXTRAS[cut]?.(head, hair, ctx)
}

/** Janggut / kumis memakai warna rambut. */
function buildFacialHair(head: THREE.Group, look: AgentLook, hair: THREE.Material, ctx: BuildContext) {
  if (look.facial === 'beard') ctx.add(head, G.dome(0.257, 0, Math.PI, Math.PI * 0.66, Math.PI * 0.34), hair, 0, SKULL_Y, 0, [1, 0.94, 0.95])
  if (look.facial === 'beard' || look.facial === 'mustache') {
    ctx.add(head, G.capsule(0.017, 0.07), hair, 0, SKULL_Y - 0.057, 0.233).rotation.z = Math.PI / 2
  }
}

type AccessoryBuilder = (head: THREE.Group, p: Palette, look: AgentLook, kind: AgentWorker['kind'], ctx: BuildContext) => void

const ACCESSORIES: Partial<Record<AgentLook['accessory'], AccessoryBuilder>> = {
  glasses: (head, p, _look, _kind, ctx) => {
    for (const sx of [-1, 1]) {
      ctx.add(head, G.torus(0.05, 0.008), p.dark, sx * 0.085, SKULL_Y - 0.005, 0.238)
      ctx.add(head, G.box(0.008, 0.008, 0.2), p.dark, sx * 0.226, SKULL_Y, 0.12)
    }
    ctx.add(head, G.box(0.05, 0.01, 0.01), p.dark, 0, SKULL_Y, 0.24)
  },
  headset: (head, p, _look, _kind, ctx) => {
    ctx.add(head, G.torus(0.262, 0.016, Math.PI), p.dark, 0, SKULL_Y, 0)
    for (const sx of [-1, 1]) ctx.add(head, G.cyl(0.06, 0.06, 0.045), p.dark, sx * 0.253, SKULL_Y - 0.02, 0).rotation.z = Math.PI / 2
    ctx.add(head, G.capsule(0.008, 0.16), p.dark, -0.235, SKULL_Y - 0.075, 0.1).rotation.x = 1.15
    ctx.add(head, G.dot(0.02), p.dark, -0.21, SKULL_Y - 0.105, 0.19)
  },
  cap: (head, _p, look, kind, ctx) => {
    const cap = ctx.mat(kind === 'llm_call' ? look.shirt : '#1d4ed8')
    ctx.add(head, G.dome(0.268, 0, TAU, 0, Math.PI * 0.42), cap, 0, SKULL_Y, 0, [1, 0.97, 0.98]).rotation.x = -0.1
    ctx.add(head, G.sphere(0.15), cap, 0, SKULL_Y + 0.105, 0.27, [1, 0.08, 0.9])
    ctx.add(head, G.dot(0.025), cap, 0, SKULL_Y + 0.262, -0.02)
  },
  beanie: (head, _p, _look, _kind, ctx) => {
    const wool = ctx.mat('#e11d48', { roughness: 0.95 })
    ctx.add(head, G.dome(0.272, 0, TAU, 0, Math.PI * 0.47), wool, 0, SKULL_Y, 0, [1, 1.04, 0.98]).rotation.x = -0.15
    ctx.add(head, G.torus(0.258, 0.03), ctx.mat('#be123c', { roughness: 0.95 }), 0, SKULL_Y + 0.04, 0).rotation.x = Math.PI / 2 - 0.15
    ctx.add(head, G.dot(0.055), ctx.mat('#fecdd3', { roughness: 1 }), 0, SKULL_Y + 0.29, -0.04)
  }
}

/** Barang di tangan: cangkir kopi (kanan) & buku (kiri); tersembunyi sampai gesturnya dipakai. */
function buildHandProps(foreL: THREE.Group, foreR: THREE.Group, ctx: BuildContext) {
  const mug = ctx.group(foreR, 0, -0.2, 0.05)
  ctx.add(mug, G.cyl(0.04, 0.035, 0.075), ctx.mat('#f8fafc', { roughness: 0.4 }), 0, 0, 0)
  ctx.add(mug, G.cyl(0.033, 0.033, 0.006), ctx.mat('#6b3f22'), 0, 0.036, 0)
  ctx.add(mug, G.torus(0.024, 0.008), ctx.mat('#f8fafc', { roughness: 0.4 }), 0.045, 0, 0)
  mug.visible = false

  const book = ctx.group(foreL, 0.14, -0.2, 0.03)
  ctx.add(book, G.box(0.24, 0.17, 0.022), ctx.mat('#1d4ed8'), 0, 0, 0)
  ctx.add(book, G.box(0.22, 0.155, 0.012), ctx.mat('#fefce8'), 0, 0, 0.014)
  book.rotation.x = -0.9
  book.visible = false
  return { mug, book }
}

/** Kurir LLM membawa amplop di tangan kanan. */
function buildEnvelope(foreR: THREE.Group, ctx: BuildContext) {
  const envelope = ctx.group(foreR, 0, -0.2, 0.07)
  ctx.add(envelope, G.box(0.24, 0.16, 0.025), ctx.mat('#f8fafc'), 0, 0, 0)
  ctx.add(envelope, G.dot(0.03), ctx.mat('#dc2626'), 0, 0, 0.012, [1, 1, 0.4])
  envelope.rotation.x = 1.2
  return envelope
}

/** Bintang pusing (gagal), Zzz (tertidur), konfeti (merayakan); semuanya tersembunyi sampai dibutuhkan. */
function buildStatusIndicators(root: THREE.Group, ctx: BuildContext) {
  const stars = ctx.group(root, 0, HEAD_TOP + 0.12, 0)
  const starMat = new THREE.MeshBasicMaterial({ color: '#facc15' })
  ctx.materials.push(starMat)
  for (let i = 0; i < 3; i++) {
    const a = (i / 3) * TAU
    stars.add(new THREE.Mesh(G.octa(0.055), starMat).translateX(Math.cos(a) * 0.26).translateZ(Math.sin(a) * 0.26))
  }
  stars.visible = false

  const zzz = ctx.group(root)
  for (let i = 0; i < 3; i++) {
    const m = new THREE.SpriteMaterial({ map: zzzTexture(), transparent: true, depthWrite: false })
    ctx.materials.push(m)
    const z = new THREE.Sprite(m)
    z.scale.setScalar(0.12 + i * 0.045)
    zzz.add(z)
  }
  zzz.visible = false

  const confetti = ctx.group(root, 0, HEAD_TOP + 0.1, 0)
  ;['#f43f5e', '#facc15', '#22c55e', '#38bdf8', '#a855f7', '#fb923c', '#f472b6', '#34d399'].forEach((color) => {
    const m = new THREE.MeshBasicMaterial({ color, side: THREE.DoubleSide })
    ctx.materials.push(m)
    confetti.add(new THREE.Mesh(G.box(0.05, 0.05, 0.004), m))
  })
  confetti.visible = false
  return { stars, zzz, confetti }
}

/** Jumlah sendi yang dihaluskan × 3 komponen + posisi badan (lihat `JOINT_COUNT`). */
const JOINT_COUNT = 11

export function buildCharacter(look: AgentLook, kind: AgentWorker['kind']): Character3D {
  const ctx = createBuildContext()
  const { mat } = ctx
  const p: Palette = {
    skin: mat(look.skin, { roughness: 0.85 }),
    hair: mat(look.hair, { roughness: 0.9 }),
    shirt: mat(look.shirt),
    pants: mat(look.pants),
    dark: mat('#1f2937', { roughness: 0.35 }),
    shoe: mat('#27272a', { roughness: 0.5 }),
    white: mat('#f8fafc', { roughness: 0.5 })
  }
  const root = new THREE.Group()
  const body = ctx.group(root)

  const L = buildLeg(body, -0.085, p, ctx)
  const R = buildLeg(body, 0.085, p, ctx)
  const torso = buildTorso(body, look, p, ctx)
  const aL = buildArm(torso, -0.2, p, ctx)
  const aR = buildArm(torso, 0.2, p, ctx)
  const { head, eyes, mouth } = buildHead(torso, look, p, ctx)
  const tail = buildHair(head, look, p.hair, ctx)
  buildFacialHair(head, look, p.hair, ctx)
  ACCESSORIES[look.accessory]?.(head, p, look, kind, ctx)
  const { mug, book } = buildHandProps(aL.fore, aR.fore, ctx)
  const envelope = kind === 'llm_call' ? buildEnvelope(aR.fore, ctx) : undefined
  const { stars, zzz, confetti } = buildStatusIndicators(root, ctx)

  // Gabungkan mesh tiap sendi menjadi satu (warna per-vertex): ±12 draw call per karakter, satu material untuk semua.
  const groups: THREE.Object3D[] = []
  body.traverse((o) => { if ((o as THREE.Group).isGroup) groups.push(o) })
  const geometries = groups.map(g => bakeVertexColors(g, bodyMaterial())).filter(g => g !== null)
  // Material per warna hanya pembawa warna saat membangun; setelah digabung tidak dipakai lagi.
  ctx.materials.splice(0).forEach((m) => { if (m instanceof THREE.MeshStandardMaterial) m.dispose(); else ctx.materials.push(m) })

  return {
    root, body, torso, head, legL: L.leg, legR: R.leg, shinL: L.shin, shinR: R.shin, armL: aL.arm, armR: aR.arm, foreL: aL.fore, foreR: aR.fore,
    eyes, mouth, tail, mug, book, stars, zzz, confetti, envelope,
    materials: ctx.materials, geometries, smooth: new Float32Array(JOINT_COUNT * 3 + 3), posed: false
  }
}

/** Gestur tambahan di atas mood: berbicara (rapat/obrolan), membaca (perpustakaan), minum kopi (rehat). */
export type CharacterGesture = 'talk' | 'read' | 'coffee'

export interface PoseState {
  walking: boolean
  sitting: boolean
  mood: ActorMood
  celebrating: boolean
  /** Detik sejak mulai (jam animasi). */
  t: number
  /** Fase acak per karakter agar tidak bergerak serempak. */
  phase: number
  gesture?: CharacterGesture
  /** Selisih waktu frame (detik): perpindahan pose dihaluskan; tanpa nilai ini pose langsung diterapkan. */
  dt?: number
}

interface Face { eye: number, mouthW: number, mouthH: number }

/** Lengan menekuk ke depan: `upper` di bahu, `lower` di siku (negatif = ke depan). */
const bend = (arm: THREE.Group, fore: THREE.Group, upper: number, lower: number, splay = 0) => {
  arm.rotation.x = upper
  arm.rotation.z = splay
  fore.rotation.x = lower
}

function poseWalk(c: Character3D, t: number) {
  const w = t * 9.5
  const s = Math.sin(w)
  const k = Math.cos(w)
  c.legL.rotation.x = -s * 0.62
  c.legR.rotation.x = s * 0.62
  c.shinL.rotation.x = Math.max(0, k) * 0.85 + 0.08
  c.shinR.rotation.x = Math.max(0, -k) * 0.85 + 0.08
  bend(c.armL, c.foreL, s * 0.55, -0.35 - Math.max(0, -s) * 0.3, -0.1)
  if (!c.envelope) bend(c.armR, c.foreR, -s * 0.55, -0.35 - Math.max(0, s) * 0.3, 0.1)
  c.body.position.y = Math.abs(k) * 0.035
  c.torso.rotation.set(0.07, s * 0.09, s * 0.03)
  c.head.rotation.set(-0.04, -s * 0.06, 0)
}

/** Menyeruput kopi secara berkala; mengembalikan true saat cangkir di mulut. */
function sip(c: Character3D, t: number, rest: [number, number]) {
  const cycle = t % 6.5
  const k = cycle < 1.6 ? Math.sin((cycle / 1.6) * Math.PI) : 0
  bend(c.armR, c.foreR, rest[0] + (-0.75 - rest[0]) * k, rest[1] + (-2.05 - rest[1]) * k, 0.1 - k * 0.25)
  if (k > 0.2) c.head.rotation.x = -0.12 * k
  return k > 0.7
}

function poseTalk(c: Character3D, t: number, seated: boolean) {
  bend(c.armR, c.foreR, -0.5 + Math.sin(t * 2.7) * 0.22, -1.15 + Math.sin(t * 4.1) * 0.32, 0.12 + Math.sin(t * 1.9) * 0.14)
  bend(c.armL, c.foreL, seated ? -0.45 : -0.1, seated ? -1.0 : -0.5 + Math.sin(t * 3.3 + 1) * 0.18, -0.12)
  c.head.rotation.set(Math.sin(t * 2.4) * 0.07, Math.sin(t * 0.8) * 0.32, Math.sin(t * 1.3) * 0.05)
}

function poseRead(c: Character3D, t: number) {
  const turn = t % 7 < 0.6 ? Math.sin(((t % 7) / 0.6) * Math.PI) : 0
  bend(c.armL, c.foreL, -0.62, -1.25, 0.18)
  bend(c.armR, c.foreR, -0.62 - turn * 0.15, -1.25 - turn * 0.25, -0.18 - turn * 0.2)
  c.head.rotation.set(0.3, Math.sin(t * 0.6) * 0.1, 0)
}

function poseSit(c: Character3D, s: PoseState, t: number, face: Face) {
  const working = s.mood === 'working'
  c.body.position.y = 0.05
  c.legL.rotation.x = -Math.PI / 2
  c.legR.rotation.x = -Math.PI / 2
  // Kaki pendek menggantung dari kursi: berayun pelan saat santai.
  const swing = working ? 0.06 : 0.24
  c.shinL.rotation.x = Math.PI / 2 - 0.12 + Math.sin(t * 2.1) * swing
  c.shinR.rotation.x = Math.PI / 2 - 0.12 + Math.sin(t * 2.1 + 2.4) * swing

  if (s.gesture === 'read') { c.torso.rotation.x = 0.06; poseRead(c, t); return }
  if (s.gesture === 'talk' && working) { c.torso.rotation.x = 0.04; poseTalk(c, t, true); face.mouthH = 0.45 + Math.abs(Math.sin(t * 9)) * 0.75; return }
  if (s.gesture === 'coffee') {
    c.torso.rotation.x = -0.08
    bend(c.armL, c.foreL, -0.3, -0.9, -0.1)
    c.head.rotation.set(0, Math.sin(t * 0.5) * 0.4, 0)
    if (sip(c, t, [-0.35, -1.1])) face.mouthW = 0.5
    return
  }
  if (working) {
    // Mengetik; sesekali berhenti dan berpikir dengan tangan di dagu.
    const thinking = t % 15 > 12.2
    c.torso.rotation.x = thinking ? 0 : 0.1
    bend(c.armL, c.foreL, -0.9, -0.9 + (thinking ? 0 : Math.sin(t * 17) * 0.11), -0.06)
    if (thinking) bend(c.armR, c.foreR, -0.95, -2.0, -0.28)
    else bend(c.armR, c.foreR, -0.9, -0.9 + Math.cos(t * 15) * 0.11, 0.06)
    c.head.rotation.set(thinking ? -0.12 : 0.12 + Math.sin(t * 1.7) * 0.02, thinking ? 0.2 : Math.sin(t * 0.7) * 0.13, thinking ? 0.12 : 0)
    return
  }
  // Santai di kursi: bersandar, melihat sekeliling.
  c.torso.rotation.x = -0.12
  bend(c.armL, c.foreL, -0.3, -0.9, -0.1)
  bend(c.armR, c.foreR, -0.3, -0.9, 0.1)
  c.head.rotation.set(Math.sin(t * 0.33) * 0.08, Math.sin(t * 0.5) * 0.5, 0)
}

function poseStand(c: Character3D, s: PoseState, t: number, face: Face) {
  c.body.position.y = Math.sin(t * 2) * 0.006
  c.body.position.x = Math.sin(t * 0.6) * 0.012
  c.torso.rotation.z = Math.sin(t * 0.6) * 0.03
  bend(c.armL, c.foreL, Math.sin(t * 1.1) * 0.05, -0.18, -0.1)
  if (!c.envelope) bend(c.armR, c.foreR, Math.sin(t * 1.1 + 1) * 0.05, -0.18, 0.1)
  c.head.rotation.set(0, Math.sin(t * 0.45) * 0.35, 0)

  if (s.gesture === 'read') { poseRead(c, t); return }
  if (s.gesture === 'coffee' && !c.envelope) {
    if (sip(c, t, [-0.4, -1.45])) face.mouthW = 0.5
    return
  }
  if (c.envelope) return
  if (s.mood === 'working' || s.gesture === 'talk') {
    poseTalk(c, t, false)
    face.mouthH = 0.45 + Math.abs(Math.sin(t * 9)) * 0.75
    return
  }
  if (s.mood === 'done' || s.mood === 'left') {
    // Melambai
    bend(c.armR, c.foreR, -2.75, -0.35, 0.35 + Math.sin(t * 8) * 0.3)
    face.eye = 0.4
    face.mouthH = 0.8
  }
}

/** Mood di atas pose dasar: tertidur, pusing (gagal). */
function poseMood(c: Character3D, s: PoseState, t: number, face: Face) {
  if (s.mood === 'sleeping') {
    c.torso.rotation.x = s.sitting ? 0.28 : 0.1
    c.head.rotation.set(0.5, 0, 0.16)
    bend(c.armL, c.foreL, -0.2, -0.5, -0.1)
    bend(c.armR, c.foreR, -0.2, -0.5, 0.1)
    face.eye = 0.1
    face.mouthW = 0.45
    face.mouthH = 0.55
  }
  if (s.mood === 'error') {
    // Memegang kepala dan menggeleng
    c.torso.rotation.x = 0.06
    bend(c.armL, c.foreL, -2.45, -1.25, 0.45)
    bend(c.armR, c.foreR, -2.45, -1.25, -0.45)
    c.head.rotation.set(0.1, Math.sin(t * 7) * 0.24, Math.sin(t * 3.5) * 0.06)
    face.eye = 0.55
    face.mouthW = 0.6
    face.mouthH = 0.9
  }
}

function poseCelebrate(c: Character3D, t: number, face: Face) {
  const jump = Math.abs(Math.sin(t * 6.5))
  c.body.position.y += jump * 0.24
  c.body.scale.set(1 - (jump - 0.5) * 0.08, 1 + (jump - 0.5) * 0.14, 1 - (jump - 0.5) * 0.08)
  c.legL.rotation.x = -jump * 0.25
  c.legR.rotation.x = -jump * 0.25
  c.shinL.rotation.x = jump * 0.7
  c.shinR.rotation.x = jump * 0.7
  bend(c.armL, c.foreL, -2.85, -0.25, -0.3 - Math.sin(t * 13) * 0.22)
  bend(c.armR, c.foreR, -2.85, -0.25, 0.3 + Math.sin(t * 13) * 0.22)
  c.torso.rotation.set(-0.08, 0, 0)
  c.head.rotation.set(-0.22, 0, Math.sin(t * 6.5) * 0.08)
  face.eye = 0.35
  face.mouthW = 1.25
  face.mouthH = 1.1
}

/** Sendi yang dihaluskan antar-frame agar perpindahan pose (duduk, berdiri, berhenti jalan) tidak meloncat. */
const joints = (c: Character3D) => [c.body, c.torso, c.head, c.legL, c.legR, c.shinL, c.shinR, c.armL, c.armR, c.foreL, c.foreR]

function blendJoints(c: Character3D, dt: number | undefined) {
  const a = c.posed && dt !== undefined ? 1 - Math.exp(-dt * 18) : 1
  const sm = c.smooth
  const list = joints(c)
  list.forEach((j, i) => {
    const o = i * 3
    sm[o] = sm[o]! + (j.rotation.x - sm[o]!) * a
    sm[o + 1] = sm[o + 1]! + (j.rotation.y - sm[o + 1]!) * a
    sm[o + 2] = sm[o + 2]! + (j.rotation.z - sm[o + 2]!) * a
    j.rotation.set(sm[o]!, sm[o + 1]!, sm[o + 2]!)
  })
  const o = list.length * 3
  sm[o] = sm[o]! + (c.body.position.x - sm[o]!) * a
  sm[o + 1] = sm[o + 1]! + (c.body.position.y - sm[o + 1]!) * a
  c.body.position.x = sm[o]!
  c.body.position.y = sm[o + 1]!
  c.posed = true
}

function poseFace(c: Character3D, s: PoseState, t: number, face: Face) {
  // Kedipan: selang berbeda tiap karakter.
  const period = 2.8 + (s.phase % 2.3)
  const blink = (t * 1.0 + s.phase * 3.1) % period < 0.13
  const eye = blink ? Math.min(face.eye, 0.1) : face.eye
  c.eyes.scale.set(1, eye, 1)
  c.mouth.scale.set(face.mouthW, face.mouthH, 1)
}

function poseExtras(c: Character3D, s: PoseState, t: number) {
  // Napas
  const slow = s.mood === 'sleeping'
  const b = Math.sin(t * (slow ? 1.1 : 1.9)) * (slow ? 0.03 : 0.014)
  c.torso.scale.set(1 + b * 0.6, 1 + b, 1 + b * 1.4)
  if (c.tail) c.tail.rotation.x = 0.25 + (s.walking ? Math.sin(t * 19) * 0.22 : Math.sin(t * 1.3) * 0.05)

  const seated = s.sitting && !s.walking
  c.mug.visible = s.gesture === 'coffee' && !s.walking && !c.envelope && s.mood !== 'error' && !s.celebrating
  c.book.visible = s.gesture === 'read' && !s.walking && s.mood !== 'error' && !s.celebrating
  // Cangkir tetap tegak walau lengan bergerak.
  if (c.mug.visible) c.mug.rotation.x = -(c.armR.rotation.x + c.foreR.rotation.x + c.torso.rotation.x)

  c.stars.visible = s.mood === 'error'
  if (c.stars.visible) c.stars.rotation.y = t * 4
  c.zzz.visible = s.mood === 'sleeping'
  if (c.zzz.visible) {
    c.zzz.children.forEach((z, i) => {
      const k = (t * 0.35 + i / 3) % 1
      z.position.set(0.2 + k * 0.28 + Math.sin(t * 2 + i) * 0.03, HEAD_TOP - (seated ? 0.12 : 0) + k * 0.5, 0.05)
      ;((z as THREE.Sprite).material as THREE.SpriteMaterial).opacity = Math.sin(k * Math.PI)
    })
  }
  c.confetti.visible = s.celebrating
  if (c.confetti.visible) {
    c.confetti.children.forEach((f, i) => {
      const k = (t * 0.9 + i * 0.37) % 1
      const a = i * 0.785 + t * 1.5
      f.position.set(Math.cos(a) * (0.2 + k * 0.35), 0.45 - k * 0.9, Math.sin(a) * (0.2 + k * 0.35))
      f.rotation.set(t * 6 + i, t * 4 + i * 2, 0)
    })
  }
}

/** Pose per frame: jalan, duduk mengetik/berpikir, berbicara, membaca, minum kopi, santai, tertidur, pusing (gagal), merayakan. */
export function poseCharacter(c: Character3D, s: PoseState) {
  const t = s.t + s.phase
  for (const j of joints(c)) j.rotation.set(0, 0, 0)
  c.body.position.set(0, 0, 0)
  c.body.scale.set(1, 1, 1)
  c.armL.rotation.z = -0.1
  c.armR.rotation.z = 0.1
  if (c.envelope) bend(c.armR, c.foreR, -0.35, -1.15, 0.05)
  const face: Face = { eye: 1, mouthW: 1, mouthH: 1 }

  if (s.walking) poseWalk(c, t)
  else if (s.sitting) poseSit(c, s, t, face)
  else poseStand(c, s, t, face)
  if (!s.walking) poseMood(c, s, t, face)
  if (s.celebrating) poseCelebrate(c, t, face)

  blendJoints(c, s.dt)
  poseFace(c, s, t, face)
  poseExtras(c, s, t)
}

export function disposeCharacter(c: Character3D) {
  c.geometries.forEach(g => g.dispose())
  c.materials.forEach(m => m.dispose())
}
