<script setup lang="ts">
import type { Project } from '~/types'

defineProps<{
  activeProject: Project | null
  activeProjectId: number | null
  loading: boolean
  initingProject: boolean
  autopilotEnabled: boolean
  batchCount: number
}>()
const autopilotAgent = defineModel<'agy' | 'cline' | 'opencode' | 'auto'>('autopilotAgent', { required: true })
const emit = defineEmits<{
  openGuidelines: []
  refresh: []
  initProject: []
  toggleAutopilot: [value: boolean]
  openMerge: []
  openBatch: []
  openPrompts: []
}>()
const handleToggleAutopilot = (value: boolean) => emit('toggleAutopilot', value)

// Disembunyikan sementara: "Kerjakan Otomatis" & "Batch" menjalankan agent CLI di luar IDE dan tidak
// mengeksekusi seperti di IDE. Todo dikerjakan langsung oleh sub agent di aplikasi AI coding (lewat MCP).
const showAutoRun = false
</script>

<template>
  <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-3 border-b border-neutral-200 dark:border-white/10">
    <div class="min-w-0">
      <div class="flex items-center gap-2.5 flex-wrap">
        <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-clipboard-check" class="w-6 h-6 text-violet-500 shrink-0" />
          <span>Perencanaan & Agenda Bertingkat</span>
        </h1>
        <UBadge v-if="activeProject" size="sm" color="primary" variant="subtle">
          {{ activeProject.name }}
        </UBadge>
      </div>
      <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1">
        Fokus perencanaan teknis proyek aktif (Bundle & Komponen/Module) dengan rubrik terukur, pelacak tugas sisa, dan saran AI Agent.
      </p>
    </div>

    <!-- Action Buttons Row: Rata Kanan -->
    <div class="flex items-center justify-end flex-wrap gap-2.5 shrink-0 self-end md:self-center ml-auto">
      <UButton
        icon="i-lucide-book-open"
        color="neutral"
        variant="outline"
        size="sm"
        title="Buka daftar buku panduan dan rubrik penilaian AI"
        @click="emit('openGuidelines')"
      >
        Lihat Patokan
      </UButton>

      <UButton
        icon="i-lucide-refresh-cw"
        color="neutral"
        variant="outline"
        size="sm"
        :loading="loading"
        title="Muat ulang data todo proyek aktif"
        @click="emit('refresh')"
      >
        Refresh
      </UButton>

      <UButton
        v-if="activeProjectId"
        icon="i-lucide-clipboard-copy"
        color="neutral"
        variant="outline"
        size="sm"
        title="Prompt siap copas untuk menyuruh Claude Code, Antigravity, Cline, Cursor, atau Codex mengerjakan todo lewat MCP vibeasik"
        @click="emit('openPrompts')"
      >
        Prompt Agent
      </UButton>

      <UButton
        v-if="activeProjectId"
        icon="i-lucide-sparkles"
        color="info"
        variant="outline"
        size="sm"
        :loading="initingProject"
        title="Siapkan proyek untuk AI agent: tulis .vibeasik.json (project ID & TODO) dan peta kode .vibeasik/CODEMAP.md yang diperbarui otomatis saat kode berubah."
        @click="emit('initProject')"
      >
        Init vibeasik
      </UButton>

      <!-- Toggle Kerjakan Otomatis (Autopilot) -->
      <div v-if="showAutoRun" class="flex items-center gap-1 p-0.5 rounded-xl bg-neutral-100 dark:bg-neutral-800/80 border border-neutral-200 dark:border-white/10 shadow-2xs">
        <div class="flex items-center gap-2 px-2.5 py-1">
          <span class="relative flex h-2 w-2">
            <span v-if="autopilotEnabled" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
            <span class="relative inline-flex rounded-full h-2 w-2" :class="autopilotEnabled ? 'bg-emerald-500' : 'bg-neutral-400'" />
          </span>
          <span
            class="text-xs font-semibold text-neutral-800 dark:text-neutral-200 select-none cursor-pointer"
            @click="handleToggleAutopilot(!autopilotEnabled)"
          >
            Kerjakan Otomatis
          </span>
          <USwitch
            :model-value="autopilotEnabled"
            color="success"
            size="sm"
            title="Aktifkan untuk mengerjakan todo proyek aktif satu per satu secara otomatis"
            @update:model-value="handleToggleAutopilot"
          />
        </div>

        <!-- Quick Config Popover for Autopilot -->
        <UPopover mode="click" :content="{ side: 'bottom', align: 'end' }">
          <UButton
            icon="i-lucide-sliders-horizontal"
            color="neutral"
            variant="ghost"
            size="xs"
            title="Pengaturan CLI & Model Pengerjaan Otomatis"
          />
          <template #content>
            <div class="p-3.5 w-72 space-y-3 text-xs">
              <div class="flex items-center justify-between pb-1.5 border-b border-neutral-100 dark:border-neutral-800">
                <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1.5">
                  <UIcon name="i-lucide-bot" class="w-4 h-4 text-sky-500" />
                  <span>Konfigurasi Autopilot</span>
                </span>
                <UBadge size="xs" :color="autopilotEnabled ? 'success' : 'neutral'" variant="subtle">
                  {{ autopilotEnabled ? 'Aktif' : 'Standby' }}
                </UBadge>
              </div>

              <div class="space-y-1">
                <label class="text-xs font-semibold text-neutral-600 dark:text-neutral-400">Pilihan CLI Coding Agent:</label>
                <select
                  v-model="autopilotAgent"
                  class="w-full px-2 py-1.5 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 text-xs focus:ring-1 focus:ring-sky-500 cursor-pointer"
                >
                  <option value="agy">
                    ⭐ Antigravity CLI (7 Akun Aktif, Terbukti Mengubah Kode) [Rekomendasi]
                  </option>
                  <option value="cline">
                    🚀 Cline CLI (Otonom, gunakan provider di cline auth)
                  </option>
                  <option value="opencode">
                    ⚡ OpenCode (Provider Kustom / API Key)
                  </option>
                  <option value="auto">
                    🌐 Otomatis (Router Terbaik)
                  </option>
                </select>
              </div>

              <div class="p-2 rounded-lg bg-sky-50/80 dark:bg-sky-950/30 border border-sky-200/80 dark:border-sky-900/40 text-xs text-sky-900 dark:text-sky-200 space-y-1">
                <p class="font-semibold flex items-center gap-1">
                  <UIcon name="i-lucide-check-circle" class="w-3.5 h-3.5 text-sky-600 dark:text-sky-400" />
                  <span>Status Eksekusi Coding:</span>
                </p>
                <p class="text-neutral-600 dark:text-neutral-400 leading-tight">
                  • <strong>Antigravity CLI:</strong> Mengubah kode langsung via Gemini Flash, rotasi 7 akun tanpa limit.<br>
                  • <strong>Cline CLI:</strong> Tool calling otonom via CLI Cline.<br>
                  • <strong>OpenCode:</strong> Memerlukan provider API valid (bukan mock/troll).
                </p>
              </div>
            </div>
          </template>
        </UPopover>
      </div>

      <UButton
        v-if="activeProjectId"
        icon="i-lucide-layers"
        color="neutral"
        variant="outline"
        size="sm"
        title="Lebur todo serupa (rule/berkas sama atau isi mirip) menjadi satu todo gabungan yang dikerjakan satu agent sekaligus"
        @click="emit('openMerge')"
      >
        Lebur Serupa
      </UButton>

      <UButton
        v-if="showAutoRun"
        icon="i-lucide-bot"
        color="neutral"
        variant="outline"
        size="sm"
        :disabled="!batchCount"
        title="Kirim semua todo pending yang tampil ke agent coding; agent & effort dipilih sesuai kerumitan tiap todo"
        @click="emit('openBatch')"
      >
        Batch ({{ batchCount }})
      </UButton>
    </div>
  </div>
</template>
