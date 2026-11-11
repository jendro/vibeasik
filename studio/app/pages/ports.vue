<script setup lang="ts">
import type { Project } from '~/composables/useApi'

definePageMeta({ fluid: true })

const { getProjects, checkPorts } = useApi()
const { activeProjectId, activeProject } = useActiveProject()

const projects = ref<Project[]>([])
const loading = ref(true)
const checking = ref(false)
const portStatusMap = ref<Record<string, boolean>>({})
const filterActiveProjectOnly = ref(false)

interface PortEntry {
  port: number
  isListening: boolean
  usedBy: {
    projectId: number
    projectName: string
    componentName: string
    role: string
    language?: string
    framework?: string
  }[]
}

const loadData = async () => {
  loading.value = true
  try {
    projects.value = await getProjects()
    await refreshPortStatuses()
  } catch (err) {
    console.error('Failed to load projects:', err)
  } finally {
    loading.value = false
  }
}

const refreshPortStatuses = async () => {
  checking.value = true
  try {
    const allPorts = new Set<number>()
    projects.value.forEach((p) => {
      p.components?.forEach((c) => {
        if (c.port > 0) allPorts.add(c.port)
      })
    })

    const portArray = Array.from(allPorts)
    if (portArray.length > 0) {
      portStatusMap.value = await checkPorts(portArray)
    }
  } catch (err) {
    console.error('Failed to check ports:', err)
  } finally {
    checking.value = false
  }
}

onMounted(() => {
  loadData()
})

const portTable = computed<PortEntry[]>(() => {
  const map: Record<number, PortEntry> = {}

  projects.value.forEach((p) => {
    p.components?.forEach((c) => {
      if (c.port > 0) {
        if (!map[c.port]) {
          map[c.port] = {
            port: c.port,
            isListening: false,
            usedBy: []
          }
        }
        map[c.port]!.usedBy.push({
          projectId: p.id,
          projectName: p.name,
          componentName: c.name,
          role: c.role,
          language: c.language,
          framework: c.framework
        })
      }
    })
  })

  // Assign listening status
  Object.keys(map).forEach((pStr) => {
    const entry = map[Number(pStr)]
    if (entry) {
      entry.isListening = !!portStatusMap.value[pStr]
    }
  })

  return Object.values(map).sort((a, b) => a.port - b.port)
})

const displayPortTable = computed<PortEntry[]>(() => {
  if (filterActiveProjectOnly.value && activeProjectId.value) {
    return portTable.value.filter(entry => entry.usedBy.some(u => u.projectId === activeProjectId.value))
  }
  return portTable.value
})

// Periksa apakah port memiliki bentrok internal dalam satu proyek bundle yang sama
const hasInternalClash = (entry: PortEntry) => {
  const pIds = entry.usedBy.map(u => u.projectId)
  return new Set(pIds).size < pIds.length
}

// Klasifikasi status konflik port
const getPortRisk = (entry: PortEntry) => {
  if (entry.usedBy.length <= 1) {
    return {
      type: 'unique',
      label: 'Unique',
      color: 'neutral' as const,
      tooltip: 'Port dialokasikan eksklusif untuk 1 servis'
    }
  }
  if (hasInternalClash(entry)) {
    return {
      type: 'internal_clash',
      label: `Internal Clash (${entry.usedBy.length})`,
      color: 'error' as const,
      tooltip: 'Bentrok internal: komponen dalam 1 bundle proyek yang sama menggunakan port kembar'
    }
  }
  if (entry.isListening) {
    return {
      type: 'active_conflict',
      label: 'Active Conflict (Live)',
      color: 'warning' as const,
      tooltip: 'Port sedang aktif listening di laptop dan diklaim oleh beberapa aplikasi'
    }
  }
  return {
    type: 'cross_project',
    label: `Bentrok Antar Proyek (${entry.usedBy.length})`,
    color: 'warning' as const,
    tooltip: 'Port sama dipakai proyek lain: tidak bisa dijalankan serentak. Engine memindahkannya ke blok port proyek (PORTS.md) saat audit / engine start'
  }
}

// Setiap port yang dipakai lebih dari satu komponen adalah bentrok (Aturan 35/45: blok port per proyek)
const conflictPorts = computed(() => displayPortTable.value.filter(entry => entry.usedBy.length > 1))
</script>

<template>
  <div class="w-full max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 pt-6 space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b border-neutral-200 dark:border-white/10">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-network" class="w-6 h-6 text-sky-500" />
          Port Conflict & Service Monitor
        </h1>
        <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1">
          Detects local port collisions across polyglot project bundles and monitors listening TCP socket states.
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <UButton
          v-if="activeProjectId"
          size="sm"
          :variant="filterActiveProjectOnly ? 'solid' : 'outline'"
          :color="filterActiveProjectOnly ? 'primary' : 'neutral'"
          icon="i-lucide-filter"
          title="Filter daftar port khusus untuk proyek aktif"
          @click="filterActiveProjectOnly = !filterActiveProjectOnly"
        >
          {{ filterActiveProjectOnly ? `Port: ${activeProject?.name || 'Proyek Aktif'}` : 'Filter Proyek Aktif' }}
        </UButton>

        <UButton
          icon="i-lucide-refresh-cw"
          color="neutral"
          variant="outline"
          size="sm"
          :loading="checking"
          title="Periksa ulang status TCP port aktif secara langsung di laptop"
          @click="refreshPortStatuses"
        >
          Segarkan Status Port Live
        </UButton>
      </div>
    </div>

    <!-- Blok port per proyek dikelola engine (Aturan 35/45) -->
    <div class="p-4 rounded-xl border border-violet-200 dark:border-violet-900/50 bg-violet-50/50 dark:bg-violet-950/20 text-xs space-y-1.5 text-violet-900 dark:text-violet-300">
      <div class="flex items-center gap-2 font-bold">
        <UIcon name="i-lucide-columns-3" class="w-4 h-4 text-violet-600 dark:text-violet-400 shrink-0" />
        <span>Blok 10 port per proyek — semua proyek bisa dijalankan serentak</span>
      </div>
      <p>
        Slot <span class="font-mono">N</span> = <span class="font-mono">4100 + N×10 … +9</span>; offset
        <span class="font-mono">+0..+4</span> frontend, <span class="font-mono">+5..+9</span> backend/service.
        Port <span class="font-mono">87xx</span> tetap milik vibeasik. Engine memberi slot otomatis saat proyek didaftarkan,
        menyetel <span class="font-mono">run_port</span> tiap komponen, menyelaraskan <span class="font-mono">PORT</span> &amp; URL API di
        <span class="font-mono">.env</span>, dan menulis <span class="font-mono">PORTS.md</span>.
      </p>
    </div>

    <!-- Conflict Warning Banner (Only for true clashes: internal or active socket) -->
    <div
      v-if="conflictPorts.length > 0"
      class="p-4 rounded-xl border border-rose-300 dark:border-rose-900/60 bg-rose-50 dark:bg-rose-950/20 text-xs space-y-2"
    >
      <div class="flex items-center gap-2 font-bold text-rose-800 dark:text-rose-300">
        <UIcon name="i-lucide-alert-triangle" class="w-4 h-4 text-rose-600 dark:text-rose-400" />
        <span>{{ conflictPorts.length }} Bentrok Port Perlu Tindakan</span>
      </div>
      <p class="text-rose-700 dark:text-rose-400">
        Port sama dipakai lebih dari satu komponen (di dalam proyek atau antar proyek). Jalankan audit proyek atau restart engine agar dipindah ke blok port proyeknya.
      </p>
      <div class="flex flex-wrap gap-2 pt-1 font-mono">
        <span
          v-for="cp in conflictPorts"
          :key="cp.port"
          class="px-2 py-0.5 rounded bg-rose-200/60 dark:bg-rose-900/60 text-rose-900 dark:text-rose-200 font-bold"
        >
          :{{ cp.port }} ({{ cp.usedBy.length }} servis)
        </span>
      </div>
    </div>

    <!-- Registry port unik yang dialokasikan vibeasik untuk tombol Run -->
    <PortsPortRegistryTable />

    <!-- Port Matrix Table (following good-table-design) -->
    <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs overflow-hidden">
      <div class="px-5 py-4 border-b border-neutral-100 dark:border-white/10 flex items-center justify-between">
        <span class="text-xs font-semibold uppercase tracking-wider text-neutral-500">
          Local Port Mapping Matrix ({{ displayPortTable.length }} Assigned Ports)
        </span>
        <span class="text-xs font-mono text-neutral-400">127.0.0.1 TCP</span>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead>
            <tr class="border-b border-neutral-200 dark:border-white/10 bg-neutral-50/70 dark:bg-slate-800/40 text-neutral-500">
              <th class="py-3 px-4 font-semibold">
                Port
              </th>
              <th class="py-3 px-4 font-semibold">
                Live State
              </th>
              <th class="py-3 px-4 font-semibold">
                Conflict Risk
              </th>
              <th class="py-3 px-4 font-semibold">
                Assigned Services
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800 font-mono">
            <tr
              v-for="entry in displayPortTable"
              :key="entry.port"
              class="hover:bg-neutral-50/50 dark:hover:bg-slate-800/30 transition-colors"
            >
              <td class="py-3.5 px-4 font-bold text-sm text-neutral-900 dark:text-white">
                :{{ entry.port }}
              </td>

              <td class="py-3.5 px-4">
                <div class="flex items-center gap-1.5 font-sans text-xs">
                  <span
                    class="w-2 h-2 rounded-full"
                    :class="entry.isListening ? 'bg-sky-500 animate-pulse' : 'bg-neutral-400'"
                  />
                  <span :class="entry.isListening ? 'text-sky-600 dark:text-sky-400 font-bold' : 'text-neutral-400'">
                    {{ entry.isListening ? 'ACTIVE / LISTENING' : 'FREE' }}
                  </span>
                </div>
              </td>

              <td class="py-3.5 px-4 font-sans">
                <UBadge
                  :color="getPortRisk(entry).color"
                  variant="subtle"
                  size="xs"
                  :title="getPortRisk(entry).tooltip"
                >
                  {{ getPortRisk(entry).label }}
                </UBadge>
              </td>

              <td class="py-3.5 px-4">
                <div class="space-y-1 font-sans">
                  <div
                    v-for="(svc, idx) in entry.usedBy"
                    :key="idx"
                    class="flex flex-wrap items-center gap-1.5 text-xs"
                  >
                    <NuxtLink
                      :to="`/projects/${svc.projectId}`"
                      class="text-neutral-700 dark:text-neutral-300 font-medium hover:text-sky-600 dark:hover:text-sky-400"
                    >
                      {{ svc.projectName }}
                    </NuxtLink>
                    <span class="text-neutral-400 font-mono text-xs">→ {{ svc.componentName }}</span>
                    <TechBadge v-if="svc.language" :tech="svc.language" size="xs" />
                    <TechBadge
                      v-if="svc.framework && svc.framework.toLowerCase() !== svc.language?.toLowerCase()"
                      :tech="svc.framework"
                      size="xs"
                    />
                  </div>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
