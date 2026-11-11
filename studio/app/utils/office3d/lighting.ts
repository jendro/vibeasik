import * as THREE from 'three'
import { OFFICE_THEME, hourOf, nightFactor } from './theme'
import { glowTexture } from './textures'

/** Material menyala yang kekuatannya mengikuti waktu (dibuat sekali, dipakai banyak perabot). */
export interface LightMaterials {
  /** Bola lampu jalan & lampu mobil. */
  lamp: THREE.MeshStandardMaterial
  /** Kap lampu ruangan, lampu meja, strip LED dinding. */
  bulb: THREE.MeshStandardMaterial
  /** Lampu hias kafe & lentera taman. */
  string: THREE.MeshStandardMaterial
  /** Kaca jendela: indeks 0 = gedung Model LLM, 1.. = lantai gedung (dalam), lalu fasad luar per lantai. */
  windows: THREE.MeshStandardMaterial[]
  /** Genangan cahaya di lantai/tanah di bawah lampu (aditif, hanya terlihat saat gelap). */
  pool: THREE.MeshBasicMaterial
}

export function createLightMaterials(own: <T extends { dispose: () => void }>(o: T) => T): LightMaterials {
  const glow = own(glowTexture())
  return {
    lamp: own(new THREE.MeshStandardMaterial({ color: '#fef3c7', emissive: '#fde68a', emissiveIntensity: 0.2 })),
    bulb: own(new THREE.MeshStandardMaterial({ color: '#fff7e0', emissive: '#ffd58a', emissiveIntensity: 0.3 })),
    string: own(new THREE.MeshStandardMaterial({ color: '#fff1c7', emissive: '#ffb74a', emissiveIntensity: 0.2 })),
    windows: [],
    pool: own(new THREE.MeshBasicMaterial({
      map: glow, color: '#ffc978', transparent: true, opacity: 0, blending: THREE.AdditiveBlending, depthWrite: false, toneMapped: false
    }))
  }
}

export interface SkyState {
  /** Warna langit & kabut saat ini. */
  color: THREE.Color
  /** 0 = siang penuh, 1 = malam penuh. */
  night: number
}

export interface LightingOptions {
  /** Ukuran peta (meter) dan tinggi satu lantai. */
  width: number
  depth: number
  floorHeight: number
  /** Titik lampu ruangan per lantai (meter, relatif alas lantai). */
  interior: THREE.Vector3[][]
  /** Titik lampu luar (kafe, taman, pintu depan, plaza). */
  outdoor: Array<{ at: THREE.Vector3, color?: string, power?: number }>
}

export interface Lighting {
  hemi: THREE.HemisphereLight
  sun: THREE.DirectionalLight
  /** Pakai lampu ruangan lantai yang sedang dilihat, atau lampu luar (null = taman/peta luar). */
  setFloor: (floor: number | null) => void
  setTime: (date: Date, busyFloors: boolean[]) => SkyState
  update: (t: number) => void
}

const color = (hex: string) => new THREE.Color().setStyle(hex, THREE.SRGBColorSpace)
const lerp = (a: number, b: number, k: number) => a + (b - a) * k

/** Bintang di kubah langit + bulatan matahari/bulan. */
function createSky(root: THREE.Group, o: LightingOptions, own: <T extends { dispose: () => void }>(v: T) => T) {
  const center = new THREE.Vector3(o.width / 2, 0, o.depth / 2)
  const pts: number[] = []
  let seed = 7
  const rnd = () => { seed = (seed * 1103515245 + 12345) >>> 0; return seed / 4294967296 }
  for (let i = 0; i < 420; i++) {
    const a = rnd() * Math.PI * 2
    const e = 0.06 + rnd() * 1.4
    pts.push(center.x + Math.cos(a) * Math.cos(e) * 125, Math.sin(e) * 125, center.z + Math.sin(a) * Math.cos(e) * 125)
  }
  const starGeo = own(new THREE.BufferGeometry())
  starGeo.setAttribute('position', new THREE.Float32BufferAttribute(pts, 3))
  const starMat = own(new THREE.PointsMaterial({ color: '#e0e7ff', size: 1.8, sizeAttenuation: false, transparent: true, opacity: 0, fog: false, depthWrite: false }))
  const stars = new THREE.Points(starGeo, starMat)
  stars.frustumCulled = false
  const orbMat = own(new THREE.MeshBasicMaterial({ color: '#fff7d6', fog: false, toneMapped: false }))
  const orb = new THREE.Mesh(own(new THREE.SphereGeometry(4.5, 20, 14)), orbMat)
  root.add(stars, orb)
  return { center, stars, starMat, orb, orbMat }
}

export function createLighting(root: THREE.Group, mats: LightMaterials, o: LightingOptions, own: <T extends { dispose: () => void }>(v: T) => T): Lighting {
  const hemi = new THREE.HemisphereLight('#dbeafe', '#6b8f5e', 1.4)
  const sun = new THREE.DirectionalLight('#fff7e6', 2.4)
  sun.castShadow = true
  sun.shadow.mapSize.set(2048, 2048)
  const sc = sun.shadow.camera
  sc.left = -15; sc.right = 15; sc.top = 12; sc.bottom = -12; sc.near = 1; sc.far = 60
  sun.shadow.bias = -0.0005
  sun.shadow.normalBias = 0.02
  sun.target.position.set(o.width / 2, 0, o.depth / 2)
  root.add(hemi, sun, sun.target)

  // Satu rangkaian lampu titik berjumlah tetap (tanpa bayangan), dipakai bergantian: lampu ruangan lantai yang dilihat,
  // atau lampu luar saat melihat taman/peta luar. Jumlah lampu yang tetap mencegah shader dikompilasi ulang, dan
  // rangkaian tunggal menjaga biaya per piksel tetap ringan di GPU terpadu.
  const count = Math.max(o.outdoor.length, ...o.interior.map(f => f.length))
  const rig = Array.from({ length: count }, () => {
    const l = new THREE.PointLight('#ffffff', 0, 10, 1.5)
    root.add(l)
    return l
  })
  /** Kekuatan relatif & warna khusus tiap lampu pada susunan saat ini (0 = padam). */
  const power = new Float32Array(count)
  const tint: Array<string | undefined> = Array.from({ length: count })
  let indoors = false
  let night = 0
  const applyRig = () => {
    const th = OFFICE_THEME
    const base = indoors ? lerp(th.lamp.interiorDay, th.lamp.interiorNight, night) : night * th.lamp.outdoorNight
    rig.forEach((l, i) => {
      l.color.copy(color(tint[i] ?? th.lamp.color))
      l.intensity = base * power[i]!
    })
  }
  const setFloor = (floor: number | null) => {
    indoors = floor !== null
    const f = Math.min(o.interior.length - 1, Math.max(0, floor ?? 0))
    rig.forEach((l, i) => {
      const spot = indoors ? o.interior[f]?.[i] : o.outdoor[i]?.at
      power[i] = !spot ? 0 : indoors ? 1 : o.outdoor[i]!.power ?? 1
      tint[i] = indoors ? undefined : o.outdoor[i]?.color
      if (spot) l.position.set(spot.x, spot.y + (indoors ? f * o.floorHeight : 0), spot.z)
    })
    applyRig()
  }
  setFloor(null)

  const sky = createSky(root, o, own)
  const state: SkyState = { color: new THREE.Color(), night: 0 }
  const tmp = new THREE.Color()

  const setTime = (date: Date, busyFloors: boolean[]): SkyState => {
    const th = OFFICE_THEME
    const hour = hourOf(date)
    const n = nightFactor(hour)
    const day = Math.max(0, Math.sin(((hour - 6) / 12) * Math.PI))
    // Malam: bulan menempuh busur yang sama dua belas jam kemudian.
    const moon = n > 0.5
    const h = moon ? (hour + 12) % 24 : hour
    const az = ((h - 6) / 12) * Math.PI
    const lift = moon ? Math.max(0.3, Math.sin(az)) : day
    sun.position.set(o.width / 2 + Math.cos(az) * 18, 6 + lift * 16, o.depth / 2 - 8 + Math.sin(az) * 2)
    sun.intensity = lerp(0.9 + day * (th.sun.dayIntensity - 0.9), th.sun.moonIntensity, n)
    sun.color.copy(color(th.sun.day)).lerp(tmp.copy(color(th.sun.low)), 1 - Math.min(1, day / 0.35)).lerp(tmp.copy(color(th.sun.moon)), n)
    hemi.intensity = lerp(th.ambient.dayIntensity - 0.35 + day * 0.35, th.ambient.nightIntensity, n)
    hemi.color.copy(color(th.ambient.day)).lerp(tmp.copy(color(th.ambient.night)), n)
    hemi.groundColor.copy(color(th.ambient.ground)).lerp(tmp.set('#2b3352'), n)

    night = n
    applyRig()
    mats.lamp.emissiveIntensity = 0.1 + n * 1.9
    mats.bulb.emissiveIntensity = 0.3 + n * 1.7
    mats.string.emissiveIntensity = 0.2 + n * 2
    mats.pool.opacity = n * th.lamp.glowNight
    mats.windows.forEach((m, i) => {
      if (i === 0) { m.emissiveIntensity = 0.25 + n * 0.75; return } // gedung LLM
      const busy = busyFloors[(i - 1) % Math.max(1, busyFloors.length)] ?? false
      m.emissiveIntensity = busy ? lerp(th.window.dayBusy, th.window.nightBusy, n) : n * th.window.nightIdle
    })

    // Langit: siang → senja → malam
    state.night = n
    state.color.copy(color(th.sky.day))
    if (n < 0.5) state.color.lerp(tmp.copy(color(th.sky.dusk)), n * 2)
    else state.color.copy(color(th.sky.dusk)).lerp(tmp.copy(color(th.sky.night)), (n - 0.5) * 2)
    sky.starMat.opacity = Math.max(0, (n - 0.4) / 0.6) * 0.95
    sky.stars.visible = sky.starMat.opacity > 0.01
    sky.orb.position.copy(sun.position).sub(sun.target.position).normalize().multiplyScalar(112).add(sky.center)
    sky.orbMat.color.set(moon ? '#e8edff' : day < 0.35 ? '#ffc98a' : '#fff7d6')
    return state
  }

  const update = (t: number) => {
    // Lampu hias berkedip sangat halus agar terasa hidup.
    if (state.night > 0.05) mats.string.emissiveIntensity = (0.2 + state.night * 2) * (0.9 + Math.sin(t * 2.3) * 0.1)
  }

  return { hemi, sun, setFloor, setTime, update }
}
