<script setup lang="ts">
// Kepala halaman Studio yang seragam: judul, keterangan singkat, info tambahan (meta) dan aksi di kanan.
// Proyek aktif sudah tampil di breadcrumb navbar, jadi tidak diulang di sini.
defineProps<{
  title: string
  description?: string
  icon?: string
}>()
</script>

<template>
  <header class="flex flex-col lg:flex-row lg:items-end justify-between gap-4 pb-5 border-b border-neutral-200 dark:border-white/[0.07]">
    <div class="min-w-0 space-y-1.5">
      <h2 class="flex items-center gap-2.5 text-xl sm:text-2xl font-semibold tracking-tight text-neutral-950 dark:text-white">
        <UIcon v-if="icon" :name="icon" class="size-5 sm:size-6 shrink-0 text-neutral-400 dark:text-slate-500" />
        <span class="truncate">{{ title }}</span>
        <slot name="badge" />
      </h2>
      <p v-if="description" class="text-sm text-neutral-600 dark:text-slate-400 max-w-3xl">
        {{ description }}
      </p>
      <div v-if="$slots.meta" class="flex flex-wrap items-center gap-x-4 gap-y-1 pt-1 text-xs text-neutral-500 dark:text-slate-400">
        <slot name="meta" />
      </div>
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2 shrink-0">
      <slot name="actions" />
    </div>
  </header>
</template>
