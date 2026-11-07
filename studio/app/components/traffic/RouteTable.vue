<script setup lang="ts">
import type { TrafficGroupRow, TrafficRouteRow } from '~/types'

// Statistik per endpoint & per kelompok rute (dasar menimbang cache, optimasi & pemecahan service).
const props = defineProps<{ routes: TrafficRouteRow[], groups: TrafficGroupRow[] }>()

type SortKey = 'count' | 'time_share' | 'p95' | 'err_rate' | 'avg_out'
const view = ref<'routes' | 'groups'>('routes')
const sortKey = ref<SortKey>('count')
const query = ref('')
const limit = ref(25)

const sortItems = [
  { label: 'Terbanyak', value: 'count' },
  { label: 'Porsi waktu server', value: 'time_share' },
  { label: 'Paling lambat (p95)', value: 'p95' },
  { label: 'Error 5xx', value: 'err_rate' },
  { label: 'Respons terbesar', value: 'avg_out' }
]

const rows = computed(() => {
  const q = query.value.trim().toLowerCase()
  return props.routes
    .filter(r => !q || `${r.method} ${r.route}`.toLowerCase().includes(q))
    .slice()
    .sort((a, b) => b[sortKey.value] - a[sortKey.value])
})
const maxCount = computed(() => Math.max(1, ...props.routes.map(r => r.count)))
const latencyCls = (v: number) => v >= 3000 ? 'text-rose-600 dark:text-rose-400' : v >= 1000 ? 'text-amber-600 dark:text-amber-400' : 'text-neutral-700 dark:text-neutral-200'
const methodCls: Record<string, string> = {
  GET: 'text-sky-700 dark:text-sky-300', POST: 'text-emerald-700 dark:text-emerald-300', PUT: 'text-amber-700 dark:text-amber-300',
  PATCH: 'text-amber-700 dark:text-amber-300', DELETE: 'text-rose-700 dark:text-rose-300'
}
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center gap-2">
      <div class="flex rounded-lg border border-neutral-200 dark:border-neutral-800 p-0.5">
        <UButton size="xs" :variant="view === 'routes' ? 'soft' : 'ghost'" :color="view === 'routes' ? 'primary' : 'neutral'" @click="view = 'routes'">
          Endpoint · {{ routes.length }}
        </UButton>
        <UButton size="xs" :variant="view === 'groups' ? 'soft' : 'ghost'" :color="view === 'groups' ? 'primary' : 'neutral'" @click="view = 'groups'">
          Kelompok · {{ groups.length }}
        </UButton>
      </div>
      <template v-if="view === 'routes'">
        <UInput v-model="query" size="xs" icon="i-lucide-search" placeholder="Cari rute" class="w-44" />
        <USelect v-model="sortKey" :items="sortItems" size="xs" class="w-48" />
      </template>
    </div>

    <div v-if="view === 'routes'" class="overflow-x-auto rounded-xl border border-neutral-200 dark:border-neutral-800">
      <table class="w-full text-xs">
        <thead class="bg-neutral-50 dark:bg-neutral-800/60 text-neutral-500 text-[10px] uppercase tracking-wide">
          <tr>
            <th class="text-left font-semibold px-3 py-2">
              Endpoint
            </th>
            <th class="text-right font-semibold px-2 py-2">
              Request
            </th>
            <th class="text-right font-semibold px-2 py-2" title="Porsi total waktu server">
              Waktu
            </th>
            <th class="text-right font-semibold px-2 py-2">
              p50
            </th>
            <th class="text-right font-semibold px-2 py-2">
              p95
            </th>
            <th class="text-right font-semibold px-2 py-2">
              p99
            </th>
            <th class="text-right font-semibold px-2 py-2">
              5xx
            </th>
            <th class="text-right font-semibold px-2 py-2">
              4xx
            </th>
            <th class="text-right font-semibold px-3 py-2">
              Respons
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800">
          <tr v-for="r in rows.slice(0, limit)" :key="r.method + r.route" class="hover:bg-neutral-50/70 dark:hover:bg-neutral-800/40">
            <td class="px-3 py-1.5 min-w-[220px]">
              <div class="font-mono break-all">
                <span class="font-semibold" :class="methodCls[r.method] || 'text-neutral-600'">{{ r.method }}</span> {{ r.route }}
              </div>
              <div class="mt-1 h-1 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
                <div class="h-full rounded-full bg-primary-500 dark:bg-primary-400" :style="{ width: (r.count / maxCount * 100) + '%' }" />
              </div>
            </td>
            <td class="px-2 py-1.5 text-right tabular-nums whitespace-nowrap">
              {{ fmtNum(r.count) }}<div class="text-[10px] text-neutral-400">
                {{ fmtNum(r.rpm) }}/mnt
              </div>
            </td>
            <td class="px-2 py-1.5 text-right tabular-nums whitespace-nowrap">
              {{ fmtPct(r.time_share) }}
            </td>
            <td class="px-2 py-1.5 text-right tabular-nums whitespace-nowrap">
              {{ fmtMs(r.p50) }}
            </td>
            <td class="px-2 py-1.5 text-right tabular-nums whitespace-nowrap font-semibold" :class="latencyCls(r.p95)">
              {{ fmtMs(r.p95) }}
            </td>
            <td class="px-2 py-1.5 text-right tabular-nums whitespace-nowrap" :class="latencyCls(r.p99)">
              {{ fmtMs(r.p99) }}
            </td>
            <td class="px-2 py-1.5 text-right tabular-nums whitespace-nowrap" :class="r.err_rate >= 0.02 ? 'text-rose-600 dark:text-rose-400 font-semibold' : ''">
              {{ r.err_5xx ? fmtPct(r.err_rate) : '–' }}
            </td>
            <td class="px-2 py-1.5 text-right tabular-nums whitespace-nowrap text-neutral-500">
              {{ r.err_4xx ? fmtNum(r.err_4xx) : '–' }}
            </td>
            <td class="px-3 py-1.5 text-right tabular-nums whitespace-nowrap text-neutral-500">
              {{ fmtBytes(r.avg_out) }}
            </td>
          </tr>
          <tr v-if="!rows.length">
            <td colspan="9" class="px-3 py-6 text-center text-neutral-500">
              Belum ada request di jendela ini.
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="rows.length > limit" class="p-2 text-center border-t border-neutral-100 dark:border-neutral-800">
        <UButton size="xs" color="neutral" variant="ghost" @click="limit += 50">
          Tampilkan lebih banyak ({{ rows.length - limit }} lagi)
        </UButton>
      </div>
    </div>

    <div v-else class="space-y-2">
      <p class="text-xs text-neutral-500 leading-relaxed">
        Kelompok = segmen pertama rute (setelah /api & /vN). Kelompok yang porsi waktunya jauh di atas porsi request-nya adalah kandidat dipisah jadi service/worker sendiri.
      </p>
      <div v-for="g in groups" :key="g.group" class="rounded-xl border border-neutral-200 dark:border-neutral-800 p-3">
        <div class="flex flex-wrap items-baseline justify-between gap-2">
          <span class="font-mono text-sm font-semibold text-neutral-900 dark:text-white">{{ g.group }}</span>
          <span class="text-[11px] text-neutral-500">{{ g.routes }} rute · {{ fmtNum(g.count) }} request · p95 {{ fmtMs(g.p95) }} · 5xx {{ fmtPct(g.err_rate) }}</span>
        </div>
        <div class="mt-2 space-y-1">
          <div class="flex items-center gap-2 text-[11px]">
            <span class="w-24 shrink-0 text-neutral-500">Porsi request</span>
            <div class="flex-1 h-2 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
              <div class="h-full rounded-full bg-primary-500 dark:bg-primary-400" :style="{ width: g.share * 100 + '%' }" />
            </div>
            <span class="w-12 text-right tabular-nums whitespace-nowrap">{{ fmtPct(g.share) }}</span>
          </div>
          <div class="flex items-center gap-2 text-[11px]">
            <span class="w-24 shrink-0 text-neutral-500">Porsi waktu</span>
            <div class="flex-1 h-2 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
              <div class="h-full rounded-full bg-amber-500 dark:bg-amber-400" :style="{ width: g.time_share * 100 + '%' }" />
            </div>
            <span class="w-12 text-right tabular-nums whitespace-nowrap">{{ fmtPct(g.time_share) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
