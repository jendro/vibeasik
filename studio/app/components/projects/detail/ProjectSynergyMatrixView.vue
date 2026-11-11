<script setup lang="ts">
import type { SynergyRequestFlow } from '~/types'
import type { SynergyInspectPage, SynergyInspectTab, SynergyMatrixRow } from '~/utils/synergy'

defineProps<{ matrixRows: SynergyMatrixRow[] }>()
const emit = defineEmits<{
  inspect: [flow: SynergyRequestFlow, page: SynergyInspectPage, tab?: SynergyInspectTab]
}>()

const { getMethodColorClass: methodColorClass } = useFormatters()
const getMethodColorClass = (method: string) => methodColorClass(method, 'sky')
const openInspector = (flow: SynergyRequestFlow, page: SynergyInspectPage, tab?: SynergyInspectTab) => emit('inspect', flow, page, tab)
</script>

<template>
  <div class="space-y-4">
    <div class="rounded-xl border border-neutral-200/80 dark:border-white/[0.08] bg-white dark:bg-[#04111f] overflow-hidden">
      <div class="p-3.5 sm:p-4 bg-neutral-50/60 dark:bg-neutral-800/40 border-b border-neutral-200/70 dark:border-white/[0.08] flex flex-wrap items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-table-properties" class="w-4 h-4 text-sky-500" />
          <span class="text-xs font-bold text-neutral-800 dark:text-neutral-200">
            Matriks Audit Menyeluruh ({{ matrixRows.length }} Baris Terhubung)
          </span>
        </div>
        <div class="text-xs text-neutral-400 font-mono">
          Klik tombol "Periksa" pada cell Halaman FE untuk inspeksi alur & cURL
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-neutral-50 dark:bg-neutral-800/80 text-neutral-500 dark:text-neutral-400 font-mono text-xs border-b border-neutral-200 dark:border-white/[0.08]">
            <tr>
              <th class="py-2.5 px-3 min-w-[200px]">
                Halaman FE & Navigasi
              </th>
              <th class="py-2.5 px-3 min-w-[200px]">
                Endpoint API & Handler
              </th>
              <th class="py-2.5 px-3 min-w-[160px]">
                Data Dikirim & Validasi
              </th>
              <th class="py-2.5 px-3 min-w-[150px]">
                Tabel DB & Kolom
              </th>
              <th class="py-2.5 px-3 min-w-[160px]">
                Respon & UI Impact
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/80">
            <tr
              v-for="(row, rIdx) in matrixRows"
              :key="rIdx"
              class="hover:bg-neutral-50/70 dark:hover:bg-neutral-800/40 transition-colors"
            >
              <!-- Halaman FE & Navigasi + Tombol Periksa -->
              <td class="py-2.5 px-3 align-top space-y-1.5">
                <div class="flex items-start justify-between gap-2">
                  <div class="min-w-0 flex-1">
                    <div class="font-bold text-neutral-900 dark:text-white text-xs leading-snug" :title="row.pageTitle">
                      {{ row.pageTitle }}
                    </div>
                    <div class="font-mono text-xs text-sky-600 dark:text-sky-400 truncate mt-0.5" :title="row.pagePath">
                      {{ row.pagePath }}
                    </div>
                  </div>
                  <UButton
                    v-if="row.rawFlow"
                    size="xs"
                    color="primary"
                    variant="soft"
                    icon="i-lucide-search"
                    class="cursor-pointer font-semibold text-xs px-2 py-0.5 shrink-0"
                    @click="openInspector(row.rawFlow, row.rawPage, 'payload')"
                  >
                    Periksa
                  </UButton>
                </div>
                <div v-if="row.filePath" class="text-xs font-mono text-neutral-400 truncate max-w-[220px]" :title="row.filePath">
                  {{ row.filePath }}
                </div>
                <div v-if="row.navigationTargets.length > 0" class="pt-0.5 text-xs text-neutral-500 space-y-0.5">
                  <div v-for="nt in row.navigationTargets.slice(0, 2)" :key="nt" class="truncate font-mono" :title="nt">
                    ➔ {{ nt }}
                  </div>
                </div>
              </td>

              <!-- Endpoint API & Handler (Gabungan Metode, Rute, Handler & Middleware) -->
              <td class="py-2.5 px-3 align-top space-y-1">
                <div class="flex items-center gap-1.5 flex-wrap">
                  <span
                    class="px-1.5 py-0.5 rounded text-xs font-semibold font-mono border shrink-0"
                    :class="getMethodColorClass(row.method)"
                  >
                    {{ row.method }}
                  </span>
                  <span class="font-mono font-bold text-neutral-800 dark:text-neutral-200 text-xs break-all">
                    {{ row.routePath }}
                  </span>
                </div>
                <div class="font-mono text-xs text-neutral-700 dark:text-neutral-300 flex items-center gap-1.5 flex-wrap pt-0.5">
                  <span class="font-semibold text-neutral-800 dark:text-neutral-200">{{ row.handler }}</span>
                  <span
                    v-if="row.middleware && row.middleware !== '-'"
                    class="text-xs text-neutral-500 dark:text-neutral-400 font-sans px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700/60"
                  >
                    {{ row.middleware }}
                  </span>
                </div>
              </td>

              <!-- Data Dikirim & Validasi -->
              <td class="py-2.5 px-3 align-top space-y-1">
                <div class="font-mono text-xs text-neutral-700 dark:text-neutral-300 line-clamp-2" :title="row.payload">
                  {{ row.payload }}
                </div>
                <div class="text-xs text-neutral-500 line-clamp-2" :title="row.validations.join('; ')">
                  {{ row.validations[0] || '-' }}
                </div>
              </td>

              <!-- Tabel DB & Kolom -->
              <td class="py-2.5 px-3 align-top space-y-1">
                <div class="font-bold font-mono text-amber-700 dark:text-amber-400 text-xs break-words">
                  {{ row.dbTables.join(', ') || '-' }}
                </div>
                <div v-if="row.storedFields.length > 0" class="text-xs text-neutral-400 font-mono line-clamp-2" :title="row.storedFields.join(', ')">
                  Kolom: {{ row.storedFields.join(', ') }}
                </div>
              </td>

              <!-- Respon & UI Impact -->
              <td class="py-2.5 px-3 align-top space-y-1">
                <span class="px-1.5 py-0.5 rounded text-xs font-bold font-mono bg-emerald-50 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800 inline-block">
                  HTTP {{ row.statusCode }}
                </span>
                <p class="text-xs text-neutral-600 dark:text-neutral-400 line-clamp-2 pt-0.5 font-medium" :title="row.uiImpact">
                  {{ row.uiImpact }}
                </p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
