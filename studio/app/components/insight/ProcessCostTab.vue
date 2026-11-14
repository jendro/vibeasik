<script setup lang="ts">
import type { CostClass, CostSource, ProcessCostPage, Project } from '~/types'
import type { StatItem } from '~/components/AppStatStrip.vue'
import { COST_CLASSES, costHandlingNote, fmtTicks, shortFormula } from '~/utils/processCost'

// Tab "Biaya Proses": rumus tick T(n), Big-O, dan jumlah query tiap rute/fungsi backend Go & PHP (Laravel, CodeIgniter, Slim, PHP biasa), paging dari server.
const props = defineProps<{ project: Project }>()

const { getProcessCost, createProcessCostTodo } = useApi()
const { selectActiveProject } = useActiveProject()
const toast = useToast()
const PAGE = 50
const page = shallowRef<ProcessCostPage | null>(null)
const loading = ref(false)
const failed = ref('')
const kind = ref<'routes' | 'funcs'>('routes')
const search = ref('')
const klass = ref<CostClass | ''>('')
const onlyNPlusOne = ref(false)
const offset = ref(0)
let ctl: AbortController | null = null
let timer: ReturnType<typeof setTimeout> | null = null

const load = async () => {
  ctl?.abort() // request usang (filter/halaman sudah berganti) dibatalkan
  const mine = ctl = new AbortController()
  loading.value = true
  failed.value = ''
  try {
    const res = await getProcessCost(props.project.id, {
      kind: kind.value, q: search.value.trim(), class: klass.value, n_plus_one: onlyNPlusOne.value ? '1' : '', limit: PAGE, offset: offset.value
    }, mine.signal)
    if (ctl === mine) page.value = res // data lama dipertahankan sampai jawaban baru tiba
  } catch (err: any) {
    if (mine.signal.aborted) return
    failed.value = err?.data?.error || err?.message || 'Gagal menghitung biaya proses'
  } finally {
    if (ctl === mine) loading.value = false
  }
}

const reload = () => { offset.value = 0; load() }

// Cek ulang: endpoint selalu menghitung dari kode di disk, jadi memuat ulang = menghitung ulang Big-O. Hasilnya
// dibandingkan dengan hitungan sebelumnya supaya terlihat apakah perubahan kode memperbaiki atau memperburuk.
const rechecking = ref(false)
const heavyCount = (s?: ProcessCostPage['summary']) => s ? heavy(s.route_classes) : 0
const recheck = async () => {
  const before = page.value?.summary
  rechecking.value = true
  await load()
  rechecking.value = false
  const after = page.value?.summary
  if (failed.value || !after) {
    toast.add({ title: 'Gagal menghitung ulang Big-O', description: failed.value, color: 'error' })
    return
  }
  const changes: string[] = []
  const diff = (label: string, a: number, b: number) => { if (a !== b) changes.push(`${label} ${a} → ${b}`) }
  if (before) {
    diff('Rute', before.routes, after.routes)
    diff('Rute O(n²) ke atas', heavyCount(before), heavyCount(after))
    diff('Rute N+1', before.n_plus_one_routes, after.n_plus_one_routes)
    diff('Fungsi', before.funcs, after.funcs)
  }
  toast.add({
    title: 'Big-O dihitung ulang dari kode terbaru',
    description: changes.length ? changes.join(' · ') : 'Tidak ada perubahan dibanding hitungan sebelumnya.',
    color: changes.length ? 'success' : 'neutral',
    icon: 'i-lucide-sigma'
  })
}
const computedLabel = computed(() => {
  const at = page.value?.computed_at
  if (!at) return ''
  const time = new Date(at).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  const ms = page.value?.duration_ms || 0
  return `Dihitung dari kode di disk pukul ${time}${ms ? ` (${ms >= 1000 ? `${(ms / 1000).toFixed(1)} dtk` : `${ms} md`})` : ''}`
})
watch([kind, klass, onlyNPlusOne], reload)
watch(search, () => {
  if (timer) clearTimeout(timer)
  timer = setTimeout(reload, 300)
})
watch(() => props.project.id, () => { page.value = null; reload() })
onMounted(load)
onBeforeUnmount(() => { ctl?.abort(); if (timer) clearTimeout(timer) })

const summary = computed(() => page.value?.summary)
const classCounts = computed(() => (kind.value === 'routes' ? summary.value?.route_classes : summary.value?.func_classes) || {})
const classTotal = computed(() => Object.values(classCounts.value).reduce((n, v) => n + (v || 0), 0))
const distribution = computed(() => COST_CLASSES
  .map(c => ({ ...c, n: classCounts.value[c.key] || 0 }))
  .filter(c => c.n > 0)
  .map(c => ({ ...c, pct: classTotal.value ? c.n / classTotal.value * 100 : 0 })))

const heavy = (c: Partial<Record<CostClass, number>> = {}) => (c.quadratic || 0) + (c.cubic || 0) + (c.polynomial || 0)
const stats = computed<StatItem[]>(() => {
  const s = summary.value
  if (!s) return []
  const light = (s.route_classes.constant || 0) + (s.route_classes.log || 0) + (s.route_classes.linear || 0) + (s.route_classes.linearithmic || 0)
  return [
    { label: 'Rute dihitung', value: s.routes, hint: [`${s.funcs} fungsi lokal`, ...(s.frameworks || [])].join(' · ') },
    { label: 'Rute ringan', value: light, hint: 'O(1) sampai O(n log n)', dot: 'bg-emerald-500' },
    { label: 'Rute O(n²) ke atas', value: heavy(s.route_classes), hint: 'loop bersarang atas data', dot: 'bg-amber-500', tone: heavy(s.route_classes) ? 'text-amber-600 dark:text-amber-400' : undefined },
    { label: 'Rute N+1 query', value: s.n_plus_one_routes, hint: 'query di dalam loop', dot: 'bg-rose-500', tone: s.n_plus_one_routes ? 'text-rose-600 dark:text-rose-400' : undefined },
    { label: 'Total T(1) semua rute', value: fmtTicks(s.route_ticks_1), hint: 'tick untuk 1 data' }
  ]
})

const history = computed(() => page.value?.history || [])
const historyLabels = computed(() => recapDayLabels(history.value.map(h => h.snap_date)))
const heavySeries = computed(() => history.value.map(h => h.quadratic + h.worse + h.n_plus_one))

// Penanganan biaya tinggi: sumber biaya otomatis ikut Audit; yang belum ditangani bisa langsung dijadikan todo.
const handling = computed(() => page.value?.handling)
const note = computed(() => handling.value && handling.value.sources ? costHandlingNote(handling.value) : null)
const busySource = ref('')
const openTodos = () => { selectActiveProject(props.project.id); navigateTo('/todos') }
const openAudit = () => navigateTo(`/audit-report?project_id=${props.project.id}`)
const makeTodo = async (src: CostSource) => {
  busySource.value = `${src.kind}|${src.file}|${src.line}`
  try {
    const res = await createProcessCostTodo(props.project.id, src)
    toast.add({
      title: res.already_exists ? `Dilebur ke todo #${res.todo.id}` : `Todo #${res.todo.id} dibuat`,
      description: res.already_exists ? `Berkas ini sudah punya todo; sumber biayanya digabung ke sana: ${res.todo.title}` : res.todo.title,
      color: res.already_exists ? 'neutral' : 'success',
      icon: 'i-lucide-list-checks',
      actions: [{ label: 'Buka Todo', onClick: openTodos }]
    })
    await load()
  } catch (err: any) {
    toast.add({ title: 'Gagal membuat todo', description: err?.data?.error || err?.message || '', color: 'error' })
  } finally {
    busySource.value = ''
  }
}

const total = computed(() => page.value?.total || 0)
const from = computed(() => total.value ? offset.value + 1 : 0)
const to = computed(() => Math.min(total.value, offset.value + PAGE))
const go = (delta: number) => { offset.value = Math.max(0, offset.value + delta * PAGE); load() }
const card = 'min-w-0 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] p-5'
</script>

<template>
  <div class="space-y-4">
    <div v-if="loading && !page" class="space-y-4">
      <div class="h-24 rounded-2xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] animate-pulse" />
      <div class="h-72 rounded-2xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] animate-pulse" />
    </div>

    <div v-else-if="failed && !page" class="rounded-2xl border border-rose-300 dark:border-rose-800 bg-rose-50 dark:bg-rose-950/30 p-5 text-sm text-rose-800 dark:text-rose-200 flex items-center justify-between gap-3">
      <span>{{ failed }}</span>
      <UButton size="xs" color="neutral" variant="outline" @click="load">
        Coba lagi
      </UButton>
    </div>

    <template v-else-if="page && summary">
      <div v-if="!summary.supported" class="rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-10 text-center space-y-2">
        <UIcon name="i-lucide-sigma" class="size-8 text-neutral-400 mx-auto" />
        <p class="text-sm font-medium text-neutral-800 dark:text-neutral-100">
          Belum ada rute backend Go atau PHP yang bisa dihitung
        </p>
        <p class="text-sm text-neutral-500 max-w-md mx-auto">
          Penghitung tick membaca rute Fiber, Gin, Echo, Chi, dan net/http dari kode Go, serta rute Laravel/Lumen, CodeIgniter 4, Slim, dan halaman PHP biasa dari kode PHP. Proyek ini belum punya backend ber-rute di folder yang dipantau.
        </p>
      </div>

      <template v-else>
        <div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
          <p class="flex items-center gap-1.5 text-xs text-neutral-500">
            <UIcon name="i-lucide-clock" class="size-3.5" />
            <span>{{ computedLabel }}. Setiap rumus punya bukti: klik baris untuk melihat potongan kodenya.</span>
          </p>
          <AppActionButton
            kind="scan"
            label="Cek ulang Big-O"
            desc="Hitung ulang rumus tick dan Big-O semua rute dari kode terbaru di disk"
            :loading="rechecking"
            :disabled="loading"
            @click="recheck"
          />
        </div>

        <AppStatStrip :items="stats" />

        <div class="grid grid-cols-1 gap-3" :class="history.length > 1 ? 'lg:grid-cols-3' : ''">
          <section :class="[card, history.length > 1 ? 'lg:col-span-2' : '']" class="space-y-3">
            <div class="flex items-baseline justify-between gap-2">
              <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">
                Sebaran kelas pertumbuhan {{ kind === 'routes' ? 'rute' : 'fungsi' }}
              </h3>
              <span class="text-xs text-neutral-500">klik kelas untuk menyaring</span>
            </div>
            <div class="flex h-3 w-full overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800" role="img" :aria-label="distribution.map(d => `${d.bigO} ${d.n}`).join(', ')">
              <div v-for="d in distribution" :key="d.key" :class="d.bar" :style="{ width: `${d.pct}%` }" class="h-full border-r-2 border-white dark:border-[#04111f] last:border-r-0" />
            </div>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="d in distribution"
                :key="d.key"
                type="button"
                class="inline-flex items-center gap-2 h-9 rounded-xl border px-3 text-sm transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
                :class="klass === d.key ? 'border-neutral-900 dark:border-white bg-neutral-50 dark:bg-white/[0.06]' : 'border-neutral-200 dark:border-white/10 hover:bg-neutral-50 dark:hover:bg-white/[0.04]'"
                :aria-pressed="klass === d.key"
                :title="d.meaning"
                @click="klass = klass === d.key ? '' : d.key"
              >
                <span class="size-2 rounded-full" :class="d.bar" />
                <span class="font-mono text-xs font-semibold" :class="d.text">{{ d.bigO }}</span>
                <span class="text-neutral-600 dark:text-neutral-300">{{ d.label }}</span>
                <span class="tabular-nums font-semibold text-neutral-900 dark:text-white">{{ d.n }}</span>
              </button>
            </div>
            <p v-if="summary.worst_route" class="text-sm text-neutral-600 dark:text-neutral-300">
              Paling mahal saat data tumbuh:
              <code class="font-mono text-xs text-neutral-900 dark:text-white">{{ summary.worst_route.method }} {{ summary.worst_route.path }}</code>
              — T(n) = <code class="font-mono text-xs text-neutral-900 dark:text-white" :title="summary.worst_route.formula">{{ shortFormula(summary.worst_route.formula) }}</code>
            </p>
          </section>

          <section v-if="history.length > 1" :class="card" class="space-y-2">
            <h3 class="text-sm font-semibold text-neutral-900 dark:text-white">
              Rute berat dari waktu ke waktu
            </h3>
            <p class="text-xs text-neutral-500">
              O(n²) ke atas + N+1 query. Turun = algoritma membaik.
            </p>
            <InsightMiniSeriesChart :values="heavySeries" :labels="historyLabels" mode="line" tone="amber" unit="rute" :height="120" label="Jumlah rute berat per hari" />
          </section>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <div class="inline-flex rounded-lg border border-neutral-200 dark:border-white/10 p-0.5" role="group" label="Jenis daftar">
            <button
              v-for="k in [{ id: 'routes', label: `Rute (${summary.routes})` }, { id: 'funcs', label: `Fungsi (${summary.funcs})` }]"
              :key="k.id"
              type="button"
              class="h-8 px-3 rounded-md text-sm transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
              :class="kind === k.id ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900 font-medium' : 'text-neutral-600 dark:text-slate-300 hover:bg-neutral-100 dark:hover:bg-white/[0.06]'"
              :aria-pressed="kind === k.id"
              @click="kind = k.id as 'routes' | 'funcs'"
            >
              {{ k.label }}
            </button>
          </div>
          <UInput v-model="search" icon="i-lucide-search" size="sm" :placeholder="kind === 'routes' ? 'Cari path atau handler…' : 'Cari fungsi atau berkas…'" class="w-64" label="Cari" />
          <UButton
            size="sm"
            :color="onlyNPlusOne ? 'error' : 'neutral'"
            :variant="onlyNPlusOne ? 'soft' : 'outline'"
            icon="i-lucide-triangle-alert"
            :aria-pressed="onlyNPlusOne"
            @click="onlyNPlusOne = !onlyNPlusOne"
          >
            Hanya N+1
          </UButton>
          <UButton v-if="klass || onlyNPlusOne || search" size="sm" color="neutral" variant="ghost" icon="i-lucide-x" @click="klass = ''; onlyNPlusOne = false; search = ''">
            Hapus saringan
          </UButton>
          <span class="ml-auto text-xs text-neutral-500 tabular-nums">{{ from }}–{{ to }} dari {{ total }}</span>
        </div>

        <div
          v-if="note"
          class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-xl border px-4 py-3 text-sm"
          :class="handling!.open || handling!.recheck ? 'border-amber-300/60 bg-amber-50 dark:border-amber-500/30 dark:bg-amber-950/20' : 'border-neutral-200 bg-neutral-50 dark:border-white/[0.08] dark:bg-white/[0.03]'"
          role="status"
        >
          <div class="flex items-start gap-2.5 min-w-0">
            <UIcon :name="handling!.open || handling!.recheck ? 'i-lucide-triangle-alert' : 'i-lucide-list-checks'" class="size-4 mt-0.5 shrink-0" :class="handling!.open || handling!.recheck ? 'text-amber-600 dark:text-amber-400' : 'text-neutral-500'" />
            <div class="min-w-0 space-y-0.5">
              <p class="text-neutral-900 dark:text-white">
                <span class="font-semibold">{{ note.title }}</span>{{ note.where ? `: ${note.where}` : '' }}.
              </p>
              <p class="text-xs text-neutral-600 dark:text-neutral-300">
                {{ note.next }} Sumber biaya (N+1 &amp; O(n²) ke atas) otomatis ikut Audit; todo dari sini dimulai dari fungsi sumbernya lalu naik ke rute, dan semua sumber biaya satu berkas dilebur menjadi satu todo.
              </p>
            </div>
          </div>
          <div class="flex gap-2 shrink-0">
            <UButton v-if="handling!.in_todo" size="sm" color="neutral" variant="soft" icon="i-lucide-list-checks" @click="openTodos">
              Buka Todo ({{ handling!.todo_count }})
            </UButton>
            <UButton v-if="handling!.in_audit || handling!.recheck" size="sm" color="neutral" variant="soft" icon="i-lucide-shield-check" @click="openAudit">
              Buka Audit
            </UButton>
          </div>
        </div>

        <div v-if="total === 0" class="rounded-2xl border border-dashed border-neutral-300 dark:border-neutral-700 p-8 text-center text-sm text-neutral-500">
          Tidak ada {{ kind === 'routes' ? 'rute' : 'fungsi' }} yang cocok dengan saringan ini.
        </div>
        <InsightProcessCostTable
          v-else
          :kind="page.kind"
          :routes="page.routes"
          :funcs="page.funcs"
          :busy="busySource"
          :class="loading ? 'opacity-60' : ''"
          @make-todo="makeTodo"
          @open-todo="openTodos"
        />

        <div v-if="total > PAGE" class="flex justify-end gap-2">
          <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-chevron-left" :disabled="offset === 0 || loading" @click="go(-1)">
            Sebelumnya
          </UButton>
          <UButton size="sm" color="neutral" variant="outline" trailing-icon="i-lucide-chevron-right" :disabled="to >= total || loading" @click="go(1)">
            Berikutnya
          </UButton>
        </div>

        <InsightProcessCostExplainer />
      </template>
    </template>
  </div>
</template>
