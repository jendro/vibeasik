<script setup lang="ts">
import type { TrafficRecentEvent } from '~/types'

// Ekor request terbaru (maks 1000 per sumber disimpan engine). Data diperbarui oleh halaman induk.
defineProps<{ events: TrafficRecentEvent[] }>()

const statusCls = (s: number) => s >= 500 ? 'text-rose-600 dark:text-rose-400' : s >= 400 ? 'text-amber-600 dark:text-amber-400' : 'text-emerald-600 dark:text-emerald-400'
</script>

<template>
  <div class="rounded-xl border border-neutral-200 dark:border-white/[0.08] overflow-x-auto">
    <table class="w-full text-xs font-mono">
      <thead class="bg-neutral-50 dark:bg-neutral-800/60 text-neutral-500 text-xs uppercase tracking-wide font-sans">
        <tr>
          <th class="text-left font-semibold px-3 py-1.5">
            Waktu
          </th>
          <th class="text-left font-semibold px-2 py-1.5">
            Request
          </th>
          <th class="text-right font-semibold px-2 py-1.5">
            Status
          </th>
          <th class="text-right font-semibold px-2 py-1.5">
            Lama
          </th>
          <th class="text-left font-semibold px-2 py-1.5">
            Dari
          </th>
          <th class="text-left font-semibold px-3 py-1.5">
            Klien
          </th>
        </tr>
      </thead>
      <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800">
        <tr v-for="e in events" :key="e.id">
          <td class="px-3 py-1 whitespace-nowrap text-neutral-500">
            {{ new Date(e.at).toLocaleTimeString('id-ID') }}
          </td>
          <td class="px-2 py-1 break-all min-w-[200px]">
            <span class="font-semibold">{{ e.method }}</span> {{ e.path }}
            <span v-if="e.route !== e.path" class="text-neutral-400">→ {{ e.route }}</span>
          </td>
          <td class="px-2 py-1 text-right font-semibold" :class="statusCls(e.status)">
            {{ e.status }}
          </td>
          <td class="px-2 py-1 text-right tabular-nums whitespace-nowrap">
            {{ fmtMs(e.dur_ms) }}
          </td>
          <td class="px-2 py-1 whitespace-nowrap text-neutral-600 dark:text-neutral-300">
            {{ e.ip }}<span v-if="e.city || e.country" class="text-neutral-400"> · {{ e.city ? `${e.city}, ${e.country}` : e.country }}</span><span v-if="e.org" class="text-neutral-400"> · {{ e.org }}</span><span v-if="e.referer" class="text-neutral-400"> · {{ e.referer }}</span>
          </td>
          <td class="px-3 py-1 whitespace-nowrap text-neutral-500">
            {{ e.client }} <span class="text-neutral-400">({{ e.ua_class }})</span><span v-if="e.instance" class="text-neutral-400"> · {{ e.instance }}</span>
          </td>
        </tr>
        <tr v-if="!events.length">
          <td colspan="6" class="px-3 py-6 text-center text-neutral-500 font-sans">
            Belum ada request masuk. Pasang SDK lalu akses aplikasi live.
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
