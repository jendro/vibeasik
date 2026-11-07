<script setup lang="ts">
useHead({ title: 'Beranda — vibeasik' })

const { apiBase } = useApi()
const { info, load: loadEdition } = useEdition()
const toast = useToast()

const engineOk = ref<boolean | null>(null)
const projectCount = ref<number | null>(null)

onMounted(async () => {
  loadEdition()
  try {
    const h = await $fetch<{ status: string }>(`${apiBase}/health`, { timeout: 2000 })
    engineOk.value = h.status === 'ok'
  } catch {
    engineOk.value = false
  }
  try {
    const list = await $fetch<unknown[]>(`${apiBase}/api/projects`, { timeout: 3000 })
    projectCount.value = Array.isArray(list) ? list.length : 0
  } catch {
    projectCount.value = null
  }
})

const edisi = computed(() => info.value?.edition || '—')
const host = computed(() => (import.meta.client ? window.location.host : 'localhost'))

const status = computed(() => [
  {
    icon: 'i-lucide-server',
    label: 'Engine lokal',
    value: engineOk.value === null ? 'memeriksa…' : engineOk.value ? 'Berjalan' : 'Tidak terjangkau',
    hint: host.value,
    tone: engineOk.value === false ? 'text-rose-500' : 'text-emerald-500'
  },
  {
    icon: 'i-lucide-folder-git-2',
    label: 'Proyek terdaftar',
    value: projectCount.value === null ? '—' : String(projectCount.value),
    hint: 'Di komputer ini',
    tone: 'text-sky-500'
  },
  {
    icon: 'i-lucide-package-check',
    label: 'Edisi',
    value: edisi.value,
    hint: 'Fitur Pro terkunci sampai masuk akun',
    tone: 'text-violet-500'
  }
])

const steps = [
  {
    title: 'Daftarkan proyek Anda',
    text: 'Klik Daftarkan proyek (atau Import), lalu pilih folder proyek. vibeasik mengenali backend, frontend, dan stack-nya.',
    cmd: 'vibeasik scan ~/proyek-saya'
  },
  {
    title: 'Jalankan audit',
    text: 'Buka Audit & Diagnostik dan periksa temuan keamanan, arsitektur, dan API. Setiap temuan menjelaskan apa yang salah dan cara memperbaikinya.',
    cmd: 'vibeasik diagnose ~/proyek-saya'
  },
  {
    title: 'Atur port tanpa bentrok',
    text: 'Setiap proyek mendapat blok port sendiri. Lihat dan kelola di halaman Port; jangan memilih port manual.',
    cmd: 'vibeasik ports'
  },
  {
    title: 'Sambungkan agent coding',
    text: 'Daftarkan vibeasik sebagai server MCP ke Claude Code, Cursor, Windsurf, atau Antigravity agar agent memakai konteks dan todo proyek.',
    cmd: 'vibeasik mcp install'
  },
  {
    title: 'Ikut Pusat Pengetahuan (opsional)',
    text: 'Masuk dengan akun gratis untuk menarik paket panduan dan checklist. Berbagi pengetahuan dapat dilihat dan diatur di halaman Berbagi Pengetahuan.',
    cmd: 'vibeasik login'
  }
]

const features = [
  { icon: 'i-lucide-search-check', title: 'Audit kode buatan AI', text: '85 rule dalam 12 kategori: arsitektur, keamanan, API, database, deployment.', to: '/audit-report' },
  { icon: 'i-lucide-layout-grid', title: 'Semua proyek di satu tempat', text: 'Stack, perintah dev, dan todo per proyek yang bisa dibaca agent.', to: '/cockpit' },
  { icon: 'i-lucide-plug', title: 'Port tanpa bentrok', text: 'Blok port per proyek. Tidak ada lagi dua dev server berebut port 3000.', to: '/ports' },
  { icon: 'i-lucide-bot', title: 'Server MCP untuk agent', text: 'Satu perintah pemasangan untuk agent coding favorit Anda.', to: '/mcp' },
  { icon: 'i-lucide-map', title: 'Peta kode & rute', text: 'Konteks ringkas tentang struktur kode agar agent tidak menebak-nebak.', to: '/code-map' },
  { icon: 'i-lucide-library', title: 'Pusat Pengetahuan', text: 'Panduan dan checklist yang tumbuh bersama pengguna; pencarian tetap lokal.', to: '/knowledge' }
]

const privacy = [
  'Kode, fakta proyek, dan kunci API tetap di komputer Anda dan tidak diunggah.',
  'Engine hanya mendengarkan 127.0.0.1 dan wajib memakai token lokal.',
  'Yang dikirim ke Pusat Pengetahuan hanya pokok ilmu yang lolos penyaringan, dengan pratinjau dan riwayat yang bisa Anda lihat.'
]

async function copy(cmd: string) {
  try {
    await navigator.clipboard.writeText(cmd)
    toast.add({ title: 'Perintah disalin', description: cmd, color: 'success' })
  } catch {
    toast.add({ title: 'Tidak bisa menyalin', description: 'Salin manual dari kotak perintah.', color: 'warning' })
  }
}
</script>

<template>
  <div class="max-w-5xl mx-auto space-y-10 pb-10">
    <!-- Hero -->
    <section class="space-y-4 pt-2">
      <UBadge color="primary" variant="subtle" size="md" icon="i-lucide-laptop">
        Berjalan di komputer Anda · lokal-dulu
      </UBadge>
      <h1 class="text-3xl sm:text-4xl font-extrabold tracking-tight text-neutral-900 dark:text-white">
        Standar untuk kode<br class="hidden sm:block"> buatan AI
      </h1>
      <p class="max-w-2xl text-neutral-600 dark:text-neutral-300">
        vibeasik mengaudit kode hasil AI, mengatur proyek dan port, dan menjadi server MCP untuk agent coding favorit Anda.
        Kode Anda tidak pernah diunggah.
      </p>
      <div class="flex flex-wrap gap-2 pt-1">
        <UButton to="/cockpit" icon="i-lucide-layout-grid" size="lg">
          Buka Cockpit
        </UButton>
        <UButton to="/scan" icon="i-lucide-plus" color="neutral" variant="outline" size="lg">
          Daftarkan proyek
        </UButton>
        <UButton to="#tutorial" icon="i-lucide-book-open" color="neutral" variant="ghost" size="lg">
          Cara memakai
        </UButton>
      </div>
    </section>

    <!-- Status mesin -->
    <section class="grid grid-cols-1 sm:grid-cols-3 gap-3" aria-label="Status">
      <div
        v-for="s in status"
        :key="s.label"
        class="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-4"
      >
        <div class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-neutral-500">
          <UIcon :name="s.icon" class="w-4 h-4" />
          {{ s.label }}
        </div>
        <div class="mt-2 text-2xl font-bold" :class="s.tone">
          {{ s.value }}
        </div>
        <div class="text-xs text-neutral-500 mt-0.5">
          {{ s.hint }}
        </div>
      </div>
    </section>

    <!-- Tutorial -->
    <section id="tutorial" class="scroll-mt-20 space-y-4">
      <div>
        <div class="text-xs font-mono font-semibold uppercase tracking-wider text-sky-600 dark:text-sky-400">
          Tutorial
        </div>
        <h2 class="text-2xl font-bold text-neutral-900 dark:text-white">
          Cara menggunakan vibeasik
        </h2>
        <p class="text-sm text-neutral-600 dark:text-neutral-400">
          Lima langkah dari proyek pertama sampai agent Anda memakainya. Perintah di bawah dijalankan di terminal; semuanya juga bisa lewat Studio ini.
        </p>
      </div>
      <ol class="space-y-3">
        <li
          v-for="(s, i) in steps"
          :key="s.title"
          class="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-4 flex gap-4"
        >
          <div class="shrink-0 w-8 h-8 rounded-full bg-sky-600 text-white text-sm font-bold flex items-center justify-center">
            {{ i + 1 }}
          </div>
          <div class="min-w-0 flex-1 space-y-2">
            <h3 class="font-semibold text-neutral-900 dark:text-white">
              {{ s.title }}
            </h3>
            <p class="text-sm text-neutral-600 dark:text-neutral-400">
              {{ s.text }}
            </p>
            <button
              type="button"
              class="w-full text-left flex items-center justify-between gap-3 rounded-lg bg-neutral-950 text-neutral-100 px-3 py-2 font-mono text-xs hover:bg-neutral-900"
              :aria-label="`Salin perintah ${s.cmd}`"
              @click="copy(s.cmd)"
            >
              <span class="truncate"><span class="text-sky-400">$ </span>{{ s.cmd }}</span>
              <UIcon name="i-lucide-copy" class="w-3.5 h-3.5 shrink-0 text-neutral-400" />
            </button>
          </div>
        </li>
      </ol>
    </section>

    <!-- Fitur -->
    <section class="space-y-4">
      <div>
        <div class="text-xs font-mono font-semibold uppercase tracking-wider text-sky-600 dark:text-sky-400">
          Fitur
        </div>
        <h2 class="text-2xl font-bold text-neutral-900 dark:text-white">
          Yang dikerjakan vibeasik untuk Anda
        </h2>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        <NuxtLink
          v-for="f in features"
          :key="f.title"
          :to="f.to"
          class="group rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-4 space-y-2 hover:border-sky-500/50 transition-colors"
        >
          <div class="w-9 h-9 rounded-lg bg-sky-500/10 text-sky-600 dark:text-sky-400 flex items-center justify-center">
            <UIcon :name="f.icon" class="w-5 h-5" />
          </div>
          <h3 class="font-semibold text-neutral-900 dark:text-white group-hover:text-sky-600 dark:group-hover:text-sky-400">{{ f.title }}</h3>
          <p class="text-sm text-neutral-600 dark:text-neutral-400">{{ f.text }}</p>
        </NuxtLink>
      </div>
    </section>

    <!-- Privasi -->
    <section class="rounded-xl border border-sky-500/30 bg-sky-500/5 p-5 space-y-3">
      <h2 class="text-lg font-bold text-neutral-900 dark:text-white flex items-center gap-2">
        <UIcon name="i-lucide-shield-check" class="w-5 h-5 text-sky-500" />
        Lokal-dulu, dengan sengaja
      </h2>
      <ul class="space-y-1.5 text-sm text-neutral-700 dark:text-neutral-300 list-disc pl-5">
        <li v-for="p in privacy" :key="p">
          {{ p }}
        </li>
      </ul>
      <div class="flex flex-wrap gap-2 pt-1">
        <UButton to="/knowledge-sync" size="sm" color="neutral" variant="outline" icon="i-lucide-hand-heart">
          Berbagi Pengetahuan
        </UButton>
        <UButton to="https://vibeasik.com/terms" target="_blank" size="sm" color="neutral" variant="ghost" icon="i-lucide-file-text">
          Syarat &amp; Ketentuan
        </UButton>
        <UButton to="https://github.com/vibeasik/vibeasik" target="_blank" size="sm" color="neutral" variant="ghost" icon="i-lucide-github">
          GitHub
        </UButton>
      </div>
    </section>
  </div>
</template>
