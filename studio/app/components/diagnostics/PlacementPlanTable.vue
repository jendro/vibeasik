<script setup lang="ts">
import type { PlacementStep } from '~/types/diagnostics'

// Rencana penempatan logika hasil LLM: fungsi apa pindah ke berkas/layer mana sesuai peta layer proyek,
// atau dipakai ulang/disatukan dengan fungsi yang sudah ada (cegah fungsi kembar).
defineProps<{ plan: PlacementStep[] }>()

const actionLabel: Record<string, string> = {
  pindah: 'Pindah',
  pakai_ulang: 'Pakai ulang',
  satukan: 'Satukan',
  tetap: 'Tetap',
  baru: 'Baru'
}
</script>

<template>
  <div class="space-y-2">
    <div class="flex items-center gap-1.5 text-xs font-semibold text-neutral-700 dark:text-neutral-300">
      <UIcon name="i-lucide-map" class="w-3.5 h-3.5 text-emerald-500" />
      <span>Peta Penempatan Logika (sesuai Peta Layer Proyek):</span>
    </div>
    <div class="overflow-x-auto rounded-lg border border-neutral-200/80 dark:border-white/[0.08]">
      <table class="w-full text-xs">
        <thead class="bg-neutral-50 dark:bg-neutral-800/40 text-neutral-500 text-left">
          <tr>
            <th class="px-2.5 py-1.5 font-semibold">
              Simbol
            </th>
            <th class="px-2.5 py-1.5 font-semibold">
              Aksi
            </th>
            <th class="px-2.5 py-1.5 font-semibold">
              Berkas Tujuan
            </th>
            <th class="px-2.5 py-1.5 font-semibold">
              Layer
            </th>
            <th class="px-2.5 py-1.5 font-semibold">
              Pakai Ulang / Alasan
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(step, idx) in plan"
            :key="idx"
            class="border-t border-neutral-100 dark:border-white/[0.08] align-top text-neutral-800 dark:text-neutral-200"
          >
            <td class="px-2.5 py-1.5 font-mono">
              {{ step.symbol || '-' }}
            </td>
            <td class="px-2.5 py-1.5 whitespace-nowrap">
              {{ actionLabel[step.action] || step.action }}
            </td>
            <td class="px-2.5 py-1.5 font-mono break-all">
              {{ step.target || '-' }}
              <span v-if="step.warning" class="block mt-0.5 font-sans text-xs text-amber-600 dark:text-amber-400">
                ⚠️ {{ step.warning }}
              </span>
            </td>
            <td class="px-2.5 py-1.5">
              {{ step.layer || '-' }}
            </td>
            <td class="px-2.5 py-1.5 leading-relaxed">
              <span v-if="step.reuse" class="block font-mono text-emerald-700 dark:text-emerald-400">{{ step.reuse }}</span>
              <span v-if="step.reason">{{ step.reason }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
