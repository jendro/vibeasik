import type { InitVibeasikResult } from '~/types'

/**
 * "Init vibeasik" sebuah proyek: menulis .vibeasik.json (tautan AI) ke root & setiap komponen, membangun
 * peta kode ke .vibeasik/CODEMAP.md (diperbarui otomatis oleh engine), dan menambahkan .vibeasik/ ke .gitignore.
 */
export const useInitVibeasik = () => {
  const { apiBase } = useApi()
  const toast = useToast()
  const initing = ref(false)

  const initProject = async (projectId: number, projectName = '') => {
    initing.value = true
    try {
      const res = await $fetch<InitVibeasikResult>(`${apiBase}/api/projects/${projectId}/init-vibeasik`, { method: 'POST' })
      const files = res.code_map.build.file_count
      toast.add({
        title: `vibeasik siap di ${projectName || 'proyek ini'}`,
        description: `.vibeasik.json di ${res.files_created.length} folder · peta kode ${files} berkas di ${res.code_map_markdown || '.vibeasik/CODEMAP.md'}`
          + (res.gitignore_updated ? ' · .vibeasik/ ditambahkan ke .gitignore' : '')
          + (res.agent_rules?.length ? ` · aturan agent: ${res.agent_rules.map(f => f.split('/').pop()).join(', ')}` : '')
          + ` · diperbarui otomatis tiap ${res.auto_sync_minutes} menit bila kode berubah.`,
        icon: 'i-lucide-sparkles',
        color: 'success'
      })
      return res
    } catch (err: any) {
      toast.add({
        title: 'Init vibeasik gagal',
        description: err?.data?.error || err?.message || 'Terjadi kesalahan',
        icon: 'i-lucide-alert-circle',
        color: 'error'
      })
      return null
    } finally {
      initing.value = false
    }
  }

  return { initing, initProject }
}
