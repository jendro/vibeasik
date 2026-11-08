<script setup lang="ts">
const props = defineProps<{
  scanning: boolean
  errorMsg?: string
  quickPaths?: { label: string, path: string }[]
}>()

const targetPath = defineModel<string>('path', { default: '' })

const emit = defineEmits<{
  scan: []
  selectQuickPath: [path: string]
}>()

const handleEnter = () => {
  if (targetPath.value.trim() && !props.scanning) {
    emit('scan')
  }
}
</script>

<template>
  <div class="p-6 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs space-y-4">
    <div class="space-y-2">
      <label class="block text-xs font-semibold uppercase tracking-wider text-neutral-500">
        Direktori Filesystem Laptop
      </label>
      <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
        <UInput
          v-model="targetPath"
          icon="i-lucide-folder"
          placeholder="/home/username/path/to/project"
          size="lg"
          class="flex-1 font-mono text-xs"
          @keyup.enter="handleEnter"
        />
        <UButton
          icon="i-lucide-search"
          color="primary"
          size="lg"
          :loading="scanning"
          title="Pindai struktur folder untuk mendeteksi modul, komponen, dan port"
          @click="emit('scan')"
        >
          Pindai Folder
        </UButton>
      </div>
    </div>

    <!-- Quick Path Shortcuts -->
    <div v-if="quickPaths && quickPaths.length > 0" class="flex flex-wrap items-center gap-2 pt-1 text-xs">
      <span class="text-neutral-400">Pintasan cepat:</span>
      <button
        v-for="qp in quickPaths"
        :key="qp.path"
        type="button"
        class="px-2.5 py-1 rounded-md bg-neutral-100 dark:bg-neutral-800 hover:bg-neutral-200 dark:hover:bg-neutral-700 text-neutral-700 dark:text-neutral-300 font-mono text-xs cursor-pointer transition-colors"
        @click="emit('selectQuickPath', qp.path)"
      >
        {{ qp.label }}
      </button>
    </div>

    <!-- Error Alert -->
    <div v-if="errorMsg" class="p-3 rounded-lg border border-rose-300 dark:border-rose-900/60 bg-rose-50 dark:bg-rose-950/20 text-xs text-rose-700 dark:text-rose-400 flex items-center gap-2">
      <UIcon name="i-lucide-alert-circle" class="w-4 h-4 shrink-0" />
      <span>{{ errorMsg }}</span>
    </div>
  </div>
</template>
