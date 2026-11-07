<script setup lang="ts">
import { resolveTech, type TechInfo } from '~/composables/useTechLogo'

const props = withDefaults(
  defineProps<{
    tech?: string | null
    size?: 'xs' | 'sm' | 'md'
    variant?: 'subtle' | 'outline' | 'ghost'
    showIcon?: boolean
    showLabel?: boolean
    customLabel?: string
    colored?: boolean
  }>(),
  {
    tech: '',
    size: 'xs',
    variant: 'subtle',
    showIcon: true,
    showLabel: true,
    colored: true
  }
)

const techInfo = computed<TechInfo>(() => resolveTech(props.tech))
const displayText = computed(() => props.customLabel || techInfo.value.name)

const sizeClasses = computed(() => {
  switch (props.size) {
    case 'md':
      return {
        badge: 'px-2.5 py-1 text-xs gap-1.5',
        icon: 'w-4 h-4'
      }
    case 'sm':
      return {
        badge: 'px-2 py-0.5 text-xs gap-1.5',
        icon: 'w-3.5 h-3.5'
      }
    case 'xs':
    default:
      return {
        badge: 'px-1.5 py-0.5 text-[11px] gap-1',
        icon: 'w-3 h-3'
      }
  }
})

const variantClasses = computed(() => {
  switch (props.variant) {
    case 'outline':
      return 'border border-neutral-200 dark:border-neutral-700 bg-transparent text-neutral-800 dark:text-neutral-200 hover:border-neutral-300 dark:hover:border-neutral-600'
    case 'ghost':
      return 'bg-transparent text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800'
    case 'subtle':
    default:
      return 'border border-neutral-200/80 dark:border-neutral-700/60 bg-neutral-100/90 dark:bg-neutral-800/80 text-neutral-800 dark:text-neutral-200'
  }
})
</script>

<template>
  <span
    class="inline-flex items-center font-mono font-medium rounded-md transition-colors select-none shrink-0"
    :class="[sizeClasses.badge, variantClasses]"
    :title="techInfo.description"
  >
    <UIcon
      v-if="showIcon"
      :name="techInfo.icon"
      :class="[sizeClasses.icon, 'shrink-0 transition-transform duration-150']"
      :style="colored && techInfo.color ? { color: techInfo.color } : undefined"
    />
    <span v-if="showLabel" class="truncate font-sans font-medium tracking-tight">
      {{ displayText }}
    </span>
  </span>
</template>
