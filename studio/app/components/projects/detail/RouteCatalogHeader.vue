<script setup lang="ts">
defineProps<{
  routesCount: number
  isFrontend: boolean
  loading: boolean
  syncing: boolean
  enriching: boolean
  enrichLabel: string
  feedbackMessage: string | null
  feedbackSuccess: boolean
  enrichedCount: number
  totalTargetRequestsCount: number
  domainList: { domain: string, count: number }[]
  methodList: { method: string, count: number }[]
}>()
const searchQuery = defineModel<string>('searchQuery', { required: true })
const selectedDomain = defineModel<string>('selectedDomain', { required: true })
const selectedMethod = defineModel<string>('selectedMethod', { required: true })
const viewMode = defineModel<'grouped' | 'table'>('viewMode', { required: true })
const emit = defineEmits<{ sync: [], enrich: [] }>()
const handleSyncFromCode = () => emit('sync')
const handleEnrichWithLLM = () => emit('enrich')
</script>

<template>
  <div class="p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs space-y-4">
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
      <div>
        <div class="flex flex-wrap items-center gap-2.5">
          <h3 class="font-bold text-neutral-900 dark:text-white text-sm uppercase tracking-wider flex items-center gap-2">
            <UIcon :name="isFrontend ? 'i-lucide-panels-top-left' : 'i-lucide-route'" class="w-4 h-4 text-sky-400" />
            <span>{{ isFrontend ? 'Daftar Halaman & Request API' : 'Daftar Router & API Endpoints Database' }}</span>
          </h3>
          <UBadge size="xs" color="success" variant="subtle" class="font-bold">
            {{ routesCount }} {{ isFrontend ? 'Halaman di DB' : 'Rute di DB' }}
          </UBadge>
          <UBadge size="xs" color="neutral" variant="subtle">
            {{ domainList.length }} {{ isFrontend ? 'Modul Fitur' : 'Domain Bisnis' }}
          </UBadge>
          <UBadge v-if="isFrontend && totalTargetRequestsCount > 0" size="xs" color="info" variant="subtle" class="font-semibold">
            {{ totalTargetRequestsCount }} Endpoint API Terpetakan
          </UBadge>
          <UBadge v-if="enrichedCount > 0" size="xs" color="primary" variant="subtle">
            {{ enrichedCount }} Terkategori LLM
          </UBadge>
        </div>
        <p class="text-xs text-neutral-500 mt-1">
          {{ isFrontend
            ? 'Katalog halaman antarmuka pengguna tersimpan permanen di database dan memetakan seluruh request endpoint API yang dipanggil di tiap halaman.'
            : 'Router tersimpan secara permanen di database. Sinkronkan dengan file kode sumber kapan saja dan kelompokkan fungsionalitasnya menggunakan AI / LLM.' }}
        </p>
      </div>

      <!-- 2 Action Buttons -->
      <div class="flex flex-wrap items-center gap-2 shrink-0">
        <!-- Button 1: Auto Sinkron Kode -->
        <UButton
          icon="i-lucide-refresh-cw"
          color="primary"
          variant="soft"
          size="xs"
          :loading="syncing"
          :title="isFrontend
            ? 'Pindai kode sumber aktual: sinkronkan daftar halaman dan deteksi request API terbaru dari komponen/halaman'
            : 'Pindai kode sumber aktual: tambah rute baru, perbarui perubahan, dan hapus rute yang hilang dari file kode'"
          @click="handleSyncFromCode"
        >
          {{ syncing ? 'Menyinkronkan...' : (isFrontend ? 'Auto Sinkron Halaman' : 'Auto Sinkron Kode') }}
        </UButton>

        <!-- Button 2: Auto Keterangan & Kelompokkan via LLM -->
        <UButton
          icon="i-lucide-sparkles"
          color="primary"
          variant="solid"
          size="xs"
          :loading="enriching"
          :title="isFrontend
            ? 'Kirim daftar halaman ke LLM untuk dikelompokkan ke modul fitur serta dilengkapi keterangan profesional'
            : 'Kirim daftar rute ke model LLM untuk dikelompokkan berdasarkan fungsi & domain serta dilengkapi keterangan profesional'"
          @click="handleEnrichWithLLM"
        >
          {{ enriching ? (enrichLabel || 'Menganalisis LLM...') : 'Kirim ke LLM & Kelompokkan' }}
        </UButton>
      </div>
    </div>

    <!-- Feedback / Sync Notification Banner -->
    <div
      v-if="feedbackMessage"
      class="p-3 rounded-lg border text-xs flex items-center justify-between gap-3 transition-all"
      :class="feedbackSuccess ? 'bg-sky-50/70 dark:bg-sky-950/40 border-sky-200 dark:border-sky-800 text-sky-800 dark:text-sky-200' : 'bg-rose-50/70 dark:bg-rose-950/40 border-rose-200 dark:border-rose-800 text-rose-800 dark:text-rose-200'"
    >
      <div class="flex items-center gap-2">
        <UIcon :name="feedbackSuccess ? 'i-lucide-check-circle' : 'i-lucide-alert-circle'" class="w-4 h-4 shrink-0" />
        <span>{{ feedbackMessage }}</span>
      </div>
      <button
        type="button"
        class="text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200 cursor-pointer"
        @click="feedbackMessage = null"
      >
        <UIcon name="i-lucide-x" class="w-3.5 h-3.5" />
      </button>
    </div>

    <!-- Toolbar: Search, Filters & View Toggle -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-2 border-t border-neutral-100 dark:border-neutral-800/80">
      <!-- Search Input -->
      <div class="w-full sm:w-80">
        <UInput
          v-model="searchQuery"
          icon="i-lucide-search"
          :placeholder="isFrontend ? 'Cari rute halaman, file vue, modul, atau request API...' : 'Cari path, method, handler, domain...'"
          size="xs"
        />
      </div>

      <!-- Right Controls: Method filter & View Mode -->
      <div class="flex flex-wrap items-center gap-2">
        <!-- Method Filter Dropdown / Select -->
        <div class="flex items-center gap-1 text-xs text-neutral-500">
          <span class="hidden md:inline text-[11px]">Method:</span>
          <select
            v-model="selectedMethod"
            class="px-2 py-1 rounded-md border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 text-xs font-mono font-bold text-neutral-700 dark:text-neutral-300"
          >
            <option value="ALL">
              Semua Method ({{ routesCount }})
            </option>
            <option v-for="m in methodList" :key="m.method" :value="m.method">
              {{ m.method }} ({{ m.count }})
            </option>
          </select>
        </div>

        <!-- View Mode Toggle -->
        <div class="flex items-center p-0.5 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-100/80 dark:bg-neutral-800/60 text-xs">
          <button
            type="button"
            class="px-2.5 py-1 rounded-md flex items-center gap-1.5 transition-colors cursor-pointer text-[11px]"
            :class="viewMode === 'grouped' ? 'bg-white dark:bg-neutral-900 font-bold text-neutral-900 dark:text-white shadow-2xs' : 'text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-200'"
            @click="viewMode = 'grouped'"
          >
            <UIcon name="i-lucide-layers" class="w-3.5 h-3.5" />
            <span>{{ isFrontend ? 'Kelompok Modul' : 'Kelompok Domain' }}</span>
          </button>
          <button
            type="button"
            class="px-2.5 py-1 rounded-md flex items-center gap-1.5 transition-colors cursor-pointer text-[11px]"
            :class="viewMode === 'table' ? 'bg-white dark:bg-neutral-900 font-bold text-neutral-900 dark:text-white shadow-2xs' : 'text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-200'"
            @click="viewMode = 'table'"
          >
            <UIcon name="i-lucide-table" class="w-3.5 h-3.5" />
            <span>Tabel Lengkap</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Domain Selector Horizontal Pills -->
    <div v-if="domainList.length > 0" class="flex flex-wrap items-center gap-1.5 pt-1 text-xs">
      <span class="text-[11px] font-semibold text-neutral-400 mr-1">{{ isFrontend ? 'Modul:' : 'Domain:' }}</span>
      <button
        type="button"
        class="px-2.5 py-1 rounded-full text-[11px] font-medium transition-colors cursor-pointer"
        :class="selectedDomain === 'ALL' ? 'bg-primary-500 text-white font-bold' : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:bg-neutral-200 dark:hover:bg-neutral-700'"
        @click="selectedDomain = 'ALL'"
      >
        Semua ({{ routesCount }})
      </button>

      <button
        v-for="d in domainList"
        :key="d.domain"
        type="button"
        class="px-2.5 py-1 rounded-full text-[11px] font-medium transition-colors cursor-pointer flex items-center gap-1"
        :class="selectedDomain === d.domain ? 'bg-primary-500 text-white font-bold' : 'bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400 hover:bg-neutral-200 dark:hover:bg-neutral-700'"
        @click="selectedDomain = d.domain"
      >
        <span>{{ d.domain }}</span>
        <span class="opacity-70 text-[10px]">({{ d.count }})</span>
      </button>
    </div>
  </div>
</template>
