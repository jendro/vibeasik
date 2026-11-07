<script setup lang="ts">
defineProps<{
  totalSelected: number
  totalComponents: number
  totalProjects: number
  importing: boolean
  validationError?: string
}>()

const emit = defineEmits<{
  confirm: []
  reset: []
}>()
</script>

<template>
  <div class="sticky bottom-4 z-30 p-4 rounded-xl border border-neutral-300 dark:border-white/20 bg-white/95 dark:bg-slate-900/95 backdrop-blur-md shadow-lg flex flex-col sm:flex-row items-center justify-between gap-3">
    <div class="space-y-0.5 text-center sm:text-left">
      <div class="flex items-center justify-center sm:justify-start gap-2">
        <span class="text-sm font-bold text-neutral-900 dark:text-white">
          {{ totalSelected }} Komponen Terpilih
        </span>
        <span class="text-xs text-neutral-400">
          (dari {{ totalComponents }} komponen dalam {{ totalProjects }} project)
        </span>
      </div>

      <div v-if="validationError" class="text-xs font-medium text-rose-600 dark:text-rose-400 flex items-center justify-center sm:justify-start gap-1">
        <UIcon name="i-lucide-alert-triangle" class="w-3.5 h-3.5 shrink-0" />
        <span>{{ validationError }}</span>
      </div>
      <div v-else class="text-[11px] text-neutral-500 dark:text-neutral-400">
        Siap didaftarkan ke dashboard pemantauan & manajemen port lokal.
      </div>
    </div>

    <div class="flex items-center gap-2.5 w-full sm:w-auto">
      <UButton
        color="neutral"
        variant="ghost"
        size="md"
        class="flex-1 sm:flex-none justify-center"
        :disabled="importing"
        @click="emit('reset')"
      >
        Pindai Ulang
      </UButton>

      <UButton
        icon="i-lucide-check-circle"
        color="primary"
        size="md"
        class="flex-1 sm:flex-none justify-center"
        :loading="importing"
        :disabled="totalSelected === 0 || !!validationError || importing"
        title="Daftarkan seluruh komponen dan project yang telah dipilih ke pemantauan"
        @click="emit('confirm')"
      >
        Konfirmasi & Daftarkan ke Pemantauan
      </UButton>
    </div>
  </div>
</template>
