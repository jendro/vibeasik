<script setup lang="ts">
import type { Project, ProjectComponent, RoleBadgeColor } from '~/types'

const activeTab = defineModel<string>('activeTab', { default: 'blueprint' })

const props = defineProps<{
  comp: ProjectComponent
  project: Project
  portActive?: boolean
  routesCount?: number
  findingsCount?: number
  hasFastScan?: boolean
}>()

defineEmits<{
  edit: []
  delete: []
}>()

const toast = useToast()

const copyToClipboard = async (text: string, label = 'Teks berhasil disalin!') => {
  try {
    await navigator.clipboard.writeText(text)
    toast.add({
      title: 'Disalin ke Clipboard',
      description: label,
      icon: 'i-lucide-clipboard-check',
      color: 'neutral'
    })
  } catch {
    // ignore
  }
}

const getCompRoleBadgeColor = (role: string): RoleBadgeColor => {
  switch (role) {
    case 'backend': return 'info'
    case 'frontend_admin': return 'secondary'
    case 'frontend_client': return 'success'
    case 'frontend_mobile': return 'warning'
    default: return 'neutral'
  }
}

const getCompRoleLabel = (role: string): string => {
  switch (role) {
    case 'backend': return 'Backend API'
    case 'frontend_admin': return 'Admin Dashboard'
    case 'frontend_client': return 'Frontend Web'
    case 'frontend_mobile': return 'Mobile App'
    case 'cli': return 'CLI Tool'
    case 'worker': return 'Worker Service'
    case 'docs': return 'Dokumentasi'
    default: return role
  }
}

interface AppTabItem {
  id: string
  label: string
  icon: string
  badge?: number | string
  badgeColor?: 'neutral' | 'warning' | 'success' | 'info' | 'primary'
}

const isFrontend = computed(() => {
  const r = props.comp?.role || ''
  return r.includes('frontend') || r === 'frontend_client' || r === 'frontend_admin' || r === 'frontend_mobile'
})

const tabs = computed<AppTabItem[]>(() => [
  {
    id: 'blueprint',
    label: isFrontend.value ? 'Daftar Halaman' : 'Blueprint & Rute',
    icon: isFrontend.value ? 'i-lucide-panels-top-left' : 'i-lucide-layout-grid',
    badge: props.routesCount,
    badgeColor: 'neutral'
  },
  {
    id: 'fast_scan',
    label: 'Hasil Pindai LLM',
    icon: 'i-lucide-scan-text',
    badge: props.hasFastScan ? 'Tersedia' : undefined,
    badgeColor: 'success'
  },
  {
    id: 'diagnostics',
    label: 'Codebase Diagnostics',
    icon: 'i-lucide-shield-check',
    badge: props.findingsCount,
    badgeColor: 'warning'
  },
  {
    id: 'ai',
    label: 'AI Advisor & Optimasi',
    icon: 'i-lucide-sparkles'
  },
  {
    id: 'mcp',
    label: 'Agent MCP Hub',
    icon: 'i-lucide-cpu'
  }
])
</script>

<template>
  <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 p-4 sm:p-5 shadow-xs space-y-3.5">
    <!-- Top Action Row: Back to Project (left) & Actions (right) -->
    <div class="flex items-center justify-between text-xs pb-2 border-b border-neutral-100 dark:border-neutral-800/80">
      <NuxtLink
        :to="`/projects/${project.id}`"
        class="inline-flex items-center gap-1.5 text-neutral-500 hover:text-neutral-900 dark:hover:text-white transition-colors font-medium"
      >
        <UIcon name="i-lucide-arrow-left" class="w-3.5 h-3.5" />
        <span>Kembali ke Proyek: {{ project.name }}</span>
      </NuxtLink>

      <div class="flex items-center gap-1.5">
        <span class="font-mono text-[11px] text-neutral-400 mr-1">App ID: #{{ comp.id }}</span>
        <UButton
          icon="i-lucide-pencil"
          color="neutral"
          variant="outline"
          size="xs"
          title="Edit konfigurasi aplikasi ini"
          @click="$emit('edit')"
        >
          Edit
        </UButton>
        <UButton
          icon="i-lucide-trash-2"
          color="error"
          variant="ghost"
          size="xs"
          title="Hapus aplikasi ini dari proyek"
          @click="$emit('delete')"
        >
          Hapus
        </UButton>
      </div>
    </div>

    <!-- Application Info & Badges -->
    <div class="space-y-2.5">
      <div class="flex flex-wrap items-center gap-2 sm:gap-2.5">
        <UBadge size="sm" :color="getCompRoleBadgeColor(comp.role)" variant="subtle" class="font-bold">
          {{ getCompRoleLabel(comp.role) }}
        </UBadge>

        <h1 class="text-xl sm:text-2xl font-bold tracking-tight text-neutral-900 dark:text-white">
          {{ comp.name }}
        </h1>

        <NuxtLink
          :to="`/projects/${project.id}`"
          class="text-xs text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-200 bg-neutral-100 dark:bg-neutral-800 px-2 py-0.5 rounded font-mono transition-colors"
          title="Buka ringkasan proyek induk"
        >
          Proyek: {{ project.name }}
        </NuxtLink>

        <!-- Listening Port Status Badge -->
        <div v-if="comp.port > 0" class="flex items-center gap-1.5 ml-auto sm:ml-0 bg-neutral-100 dark:bg-neutral-800/80 px-2.5 py-0.5 rounded-full border border-neutral-200 dark:border-neutral-700/60 font-mono text-xs">
          <span class="font-bold text-neutral-800 dark:text-neutral-200">:{{ comp.port }}</span>
          <span
            class="w-2 h-2 rounded-full"
            :class="portActive ? 'bg-sky-400 animate-pulse' : 'bg-neutral-400'"
          />
          <span
            class="text-[10px] uppercase font-semibold"
            :class="portActive ? 'text-sky-600 dark:text-sky-400' : 'text-neutral-400'"
          >
            {{ portActive ? 'LISTENING' : 'OFFLINE' }}
          </span>
        </div>
      </div>

      <!-- Path & Tech Stack -->
      <div class="flex flex-wrap items-center gap-3 text-xs font-mono text-neutral-500">
        <span class="flex items-center gap-1 bg-white dark:bg-neutral-800 px-2 py-0.5 rounded border border-neutral-200 dark:border-neutral-700/60 truncate max-w-lg">
          <UIcon name="i-lucide-folder" class="w-3.5 h-3.5 text-neutral-400 shrink-0" />
          <span class="truncate">{{ comp.absolute_path || comp.relative_path }}</span>
        </span>

        <button
          type="button"
          class="text-sky-600 dark:text-sky-400 hover:underline cursor-pointer"
          title="Salin path direktori aplikasi"
          @click="copyToClipboard(comp.absolute_path || comp.relative_path, 'Path aplikasi disalin!')"
        >
          copy path
        </button>

        <div class="flex flex-wrap items-center gap-1.5 font-sans">
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
    </div>

    <!-- Navigation Tabs -->
    <div class="flex items-center gap-1.5 pt-2 border-t border-neutral-100 dark:border-neutral-800 text-xs font-medium overflow-x-auto pb-0.5">
      <button
        v-for="t in tabs"
        :key="t.id"
        type="button"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-colors cursor-pointer shrink-0 text-xs"
        :class="activeTab === t.id
          ? 'bg-sky-50 dark:bg-sky-950/50 text-sky-700 dark:text-sky-300 font-semibold'
          : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
        @click="activeTab = t.id"
      >
        <UIcon :name="t.icon" class="w-3.5 h-3.5" />
        <span>{{ t.label }}</span>
        <UBadge
          v-if="t.badge !== undefined && (typeof t.badge === 'string' ? !!t.badge : t.badge > 0)"
          size="xs"
          :color="activeTab === t.id ? 'success' : t.badgeColor || 'neutral'"
          variant="subtle"
        >
          {{ t.badge }}
        </UBadge>
      </button>
    </div>
  </div>
</template>
