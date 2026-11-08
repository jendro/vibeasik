<script setup lang="ts">
import type { FastScanResult } from '~/types'

defineProps<{ scanResult: FastScanResult }>()

const { copyToClipboard } = useCopyToClipboard()
</script>

<template>
  <div class="space-y-4">
    <div class="p-4 rounded-xl border border-amber-200 dark:border-amber-900/60 bg-amber-50/50 dark:bg-amber-950/20 space-y-3 shadow-2xs">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-amber-500/10 border border-amber-300 dark:border-amber-700 flex items-center justify-center text-amber-600 dark:text-amber-400">
            <UIcon name="i-lucide-shield-check" class="w-4 h-4" />
          </div>
          <div>
            <div class="text-xs font-bold text-amber-900 dark:text-amber-200 flex items-center gap-2">
              <span>Double Security Censorship Shield</span>
              <UBadge size="xs" color="warning" variant="subtle">
                {{ scanResult.security_audit?.double_security_level || 'ACTIVE_ENFORCED' }}
              </UBadge>
            </div>
            <p class="text-xs text-amber-700 dark:text-amber-300 mt-0.5">
              {{ scanResult.security_audit?.message || 'Semua kunci rahasia, token otentikasi, dan password disensor secara berlapis sebelum dikirim ke LLM.' }}
            </p>
          </div>
        </div>

        <NuxtLink
          to="/settings?tab=censored_keys"
          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold bg-white dark:bg-neutral-800 border border-amber-300 dark:border-amber-700 text-amber-800 dark:text-amber-300 hover:bg-amber-100/50 dark:hover:bg-neutral-700 transition-colors shadow-xs"
        >
          <UIcon name="i-lucide-sliders" class="w-3.5 h-3.5" />
          <span>Buka Security Vault Settings</span>
        </NuxtLink>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5 pt-1">
        <div class="p-3 rounded-lg bg-white/80 dark:bg-neutral-900/80 border border-amber-200/60 dark:border-amber-900/40">
          <div class="text-xs uppercase font-bold text-neutral-400">
            Total Kunci Tersensor
          </div>
          <div class="text-lg font-bold font-mono text-amber-600 dark:text-amber-400 mt-0.5">
            {{ scanResult.security_audit?.redacted_count || 0 }} rahasia
          </div>
        </div>
        <div class="p-3 rounded-lg bg-white/80 dark:bg-neutral-900/80 border border-amber-200/60 dark:border-amber-900/40">
          <div class="text-xs uppercase font-bold text-neutral-400">
            Match Keyword Khusus (Pass 1)
          </div>
          <div class="text-lg font-bold font-mono text-purple-600 dark:text-purple-400 mt-0.5">
            {{ scanResult.security_audit?.custom_matches || 0 }} kecocokan
          </div>
        </div>
        <div class="p-3 rounded-lg bg-white/80 dark:bg-neutral-900/80 border border-amber-200/60 dark:border-amber-900/40">
          <div class="text-xs uppercase font-bold text-neutral-400">
            Pola Ancaman Bawaan (Pass 2)
          </div>
          <div class="text-lg font-bold font-mono text-sky-600 dark:text-sky-400 mt-0.5">
            {{ scanResult.security_audit?.built_in_matches || 0 }} dinetralkan
          </div>
        </div>
      </div>
    </div>

    <!-- Redacted Secrets Table -->
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3 shadow-2xs">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-key-round" class="w-4 h-4 text-amber-500" />
          <span class="text-xs font-bold uppercase tracking-wider text-neutral-800 dark:text-neutral-200">
            Daftar Token & Kunci yang Dialias / Disensor Pada Komponen Ini
          </span>
        </div>
        <span class="text-xs text-neutral-500 font-mono">
          {{ scanResult.security_audit?.records?.length || 0 }} item terdaftar
        </span>
      </div>

      <div v-if="scanResult.security_audit?.records && scanResult.security_audit.records.length > 0" class="overflow-x-auto rounded-lg border border-neutral-200 dark:border-neutral-800">
        <table class="w-full text-left text-xs font-mono">
          <thead class="bg-neutral-50 dark:bg-neutral-800 text-neutral-500 uppercase text-xs tracking-wider border-b border-neutral-200 dark:border-neutral-800">
            <tr>
              <th class="p-2.5 w-1/4">
                Alias Terkirim ke LLM
              </th>
              <th class="p-2.5 w-1/4">
                Kategori Kunci
              </th>
              <th class="p-2.5 w-1/2">
                Masking Nilai Asli (Aman)
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800">
            <tr
              v-for="(rec, rIdx) in scanResult.security_audit.records"
              :key="rIdx"
              class="hover:bg-neutral-50/60 dark:hover:bg-neutral-800/40 transition-colors"
            >
              <td class="p-2.5 font-bold text-amber-600 dark:text-amber-400 whitespace-nowrap">
                <div class="flex items-center gap-1.5">
                  <span>{{ rec.alias }}</span>
                  <button
                    type="button"
                    class="text-neutral-400 hover:text-amber-600 transition-colors cursor-pointer"
                    title="Salin alias"
                    @click="copyToClipboard(rec.alias, 'Alias disalin!')"
                  >
                    <UIcon name="i-lucide-copy" class="w-3 h-3" />
                  </button>
                </div>
              </td>
              <td class="p-2.5 whitespace-nowrap">
                <UBadge size="xs" color="neutral" variant="subtle">
                  {{ rec.category }}
                </UBadge>
              </td>
              <td class="p-2.5 text-neutral-500 dark:text-neutral-400">
                <span class="bg-neutral-100 dark:bg-neutral-800 px-2 py-0.5 rounded text-xs">
                  {{ rec.masked_preview }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-else class="p-6 text-center text-xs text-neutral-500 italic bg-neutral-50/50 dark:bg-neutral-800/30 rounded-lg border border-dashed border-neutral-200 dark:border-neutral-800">
        Tidak ada kredensial atau string rahasia terdeteksi pada scan komponen ini (Aman dan bersih).
      </div>
    </div>
  </div>
</template>
