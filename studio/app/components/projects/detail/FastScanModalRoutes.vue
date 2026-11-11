<script setup lang="ts">
import type { FastScanResult } from '~/types'

const props = defineProps<{ scanResult: FastScanResult }>()

const { copyToClipboard: copyText } = useCopyToClipboard()
const copyToClipboard = (text: string, label = 'Copied!') => copyText(text, label)

const routeSearchQuery = ref('')
const { getMethodBadgeColor } = useFormatters()

const filteredRoutes = computed(() => {
  if (!props.scanResult?.routes) return []
  const q = routeSearchQuery.value.trim().toLowerCase()
  if (!q) return props.scanResult.routes
  return props.scanResult.routes.filter(r =>
    r.path.toLowerCase().includes(q)
    || r.method.toLowerCase().includes(q)
    || r.handler.toLowerCase().includes(q)
    || r.description.toLowerCase().includes(q)
    || (r.middleware && r.middleware.toLowerCase().includes(q))
  )
})
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-2 p-3 rounded-xl bg-white dark:bg-[#04111f] border border-neutral-200 dark:border-white/[0.08]">
      <div class="flex items-center gap-2">
        <UIcon name="i-lucide-route" class="w-4 h-4 text-primary-600 dark:text-primary-400" />
        <span class="text-xs font-bold text-neutral-800 dark:text-neutral-200">
          Daftar Router & API Endpoints
        </span>
        <UBadge size="xs" color="primary" variant="subtle">
          {{ filteredRoutes.length }} dari {{ scanResult.routes?.length || 0 }} Endpoints
        </UBadge>
      </div>

      <div class="w-full sm:w-64">
        <UInput
          v-model="routeSearchQuery"
          icon="i-lucide-search"
          placeholder="Cari router, path, handler..."
          size="xs"
        />
      </div>
    </div>

    <div v-if="filteredRoutes.length > 0" class="overflow-x-auto rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f]">
      <table class="w-full text-left text-xs">
        <thead class="bg-neutral-50 dark:bg-neutral-800 text-neutral-500 uppercase text-xs tracking-wider border-b border-neutral-200 dark:border-white/[0.08]">
          <tr>
            <th class="p-3 w-20">
              Method
            </th>
            <th class="p-3">
              Endpoint Path
            </th>
            <th class="p-3">
              Fungsi yang Diakses (Handler)
            </th>
            <th class="p-3">
              Keamanan / Auth
            </th>
            <th class="p-3">
              Penjelasan Fungsi
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/60 font-mono text-xs">
          <tr
            v-for="(r, rIdx) in filteredRoutes"
            :key="rIdx"
            class="hover:bg-neutral-50/60 dark:hover:bg-neutral-800/40 transition-colors"
          >
            <td class="p-3 whitespace-nowrap">
              <UBadge
                size="xs"
                :color="getMethodBadgeColor(r.method)"
                variant="subtle"
                class="font-mono font-bold uppercase tracking-wider"
              >
                {{ r.method }}
              </UBadge>
            </td>

            <td class="p-3 font-semibold text-neutral-900 dark:text-neutral-100 whitespace-nowrap">
              <div class="flex items-center gap-1.5">
                <span class="select-all">{{ r.path }}</span>
                <button
                  type="button"
                  class="text-neutral-400 hover:text-primary-600 transition-colors cursor-pointer"
                  title="Copy path"
                  @click="copyToClipboard(r.path, 'Path copied!')"
                >
                  <UIcon name="i-lucide-copy" class="w-3 h-3" />
                </button>
              </div>
            </td>

            <td class="p-3 whitespace-nowrap">
              <div class="flex items-center gap-1 text-purple-700 dark:text-purple-300 font-semibold bg-purple-50 dark:bg-purple-950/40 px-2 py-0.5 rounded border border-purple-200/50 dark:border-purple-900/50 w-fit">
                <UIcon name="i-lucide-code-2" class="w-3 h-3 text-purple-500" />
                <span>{{ r.handler }}</span>
              </div>
            </td>

            <td class="p-3 whitespace-nowrap">
              <span
                class="text-xs px-2 py-0.5 rounded font-sans inline-flex items-center gap-1"
                :class="r.middleware && r.middleware.includes('Protected') ? 'bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300 border border-amber-200/60 dark:border-amber-900/50' : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400'"
              >
                <UIcon
                  :name="r.middleware && r.middleware.includes('Protected') ? 'i-lucide-lock' : 'i-lucide-globe'"
                  class="w-3 h-3"
                />
                <span>{{ r.middleware || 'Public' }}</span>
              </span>
            </td>

            <td class="p-3 font-sans text-xs text-neutral-600 dark:text-neutral-300 leading-relaxed min-w-[240px]">
              {{ r.description }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else class="p-8 text-center text-xs text-neutral-500 italic bg-white dark:bg-[#04111f] rounded-xl border border-neutral-200 dark:border-white/[0.08]">
      Tidak ada router yang cocok dengan pencarian "{{ routeSearchQuery }}".
    </div>
  </div>
</template>
