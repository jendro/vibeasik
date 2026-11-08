<script setup lang="ts">
import type { ProjectComponent } from '~/types'

const props = defineProps<{ comp: ProjectComponent }>()
const { copyToClipboard } = useCopyToClipboard()

const mcpConfigJson = computed(() => {
  const compDir = props.comp.absolute_path || '.'
  return JSON.stringify({
    mcpServers: {
      vibeasik: {
        command: 'vibeasik',
        args: ['mcp'],
        env: {
          ACTIVE_APP_PATH: compDir,
          ACTIVE_APP_NAME: props.comp.name || ''
        }
      }
    }
  }, null, 2)
})
</script>

<template>
  <div class="space-y-6">
    <div class="p-5 sm:p-6 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs space-y-4">
      <div>
        <h4 class="font-bold text-neutral-900 dark:text-white text-sm flex items-center gap-2">
          <UIcon name="i-lucide-cpu" class="w-4 h-4 text-sky-400" />
          <span>Model Context Protocol (MCP) untuk {{ comp.name }}</span>
        </h4>
        <p class="text-xs text-neutral-500 mt-1 leading-relaxed">
          Gunakan konfigurasi ini agar AI agent (Google Antigravity, Claude Code, Cursor) memiliki konteks langsung dan direktori kerja di aplikasi ini.
        </p>
      </div>

      <!-- Target Directory Indicator -->
      <div class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-700/60 text-xs font-mono flex items-center justify-between">
        <span class="text-neutral-500">Direktori Kerja Agent:</span>
        <span class="text-neutral-900 dark:text-neutral-100 font-bold truncate max-w-sm">{{ comp.absolute_path }}</span>
      </div>

      <!-- MCP Configuration JSON -->
      <div class="p-4 rounded-xl bg-neutral-950 text-neutral-200 space-y-2 text-xs">
        <div class="flex items-center justify-between">
          <span class="font-mono text-sky-400 font-semibold text-xs">stdio Config JSON</span>
          <UButton
            icon="i-lucide-copy"
            color="neutral"
            variant="ghost"
            size="xs"
            @click="copyToClipboard(mcpConfigJson, 'Konfigurasi MCP disalin!')"
          >
            Salin Konfigurasi
          </UButton>
        </div>
        <pre class="p-3 rounded-lg bg-[#010810] font-mono text-sky-300 border border-white/10 text-xs overflow-x-auto">{{ mcpConfigJson }}</pre>
      </div>

      <!-- Agent Usage Guidance -->
      <div class="p-4 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-200 dark:border-neutral-800 text-xs space-y-2">
        <div class="font-bold text-neutral-900 dark:text-white">
          Cara Menggunakan:
        </div>
        <ul class="list-disc list-inside space-y-1 text-neutral-600 dark:text-neutral-300">
          <li>Buka terminal di folder <code class="font-mono text-sky-600 dark:text-sky-400">{{ comp.absolute_path }}</code></li>
          <li>Jalankan agent dengan perintah <code class="font-mono">agy</code> atau <code class="font-mono">claude</code></li>
          <li>Agent akan otomatis memanfaatkan tool MCP vibeasik untuk memeriksa rute, port, dan dependensi.</li>
        </ul>
      </div>
    </div>
  </div>
</template>
