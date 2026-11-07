<script setup lang="ts">
defineProps<{
  activeProvider: 'system_one' | 'standard_llm'
  saving?: boolean
  bannerMessage?: string | null
  bannerType?: 'success' | 'info' | 'error'
}>()

defineEmits<{
  selectProvider: [provider: 'system_one' | 'standard_llm']
}>()
</script>

<template>
  <div class="space-y-4">
    <!-- Banner notification -->
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0 -translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-2"
    >
      <div
        v-if="bannerMessage"
        class="p-3.5 rounded-xl border text-xs flex items-center justify-between gap-3 shadow-xs"
        :class="bannerType === 'error'
          ? 'bg-rose-50 dark:bg-rose-950/40 border-rose-200 dark:border-rose-900/60 text-rose-800 dark:text-rose-200'
          : bannerType === 'info'
            ? 'bg-blue-50 dark:bg-blue-950/40 border-blue-200 dark:border-blue-900/60 text-blue-800 dark:text-blue-200'
            : 'bg-sky-50 dark:bg-sky-950/40 border-sky-200 dark:border-sky-900/60 text-sky-800 dark:text-sky-200'"
      >
        <div class="flex items-center gap-2">
          <UIcon
            :name="bannerType === 'error' ? 'i-lucide-alert-triangle' : bannerType === 'info' ? 'i-lucide-info' : 'i-lucide-check-circle'"
            class="w-4 h-4 shrink-0"
          />
          <span>{{ bannerMessage }}</span>
        </div>
      </div>
    </Transition>

    <!-- Header & Provider Switcher Bar -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-4 border-b border-neutral-200 dark:border-white/10">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white flex items-center gap-2.5">
          <UIcon name="i-lucide-cpu" class="w-7 h-7 text-primary-500" />
          Multi-Engine AI & Security Hub
        </h1>
        <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-1">
          Konfigurasi koneksi LLM, endpoint System One (Jev / Laya), port aktif, dan aturan penyensoran kata kunci rahasia.
        </p>
      </div>

      <!-- Provider Switcher -->
      <div class="flex items-center gap-1.5 p-1 rounded-xl bg-neutral-100 dark:bg-neutral-800/80 border border-neutral-200 dark:border-neutral-700/60">
        <button
          type="button"
          class="flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
          :class="activeProvider === 'system_one'
            ? 'bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white shadow-xs border border-neutral-200/80 dark:border-neutral-700'
            : 'text-neutral-500 hover:text-neutral-900 dark:hover:text-white'"
          :disabled="saving"
          @click="$emit('selectProvider', 'system_one')"
        >
          <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 text-primary-500" />
          <span>System One (Jev)</span>
          <UBadge v-if="activeProvider === 'system_one'" size="xs" color="primary" variant="subtle">
            Aktif
          </UBadge>
        </button>

        <button
          type="button"
          class="flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
          :class="activeProvider === 'standard_llm'
            ? 'bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white shadow-xs border border-neutral-200/80 dark:border-neutral-700'
            : 'text-neutral-500 hover:text-neutral-900 dark:hover:text-white'"
          :disabled="saving"
          @click="$emit('selectProvider', 'standard_llm')"
        >
          <UIcon name="i-lucide-bot" class="w-3.5 h-3.5 text-blue-500" />
          <span>Standard LLM</span>
          <UBadge v-if="activeProvider === 'standard_llm'" size="xs" color="info" variant="subtle">
            Aktif
          </UBadge>
        </button>
      </div>
    </div>
  </div>
</template>
