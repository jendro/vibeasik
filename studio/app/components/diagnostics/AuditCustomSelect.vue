<script setup lang="ts">
export interface SelectOption {
  value: string | number
  label: string
  icon?: string
  color?: string
  badge?: string | number
}

const props = withDefaults(
  defineProps<{
    modelValue: string | number
    options: SelectOption[]
    placeholder?: string
    buttonClass?: string
  }>(),
  {
    placeholder: 'Pilih opsi...',
    buttonClass: 'w-36 sm:w-40'
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', val: string | number): void
  (e: 'change', val: string | number): void
}>()

const isOpen = ref(false)
const containerRef = ref<HTMLElement | null>(null)

const selectedOption = computed(() => {
  return props.options.find(opt => String(opt.value) === String(props.modelValue))
})

const toggleDropdown = () => {
  isOpen.value = !isOpen.value
}

const closeDropdown = () => {
  isOpen.value = false
}

const selectOption = (val: string | number) => {
  emit('update:modelValue', val)
  emit('change', val)
  closeDropdown()
}

const handleClickOutside = (event: MouseEvent) => {
  if (containerRef.value && !containerRef.value.contains(event.target as Node)) {
    closeDropdown()
  }
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    closeDropdown()
  } else if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    toggleDropdown()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div ref="containerRef" class="relative inline-block text-left select-none shrink-0">
    <!-- Trigger Button: Lebar tetap, teks boleh terpotong (truncate) -->
    <button
      type="button"
      class="inline-flex items-center justify-between gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 text-neutral-800 dark:text-neutral-200 hover:border-neutral-300 dark:hover:border-neutral-700 hover:bg-neutral-50 dark:hover:bg-neutral-800/60 focus:outline-hidden focus:ring-2 focus:ring-primary-500/30 transition-all cursor-pointer shadow-2xs"
      :class="[buttonClass, isOpen ? 'border-primary-500/60 ring-2 ring-primary-500/20' : '']"
      :aria-expanded="isOpen"
      @click="toggleDropdown"
      @keydown="handleKeyDown"
    >
      <div class="flex items-center gap-1.5 min-w-0 flex-1">
        <UIcon
          v-if="selectedOption?.icon"
          :name="selectedOption.icon"
          class="w-3.5 h-3.5 shrink-0"
          :class="selectedOption.color || 'text-neutral-500 dark:text-neutral-400'"
        />
        <span class="truncate text-left font-medium">
          {{ selectedOption?.label || placeholder }}
        </span>
      </div>

      <UIcon
        name="i-lucide-chevron-down"
        class="w-3.5 h-3.5 shrink-0 text-neutral-400 group-hover:text-neutral-600 dark:group-hover:text-neutral-300 transition-transform duration-200 ml-1"
        :class="isOpen ? 'rotate-180' : ''"
      />
    </button>

    <!-- Dropdown Options Menu: min-w-max agar teks pilihan terlihat lengkap tanpa terpotong -->
    <Transition
      enter-active-class="transition duration-100 ease-out"
      enter-from-class="transform scale-95 opacity-0"
      enter-to-class="transform scale-100 opacity-100"
      leave-active-class="transition duration-75 ease-in"
      leave-from-class="transform scale-100 opacity-100"
      leave-to-class="transform scale-95 opacity-0"
    >
      <div
        v-if="isOpen"
        class="absolute right-0 top-full mt-1.5 z-50 min-w-max w-max max-w-xs sm:max-w-sm rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 shadow-xl p-1.5 space-y-0.5 text-xs focus:outline-hidden whitespace-nowrap"
      >
        <button
          v-for="opt in options"
          :key="String(opt.value)"
          type="button"
          class="w-full flex items-center justify-between gap-3 px-2.5 py-1.5 rounded-lg text-left transition-colors cursor-pointer group"
          :class="String(opt.value) === String(modelValue)
            ? 'bg-neutral-100 dark:bg-neutral-800 text-neutral-900 dark:text-white font-semibold'
            : 'text-neutral-700 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-800/60'"
          @click="selectOption(opt.value)"
        >
          <div class="flex items-center gap-2 min-w-0">
            <UIcon
              v-if="opt.icon"
              :name="opt.icon"
              class="w-3.5 h-3.5 shrink-0"
              :class="opt.color || (String(opt.value) === String(modelValue) ? 'text-primary-500' : 'text-neutral-400 dark:text-neutral-500')"
            />
            <span class="whitespace-nowrap">{{ opt.label }}</span>
          </div>

          <UIcon
            v-if="String(opt.value) === String(modelValue)"
            name="i-lucide-check"
            class="w-3.5 h-3.5 text-primary-500 shrink-0 ml-2"
          />
        </button>
      </div>
    </Transition>
  </div>
</template>
