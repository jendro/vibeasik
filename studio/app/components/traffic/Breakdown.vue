<script setup lang="ts">
import type { TrafficReport } from '~/types'

// "Dari mana": IP teratas (puncak per menit = dasar rate limit), negara, perujuk, jenis klien, status, instance & 404.
const props = defineProps<{ report: TrafficReport }>()

const uaLabel: Record<string, string> = { desktop: 'Browser desktop', mobile: 'Browser mobile', api: 'Klien API/aplikasi', bot: 'Bot/crawler', unknown: 'Tidak dikenal' }

const lists = computed(() => [
  { title: 'Negara', icon: 'i-lucide-globe', items: props.report.countries, empty: 'Belum ada header negara dari CDN (CF-IPCountry, dst.).' },
  { title: 'Perujuk', icon: 'i-lucide-link', items: props.report.referers, empty: 'Tidak ada perujuk (akses langsung/aplikasi).' },
  { title: 'Jenis klien', icon: 'i-lucide-monitor-smartphone', items: props.report.ua_classes.map(u => ({ ...u, value: uaLabel[u.value] || u.value })), empty: '–' },
  { title: 'Browser / klien', icon: 'i-lucide-app-window', items: props.report.clients, empty: '–' },
  { title: 'Status HTTP', icon: 'i-lucide-hash', items: props.report.status, empty: '–' },
  { title: 'Instance', icon: 'i-lucide-server', items: props.report.instances, empty: '–' }
])
</script>

<template>
  <div class="space-y-4">
    <div class="rounded-xl border border-neutral-200 dark:border-neutral-800 overflow-x-auto">
      <div class="px-3 py-2 text-xs font-bold text-neutral-800 dark:text-neutral-100 flex items-center gap-1.5 border-b border-neutral-100 dark:border-neutral-800">
        <UIcon name="i-lucide-map-pin" class="w-4 h-4 text-neutral-400" /> IP teratas
        <span class="font-normal text-neutral-500">· {{ fmtNum(report.totals.unique_ips) }} IP unik</span>
      </div>
      <table class="w-full text-xs">
        <thead class="text-neutral-500 text-xs uppercase tracking-wide">
          <tr>
            <th class="text-left font-semibold px-3 py-1.5">
              IP
            </th>
            <th class="text-right font-semibold px-2 py-1.5">
              Request
            </th>
            <th class="text-right font-semibold px-2 py-1.5">
              Puncak/mnt
            </th>
            <th class="text-right font-semibold px-2 py-1.5">
              Error
            </th>
            <th class="text-right font-semibold px-3 py-1.5">
              Auth (puncak/mnt)
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800">
          <tr v-for="ip in report.top_ips" :key="ip.ip">
            <td class="px-3 py-1.5 font-mono">
              {{ ip.ip }}
            </td>
            <td class="px-2 py-1.5 text-right tabular-nums">
              {{ fmtNum(ip.count) }} <span class="text-neutral-400">({{ fmtPct(ip.share) }})</span>
            </td>
            <td class="px-2 py-1.5 text-right tabular-nums font-semibold" :class="ip.peak_per_min >= 120 ? 'text-amber-600 dark:text-amber-400' : ''">
              {{ report.gran === 'm' ? fmtNum(ip.peak_per_min) : '–' }}
            </td>
            <td class="px-2 py-1.5 text-right tabular-nums" :class="ip.err_rate >= 0.3 ? 'text-rose-600 dark:text-rose-400' : 'text-neutral-500'">
              {{ fmtPct(ip.err_rate) }}
            </td>
            <td class="px-3 py-1.5 text-right tabular-nums" :class="ip.auth_peak_per_min >= 10 ? 'text-rose-600 dark:text-rose-400 font-semibold' : 'text-neutral-500'">
              {{ ip.auth_hits ? `${fmtNum(ip.auth_hits)} (${report.gran === 'm' ? ip.auth_peak_per_min : '–'})` : '–' }}
            </td>
          </tr>
          <tr v-if="!report.top_ips.length">
            <td colspan="5" class="px-3 py-4 text-center text-neutral-500">
              Belum ada data IP.
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="report.gran !== 'm'" class="px-3 py-2 text-xs text-neutral-500 border-t border-neutral-100 dark:border-neutral-800">
        Puncak per menit hanya tersedia untuk jendela ≤ 24 jam (rangkuman per menit disimpan 48 jam).
      </p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
      <div v-for="l in lists" :key="l.title" class="rounded-xl border border-neutral-200 dark:border-neutral-800 p-3">
        <div class="text-xs font-bold text-neutral-800 dark:text-neutral-100 flex items-center gap-1.5 mb-2">
          <UIcon :name="l.icon" class="w-4 h-4 text-neutral-400" /> {{ l.title }}
        </div>
        <div v-if="l.items.length" class="space-y-1.5">
          <div v-for="it in l.items.slice(0, 8)" :key="it.value" class="text-xs">
            <div class="flex justify-between gap-2">
              <span class="truncate font-mono text-neutral-700 dark:text-neutral-200" :title="it.value">{{ it.value || '(kosong)' }}</span>
              <span class="tabular-nums text-neutral-500 shrink-0">{{ fmtNum(it.count) }} · {{ fmtPct(it.share) }}</span>
            </div>
            <div class="mt-0.5 h-1 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
              <div class="h-full rounded-full bg-primary-500 dark:bg-primary-400" :style="{ width: Math.min(100, it.share * 100) + '%' }" />
            </div>
          </div>
        </div>
        <p v-else class="text-xs text-neutral-500">
          {{ l.empty }}
        </p>
      </div>
    </div>

    <div v-if="report.not_found.length" class="rounded-xl border border-neutral-200 dark:border-neutral-800 p-3">
      <div class="text-xs font-bold text-neutral-800 dark:text-neutral-100 flex items-center gap-1.5 mb-2">
        <UIcon name="i-lucide-file-question" class="w-4 h-4 text-neutral-400" /> Path 404 teratas
        <span class="font-normal text-neutral-500">· path pemindai (.env, wp-, .git) = target bot pencari celah</span>
      </div>
      <div class="flex flex-wrap gap-1.5">
        <UBadge v-for="nf in report.not_found" :key="nf.value" size="sm" color="neutral" variant="soft" class="font-mono">
          {{ nf.value }} · {{ nf.count }}
        </UBadge>
      </div>
    </div>
  </div>
</template>
