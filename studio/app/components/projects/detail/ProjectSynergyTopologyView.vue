<script setup lang="ts">
import type { ProjectSynergyMap } from '~/types'

defineProps<{ synergyMap: ProjectSynergyMap | null }>()
</script>

<template>
  <div class="space-y-5">
    <!-- Backend Nodes Summary -->
    <div class="p-5 rounded-xl border border-neutral-200/80 dark:border-white/[0.08] bg-white dark:bg-[#04111f] space-y-3">
      <h3 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
        <UIcon name="i-lucide-server" class="w-4 h-4 text-purple-500" />
        <span>Servis & Node Komponen Backend</span>
      </h3>
      <p class="text-xs text-neutral-500">
        Daftar server backend yang melayani request API dan berinteraksi langsung dengan basis data.
      </p>

      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 pt-1">
        <div
          v-for="be in synergyMap?.backend_nodes || []"
          :key="be.component_name"
          class="p-4 rounded-xl bg-neutral-50/70 dark:bg-neutral-800/40 border border-neutral-200/70 dark:border-white/[0.08] space-y-2"
        >
          <div class="flex items-center justify-between">
            <span class="font-bold text-xs text-neutral-900 dark:text-white">{{ be.component_name }}</span>
            <UBadge size="xs" color="primary" variant="subtle" class="font-mono text-xs">
              Port {{ be.port }}
            </UBadge>
          </div>
          <div class="text-xs text-neutral-500 flex items-center justify-between">
            <span>Framework:</span>
            <span class="font-semibold text-neutral-800 dark:text-neutral-200 font-mono">{{ be.framework || 'Go / Fiber' }}</span>
          </div>
          <div class="text-xs text-neutral-500 flex items-center justify-between">
            <span>Total Endpoint:</span>
            <span class="font-bold text-sky-600 dark:text-sky-400 font-mono">{{ be.routes_count }} rute</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Database Map Grid -->
    <div class="p-5 rounded-xl border border-neutral-200/80 dark:border-white/[0.08] bg-white dark:bg-[#04111f] space-y-3">
      <h3 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
        <UIcon name="i-lucide-database" class="w-4 h-4 text-amber-500" />
        <span>Katalog Tabel Basis Data & Kolom Kunci</span>
      </h3>
      <p class="text-xs text-neutral-500">
        Entitas database yang menyimpan data hasil proses transaksi API dari seluruh aplikasi bundle.
      </p>

      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 pt-1">
        <div
          v-for="tbl in synergyMap?.database_map || []"
          :key="tbl.table_name"
          class="p-4 rounded-xl bg-neutral-50/70 dark:bg-neutral-800/40 border border-neutral-200/70 dark:border-white/[0.08] space-y-2"
        >
          <div class="flex items-center justify-between pb-1 border-b border-neutral-200/60 dark:border-neutral-700/60">
            <span class="font-mono font-bold text-xs text-amber-700 dark:text-amber-400">
              {{ tbl.table_name }}
            </span>
            <span class="text-xs text-neutral-400 font-mono">
              {{ tbl.fields?.length || 0 }} field
            </span>
          </div>
          <div class="space-y-1">
            <span class="text-xs uppercase font-semibold text-neutral-400">Field Tersimpan:</span>
            <div class="flex flex-wrap gap-1">
              <span
                v-for="f in tbl.fields"
                :key="f"
                class="px-1.5 py-0.5 rounded bg-white dark:bg-[#04111f] border border-neutral-200/80 dark:border-neutral-700 font-mono text-xs text-neutral-700 dark:text-neutral-300"
              >
                {{ f }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
