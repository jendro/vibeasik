<script setup lang="ts">
import type { RouteSimulation, Finding } from '~/types'

const props = defineProps<{
  open: boolean
  simulations?: RouteSimulation[]
  findings?: Finding[]
  lastCheckAt?: string | null
  score?: number | null
}>()

const emit = defineEmits<{
  (e: 'update:open', val: boolean): void
  (e: 'resolveAll'): void
}>()

const pendingDeepFindings = computed(() => {
  return (props.findings || []).filter(f =>
    f.status !== 'resolved'
    && f.status !== 'ignored'
  )
})
const pendingDeepCount = computed(() => pendingDeepFindings.value.length)

const searchQuery = ref('')
const selectedMethod = ref('ALL')
const selectedStatus = ref('ALL')
const selectedValidation = ref('ALL')

const methods = ['ALL', 'GET', 'POST', 'PUT', 'PATCH', 'DELETE']

const list = computed(() => props.simulations || [])

const stats = computed(() => {
  const total = list.value.length
  let fast = 0
  let slow = 0
  let missingValidation = 0
  let highRiskQuery = 0
  let totalLatency = 0

  for (const s of list.value) {
    totalLatency += s.simulated_latency_ms || 0
    if (s.latency_status === 'FAST') fast++
    if (s.latency_status === 'SLOW' || s.latency_status === 'CRITICAL') slow++
    if (!s.has_form_validation && (s.method === 'POST' || s.method === 'PUT' || s.method === 'PATCH')) {
      missingValidation++
    }
    if (s.query_vulnerability_risk === 'HIGH') highRiskQuery++
  }

  const avgLatency = total > 0 ? Math.round(totalLatency / total) : 0
  return {
    total,
    fast,
    slow,
    missingValidation,
    highRiskQuery,
    avgLatency
  }
})

const filteredList = computed(() => {
  return list.value.filter((s) => {
    if (selectedMethod.value !== 'ALL' && s.method.toUpperCase() !== selectedMethod.value) {
      return false
    }
    if (selectedStatus.value !== 'ALL' && s.latency_status.toUpperCase() !== selectedStatus.value) {
      return false
    }
    if (selectedValidation.value === 'VALIDATED' && !s.has_form_validation) {
      return false
    }
    if (selectedValidation.value === 'MISSING' && s.has_form_validation) {
      return false
    }
    if (searchQuery.value.trim() !== '') {
      const q = searchQuery.value.toLowerCase()
      const matchPath = s.path.toLowerCase().includes(q)
      const matchHandler = s.handler.toLowerCase().includes(q)
      const matchComp = s.component.toLowerCase().includes(q)
      if (!matchPath && !matchHandler && !matchComp) return false
    }
    return true
  })
})

const getMethodColor = (m: string) => {
  switch (m.toUpperCase()) {
    case 'GET':
      return 'bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 border-sky-200 dark:border-sky-800'
    case 'POST':
      return 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300 border-emerald-200 dark:border-emerald-800'
    case 'PUT':
    case 'PATCH':
      return 'bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300 border-amber-200 dark:border-amber-800'
    case 'DELETE':
      return 'bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-300 border-rose-200 dark:border-rose-800'
    default:
      return 'bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 border-neutral-200 dark:border-neutral-700'
  }
}

const getLatencyBadge = (status: string, ms: number) => {
  switch (status.toUpperCase()) {
    case 'FAST':
      return {
        class: 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300 border-emerald-200 dark:border-emerald-800',
        label: `${ms}ms · Cepat`,
        icon: 'i-lucide-zap'
      }
    case 'NORMAL':
      return {
        class: 'bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 border-sky-200 dark:border-sky-800',
        label: `${ms}ms · Normal`,
        icon: 'i-lucide-gauge'
      }
    case 'SLOW':
      return {
        class: 'bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300 border-amber-200 dark:border-amber-800',
        label: `${ms}ms · Lambat`,
        icon: 'i-lucide-clock'
      }
    case 'CRITICAL':
    default:
      return {
        class: 'bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-300 border-rose-200 dark:border-rose-800',
        label: `${ms}ms · Kritis`,
        icon: 'i-lucide-alert-triangle'
      }
  }
}

const onKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.open) {
    emit('update:open', false)
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeyDown)
})
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-end bg-black/60 backdrop-blur-xs transition-opacity duration-200"
    @click.self="emit('update:open', false)"
  >
    <div
      class="w-full max-w-4xl h-full bg-white dark:bg-[#04111f] border-l border-neutral-200 dark:border-white/[0.08] shadow-2xl flex flex-col duration-200 overflow-hidden"
    >
      <!-- Top Header -->
      <div class="p-5 border-b border-neutral-200 dark:border-white/[0.08] flex items-start justify-between gap-4 bg-white dark:bg-[#04111f] shrink-0">
        <div class="flex items-start gap-3 min-w-0">
          <span class="p-2.5 rounded-xl bg-primary-100 dark:bg-primary-950/60 text-primary-700 dark:text-primary-400 border border-primary-200 dark:border-primary-800 shrink-0 mt-0.5">
            <UIcon name="i-lucide-activity" class="w-5 h-5" />
          </span>
          <div class="space-y-1 min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <h2 class="text-base font-bold text-neutral-900 dark:text-white truncate">
                Simulasi Real-Time Router & Latensi Eksekusi
              </h2>
              <span class="px-2 py-0.5 rounded-full text-xs font-bold font-mono bg-primary-100 dark:bg-primary-950/60 text-primary-700 dark:text-primary-300 border border-primary-200 dark:border-primary-800">
                {{ stats.total }} Rute Diuji
              </span>
              <span
                v-if="pendingDeepCount > 0"
                class="px-2 py-0.5 rounded-full text-xs font-bold font-mono bg-rose-100 dark:bg-rose-950/60 text-rose-700 dark:text-rose-300 border border-rose-200 dark:border-rose-800"
              >
                {{ pendingDeepCount }} Masalah Kritis/Peringatan
              </span>
            </div>
            <p class="text-xs text-neutral-500 dark:text-neutral-400 leading-relaxed">
              Hasil simulasi latensi dinamis, integritas validasi form/payload (Zod / Validator), proteksi rate-limit, dan analisis risiko kueri database pada rute API.
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <AppActionButton
            v-if="pendingDeepCount > 0"
            kind="process"
            llm
            primary
            size="xs"
            :label="`Jadikan todo (${pendingDeepCount})`"
            desc="Susun solusi semua temuan Deep Check lalu jadikan todo, bertahap"
            @click="emit('resolveAll'); emit('update:open', false)"
          />

          <button
            type="button"
            class="p-1.5 rounded-lg text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 cursor-pointer shrink-0 transition-colors"
            title="Tutup Panel"
            @click="emit('update:open', false)"
          >
            <UIcon name="i-lucide-x" class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Quick Metrics Ribbon -->
      <div class="p-4 grid grid-cols-2 sm:grid-cols-4 gap-3 bg-neutral-50/70 dark:bg-neutral-900/60 border-b border-neutral-200 dark:border-white/[0.08] shrink-0">
        <div class="p-2.5 rounded-lg bg-white dark:bg-neutral-800/80 border border-neutral-200 dark:border-neutral-700/80">
          <div class="text-xs font-semibold text-neutral-500">
            Rata-rata Latensi
          </div>
          <div class="text-lg font-bold font-mono text-neutral-900 dark:text-white">
            ~{{ stats.avgLatency }} ms
          </div>
          <div class="text-xs text-emerald-600 dark:text-emerald-400 font-medium">
            {{ stats.fast }} rute cepat (&lt;40ms)
          </div>
        </div>
        <div class="p-2.5 rounded-lg bg-white dark:bg-neutral-800/80 border border-neutral-200 dark:border-neutral-700/80">
          <div class="text-xs font-semibold text-neutral-500">
            Latensi Lambat
          </div>
          <div class="text-lg font-bold font-mono text-amber-600 dark:text-amber-400">
            {{ stats.slow }}
          </div>
          <div class="text-xs text-neutral-500">
            Perlu caching / optimasi
          </div>
        </div>
        <div class="p-2.5 rounded-lg bg-white dark:bg-neutral-800/80 border border-neutral-200 dark:border-neutral-700/80">
          <div class="text-xs font-semibold text-neutral-500">
            Tanpa Validasi Form
          </div>
          <div class="text-lg font-bold font-mono text-rose-600 dark:text-rose-400">
            {{ stats.missingValidation }}
          </div>
          <div class="text-xs text-neutral-500">
            Mutasi POST/PUT/PATCH
          </div>
        </div>
        <div class="p-2.5 rounded-lg bg-white dark:bg-neutral-800/80 border border-neutral-200 dark:border-neutral-700/80">
          <div class="text-xs font-semibold text-neutral-500">
            Risiko Kueri Tinggi
          </div>
          <div class="text-lg font-bold font-mono text-rose-600 dark:text-rose-400">
            {{ stats.highRiskQuery }}
          </div>
          <div class="text-xs text-neutral-500">
            Celah unindexed / unpaginated
          </div>
        </div>
      </div>

      <!-- Filter & Search Controls -->
      <div class="p-3.5 border-b border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] flex flex-wrap items-center gap-2.5 shrink-0">
        <!-- Search Input -->
        <div class="relative flex-1 min-w-[200px]">
          <UIcon name="i-lucide-search" class="w-3.5 h-3.5 text-neutral-400 absolute left-2.5 top-1/2 -translate-y-1/2 pointer-events-none" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Cari endpoint (/api/...), handler, atau komponen..."
            class="w-full pl-8 pr-7 py-1 text-xs rounded-lg border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white placeholder-neutral-400 focus:outline-none focus:ring-1 focus:ring-primary-500"
          >
          <button
            v-if="searchQuery"
            type="button"
            class="absolute right-2 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200"
            @click="searchQuery = ''"
          >
            <UIcon name="i-lucide-x" class="w-3.5 h-3.5" />
          </button>
        </div>

        <!-- Method Filter -->
        <div class="flex items-center gap-1">
          <button
            v-for="m in methods"
            :key="m"
            type="button"
            class="px-2 py-0.5 rounded text-xs font-mono font-bold transition-all cursor-pointer"
            :class="selectedMethod === m
              ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900'
              : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="selectedMethod = m"
          >
            {{ m }}
          </button>
        </div>

        <!-- Latency Filter -->
        <select
          v-model="selectedStatus"
          class="text-xs px-2 py-1 rounded border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 focus:outline-none"
        >
          <option value="ALL">
            Semua Latensi
          </option>
          <option value="FAST">
            Cepat (&lt;40ms)
          </option>
          <option value="NORMAL">
            Normal (40-100ms)
          </option>
          <option value="SLOW">
            Lambat (100-300ms)
          </option>
          <option value="CRITICAL">
            Kritis (&gt;300ms)
          </option>
        </select>

        <!-- Validation Filter -->
        <select
          v-model="selectedValidation"
          class="text-xs px-2 py-1 rounded border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 focus:outline-none"
        >
          <option value="ALL">
            Semua Validasi
          </option>
          <option value="VALIDATED">
            Tervalidasi (Form OK)
          </option>
          <option value="MISSING">
            Tanpa Validasi Form
          </option>
        </select>
      </div>

      <!-- Simulation List Body -->
      <div class="flex-1 overflow-y-auto p-4 space-y-3">
        <!-- Empty State -->
        <div
          v-if="filteredList.length === 0"
          class="p-12 text-center rounded-xl border border-dashed border-neutral-200 dark:border-white/[0.08] space-y-2"
        >
          <UIcon name="i-lucide-search-x" class="w-8 h-8 text-neutral-400 mx-auto" />
          <h4 class="text-sm font-bold text-neutral-800 dark:text-neutral-200">
            Tidak ada simulasi rute yang cocok
          </h4>
          <p class="text-xs text-neutral-500 max-w-sm mx-auto">
            Ubah kata kunci pencarian atau sesuaikan filter metode HTTP dan keparahan latensi.
          </p>
        </div>

        <!-- Simulation Cards -->
        <div
          v-for="(sim, idx) in filteredList"
          :key="`${sim.method}_${sim.path}_${idx}`"
          class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-neutral-900/90 space-y-2.5 hover:border-neutral-300 dark:hover:border-neutral-700 transition-colors"
        >
          <!-- Header: Method + Path + Latency Pill -->
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span class="px-2 py-0.5 rounded font-mono font-bold text-xs border shrink-0" :class="getMethodColor(sim.method)">
                {{ sim.method }}
              </span>
              <span class="font-mono font-semibold text-xs text-neutral-900 dark:text-white truncate">
                {{ sim.path }}
              </span>
            </div>

            <!-- Latency Pill -->
            <div class="flex items-center gap-1.5 shrink-0">
              <span
                class="px-2 py-0.5 rounded text-xs font-mono font-bold border flex items-center gap-1"
                :class="getLatencyBadge(sim.latency_status, sim.simulated_latency_ms).class"
              >
                <UIcon :name="getLatencyBadge(sim.latency_status, sim.simulated_latency_ms).icon" class="w-3.5 h-3.5" />
                <span>{{ getLatencyBadge(sim.latency_status, sim.simulated_latency_ms).label }}</span>
              </span>
            </div>
          </div>

          <!-- Metadata Badges: Component, Handler, Validation, Rate Limit, Query Risk -->
          <div class="flex flex-wrap items-center gap-2 text-xs">
            <!-- Handler / Component -->
            <span class="text-neutral-500 dark:text-neutral-400 flex items-center gap-1">
              <UIcon name="i-lucide-code-2" class="w-3.5 h-3.5 text-neutral-400" />
              <code class="font-mono text-xs bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">{{ sim.component }}: {{ sim.handler }}</code>
            </span>

            <span class="text-neutral-300 dark:text-neutral-700">·</span>

            <!-- Form Validation Status -->
            <span
              v-if="sim.has_form_validation"
              class="inline-flex items-center gap-1 text-emerald-600 dark:text-emerald-400 font-medium"
            >
              <UIcon name="i-lucide-check-circle" class="w-3.5 h-3.5" />
              <span>Form Validated</span>
            </span>
            <span
              v-else-if="sim.method === 'POST' || sim.method === 'PUT' || sim.method === 'PATCH'"
              class="inline-flex items-center gap-1 text-rose-600 dark:text-rose-400 font-medium"
            >
              <UIcon name="i-lucide-alert-circle" class="w-3.5 h-3.5" />
              <span>Tanpa Validasi Skema (Form)</span>
            </span>
            <span
              v-else
              class="text-neutral-400 text-xs"
            >
              Read-Only Endpoint
            </span>

            <span class="text-neutral-300 dark:text-neutral-700">·</span>

            <!-- Rate Limiter -->
            <span
              v-if="sim.has_rate_limiter"
              class="inline-flex items-center gap-1 text-sky-600 dark:text-sky-400 font-medium"
            >
              <UIcon name="i-lucide-shield-check" class="w-3.5 h-3.5" />
              <span>Rate Limited</span>
            </span>
            <span
              v-else
              class="inline-flex items-center gap-1 text-neutral-500 dark:text-neutral-400"
            >
              <UIcon name="i-lucide-shield-off" class="w-3.5 h-3.5 text-neutral-400" />
              <span>No Rate Limit</span>
            </span>

            <span class="text-neutral-300 dark:text-neutral-700">·</span>

            <!-- Query Vulnerability Risk -->
            <span
              class="inline-flex items-center gap-1 font-medium"
              :class="sim.query_vulnerability_risk === 'HIGH' ? 'text-rose-600 dark:text-rose-400 font-bold' : sim.query_vulnerability_risk === 'MEDIUM' ? 'text-amber-600 dark:text-amber-400' : 'text-neutral-500'"
            >
              <UIcon name="i-lucide-database" class="w-3.5 h-3.5" />
              <span>Risiko Kueri: {{ sim.query_vulnerability_risk }}</span>
            </span>
          </div>

          <!-- Simulation Detail / Breakdown -->
          <div v-if="sim.details" class="p-2 rounded bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200/60 dark:border-neutral-700/60 text-xs text-neutral-600 dark:text-neutral-300 font-mono">
            {{ sim.details }}
          </div>
        </div>
      </div>

      <!-- Drawer Footer -->
      <div class="p-4 border-t border-neutral-200 dark:border-white/[0.08] bg-neutral-50/80 dark:bg-neutral-900/80 flex items-center justify-between gap-3 text-xs shrink-0 flex-wrap">
        <div class="flex items-center gap-2 min-w-0">
          <span class="text-neutral-500 dark:text-neutral-400">
            Disimulasikan menggunakan <strong class="text-neutral-700 dark:text-neutral-300">Vibeasik Deep Check Engine</strong>
          </span>
          <span v-if="pendingDeepCount > 0" class="text-neutral-300 dark:text-neutral-700">·</span>
          <span v-if="pendingDeepCount > 0" class="font-semibold text-rose-600 dark:text-rose-400 font-mono">
            {{ pendingDeepCount }} temuan belum diselesaikan
          </span>
        </div>

        <div class="flex items-center gap-2">
          <AppActionButton
            v-if="pendingDeepCount > 0"
            kind="process"
            llm
            primary
            size="xs"
            :label="`Jadikan todo (${pendingDeepCount})`"
            desc="Susun solusi semua temuan Deep Check lalu jadikan todo, bertahap"
            @click="emit('resolveAll'); emit('update:open', false)"
          />

          <UButton
            size="xs"
            color="neutral"
            variant="outline"
            @click="emit('update:open', false)"
          >
            Tutup
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>
