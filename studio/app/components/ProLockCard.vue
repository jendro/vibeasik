<script setup lang="ts">
// Kartu pengganti bagian layar yang membutuhkan fitur berbayar.
const props = defineProps<{ feature: string, title?: string, description?: string }>()
const { upgradeUrl } = useEdition()
const info = computed(() => PRO_FEATURES[props.feature])
</script>

<template>
  <div class="rounded-xl border border-dashed border-amber-300 dark:border-amber-700/60 bg-amber-50/60 dark:bg-amber-950/20 p-4 flex items-start gap-3">
    <UIcon name="i-lucide-lock" class="w-5 h-5 shrink-0 text-amber-600 dark:text-amber-400 mt-0.5" />
    <div class="min-w-0 space-y-1">
      <div class="text-sm font-semibold text-neutral-800 dark:text-neutral-100">
        {{ title || info?.label || 'Fitur Pro' }}
        <UBadge size="xs" color="warning" variant="subtle" class="ml-1 align-middle">
          Pro
        </UBadge>
      </div>
      <p class="text-xs text-neutral-600 dark:text-neutral-400">
        {{ description || info?.description }}
      </p>
      <div class="flex gap-2 pt-1">
        <UButton size="xs" color="warning" :to="{ path: '/upgrade', query: { feature } }" icon="i-lucide-sparkles">
          Lihat paket
        </UButton>
        <UButton size="xs" variant="ghost" color="neutral" :to="upgradeUrl" target="_blank" trailing-icon="i-lucide-external-link">
          vibeasik.com
        </UButton>
      </div>
    </div>
  </div>
</template>
