<script setup lang="ts">
import type { ProjectComponent, RoleBadgeColor, ProfileApp } from '~/types'
import LastScanPopover from './LastScanPopover.vue'

const props = defineProps<{
  comp: ProjectComponent
  portActive?: boolean
  scanning?: boolean
  creatingDir?: boolean
  projectId?: number
  appProfile?: ProfileApp
}>()

defineEmits<{
  openDetail: [comp: ProjectComponent]
  fastScan: [comp: ProjectComponent]
  openExistingScan: [comp: ProjectComponent]
  createDir: [comp: ProjectComponent]
  edit: [comp: ProjectComponent]
  delete: [id: number]
}>()

const toast = useToast()

const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    toast.add({
      title: 'Disalin ke Clipboard',
      description: 'Dev command berhasil disalin!',
      icon: 'i-lucide-clipboard-check',
      color: 'neutral'
    })
  } catch {
    // ignore
  }
}

const getRoleConfig = (role: string) => {
  switch (role) {
    case 'backend':
      return {
        label: 'BE',
        fullLabel: 'Backend API',
        icon: 'i-lucide-server',
        badgeColor: 'info' as RoleBadgeColor
      }
    case 'frontend_client':
      return {
        label: 'FE Web',
        fullLabel: 'Frontend Web',
        icon: 'i-lucide-globe',
        badgeColor: 'success' as RoleBadgeColor
      }
    case 'frontend_admin':
      return {
        label: 'FE Admin',
        fullLabel: 'Admin Dashboard',
        icon: 'i-lucide-layout-dashboard',
        badgeColor: 'secondary' as RoleBadgeColor
      }
    case 'frontend_mobile':
      return {
        label: 'Mobile',
        fullLabel: 'Mobile App',
        icon: 'i-lucide-smartphone',
        badgeColor: 'warning' as RoleBadgeColor
      }
    case 'docs':
      return {
        label: 'Docs',
        fullLabel: 'Dokumentasi',
        icon: 'i-lucide-book-open',
        badgeColor: 'neutral' as RoleBadgeColor
      }
    case 'cli':
      return {
        label: 'CLI',
        fullLabel: 'CLI Tool',
        icon: 'i-lucide-terminal',
        badgeColor: 'neutral' as RoleBadgeColor
      }
    default:
      return {
        label: role === 'other' ? 'Modul' : role,
        fullLabel: 'Modul Pendukung',
        icon: 'i-lucide-box',
        badgeColor: 'neutral' as RoleBadgeColor
      }
  }
}

const roleConfig = computed(() => getRoleConfig(props.comp.role))

const envKeys = computed<string[]>(() => {
  if (!props.comp.env_summary) return []
  try {
    return JSON.parse(props.comp.env_summary)
  } catch {
    return []
  }
})

const targetUrl = computed(() => {
  const pid = props.projectId || props.comp.project_id
  return pid ? `/projects/${pid}/apps/${props.comp.id}` : '#'
})
</script>

<template>
  <div
    class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 p-4.5 sm:p-5 shadow-xs flex flex-col justify-between space-y-4 hover:border-neutral-300 dark:hover:border-neutral-700 transition-colors"
  >
    <div class="space-y-3">
      <!-- Component Header: Judul (fokus utama) di kiri, Logo / Badge BE, FE, mobile di pojok kanan atas -->
      <div class="flex items-start justify-between gap-3">
        <!-- Judul & Port di bawah judul -->
        <div class="min-w-0 flex-1 space-y-1">
          <NuxtLink
            :to="targetUrl"
            class="text-left font-bold text-base sm:text-lg text-neutral-900 dark:text-white truncate leading-snug hover:text-sky-600 dark:hover:text-sky-400 transition-colors cursor-pointer block w-full"
            :title="comp.name"
            @click="$emit('openDetail', comp)"
          >
            {{ comp.name }}
          </NuxtLink>

          <!-- Port tepat di bawah judul -->
          <div v-if="comp.port > 0" class="flex items-center gap-2">
            <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-mono font-bold bg-neutral-100 dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 border border-neutral-200/60 dark:border-neutral-700/60">
              <span>:{{ comp.port }}</span>
              <span
                class="w-2 h-2 rounded-full"
                :class="portActive ? 'bg-sky-400 animate-pulse' : 'bg-neutral-400'"
                :title="portActive ? 'Port aktif & listening' : 'Port offline / closed'"
              />
            </span>
            <span
              class="text-[10px] font-mono font-medium uppercase tracking-wider"
              :class="portActive ? 'text-sky-600 dark:text-sky-400 font-semibold' : 'text-neutral-400'"
            >
              {{ portActive ? 'LISTENING' : 'OFFLINE' }}
            </span>
          </div>
          <div v-else class="text-[11px] text-neutral-400 font-mono">
            Tanpa port
          </div>
        </div>

        <!-- Logo / Badge BE, FE, Mobile di Pojok Kanan Atas -->
        <div class="shrink-0 flex items-center gap-1.5">
          <TechBadge
            v-if="comp.framework || comp.language"
            :tech="comp.framework || comp.language"
            size="xs"
            variant="subtle"
          />
          <UBadge
            size="sm"
            :color="roleConfig.badgeColor"
            variant="subtle"
            class="font-semibold flex items-center gap-1"
          >
            <UIcon :name="roleConfig.icon" class="w-3.5 h-3.5" />
            <span>{{ roleConfig.label }}</span>
          </UBadge>
        </div>
      </div>

      <!-- Path & Tech Stack -->
      <ProjectsDetailAppProfileBlurb v-if="appProfile" :app="appProfile" />

      <div class="space-y-2 text-xs">
        <div class="text-neutral-500 font-mono truncate flex items-center gap-1.5" :title="comp.absolute_path">
          <UIcon name="i-lucide-folder" class="w-3.5 h-3.5 text-neutral-400 shrink-0" />
          <span class="truncate">{{ comp.relative_path || '.' }}</span>
        </div>

        <div class="flex flex-wrap items-center gap-1.5">
          <TechBadge v-if="comp.language" :tech="comp.language" size="xs" />
          <TechBadge
            v-if="comp.framework && comp.framework.toLowerCase() !== comp.language?.toLowerCase()"
            :tech="comp.framework"
            size="xs"
          />
          <TechBadge
            v-if="comp.package_manager"
            :tech="comp.package_manager"
            size="xs"
            :custom-label="`pkg: ${comp.package_manager}`"
          />
        </div>
      </div>

      <!-- Dev Command Box -->
      <div v-if="comp.dev_command" class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-100 dark:border-neutral-800 text-xs font-mono space-y-1">
        <div class="flex items-center justify-between text-neutral-400 text-[10px] uppercase">
          <span>Dev Command</span>
          <button
            class="text-sky-600 dark:text-sky-400 hover:underline cursor-pointer"
            @click="copyToClipboard(comp.dev_command)"
          >
            copy
          </button>
        </div>
        <div class="text-neutral-800 dark:text-neutral-200 truncate select-all">
          {{ comp.dev_command }}
        </div>
      </div>

      <!-- Dev runner: jalankan komponen di port unik yang dialokasikan vibeasik -->
      <ProjectsDetailComponentRunPanel v-if="comp.role !== 'frontend_mobile'" :comp="comp" />

      <!-- Environment summary if any -->
      <div v-if="envKeys.length > 0" class="text-[11px] text-neutral-500 font-mono">
        <span class="text-neutral-400">Env ({{ comp.env_file }}):</span>
        <div class="flex flex-wrap gap-1 mt-1">
          <span
            v-for="k in envKeys.slice(0, 4)"
            :key="k"
            class="px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 text-[10px]"
          >
            {{ k }}
          </span>
          <span v-if="envKeys.length > 4" class="text-neutral-400 text-[10px]">
            +{{ envKeys.length - 4 }} more
          </span>
        </div>
      </div>
    </div>

    <!-- Component Actions Footer -->
    <div class="pt-3 border-t border-neutral-100 dark:border-neutral-800 flex items-center justify-between gap-2">
      <div class="flex items-center gap-1.5">
        <UButton
          :to="targetUrl"
          icon="i-lucide-folder-open"
          color="primary"
          variant="solid"
          size="xs"
          title="Buka detail aplikasi: blueprint, diagnostics, AI advisor, dan MCP"
          @click="$emit('openDetail', comp)"
        >
          Buka Aplikasi
        </UButton>
        <LastScanPopover
          :scan-data="comp.fast_scan_data"
          :component-name="comp.name"
          :component-role="comp.role"
          align="start"
          side="top"
        >
          <UButton
            icon="i-lucide-scan"
            color="neutral"
            variant="outline"
            size="xs"
            :loading="scanning"
            title="Pindai cepat rute dan lingkungan via LLM"
            @click="$emit('fastScan', comp)"
          >
            {{ comp.fast_scan_data ? 'Re-Scan' : 'Scan LLM' }}
          </UButton>
        </LastScanPopover>
      </div>

      <div class="flex items-center gap-1">
        <UButton
          icon="i-lucide-pencil"
          color="neutral"
          variant="ghost"
          size="xs"
          title="Edit konfigurasi aplikasi ini"
          @click="$emit('edit', comp)"
        />
        <UButton
          icon="i-lucide-trash-2"
          color="error"
          variant="ghost"
          size="xs"
          title="Hapus aplikasi ini dari proyek"
          @click="$emit('delete', comp.id)"
        />
      </div>
    </div>
  </div>
</template>
