<script setup lang="ts">
import type { BrowseDirectoryResponse } from '~/types'

const props = defineProps<{
  browsingPath: string
  browseData: BrowseDirectoryResponse | null
  loading?: boolean
}>()

const emit = defineEmits<{
  navigate: [path: string]
  select: [path: string, folderName?: string]
}>()

const searchQuery = ref('')

let searchTimer: any = null
watch(searchQuery, () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    emit('navigate', props.browsingPath)
  }, 200)
})
</script>

<template>
  <div class="border border-neutral-200 dark:border-white/[0.08] rounded-lg bg-neutral-50 dark:bg-neutral-950 overflow-hidden">
    <!-- Explorer Toolbar -->
    <div class="p-2.5 bg-neutral-100/90 dark:bg-[#04111f] border-b border-neutral-200 dark:border-white/[0.08] flex items-center justify-between gap-2 text-xs">
      <div class="flex items-center gap-1.5 min-w-0 font-mono text-xs text-neutral-700 dark:text-neutral-300">
        <UIcon name="i-lucide-folder" class="w-4 h-4 text-neutral-500 dark:text-neutral-400 shrink-0" />
        <span class="truncate" :title="browsingPath">{{ browsingPath }}</span>
      </div>

      <div class="flex items-center gap-1.5 shrink-0">
        <UButton
          v-if="browseData?.parent_path && browseData.parent_path !== browsingPath"
          icon="i-lucide-corner-left-up"
          color="neutral"
          variant="ghost"
          size="xs"
          @click="$emit('navigate', browseData.parent_path)"
        >
          Naik
        </UButton>
        <UButton
          color="primary"
          variant="solid"
          size="xs"
          :disabled="!browseData?.current_path"
          @click="$emit('select', browseData?.current_path || browsingPath)"
        >
          Pilih Folder Ini
        </UButton>
      </div>
    </div>

    <!-- Search within folder -->
    <div class="p-2.5 border-b border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-neutral-900/60">
      <UInput
        v-model="searchQuery"
        icon="i-lucide-search"
        placeholder="Cari nama folder..."
        size="xs"
        class="w-full"
      />
    </div>

    <!-- Folder Listing -->
    <div class="max-h-48 overflow-y-auto divide-y divide-neutral-200 dark:divide-neutral-800 text-xs">
      <div v-if="loading" class="p-4 text-center text-neutral-500 dark:text-neutral-400 text-xs">
        Memuat direktori...
      </div>

      <div
        v-else-if="!browseData?.directories || browseData.directories.length === 0"
        class="p-4 text-center text-neutral-500 dark:text-neutral-400 text-xs"
      >
        Tidak ada subfolder di dalam direktori ini.
      </div>

      <div
        v-for="dir in browseData?.directories"
        :key="dir.path"
        class="group flex items-center justify-between px-3 py-2 hover:bg-neutral-100 dark:hover:bg-neutral-900 transition-colors"
      >
        <div
          class="flex items-center gap-2 min-w-0 flex-1 cursor-pointer"
          @click="$emit('select', dir.path, dir.name)"
        >
          <UIcon name="i-lucide-folder" class="w-4 h-4 text-neutral-500 dark:text-neutral-400 group-hover:text-sky-500 shrink-0" />
          <span class="font-medium truncate text-neutral-700 dark:text-neutral-200 group-hover:text-neutral-900 dark:group-hover:text-white">{{ dir.name }}</span>
        </div>

        <div class="flex items-center gap-1 shrink-0 ml-2">
          <UButton
            size="xs"
            color="success"
            variant="ghost"
            @click="$emit('select', dir.path, dir.name)"
          >
            Pilih
          </UButton>
          <UButton
            icon="i-lucide-chevron-right"
            color="neutral"
            variant="ghost"
            size="xs"
            title="Buka subfolder"
            @click="$emit('navigate', dir.path)"
          />
        </div>
      </div>
    </div>
  </div>
</template>
