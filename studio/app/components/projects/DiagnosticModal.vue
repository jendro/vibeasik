<script setup lang="ts">
import type { Diagnostic, Project, Finding, ProjectComponent } from '~/types'

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  diagnostic: Diagnostic | null
  project?: Project | null
  componentDiagnostics?: Record<number, Diagnostic> | null
}>()

const selectedAppId = ref<number | 'all'>('all')

const getScoreColor = (score: number) => {
  if (score >= 90) return 'text-sky-600 dark:text-sky-400 border-sky-500/30 bg-sky-50 dark:bg-sky-950/30'
  if (score >= 70) return 'text-amber-600 dark:text-amber-400 border-amber-500/30 bg-amber-50 dark:bg-amber-950/30'
  return 'text-rose-600 dark:text-rose-400 border-rose-500/30 bg-rose-50 dark:bg-rose-950/30'
}

const getRoleBadgeColor = (role: string) => {
  switch (role) {
    case 'backend': return 'info'
    case 'frontend_admin': return 'secondary'
    case 'frontend_client': return 'success'
    case 'frontend_mobile': return 'warning'
    default: return 'neutral'
  }
}

const getRoleLabel = (role: string) => {
  switch (role) {
    case 'backend': return 'BE'
    case 'frontend_admin': return 'FE Admin'
    case 'frontend_client': return 'FE Web'
    case 'frontend_mobile': return 'Mobile'
    case 'docs': return 'Docs'
    case 'cli': return 'CLI'
    case 'worker': return 'Worker'
    default: return role === 'other' ? 'Lainnya' : role
  }
}

const allFindings = computed<Finding[]>(() => {
  if (!props.diagnostic?.findings_json) return []
  try {
    return JSON.parse(props.diagnostic.findings_json)
  } catch {
    return []
  }
})

const getComponentScore = (comp: ProjectComponent): number => {
  const diag = props.componentDiagnostics?.[comp.id]
  if (diag) {
    return diag.score
  }
  if (comp.health_score !== undefined && comp.health_score !== null) {
    return comp.health_score
  }
  return 100
}

const filteredFindings = computed<Finding[]>(() => {
  if (selectedAppId.value === 'all') {
    return allFindings.value
  }
  const comp = props.project?.components?.find(c => c.id === selectedAppId.value)
  if (!comp) return allFindings.value

  const rel = (comp.relative_path || comp.name).toLowerCase()
  return allFindings.value.filter((f) => {
    const target = (f.target_file || '').toLowerCase()
    const desc = (f.description || '').toLowerCase()
    const title = (f.title || '').toLowerCase()
    return (
      (target && (target.includes(rel) || target.includes(comp.name.toLowerCase())))
      || desc.includes(`'${comp.name.toLowerCase()}'`)
      || title.includes(comp.name.toLowerCase())
    )
  })
})

// Reset selectedAppId when modal opens
watch(open, (isOpen) => {
  if (isOpen) {
    selectedAppId.value = 'all'
  }
})
</script>

<template>
  <UModal
    v-model:open="open"
    title="Laporan Hasil Audit Kesehatan Kode"
    description="Hasil inspeksi kesehatan konfigurasi proyek dan seluruh aplikasi satu persatu."
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <div v-if="diagnostic" class="space-y-5">
        <!-- Overall Project Header -->
        <div class="flex items-center justify-between pb-3 border-b border-neutral-200 dark:border-neutral-800">
          <div>
            <div class="flex items-center gap-2">
              <h3 class="text-base font-bold text-neutral-900 dark:text-white">
                {{ project?.name || 'Proyek Lokal' }}
              </h3>
              <UBadge size="xs" color="primary" variant="subtle">
                Audit Selesai
              </UBadge>
            </div>
            <p class="text-xs text-neutral-500 font-mono mt-0.5">
              {{ diagnostic.summary }}
            </p>
          </div>
          <div
            class="px-3 py-1.5 rounded-lg text-sm font-mono font-bold border shrink-0 text-center"
            :class="getScoreColor(diagnostic.score)"
          >
            <div class="text-base leading-none">
              {{ diagnostic.score }}<span class="text-xs font-normal">/100</span>
            </div>
            <div class="text-[9px] uppercase tracking-wider font-semibold mt-0.5">
              Skor Proyek
            </div>
          </div>
        </div>

        <!-- Breakdown: Seluruh Aplikasi Diperiksa Satu Persatu -->
        <div v-if="project?.components && project.components.length > 0" class="space-y-2">
          <div class="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-neutral-500">
            <span>Aplikasi yang Diperiksa ({{ project.components.length }} Aplikasi Satu Persatu)</span>
            <span class="text-[11px] normal-case font-normal text-neutral-400">Klik aplikasi untuk memfilter temuan</span>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
            <!-- All Filter Card -->
            <button
              type="button"
              class="flex items-center justify-between p-2.5 rounded-lg border text-left transition-all cursor-pointer"
              :class="selectedAppId === 'all'
                ? 'border-sky-500 bg-sky-50/50 dark:bg-sky-950/30 text-sky-950 dark:text-sky-100'
                : 'border-neutral-200 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-800 hover:border-neutral-300 dark:hover:border-neutral-700'"
              @click="selectedAppId = 'all'"
            >
              <div class="flex items-center gap-2 min-w-0 truncate">
                <UIcon name="i-lucide-layers" class="w-4 h-4 text-sky-500 shrink-0" />
                <span class="text-xs font-medium truncate">Semua Temuan Proyek</span>
              </div>
              <UBadge size="xs" color="neutral" variant="subtle">
                {{ allFindings.length }} isu
              </UBadge>
            </button>

            <!-- Component Item Cards -->
            <button
              v-for="comp in project.components"
              :key="comp.id"
              type="button"
              class="flex items-center justify-between p-2.5 rounded-lg border text-left transition-all cursor-pointer font-mono"
              :class="selectedAppId === comp.id
                ? 'border-sky-500 bg-sky-50/50 dark:bg-sky-950/30 text-sky-950 dark:text-sky-100'
                : 'border-neutral-200 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-800 hover:border-neutral-300 dark:hover:border-neutral-700'"
              @click="selectedAppId = comp.id"
            >
              <div class="flex items-center gap-2 min-w-0 truncate">
                <UBadge size="xs" :color="getRoleBadgeColor(comp.role)" variant="subtle" class="shrink-0">
                  {{ getRoleLabel(comp.role) }}
                </UBadge>
                <div class="truncate">
                  <div class="text-xs font-semibold text-neutral-800 dark:text-neutral-200 truncate">
                    {{ comp.name }}
                  </div>
                  <div class="text-[10px] text-neutral-400 truncate">
                    {{ comp.language }} {{ comp.framework ? `(${comp.framework})` : '' }} {{ comp.port ? `:${comp.port}` : '' }}
                  </div>
                </div>
              </div>

              <div
                class="px-2 py-0.5 rounded text-[11px] font-bold border shrink-0"
                :class="getScoreColor(getComponentScore(comp))"
              >
                {{ getComponentScore(comp) }}/100
              </div>
            </button>
          </div>
        </div>

        <!-- Findings List -->
        <div class="space-y-2">
          <div class="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-neutral-500">
            <span>
              {{ selectedAppId === 'all' ? 'Seluruh Temuan & Peringatan' : `Temuan Khusus: ${project?.components?.find(c => c.id === selectedAppId)?.name}` }}
              ({{ filteredFindings.length }})
            </span>
            <span v-if="selectedAppId !== 'all'" class="text-[11px] font-normal normal-case text-sky-600 dark:text-sky-400 cursor-pointer" @click="selectedAppId = 'all'">
              Tampilkan Semua
            </span>
          </div>

          <div
            v-if="filteredFindings.length === 0"
            class="p-6 text-center rounded-xl border border-dashed border-neutral-200 dark:border-neutral-800 text-xs text-neutral-500"
          >
            <UIcon name="i-lucide-check-circle" class="w-6 h-6 mx-auto text-emerald-500 mb-1" />
            <div>Tidak ada masalah atau peringatan terdeteksi pada bagian ini. Kondisi kode optimal.</div>
          </div>

          <div
            v-for="(finding, idx) in filteredFindings"
            :key="idx"
            class="p-3 rounded-lg border text-xs space-y-1"
            :class="finding.severity === 'CRITICAL'
              ? 'border-rose-200 dark:border-rose-900/60 bg-rose-50/50 dark:bg-rose-950/20'
              : finding.severity === 'WARNING'
                ? 'border-amber-200 dark:border-amber-900/60 bg-amber-50/50 dark:bg-amber-950/20'
                : 'border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-800/40'"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="font-bold text-neutral-900 dark:text-neutral-100 flex items-center gap-1.5 truncate">
                <UIcon
                  :name="finding.severity === 'CRITICAL' ? 'i-lucide-alert-triangle' : finding.severity === 'WARNING' ? 'i-lucide-alert-circle' : 'i-lucide-info'"
                  class="w-3.5 h-3.5 shrink-0"
                  :class="finding.severity === 'CRITICAL' ? 'text-rose-500' : finding.severity === 'WARNING' ? 'text-amber-500' : 'text-blue-500'"
                />
                <span class="truncate">{{ finding.title }}</span>
              </span>
              <div class="flex items-center gap-1.5 shrink-0">
                <span v-if="finding.target_file" class="text-[10px] font-mono text-neutral-400 truncate max-w-[150px]">
                  {{ finding.target_file }}{{ finding.line ? `:${finding.line}` : '' }}
                </span>
                <UBadge size="xs" :color="finding.severity === 'CRITICAL' ? 'error' : finding.severity === 'WARNING' ? 'warning' : 'neutral'">
                  {{ finding.severity }}
                </UBadge>
              </div>
            </div>
            <p class="text-neutral-600 dark:text-neutral-400">
              {{ finding.description }}
            </p>
            <div class="text-sky-600 dark:text-sky-400 font-mono mt-1 text-[11px]">
              Solusi: {{ finding.suggestion }}
            </div>
          </div>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-between w-full">
        <UButton
          v-if="diagnostic"
          color="primary"
          variant="solid"
          size="sm"
          icon="i-lucide-sparkles"
          :to="'/audit-report?project_id=' + diagnostic.project_id"
          @click="open = false"
        >
          Buka Laporan Lengkap & Selesaikan via LLM
        </UButton>
        <div v-else />
        <UButton color="neutral" variant="outline" size="sm" @click="open = false">
          Tutup
        </UButton>
      </div>
    </template>
  </UModal>
</template>
