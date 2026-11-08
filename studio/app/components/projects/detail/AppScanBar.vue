<script setup lang="ts">
import type { FastScanResult } from '~/types'

const props = defineProps<{
  scanResult: FastScanResult | null
  scanning?: boolean
  isFrontend?: boolean
}>()

defineEmits<{
  scan: []
  openResult: []
}>()

const showInfo = ref(!props.scanResult)

const scannedLabel = computed(() => props.scanResult?.scanned_at
  ? new Date(props.scanResult.scanned_at).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' })
  : '')

// Mirrors what FastScanComponent in local-engine actually reads, produces and changes.
const sections = computed(() => [
  {
    title: 'Yang dibaca',
    icon: 'i-lucide-file-search',
    items: [
      'Struktur folder 3 level (tanpa node_modules, vendor, dist, build, .git)',
      'Manifest & config: package.json, go.mod, pubspec.yaml, composer.json, nuxt/next/vite config, Dockerfile, docker-compose',
      '.env / .env.example, file database & router, README',
      'Hingga 8 file per folder controllers, handlers, routes, api, composables'
    ],
    note: 'Kunci rahasia disensor dulu sebelum dikirim ke LLM.'
  },
  {
    title: 'Yang dihasilkan',
    icon: 'i-lucide-list-checks',
    items: [
      'Arsitektur, plugin & pola desain',
      'Cara menjalankan: install, dev, build, test',
      'Database, ORM, metode login & daftar env key',
      props.isFrontend ? 'Daftar halaman & request API yang dipanggil' : 'Daftar endpoint API beserta handler & middleware',
      'Analisis lengkap + AI Context Pack'
    ],
    note: 'Rute digabung dengan deteksi statis agar tidak ada yang terlewat.'
  },
  {
    title: 'Yang berubah',
    icon: 'i-lucide-refresh-cw',
    items: [
      'Hasil pindai lama diganti, termasuk metadata yang Anda edit manual',
      `Katalog ${props.isFrontend ? 'halaman' : 'rute'} di bawah ikut disinkronkan`,
      'Bahasa, framework, dev command & port hanya diisi jika masih kosong'
    ],
    note: 'Untuk menimpa nilai yang sudah ada, pakai "Terapkan Parameter ke Profil" di Hasil Lengkap. Tanpa LLM aktif, hasil memakai deteksi statis.'
  }
])
</script>

<template>
  <div class="rounded-xl bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 shadow-2xs text-xs">
    <div class="flex flex-wrap items-center justify-between gap-3 p-3.5">
      <div class="flex items-center gap-2.5 min-w-0 flex-1 basis-60">
        <div class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0" :class="scanResult ? 'bg-sky-50 dark:bg-sky-950/60 text-sky-600 dark:text-sky-400' : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-400'">
          <UIcon name="i-lucide-scan-text" class="w-4 h-4" />
        </div>
        <div class="min-w-0">
          <div class="font-semibold text-neutral-900 dark:text-white">
            Pindai LLM
          </div>
          <div class="text-neutral-500">
            {{ scanResult ? `Terakhir dipindai ${scannedLabel}` : 'Belum pernah dipindai — LLM membaca kode aplikasi ini untuk mengisi data di bawah.' }}
          </div>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <UButton
          color="neutral"
          variant="ghost"
          size="xs"
          :icon="showInfo ? 'i-lucide-chevron-up' : 'i-lucide-circle-help'"
          @click="showInfo = !showInfo"
        >
          Apa yang dipindai?
        </UButton>
        <UButton
          v-if="scanResult"
          icon="i-lucide-panel-right-open"
          color="neutral"
          variant="outline"
          size="xs"
          title="Buka semua hasil pindai: arsitektur, cara menjalankan, env, rute, struktur folder, AI Context Pack"
          @click="$emit('openResult')"
        >
          Lihat Hasil Lengkap
        </UButton>
        <UButton
          icon="i-lucide-scan"
          color="primary"
          :variant="scanResult ? 'soft' : 'solid'"
          size="xs"
          :loading="scanning"
          @click="$emit('scan')"
        >
          {{ scanning ? 'Memindai…' : scanResult ? 'Pindai Ulang' : 'Pindai Sekarang' }}
        </UButton>
      </div>
    </div>

    <div v-if="showInfo" class="grid grid-cols-1 xl:grid-cols-3 gap-3 px-3.5 pb-3.5">
      <div
        v-for="s in sections"
        :key="s.title"
        class="rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-100 dark:border-neutral-800 p-3 space-y-1.5"
      >
        <div class="font-semibold text-neutral-800 dark:text-neutral-100 flex items-center gap-1.5">
          <UIcon :name="s.icon" class="w-3.5 h-3.5 text-sky-500" />{{ s.title }}
        </div>
        <ul class="list-disc list-inside space-y-0.5 text-neutral-600 dark:text-neutral-300">
          <li v-for="item in s.items" :key="item">
            {{ item }}
          </li>
        </ul>
        <p class="text-xs text-neutral-400">
          {{ s.note }}
        </p>
      </div>
    </div>
  </div>
</template>
