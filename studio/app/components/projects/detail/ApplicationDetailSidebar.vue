<script setup lang="ts">
import type { Project, ProjectComponent, Diagnostic } from '~/types'
import { resolveTech } from '~/composables/useTechLogo'

const props = defineProps<{
  comp: ProjectComponent
  project: Project
  portActive?: boolean
  componentDiagnostic: Diagnostic | null
  hasBlueprint?: boolean
}>()

defineEmits<{
  checkPort: []
  edit: []
  createDir: []
  delete: []
  openAddModal: []
  switchTab: [tab: string]
}>()

const { copyToClipboard } = useCopyToClipboard()

const getScoreColor = (score: number) => {
  if (score >= 90) return 'text-sky-600 dark:text-sky-400 border-sky-500/30 bg-sky-50 dark:bg-sky-950/30'
  if (score >= 70) return 'text-amber-600 dark:text-amber-400 border-amber-500/30 bg-amber-50 dark:bg-amber-950/30'
  return 'text-rose-600 dark:text-rose-400 border-rose-500/30 bg-rose-50 dark:bg-rose-950/30'
}

const otherComponents = computed(() => {
  return props.project.components?.filter(c => c.id !== props.comp.id) || []
})

const { getRoleBadgeColor: getCompRoleBadgeColor } = useFormatters()

const getCompRoleLabel = (role: string): string => {
  switch (role) {
    case 'backend': return 'Backend API'
    case 'frontend_admin': return 'Admin Dashboard'
    case 'frontend_client': return 'Frontend Web'
    case 'frontend_mobile': return 'Mobile App'
    case 'cli': return 'CLI Tool'
    case 'worker': return 'Worker'
    case 'docs': return 'Dokumentasi'
    default: return role || 'App'
  }
}

const techInfo = computed(() => {
  return resolveTech(props.comp.framework || props.comp.language)
})
</script>

<template>
  <aside class="w-full lg:w-80 xl:w-88 2xl:w-96 shrink-0 border-t lg:border-t-0 lg:border-l border-neutral-200 dark:border-white/10 bg-white dark:bg-[#020c18] lg:fixed lg:right-0 lg:top-16 lg:bottom-0 lg:overflow-y-auto flex flex-col divide-y divide-neutral-200 dark:divide-white/10 select-none z-20">
    <!-- SECTION 1: Status Aplikasi & Health -->
    <div class="relative overflow-hidden p-4 sm:p-5 space-y-3.5">
      <!-- Watermark Logo Framework sebagai Background -->
      <div
        v-if="techInfo.icon"
        class="absolute -right-3 -top-3 pointer-events-none select-none overflow-hidden opacity-[0.06] dark:opacity-[0.10] transition-opacity"
        aria-hidden="true"
      >
        <UIcon
          :name="techInfo.icon"
          class="w-36 h-36"
          :style="techInfo.color ? { color: techInfo.color } : undefined"
        />
      </div>

      <!-- App Header: Info Aplikasi & Framework dipojok kanan atas -->
      <div class="relative z-10 space-y-2">
        <div class="flex items-start justify-between gap-2.5">
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-1.5">
              <span
                class="w-2 h-2 rounded-full shrink-0"
                :class="portActive ? 'bg-sky-500 animate-pulse' : 'bg-neutral-300 dark:bg-neutral-600'"
              />
              <span class="text-[11px] font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">
                Status Aplikasi
              </span>
            </div>
            <!-- Nama Aplikasi dibuat besar & jelas -->
            <h2
              class="text-base font-bold text-neutral-900 dark:text-white truncate mt-1 leading-snug"
              :title="comp.name"
            >
              {{ comp.name }}
            </h2>
            <div class="flex items-center gap-1.5 text-[11px] text-neutral-500 dark:text-neutral-400 font-mono mt-0.5 truncate">
              <UBadge
                size="xs"
                :color="getCompRoleBadgeColor(comp.role)"
                variant="subtle"
                class="font-sans shrink-0 text-[10px]"
              >
                {{ getCompRoleLabel(comp.role) }}
              </UBadge>
              <span class="truncate" :title="comp.relative_path || comp.absolute_path">
                {{ comp.relative_path || comp.absolute_path }}
              </span>
            </div>
          </div>

          <!-- Framework yg digunakan di pojok kanan atas -->
          <div class="shrink-0 flex flex-col items-end gap-1">
            <TechBadge
              v-if="comp.framework || comp.language"
              :tech="comp.framework || comp.language"
              size="xs"
              variant="subtle"
            />
            <span class="text-[10px] font-mono text-neutral-400">ID #{{ comp.id }}</span>
          </div>
        </div>
      </div>

      <!-- Health Score / Status Box -->
      <div
        v-if="componentDiagnostic"
        class="relative z-10 flex items-center gap-3 p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/80"
      >
        <div
          class="px-3.5 py-2.5 rounded-xl text-center border font-mono shrink-0"
          :class="getScoreColor(componentDiagnostic.score)"
        >
          <div class="text-2xl font-bold leading-none">
            {{ componentDiagnostic.score }}<span class="text-xs font-normal">/100</span>
          </div>
          <div class="text-[9px] uppercase font-semibold mt-0.5">
            Code Health
          </div>
        </div>
        <div class="text-xs space-y-0.5 min-w-0">
          <div class="font-bold text-neutral-900 dark:text-white truncate">
            {{ componentDiagnostic.score >= 90 ? 'Konfigurasi Optimal' : componentDiagnostic.score >= 70 ? 'Perlu Optimasi' : 'Peringatan Kesehatan' }}
          </div>
          <p class="text-[11px] text-neutral-500 dark:text-neutral-400 line-clamp-2">
            {{ componentDiagnostic.summary || 'Audit diagnostik selesai' }}
          </p>
        </div>
      </div>

      <div
        v-else
        class="relative z-10 flex items-center gap-3 p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/80 transition-colors hover:bg-neutral-100 dark:hover:bg-neutral-800/70 cursor-pointer"
        :title="hasBlueprint ? 'Buka hasil lengkap Pindai LLM' : 'Pindai aplikasi ini dari tab Blueprint & Rute'"
        @click="$emit('switchTab', hasBlueprint ? 'fast_scan' : 'blueprint')"
      >
        <div
          class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0"
          :class="hasBlueprint ? 'bg-sky-100 dark:bg-sky-950/60 text-sky-600 dark:text-sky-400' : 'bg-neutral-200 dark:bg-neutral-700 text-neutral-400'"
        >
          <UIcon :name="hasBlueprint ? 'i-lucide-check-circle' : 'i-lucide-activity'" class="w-5 h-5" />
        </div>
        <div class="text-xs min-w-0 flex-1">
          <div class="font-bold text-neutral-800 dark:text-neutral-200 flex items-center justify-between">
            <span>{{ hasBlueprint ? 'Sudah Dipindai LLM' : 'Belum Dipindai LLM' }}</span>
            <span class="text-[10px] text-sky-600 dark:text-sky-400 font-semibold">{{ hasBlueprint ? 'Lihat hasil' : 'Pindai' }} &rarr;</span>
          </div>
          <p class="text-[11px] text-neutral-500 dark:text-neutral-400">
            {{ hasBlueprint ? 'Arsitektur, env, rute & cara menjalankan' : 'Audit kode tersedia di menu Audit & Diagnostik' }}
          </p>
        </div>
      </div>

      <!-- Quick Metrics inside Status -->
      <div class="relative z-10 grid grid-cols-2 gap-2 text-xs">
        <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/80">
          <div class="flex items-center justify-between text-[10px] text-neutral-400 uppercase font-medium">
            <span>Status Port</span>
            <button
              v-if="comp.port > 0"
              type="button"
              class="normal-case text-sky-600 dark:text-sky-400 hover:underline cursor-pointer flex items-center gap-0.5"
              title="Periksa apakah port listening di laptop"
              @click="$emit('checkPort')"
            >
              <UIcon name="i-lucide-refresh-cw" class="w-3 h-3" />cek
            </button>
          </div>
          <div class="font-bold font-mono text-sm mt-0.5 flex items-center gap-1.5" :class="portActive ? 'text-sky-600 dark:text-sky-400' : 'text-neutral-500'">
            <span>{{ comp.port > 0 ? ':' + comp.port : 'Tanpa port' }}</span>
            <span v-if="portActive" class="w-1.5 h-1.5 rounded-full bg-sky-400 animate-pulse" />
          </div>
          <div class="text-[9px] uppercase font-semibold font-mono" :class="portActive ? 'text-sky-600 dark:text-sky-400' : 'text-neutral-400'">
            {{ comp.port > 0 ? (portActive ? 'LISTENING' : 'OFFLINE') : '-' }}
          </div>
        </div>

        <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/80">
          <div class="text-[10px] text-neutral-400 uppercase font-medium">
            Stack
          </div>
          <div class="font-bold text-neutral-800 dark:text-neutral-200 text-xs mt-0.5 truncate capitalize">
            {{ comp.framework || comp.language || 'Auto' }}
          </div>
          <div class="text-[10px] text-neutral-500 dark:text-neutral-400 truncate">
            {{ comp.package_manager ? `pkg: ${comp.package_manager}` : comp.language }}
          </div>
        </div>
      </div>

      <!-- Dev Command snippet -->
      <div v-if="comp.dev_command" class="relative z-10 p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800/80 text-xs font-mono space-y-1">
        <div class="flex items-center justify-between text-neutral-400 text-[10px] uppercase">
          <span>Dev Command</span>
          <button
            type="button"
            class="text-sky-600 dark:text-sky-400 hover:underline cursor-pointer"
            @click="copyToClipboard(comp.dev_command, 'Dev command disalin!')"
          >
            copy
          </button>
        </div>
        <div class="text-neutral-800 dark:text-neutral-200 truncate select-all">
          {{ comp.dev_command }}
        </div>
      </div>
    </div>

    <!-- SECTION 2: Aplikasi Lain -->
    <div class="p-4 sm:p-5 space-y-3 pb-16 sm:pb-20 flex-1">
      <div class="flex items-center justify-between pb-0.5">
        <span class="text-xs font-semibold text-neutral-900 dark:text-white flex items-center gap-1.5">
          <UIcon name="i-lucide-layers" class="w-4 h-4 text-sky-400" />
          <span>Aplikasi Lain ({{ otherComponents.length }})</span>
        </span>
      </div>

      <div v-if="otherComponents.length === 0" class="p-3 text-center rounded-lg border border-dashed border-neutral-200 dark:border-neutral-800 text-xs text-neutral-400">
        Hanya 1 aplikasi dalam proyek ini.
      </div>

      <div v-else class="space-y-1.5 max-h-56 overflow-y-auto pr-0.5">
        <NuxtLink
          v-for="oc in otherComponents"
          :key="oc.id"
          :to="`/projects/${project.id}/apps/${oc.id}`"
          class="flex items-center justify-between p-2 rounded-lg border border-neutral-200 dark:border-neutral-700/60 bg-white dark:bg-neutral-900 hover:border-sky-400 dark:hover:border-sky-600 transition-colors text-xs"
        >
          <div class="min-w-0 pr-2">
            <div class="font-semibold text-neutral-900 dark:text-white truncate">{{ oc.name }}</div>
            <div class="text-[10px] text-neutral-400 font-mono flex items-center gap-1 mt-0.5">
              <span>{{ oc.framework || oc.language }}</span>
              <span v-if="oc.port > 0">:{{ oc.port }}</span>
            </div>
          </div>
          <UBadge size="xs" :color="getCompRoleBadgeColor(oc.role)" variant="subtle" class="shrink-0 text-[10px]">
            {{ oc.role === 'backend' ? 'BE' : oc.role === 'frontend_admin' ? 'Admin' : oc.role === 'frontend_client' ? 'FE' : oc.role === 'frontend_mobile' ? 'Mobile' : oc.role }}
          </UBadge>
        </NuxtLink>
      </div>

      <div class="pt-1">
        <UButton
          icon="i-lucide-plus"
          color="neutral"
          variant="subtle"
          size="xs"
          block
          title="Daftarkan aplikasi baru ke proyek ini"
          @click="$emit('openAddModal')"
        >
          + Tambah Aplikasi ke Proyek
        </UButton>
      </div>
    </div>
  </aside>
</template>
