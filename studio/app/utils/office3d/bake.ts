import * as THREE from 'three'
import { mergeGeometries } from 'three/examples/jsm/utils/BufferGeometryUtils.js'

/**
 * Penggabung mesh statis. Scene kantor tersusun dari ribuan bentuk dasar kecil; tiap mesh = satu draw call, dan
 * jumlah draw call itulah yang membatasi kecepatan di GPU terpadu. Mesh yang tidak bergerak sendiri-sendiri digabung
 * menjadi satu geometri per grup, sehingga perabot/karakter tetap berbentuk sama tetapi jauh lebih ringan digambar.
 */

type Leaf = THREE.Mesh<THREE.BufferGeometry, THREE.Material>

/** Anak langsung yang berupa mesh biasa (satu material, tanpa anak) — hanya ini yang aman digabung. */
const leaves = (group: THREE.Object3D, skip?: (o: THREE.Object3D) => boolean) => group.children.filter((c): c is Leaf => {
  const m = c as THREE.Mesh
  return !!m.isMesh && !Array.isArray(m.material) && !c.children.length && c.visible && !skip?.(c)
})

/** Salinan geometri mesh dalam koordinat induknya (tanpa indeks, agar bentuk berindeks & tidak bisa digabung). */
function flattened(mesh: Leaf) {
  mesh.updateMatrix()
  const g = mesh.geometry.index ? mesh.geometry.toNonIndexed() : mesh.geometry.clone()
  g.applyMatrix4(mesh.matrix)
  return g
}

function merged(parts: THREE.BufferGeometry[]) {
  const out = parts.length > 1 ? mergeGeometries(parts) : parts[0]!.clone()
  parts.forEach(p => p.dispose())
  return out
}

/**
 * Gabungkan mesh anak langsung `group` yang memakai material sama (dan pengaturan bayangan sama) menjadi satu mesh.
 * Material tetap objek yang sama, jadi warna/kilau yang diubah lewat material masih berlaku. Geometri hasil
 * gabungan diserahkan ke `own` untuk di-dispose pemiliknya.
 */
export function bakeByMaterial<T extends THREE.Object3D>(group: T, own: (g: THREE.BufferGeometry) => void): T {
  const buckets = new Map<string, Leaf[]>()
  for (const m of leaves(group)) {
    const key = `${m.material.uuid}|${m.castShadow}|${m.receiveShadow}|${m.renderOrder}`
    buckets.set(key, [...(buckets.get(key) ?? []), m])
  }
  for (const list of buckets.values()) {
    if (list.length < 2) continue
    const geo = merged(list.map(flattened))
    if (!geo) continue
    own(geo)
    const first = list[0]!
    const mesh = new THREE.Mesh(geo, first.material)
    mesh.castShadow = first.castShadow
    mesh.receiveShadow = first.receiveShadow
    mesh.renderOrder = first.renderOrder
    list.forEach(m => group.remove(m))
    group.add(mesh)
  }
  return group
}

/**
 * Gabungkan semua mesh anak langsung `group` menjadi satu mesh berwarna per-vertex (warna diambil dari material tiap
 * mesh) dengan `material` bersama. Dipakai karakter: satu draw call per sendi dan satu material untuk semua karakter.
 * Mengembalikan geometri gabungan (untuk di-dispose), atau null bila tidak ada yang digabung.
 */
export function bakeVertexColors(group: THREE.Object3D, material: THREE.Material, skip?: (o: THREE.Object3D) => boolean) {
  const list = leaves(group, skip).filter(m => (m.material as THREE.MeshStandardMaterial).color)
  if (!list.length) return null
  const geo = merged(list.map((m) => {
    const g = flattened(m)
    const c = (m.material as THREE.MeshStandardMaterial).color
    const n = g.getAttribute('position').count
    const colors = new Float32Array(n * 3)
    for (let i = 0; i < n; i++) colors.set([c.r, c.g, c.b], i * 3)
    g.setAttribute('color', new THREE.BufferAttribute(colors, 3))
    return g
  }))
  if (!geo) return null
  const mesh = new THREE.Mesh(geo, material)
  mesh.castShadow = list.some(m => m.castShadow)
  list.forEach(m => group.remove(m))
  group.add(mesh)
  return geo
}
