<script setup lang="ts">
import type { RuntimeSimComponent } from '~/types'

const props = defineProps<{
  components: RuntimeSimComponent[]
  running: boolean
  shotUrl?: (file?: string) => string
}>()

// Screenshot baru berupa nama berkas di engine; laporan lama masih berupa data URL.
const shotSrc = (file?: string) => props.shotUrl ? props.shotUrl(file) : (file || '')

const previewShot = ref<{ name: string, src: string } | null>(null)

const stateBadge = (state: string) => {
  switch (state) {
    case 'running':
      return { label: 'Berjalan', dot: 'bg-emerald-500 animate-pulse', cls: 'text-emerald-700 dark:text-emerald-300 bg-emerald-50 dark:bg-emerald-950/40 border-emerald-200 dark:border-emerald-800' }
    case 'booting':
      return { label: 'Menyala...', dot: 'bg-amber-500 animate-pulse', cls: 'text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-950/40 border-amber-200 dark:border-amber-800' }
    case 'stopped':
      return { label: 'Dimatikan', dot: 'bg-neutral-400', cls: 'text-neutral-600 dark:text-neutral-300 bg-neutral-100 dark:bg-neutral-800 border-neutral-200 dark:border-neutral-700' }
    case 'crashed':
      return { label: 'Crash', dot: 'bg-rose-500', cls: 'text-rose-700 dark:text-rose-300 bg-rose-50 dark:bg-rose-950/40 border-rose-200 dark:border-rose-800' }
    case 'timeout':
      return { label: 'Tidak Siap', dot: 'bg-rose-400', cls: 'text-rose-700 dark:text-rose-300 bg-rose-50 dark:bg-rose-950/40 border-rose-200 dark:border-rose-800' }
    case 'failed':
      return { label: 'Gagal', dot: 'bg-rose-500', cls: 'text-rose-700 dark:text-rose-300 bg-rose-50 dark:bg-rose-950/40 border-rose-200 dark:border-rose-800' }
    case 'skipped':
      return { label: 'Dilewati', dot: 'bg-neutral-300', cls: 'text-neutral-500 bg-neutral-50 dark:bg-neutral-900 border-neutral-200 dark:border-neutral-800' }
    default:
      return { label: 'Menunggu', dot: 'bg-neutral-300', cls: 'text-neutral-500 bg-neutral-50 dark:bg-neutral-900 border-neutral-200 dark:border-neutral-800' }
  }
}

const sourceLabel: Record<string, string> = {
  launched: 'dinyalakan simulasi',
  reused: 'sudah berjalan',
  external: 'proses luar'
}
</script>

<template>
  <div class="space-y-2.5">
    <div
      v-for="comp in components"
      :key="comp.component_id"
      class="p-3 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/70 space-y-2"
      :class="{ 'opacity-60': comp.state === 'skipped' }"
    >
      <div class="flex items-start justify-between gap-2">
        <div class="min-w-0">
          <div class="flex items-center gap-1.5 flex-wrap">
            <span class="text-sm font-bold text-neutral-900 dark:text-white truncate">{{ comp.name }}</span>
            <span class="text-[10px] font-mono text-neutral-400">{{ comp.role }}</span>
          </div>
          <code v-if="comp.command" class="block mt-0.5 text-[10px] font-mono text-neutral-500 truncate" :title="comp.command">$ {{ comp.command }}</code>
        </div>
        <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full border text-[10px] font-bold shrink-0" :class="stateBadge(comp.state).cls">
          <span class="w-1.5 h-1.5 rounded-full" :class="stateBadge(comp.state).dot" />
          {{ stateBadge(comp.state).label }}
        </span>
      </div>

      <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-neutral-600 dark:text-neutral-300">
        <a
          v-if="comp.url && comp.state === 'running'"
          :href="comp.url"
          target="_blank"
          rel="noopener"
          class="inline-flex items-center gap-1 font-mono font-semibold text-primary-600 dark:text-primary-400 hover:underline"
        >
          <UIcon name="i-lucide-external-link" class="w-3 h-3" />{{ comp.url.replace('http://', '') }}
        </a>
        <span v-else-if="comp.port" class="font-mono">:{{ comp.port }}</span>
        <span v-if="comp.pid" class="font-mono text-neutral-400">PID {{ comp.pid }}</span>
        <span v-if="comp.boot_ms" class="inline-flex items-center gap-1"><UIcon name="i-lucide-timer" class="w-3 h-3 text-neutral-400" />boot {{ (comp.boot_ms / 1000).toFixed(1) }}s</span>
        <span v-if="comp.memory_mb" class="inline-flex items-center gap-1"><UIcon name="i-lucide-memory-stick" class="w-3 h-3 text-neutral-400" />{{ Math.round(comp.memory_mb) }} MB</span>
        <span class="inline-flex items-center gap-1"><UIcon name="i-lucide-route" class="w-3 h-3 text-neutral-400" />{{ comp.routes_found }} rute</span>
        <span v-if="comp.error_lines" class="inline-flex items-center gap-1 text-rose-600 dark:text-rose-400 font-semibold"><UIcon name="i-lucide-bug" class="w-3 h-3" />{{ comp.error_lines }} error di log</span>
        <span v-if="comp.source" class="text-[10px] text-neutral-400">· {{ sourceLabel[comp.source] || comp.source }}</span>
      </div>

      <p v-if="comp.note" class="text-[11px] leading-snug text-neutral-500 dark:text-neutral-400">
        {{ comp.note }}
      </p>

      <button
        v-if="comp.screenshot"
        type="button"
        class="block w-full rounded-lg overflow-hidden border border-neutral-200 dark:border-neutral-700 cursor-zoom-in group"
        title="Klik untuk memperbesar tampilan aplikasi"
        @click="previewShot = { name: comp.name, src: shotSrc(comp.screenshot) }"
      >
        <div class="flex items-center gap-1 px-2 py-1 bg-neutral-100 dark:bg-neutral-800 border-b border-neutral-200 dark:border-neutral-700">
          <span class="w-2 h-2 rounded-full bg-rose-400" /><span class="w-2 h-2 rounded-full bg-amber-400" /><span class="w-2 h-2 rounded-full bg-emerald-400" />
          <span class="ml-2 text-[9px] font-mono text-neutral-500 truncate">{{ comp.url }}</span>
        </div>
        <img :src="shotSrc(comp.screenshot)" :alt="`Tampilan ${comp.name}`" class="w-full max-h-48 object-cover object-top group-hover:opacity-90 transition-opacity">
      </button>
      <div
        v-else-if="running && comp.state === 'running' && comp.role.startsWith('frontend')"
        class="h-16 rounded-lg border border-dashed border-neutral-300 dark:border-neutral-700 flex items-center justify-center gap-1.5 text-[10px] text-neutral-400"
      >
        <UIcon name="i-lucide-camera" class="w-3.5 h-3.5" />Screenshot diambil saat uji halaman
      </div>
    </div>

    <div
      v-if="previewShot"
      class="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-center justify-center p-6"
      @click.self="previewShot = null"
    >
      <div class="max-w-5xl w-full space-y-2">
        <div class="flex items-center justify-between text-white text-sm font-semibold">
          <span>Tampilan {{ previewShot.name }} saat dijalankan</span>
          <button type="button" class="p-1 rounded hover:bg-white/10 cursor-pointer" @click="previewShot = null">
            <UIcon name="i-lucide-x" class="w-5 h-5" />
          </button>
        </div>
        <img :src="previewShot.src" :alt="previewShot.name" class="w-full rounded-xl shadow-2xl">
      </div>
    </div>
  </div>
</template>
