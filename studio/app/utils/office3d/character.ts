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

export function buildCharacter(look: AgentLook, kind: AgentWorker['kind']): Character3D {
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

  const skin = mat(look.skin)
  const hair = mat(look.hair)
  const shirt = mat(look.shirt)
  const pants = mat(look.pants)
  const dark = mat('#1f2937', { roughness: 0.4 })
  const shoe = mat('#27272a')

  const root = new THREE.Group()
  const body = new THREE.Group()
  root.add(body)

  // Kaki (poros di pinggul)
  const leg = (x: number) => {
    const g = new THREE.Group()
    g.position.set(x, HIP_Y, 0)
    g.add(box(0.13, 0.36, 0.15, pants, 0, -0.18, 0))
    g.add(box(0.14, 0.07, 0.2, shoe, 0, -0.385, 0.03))
    body.add(g)
    return g
  }
  const legL = leg(-0.09)
  const legR = leg(0.09)

  // Badan
  body.add(box(0.4, 0.44, 0.26, shirt, 0, HIP_Y + 0.21, 0))
  body.add(box(0.41, 0.08, 0.27, pants, 0, HIP_Y + 0.02, 0)) // ikat pinggang
  if (look.accessory === 'tie') body.add(box(0.06, 0.26, 0.02, mat('#b91c1c'), 0, HIP_Y + 0.26, 0.135))
  if (look.accessory === 'scarf' || look.hairStyle === 'hijab') {
    const g = new THREE.TorusGeometry(0.15, 0.05, 8, 18)
    geometries.push(g)
    const scarf = new THREE.Mesh(g, look.hairStyle === 'hijab' ? hair : mat('#dc2626'))
    scarf.rotation.x = Math.PI / 2
    scarf.position.y = HIP_Y + 0.44
    body.add(scarf)
  }

  // Lengan (poros di bahu)
  const arm = (x: number) => {
    const g = new THREE.Group()
    g.position.set(x, HIP_Y + 0.4, 0)
    g.add(box(0.11, 0.34, 0.12, shirt, 0, -0.15, 0))
    g.add(box(0.1, 0.08, 0.1, skin, 0, -0.33, 0))
    body.add(g)
    return g
  }
  const armL = arm(-0.26)
  const armR = arm(0.26)

  // Kurir LLM membawa amplop
  let envelope: THREE.Mesh | undefined
  if (kind === 'llm_call') {
    envelope = box(0.22, 0.15, 0.03, mat('#f8fafc'), 0, -0.3, 0.1)
    envelope.add(box(0.05, 0.05, 0.035, mat('#dc2626'), 0, 0, 0))
    armR.add(envelope)
    armR.rotation.x = -0.5
  }

  // Kepala
  const head = new THREE.Group()
  head.position.y = HIP_Y + 0.46
  body.add(head)
  head.add(box(0.4, 0.38, 0.36, skin, 0, 0.2, 0))
  head.add(box(0.06, 0.07, 0.02, dark, -0.09, 0.22, 0.185))
  head.add(box(0.06, 0.07, 0.02, dark, 0.09, 0.22, 0.185))
  head.add(box(0.1, 0.025, 0.02, mat('#be6b5a'), 0, 0.11, 0.185))

  const cut = look.hairStyle ?? ('short' as const)
  if (cut === 'hijab') {
    head.add(box(0.46, 0.45, 0.36, hair, 0, 0.21, -0.04))
  } else if (cut !== 'bald') {
    head.add(box(0.43, 0.12, 0.4, hair, 0, 0.42, -0.01))
    head.add(box(0.43, 0.3, 0.08, hair, 0, 0.27, -0.17))
    if (cut === 'long') head.add(box(0.42, 0.42, 0.1, hair, 0, 0.12, -0.18))
    if (cut === 'bun') head.add(sphere(0.1, hair, 0, 0.53, -0.08))
    if (cut === 'ponytail') head.add(box(0.1, 0.3, 0.1, hair, 0, 0.18, -0.25))
    if (cut === 'curly') [-0.14, 0, 0.14].forEach(x => head.add(sphere(0.09, hair, x, 0.47, 0)))
    if (cut === 'spiky') {
      [-0.12, 0, 0.12].forEach((x) => {
        const g = new THREE.ConeGeometry(0.06, 0.16, 6)
        geometries.push(g)
        const c = new THREE.Mesh(g, hair)
        c.position.set(x, 0.53, 0)
        head.add(c)
      })
    }
  }
  if (look.facial === 'beard') head.add(box(0.36, 0.12, 0.06, hair, 0, 0.06, 0.17))
  if (look.facial === 'mustache') head.add(box(0.14, 0.03, 0.03, hair, 0, 0.14, 0.19))

  if (look.accessory === 'glasses') {
    head.add(box(0.1, 0.08, 0.015, dark, -0.09, 0.22, 0.195))
    head.add(box(0.1, 0.08, 0.015, dark, 0.09, 0.22, 0.195))
    head.add(box(0.08, 0.015, 0.015, dark, 0, 0.24, 0.195))
  }
  if (look.accessory === 'headset') {
    const g = new THREE.TorusGeometry(0.23, 0.02, 6, 20, Math.PI)
    geometries.push(g)
    const band = new THREE.Mesh(g, dark)
    band.position.y = 0.24
    head.add(band)
    head.add(box(0.05, 0.1, 0.1, dark, -0.22, 0.2, 0))
    head.add(box(0.05, 0.1, 0.1, dark, 0.22, 0.2, 0))
    head.add(box(0.02, 0.02, 0.16, dark, -0.21, 0.12, 0.1))
  }
  if (look.accessory === 'cap') {
    const cap = mat(kind === 'llm_call' ? look.shirt : '#1d4ed8')
    head.add(box(0.44, 0.1, 0.4, cap, 0, 0.44, 0))
    head.add(box(0.4, 0.025, 0.16, cap, 0, 0.4, 0.26))
  }
  if (look.accessory === 'beanie') head.add(sphere(0.23, mat('#e11d48'), 0, 0.4, 0, 0.62))

  // Bintang pusing (gagal) & Zzz (tertidur)
  const stars = new THREE.Group()
  stars.position.y = HEAD_TOP + 0.12
  const starMat = new THREE.MeshBasicMaterial({ color: '#facc15' })
  materials.push(starMat)
  const starGeo = new THREE.OctahedronGeometry(0.05)
  geometries.push(starGeo)
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
  materials.push(zMat)
  for (let i = 0; i < 3; i++) {
    const g = new THREE.BoxGeometry(0.06 + i * 0.02, 0.06 + i * 0.02, 0.01)
    geometries.push(g)
    const z = new THREE.Mesh(g, zMat)
    z.position.set(0.15 + i * 0.1, HEAD_TOP + i * 0.12, 0)
    z.rotation.z = 0.5
    zzz.add(z)
  }
  zzz.visible = false
  root.add(zzz)

  return { root, body, head, legL, legR, armL, armR, stars, zzz, envelope, materials, geometries }
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
