<script setup lang="ts">
import type { CensoredKeyword } from '~/types'

const open = defineModel<boolean>('open', { default: false })

defineProps<{
  keyword: CensoredKeyword | null
  deleting?: boolean
}>()

defineEmits<{
  confirm: []
}>()
</script>

<template>
  <UModal
    v-model:open="open"
    title="Hapus Keyword Sensor"
    description="Apakah Anda yakin ingin menghapus keyword sensor ini dari vault?"
    :ui="{ content: 'sm:max-w-md' }"
  >
    <template #body>
      <div v-if="keyword" class="space-y-2 text-xs">
        <p class="text-neutral-600 dark:text-neutral-300">
          Keyword sensor berikut akan dihapus secara permanen dari database vault:
        </p>
        <div class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700 space-y-1">
          <div class="font-mono font-bold text-neutral-900 dark:text-white">
            {{ keyword.keyword }}
          </div>
          <div class="text-[11px] text-neutral-500">
            Kategori: {{ keyword.category }}
          </div>
          <div v-if="keyword.description" class="text-[11px] text-neutral-400">
            {{ keyword.description }}
          </div>
        </div>
        <p class="text-amber-600 dark:text-amber-400 text-[11px]">
          Setelah dihapus, keyword ini tidak lagi disensor secara khusus oleh Pass 1 (namun pola bawaan sistem Pass 2 tetap melindungi token standar).
        </p>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-end gap-2.5 w-full">
        <UButton color="neutral" variant="outline" @click="open = false">
          Batal
        </UButton>
        <UButton color="error" :loading="deleting" @click="$emit('confirm')">
          Hapus Keyword
        </UButton>
      </div>
    </template>
  </UModal>
</template>
