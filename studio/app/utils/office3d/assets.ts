import * as THREE from 'three'
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js'
import { applyOfficeTheme } from './theme'

/**
 * Aset 3D opsional untuk Kantor Agent. Setiap perabot prosedural ditandai `userData.prop` (lihat `PROP_NAMES`);
 * bila `public/office3d/assets.json` memetakan nama itu ke file .glb/.gltf, perabot diganti model tersebut —
 * otomatis diskalakan agar muat di kotak perabot aslinya (alas di lantai, titik tengah sama).
 *
 * Contoh assets.json:
 *   { "models": { "desk": { "url": "/office3d/desk.glb" }, "tree": { "url": "/office3d/tree.glb", "rotation": 90 } } }
 * Opsi per model: `scale` (abaikan auto-fit), `rotation` (derajat, sumbu Y), `offset` [x, y, z] meter, `fit` (default true).
 *
 * Manifest yang sama boleh memuat `theme` untuk mengubah suasana (warna langit, kekuatan lampu, jam senja) tanpa
 * mengubah kode — lihat `OfficeTheme`. Contoh: { "theme": { "sky": { "night": "#0b1030" }, "lamp": { "interiorNight": 22 } } }
 */
export const PROP_NAMES = [
  'desk', 'chair', 'plant', 'sofa', 'bookshelf', 'server_rack', 'tree', 'bush', 'bench', 'lamp_post',
  'round_table', 'cafe_chair', 'counter', 'meeting_table', 'coffee_machine', 'car', 'flipchart', 'fridge', 'tv',
  'floor_lamp', 'desk_lamp', 'rug', 'water_cooler', 'lantern'
] as const
export type PropName = typeof PROP_NAMES[number]

export interface AssetSpec { url: string, scale?: number, rotation?: number, offset?: [number, number, number], fit?: boolean }
interface LoadedAsset { scene: THREE.Object3D, spec: AssetSpec }

export const ASSET_MANIFEST_URL = '/office3d/assets.json'

/**
 * Muat semua model yang terdaftar; manifest yang tidak ada / model yang gagal dimuat dilewati (perabot prosedural tetap).
 * Kunci `theme` di manifest langsung diterapkan ke tema scene.
 */
export async function loadOfficeAssets(): Promise<Map<string, LoadedAsset>> {
  const out = new Map<string, LoadedAsset>()
  let models: Record<string, AssetSpec>
  try {
    const res = await fetch(ASSET_MANIFEST_URL, { cache: 'no-cache' })
    if (!res.ok) return out
    const manifest = await res.json()
    applyOfficeTheme(manifest?.theme)
    models = manifest?.models ?? {}
  } catch {
    return out
  }
  const loader = new GLTFLoader()
  await Promise.all(Object.entries(models).map(async ([name, spec]) => {
    if (!spec?.url) return
    try {
      const gltf = await loader.loadAsync(spec.url)
      gltf.scene.traverse((o) => {
        if ((o as THREE.Mesh).isMesh) { o.castShadow = true; o.receiveShadow = true }
      })
      out.set(name, { scene: gltf.scene, spec })
    } catch (e) {
      console.warn(`[office3d] gagal memuat aset "${name}" (${spec.url})`, e)
    }
  }))
  return out
}

/** Ganti isi setiap grup perabot bertanda `userData.prop` dengan model aset yang dimuat. */
export function applyOfficeAssets(root: THREE.Object3D, assets: Map<string, LoadedAsset>) {
  if (!assets.size) return
  const targets: THREE.Object3D[] = []
  root.traverse((o) => { if (o.userData.prop && assets.has(o.userData.prop) && !o.userData.assetApplied) targets.push(o) })
  const box = new THREE.Box3()
  const size = new THREE.Vector3()
  const center = new THREE.Vector3()
  for (const g of targets) {
    const asset = assets.get(g.userData.prop)!
    // Kotak perabot asli dalam koordinat lokalnya
    const parent = g.parent
    const saved = { p: g.position.clone(), q: g.quaternion.clone(), s: g.scale.clone() }
    parent?.remove(g)
    g.position.set(0, 0, 0)
    g.quaternion.identity()
    g.scale.set(1, 1, 1)
    g.updateMatrixWorld(true)
    box.setFromObject(g)
    const fbSize = box.getSize(new THREE.Vector3())
    const fbCenter = box.getCenter(new THREE.Vector3())
    const fbBottom = box.min.y
    for (const c of [...g.children]) {
      g.remove(c)
      c.traverse(o => (o as THREE.Mesh).geometry?.dispose())
    }

    const model = asset.scene.clone(true)
    model.rotation.y = THREE.MathUtils.degToRad(asset.spec.rotation ?? 0)
    model.updateMatrixWorld(true)
    box.setFromObject(model)
    box.getSize(size)
    let k = asset.spec.scale ?? 1
    if (asset.spec.scale === undefined && asset.spec.fit !== false && size.x > 0 && size.y > 0 && size.z > 0) {
      k = Math.min(fbSize.x / size.x, fbSize.y / size.y, fbSize.z / size.z)
    }
    model.scale.setScalar(k)
    model.updateMatrixWorld(true)
    box.setFromObject(model)
    box.getCenter(center)
    const [ox, oy, oz] = asset.spec.offset ?? [0, 0, 0]
    model.position.set(fbCenter.x - center.x + ox, fbBottom - box.min.y + oy, fbCenter.z - center.z + oz)
    g.add(model)
    g.userData.assetApplied = true

    g.position.copy(saved.p)
    g.quaternion.copy(saved.q)
    g.scale.copy(saved.s)
    parent?.add(g)
  }
}
