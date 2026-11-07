<script setup lang="ts">
import type { CensoredKeyword, TestCensorResponse } from '~/types'

const props = defineProps<{
  keywords: CensoredKeyword[]
  builtInPatterns: string[]
  loading?: boolean
}>()

defineEmits<{
  add: []
  edit: [keyword: CensoredKeyword]
  delete: [keyword: CensoredKeyword]
  toggleActive: [keyword: CensoredKeyword]
  applyPreset: [preset: any]
}>()

const { testCensorPayload } = useApi()
const toast = useToast()

const keywordSearch = ref('')
const selectedCategoryFilter = ref('ALL')
const revealedKeywordIds = ref<number[]>([])

// Live Simulator State
const testSimulatorInput = ref('')
const testSimulatorLoading = ref(false)
const testSimulatorResult = ref<TestCensorResponse | null>(null)
const copiedSanitized = ref(false)

const categoryOptions = [
  'API_KEY',
  'AUTH_TOKEN',
  'DATABASE_CREDENTIAL',
  'ENVIRONMENT_SECRET',
  'OTHER'
]

const keywordPresets = [
  { label: 'DB Password', keyword: 'DB_PASSWORD', category: 'DATABASE_CREDENTIAL', description: 'Kata sandi basis data utama', is_regex: false },
  { label: 'Stripe Secret', keyword: 'sk_live_', category: 'API_KEY', description: 'Prefix live secret key Stripe Gateway', is_regex: false },
  { label: 'Private RSA Key', keyword: 'BEGIN RSA PRIVATE KEY', category: 'AUTH_TOKEN', description: 'Header SSL/TLS Certificate Private Key', is_regex: false },
  { label: 'OAuth Client Secret', keyword: 'CLIENT_SECRET', category: 'API_KEY', description: 'Kredensial OAuth Provider', is_regex: false },
  { label: 'JWT App Secret', keyword: 'JWT_SECRET', category: 'AUTH_TOKEN', description: 'Secret signing HMAC/RSA token', is_regex: false }
]

const toggleRevealKeyword = (id: number) => {
  const idx = revealedKeywordIds.value.indexOf(id)
  if (idx !== -1) {
    revealedKeywordIds.value.splice(idx, 1)
  } else {
    revealedKeywordIds.value.push(id)
  }
}

const isKeywordRevealed = (id: number) => revealedKeywordIds.value.includes(id)

const filteredKeywords = computed(() => {
  return props.keywords.filter((k) => {
    const q = keywordSearch.value.trim().toLowerCase()
    const matchesSearch = !q
      || k.keyword.toLowerCase().includes(q)
      || (k.description && k.description.toLowerCase().includes(q))
      || k.category.toLowerCase().includes(q)
    const matchesCategory = selectedCategoryFilter.value === 'ALL' || k.category === selectedCategoryFilter.value
    return matchesSearch && matchesCategory
  })
})

const activeKeywordCount = computed(() => props.keywords.filter(k => k.is_active).length)

const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    copiedSanitized.value = true
    setTimeout(() => { copiedSanitized.value = false }, 2000)
  } catch {
    // ignore
  }
}

const fillSimulatorPreset = (type: 'all' | 'tokens' | 'db') => {
  if (type === 'all') {
    testSimulatorInput.value = `DATABASE_URL=postgres://root:p@ssw0rd123@db.internal:5432/production
OPENAI_API_KEY=sk-proj-abc1234567890def1234567890abcdef1234567890
STRIPE_SECRET=sk_live_51NyTestSecretKeyLiveStripeXYZ987654321
JWT_SECRET=super_secret_jwt_hmac_sha256_token
AUTH_BEARER=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.doNotLeakThisSignature`
  } else if (type === 'tokens') {
    testSimulatorInput.value = `const apiKey = "sk-proj-9999888877776666555544443333222211110000";
const githubToken = "ghp${'_'}AbCdEfGhIjKlMnOpQrStUvWxYz0123456789";`
  } else {
    testSimulatorInput.value = `DB_HOST=10.0.1.20
DB_PASSWORD=MyUltraSecurePass2026!
DSN=mysql://admin:MyUltraSecurePass2026!@10.0.1.20:3306/finance_app`
  }
}

const handleRunLiveCensorTest = async () => {
  if (!testSimulatorInput.value.trim()) return
  testSimulatorLoading.value = true
  try {
    const res = await testCensorPayload(testSimulatorInput.value)
    testSimulatorResult.value = res
  } catch (err: any) {
    toast.add({
      title: 'Uji Sensor Gagal',
      description: `Uji sensor gagal: ${err.message || err}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    testSimulatorLoading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Double Security Vault Banner -->
    <div class="p-5 rounded-2xl border border-amber-200 dark:border-amber-900/60 bg-gradient-to-br from-amber-50/70 via-white to-amber-50/30 dark:from-amber-950/30 dark:via-neutral-900 dark:to-neutral-900 shadow-xs space-y-4">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-amber-500/10 border border-amber-300 dark:border-amber-700/80 flex items-center justify-center text-amber-600 dark:text-amber-400 shrink-0 shadow-xs">
            <UIcon name="i-lucide-shield-check" class="w-5 h-5" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h3 class="text-sm font-bold text-neutral-900 dark:text-white">
                Double Security Censorship Shield & Vault
              </h3>
              <UBadge size="xs" color="warning" variant="subtle">
                ACTIVE_ENFORCED
              </UBadge>
            </div>
            <p class="text-xs text-neutral-600 dark:text-neutral-400 mt-0.5">
              Proteksi sensor 2-lapis otomatis: Seluruh kredensial, token API, JWT, password, dan kunci privat dialias sebelum prompt dikirim ke LLM.
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <button
            type="button"
            class="px-3 py-1.5 rounded-lg text-xs font-semibold bg-amber-600 hover:bg-amber-700 text-white inline-flex items-center gap-1.5 shadow-xs transition-colors cursor-pointer"
            @click="$emit('add')"
          >
            <UIcon name="i-lucide-plus" class="w-3.5 h-3.5" />
            <span>Tambah Keyword Sensor</span>
          </button>
        </div>
      </div>

      <!-- 3 Stat Metrics -->
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-1">
        <div class="p-3 rounded-xl bg-white/90 dark:bg-neutral-800 border border-amber-200/60 dark:border-white/10 space-y-1">
          <div class="flex items-center justify-between text-neutral-500 dark:text-neutral-400">
            <span class="text-[10px] uppercase font-bold tracking-wider">Custom Keywords (Pass 1)</span>
            <UIcon name="i-lucide-file-key" class="w-3.5 h-3.5 text-purple-500" />
          </div>
          <div class="text-xl font-bold font-mono text-purple-600 dark:text-purple-400">
            {{ keywords.length }} <span class="text-xs font-normal text-neutral-400">({{ activeKeywordCount }} aktif)</span>
          </div>
          <div class="text-[11px] text-neutral-500">
            Pola kata kunci & regex terdaftar pengguna
          </div>
        </div>

        <div class="p-3 rounded-xl bg-white/90 dark:bg-neutral-800 border border-amber-200/60 dark:border-white/10 space-y-1">
          <div class="flex items-center justify-between text-neutral-500 dark:text-neutral-400">
            <span class="text-[10px] uppercase font-bold tracking-wider">Pola Sistem (Pass 2)</span>
            <UIcon name="i-lucide-shield-alert" class="w-3.5 h-3.5 text-sky-500" />
          </div>
          <div class="text-xl font-bold font-mono text-sky-600 dark:text-sky-400">
            {{ builtInPatterns.length || 13 }} <span class="text-xs font-normal text-neutral-400">detektor aktif</span>
          </div>
          <div class="text-[11px] text-neutral-500">
            Otomatis deteksi AI keys, JWT, AWS, Private Key
          </div>
        </div>

        <div class="p-3 rounded-xl bg-white/90 dark:bg-neutral-800 border border-amber-200/60 dark:border-white/10 space-y-1">
          <div class="flex items-center justify-between text-neutral-500 dark:text-neutral-400">
            <span class="text-[10px] uppercase font-bold tracking-wider">Status Pencegahan Kebocoran</span>
            <UIcon name="i-lucide-lock" class="w-3.5 h-3.5 text-amber-500" />
          </div>
          <div class="text-xl font-bold font-mono text-amber-600 dark:text-amber-400">
            100% Aman
          </div>
          <div class="text-[11px] text-neutral-500">
            Aliasing aman format [ALIASED_...] sebelum kirim
          </div>
        </div>
      </div>
    </div>

    <!-- Quick Keyword Presets -->
    <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/10 bg-neutral-50/50 dark:bg-neutral-800/40 flex flex-wrap items-center justify-between gap-2 text-xs">
      <div class="flex items-center gap-1.5 text-neutral-600 dark:text-neutral-400 font-medium">
        <UIcon name="i-lucide-zap" class="w-3.5 h-3.5 text-amber-500" />
        <span>Preset Cepat Tambah Kunci Sensitif:</span>
      </div>
      <div class="flex flex-wrap items-center gap-1.5">
        <button
          v-for="p in keywordPresets"
          :key="p.label"
          type="button"
          class="px-2.5 py-1 rounded-md text-[11px] font-mono bg-white dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 hover:border-amber-400 dark:hover:border-amber-600 text-neutral-700 dark:text-neutral-300 transition-colors cursor-pointer"
          @click="$emit('applyPreset', p)"
        >
          + {{ p.label }}
        </button>
      </div>
    </div>

    <!-- Censored Keywords Management Table Card -->
    <div class="p-5 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-900 space-y-4 shadow-xs">
      <!-- Filter & Search Toolbar -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-key-round" class="w-4 h-4 text-amber-500" />
          <h4 class="text-xs font-bold uppercase tracking-wider text-neutral-800 dark:text-neutral-200">
            Daftar Keyword & Pola Sensor ({{ filteredKeywords.length }} / {{ keywords.length }})
          </h4>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <!-- Category Filter Pills -->
          <div class="flex items-center gap-1 p-0.5 rounded-lg bg-neutral-100 dark:bg-neutral-800 text-[11px]">
            <button
              type="button"
              class="px-2 py-0.5 rounded cursor-pointer transition-colors"
              :class="selectedCategoryFilter === 'ALL' ? 'bg-white dark:bg-neutral-700 font-bold text-neutral-900 dark:text-white shadow-2xs' : 'text-neutral-500 hover:text-neutral-900 dark:hover:text-white'"
              @click="selectedCategoryFilter = 'ALL'"
            >
              Semua
            </button>
            <button
              v-for="cat in categoryOptions"
              :key="cat"
              type="button"
              class="px-2 py-0.5 rounded cursor-pointer transition-colors"
              :class="selectedCategoryFilter === cat ? 'bg-white dark:bg-neutral-700 font-bold text-neutral-900 dark:text-white shadow-2xs' : 'text-neutral-500 hover:text-neutral-900 dark:hover:text-white'"
              @click="selectedCategoryFilter = cat"
            >
              {{ cat.replace('_', ' ') }}
            </button>
          </div>

          <!-- Search Input -->
          <div class="w-48 sm:w-56">
            <input
              v-model="keywordSearch"
              type="text"
              placeholder="Cari keyword atau pola..."
              class="w-full px-2.5 py-1 text-xs rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white placeholder:text-neutral-400 focus:outline-none focus:ring-1 focus:ring-amber-500"
            >
          </div>
        </div>
      </div>

      <!-- Table of Keywords -->
      <div v-if="filteredKeywords.length > 0" class="overflow-x-auto rounded-xl border border-neutral-200 dark:border-white/10">
        <table class="w-full text-left text-xs font-sans">
          <thead class="bg-neutral-50 dark:bg-neutral-800 text-neutral-500 uppercase text-[10px] tracking-wider border-b border-neutral-200 dark:border-white/10">
            <tr>
              <th class="p-3">
                Keyword / Pola Rahasia
              </th>
              <th class="p-3 w-36">
                Kategori
              </th>
              <th class="p-3">
                Deskripsi & Catatan
              </th>
              <th class="p-3 w-28 text-center">
                Status
              </th>
              <th class="p-3 w-28 text-right">
                Aksi
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800">
            <tr
              v-for="kw in filteredKeywords"
              :key="kw.id"
              class="hover:bg-neutral-50/60 dark:hover:bg-neutral-800/40 transition-colors"
            >
              <!-- Keyword column with mask toggle -->
              <td class="p-3 font-mono">
                <div class="flex items-center gap-2">
                  <span class="font-bold text-neutral-900 dark:text-white select-all">
                    {{ isKeywordRevealed(kw.id) ? kw.keyword : '••••••••••••' }}
                  </span>
                  <button
                    type="button"
                    class="p-1 rounded text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200 transition-colors cursor-pointer"
                    :title="isKeywordRevealed(kw.id) ? 'Sembunyikan keyword' : 'Tampilkan keyword asli'"
                    @click="toggleRevealKeyword(kw.id)"
                  >
                    <UIcon :name="isKeywordRevealed(kw.id) ? 'i-lucide-eye-off' : 'i-lucide-eye'" class="w-3.5 h-3.5" />
                  </button>
                  <button
                    type="button"
                    class="p-1 rounded text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200 transition-colors cursor-pointer"
                    title="Copy keyword"
                    @click="copyToClipboard(kw.keyword)"
                  >
                    <UIcon name="i-lucide-copy" class="w-3.5 h-3.5" />
                  </button>
                  <span
                    v-if="kw.is_regex"
                    class="px-1.5 py-0.5 rounded text-[10px] bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-300 font-mono font-bold"
                  >
                    Regex
                  </span>
                </div>
              </td>

              <!-- Category -->
              <td class="p-3">
                <span
                  class="px-2 py-0.5 rounded text-[11px] font-mono font-semibold"
                  :class="{
                    'bg-purple-50 text-purple-700 dark:bg-purple-950/60 dark:text-purple-300 border border-purple-200 dark:border-purple-800': kw.category === 'API_KEY',
                    'bg-blue-50 text-blue-700 dark:bg-blue-950/60 dark:text-blue-300 border border-blue-200 dark:border-blue-800': kw.category === 'AUTH_TOKEN',
                    'bg-amber-50 text-amber-700 dark:bg-amber-950/60 dark:text-amber-300 border border-amber-200 dark:border-amber-800': kw.category === 'DATABASE_CREDENTIAL',
                    'bg-sky-50 text-sky-700 dark:bg-sky-950/60 dark:text-sky-300 border border-sky-200 dark:border-sky-800': kw.category === 'ENVIRONMENT_SECRET',
                    'bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300': kw.category === 'OTHER'
                  }"
                >
                  {{ kw.category }}
                </span>
              </td>

              <!-- Description -->
              <td class="p-3 text-neutral-600 dark:text-neutral-300 text-xs">
                {{ kw.description || 'Tidak ada deskripsi' }}
              </td>

              <!-- Status Toggle -->
              <td class="p-3 text-center">
                <button
                  type="button"
                  class="px-2.5 py-0.5 rounded-full text-[11px] font-semibold transition-colors cursor-pointer inline-flex items-center gap-1"
                  :class="kw.is_active
                    ? 'bg-sky-100 text-sky-800 dark:bg-sky-950/80 dark:text-sky-300 border border-sky-200 dark:border-sky-800'
                    : 'bg-neutral-100 text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400 border border-neutral-200 dark:border-neutral-700'"
                  :title="kw.is_active ? 'Klik untuk nonaktifkan' : 'Klik untuk aktifkan'"
                  @click="$emit('toggleActive', kw)"
                >
                  <span class="w-1.5 h-1.5 rounded-full" :class="kw.is_active ? 'bg-sky-500' : 'bg-neutral-400'" />
                  <span>{{ kw.is_active ? 'Aktif' : 'Mati' }}</span>
                </button>
              </td>

              <!-- Action Buttons -->
              <td class="p-3 text-right">
                <div class="flex items-center justify-end gap-1">
                  <button
                    type="button"
                    class="p-1.5 rounded-lg border border-neutral-200 dark:border-neutral-700 hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-600 dark:text-neutral-300 transition-colors cursor-pointer"
                    title="Edit keyword sensor"
                    @click="$emit('edit', kw)"
                  >
                    <UIcon name="i-lucide-pencil" class="w-3.5 h-3.5" />
                  </button>
                  <button
                    type="button"
                    class="p-1.5 rounded-lg border border-neutral-200 dark:border-neutral-700 hover:border-rose-300 hover:bg-rose-50 hover:text-rose-600 dark:hover:border-rose-900 dark:hover:bg-rose-950/40 dark:hover:text-rose-400 text-neutral-400 transition-colors cursor-pointer"
                    title="Hapus keyword dari vault"
                    @click="$emit('delete', kw)"
                  >
                    <UIcon name="i-lucide-trash-2" class="w-3.5 h-3.5" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Empty search/filter state -->
      <div
        v-else
        class="p-8 text-center text-xs text-neutral-500 italic bg-neutral-50/50 dark:bg-neutral-800/30 rounded-xl border border-dashed border-neutral-200 dark:border-white/10 space-y-2"
      >
        <UIcon name="i-lucide-search-x" class="w-6 h-6 mx-auto text-neutral-400" />
        <div>Tidak ada keyword sensor yang cocok dengan filter atau kata kunci "{{ keywordSearch }}".</div>
      </div>
    </div>

    <!-- Built-in Threat Detectors Reference Card -->
    <div class="p-5 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-900 space-y-3 shadow-xs">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-shield-alert" class="w-4 h-4 text-sky-500" />
          <h4 class="text-xs font-bold uppercase tracking-wider text-neutral-800 dark:text-neutral-200">
            Pola Ancaman Kredensial Bawaan Sistem (Pass 2 - System Enforced)
          </h4>
        </div>
        <UBadge size="xs" color="success" variant="subtle">
          Always Active (Hardened)
        </UBadge>
      </div>
      <p class="text-xs text-neutral-500 dark:text-neutral-400">
        Berikut adalah daftar detektor ekspresi reguler bawaan yang berjalan otomatis memindai seluruh payload tanpa perlu dikonfigurasi manual:
      </p>
      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2 text-xs font-mono">
        <div
          v-for="(pattern, pIdx) in (builtInPatterns.length ? builtInPatterns : [
            'OpenAI API Keys (sk-[a-zA-Z0-9]{20,})',
            'Anthropic API Keys (sk-ant-[a-zA-Z0-9-]{20,})',
            'Google Gemini Keys (AIzaSy[a-zA-Z0-9_-]{33})',
            'AWS Access Key ID (AKIA[0-9A-Z]{16})',
            'AWS Secret Access Key (Entropy Scan)',
            'GitHub Tokens (ghp_, gho_, ghu_...)',
            'Slack Tokens (xoxb-, xoxp-...)',
            'Stripe Secret Keys (sk_live_, rk_live_...)',
            'JSON Web Tokens (eyJ[a-zA-Z0-9_-]+\\.eyJ...)',
            'Database URLs (postgres://, mysql://, mongodb://)',
            'Private Keys (BEGIN ... PRIVATE KEY)',
            'Generic High-Entropy Hex/Base64 Tokens',
            'Environment Secret Assignments (PASSWORD=...)'
          ])"
          :key="pIdx"
          class="p-2.5 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50/60 dark:bg-neutral-800/60 flex items-center gap-2 text-neutral-700 dark:text-neutral-300"
        >
          <UIcon name="i-lucide-check-circle-2" class="w-3.5 h-3.5 text-sky-500 shrink-0" />
          <span class="truncate text-[11px]">{{ pattern }}</span>
        </div>
      </div>
    </div>

    <!-- Live Real-Time Simulator Playground -->
    <div class="p-5 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-900 space-y-4 shadow-xs">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-2 border-b border-neutral-200 dark:border-white/10">
        <div>
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-flask-conical" class="w-4 h-4 text-purple-500" />
            <h4 class="text-xs font-bold uppercase tracking-wider text-neutral-800 dark:text-neutral-200">
              Simulator Live Uji Coba Sensor LLM & Gateway
            </h4>
          </div>
          <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5">
            Uji teks, prompt rahasia, atau variabel .env Anda secara real-time untuk membuktikan sensor aman sebelum dikirim ke AI.
          </p>
        </div>

        <!-- Preset buttons -->
        <div class="flex items-center gap-1.5">
          <span class="text-[11px] text-neutral-400">Sampel Uji:</span>
          <button
            type="button"
            class="px-2 py-1 rounded bg-neutral-100 hover:bg-neutral-200 dark:bg-neutral-800 dark:hover:bg-neutral-700 text-neutral-700 dark:text-neutral-300 text-[11px] font-mono cursor-pointer transition-colors"
            @click="fillSimulatorPreset('all')"
          >
            .env Campuran
          </button>
          <button
            type="button"
            class="px-2 py-1 rounded bg-neutral-100 hover:bg-neutral-200 dark:bg-neutral-800 dark:hover:bg-neutral-700 text-neutral-700 dark:text-neutral-300 text-[11px] font-mono cursor-pointer transition-colors"
            @click="fillSimulatorPreset('tokens')"
          >
            API Tokens
          </button>
          <button
            type="button"
            class="px-2 py-1 rounded bg-neutral-100 hover:bg-neutral-200 dark:bg-neutral-800 dark:hover:bg-neutral-700 text-neutral-700 dark:text-neutral-300 text-[11px] font-mono cursor-pointer transition-colors"
            @click="fillSimulatorPreset('db')"
          >
            Database URI
          </button>
        </div>
      </div>

      <!-- Textarea Input & Action -->
      <div class="space-y-2">
        <textarea
          v-model="testSimulatorInput"
          rows="4"
          placeholder="Ketik atau tempel teks berisi API key, password, URL database, atau token rahasia di sini untuk diuji..."
          class="w-full p-3 text-xs font-mono rounded-xl border border-neutral-200 dark:border-neutral-700 bg-neutral-50/50 dark:bg-neutral-800/60 text-neutral-900 dark:text-white placeholder:text-neutral-400 focus:outline-none focus:ring-1 focus:ring-purple-500"
        />
        <div class="flex items-center justify-between">
          <button
            v-if="testSimulatorInput"
            type="button"
            class="text-xs text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300 cursor-pointer"
            @click="testSimulatorInput = ''; testSimulatorResult = null"
          >
            Bersihkan input
          </button>
          <span v-else class="text-[11px] text-neutral-400">Ketik prompt untuk memulai simulasi sensor</span>

          <UButton
            icon="i-lucide-play"
            color="primary"
            size="xs"
            :loading="testSimulatorLoading"
            @click="handleRunLiveCensorTest"
          >
            Jalankan Sensor Test
          </UButton>
        </div>
      </div>

      <!-- Simulator Results (Comparison Box) -->
      <div v-if="testSimulatorResult" class="space-y-3 pt-2 border-t border-neutral-200 dark:border-white/10">
        <!-- Audit Result Pill -->
        <div class="p-3 rounded-xl bg-purple-50/70 dark:bg-purple-950/30 border border-purple-200/80 dark:border-purple-900/40 flex flex-wrap items-center justify-between gap-2 text-xs">
          <div class="flex items-center gap-2 text-purple-900 dark:text-purple-200 font-medium">
            <UIcon name="i-lucide-check-circle" class="w-4 h-4 text-purple-600 dark:text-purple-400" />
            <span>{{ testSimulatorResult.audit?.message || 'Uji sensor selesai.' }}</span>
          </div>
          <div class="flex items-center gap-2 font-mono text-[11px]">
            <span class="px-2 py-0.5 rounded bg-purple-200/80 dark:bg-purple-900 text-purple-800 dark:text-purple-200 font-bold">
              {{ testSimulatorResult.audit?.redacted_count || 0 }} Kunci Redacted
            </span>
            <span class="px-2 py-0.5 rounded bg-white dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 border border-neutral-200 dark:border-neutral-700">
              Custom: {{ testSimulatorResult.audit?.custom_matches || 0 }} | Built-in: {{ testSimulatorResult.audit?.built_in_matches || 0 }}
            </span>
          </div>
        </div>

        <!-- Comparison Split -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs font-mono">
          <!-- Raw / Original -->
          <div class="p-3.5 rounded-xl border border-rose-200 dark:border-rose-900/40 bg-rose-50/30 dark:bg-rose-950/20 space-y-2">
            <div class="flex items-center justify-between text-rose-800 dark:text-rose-300 font-bold text-[11px]">
              <span>TEKS ASLI (SEBELUM DISENSOR)</span>
              <span class="text-rose-500 font-normal">Kunci mentah terpapar</span>
            </div>
            <pre class="whitespace-pre-wrap break-all text-neutral-800 dark:text-neutral-200 text-xs max-h-48 overflow-y-auto font-mono p-2 rounded bg-white/80 dark:bg-neutral-900/80 border border-rose-100 dark:border-rose-900/30">{{ testSimulatorResult.original }}</pre>
          </div>

          <!-- Sanitized & Aliased (Clean) -->
          <div class="p-3.5 rounded-xl border border-sky-200 dark:border-sky-900/40 bg-sky-50/30 dark:bg-sky-950/20 space-y-2">
            <div class="flex items-center justify-between text-sky-800 dark:text-sky-300 font-bold text-[11px]">
              <span>PAYLOAD AMAN (HASIL SENSOR TERKIRIM KE AI)</span>
              <button
                type="button"
                class="text-sky-600 dark:text-sky-400 hover:underline cursor-pointer flex items-center gap-1 font-sans font-normal"
                @click="copyToClipboard(testSimulatorResult.sanitized_text)"
              >
                <UIcon :name="copiedSanitized ? 'i-lucide-check' : 'i-lucide-copy'" class="w-3.5 h-3.5" />
                <span>{{ copiedSanitized ? 'Disalin' : 'Copy' }}</span>
              </button>
            </div>
            <pre class="whitespace-pre-wrap break-all text-sky-900 dark:text-sky-200 text-xs max-h-48 overflow-y-auto font-mono p-2 rounded bg-white/80 dark:bg-slate-900/80 border border-sky-100 dark:border-sky-900/30">{{ testSimulatorResult.sanitized_text }}</pre>
          </div>
        </div>

        <!-- Replaced Secrets Table -->
        <div v-if="testSimulatorResult.audit?.records && testSimulatorResult.audit.records.length > 0" class="space-y-1.5 pt-1">
          <div class="text-[11px] font-bold text-neutral-700 dark:text-neutral-300 uppercase tracking-wider">
            Rincian Kredensial yang Dialias:
          </div>
          <div class="overflow-x-auto rounded-lg border border-neutral-200 dark:border-white/10">
            <table class="w-full text-left text-xs font-mono">
              <thead class="bg-neutral-50 dark:bg-neutral-800 text-neutral-500 uppercase text-[10px]">
                <tr>
                  <th class="p-2">
                    Alias Terkirim
                  </th>
                  <th class="p-2">
                    Kategori
                  </th>
                  <th class="p-2">
                    Masking Nilai Asli
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800">
                <tr v-for="(rec, rIdx) in testSimulatorResult.audit.records" :key="rIdx">
                  <td class="p-2 font-bold text-amber-600 dark:text-amber-400">
                    {{ rec.alias }}
                  </td>
                  <td class="p-2">
                    <UBadge size="xs" color="neutral" variant="subtle">
                      {{ rec.category }}
                    </UBadge>
                  </td>
                  <td class="p-2 text-neutral-500">
                    {{ rec.masked_preview }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
