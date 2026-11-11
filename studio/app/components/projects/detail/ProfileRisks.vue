<script setup lang="ts">
import type { ProfileRisk, ProfileRiskStatus, ProjectProfile } from '~/types'

// Daftar risiko profil proyek beserta status penanganannya. Risiko disusun saat analisis profil (bukan
// pemantauan langsung); pemilik menindaklanjuti dengan menjadikannya todo, menandai beres, atau menerimanya.
const props = defineProps<{ projectId: number, risks: ProfileRisk[] }>()
const emit = defineEmits<{ updated: [profile: ProjectProfile] }>()

const toast = useToast()
const { createTodo, updateProjectRisk } = useApi()
const busy = ref('')

const SEVERITY: Record<ProfileRisk['severity'], { label: string, order: number, chip: string }> = {
  high: { label: 'Tinggi', order: 0, chip: 'bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-300' },
  medium: { label: 'Sedang', order: 1, chip: 'bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300' },
  low: { label: 'Rendah', order: 2, chip: 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-300' }
}
const STATUS: Record<ProfileRiskStatus, { label: string, icon: string, tone: string, done: boolean }> = {
  open: { label: 'Belum ditangani', icon: 'i-lucide-circle-alert', tone: 'text-rose-700 dark:text-rose-400', done: false },
  planned: { label: 'Dijadwalkan', icon: 'i-lucide-calendar-clock', tone: 'text-sky-700 dark:text-sky-400', done: false },
  in_progress: { label: 'Sedang dikerjakan', icon: 'i-lucide-loader', tone: 'text-sky-700 dark:text-sky-400', done: false },
  handled: { label: 'Sudah ditangani', icon: 'i-lucide-circle-check', tone: 'text-emerald-700 dark:text-emerald-400', done: true },
  accepted: { label: 'Diterima pemilik', icon: 'i-lucide-hand', tone: 'text-neutral-600 dark:text-neutral-300', done: true }
}
const statusOf = (r: ProfileRisk): ProfileRiskStatus => r.status || 'open'

// Yang belum beres di atas, lalu menurut tingkat bahaya.
const sorted = computed(() => [...props.risks].sort((a, b) =>
  Number(STATUS[statusOf(a)].done) - Number(STATUS[statusOf(b)].done) || SEVERITY[a.severity].order - SEVERITY[b.severity].order))
const doneCount = computed(() => props.risks.filter(r => STATUS[statusOf(r)].done).length)
const followedUp = computed(() => props.risks.filter(r => ['planned', 'in_progress'].includes(statusOf(r))).length)
const untouched = computed(() => props.risks.length - doneCount.value - followedUp.value)
const percent = computed(() => props.risks.length ? Math.round(doneCount.value / props.risks.length * 100) : 0)

const fail = (err: any) => toast.add({ title: 'Gagal Memperbarui Risiko', description: err?.data?.error || err?.message || 'Terjadi kesalahan', icon: 'i-lucide-alert-circle', color: 'error' })

const setStatus = async (r: ProfileRisk, status: 'open' | 'handled' | 'accepted') => {
  busy.value = r.title
  try {
    emit('updated', await updateProjectRisk(props.projectId, { title: r.title, status }))
  } catch (err) {
    fail(err)
  } finally {
    busy.value = ''
  }
}

const PRIORITY = { high: 'high', medium: 'medium', low: 'low' } as const
const makeTodo = async (r: ProfileRisk) => {
  busy.value = r.title
  try {
    const todo = await createTodo({
      project_id: props.projectId,
      level: 'project',
      title: `Tangani risiko: ${r.title}`.slice(0, 200),
      description: r.detail,
      priority: PRIORITY[r.severity],
      category: /keamanan|secret|rahasia|token|auth|kredensial|password/i.test(`${r.title} ${r.detail}`) ? 'security' : 'general'
    })
    emit('updated', await updateProjectRisk(props.projectId, { title: r.title, todo_id: todo.id }))
    toast.add({ title: 'Risiko Dijadikan Todo', description: `Todo #${todo.id} dibuat. Status risiko akan mengikuti todo itu.`, icon: 'i-lucide-check-circle', color: 'success' })
  } catch (err) {
    fail(err)
  } finally {
    busy.value = ''
  }
}
</script>

<template>
  <section class="rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] p-5 space-y-4">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="space-y-1 min-w-0">
        <h2 class="text-sm font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-triangle-alert" class="size-4 text-amber-500" />
          Risiko proyek
        </h2>
        <p class="text-sm text-neutral-600 dark:text-neutral-300 max-w-2xl leading-relaxed">
          Hal yang bisa merugikan proyek bila dibiarkan — celah keamanan, fitur setengah jadi, tanpa test, aplikasi mati.
          Disusun otomatis dari bukti di kode saat analisis profil terakhir, jadi daftar ini potret, bukan pemantauan langsung.
        </p>
      </div>
      <div v-if="risks.length" class="w-full sm:w-56 space-y-1.5">
        <div class="flex justify-between text-sm">
          <span class="font-semibold text-neutral-900 dark:text-white tabular-nums">{{ doneCount }} dari {{ risks.length }} beres</span>
          <span class="text-neutral-500 tabular-nums">{{ percent }}%</span>
        </div>
        <div class="h-1.5 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden" role="progressbar" :aria-valuenow="percent" aria-valuemin="0" aria-valuemax="100" aria-label="Risiko yang sudah beres">
          <div class="h-full rounded-full bg-emerald-500" :style="{ width: `${percent}%` }" />
        </div>
        <div class="text-xs text-neutral-500">
          {{ followedUp }} sedang ditindaklanjuti · {{ untouched }} belum disentuh
        </div>
      </div>
    </div>

    <p v-if="!risks.length" class="text-sm text-neutral-500">
      Tidak ada risiko yang tercatat pada analisis terakhir.
    </p>

    <ul v-else class="divide-y divide-neutral-200 dark:divide-white/[0.06]">
      <li v-for="r in sorted" :key="r.title" class="py-3 first:pt-0 last:pb-0 flex flex-col gap-2 lg:flex-row lg:items-start lg:gap-4">
        <div class="min-w-0 flex-1 space-y-1">
          <div class="flex flex-wrap items-center gap-2">
            <span class="rounded-md px-1.5 py-0.5 text-xs font-medium" :class="SEVERITY[r.severity].chip">{{ SEVERITY[r.severity].label }}</span>
            <span class="text-sm font-medium" :class="STATUS[statusOf(r)].done ? 'text-neutral-500 line-through decoration-neutral-300 dark:decoration-neutral-600' : 'text-neutral-900 dark:text-white'">{{ r.title }}</span>
          </div>
          <p v-if="r.detail" class="text-sm text-neutral-600 dark:text-neutral-300 leading-relaxed">
            {{ r.detail }}
          </p>
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
            <span class="inline-flex items-center gap-1 font-medium" :class="STATUS[statusOf(r)].tone">
              <UIcon :name="STATUS[statusOf(r)].icon" class="size-3.5" />{{ STATUS[statusOf(r)].label }}
            </span>
            <NuxtLink v-if="r.todo_id" :to="`/todos?project_id=${projectId}&todo=${r.todo_id}`" class="text-sky-700 dark:text-sky-400 hover:underline">
              Todo #{{ r.todo_id }}
            </NuxtLink>
            <span v-if="r.note" class="text-neutral-500">Catatan: {{ r.note }}</span>
          </div>
        </div>
        <div class="flex shrink-0 flex-wrap gap-1.5">
          <template v-if="statusOf(r) === 'open'">
            <UButton size="xs" color="neutral" variant="solid" icon="i-lucide-list-plus" :loading="busy === r.title" @click="makeTodo(r)">
              Jadikan todo
            </UButton>
            <UButton size="xs" color="neutral" variant="outline" :disabled="busy === r.title" @click="setStatus(r, 'handled')">
              Sudah ditangani
            </UButton>
            <UButton size="xs" color="neutral" variant="ghost" :disabled="busy === r.title" @click="setStatus(r, 'accepted')">
              Terima risiko
            </UButton>
          </template>
          <UButton v-else size="xs" color="neutral" variant="ghost" icon="i-lucide-rotate-ccw" :loading="busy === r.title" @click="setStatus(r, 'open')">
            Buka lagi
          </UButton>
        </div>
      </li>
    </ul>
  </section>
</template>
