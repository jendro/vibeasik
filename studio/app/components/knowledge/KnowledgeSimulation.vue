<script setup lang="ts">
import type { KnowledgeHit, KnowledgeKind, KnowledgePromptPreview } from '~/types'
import { KIND_META } from '~/utils/knowledge'

/**
 * Simulasi temu-kembali di peta jaring: memperagakan apa yang terjadi saat pekerja AI menerima satu pekerjaan —
 * pekerjaan dipahami, butir yang cocok langsung menyala, tautan disusuri ke butir tetangga, lalu ringkasan apa yang
 * dibawa ke pekerja AI & masalah apa yang dicegah. Memanggil POST /api/knowledge/preview: pegawai dipilih leader &
 * prompt disusun persis seperti todo sungguhan, tetapi todo tidak dibuat dan pemakaian/celah tidak dicatat.
 */
// neighborsOf: tetangga bertaut (di peta) dari sekumpulan butir — yang ikut ditimbang di langkah penyusuran tautan.
const props = defineProps<{ total: number, neighborsOf: (ids: number[]) => number[] }>()
const emit = defineEmits<{
  (e: 'start', ids: number[]): void
  (e: 'lit', ids: number[]): void
  (e: 'ghost', ids: number[]): void
  (e: 'fit', ids: number[]): void
  (e: 'open', id: number): void
  (e: 'preview', p: KnowledgePromptPreview | null): void
  (e: 'reset'): void
}>()

const api = useApi()

// Proyek opsional: dengan proyek, pekerjaan lintas aplikasi (backend/frontend/mobile) dipecah per aplikasi —
// satu prompt per pegawai sesuai keahliannya.
const projects = ref<{ label: string, value: number }[]>([])
const projectId = ref(0)
onMounted(async () => {
  try {
    projects.value = (await api.getProjects()).map(p => ({ label: p.name, value: p.id }))
  } catch {
    projects.value = []
  }
})

const SCENARIOS = [
  'Buat endpoint login API Go dengan JWT',
  'Audit kebocoran secret di file .env sebelum deploy',
  'Rapikan halaman checkout Flutter agar responsif',
  'Daftar produk lambat karena query database'
]

// Apa yang diselesaikan tiap jenis butir saat dibawa ke pekerja AI.
const SOLVES: Record<KnowledgeKind, string> = {
  rule: 'standar wajib — AI tidak melanggar aturan keamanan & arsitektur',
  procedure: 'langkah kerja terbukti — AI tidak menebak-nebak cara',
  lesson: 'kesalahan lama — bug & vonis audit yang sama tidak terulang',
  concept: 'konsep & angka acuan — AI paham target yang harus dicapai'
}

const text = ref('')
const step = ref(0) // 0 diam · 1 memahami · 2 cocok langsung · 3 susuri tautan · 4 dibawa ke AI · 5 dinilai & belajar
const result = ref<KnowledgePromptPreview | null>(null)
const considered = ref(0)
const error = ref('')
let runId = 0

const direct = computed(() => (result.value?.hits || []).filter(h => h.via !== 'tautan'))
const linked = computed(() => (result.value?.hits || []).filter(h => h.via === 'tautan'))
const byKind = computed(() => {
  const g = new Map<KnowledgeKind, KnowledgeHit[]>()
  for (const h of result.value?.hits || []) g.set(h.kind, [...(g.get(h.kind) || []), h])
  return [...g.entries()]
})

const wait = (ms: number) => new Promise(r => setTimeout(r, ms))

const run = async (q?: string) => {
  if (q) text.value = q
  if (!text.value.trim()) return
  const id = ++runId
  error.value = ''
  result.value = null
  considered.value = 0
  step.value = 1
  emit('lit', [])
  emit('ghost', [])
  emit('preview', null)
  let res: KnowledgePromptPreview
  try {
    res = await api.previewKnowledgePrompt({ text: text.value.trim(), project_id: projectId.value || undefined })
  } catch (err: any) {
    error.value = err?.data?.error || err?.message || 'Gagal menjalankan temu-kembali'
    step.value = 0
    return
  }
  if (id !== runId) return
  result.value = res
  const hits = res.hits || []
  const hitIds = new Set(hits.map(h => h.id))
  const around = props.neighborsOf(direct.value.map(h => h.id)).filter(n => !hitIds.has(n))
  emit('start', [...hitIds, ...around])
  await wait(900)
  if (id !== runId) return

  // Kecocokan langsung menyala satu per satu, lalu tetangga lewat tautan.
  step.value = 2
  emit('fit', hits.map(h => h.id))
  const shown: number[] = []
  for (const h of direct.value) {
    shown.push(h.id)
    emit('lit', [...shown])
    await wait(260)
    if (id !== runId) return
  }
  await wait(500)
  // Tetangga bertaut ikut ditimbang (menyala samar); yang skornya cukup masuk, sisanya padam.
  step.value = 3
  considered.value = around.length
  emit('ghost', around)
  await wait(around.length ? 1600 : 300)
  if (id !== runId) return
  for (const h of linked.value) {
    shown.push(h.id)
    emit('lit', [...shown])
    await wait(380)
    if (id !== runId) return
  }
  await wait(linked.value.length ? 600 : 200)
  if (id !== runId) return
  emit('ghost', [])
  step.value = 4
  emit('preview', res)
  await wait(1200)
  if (id !== runId) return
  step.value = 5
}

const reset = () => {
  runId++
  step.value = 0
  result.value = null
  emit('preview', null)
  emit('reset')
}

const STEPS = [
  { n: 1, label: 'Memahami pekerjaan', icon: 'i-lucide-scan-text' },
  { n: 2, label: 'Mencari butir yang cocok', icon: 'i-lucide-search' },
  { n: 3, label: 'Menyusuri tautan', icon: 'i-lucide-waypoints' },
  { n: 4, label: 'Dibawa ke pekerja AI', icon: 'i-lucide-bot' },
  { n: 5, label: 'Dinilai & belajar', icon: 'i-lucide-graduation-cap' }
]
const pct = (v: number) => `${Math.round(v * 100)}%`
defineExpose({ run, text })
</script>

<template>
  <div class="space-y-3">
    <div>
      <p class="text-xs uppercase font-semibold text-neutral-500">
        Simulasi pekerjaan AI
      </p>
      <p class="text-xs text-neutral-500 mt-1 leading-relaxed">
        Tulis satu pekerjaan, lalu lihat butir mana yang diambil pekerja AI dari jaring ini.
      </p>
    </div>
    <form class="flex gap-1.5" @submit.prevent="run()">
      <UInput v-model="text" placeholder="mis. buat fitur upload foto profil" class="flex-1" size="sm" />
      <UButton type="submit" size="sm" icon="i-lucide-play" :loading="step === 1" />
    </form>
    <USelect
      v-model="projectId"
      :items="[{ label: 'Tanpa proyek (pekerjaan umum)', value: 0 }, ...projects]"
      size="sm"
      class="w-full"
    />
    <p v-if="projectId" class="text-xs text-neutral-500 -mt-1.5">
      Bila pekerjaannya menyentuh beberapa aplikasi proyek (backend, frontend, mobile), todo dipecah per aplikasi.
    </p>
    <div v-if="step === 0" class="flex flex-wrap gap-1.5">
      <button
        v-for="s in SCENARIOS"
        :key="s"
        type="button"
        class="text-left text-xs px-2 py-1 rounded-md border border-neutral-200 dark:border-neutral-700 text-neutral-600 dark:text-neutral-300 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-400 transition"
        @click="run(s)"
      >
        {{ s }}
      </button>
    </div>
    <p v-if="error" class="text-xs text-rose-600 dark:text-rose-400">
      {{ error }}
    </p>

    <template v-if="step > 0">
      <ol class="space-y-1.5">
        <li
          v-for="s in STEPS"
          :key="s.n"
          class="flex items-center gap-2 text-xs transition-opacity"
          :class="step >= s.n ? 'opacity-100' : 'opacity-35'"
        >
          <span
            class="size-5 rounded-full flex items-center justify-center shrink-0"
            :class="step > s.n ? 'bg-emerald-500 text-white' : step === s.n ? 'bg-primary-500 text-white animate-pulse' : 'bg-neutral-200 dark:bg-neutral-800 text-neutral-500'"
          >
            <UIcon :name="step > s.n ? 'i-lucide-check' : s.icon" class="size-3" />
          </span>
          <span class="font-medium text-neutral-800 dark:text-neutral-200">{{ s.label }}</span>
          <span v-if="s.n === 2 && step >= 2" class="ml-auto text-neutral-500">{{ direct.length }} butir</span>
          <span v-if="s.n === 3 && step >= 3" class="ml-auto text-neutral-500">{{ considered }} ditimbang · +{{ linked.length }} masuk</span>
        </li>
      </ol>

      <p v-if="step === 3" class="text-xs text-neutral-500 leading-relaxed">
        Butir yang bertaut ke hasil teratas ikut ditimbang (garis putus-putus) — begitu cara AI menemukan pelajaran terkait walau katanya berbeda. Hanya yang skornya cukup tinggi yang masuk.
      </p>
      <div v-if="step >= 1 && result" class="text-xs text-neutral-500 leading-relaxed">
        Dikenali sebagai:
        <UBadge v-for="d in result.domains || []" :key="d" size="xs" color="neutral" variant="outline" class="mr-1">
          {{ d }}
        </UBadge>
        <span v-if="!(result.domains || []).length">pekerjaan umum (tanpa ekosistem khusus)</span>
      </div>
      <div v-if="step >= 1 && result?.parts?.length" class="text-xs text-neutral-500 leading-relaxed space-y-0.5">
        <p>Menyentuh {{ result.parts.length }} aplikasi → dipecah per aplikasi:</p>
        <p v-for="p in result.parts" :key="p.title">
          • {{ p.apps.join(', ') }} → <b class="text-neutral-800 dark:text-neutral-200">{{ p.employee?.name || 'belum ada pegawai' }}</b>
        </p>
      </div>
      <div v-else-if="step >= 1 && result?.employee" class="text-xs text-neutral-500 leading-relaxed">
        Dikerjakan oleh <b class="text-neutral-800 dark:text-neutral-200">{{ result.employee.name }}</b> — dipilih leader sesuai keahlian & beban kerja.
      </div>

      <div v-if="step >= 4 && result" class="space-y-2.5">
        <div class="p-2.5 rounded-lg bg-primary-50 dark:bg-primary-950/30 text-xs text-neutral-700 dark:text-neutral-300 leading-relaxed">
          Dari <b>{{ props.total }}</b> butir, hanya <b>{{ result.hits?.length || 0 }}</b> yang paling relevan & terbukti dibawa ke prompt pekerja AI — bukan seluruh guideline sekaligus. Prompt final & semua yang dihasilkan tampil di bawah peta.
        </div>
        <div v-for="[k, hs] in byKind" :key="k" class="space-y-1">
          <p class="text-xs font-semibold text-neutral-700 dark:text-neutral-300">
            <UIcon :name="KIND_META[k].icon" class="size-3 align-[-1px]" /> {{ KIND_META[k].label }} ({{ hs.length }}) · <span class="font-normal text-neutral-500">{{ SOLVES[k] }}</span>
          </p>
          <button
            v-for="h in hs"
            :key="h.id"
            type="button"
            class="w-full text-left flex items-start gap-1.5 px-2 py-1 rounded-md hover:bg-neutral-100 dark:hover:bg-neutral-800 text-xs"
            @click="emit('open', h.id)"
          >
            <UIcon :name="h.via === 'tautan' ? 'i-lucide-waypoints' : 'i-lucide-target'" class="size-3 mt-0.5 shrink-0 text-neutral-400" :title="h.via === 'tautan' ? 'Ditemukan lewat tautan' : 'Cocok langsung'" />
            <span class="flex-1 min-w-0 truncate text-neutral-800 dark:text-neutral-200">{{ h.title }}</span>
            <span class="text-xs text-neutral-400 shrink-0">{{ pct(h.score) }}</span>
          </button>
        </div>
      </div>

      <div v-if="step >= 5" class="p-2.5 rounded-lg bg-emerald-50 dark:bg-emerald-950/30 text-xs text-neutral-700 dark:text-neutral-300 leading-relaxed">
        Setelah pekerjaan selesai, nilai QA & vonis audit kembali ke butir-butir ini: yang membantu <b>✓</b> naik dan makin sering terambil (node membesar), yang keliru <b>✗</b> turun lalu ditinjau. Topik yang belum ada butirnya dicatat sebagai celah untuk dipelajari.
      </div>

      <div class="flex gap-2">
        <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-rotate-ccw" @click="reset">
          Selesai
        </UButton>
        <UButton v-if="step >= 5" size="xs" color="neutral" variant="ghost" icon="i-lucide-repeat" @click="run()">
          Ulangi
        </UButton>
      </div>
    </template>
  </div>
</template>
