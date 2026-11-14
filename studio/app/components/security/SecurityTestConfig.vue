<script setup lang="ts">
import type { ProjectComponent } from '~/types'
import { securityBaseUrlProblem } from '~/utils/securityTest'

// Checklist Uji Keamanan (bentuk sama dengan Uji Runtime): kiri = server yang diuji, kanan = yang diuji.
export interface SecurityTestOptions {
  component_id: number
  base_url: string
  start_server: boolean
  burst: boolean
  keep_running: boolean
  attacks: boolean
  auth_header: string
}

const options = defineModel<SecurityTestOptions>({ required: true })
const props = defineProps<{
  components: ProjectComponent[]
  disabled?: boolean
}>()

const cardCls = 'rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f]'
const optionRow = 'flex items-start gap-3 px-4 py-3'

const SERVER_ROLES = ['backend', 'worker']
const backends = computed(() => props.components.filter(c => SERVER_ROLES.includes(c.role)))
const others = computed(() => props.components.filter(c => !SERVER_ROLES.includes(c.role) && !['frontend_mobile', 'docs', 'cli'].includes(c.role)))
const useLive = computed({
  get: () => options.value.component_id === -1,
  set: (on: boolean) => { options.value.component_id = on ? -1 : 0 }
})
const urlError = computed(() => (useLive.value ? securityBaseUrlProblem(options.value.base_url) || (options.value.base_url.trim() ? '' : 'Isi URL server live') : ''))
const portOf = (c: ProjectComponent) => c.run_port || c.port
</script>

<template>
  <fieldset :disabled="disabled" class="grid grid-cols-1 xl:grid-cols-5 gap-4" :class="{ 'opacity-60': disabled }">
    <!-- Server yang diuji -->
    <section :class="[cardCls, 'xl:col-span-3 overflow-hidden']">
      <div class="px-4 py-3 border-b border-neutral-200 dark:border-white/[0.08] text-sm font-semibold text-neutral-900 dark:text-white">
        Server yang diuji
      </div>
      <ul class="divide-y divide-neutral-100 dark:divide-white/[0.06]">
        <li v-if="backends.length > 1" class="px-4 py-2.5">
          <label class="flex items-start gap-3 cursor-pointer">
            <input v-model="options.component_id" type="radio" :value="0" class="mt-1 accent-neutral-900 dark:accent-white">
            <div class="min-w-0">
              <div class="text-sm font-semibold text-neutral-900 dark:text-white">Semua komponen backend</div>
              <div class="text-xs text-neutral-500">Kode semua backend dibaca; server <b>{{ backends[0]?.name }}</b> yang dinyalakan & diuji.</div>
            </div>
          </label>
        </li>
        <li v-for="c in [...backends, ...others]" :key="c.id" class="px-4 py-2.5">
          <label class="flex items-start gap-3 cursor-pointer">
            <input
              v-model="options.component_id"
              type="radio"
              :value="backends.length === 1 && c.id === backends[0]?.id ? 0 : c.id"
              class="mt-1 accent-neutral-900 dark:accent-white"
            >
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span class="text-sm font-semibold text-neutral-900 dark:text-white truncate">{{ c.name }}</span>
                <UBadge size="xs" color="neutral" variant="subtle">{{ c.role }}</UBadge>
                <span v-if="portOf(c)" class="font-mono text-xs tabular-nums text-neutral-500">:{{ portOf(c) }}</span>
              </div>
              <div class="text-xs text-neutral-500 mt-0.5 truncate">
                <span v-if="c.dev_command" class="font-mono">{{ c.dev_command }}</span>
                <span v-else class="text-amber-600 dark:text-amber-400">Belum punya dev command — server tidak bisa dinyalakan otomatis</span>
              </div>
            </div>
          </label>
        </li>
        <li class="px-4 py-2.5 space-y-2">
          <label class="flex items-start gap-3 cursor-pointer">
            <input v-model="useLive" type="checkbox" class="mt-1 accent-neutral-900 dark:accent-white">
            <div class="min-w-0">
              <div class="text-sm font-semibold text-neutral-900 dark:text-white">Server live (URL)</div>
              <div class="text-xs text-neutral-500">Uji server yang sudah berjalan di tempat lain — tidak ada yang dinyalakan.</div>
            </div>
          </label>
          <UFormField v-if="useLive" size="sm" :error="urlError || undefined" class="pl-7">
            <UInput v-model="options.base_url" size="sm" icon="i-lucide-globe" placeholder="https://api.contoh.com atau 127.0.0.1:4101" class="w-full" />
          </UFormField>
        </li>
        <li v-if="!components.length" class="px-4 py-8 text-center text-xs text-neutral-500">
          Proyek ini belum punya komponen. Scan proyek dulu agar backend-nya terdeteksi.
        </li>
      </ul>
    </section>

    <!-- Yang diuji -->
    <section :class="[cardCls, 'xl:col-span-2 divide-y divide-neutral-100 dark:divide-white/[0.06] h-fit']">
      <div class="px-4 py-3 text-sm font-semibold text-neutral-900 dark:text-white">
        Yang diuji
      </div>
      <div :class="optionRow">
        <UCheckbox class="mt-0.5" :model-value="true" disabled />
        <div class="text-xs text-neutral-600 dark:text-neutral-300">
          <div class="text-sm font-medium text-neutral-900 dark:text-white">
            Router & kode backend
          </div>
          Rate limit, query berparameter, header, CORS, validasi input, autentikasi.
        </div>
      </div>
      <div :class="optionRow">
        <UCheckbox v-model="options.start_server" class="mt-0.5" :disabled="useLive" />
        <div class="text-xs text-neutral-600 dark:text-neutral-300">
          <div class="text-sm font-medium text-neutral-900 dark:text-white">
            Nyalakan server bila mati
          </div>
          Dev command dijalankan, ditunggu sampai port siap, lalu dimatikan lagi setelah uji.
        </div>
      </div>
      <div :class="optionRow">
        <UCheckbox v-model="options.attacks" class="mt-0.5" />
        <div class="min-w-0 flex-1 text-xs text-neutral-600 dark:text-neutral-300 space-y-2">
          <div>
            <div class="text-sm font-medium text-neutral-900 dark:text-white">
              Serang semua endpoint
            </div>
            Payload peretas ke setiap endpoint GET & rute login: SQL/NoSQL injection (termasuk berbasis waktu), XSS, path traversal, command injection, template injection, open redirect, CRLF, JWT palsu, akses tanpa login, berkas .env/.git. DELETE tidak pernah dikirim.
          </div>
          <UFormField v-if="options.attacks" size="sm" label="Header login akun uji (opsional)" help="Bearer token atau Cookie: … agar endpoint di balik login ikut diserang. Tidak disimpan.">
            <UInput v-model="options.auth_header" size="sm" type="password" icon="i-lucide-key-round" placeholder="Bearer eyJ… atau Cookie: session=…" class="w-full" autocomplete="off" />
          </UFormField>
        </div>
      </div>
      <div :class="optionRow">
        <UCheckbox v-model="options.burst" class="mt-0.5" />
        <div class="text-xs text-neutral-600 dark:text-neutral-300">
          <div class="text-sm font-medium text-neutral-900 dark:text-white">
            Uji burst rate limit
          </div>
          25 request cepat (dan 12 ke rute login) untuk membuktikan 429. Matikan untuk server produksi yang rapuh.
        </div>
      </div>
      <div :class="optionRow">
        <UCheckbox v-model="options.keep_running" class="mt-0.5" :disabled="useLive || !options.start_server" />
        <div class="text-xs text-neutral-600 dark:text-neutral-300">
          <div class="text-sm font-medium text-neutral-900 dark:text-white">
            Biarkan server tetap menyala
          </div>
          Server yang dinyalakan uji tidak dimatikan setelah selesai.
        </div>
      </div>
      <p class="flex gap-2 px-4 py-3 text-xs text-neutral-500 dark:text-slate-400">
        <UIcon name="i-lucide-info" class="size-3.5 mt-0.5 shrink-0" />
        Probe hanya mengirim request biasa (GET, OPTIONS, POST body kosong ke rute login) — tidak ada data yang diubah. Gunakan hanya pada server yang Anda kelola.
      </p>
    </section>
  </fieldset>
</template>
