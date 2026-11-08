<script setup lang="ts">
import type { DomainItem, DeployPlanResponse, CloudAuditResponse } from '~/types'

definePageMeta({ fluid: true })

const { searchDomains, generateDeployPlan, runCloudSecurityAudit } = useVibeasik()

// Active tab
const activeTab = ref<'domain' | 'deploy' | 'security'>('domain')

// --- Domain Search State ---
const domainQuery = ref('')
const domainSearching = ref(false)
const domainResults = ref<DomainItem[]>([])
const selectedDomain = ref<DomainItem | null>(null)
const domainOrdered = ref(false)

const handleSearchDomain = async () => {
  if (!domainQuery.value.trim()) return
  domainSearching.value = true
  domainOrdered.value = false
  try {
    const res = await searchDomains(domainQuery.value)
    domainResults.value = res.suggestions || []
  } catch (err) {
    console.error('Domain search error:', err)
  } finally {
    domainSearching.value = false
  }
}

const formatIDR = (val: number) => {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(val)
}

// --- Deploy Plan Generator State ---
const deployForm = reactive({
  projectName: 'MyAwesomeApp',
  language: 'go',
  framework: 'fiber',
  port: 8080,
  domain: 'app.vibeasik.com',
  targetEnv: 'vps',
  useDatabase: true,
  dbType: 'mysql'
})

const generatingPlan = ref(false)
const deployPlan = ref<DeployPlanResponse | null>(null)
const copiedFile = ref<string | null>(null)

const handleGenerateDeployPlan = async () => {
  generatingPlan.value = true
  try {
    const plan = await generateDeployPlan({
      project_name: deployForm.projectName,
      language: deployForm.language,
      framework: deployForm.framework,
      port: deployForm.port,
      domain: deployForm.domain,
      target_env: deployForm.targetEnv,
      use_database: deployForm.useDatabase,
      db_type: deployForm.dbType
    })
    deployPlan.value = plan
  } catch (err) {
    console.error('Gagal generate plan deploy:', err)
  } finally {
    generatingPlan.value = false
  }
}

const copyFileContent = (content: string, filename: string) => {
  navigator.clipboard.writeText(content)
  copiedFile.value = filename
  setTimeout(() => {
    copiedFile.value = null
  }, 2000)
}

// --- Cloud Security Audit State ---
const auditing = ref(false)
const auditResult = ref<CloudAuditResponse | null>(null)

const runSampleAudit = async () => {
  auditing.value = true
  try {
    const res = await runCloudSecurityAudit({
      project_name: deployForm.projectName,
      language: deployForm.language,
      framework: deployForm.framework,
      files_sample: {
        'main.go': `package main\nimport "github.com/gofiber/fiber/v2"\nfunc main() {\n    app := fiber.New()\n    app.Listen(":8080")\n}`,
        '.env.example': `PORT=8080\nDB_HOST=localhost\nAPP_SECRET=your_jwt_secret_here`,
        '.gitignore': `.env\nvendor/\nbin/\n*.log`
      },
      env_keys: ['PORT', 'DB_HOST', 'APP_SECRET']
    })
    auditResult.value = res
  } catch (err) {
    console.error('Audit failed:', err)
  } finally {
    auditing.value = false
  }
}
</script>

<template>
  <div class="w-full px-4 sm:px-6 lg:px-8 pt-6 space-y-6">
    <!-- Header Banner -->
    <div class="p-6 rounded-2xl bg-white dark:bg-slate-900/60 border border-neutral-200 dark:border-white/10 shadow-xs flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
      <div class="space-y-1">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-rocket" class="w-7 h-7 text-sky-500" />
          <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white">
            Delivery & Domain Orchestrator
          </h1>
          <UBadge color="primary" variant="subtle" size="sm">
            One-Click Production
          </UBadge>
        </div>
        <p class="text-sm text-neutral-600 dark:text-slate-300 max-w-2xl">
          Beli nama domain, buat blueprint deploy VPS/Docker dengan HTTPS otomatis, dan pastikan kode Anda lolos audit keamanan sebelum dirilis.
        </p>
      </div>

      <!-- Navigation Tabs -->
      <div class="flex items-center bg-neutral-100 dark:bg-slate-800/80 p-1 rounded-xl border border-neutral-200/60 dark:border-white/10 shrink-0">
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5"
          :class="activeTab === 'domain' ? 'bg-white dark:bg-sky-600 text-sky-900 dark:text-white shadow-xs' : 'text-neutral-600 dark:text-slate-300 hover:text-neutral-900 dark:hover:text-white'"
          @click="activeTab = 'domain'"
        >
          <UIcon name="i-lucide-globe" class="w-4 h-4" />
          Beli Domain
        </button>
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5"
          :class="activeTab === 'deploy' ? 'bg-white dark:bg-sky-600 text-sky-900 dark:text-white shadow-xs' : 'text-neutral-600 dark:text-slate-300 hover:text-neutral-900 dark:hover:text-white'"
          @click="activeTab = 'deploy'"
        >
          <UIcon name="i-lucide-box" class="w-4 h-4" />
          Deploy Blueprint
        </button>
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5"
          :class="activeTab === 'security' ? 'bg-white dark:bg-sky-600 text-sky-900 dark:text-white shadow-xs' : 'text-neutral-600 dark:text-slate-300 hover:text-neutral-900 dark:hover:text-white'"
          @click="activeTab = 'security'"
        >
          <UIcon name="i-lucide-shield-check" class="w-4 h-4" />
          Pre-flight Audit
        </button>
      </div>
    </div>

    <!-- ================= TAB 1: DOMAIN SEARCH ================= -->
    <div v-if="activeTab === 'domain'" class="space-y-6">
      <div class="p-6 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-4">
        <div class="space-y-1">
          <h2 class="text-lg font-bold text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-search" class="w-5 h-5 text-sky-500" />
            Cari & Beli Nama Domain untuk Aplikasi Anda
          </h2>
          <p class="text-xs text-neutral-500">
            Ketik ide nama aplikasi Anda (contoh: <code>aplikasiku</code>). Sistem otomatis mengecek ketersediaan TLD (.com, .id, .my.id, .app) lengkap dengan auto-DNS.
          </p>
        </div>

        <form class="flex gap-2 max-w-xl" @submit.prevent="handleSearchDomain">
          <UInput
            v-model="domainQuery"
            icon="i-lucide-globe"
            placeholder="Ketik nama domain (misal: bisniskreatif)..."
            class="flex-1"
          />
          <UButton
            type="submit"
            color="primary"
            :loading="domainSearching"
            icon="i-lucide-search"
          >
            Cari Domain
          </UButton>
        </form>

        <!-- Domain Results List -->
        <div v-if="domainResults.length > 0" class="pt-4 border-t border-neutral-100 dark:border-white/10 space-y-3">
          <h3 class="text-xs font-bold uppercase tracking-wider text-neutral-400 font-mono">
            Hasil Pencarian Domain:
          </h3>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div
              v-for="d in domainResults"
              :key="d.domain"
              class="p-4 rounded-xl border transition-all flex items-center justify-between gap-3"
              :class="[
                d.available
                  ? 'border-neutral-200 dark:border-white/10 hover:border-sky-500/50 bg-neutral-50/50 dark:bg-slate-800/30'
                  : 'border-red-200 dark:border-red-950/40 bg-red-50/20 dark:bg-red-950/10 opacity-70'
              ]"
            >
              <div class="space-y-0.5 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-bold text-sm text-neutral-900 dark:text-white truncate">{{ d.domain }}</span>
                  <UBadge v-if="d.available" size="xs" color="success" variant="subtle">
                    Tersedia
                  </UBadge>
                  <UBadge v-else size="xs" color="error" variant="subtle">
                    Terpakai
                  </UBadge>
                </div>
                <div class="text-xs text-neutral-500 flex items-center gap-2 font-mono">
                  <span>{{ formatIDR(d.price_annual) }}/thn</span>
                  <span v-if="d.auto_dns_ready" class="text-sky-500 text-xs">• Auto-DNS Ready</span>
                </div>
              </div>

              <UButton
                v-if="d.available"
                size="xs"
                color="primary"
                variant="solid"
                icon="i-lucide-shopping-cart"
                @click="selectedDomain = d; domainOrdered = true"
              >
                Pilih
              </UButton>
            </div>
          </div>
        </div>

        <!-- Order Success Notice -->
        <div v-if="domainOrdered && selectedDomain" class="p-4 rounded-xl bg-sky-50 dark:bg-sky-950/30 border border-sky-200 dark:border-sky-800 text-sky-800 dark:text-sky-300 space-y-2">
          <div class="flex items-center gap-2 font-semibold text-sm">
            <UIcon name="i-lucide-check-circle" class="w-5 h-5 text-sky-600 dark:text-sky-400" />
            Domain {{ selectedDomain.domain }} siap diproses!
          </div>
          <p class="text-xs leading-relaxed text-sky-700 dark:text-sky-300">
            A Record akan otomatis diarahkan ke server target hosting Anda. Anda bisa langsung masuk ke tab <strong>Deploy Blueprint</strong> untuk mengunduh konfigurasi servernya.
          </p>
        </div>
      </div>
    </div>

    <!-- ================= TAB 2: DEPLOY BLUEPRINT ================= -->
    <div v-else-if="activeTab === 'deploy'" class="space-y-6">
      <div class="p-6 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-6">
        <div class="space-y-1">
          <h2 class="text-lg font-bold text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-layers" class="w-5 h-5 text-sky-500" />
            Generator Blueprint Deploy (Docker + Caddy Auto-SSL)
          </h2>
          <p class="text-xs text-neutral-500 dark:text-slate-400">
            Kustomisasi konfigurasi server Anda. Sistem akan menghasilkan Dockerfile, docker-compose, dan Caddyfile yang siap dijalankan di VPS murah ($3-$5/bln).
          </p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label class="block text-xs font-semibold text-neutral-600 dark:text-neutral-400 mb-1">Nama Aplikasi</label>
            <UInput v-model="deployForm.projectName" placeholder="Nama aplikasi..." />
          </div>

          <div>
            <label class="block text-xs font-semibold text-neutral-600 dark:text-neutral-400 mb-1">Bahasa / Framework</label>
            <select
              v-model="deployForm.language"
              class="w-full text-sm rounded-lg border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-800 px-3 py-2 text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-sky-500"
            >
              <option value="go">
                Go (Fiber Backend)
              </option>
              <option value="typescript">
                TypeScript / Nuxt (Frontend)
              </option>
              <option value="python">
                Python (FastAPI)
              </option>
            </select>
          </div>

          <div>
            <label class="block text-xs font-semibold text-neutral-600 dark:text-neutral-400 mb-1">Target Domain</label>
            <UInput v-model="deployForm.domain" placeholder="app.vibeasik.com" />
          </div>
        </div>

        <div class="flex justify-end">
          <UButton
            color="primary"
            :loading="generatingPlan"
            icon="i-lucide-sparkles"
            @click="handleGenerateDeployPlan"
          >
            Hasilkan Konfigurasi Deploy
          </UButton>
        </div>

        <!-- Generated Files Output -->
        <div v-if="deployPlan" class="pt-6 border-t border-neutral-200 dark:border-white/10 space-y-4">
          <div class="flex items-center justify-between">
            <h3 class="font-bold text-sm text-neutral-900 dark:text-white flex items-center gap-2">
              <UIcon name="i-lucide-folder-check" class="w-4 h-4 text-sky-500" />
              File Konfigurasi yang Dihasilkan
            </h3>
            <span class="text-xs text-neutral-400 font-mono">{{ deployPlan.architecture }}</span>
          </div>

          <div class="space-y-4">
            <div
              v-for="file in deployPlan.generated_files"
              :key="file.filename"
              class="rounded-xl border border-neutral-800 bg-[#010810] overflow-hidden"
            >
              <div class="px-4 py-2 bg-neutral-900/80 flex items-center justify-between border-b border-white/10">
                <span class="font-mono text-xs font-bold text-neutral-200">{{ file.filename }}</span>
                <button
                  type="button"
                  class="px-2 py-1 rounded-md bg-neutral-800 hover:bg-neutral-700 text-neutral-300 text-xs font-mono flex items-center gap-1 transition-colors"
                  @click="copyFileContent(file.content, file.filename)"
                >
                  <UIcon :name="copiedFile === file.filename ? 'i-lucide-check' : 'i-lucide-copy'" class="w-3.5 h-3.5" />
                  {{ copiedFile === file.filename ? 'Tersalin' : 'Salin File' }}
                </button>
              </div>
              <pre class="p-4 font-mono text-xs text-neutral-300 overflow-x-auto whitespace-pre">{{ file.content }}</pre>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ================= TAB 3: PRE-FLIGHT SECURITY AUDIT ================= -->
    <div v-else class="space-y-6">
      <div class="p-6 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 space-y-4">
        <div class="space-y-1">
          <h2 class="text-lg font-bold text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-shield-alert" class="w-5 h-5 text-sky-500" />
            Cloud Security Gate (Pre-flight Scanner)
          </h2>
          <p class="text-xs text-neutral-500 dark:text-slate-400">
            Uji keamanan kode Anda sebelum diunggah ke internet. Memeriksa file .env terekspos, celah SQL injection, dan CORS wildcard.
          </p>
        </div>

        <div class="p-4 rounded-xl bg-neutral-50 dark:bg-slate-800/40 border border-neutral-200 dark:border-white/10 flex items-center justify-between">
          <div class="space-y-0.5">
            <span class="font-semibold text-sm text-neutral-800 dark:text-neutral-200">Audit Sampel Kode Proyek</span>
            <p class="text-xs text-neutral-500">
              Jalankan evaluasi keamanan menyeluruh terhadap struktur project saat ini.
            </p>
          </div>
          <UButton
            color="primary"
            variant="solid"
            :loading="auditing"
            icon="i-lucide-play"
            @click="runSampleAudit"
          >
            Mulai Audit Keamanan
          </UButton>
        </div>

        <!-- Audit Results Display -->
        <div v-if="auditResult" class="pt-4 border-t border-neutral-200 dark:border-white/10 space-y-4">
          <div class="flex items-center gap-4 p-4 rounded-xl bg-neutral-100 dark:bg-slate-800/60 border border-neutral-200/60 dark:border-white/10">
            <div class="text-3xl font-black font-mono text-sky-600 dark:text-sky-400">
              {{ auditResult.safety_score }}/100
            </div>
            <div>
              <div class="flex items-center gap-2">
                <span class="font-bold text-sm text-neutral-900 dark:text-white">Status Audit:</span>
                <UBadge :color="auditResult.status === 'PASSED' ? 'success' : 'warning'" size="xs">
                  {{ auditResult.status }}
                </UBadge>
              </div>
              <p class="text-xs text-neutral-500 mt-0.5">
                {{ auditResult.summary }}
              </p>
            </div>
          </div>

          <!-- Issues List -->
          <div v-if="auditResult.issues.length > 0" class="space-y-2">
            <h4 class="text-xs font-bold uppercase tracking-wider text-neutral-400 font-mono">
              Temuan Penting:
            </h4>
            <div
              v-for="iss in auditResult.issues"
              :key="iss.rule_id"
              class="p-3.5 rounded-xl border border-amber-200 dark:border-amber-900/50 bg-amber-50/50 dark:bg-amber-950/20 space-y-1.5"
            >
              <div class="flex items-center gap-2">
                <UBadge size="xs" color="warning">
                  {{ iss.severity }}
                </UBadge>
                <span class="font-semibold text-xs text-neutral-900 dark:text-white">{{ iss.title }}</span>
                <span class="text-xs font-mono text-neutral-400 ml-auto">{{ iss.file }}</span>
              </div>
              <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
                {{ iss.description }}
              </p>
              <p class="text-xs font-mono text-amber-700 dark:text-amber-400 bg-amber-100/60 dark:bg-amber-900/30 p-2 rounded-lg">
                💡 Saran Perbaikan: {{ iss.remediation }}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
