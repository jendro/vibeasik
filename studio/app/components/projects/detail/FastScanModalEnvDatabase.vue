<script setup lang="ts">
import type { FastScanResult } from '~/types'
import { addDynamicMeta as addDynamicMetaHelper, removeDynamicMeta as removeDynamicMetaHelper } from '~/utils/fastScanDynamicMeta'

const props = defineProps<{ scanResult: FastScanResult, savingScanChanges?: boolean }>()
defineEmits<{ saveChanges: [] }>()
const toast = useToast()

const newMetaKey = ref('')
const newMetaVal = ref('')

const addDynamicMeta = () => {
  const isAdded = addDynamicMetaHelper(
    props.scanResult,
    newMetaKey.value,
    newMetaVal.value,
    (description) => {
      toast.add({
        title: 'Validasi Gagal',
        description,
        icon: 'i-lucide-alert-circle',
        color: 'warning'
      })
    }
  )
  if (!isAdded) return

  newMetaKey.value = ''
  newMetaVal.value = ''
}

const removeDynamicMeta = (key: string) => {
  removeDynamicMetaHelper(props.scanResult, key)
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-2 p-3 rounded-xl bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-800 text-xs">
      <div class="flex items-center gap-2 text-neutral-600 dark:text-neutral-300">
        <UIcon name="i-lucide-database-zap" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
        <span class="font-medium">Informasi diekstrak dari .env, file konfigurasi, manifest, dan arsitektur kode.</span>
      </div>
      <UButton
        icon="i-lucide-save"
        color="primary"
        size="xs"
        variant="solid"
        :loading="savingScanChanges"
        @click="$emit('saveChanges')"
      >
        Simpan Perubahan Dinamis
      </UButton>
    </div>

    <!-- Database Section -->
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-database" class="w-4 h-4 text-primary-600 dark:text-primary-400" />
          <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Database & Persistence</span>
        </div>
        <TechBadge
          v-if="scanResult.extracted_env?.database_type"
          :tech="scanResult.extracted_env.database_type"
          size="sm"
        />
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs font-mono">
        <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700/60">
          <div class="text-xs uppercase font-bold text-neutral-400">
            Database Engine
          </div>
          <div class="mt-1 flex items-center gap-1.5 font-semibold text-neutral-800 dark:text-neutral-100">
            <TechBadge :tech="scanResult.extracted_env?.database_type || 'None'" size="xs" />
          </div>
        </div>

        <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700/60">
          <div class="text-xs uppercase font-bold text-neutral-400">
            ORM / Driver / Client
          </div>
          <div class="mt-1 flex items-center gap-1.5 font-semibold text-neutral-800 dark:text-neutral-100">
            <TechBadge :tech="scanResult.extracted_env?.orm_driver || 'Standar / Raw SQL'" size="xs" />
          </div>
        </div>

        <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/60 border border-neutral-200 dark:border-neutral-700/60">
          <div class="text-xs uppercase font-bold text-neutral-400">
            Nama DB / Connection Target
          </div>
          <div class="mt-1 font-semibold text-neutral-800 dark:text-neutral-100 truncate" :title="scanResult.extracted_env?.database_name">
            {{ scanResult.extracted_env?.database_name || '-' }}
          </div>
        </div>
      </div>
    </div>

    <!-- Login & Auth -->
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
      <div class="flex items-center gap-2">
        <UIcon name="i-lucide-key-round" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
        <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Mekanisme Login & Autentikasi</span>
      </div>

      <div class="space-y-1.5">
        <label class="block text-xs font-semibold text-neutral-500 uppercase tracking-wider">Metode Login yang Tersedia:</label>
        <div v-if="scanResult.extracted_env?.auth_methods && scanResult.extracted_env.auth_methods.length > 0" class="flex flex-wrap gap-2">
          <TechBadge
            v-for="(auth, aIdx) in scanResult.extracted_env.auth_methods"
            :key="aIdx"
            :tech="auth"
            size="sm"
          />
        </div>
        <div v-else class="text-xs text-neutral-500 italic">
          Tidak terdeteksi autentikasi khusus atau endpoint login publik.
        </div>
      </div>

      <div v-if="scanResult.extracted_env?.login_endpoints && scanResult.extracted_env.login_endpoints.length > 0" class="space-y-1.5">
        <label class="block text-xs font-semibold text-neutral-500 uppercase tracking-wider">Endpoint Login / Auth:</label>
        <div class="flex flex-wrap gap-2 font-mono text-xs">
          <span
            v-for="(ep, epIdx) in scanResult.extracted_env.login_endpoints"
            :key="epIdx"
            class="px-2 py-1 rounded bg-neutral-100 dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 border border-neutral-200 dark:border-neutral-700 select-all flex items-center gap-1.5"
          >
            <UIcon name="i-lucide-send" class="w-3 h-3 text-neutral-400" />
            <span>{{ ep }}</span>
          </span>
        </div>
      </div>

      <div v-if="scanResult.extracted_env?.login_and_access_guide" class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-800 space-y-1.5 text-xs">
        <div class="flex items-center gap-1 text-xs font-bold text-neutral-600 dark:text-neutral-300">
          <UIcon name="i-lucide-info" class="w-3.5 h-3.5 text-primary-500" />
          <span>Petunjuk Akses / Kredensial Default / Cara Login:</span>
        </div>
        <p class="leading-relaxed text-neutral-700 dark:text-neutral-300 whitespace-pre-line">
          {{ scanResult.extracted_env.login_and_access_guide }}
        </p>
      </div>
    </div>

    <!-- Dynamic Custom Metadata Manager -->
    <div class="p-4 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-3">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-tag" class="w-4 h-4 text-purple-600 dark:text-purple-400" />
          <span class="text-xs font-bold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">Metadata & Atribut Dinamis Komponen</span>
        </div>
        <span class="text-xs text-neutral-400">Dapat ditambah & disimpan fleksibel</span>
      </div>

      <div
        v-if="scanResult.extracted_env?.dynamic_metadata && Object.keys(scanResult.extracted_env.dynamic_metadata).length > 0"
        class="grid grid-cols-1 sm:grid-cols-2 gap-2"
      >
        <div
          v-for="(val, key) in scanResult.extracted_env.dynamic_metadata"
          :key="key"
          class="flex items-center justify-between gap-2 p-2.5 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50/70 dark:bg-neutral-800/40 text-xs"
        >
          <div class="min-w-0 flex-1">
            <div class="font-bold text-neutral-800 dark:text-neutral-200 truncate">
              {{ key }}
            </div>
            <div class="text-xs text-neutral-600 dark:text-neutral-400 truncate">
              {{ val }}
            </div>
          </div>
          <button
            type="button"
            class="text-neutral-400 hover:text-red-500 p-1 cursor-pointer transition-colors"
            title="Hapus properti"
            @click="removeDynamicMeta(String(key))"
          >
            <UIcon name="i-lucide-x" class="w-4 h-4" />
          </button>
        </div>
      </div>

      <!-- Add Form Inline -->
      <div class="pt-2 border-t border-neutral-100 dark:border-neutral-800 flex flex-wrap sm:flex-nowrap items-center gap-2 text-xs">
        <input
          v-model="newMetaKey"
          type="text"
          placeholder="Nama Properti (e.g. redis_prefix, queue_driver)"
          class="flex-1 px-3 py-1.5 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 text-xs focus:outline-none focus:ring-1 focus:ring-primary-500"
          @keydown.enter.prevent="addDynamicMeta"
        >
        <input
          v-model="newMetaVal"
          type="text"
          placeholder="Nilai / Catatan"
          class="flex-1 px-3 py-1.5 rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 text-xs focus:outline-none focus:ring-1 focus:ring-primary-500"
          @keydown.enter.prevent="addDynamicMeta"
        >
        <UButton
          icon="i-lucide-plus"
          size="xs"
          color="neutral"
          variant="subtle"
          @click="addDynamicMeta"
        >
          Tambah
        </UButton>
      </div>
    </div>
  </div>
</template>
