<script setup lang="ts">
import type { SynergyFrontendPage, SynergyRequestFlow } from '~/types'
import type { SynergyInspectPage, SynergyInspectTab } from '~/utils/synergy'

defineProps<{
  filteredPages: SynergyFrontendPage[]
  collapsedPages: Record<string, boolean>
}>()
const emit = defineEmits<{
  toggle: [pageId: string]
  inspect: [flow: SynergyRequestFlow, page: SynergyInspectPage, tab?: SynergyInspectTab]
}>()

const { getMethodColorClass: methodColorClass } = useFormatters()
const getMethodColorClass = (method: string) => methodColorClass(method, 'sky')
const togglePageCollapse = (pageId: string) => emit('toggle', pageId)
const openInspector = (flow: SynergyRequestFlow, page: SynergyInspectPage, tab?: SynergyInspectTab) => emit('inspect', flow, page, tab)
</script>

<template>
  <div class="space-y-4">
    <div
      v-for="p in filteredPages"
      :key="p.id"
      class="rounded-xl border border-neutral-200/80 dark:border-white/[0.08] bg-white dark:bg-[#04111f] overflow-hidden transition-all duration-200"
    >
      <!-- Page Node Header (Accordion Toggle) -->
      <div
        class="p-4 sm:p-4.5 bg-neutral-50/70 dark:bg-neutral-800/40 border-b border-neutral-200/60 dark:border-white/[0.08] flex flex-col sm:flex-row sm:items-center justify-between gap-3 cursor-pointer hover:bg-neutral-100/60 dark:hover:bg-neutral-800/60 transition-colors"
        @click="togglePageCollapse(p.id)"
      >
        <div class="flex items-center gap-3 min-w-0">
          <div class="p-2 rounded-xl bg-sky-500 text-white shrink-0">
            <UIcon name="i-lucide-monitor" class="w-4 h-4" />
          </div>
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <span class="text-xs font-bold text-neutral-900 dark:text-white truncate">
                {{ p.title }}
              </span>
              <span class="font-mono text-xs text-sky-600 dark:text-sky-400 font-semibold truncate">
                {{ p.page_path }}
              </span>
            </div>
            <p v-if="p.file_path" class="text-xs font-mono text-neutral-400 truncate mt-0.5">
              {{ p.file_path }}
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2 shrink-0">
          <!-- Navigation Destinations Count -->
          <UBadge size="xs" color="success" variant="subtle" class="font-mono text-xs">
            {{ p.navigation_targets.length }} Navigasi
          </UBadge>
          <!-- Outbound API Count -->
          <UBadge size="xs" color="primary" variant="subtle" class="font-mono text-xs">
            {{ p.outbound_requests.length }} API
          </UBadge>
          <!-- Toggle Chevron Icon -->
          <UIcon
            :name="collapsedPages[p.id] ? 'i-lucide-chevron-down' : 'i-lucide-chevron-up'"
            class="w-4 h-4 text-neutral-400"
          />
        </div>
      </div>

      <!-- Expanded Page Pipeline Body -->
      <div v-if="!collapsedPages[p.id]" class="p-4 sm:p-5 space-y-4">
        <!-- Peta Navigasi Antar Halaman -->
        <div class="p-3.5 rounded-xl bg-neutral-50/60 dark:bg-slate-950/40 border border-neutral-200/60 dark:border-white/[0.08] space-y-2">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-1.5 text-xs font-bold text-neutral-700 dark:text-neutral-300">
              <UIcon name="i-lucide-compass" class="w-3.5 h-3.5 text-emerald-500" />
              <span>Peta Navigasi Keluar (Dari Halaman Ini Pengguna Bisa Berpindah Ke):</span>
            </div>
          </div>

          <div v-if="p.navigation_targets.length > 0" class="flex flex-wrap gap-2 pt-0.5">
            <div
              v-for="(nav, nIdx) in p.navigation_targets"
              :key="nIdx"
              class="px-2.5 py-1.5 rounded-lg bg-white dark:bg-[#04111f] border border-neutral-200/70 dark:border-white/[0.08] text-xs space-y-0.5"
            >
              <div class="flex items-center gap-1 font-semibold text-neutral-800 dark:text-neutral-200 text-xs">
                <UIcon name="i-lucide-arrow-right" class="w-3 h-3 text-emerald-500 shrink-0" />
                <span class="font-mono text-emerald-700 dark:text-emerald-400">{{ nav.target_path }}</span>
                <span class="text-neutral-500 font-normal">({{ nav.target_label }})</span>
              </div>
              <div class="text-xs text-neutral-400 pl-4">
                {{ nav.trigger }}
              </div>
            </div>
          </div>
          <div v-else class="text-xs text-neutral-400 italic">
            Tidak ada navigasi keluar langsung dari halaman ini.
          </div>
        </div>

        <!-- Outbound API Branches -->
        <div class="space-y-3">
          <div class="flex items-center justify-between text-xs font-bold text-neutral-700 dark:text-neutral-300">
            <span class="flex items-center gap-1.5">
              <UIcon name="i-lucide-workflow" class="w-3.5 h-3.5 text-sky-500" />
              <span>Alur Cabang API (Klik Setiap Kotak Node Untuk Rincian Lengkap & cURL):</span>
            </span>
            <span class="text-xs text-neutral-400 font-mono">
              {{ p.outbound_requests.length }} alur request
            </span>
          </div>

          <div
            v-if="p.outbound_requests.length === 0"
            class="p-4 rounded-xl border border-dashed border-neutral-200 dark:border-white/[0.08] text-xs text-neutral-400 text-center"
          >
            Halaman statis / presentasi UI (tidak memanggil endpoint API langsung).
          </div>

          <!-- Flow Cards -->
          <div
            v-for="r in p.outbound_requests"
            :key="r.id"
            class="p-4 rounded-xl bg-white dark:bg-[#04111f] border border-neutral-200/80 dark:border-white/[0.08] space-y-3 hover:border-sky-400/50 transition-colors"
          >
            <!-- Request Header Bar -->
            <div class="flex flex-wrap items-center justify-between gap-2 pb-2 border-b border-neutral-100 dark:border-neutral-800/70">
              <div class="flex items-center gap-2">
                <span
                  class="px-2 py-0.5 rounded text-xs font-bold font-mono border uppercase shrink-0"
                  :class="getMethodColorClass(r.http_method)"
                >
                  {{ r.http_method }}
                </span>
                <span class="font-mono text-xs font-bold text-neutral-900 dark:text-neutral-100">
                  {{ r.route_path }}
                </span>
                <span class="text-xs text-neutral-500 hidden sm:inline">
                  • {{ r.action_title }}
                </span>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-xs font-mono text-neutral-400 hidden md:inline">
                  Pemicu: {{ r.trigger_event }}
                </span>
                <!-- Tombol Buka Inspector Detail -->
                <UButton
                  size="xs"
                  color="primary"
                  variant="soft"
                  icon="i-lucide-maximize-2"
                  title="Buka rincian lengkap alur & cURL"
                  class="cursor-pointer font-semibold text-xs"
                  @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'payload')"
                >
                  Detail Alur
                </UButton>
              </div>
            </div>

            <!-- High-Performance Interactive Stage Pills (Click to Inspect) -->
            <div class="flex flex-wrap items-center gap-2 text-xs">
              <!-- Step 1: Payload Pill -->
              <button
                type="button"
                class="px-2.5 py-1.5 rounded-lg bg-sky-50/70 dark:bg-sky-950/40 text-sky-800 dark:text-sky-300 border border-sky-200 dark:border-sky-800/60 font-mono text-xs cursor-pointer hover:bg-sky-100 dark:hover:bg-sky-900/60 transition-colors flex items-center gap-1.5"
                title="Klik untuk membuka detail payload"
                @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'payload')"
              >
                <UIcon name="i-lucide-file-code" class="w-3.5 h-3.5 text-sky-500" />
                <span>📤 Payload: {{ r.request_payload?.params || 'Data Kirim' }}</span>
              </button>

              <span class="text-neutral-400 text-xs">➔</span>

              <!-- Step 2: Validasi Pill -->
              <button
                type="button"
                class="px-2.5 py-1.5 rounded-lg bg-indigo-50/70 dark:bg-indigo-950/40 text-indigo-800 dark:text-indigo-300 border border-indigo-200 dark:border-indigo-800/60 font-mono text-xs cursor-pointer hover:bg-indigo-100 dark:hover:bg-indigo-900/60 transition-colors flex items-center gap-1.5"
                title="Klik untuk membuka detail validasi"
                @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'validation')"
              >
                <UIcon name="i-lucide-shield-check" class="w-3.5 h-3.5 text-indigo-500" />
                <span>🛡️ Validasi: {{ r.validation_rules?.[0] ? `${r.validation_rules.length} aturan` : 'Validasi Schema' }}</span>
              </button>

              <span class="text-neutral-400 text-xs">➔</span>

              <!-- Step 3: Handler Pill -->
              <button
                type="button"
                class="px-2.5 py-1.5 rounded-lg bg-purple-50/70 dark:bg-purple-950/40 text-purple-800 dark:text-purple-300 border border-purple-200 dark:border-purple-800/60 font-mono text-xs cursor-pointer hover:bg-purple-100 dark:hover:bg-purple-900/60 transition-colors flex items-center gap-1.5"
                title="Klik untuk membuka detail router & handler backend"
                @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'handler')"
              >
                <UIcon name="i-lucide-server" class="w-3.5 h-3.5 text-purple-500" />
                <span>⚡ Handler: {{ r.handler_function }}</span>
              </button>

              <span class="text-neutral-400 text-xs">➔</span>

              <!-- Step 4: Database Pill -->
              <button
                type="button"
                class="px-2.5 py-1.5 rounded-lg bg-amber-50/70 dark:bg-amber-950/40 text-amber-800 dark:text-amber-300 border border-amber-200 dark:border-amber-800/60 font-mono text-xs cursor-pointer hover:bg-amber-100 dark:hover:bg-amber-900/60 transition-colors flex items-center gap-1.5"
                title="Klik untuk membuka detail query database"
                @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'database')"
              >
                <UIcon name="i-lucide-database" class="w-3.5 h-3.5 text-amber-500" />
                <span>🗄️ Tabel DB: {{ r.database_operation?.target_tables?.join(', ') || 'DB' }}</span>
              </button>

              <span class="text-neutral-400 text-xs">➔</span>

              <!-- Step 5: Respon Pill -->
              <button
                type="button"
                class="px-2.5 py-1.5 rounded-lg bg-emerald-50/70 dark:bg-emerald-950/40 text-emerald-800 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800/60 font-mono text-xs cursor-pointer hover:bg-emerald-100 dark:hover:bg-emerald-900/60 transition-colors flex items-center gap-1.5"
                title="Klik untuk membuka format respon & dampak UI"
                @click="openInspector(r, { title: p.title, page_path: p.page_path, file_path: p.file_path }, 'response')"
              >
                <UIcon name="i-lucide-check-circle-2" class="w-3.5 h-3.5 text-emerald-500" />
                <span>📥 Respon: HTTP {{ r.response_spec?.status_code || 200 }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
