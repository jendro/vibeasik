<script setup lang="ts">
// Penyiapan awal: 1) akun vibeasik (opsional), 2) Syarat & Ketentuan (wajib), 3) database.
// Studio mengarahkan semua halaman ke sini sampai selesai (middleware setup-gate).
useHead({ title: 'Penyiapan awal — vibeasik' })

interface SetupView {
  complete: boolean
  terms: { version: string, accepted: boolean, text: string }
  account: { choice: string, logged_in: boolean, email?: string }
  database: {
    current: { driver: string, name: string, host?: string, port?: string, user?: string }
    data_dir: string
    chosen: boolean
    requirements: { sqlite: string[], mysql: string[] }
  }
  restart_supported: boolean
}
interface DeviceCodeResp {
  device_code: string
  user_code: string
  verification_uri_complete: string
  interval: number
}
interface Report { ok: boolean, message: string, details?: string[] }

const { apiBase } = useApi()
const toast = useToast()
const gate = useState<{ complete: boolean } | null>('vibeasik-setup', () => null)
const edition = useEdition()
const { load: loadAccount } = useAuth()

const REGISTER_URL = 'https://vibeasik.com/register'

const view = ref<SetupView | null>(null)
const step = ref(1)
const busy = ref(false)
const loadError = ref('')

// Langkah 1
const pending = ref<DeviceCodeResp | null>(null)
let timer: ReturnType<typeof setTimeout> | null = null

// Langkah 2
const agreed = ref(false)

// Langkah 3
const db = reactive({
  driver: 'sqlite',
  name: 'vibeasik',
  host: '127.0.0.1',
  port: '3306',
  user: 'root',
  password: '',
  create_if_missing: true
})
const report = ref<Report | null>(null)
const needRestart = ref(false)
const restarting = ref(false)

function api<T>(path: string, body?: unknown) {
  return $fetch<T>(`${apiBase}/api/setup${path}`, body === undefined ? {} : { method: 'POST', body })
}

async function load() {
  try {
    view.value = await api<SetupView>('')
  } catch {
    loadError.value = 'Engine vibeasik tidak terjangkau. Jalankan `vibeasik serve`, lalu muat ulang halaman ini.'
    return
  }
  const v = view.value
  agreed.value = v.terms.accepted
  const c = v.database.current
  db.driver = c.driver === 'mysql' ? 'mysql' : 'sqlite'
  db.name = c.name || 'vibeasik'
  if (c.driver === 'mysql') {
    db.host = c.host || db.host
    db.port = c.port || db.port
    db.user = c.user || db.user
  }
  step.value = !v.account.choice && !v.account.logged_in ? 1 : !v.terms.accepted ? 2 : 3
}

onMounted(load)
onBeforeUnmount(() => timer && clearTimeout(timer))

const dbBody = () => ({
  driver: db.driver,
  name: db.name,
  host: db.host,
  port: db.port,
  user: db.user,
  password: db.password,
  create_if_missing: db.create_if_missing
})

const fail = (title: string, err: any) =>
  toast.add({ title, description: err?.data?.error || String(err?.message || err), color: 'error' })

// ---- Langkah 1: akun ----
async function chooseAccount(choice: 'register' | 'login' | 'skip') {
  try {
    await api('/account', { choice })
  } catch (err) {
    return fail('Gagal menyimpan pilihan', err)
  }
  if (choice === 'skip') step.value = 2
}

function openRegister() {
  window.open(REGISTER_URL, '_blank', 'noopener')
  chooseAccount('register')
}

async function poll() {
  if (!pending.value) return
  try {
    const r = await $fetch<{ status: string }>(`${apiBase}/api/account/login/poll`, {
      method: 'POST', body: { device_code: pending.value.device_code }
    })
    if (r.status === 'done') {
      pending.value = null
      await loadAccount()
      await edition.load(true)
      await load()
      step.value = 2
      toast.add({ title: 'Akun tersambung', color: 'success', icon: 'i-lucide-check' })
      return
    }
  } catch (err) {
    pending.value = null
    return fail('Masuk gagal', err)
  }
  timer = setTimeout(poll, Math.max(pending.value?.interval || 5, 2) * 1000)
}

async function startLogin() {
  busy.value = true
  try {
    await chooseAccount('login')
    pending.value = await $fetch<DeviceCodeResp>(`${apiBase}/api/account/login`, { method: 'POST' })
    window.open(pending.value.verification_uri_complete, '_blank', 'noopener')
    timer = setTimeout(poll, pending.value.interval * 1000)
  } catch (err) {
    fail('Server vibeasik tidak bisa dihubungi', err)
  } finally {
    busy.value = false
  }
}

// ---- Langkah 2: syarat ----
async function acceptTerms() {
  if (!view.value || !agreed.value) return
  busy.value = true
  try {
    view.value = await api<SetupView>('/terms', { version: view.value.terms.version, accept: true })
    step.value = 3
  } catch (err) {
    fail('Persetujuan gagal disimpan', err)
  } finally {
    busy.value = false
  }
}

// ---- Langkah 3: database ----
async function testDb() {
  busy.value = true
  report.value = null
  try {
    report.value = await api<Report>('/database/test', dbBody())
  } catch (err) {
    fail('Uji gagal', err)
  } finally {
    busy.value = false
  }
}

async function saveDb() {
  busy.value = true
  try {
    const r = await api<{ restart_required: boolean }>('/database', dbBody())
    await api('/complete', {})
    gate.value = { complete: true }
    if (r.restart_required) {
      needRestart.value = true
      step.value = 4
    } else {
      await navigateTo('/')
    }
  } catch (err: any) {
    if (err?.data?.report) report.value = err.data.report
    fail('Database belum bisa dipakai', err)
  } finally {
    busy.value = false
  }
}

async function restartEngine() {
  restarting.value = true
  try {
    await api('/restart', {})
  } catch (err) {
    restarting.value = false
    return fail('Tidak bisa memulai ulang otomatis', err)
  }
  // Tunggu engine hidup lagi.
  await new Promise(r => setTimeout(r, 1500))
  for (let i = 0; i < 40; i++) {
    try {
      const h = await $fetch<{ status: string }>(`${apiBase}/health`, { timeout: 1500 })
      if (h.status === 'ok') return navigateTo('/', { external: true })
    } catch { /* belum hidup */ }
    await new Promise(r => setTimeout(r, 1000))
  }
  restarting.value = false
  toast.add({ title: 'Engine belum kembali', description: 'Jalankan `vibeasik serve` secara manual.', color: 'warning' })
}

const steps = ['Akun', 'Ketentuan', 'Database']
</script>

<template>
  <div class="max-w-2xl mx-auto px-4 py-8 space-y-6">
    <div class="space-y-1">
      <UBadge color="primary" variant="subtle" icon="i-lucide-sparkles">
        Penyiapan awal
      </UBadge>
      <h1 class="text-2xl sm:text-3xl font-extrabold tracking-tight">
        Selamat datang di vibeasik
      </h1>
      <p class="text-sm text-neutral-600 dark:text-neutral-400">
        Tiga langkah singkat sebelum Cockpit terbuka. Semua data tetap di komputer ini.
      </p>
    </div>

    <UAlert v-if="loadError" color="error" variant="subtle" icon="i-lucide-plug-zap" :title="loadError" />

    <template v-else-if="view">
      <ol v-if="step <= 3" class="flex items-center gap-2 text-xs font-semibold" aria-label="Langkah">
        <li v-for="(s, i) in steps" :key="s" class="flex items-center gap-2">
          <span
            class="w-6 h-6 rounded-full flex items-center justify-center"
            :class="step > i + 1 ? 'bg-emerald-500 text-white' : step === i + 1 ? 'bg-sky-600 text-white' : 'bg-neutral-200 dark:bg-neutral-800 text-neutral-500'"
          >
            <UIcon v-if="step > i + 1" name="i-lucide-check" class="w-3.5 h-3.5" />
            <template v-else>{{ i + 1 }}</template>
          </span>
          <span :class="step === i + 1 ? 'text-neutral-900 dark:text-white' : 'text-neutral-500'">{{ s }}</span>
          <span v-if="i < steps.length - 1" class="w-6 h-px bg-neutral-300 dark:bg-neutral-700" />
        </li>
      </ol>

      <!-- 1. Akun -->
      <UCard v-if="step === 1">
        <div class="space-y-4">
          <div>
            <h2 class="text-lg font-bold">
              Akun vibeasik
            </h2>
            <p class="text-sm text-neutral-600 dark:text-neutral-400">
              Daftar atau masuk dengan akun gratis untuk menarik paket Pusat Pengetahuan dan membuka fitur sesuai paket.
              Langkah ini boleh dilewati; fitur lokal tetap jalan tanpa akun.
            </p>
          </div>

          <div v-if="view.account.logged_in" class="rounded-lg border border-emerald-500/40 bg-emerald-500/5 p-3 text-sm flex items-center gap-2">
            <UIcon name="i-lucide-user-round-check" class="w-5 h-5 text-emerald-500" />
            Masuk sebagai <b>{{ view.account.email }}</b>
          </div>

          <div v-else-if="pending" class="rounded-lg border border-sky-300 dark:border-sky-700 p-4 space-y-2 text-center">
            <p class="text-sm">
              Masukkan kode ini di halaman yang terbuka:
            </p>
            <div class="text-3xl font-mono font-bold tracking-widest text-sky-600 dark:text-sky-400">
              {{ pending.user_code }}
            </div>
            <p class="text-xs text-neutral-500">
              Menunggu persetujuan…
              <a :href="pending.verification_uri_complete" target="_blank" rel="noopener" class="underline">Buka lagi halamannya</a>
            </p>
          </div>

          <div v-else class="grid gap-2 sm:grid-cols-2">
            <UButton size="lg" icon="i-lucide-user-plus" @click="openRegister">
              Daftar akun gratis
            </UButton>
            <UButton size="lg" color="neutral" variant="outline" icon="i-lucide-log-in" :loading="busy" @click="startLogin">
              Saya sudah punya akun
            </UButton>
          </div>
          <p v-if="!view.account.logged_in && !pending && view.account.choice === 'register'" class="text-xs text-neutral-500">
            Halaman pendaftaran vibeasik.com terbuka di tab baru. Setelah akun jadi, klik <b>Saya sudah punya akun</b> untuk menyambungkannya ke aplikasi ini.
          </p>

          <div class="flex items-center justify-between pt-1">
            <span />
            <UButton
              v-if="view.account.logged_in"
              trailing-icon="i-lucide-arrow-right"
              @click="step = 2"
            >
              Lanjut
            </UButton>
            <UButton v-else color="neutral" variant="link" @click="chooseAccount('skip')">
              Lewati, lanjut tanpa akun
            </UButton>
          </div>
        </div>
      </UCard>

      <!-- 2. Syarat -->
      <UCard v-else-if="step === 2">
        <div class="space-y-4">
          <div>
            <h2 class="text-lg font-bold">
              Syarat &amp; Ketentuan
            </h2>
            <p class="text-sm text-neutral-600 dark:text-neutral-400">
              Wajib disetujui untuk memakai Cockpit. Persetujuan ini sudah mencakup ketentuan Pusat Pengetahuan lokal.
            </p>
          </div>
          <pre
            class="max-h-72 overflow-auto whitespace-pre-wrap rounded-lg bg-neutral-100 dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 p-3 text-xs leading-relaxed"
            tabindex="0"
            aria-label="Ringkasan Syarat dan Ketentuan"
          >{{ view.terms.text }}</pre>
          <div class="flex flex-wrap gap-x-4 gap-y-1 text-sm">
            <a href="https://vibeasik.com/terms" target="_blank" rel="noopener" class="text-sky-600 dark:text-sky-400 underline">Syarat &amp; Ketentuan lengkap</a>
            <a href="https://vibeasik.com/legal/knowledge-license" target="_blank" rel="noopener" class="text-sky-600 dark:text-sky-400 underline">Lisensi Pengetahuan</a>
            <a href="https://github.com/vibeasik/vibeasik" target="_blank" rel="noopener" class="text-sky-600 dark:text-sky-400 underline">Kode sumber (Apache-2.0)</a>
          </div>
          <UCheckbox
            v-model="agreed"
            label="Saya telah membaca dan menyetujui Syarat & Ketentuan vibeasik, termasuk ketentuan Pusat Pengetahuan."
          />
          <div class="flex items-center justify-between">
            <UButton color="neutral" variant="ghost" icon="i-lucide-arrow-left" @click="step = 1">
              Kembali
            </UButton>
            <UButton :disabled="!agreed" :loading="busy" trailing-icon="i-lucide-arrow-right" @click="acceptTerms">
              Setuju &amp; lanjut
            </UButton>
          </div>
        </div>
      </UCard>

      <!-- 3. Database -->
      <UCard v-else-if="step === 3">
        <div class="space-y-4">
          <div>
            <h2 class="text-lg font-bold">
              Database
            </h2>
            <p class="text-sm text-neutral-600 dark:text-neutral-400">
              Tempat vibeasik menyimpan proyek, todo, dan pengetahuan Anda. Dapat diubah nanti lewat
              <code class="font-mono">{{ view.database.data_dir }}/.env</code>.
            </p>
          </div>

          <div class="grid gap-2 sm:grid-cols-2" role="radiogroup" aria-label="Jenis database">
            <button
              v-for="opt in [{ id: 'sqlite', title: 'SQLite', sub: 'Berkas lokal, tanpa instalasi (disarankan)' }, { id: 'mysql', title: 'MySQL / MariaDB', sub: 'Server database sendiri' }]"
              :key="opt.id"
              type="button"
              role="radio"
              :aria-checked="db.driver === opt.id"
              class="text-left rounded-lg border p-3 transition-colors"
              :class="db.driver === opt.id ? 'border-sky-500 bg-sky-500/5' : 'border-neutral-200 dark:border-neutral-800 hover:border-sky-500/50'"
              @click="db.driver = opt.id; report = null"
            >
              <div class="font-semibold">
                {{ opt.title }}
              </div>
              <div class="text-xs text-neutral-500">
                {{ opt.sub }}
              </div>
            </button>
          </div>

          <div class="rounded-lg bg-neutral-100 dark:bg-neutral-900 p-3">
            <div class="text-xs font-semibold uppercase tracking-wide text-neutral-500 mb-1">
              Persyaratan
            </div>
            <ul class="list-disc pl-5 space-y-1 text-sm text-neutral-700 dark:text-neutral-300">
              <li v-for="r in view.database.requirements[db.driver as 'sqlite' | 'mysql']" :key="r">
                {{ r }}
              </li>
            </ul>
          </div>

          <div class="grid gap-3 sm:grid-cols-2">
            <label class="block sm:col-span-2">
              <span class="text-sm font-medium">Nama database</span>
              <UInput v-model="db.name" class="w-full" placeholder="vibeasik" autocomplete="off" @update:model-value="report = null" />
              <span class="text-xs text-neutral-500">
                <template v-if="db.driver === 'sqlite'">Disimpan sebagai <code class="font-mono">{{ view.database.data_dir }}/{{ db.name || 'nama' }}.db</code>. </template>
                Huruf, angka, dan garis bawah; diawali huruf.
              </span>
            </label>
            <template v-if="db.driver === 'mysql'">
              <label class="block">
                <span class="text-sm font-medium">Host</span>
                <UInput v-model="db.host" class="w-full" autocomplete="off" />
              </label>
              <label class="block">
                <span class="text-sm font-medium">Port</span>
                <UInput v-model="db.port" class="w-full" autocomplete="off" />
              </label>
              <label class="block">
                <span class="text-sm font-medium">User</span>
                <UInput v-model="db.user" class="w-full" autocomplete="off" />
              </label>
              <label class="block">
                <span class="text-sm font-medium">Kata sandi</span>
                <UInput v-model="db.password" type="password" class="w-full" autocomplete="new-password" />
              </label>
              <div class="sm:col-span-2">
                <UCheckbox v-model="db.create_if_missing" label="Buat database otomatis bila belum ada" />
              </div>
            </template>
          </div>

          <UAlert
            v-if="report"
            :color="report.ok ? 'success' : 'error'"
            variant="subtle"
            :icon="report.ok ? 'i-lucide-circle-check' : 'i-lucide-circle-alert'"
            :title="report.message"
          >
            <template v-if="report.details?.length" #description>
              <ul class="list-disc pl-4">
                <li v-for="d in report.details" :key="d">
                  {{ d }}
                </li>
              </ul>
            </template>
          </UAlert>

          <div class="flex flex-wrap items-center justify-between gap-2">
            <UButton color="neutral" variant="ghost" icon="i-lucide-arrow-left" @click="step = 2">
              Kembali
            </UButton>
            <div class="flex gap-2">
              <UButton color="neutral" variant="outline" icon="i-lucide-plug-zap" :loading="busy" @click="testDb">
                Uji koneksi
              </UButton>
              <UButton :loading="busy" icon="i-lucide-check" @click="saveDb">
                Simpan &amp; selesai
              </UButton>
            </div>
          </div>
        </div>
      </UCard>

      <!-- 4. Mulai ulang -->
      <UCard v-else>
        <div class="space-y-4">
          <div class="flex items-center gap-2 text-emerald-600 dark:text-emerald-400">
            <UIcon name="i-lucide-circle-check" class="w-6 h-6" />
            <h2 class="text-lg font-bold">
              Pengaturan tersimpan
            </h2>
          </div>
          <p class="text-sm text-neutral-600 dark:text-neutral-400">
            Pilihan database baru berlaku setelah engine dimulai ulang. Data di database sebelumnya tidak dipindahkan.
          </p>
          <UButton v-if="view.restart_supported" size="lg" icon="i-lucide-rotate-cw" :loading="restarting" @click="restartEngine">
            Mulai ulang engine &amp; buka Cockpit
          </UButton>
          <UAlert
            v-else
            color="warning"
            variant="subtle"
            icon="i-lucide-terminal"
            title="Mulai ulang manual"
            description="Hentikan engine (Ctrl+C), lalu jalankan `vibeasik serve` lagi dan buka `vibeasik open`."
          />
        </div>
      </UCard>
    </template>
  </div>
</template>
