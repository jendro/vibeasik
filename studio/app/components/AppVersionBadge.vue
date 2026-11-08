<script setup lang="ts">
// Versi vibeasik di sidebar (di bawah judul). Bila ada versi baru muncul tanda; klik sekali = unduh,
// pasang, lalu mulai ulang otomatis. Pemeriksaan dilakukan engine saat start dan berkala.
const { info, stage, ready, available, load, updateNow } = useAppUpdate()
onMounted(() => load())

const isDev = computed(() => !!info.value && (info.value.current.includes('-') || !/^v\d/.test(info.value.current)))
const label = computed(() => isDev.value ? 'dev' : info.value?.current || '')
const busy = computed(() => stage.value !== 'idle')
const canClick = computed(() => available.value && (!!info.value?.can_apply || ready.value))
</script>

<template>
  <div v-if="info" class="flex items-center gap-1.5 min-w-0">
    <span class="text-[11px] font-mono text-neutral-500 dark:text-slate-400 truncate" :title="`Versi terpasang: ${info.current}`">
      {{ label }}
    </span>

    <button
      v-if="available && canClick"
      type="button"
      class="inline-flex items-center gap-1 rounded-full bg-sky-600 hover:bg-sky-700 text-white px-1.5 py-px text-[10px] font-semibold transition-colors disabled:opacity-80 disabled:cursor-wait focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-400"
      :disabled="busy"
      :title="ready ? 'Sudah terunduh. Klik untuk mulai ulang dan memakainya' : `Klik untuk mengunduh dan memasang ${info.latest}`"
      @click.prevent.stop="updateNow"
    >
      <UIcon v-if="busy" name="i-lucide-loader-circle" class="size-3 animate-spin" />
      <span v-else class="relative flex size-1.5">
        <span class="absolute inline-flex size-full rounded-full bg-white opacity-75 animate-ping" />
        <span class="relative inline-flex size-1.5 rounded-full bg-white" />
      </span>
      <template v-if="stage === 'downloading'">
        Mengunduh…
      </template>
      <template v-else-if="stage === 'restarting'">
        Memulai ulang…
      </template>
      <template v-else>
        {{ ready ? 'Mulai ulang' : `Baru ${info.latest}` }}
      </template>
    </button>

    <a
      v-else-if="available && info.url"
      :href="info.url"
      target="_blank"
      rel="noopener"
      class="inline-flex items-center gap-1 rounded-full bg-sky-600 hover:bg-sky-700 text-white px-1.5 py-px text-[10px] font-semibold"
      title="Versi baru tersedia: buka halaman unduhan"
    >
      <span class="size-1.5 rounded-full bg-white" /> Baru {{ info.latest }}
    </a>
  </div>
</template>
