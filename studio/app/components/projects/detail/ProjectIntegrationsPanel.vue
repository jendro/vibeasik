<script setup lang="ts">
import type { ProjectProfile } from '~/types'

defineProps<{
  profile: ProjectProfile
}>()

defineEmits<{
  switchTab: [tab: string]
}>()
</script>

<template>
  <section class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 p-4 sm:p-5 shadow-xs space-y-4">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
      <div>
        <h2 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-cable" class="w-4 h-4 text-sky-500" />
          Hubungan Antar Aplikasi
        </h2>
        <p class="text-xs text-neutral-500">
          Siapa memanggil siapa, layanan eksternal, dan data yang dibagi.
        </p>
      </div>
      <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-network" @click="$emit('switchTab', 'synergy')">
        Lihat Alur Data Detail
      </UButton>
    </div>

    <div v-if="profile.integrations.length" class="grid grid-cols-1 md:grid-cols-2 gap-2">
      <div
        v-for="(it, idx) in profile.integrations"
        :key="idx"
        class="rounded-lg border border-neutral-200/80 dark:border-neutral-800 bg-neutral-50/50 dark:bg-neutral-900/40 px-3 py-2.5 text-xs space-y-1"
      >
        <div class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white flex-wrap">
          <span>{{ it.from }}</span>
          <UIcon name="i-lucide-arrow-right" class="w-3.5 h-3.5 text-sky-500" />
          <span>{{ it.to }}</span>
          <span class="ml-auto px-1.5 py-0.5 rounded text-xs font-mono font-normal bg-white dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 text-neutral-500">{{ it.via }}</span>
        </div>
        <p v-if="it.purpose" class="text-xs text-neutral-500 leading-relaxed">
          {{ it.purpose }}
        </p>
      </div>
    </div>
    <p v-else class="text-xs text-neutral-400">
      Belum ada hubungan yang terdeteksi.
    </p>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 pt-1">
      <div class="space-y-1.5">
        <h3 class="text-xs font-bold uppercase tracking-wider text-neutral-400">
          Layanan Eksternal ({{ profile.external_services.length }})
        </h3>
        <p v-if="!profile.external_services.length" class="text-xs text-neutral-400">
          Tidak ada.
        </p>
        <div v-for="s in profile.external_services" :key="s.name" class="text-xs">
          <span class="font-semibold text-neutral-800 dark:text-neutral-200">{{ s.name }}</span>
          <span v-if="s.purpose" class="text-neutral-500"> — {{ s.purpose }}</span>
        </div>
      </div>
      <div class="space-y-1.5">
        <h3 class="text-xs font-bold uppercase tracking-wider text-neutral-400">
          Entitas Data ({{ profile.data_entities.length }})
        </h3>
        <p v-if="!profile.data_entities.length" class="text-xs text-neutral-400">
          Belum terdeteksi — bangun Peta Alur Data untuk melihat tabel.
        </p>
        <div v-for="d in profile.data_entities.slice(0, 12)" :key="d.name" class="text-xs">
          <code class="font-semibold text-neutral-800 dark:text-neutral-200">{{ d.name }}</code>
          <span v-if="d.description" class="text-neutral-500"> — {{ d.description }}</span>
          <span v-if="d.used_by.length" class="text-xs text-neutral-400"> ({{ d.used_by.join(', ') }})</span>
        </div>
      </div>
    </div>
  </section>
</template>
