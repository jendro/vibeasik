// Apakah semua aplikasi proyek sudah berada di satu folder induk (root proyek)?
// Flag is_unified hanya diisi saat "Satukan Folder" dijalankan; proyek yang sejak awal
// tersusun rapi di bawah root-nya juga dianggap satu induk agar lencana "Terpencar" tidak keliru.
interface LayoutProject {
  is_unified?: boolean
  root_path?: string
  components?: { absolute_path?: string }[]
}

export const isProjectUnified = (p: LayoutProject): boolean => {
  if (p.is_unified) return true
  const root = (p.root_path || '').replace(/\/+$/, '')
  if (!root || root === '-') return false
  const comps = p.components || []
  if (comps.length === 0) return false
  return comps.every((c) => {
    const abs = (c.absolute_path || '').replace(/\/+$/, '')
    return abs === root || abs.startsWith(root + '/')
  })
}
