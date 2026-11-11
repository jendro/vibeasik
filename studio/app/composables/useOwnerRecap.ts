import type { Ref } from 'vue'
import type { OwnerRecap } from '~/types'

// Cache per proyek+rentang selama sesi: pindah tab tidak memuat ulang; tombol segarkan memaksa ambil baru.
const cache = new Map<string, OwnerRecap>()

export function useOwnerRecap(projectId: Ref<number>, days: Ref<number>) {
  const { getOwnerRecap } = useApi()
  const recap = shallowRef<OwnerRecap | null>(null)
  const loading = ref(false)
  const failed = ref('')

  const load = async (force = false) => {
    const key = `${projectId.value}:${days.value}`
    const hit = cache.get(key)
    if (hit && !force) {
      recap.value = hit
      return
    }
    loading.value = true
    failed.value = ''
    try {
      const res = await getOwnerRecap(projectId.value, days.value)
      cache.set(key, res)
      // Jawaban untuk proyek/rentang yang sudah ditinggalkan tidak boleh menimpa tampilan sekarang.
      if (key === `${projectId.value}:${days.value}`) recap.value = res
    } catch (err: any) {
      failed.value = err?.data?.error || err?.message || 'Gagal memuat rekap proyek'
    } finally {
      loading.value = false
    }
  }

  watch([projectId, days], () => load(), { immediate: true })
  return { recap, loading, failed, load }
}
