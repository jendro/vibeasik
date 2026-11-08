<script setup lang="ts">
import type { Project, ProjectComponent, VisualCaptureResult, VisualAuditResponse, ProjectSnapshotFile } from '~/types'

const props = defineProps<{
  project: Project
  portStatuses: Record<string, boolean>
}>()

const toast = useToast()
const { captureProjectVisual, auditProjectVisual, getProjectVisualSnapshots } = useApi()

// Candidate frontend components with ports
const frontendComponents = computed<ProjectComponent[]>(() => {
  if (!props.project.components) return []
  return props.project.components.filter(c =>
    c.role.includes('frontend') || c.role.includes('client') || c.port > 0
  )
})

const selectedCompId = ref<number | null>(null)
const selectedRoute = ref('/')
const customPrompt = ref('')
const isCapturing = ref(false)
const isAuditing = ref(false)
const latestCapture = ref<VisualCaptureResult | null>(null)
const latestAudit = ref<VisualAuditResponse | null>(null)
const snapshots = ref<ProjectSnapshotFile[]>([])
const isLightboxOpen = ref(false)

const currentComponent = computed(() => {
  return props.project.components?.find(c => c.id === selectedCompId.value) || frontendComponents.value[0] || null
})

const isPortLive = computed(() => {
  const p = currentComponent.value?.port
  return p ? Boolean(props.portStatuses[String(p)]) : false
})

onMounted(async () => {
  if (frontendComponents.value.length > 0 && !selectedCompId.value) {
    selectedCompId.value = frontendComponents.value[0]?.id ?? null
  }
  loadSnapshots()
})

const loadSnapshots = async () => {
  try {
    snapshots.value = await getProjectVisualSnapshots(props.project.id)
  } catch {
    // ignore
  }
}

const handleCapture = async () => {
  isCapturing.value = true
  try {
    const res = await captureProjectVisual(props.project.id, {
      component_id: selectedCompId.value || undefined,
      port: currentComponent.value?.port || undefined,
      route: selectedRoute.value
    })
    latestCapture.value = res
    await loadSnapshots()
    toast.add({
      title: 'Snapshot Berhasil Diambil',
      description: `Tampilan halaman ${res.route} (${res.size_kb.toFixed(1)} KB) berhasil ditangkap.`,
      icon: 'i-lucide-camera',
      color: 'success'
    })
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Gagal mengambil screenshot'
    toast.add({
      title: 'Gagal Mengambil Snapshot',
      description: msg,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    isCapturing.value = false
  }
}

const handleAudit = async () => {
  isAuditing.value = true
  try {
    const res = await auditProjectVisual(props.project.id, {
      component_id: selectedCompId.value || undefined,
      port: currentComponent.value?.port || undefined,
      route: selectedRoute.value,
      prompt: customPrompt.value
    })
    latestAudit.value = res
    latestCapture.value = {
      project_id: res.project_id,
      component_id: res.component_id,
      component_name: res.component_name,
      target_url: res.target_url,
      port: currentComponent.value?.port || 3000,
      route: res.route,
      captured_at: res.audited_at,
      image_path: res.screenshot_path,
      width: 1440,
      height: 900,
      size_kb: 0
    }
    await loadSnapshots()
    toast.add({
      title: 'Audit AI Selesai',
      description: `Skor UI: ${res.visual_score}/100, Integritas Data: ${res.data_hygiene_score}/100.`,
      icon: 'i-lucide-sparkles',
      color: 'success'
    })
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Gagal menjalankan audit AI'
    toast.add({
      title: 'Gagal Audit AI',
      description: msg,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    isAuditing.value = false
  }
}

const quickRoutes = ['/', '/dashboard', '/login', '/admin', '/products', '/settings']
</script>

<template>
  <div class="space-y-6">
    <!-- Top Control Bar -->
    <div class="p-5 rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/60 shadow-xs space-y-4">
      <div class="flex flex-col md:flex-row md:items-center justify-between gap-3">
        <div>
          <h2 class="text-base font-bold text-slate-900 dark:text-slate-100 flex items-center gap-2">
            <UIcon name="i-lucide-camera" class="size-5 text-blue-500" />
            Live Visual & Multimodal AI Audit
          </h2>
          <p class="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            Inspeksi visual antarmuka dan integritas data proyek FE langsung dari port laptop.
          </p>
        </div>

        <!-- Port Indicator & Status -->
        <div v-if="currentComponent" class="flex items-center gap-2 shrink-0">
          <span
            class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-mono font-medium border"
            :class="isPortLive
              ? 'bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-400 border-emerald-200 dark:border-emerald-800'
              : 'bg-amber-50 dark:bg-amber-950/60 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800'"
          >
            <span class="size-2 rounded-full" :class="isPortLive ? 'bg-emerald-500 animate-pulse' : 'bg-amber-500'" />
            :{{ currentComponent.port || 3000 }} {{ isPortLive ? 'Aktif' : 'Belum Berjalan' }}
          </span>
        </div>
      </div>

      <!-- Controls: Component Selector, Route, Action Buttons -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 pt-2">
        <!-- Component Dropdown -->
        <div>
          <label class="block text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1">Target Komponen</label>
          <select
            v-model="selectedCompId"
            class="w-full text-xs rounded-lg border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-900 px-3 py-2 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            <option v-for="c in frontendComponents" :key="c.id" :value="c.id">
              {{ c.name }} (:{{ c.port }})
            </option>
          </select>
        </div>

        <!-- Target Route Input -->
        <div>
          <label class="block text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1">Rute Halaman</label>
          <UInput
            v-model="selectedRoute"
            placeholder="/dashboard"
            size="sm"
            class="w-full"
          />
        </div>

        <!-- Optional Prompt Focus -->
        <div>
          <label class="block text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1">Fokus Audit AI (Opsional)</label>
          <UInput
            v-model="customPrompt"
            placeholder="Cek apakah ada nilai NaN/undefined..."
            size="sm"
            class="w-full"
          />
        </div>

        <!-- Action Buttons -->
        <div class="flex items-end gap-2">
          <UButton
            size="sm"
            color="neutral"
            variant="outline"
            icon="i-lucide-camera"
            :loading="isCapturing"
            class="flex-1"
            @click="handleCapture"
          >
            Snapshot
          </UButton>
          <UButton
            size="sm"
            color="primary"
            variant="solid"
            icon="i-lucide-sparkles"
            :loading="isAuditing"
            class="flex-1"
            @click="handleAudit"
          >
            Audit AI
          </UButton>
        </div>
      </div>

      <!-- Quick Route Pills -->
      <div class="flex items-center gap-1.5 overflow-x-auto pt-1">
        <span class="text-xs text-slate-400 font-medium">Quick Route:</span>
        <button
          v-for="r in quickRoutes"
          :key="r"
          class="text-xs px-2 py-0.5 rounded font-mono transition-colors"
          :class="selectedRoute === r ? 'bg-blue-600 text-white font-bold' : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:bg-slate-200'"
          @click="selectedRoute = r"
        >
          {{ r }}
        </button>
      </div>
    </div>

    <!-- Live Captured Preview Screen Card -->
    <div v-if="latestCapture || latestAudit" class="space-y-4">
      <div class="rounded-2xl border border-slate-200 dark:border-slate-800 bg-slate-950 overflow-hidden shadow-lg">
        <!-- macOS Style Header Dots -->
        <div class="px-4 py-2.5 bg-slate-900 border-b border-slate-800 flex items-center justify-between text-xs text-slate-400">
          <div class="flex items-center gap-2">
            <div class="flex gap-1.5">
              <span class="size-2.5 rounded-full bg-red-500/80 inline-block" />
              <span class="size-2.5 rounded-full bg-yellow-500/80 inline-block" />
              <span class="size-2.5 rounded-full bg-green-500/80 inline-block" />
            </div>
            <span class="font-mono text-xs text-slate-300 ml-2">
              {{ latestCapture?.target_url || latestAudit?.target_url }}
            </span>
          </div>

          <div class="flex items-center gap-2">
            <UButton
              size="xs"
              color="neutral"
              variant="ghost"
              icon="i-lucide-maximize-2"
              @click="isLightboxOpen = true"
            >
              Layar Penuh
            </UButton>
          </div>
        </div>

        <!-- Preview Image Body -->
        <div class="relative max-h-120 overflow-y-auto flex items-start justify-center p-2 bg-slate-950">
          <img loading="lazy" decoding="async"
            :src="latestCapture?.image_path || latestAudit?.screenshot_path"
            alt="Target Project Live Screenshot"
            class="w-full h-auto rounded-lg shadow-sm"
          >
        </div>
      </div>

      <!-- AI Audit Result Card -->
      <ProjectsDetailVisualAuditResultCard
        v-if="latestAudit"
        :audit="latestAudit"
        :project-id="project.id"
      />
    </div>

    <!-- Empty State (No capture yet) -->
    <div
      v-else
      class="py-16 text-center flex flex-col items-center justify-center p-8 rounded-2xl border border-dashed border-slate-200 dark:border-slate-800 bg-white/50 dark:bg-slate-900/30"
    >
      <div class="size-14 rounded-2xl bg-blue-500/10 text-blue-500 flex items-center justify-center mb-3">
        <UIcon name="i-lucide-scan-eye" class="size-7" />
      </div>
      <h3 class="text-sm font-bold text-slate-900 dark:text-slate-100">
        Belum Ada Snapshot Aktif
      </h3>
      <p class="text-xs text-slate-500 mt-1 max-w-md">
        Pilih rute halaman target dan klik tombol <b>"Snapshot"</b> untuk melihat tampilan FE atau <b>"Audit AI"</b> untuk evaluasi otomatis kebenaran data dan kualitas desain.
      </p>
    </div>

    <!-- Lightbox Modal -->
    <UModal
      :open="isLightboxOpen"
      :ui="{ content: 'max-w-6xl w-full p-0 overflow-hidden' }"
      @update:open="isLightboxOpen = $event"
    >
      <template #content>
        <div class="p-3 bg-slate-950 flex flex-col">
          <div class="flex items-center justify-between pb-2 mb-2 border-b border-slate-800 text-white">
            <span class="text-xs font-mono">{{ latestCapture?.target_url || latestAudit?.target_url }}</span>
            <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-x" @click="isLightboxOpen = false" />
          </div>
          <div class="max-h-[80vh] overflow-auto flex items-center justify-center">
            <img
              :src="latestCapture?.image_path || latestAudit?.screenshot_path"
              alt="Full Resolution Screenshot"
              class="max-w-full h-auto rounded"
            >
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
