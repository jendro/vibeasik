<script setup lang="ts">
const open = defineModel<boolean>('open', { default: false })

withDefaults(defineProps<{
  title: string
  description?: string
  confirmLabel?: string
  loading?: boolean
}>(), {
  description: '',
  confirmLabel: 'Hapus'
})

defineEmits<{
  confirm: []
}>()
</script>

<template>
  <UModal
    v-model:open="open"
    :title="title"
    :description="description"
    :ui="{ content: 'sm:max-w-md' }"
  >
    <template v-if="$slots.default" #body>
      <div class="text-xs text-neutral-600 dark:text-neutral-300 space-y-2">
        <slot />
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-end gap-2.5 w-full">
        <UButton color="neutral" variant="outline" @click="open = false">
          Batal
        </UButton>
        <UButton color="error" :loading="loading" @click="$emit('confirm')">
          {{ confirmLabel }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
