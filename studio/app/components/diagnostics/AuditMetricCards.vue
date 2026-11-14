<script setup lang="ts">
// Jumlah temuan per area perbaikan (Finding.area dari engine, sama dengan panel "Tingkatkan aplikasi" di ringkasan
// proyek); klik untuk memfilter daftar (klik lagi = semua). Kategori teknis tetap di bar filter di bawahnya.
import { FINDING_AREAS } from '~/utils/improvementFlow'

const props = defineProps<{
  areaCounts: Record<string, number>
  activeArea: string
}>()

const emit = defineEmits<{
  (e: 'selectArea', area: string): void
}>()

const toggle = (key: string) => emit('selectArea', props.activeArea === key ? 'all' : key)

const items = computed(() => FINDING_AREAS.map((a) => {
  const value = props.areaCounts[a.key] || 0
  const active = props.activeArea === a.key
  return {
    label: a.label,
    value,
    hint: a.desc,
    active,
    tone: value > 0 ? undefined : 'text-neutral-300 dark:text-slate-600',
    title: `${a.desc} — klik untuk ${active ? 'menampilkan semua' : 'memfilter'}`,
    action: () => toggle(a.key)
  }
}))
</script>

<template>
  <AppStatStrip :items="items" />
</template>
