<script setup lang="ts">
import { getTechCategories, type TechInfo, type TechCategory } from '~/composables/useTechLogo'

defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  (e: 'update:open', val: boolean): void
}>()

const categories = ref<TechCategory[]>(getTechCategories())
const activeCategory = ref<string>('all')
const searchQuery = ref<string>('')
const copiedIcon = ref<string | null>(null)

const allTechnologies = computed<TechInfo[]>(() => {
  return categories.value.flatMap(c => c.items)
})

const filteredTechnologies = computed<TechInfo[]>(() => {
  const q = searchQuery.value.trim().toLowerCase()
  let list = allTechnologies.value

  if (activeCategory.value !== 'all') {
    list = list.filter(item => item.category === activeCategory.value)
  }

  if (q) {
    list = list.filter(item =>
      item.name.toLowerCase().includes(q)
      || item.id.toLowerCase().includes(q)
      || item.icon.toLowerCase().includes(q)
      || item.description.toLowerCase().includes(q)
    )
  }

  return list
})

const copyIconTag = async (icon: string) => {
  try {
    await navigator.clipboard.writeText(icon)
    copiedIcon.value = icon
    setTimeout(() => {
      if (copiedIcon.value === icon) {
        copiedIcon.value = null
      }
    }, 1800)
  } catch (err) {
    console.error('Failed to copy icon name:', err)
  }
}
</script>

<template>
  <UModal
    :open="open"
    title="Katalog Logo Teknologi & Stack Blueprint"
    description="Eksplorasi logo resmi Simple-Icons untuk teknologi yang didukung di ekosistem Go, Nuxt, Flutter, dan Python."
    :ui="{ content: 'sm:max-w-4xl' }"
    @update:open="(val) => emit('update:open', val)"
  >
    <template #body>
      <div class="space-y-4">
        <!-- Search and Filter Bar -->
        <div class="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 pb-3 border-b border-neutral-200 dark:border-neutral-800">
          <div class="relative flex-1">
            <UInput
              v-model="searchQuery"
              icon="i-lucide-search"
              placeholder="Cari teknologi (mis. Go, Nuxt, PostgreSQL, Docker, JWT)..."
              size="sm"
              class="w-full"
            />
          </div>

          <div class="text-xs text-neutral-500 font-mono shrink-0">
            <span>Menampilkan </span>
            <span class="font-bold text-neutral-800 dark:text-neutral-200">{{ filteredTechnologies.length }}</span>
            <span> dari {{ allTechnologies.length }} stack</span>
          </div>
        </div>

        <!-- Category Pills -->
        <div class="flex items-center gap-1.5 overflow-x-auto pb-1 text-xs select-none">
          <button
            type="button"
            class="px-2.5 py-1 rounded-lg border text-xs font-medium cursor-pointer transition-colors shrink-0"
            :class="activeCategory === 'all'
              ? 'bg-sky-600 text-white border-sky-600'
              : 'bg-neutral-50 dark:bg-neutral-800 border-neutral-200 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-700'"
            @click="activeCategory = 'all'"
          >
            Semua ({{ allTechnologies.length }})
          </button>
          <button
            v-for="cat in categories"
            :key="cat.id"
            type="button"
            class="px-2.5 py-1 rounded-lg border text-xs font-medium cursor-pointer transition-colors shrink-0 flex items-center gap-1.5"
            :class="activeCategory === cat.id
              ? 'bg-sky-600 text-white border-sky-600'
              : 'bg-neutral-50 dark:bg-neutral-800 border-neutral-200 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-700'"
            @click="activeCategory = cat.id"
          >
            <UIcon :name="cat.icon" class="w-3.5 h-3.5" />
            <span>{{ cat.label }}</span>
            <span class="text-[10px] opacity-75 font-mono">({{ cat.items.length }})</span>
          </button>
        </div>

        <!-- Technology Grid -->
        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2.5 max-h-[500px] overflow-y-auto pr-1">
          <div
            v-for="item in filteredTechnologies"
            :key="item.id"
            class="group p-3 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/60 hover:border-neutral-300 dark:hover:border-neutral-700 hover:bg-neutral-50/50 dark:hover:bg-neutral-800/40 transition-all flex flex-col justify-between"
          >
            <div class="flex items-start justify-between gap-2.5">
              <div class="flex items-center gap-2.5 min-w-0">
                <div
                  class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0 border border-neutral-200/80 dark:border-neutral-700/60 bg-neutral-50 dark:bg-neutral-800 group-hover:scale-105 transition-transform"
                >
                  <UIcon
                    :name="item.icon"
                    class="w-5 h-5 transition-colors"
                    :style="item.color ? { color: item.color } : undefined"
                  />
                </div>
                <div class="min-w-0">
                  <div class="font-bold text-sm text-neutral-900 dark:text-white truncate">
                    {{ item.name }}
                  </div>
                  <div class="text-[10px] font-mono text-neutral-400 truncate">
                    {{ item.category }}
                  </div>
                </div>
              </div>

              <!-- Brand Color Swatch -->
              <div
                v-if="item.color"
                class="w-3.5 h-3.5 rounded-full border border-neutral-300 dark:border-neutral-700 shrink-0 shadow-2xs"
                :style="{ backgroundColor: item.color }"
                :title="`Brand Color: ${item.color}`"
              />
            </div>

            <!-- Description -->
            <p class="text-xs text-neutral-600 dark:text-neutral-400 mt-2 line-clamp-2 leading-relaxed">
              {{ item.description }}
            </p>

            <!-- Bottom Action: Icon Tag & Copy Button -->
            <div class="mt-3 pt-2 border-t border-neutral-100 dark:border-neutral-800 flex items-center justify-between text-xs">
              <code class="text-[10px] text-neutral-500 font-mono truncate max-w-[170px]" :title="item.icon">
                {{ item.icon }}
              </code>

              <button
                type="button"
                class="text-[11px] font-sans font-medium px-2 py-0.5 rounded cursor-pointer transition-colors"
                :class="copiedIcon === item.icon
                  ? 'bg-sky-100 dark:bg-sky-950/60 text-sky-700 dark:text-sky-300 font-bold'
                  : 'text-neutral-500 hover:text-sky-600 dark:hover:text-sky-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
                @click="copyIconTag(item.icon)"
              >
                {{ copiedIcon === item.icon ? 'Tersalin!' : 'Salin Tag' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </template>
    <template #footer>
      <div class="flex items-center justify-between w-full text-xs text-neutral-500">
        <span class="flex items-center gap-1.5">
          <UIcon name="i-lucide-palette" class="w-4 h-4 text-sky-500" />
          <span>Ikon bersumber dari <strong>Simple Icons</strong> & <strong>Lucide</strong></span>
        </span>
        <UButton
          color="neutral"
          variant="outline"
          size="sm"
          @click="emit('update:open', false)"
        >
          Tutup
        </UButton>
      </div>
    </template>
  </UModal>
</template>
