<script setup lang="ts">
import type { LLMConfig } from '~/types'

const open = defineModel<boolean>('open', { default: false })

defineProps<{
  config: LLMConfig | null
  deleting?: boolean
}>()

defineEmits<{
  confirm: []
}>()
</script>

<template>
  <UModal
    v-model:open="open"
    title="Hapus Konfigurasi AI"
    description="Apakah Anda yakin ingin menghapus konfigurasi ini dari grid?"
    :ui="{ content: 'sm:max-w-md' }"
  >
    <template #body>
      <div v-if="config" class="space-y-2 text-xs">
        <p class="text-neutral-600 dark:text-neutral-300">
          Konfigurasi berikut akan dihapus secara permanen dari database:
        </p>
        <div class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700 space-y-1">
          <div class="font-bold text-neutral-900 dark:text-white">
            {{ config.name }}
          </div>
          <div class="font-mono text-[11px] text-neutral-500 truncate">
            {{ config.url }}
          </div>
        </div>
        <p v-if="config.is_active" class="text-amber-600 dark:text-amber-400 text-[11px]">
          Catatan: Konfigurasi ini sedang berstatus aktif. Jika dihapus, konfigurasi lain yang tersedia akan otomatis diaktifkan.
        </p>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-end gap-2.5 w-full">
        <UButton color="neutral" variant="outline" @click="open = false">
          Batal
        </UButton>
        <UButton color="error" :loading="deleting" @click="$emit('confirm')">
          Hapus Konfigurasi
        </UButton>
      </div>
    </template>
  </UModal>
</template>
