<script setup lang="ts">
import type { Project, ProjectComponent, SynergyRequestFlow } from '~/types'
import type { SynergyInspectPage, SynergyInspectTab } from '~/utils/synergy'

// Dialog inspektur: detail alur request (payload, validasi, handler, database, response) + cURL.
const props = defineProps<{ project: Project }>()
const toast = useToast()

const isInspectorOpen = ref(false)
const inspectingFlow = ref<SynergyRequestFlow | null>(null)
const inspectingPage = ref<SynergyInspectPage | null>(null)
const inspectorActiveTab = ref<SynergyInspectTab>('payload')

const open = (flow: SynergyRequestFlow, page: SynergyInspectPage, initialTab: SynergyInspectTab = 'payload') => {
  inspectingFlow.value = flow
  inspectingPage.value = page
  inspectorActiveTab.value = initialTab
  isInspectorOpen.value = true
}
defineExpose({ open })

const copyCurlCommand = async (req: SynergyRequestFlow) => {
  const port = props.project.components?.find((c: ProjectComponent) => c.role?.toLowerCase() === 'backend')?.port || 8080
  try {
    await navigator.clipboard.writeText(buildCurlCommand(req, port))
    toast.add({
      title: 'cURL Disalin!',
      description: `Perintah cURL untuk ${req.route_path} siap dijalankan di terminal.`,
      icon: 'i-lucide-terminal',
      color: 'success'
    })
  } catch {
    // fallback
  }
}
</script>

<template>
  <UModal
    v-model:open="isInspectorOpen"
    :title="`Inspeksi Alur: [${inspectingFlow?.http_method || 'API'}] ${inspectingFlow?.route_path || ''}`"
    :description="inspectingFlow?.action_title || 'Rincian alur end-to-end request, validasi, logika backend, dan database'"
    :ui="{ content: 'sm:max-w-4xl' }"
  >
    <template #body>
      <div v-if="inspectingFlow" class="space-y-5 text-xs max-h-[75vh] overflow-y-auto pr-1">
        <!-- Flow Context Summary Banner -->
        <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200/80 dark:border-neutral-800 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div class="space-y-0.5">
            <div class="text-[10px] text-neutral-400 uppercase font-semibold">
              Dipanggil dari Halaman:
            </div>
            <div class="font-bold text-neutral-900 dark:text-white flex items-center gap-1.5">
              <UIcon name="i-lucide-monitor" class="w-3.5 h-3.5 text-sky-500" />
              <span>{{ inspectingPage?.title }}</span>
              <span class="font-mono text-sky-600 dark:text-sky-400">({{ inspectingPage?.page_path }})</span>
            </div>
          </div>
          <!-- Tombol Salin cURL -->
          <UButton
            size="xs"
            color="primary"
            variant="outline"
            icon="i-lucide-terminal"
            class="cursor-pointer shrink-0"
            @click="copyCurlCommand(inspectingFlow)"
          >
            Salin Perintah cURL
          </UButton>
        </div>

        <!-- Inspector Stage Tab Navigation -->
        <div class="flex items-center gap-1 p-1 rounded-xl bg-neutral-100 dark:bg-neutral-800/80 border border-neutral-200 dark:border-neutral-700 overflow-x-auto">
          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all cursor-pointer"
            :class="inspectorActiveTab === 'payload'
              ? 'bg-white dark:bg-slate-900 text-sky-600 dark:text-sky-400 shadow-2xs font-bold'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="inspectorActiveTab = 'payload'"
          >
            <UIcon name="i-lucide-file-code" class="w-3.5 h-3.5 text-sky-500" />
            <span>1. Payload</span>
          </button>

          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all cursor-pointer"
            :class="inspectorActiveTab === 'validation'
              ? 'bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 shadow-2xs font-bold'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="inspectorActiveTab = 'validation'"
          >
            <UIcon name="i-lucide-shield-check" class="w-3.5 h-3.5 text-indigo-500" />
            <span>2. Validasi</span>
          </button>

          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all cursor-pointer"
            :class="inspectorActiveTab === 'handler'
              ? 'bg-white dark:bg-slate-900 text-purple-600 dark:text-purple-400 shadow-2xs font-bold'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="inspectorActiveTab = 'handler'"
          >
            <UIcon name="i-lucide-server" class="w-3.5 h-3.5 text-purple-500" />
            <span>3. Handler & Middleware</span>
          </button>

          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all cursor-pointer"
            :class="inspectorActiveTab === 'database'
              ? 'bg-white dark:bg-slate-900 text-amber-600 dark:text-amber-400 shadow-2xs font-bold'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="inspectorActiveTab = 'database'"
          >
            <UIcon name="i-lucide-database" class="w-3.5 h-3.5 text-amber-500" />
            <span>4. Basis Data (SQL/ORM)</span>
          </button>

          <button
            type="button"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all cursor-pointer"
            :class="inspectorActiveTab === 'response'
              ? 'bg-white dark:bg-slate-900 text-emerald-600 dark:text-emerald-400 shadow-2xs font-bold'
              : 'text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="inspectorActiveTab = 'response'"
          >
            <UIcon name="i-lucide-check-circle-2" class="w-3.5 h-3.5 text-emerald-500" />
            <span>5. Respon & UI</span>
          </button>
        </div>

        <!-- Tab Content 1: Payload -->
        <div v-if="inspectorActiveTab === 'payload'" class="space-y-4">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Pemicu Event Frontend:</span>
              <p class="font-semibold text-neutral-900 dark:text-white">
                {{ inspectingFlow.trigger_event }}
              </p>
            </div>
            <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Metode & Endpoint:</span>
              <p class="font-mono font-bold text-sky-600 dark:text-sky-400">
                {{ inspectingFlow.http_method }} {{ inspectingFlow.route_path }}
              </p>
            </div>
          </div>

          <div v-if="inspectingFlow.request_payload?.params" class="space-y-1">
            <span class="text-[10px] uppercase font-bold text-neutral-400">Parameter URL (Path Params):</span>
            <p class="p-2.5 rounded-lg bg-neutral-100 dark:bg-neutral-800 font-mono text-[11px] text-neutral-800 dark:text-neutral-200">
              {{ inspectingFlow.request_payload.params }}
            </p>
          </div>

          <div v-if="inspectingFlow.request_payload?.query" class="space-y-1">
            <span class="text-[10px] uppercase font-bold text-neutral-400">Query Parameters:</span>
            <p class="p-2.5 rounded-lg bg-neutral-100 dark:bg-neutral-800 font-mono text-[11px] text-neutral-800 dark:text-neutral-200">
              {{ inspectingFlow.request_payload.query }}
            </p>
          </div>

          <div v-if="inspectingFlow.request_payload?.headers" class="space-y-1">
            <span class="text-[10px] uppercase font-bold text-neutral-400">HTTP Headers:</span>
            <p class="p-2.5 rounded-lg bg-neutral-100 dark:bg-neutral-800 font-mono text-[11px] text-neutral-800 dark:text-neutral-200">
              {{ inspectingFlow.request_payload.headers }}
            </p>
          </div>

          <div v-if="inspectingFlow.request_payload?.body" class="space-y-1">
            <span class="text-[10px] uppercase font-bold text-neutral-400">JSON Request Body:</span>
            <pre class="p-3 rounded-xl bg-neutral-950 text-sky-400 font-mono text-[11px] overflow-x-auto border border-neutral-800 leading-relaxed">{{ inspectingFlow.request_payload.body }}</pre>
          </div>
        </div>

        <!-- Tab Content 2: Validasi -->
        <div v-else-if="inspectorActiveTab === 'validation'" class="space-y-3">
          <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-2">
            <span class="text-xs font-bold text-neutral-900 dark:text-white flex items-center gap-1.5">
              <UIcon name="i-lucide-shield-check" class="w-4 h-4 text-indigo-500" />
              <span>Aturan Validasi Integritas Data (Frontend & Backend):</span>
            </span>
            <ul v-if="inspectingFlow.validation_rules?.length" class="space-y-2 pt-1">
              <li
                v-for="(rule, rIdx) in inspectingFlow.validation_rules"
                :key="rIdx"
                class="flex items-start gap-2 p-2 rounded-lg bg-white dark:bg-slate-900 border border-neutral-200/60 dark:border-neutral-800 text-neutral-700 dark:text-neutral-300"
              >
                <UIcon name="i-lucide-check" class="w-4 h-4 text-emerald-500 shrink-0 mt-0.5" />
                <span class="leading-relaxed">{{ rule }}</span>
              </li>
            </ul>
            <p v-else class="text-neutral-400 italic">
              Validasi tipe data default & unmarshaling JSON schema.
            </p>
          </div>
        </div>

        <!-- Tab Content 3: Handler & Middleware -->
        <div v-else-if="inspectorActiveTab === 'handler'" class="space-y-4">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Fungsi Handler Controller:</span>
              <p class="font-mono font-bold text-purple-700 dark:text-purple-300 text-xs">
                {{ inspectingFlow.handler_function }}
              </p>
            </div>
            <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Guard / Middleware:</span>
              <p class="font-mono text-neutral-800 dark:text-neutral-200 text-xs">
                {{ inspectingFlow.middleware }}
              </p>
            </div>
          </div>

          <div v-if="inspectingFlow.business_process?.length" class="space-y-2">
            <span class="text-[10px] uppercase font-bold text-neutral-400">Tahapan Proses Logika Bisnis di Backend:</span>
            <ol class="space-y-2">
              <li
                v-for="(step, sIdx) in inspectingFlow.business_process"
                :key="sIdx"
                class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200/70 dark:border-neutral-800 flex items-start gap-2.5"
              >
                <span class="px-1.5 py-0.5 rounded font-mono font-bold text-[10px] bg-purple-100 dark:bg-purple-950 text-purple-700 dark:text-purple-300">
                  {{ sIdx + 1 }}
                </span>
                <span class="text-neutral-700 dark:text-neutral-300 leading-relaxed">{{ step }}</span>
              </li>
            </ol>
          </div>
        </div>

        <!-- Tab Content 4: Basis Data -->
        <div v-else-if="inspectorActiveTab === 'database'" class="space-y-4">
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <div class="p-3 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Database Engine:</span>
              <p class="font-bold text-neutral-900 dark:text-white">
                {{ inspectingFlow.database_operation?.engine || 'SQLite' }}
              </p>
            </div>
            <div class="p-3 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Operasi:</span>
              <p class="font-mono font-bold text-amber-600 dark:text-amber-400">
                {{ inspectingFlow.database_operation?.operation_type || 'SELECT' }}
              </p>
            </div>
            <div class="p-3 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Tabel Terhubung:</span>
              <p class="font-mono font-bold text-amber-700 dark:text-amber-300 truncate">
                {{ inspectingFlow.database_operation?.target_tables?.join(', ') || '-' }}
              </p>
            </div>
          </div>

          <div v-if="inspectingFlow.database_operation?.stored_fields?.length" class="space-y-1">
            <span class="text-[10px] uppercase font-bold text-neutral-400">Kolom Data yang Disimpan / Dimutasi:</span>
            <div class="flex flex-wrap gap-1.5 p-2 rounded-lg bg-neutral-100 dark:bg-neutral-800">
              <span
                v-for="sf in inspectingFlow.database_operation.stored_fields"
                :key="sf"
                class="px-2 py-0.5 rounded bg-white dark:bg-slate-900 border border-neutral-200 dark:border-neutral-700 font-mono text-[10px] font-bold text-amber-700 dark:text-amber-400"
              >
                {{ sf }}
              </span>
            </div>
          </div>

          <div v-if="inspectingFlow.database_operation?.query_detail" class="space-y-1">
            <span class="text-[10px] uppercase font-bold text-neutral-400">Query SQL / Eksekusi ORM:</span>
            <pre class="p-3 rounded-xl bg-neutral-950 text-amber-300 font-mono text-[11px] overflow-x-auto border border-neutral-800 leading-relaxed">{{ inspectingFlow.database_operation.query_detail }}</pre>
          </div>
        </div>

        <!-- Tab Content 5: Respon & UI Impact -->
        <div v-else-if="inspectorActiveTab === 'response'" class="space-y-4">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Status Respon HTTP:</span>
              <p class="font-mono font-bold text-emerald-600 dark:text-emerald-400 text-sm">
                HTTP {{ inspectingFlow.response_spec?.status_code || 200 }}
              </p>
            </div>
            <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
              <span class="text-[10px] uppercase font-bold text-neutral-400">Tipe Data:</span>
              <p class="font-mono font-semibold text-neutral-800 dark:text-neutral-200 text-sm">
                {{ inspectingFlow.response_spec?.data_type || 'JSON Object' }}
              </p>
            </div>
          </div>

          <div class="p-3.5 rounded-xl bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1">
            <span class="text-[10px] uppercase font-bold text-neutral-400">Dampak pada Antarmuka (UI Reaction):</span>
            <p class="font-semibold text-neutral-900 dark:text-white leading-relaxed">
              {{ inspectingFlow.ui_impact || 'Perbarui state reaktif dan render ulang komponen tampilan.' }}
            </p>
          </div>

          <div v-if="inspectingFlow.response_spec?.body_sample" class="space-y-1">
            <span class="text-[10px] uppercase font-bold text-neutral-400">Contoh Payload Respon JSON:</span>
            <pre class="p-3 rounded-xl bg-neutral-950 text-emerald-400 font-mono text-[11px] overflow-x-auto border border-neutral-800 leading-relaxed">{{ inspectingFlow.response_spec.body_sample }}</pre>
          </div>
        </div>
      </div>
    </template>
  </UModal>
</template>
