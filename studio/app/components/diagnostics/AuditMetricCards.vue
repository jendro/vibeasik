<script setup lang="ts">
// Jumlah temuan per kelompok kategori; klik untuk memfilter daftar (klik lagi = semua).
// Pengelompokan harus sama dengan filter di useAuditFindingFilters (code_smell+architecture, security+port, frontend+mobile).
const props = defineProps<{
  categoryCounts: Record<string, number>
  severityCounts: Record<string, number>
  totalFindings: number
  activeCategory: string
}>()

const emit = defineEmits<{
  (e: 'selectCategory', cat: string): void
}>()

const c = (...keys: string[]) => keys.reduce((n, k) => n + (props.categoryCounts[k] || 0), 0)
const toggle = (key: string) => emit('selectCategory', props.activeCategory === key ? 'all' : key)

const items = computed(() => [
  { key: 'deep', label: 'Deep Check', value: c('deep'), hint: 'Latensi rute, validasi form, risiko kueri' },
  { key: 'query', label: 'Database & kueri', value: c('query'), hint: 'N+1, SQL injection, SELECT *' },
  { key: 'router', label: 'Router & API', value: c('router'), hint: 'Validasi, status code, kontrak API' },
  { key: 'code_smell', label: 'Kode & arsitektur', value: c('code_smell', 'architecture'), hint: 'Code smell, duplikasi, struktur' },
  { key: 'security', label: 'Keamanan & port', value: c('security', 'port'), hint: 'Rahasia, XSS, CORS, port bentrok' },
  { key: 'ui', label: 'UI & mobile', value: c('frontend', 'mobile'), hint: 'Aksesibilitas, responsif, tampilan' }
].map(it => ({
  label: it.label,
  value: it.value,
  hint: it.hint,
  active: props.activeCategory === it.key,
  tone: it.value > 0 ? undefined : 'text-neutral-300 dark:text-slate-600',
  title: `${it.hint} — klik untuk ${props.activeCategory === it.key ? 'menampilkan semua' : 'memfilter'}`,
  action: () => toggle(it.key)
})))
</script>

<template>
  <AppStatStrip :items="items" />
</template>
