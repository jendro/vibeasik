import * as THREE from 'three'
import type { ActorMood, AgentWorker } from '~/types'
import type { AgentLook } from '~/utils/agentOffice'

/**
 * Karakter 3D chibi prosedural (satuan meter, menghadap +Z, titik asal di kaki).
 * Warna kulit/rambut/baju/celana, gaya rambut, rambut wajah, dan aksesori dari `AgentLook` yang sama dengan avatar 2D.
 */
export interface Character3D {
  root: THREE.Group
  body: THREE.Group
  head: THREE.Group
  legL: THREE.Group
  legR: THREE.Group
  armL: THREE.Group
  armR: THREE.Group
  stars: THREE.Group
  zzz: THREE.Group
  envelope?: THREE.Mesh
  materials: THREE.Material[]
  geometries: THREE.BufferGeometry[]
}

export const HIP_Y = 0.42
export const HEAD_TOP = 1.32

interface BuildContext {
  materials: THREE.Material[]
  geometries: THREE.BufferGeometry[]
  mat: (color: string, extra?: THREE.MeshStandardMaterialParameters) => THREE.MeshStandardMaterial
  box: (w: number, h: number, d: number, m: THREE.Material, x?: number, y?: number, z?: number) => THREE.Mesh
  sphere: (r: number, m: THREE.Material, x?: number, y?: number, z?: number, sy?: number) => THREE.Mesh
}

/** Daftar material/geometri yang dibuat karakter (untuk dispose) beserta pembuat mesh dasarnya. */
function createBuildContext(): BuildContext {
  const materials: THREE.Material[] = []
  const geometries: THREE.BufferGeometry[] = []
  const mat = (color: string, extra: THREE.MeshStandardMaterialParameters = {}) => {
    const m = new THREE.MeshStandardMaterial({ color: new THREE.Color().setStyle(color, THREE.SRGBColorSpace), roughness: 0.75, ...extra })
    materials.push(m)
    return m
  }
  const box = (w: number, h: number, d: number, m: THREE.Material, x = 0, y = 0, z = 0) => {
    const g = new THREE.BoxGeometry(w, h, d)
    geometries.push(g)
    const mesh = new THREE.Mesh(g, m)
    mesh.position.set(x, y, z)
    mesh.castShadow = true
    return mesh
  }
  const sphere = (r: number, m: THREE.Material, x = 0, y = 0, z = 0, sy = 1) => {
    const g = new THREE.SphereGeometry(r, 14, 10)
    geometries.push(g)
    const mesh = new THREE.Mesh(g, m)
    mesh.position.set(x, y, z)
    mesh.scale.y = sy
    mesh.castShadow = true
    return mesh
  }
  return { materials, geometries, mat, box, sphere }
}

interface Palette {
  skin: THREE.Material
  hair: THREE.Material
  shirt: THREE.Material
  pants: THREE.Material
  dark: THREE.Material
  shoe: THREE.Material
}

/** Kaki: poros di pinggul. */
function buildLegs(body: THREE.Group, p: Palette, ctx: BuildContext) {
  const leg = (x: number) => {
    const g = new THREE.Group()
    g.position.set(x, HIP_Y, 0)
    g.add(ctx.box(0.13, 0.36, 0.15, p.pants, 0, -0.18, 0))
    g.add(ctx.box(0.14, 0.07, 0.2, p.shoe, 0, -0.385, 0.03))
    body.add(g)
    return g
  }
  const legL = leg(-0.09)
  const legR = leg(0.09)
  return { legL, legR }
}

/** Badan beserta dasi/syal (hijab memakai material rambut). */
function buildTorso(body: THREE.Group, look: AgentLook, p: Palette, ctx: BuildContext) {
  body.add(ctx.box(0.4, 0.44, 0.26, p.shirt, 0, HIP_Y + 0.21, 0))
  body.add(ctx.box(0.41, 0.08, 0.27, p.pants, 0, HIP_Y + 0.02, 0))
  if (look.accessory === 'tie') body.add(ctx.box(0.06, 0.26, 0.02, ctx.mat('#b91c1c'), 0, HIP_Y + 0.26, 0.135))
  if (look.accessory !== 'scarf' && look.hairStyle !== 'hijab') return
  const g = new THREE.TorusGeometry(0.15, 0.05, 8, 18)
  ctx.geometries.push(g)
  const scarf = new THREE.Mesh(g, look.hairStyle === 'hijab' ? p.hair : ctx.mat('#dc2626'))
  scarf.rotation.x = Math.PI / 2
  scarf.position.y = HIP_Y + 0.44
  body.add(scarf)
}

/** Lengan: poros di bahu. */
function buildArms(body: THREE.Group, p: Palette, ctx: BuildContext) {
  const arm = (x: number) => {
    const g = new THREE.Group()
    g.position.set(x, HIP_Y + 0.4, 0)
    g.add(ctx.box(0.11, 0.34, 0.12, p.shirt, 0, -0.15, 0))
    g.add(ctx.box(0.1, 0.08, 0.1, p.skin, 0, -0.33, 0))
    body.add(g)
    return g
  }
  const armL = arm(-0.26)
  const armR = arm(0.26)
  return { armL, armR }
}

/** Kurir LLM membawa amplop di lengan kanan. */
function buildEnvelope(armR: THREE.Group, ctx: BuildContext) {
  const envelope = ctx.box(0.22, 0.15, 0.03, ctx.mat('#f8fafc'), 0, -0.3, 0.1)
  envelope.add(ctx.box(0.05, 0.05, 0.035, ctx.mat('#dc2626'), 0, 0, 0))
  armR.add(envelope)
  armR.rotation.x = -0.5
  return envelope
}

/** Kepala + wajah (mata & mulut). */
function buildHead(body: THREE.Group, p: Palette, ctx: BuildContext) {
  const head = new THREE.Group()
  head.position.y = HIP_Y + 0.46
  body.add(head)
  head.add(ctx.box(0.4, 0.38, 0.36, p.skin, 0, 0.2, 0))
  head.add(ctx.box(0.06, 0.07, 0.02, p.dark, -0.09, 0.22, 0.185))
  head.add(ctx.box(0.06, 0.07, 0.02, p.dark, 0.09, 0.22, 0.185))
  head.add(ctx.box(0.1, 0.025, 0.02, ctx.mat('#be6b5a'), 0, 0.11, 0.185))
  return head
}

const HAIR_EXTRAS: Partial<Record<NonNullable<AgentLook['hairStyle']>, (head: THREE.Group, hair: THREE.Material, ctx: BuildContext) => void>> = {
  long: (head, hair, ctx) => { head.add(ctx.box(0.42, 0.42, 0.1, hair, 0, 0.12, -0.18)) },
  bun: (head, hair, ctx) => { head.add(ctx.sphere(0.1, hair, 0, 0.53, -0.08)) },
  ponytail: (head, hair, ctx) => { head.add(ctx.box(0.1, 0.3, 0.1, hair, 0, 0.18, -0.25)) },
  curly: (head, hair, ctx) => { [-0.14, 0, 0.14].forEach(x => head.add(ctx.sphere(0.09, hair, x, 0.47, 0))) },
  spiky: (head, hair, ctx) => {
    [-0.12, 0, 0.12].forEach((x) => {
      const g = new THREE.ConeGeometry(0.06, 0.16, 6)
      ctx.geometries.push(g)
      const spike = new THREE.Mesh(g, hair)
      spike.position.set(x, 0.53, 0)
      head.add(spike)
    })
  }
}

/** Gaya rambut: hijab/botak lebih dulu, selain itu dasar rambut + tambahan per gaya. */
function buildHairStyle(head: THREE.Group, cut: NonNullable<AgentLook['hairStyle']>, hair: THREE.Material, ctx: BuildContext) {
  if (cut === 'hijab') {
    head.add(ctx.box(0.46, 0.45, 0.36, hair, 0, 0.21, -0.04))
    return
  }
  if (cut === 'bald') return
  head.add(ctx.box(0.43, 0.12, 0.4, hair, 0, 0.42, -0.01))
  head.add(ctx.box(0.43, 0.3, 0.08, hair, 0, 0.27, -0.17))
  HAIR_EXTRAS[cut]?.(head, hair, ctx)
}

const ACCESSORIES: Partial<Record<AgentLook['accessory'], (head: THREE.Group, dark: THREE.Material, look: AgentLook, kind: AgentWorker['kind'], ctx: BuildContext) => void>> = {
  glasses: (head, dark, _look, _kind, ctx) => {
    head.add(ctx.box(0.1, 0.08, 0.015, dark, -0.09, 0.22, 0.195))
    head.add(ctx.box(0.1, 0.08, 0.015, dark, 0.09, 0.22, 0.195))
    head.add(ctx.box(0.08, 0.015, 0.015, dark, 0, 0.24, 0.195))
  },
  headset: (head, dark, _look, _kind, ctx) => {
    const g = new THREE.TorusGeometry(0.23, 0.02, 6, 20, Math.PI)
    ctx.geometries.push(g)
    const band = new THREE.Mesh(g, dark)
    band.position.y = 0.24
    head.add(band)
    head.add(ctx.box(0.05, 0.1, 0.1, dark, -0.22, 0.2, 0))
    head.add(ctx.box(0.05, 0.1, 0.1, dark, 0.22, 0.2, 0))
    head.add(ctx.box(0.02, 0.02, 0.16, dark, -0.21, 0.12, 0.1))
  },
  cap: (head, _dark, look, kind, ctx) => {
    const cap = ctx.mat(kind === 'llm_call' ? look.shirt : '#1d4ed8')
    head.add(ctx.box(0.44, 0.1, 0.4, cap, 0, 0.44, 0))
    head.add(ctx.box(0.4, 0.025, 0.16, cap, 0, 0.4, 0.26))
  },
  beanie: (head, _dark, _look, _kind, ctx) => { head.add(ctx.sphere(0.23, ctx.mat('#e11d48'), 0, 0.4, 0, 0.62)) }
}

/** Aksesori kepala (kacamata, headset, topi, beanie). Dasi & syal ada di badan. */
function buildAccessory(head: THREE.Group, dark: THREE.Material, look: AgentLook, kind: AgentWorker['kind'], ctx: BuildContext) {
  ACCESSORIES[look.accessory]?.(head, dark, look, kind, ctx)
}

/** Bintang pusing (gagal) & Zzz (tertidur); keduanya tersembunyi sampai dibutuhkan. */
function buildStatusIndicators(root: THREE.Group, ctx: BuildContext) {
  const stars = new THREE.Group()
  stars.position.y = HEAD_TOP + 0.12
  const starMat = new THREE.MeshBasicMaterial({ color: '#facc15' })
  ctx.materials.push(starMat)
  const starGeo = new THREE.OctahedronGeometry(0.05)
  ctx.geometries.push(starGeo)
  for (let i = 0; i < 3; i++) {
    const s = new THREE.Mesh(starGeo, starMat)
    const a = (i / 3) * Math.PI * 2
    s.position.set(Math.cos(a) * 0.24, 0, Math.sin(a) * 0.24)
    stars.add(s)
  }
  stars.visible = false
  root.add(stars)

  const zzz = new THREE.Group()
  const zMat = new THREE.MeshBasicMaterial({ color: '#93c5fd' })
  ctx.materials.push(zMat)
  for (let i = 0; i < 3; i++) {
    const g = new THREE.BoxGeometry(0.06 + i * 0.02, 0.06 + i * 0.02, 0.01)
    ctx.geometries.push(g)
    const z = new THREE.Mesh(g, zMat)
    z.position.set(0.15 + i * 0.1, HEAD_TOP + i * 0.12, 0)
    z.rotation.z = 0.5
    zzz.add(z)
  }
  zzz.visible = false
  root.add(zzz)
  return { stars, zzz }
}

export function buildCharacter(look: AgentLook, kind: AgentWorker['kind']): Character3D {
  const ctx = createBuildContext()
  const { mat } = ctx
  const p: Palette = {
    skin: mat(look.skin),
    hair: mat(look.hair),
    shirt: mat(look.shirt),
    pants: mat(look.pants),
    dark: mat('#1f2937', { roughness: 0.4 }),
    shoe: mat('#27272a')
  }
  const root = new THREE.Group()
  const body = new THREE.Group()
  root.add(body)

  const { legL, legR } = buildLegs(body, p, ctx)
  buildTorso(body, look, p, ctx)
  const { armL, armR } = buildArms(body, p, ctx)
  const envelope = kind === 'llm_call' ? buildEnvelope(armR, ctx) : undefined
  const head = buildHead(body, p, ctx)
  buildHairStyle(head, look.hairStyle ?? 'short', p.hair, ctx)
  buildAccessory(head, p.dark, look, kind, ctx)
  const { stars, zzz } = buildStatusIndicators(root, ctx)

  return { root, body, head, legL, legR, armL, armR, stars, zzz, envelope, materials: ctx.materials, geometries: ctx.geometries }
}

export interface PoseState {
  walking: boolean
  sitting: boolean
  mood: ActorMood
  celebrating: boolean
  /** Detik sejak mulai (jam animasi). */
  t: number
  /** Fase acak per karakter agar tidak bergerak serempak. */
  phase: number
}

/** Pose per frame: jalan, duduk mengetik, santai, tertidur, pusing (gagal), merayakan. */
export function poseCharacter(c: Character3D, s: PoseState) {
  const t = s.t + s.phase
  const working = s.mood === 'working'
  c.body.position.set(0, 0, 0)
  c.body.rotation.set(0, 0, 0)
  c.head.rotation.set(0, 0, 0)
  c.legL.rotation.set(0, 0, 0)
  c.legR.rotation.set(0, 0, 0)
  c.armL.rotation.set(0, 0, 0)
  c.armR.rotation.set(c.envelope ? -0.5 : 0, 0, 0)

  if (s.walking) {
    const sw = Math.sin(t * 11) * 0.65
    c.legL.rotation.x = sw
    c.legR.rotation.x = -sw
    c.armL.rotation.x = -sw * 0.8
    if (!c.envelope) c.armR.rotation.x = sw * 0.8
    c.body.position.y = Math.abs(Math.cos(t * 11)) * 0.04
  } else if (s.sitting) {
    c.body.position.y = 0.05
    c.legL.rotation.x = -Math.PI / 2
    c.legR.rotation.x = -Math.PI / 2
    if (working) {
      c.armL.rotation.x = -1.25 + Math.sin(t * 16) * 0.08
      c.armR.rotation.x = -1.25 + Math.cos(t * 15) * 0.08
      c.head.rotation.x = 0.08
    } else {
      c.armL.rotation.x = -0.5
      c.armR.rotation.x = -0.5
    }
  } else {
    c.body.position.y = Math.sin(t * 2) * 0.008
    if (working && !c.envelope) c.armR.rotation.set(-0.3, 0, -0.15 + Math.sin(t * 3) * 0.1) // berbicara/menjelaskan
  }

  if (s.mood === 'sleeping') {
    c.head.rotation.x = 0.45
    c.body.rotation.x = 0.06
  }
  if (s.mood === 'error') {
    c.head.rotation.z = Math.sin(t * 6) * 0.18
    c.stars.rotation.y = t * 4
  }
  c.stars.visible = s.mood === 'error'
  c.zzz.visible = s.mood === 'sleeping'
  if (c.zzz.visible) c.zzz.position.y = (t * 0.25) % 0.25

  if (s.celebrating) {
    c.body.position.y += Math.abs(Math.sin(t * 7)) * 0.22
    c.armL.rotation.set(Math.PI, 0, -0.3)
    c.armR.rotation.set(Math.PI, 0, 0.3)
  }
}

export function disposeCharacter(c: Character3D) {
  c.geometries.forEach(g => g.dispose())
  c.materials.forEach(m => m.dispose())
}
