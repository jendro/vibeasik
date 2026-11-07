<script setup lang="ts">
definePageMeta({
  public: true
})

const toast = useToast()

interface DownloadItem {
  id: string
  name: string
  filename: string
  platform: 'linux-deb' | 'linux-appimage' | 'windows' | 'linux-tar' | 'windows-zip' | 'cli'
  platformName: string
  badge: string
  version: string
  size: string
  icon: string
  downloadUrl: string
  instruction: string
  checksum: string
  popular?: boolean
}

const currentVersion = 'v1.0.0'
const releaseDate = '21 Oktober 2026'

const downloads: DownloadItem[] = [
  {
    id: 'linux-deb',
    name: 'vibeasik Desktop (Debian / Ubuntu)',
    filename: 'vibeasik-v1.0.0-amd64.deb',
    platform: 'linux-deb',
    platformName: 'Ubuntu / Debian / Linux Mint',
    badge: 'Installer Resmi .deb',
    version: currentVersion,
    size: '102 MB',
    icon: 'i-simple-icons-ubuntu',
    downloadUrl: '/downloads/vibeasik-v1.0.0-amd64.deb',
    instruction: 'Double-click file .deb lalu klik tombol "Install", atau jalankan: sudo dpkg -i vibeasik-v1.0.0-amd64.deb',
    checksum: 'c88be00d0c87260278800f1fef27b3d819ac34ec8d5be27d44cd0fb378d4b677',
    popular: true
  },
  {
    id: 'linux-appimage',
    name: 'vibeasik Desktop (Universal Linux)',
    filename: 'vibeasik-v1.0.0.AppImage',
    platform: 'linux-appimage',
    platformName: 'Semua Distro Linux',
    badge: 'Portable AppImage',
    version: currentVersion,
    size: '128 MB',
    icon: 'i-simple-icons-linux',
    downloadUrl: '/downloads/vibeasik-v1.0.0.AppImage',
    instruction: 'Beri izin eksekusi (chmod +x vibeasik-v1.0.0.AppImage), lalu double-click untuk langsung menjalankan tanpa perlu install.',
    checksum: '84f6baa1037da6076d6fe7c78e8d824c941df0b1a9e04fea1be3717e4a9218f7'
  },
  {
    id: 'windows-exe',
    name: 'vibeasik Desktop (Windows 64-bit)',
    filename: 'vibeasik-v1.0.0-windows-x64.exe',
    platform: 'windows',
    platformName: 'Windows 10 / 11 (64-bit)',
    badge: 'Portable Executable .exe',
    version: currentVersion,
    size: '101 MB',
    icon: 'i-simple-icons-windows',
    downloadUrl: '/downloads/vibeasik-v1.0.0-windows-x64.exe',
    instruction: 'Double-click berkas .exe untuk membuka Studio Cockpit secara instan tanpa instalasi runtime manual.',
    checksum: '2d1edb7d10518182627e269dd1fbf9645125b937a7308f790e1fbef35d81688d',
    popular: true
  },
  {
    id: 'linux-tar',
    name: 'vibeasik CLI & Engine (Linux)',
    filename: 'vibeasik-v1.0.0-linux-amd64.tar.gz',
    platform: 'linux-tar',
    platformName: 'Linux Server & Terminal CLI',
    badge: 'Standalone Binaries',
    version: currentVersion,
    size: '15 MB',
    icon: 'i-lucide-terminal',
    downloadUrl: '/downloads/vibeasik-v1.0.0-linux-amd64.tar.gz',
    instruction: 'Ekstrak arsip: tar -xzf vibeasik-v1.0.0-linux-amd64.tar.gz && ./install.sh untuk mendaftarkan MCP hub.',
    checksum: '5aee7f95cfe6a29a175492ab95bca34001f0d12aaf6a260b21939946d7a57ad9'
  },
  {
    id: 'windows-zip',
    name: 'vibeasik CLI & Engine (Windows)',
    filename: 'vibeasik-v1.0.0-windows-amd64.zip',
    platform: 'windows-zip',
    platformName: 'Windows Terminal CLI',
    badge: 'Binary Archive .zip',
    version: currentVersion,
    size: '15 MB',
    icon: 'i-lucide-file-archive',
    downloadUrl: '/downloads/vibeasik-v1.0.0-windows-amd64.zip',
    instruction: 'Ekstrak arsip .zip dan jalankan vibeasik.exe via PowerShell atau Command Prompt.',
    checksum: '64cdfb40fb1de7d8868a1284ce08a7ce3cf0285e7653f9563cd5b32d8cf98d7c'
  }
]

// Client OS Detection
const detectedOS = ref<{
  name: string
  key: 'linux-deb' | 'linux-appimage' | 'windows' | 'mac' | 'unknown'
  rawInfo: string
  recommendedItem: DownloadItem | null
}>({
  name: 'Mendeteksi sistem operasi...',
  key: 'unknown',
  rawInfo: '',
  recommendedItem: null
})

const detectSystem = () => {
  if (import.meta.server) return

  const ua = navigator.userAgent.toLowerCase()
  const platform = (navigator as any).userAgentData?.platform?.toLowerCase() || navigator.platform?.toLowerCase() || ''

  let key: 'linux-deb' | 'linux-appimage' | 'windows' | 'mac' | 'unknown' = 'unknown'
  let name = 'Sistem Operasi Anda'

  if (ua.includes('win') || platform.includes('win')) {
    key = 'windows'
    name = 'Windows 10 / 11 (64-bit)'
  } else if (ua.includes('mac') || platform.includes('mac')) {
    key = 'mac'
    name = 'macOS (Apple Silicon & Intel)'
  } else if (ua.includes('linux') || platform.includes('linux') || ua.includes('x11')) {
    // Check if Ubuntu / Debian derivative is likely
    if (ua.includes('ubuntu') || ua.includes('debian') || ua.includes('mint') || ua.includes('pop')) {
      key = 'linux-deb'
      name = 'Linux (Ubuntu / Debian / Mint 64-bit)'
    } else {
      key = 'linux-appimage'
      name = 'Linux (Universal 64-bit)'
    }
  }

  let recommendedItem: DownloadItem | null = null
  if (key === 'windows') {
    recommendedItem = downloads.find(d => d.id === 'windows-exe') || null
  } else if (key === 'linux-deb') {
    recommendedItem = downloads.find(d => d.id === 'linux-deb') || null
  } else if (key === 'linux-appimage') {
    recommendedItem = downloads.find(d => d.id === 'linux-appimage') || null
  } else {
    // Fallback default recommendation
    recommendedItem = downloads[0] || null
  }

  detectedOS.value = {
    name,
    key,
    rawInfo: platform || navigator.userAgent,
    recommendedItem
  }
}

onMounted(() => {
  detectSystem()
})

const copiedChecksumId = ref<string | null>(null)
const copyChecksum = (item: DownloadItem) => {
  if (import.meta.client) {
    navigator.clipboard.writeText(item.checksum)
    copiedChecksumId.value = item.id
    toast.add({
      title: 'Checksum Tersalin',
      description: `SHA-256 untuk ${item.filename} berhasil disalin ke clipboard.`,
      icon: 'i-lucide-check',
      color: 'success'
    })
    setTimeout(() => {
      copiedChecksumId.value = null
    }, 2500)
  }
}

const copiedCommand = ref(false)
const cliInstallCommand = 'curl -fsSL https://vibeasik.com/install.sh | bash'
const copyCliCommand = () => {
  if (import.meta.client) {
    navigator.clipboard.writeText(cliInstallCommand)
    copiedCommand.value = true
    toast.add({
      title: 'Perintah CLI Tersalin',
      description: 'Tempel di terminal Anda untuk menjalankan installer otomatis.',
      icon: 'i-lucide-check',
      color: 'success'
    })
    setTimeout(() => {
      copiedCommand.value = false
    }, 2500)
  }
}

useHead({
  title: 'Download vibeasik Desktop - Aplikasi Khusus Vibecoders',
  meta: [
    {
      name: 'description',
      content: 'Unduh aplikasi desktop vibeasik untuk Linux dan Windows. Memastikan kode buatan AI aman, andal, bebas celah keamanan, dan mudah dimaintenance.'
    }
  ]
})
</script>

<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 sm:py-12 space-y-12">
    <!-- Header Hero Section: Misi untuk Vibecoders -->
    <div class="relative overflow-hidden rounded-2xl bg-neutral-900 border border-neutral-800 text-white p-8 sm:p-12">
      <!-- Ambient light effect -->
      <div class="absolute -top-24 -right-24 w-96 h-96 rounded-full bg-sky-500/10 blur-3xl pointer-events-none" />
      <div class="absolute -bottom-24 -left-24 w-96 h-96 rounded-full bg-indigo-500/10 blur-3xl pointer-events-none" />

      <div class="relative z-10 max-w-3xl space-y-4">
        <div class="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-medium bg-sky-500/20 text-sky-300 border border-sky-500/30">
          <UIcon name="i-lucide-shield-check" class="w-3.5 h-3.5" />
          <span>Aplikasi Khusus Vibecoders • Rilis Versi {{ currentVersion }}</span>
        </div>

        <h1 class="text-3xl sm:text-4xl lg:text-5xl font-extrabold tracking-tight text-white leading-tight">
          Pastikan Kode Buatan AI Anda <span class="text-sky-400">Aman & Siap Maintenance</span>
        </h1>

        <p class="text-neutral-300 text-base sm:text-lg leading-relaxed">
          Dirancang untuk menjembatani kode hasil prompting LLM (Antigravity, Cursor, Claude Code, Windsurf).
          Otomatis mendeteksi celah keamanan, mencegah port bentrok, membersihkan *code smells*, dan menanamkan
          <strong>20 MCP Tools</strong> langsung ke AI agent di laptop Anda.
        </p>

        <!-- Quick Spec Tags -->
        <div class="pt-2 flex flex-wrap gap-4 text-xs text-neutral-400 font-mono">
          <div class="flex items-center gap-1.5">
            <UIcon name="i-lucide-cpu" class="w-4 h-4 text-sky-400" />
            <span>Electron Native Desktop (Teknologi Antigravity)</span>
          </div>
          <div class="flex items-center gap-1.5">
            <UIcon name="i-lucide-zap" class="w-4 h-4 text-sky-400" />
            <span>Embedded Go Engine & SQLite</span>
          </div>
          <div class="flex items-center gap-1.5">
            <UIcon name="i-lucide-calendar" class="w-4 h-4 text-sky-400" />
            <span>Rilis: {{ releaseDate }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Smart Auto-Detection Recommendation Card -->
    <div class="rounded-xl border-2 border-sky-500/30 dark:border-sky-500/40 bg-white dark:bg-slate-900/40 shadow-sm p-6 sm:p-8 relative">
      <div class="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div class="space-y-2">
          <div class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-sky-600 dark:text-sky-400">
            <UIcon name="i-lucide-sparkles" class="w-4 h-4" />
            <span>Rekomendasi Berdasarkan Komputer Anda</span>
          </div>

          <div class="flex items-center gap-3">
            <UIcon
              :name="detectedOS.recommendedItem?.icon || 'i-lucide-laptop'"
              class="w-8 h-8 text-neutral-800 dark:text-neutral-200"
            />
            <div>
              <h2 class="text-xl sm:text-2xl font-bold text-neutral-900 dark:text-white">
                Download untuk {{ detectedOS.name }}
              </h2>
              <p class="text-sm text-neutral-500 dark:text-neutral-400">
                Berkas: <span class="font-mono text-neutral-700 dark:text-neutral-300">{{ detectedOS.recommendedItem?.filename }}</span>
                ({{ detectedOS.recommendedItem?.size }})
              </p>
            </div>
          </div>

          <p class="text-xs text-neutral-600 dark:text-neutral-300 pt-1">
            {{ detectedOS.recommendedItem?.instruction }}
          </p>
        </div>

        <!-- Download Action -->
        <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 shrink-0">
          <a
            :href="detectedOS.recommendedItem?.downloadUrl"
            download
            class="inline-flex items-center justify-center gap-2.5 px-6 py-3.5 rounded-xl font-semibold text-white bg-sky-600 hover:bg-sky-500 active:bg-sky-700 transition-colors shadow-md shadow-sky-600/20 text-center"
          >
            <UIcon name="i-lucide-download" class="w-5 h-5" />
            <span>Unduh Sekarang ({{ detectedOS.recommendedItem?.size }})</span>
          </a>

          <button
            type="button"
            class="inline-flex items-center justify-center gap-2 px-4 py-3.5 rounded-xl border border-neutral-300 dark:border-neutral-700 hover:bg-neutral-100 dark:hover:bg-neutral-800 text-neutral-700 dark:text-neutral-300 text-xs font-mono transition-colors"
            :title="`Salin Checksum SHA-256: ${detectedOS.recommendedItem?.checksum}`"
            @click="detectedOS.recommendedItem && copyChecksum(detectedOS.recommendedItem)"
          >
            <UIcon
              :name="copiedChecksumId === detectedOS.recommendedItem?.id ? 'i-lucide-check' : 'i-lucide-copy'"
              class="w-4 h-4 text-sky-500"
            />
            <span>{{ copiedChecksumId === detectedOS.recommendedItem?.id ? 'Tersalin!' : 'SHA-256' }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Complete Downloads Catalog (Semua Platform & Format) -->
    <div class="space-y-6">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-neutral-200 dark:border-white/10 pb-4">
        <div>
          <h2 class="text-xl sm:text-2xl font-bold text-neutral-900 dark:text-white">
            Semua Pilihan Berkas & Platform
          </h2>
          <p class="text-sm text-neutral-500 dark:text-neutral-400">
            Seluruh berkas telah diverifikasi dengan versioning resmi <strong class="text-neutral-800 dark:text-neutral-200">{{ currentVersion }}</strong>.
          </p>
        </div>
        <div class="text-xs font-mono text-neutral-500">
          Total 5 Format Tersedia
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
        <div
          v-for="item in downloads"
          :key="item.id"
          class="flex flex-col justify-between rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 p-5 hover:border-sky-500/50 transition-colors shadow-xs"
        >
          <div class="space-y-3">
            <div class="flex items-start justify-between gap-2">
              <div class="flex items-center gap-2.5">
                <div class="w-10 h-10 rounded-lg bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center text-neutral-700 dark:text-neutral-300">
                  <UIcon :name="item.icon" class="w-5 h-5" />
                </div>
                <div>
                  <h3 class="font-bold text-neutral-900 dark:text-white text-sm">
                    {{ item.platformName }}
                  </h3>
                  <span class="inline-block text-[11px] font-medium text-sky-600 dark:text-sky-400">
                    {{ item.badge }}
                  </span>
                </div>
              </div>
              <span class="text-xs font-mono px-2 py-0.5 rounded-md bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400">
                {{ item.size }}
              </span>
            </div>

            <div class="space-y-1">
              <div class="text-xs font-mono text-neutral-700 dark:text-neutral-300 break-all select-all">
                {{ item.filename }}
              </div>
              <p class="text-xs text-neutral-500 dark:text-neutral-400 leading-relaxed">
                {{ item.instruction }}
              </p>
            </div>
          </div>

          <!-- Bottom Actions: Download + Checksum -->
          <div class="pt-4 border-t border-neutral-100 dark:border-white/10 mt-4 space-y-2">
            <a
              :href="item.downloadUrl"
              download
              class="w-full inline-flex items-center justify-center gap-2 py-2.5 px-4 rounded-lg bg-neutral-900 dark:bg-white text-white dark:text-neutral-900 hover:bg-neutral-800 dark:hover:bg-neutral-100 text-xs font-semibold transition-colors"
            >
              <UIcon name="i-lucide-download" class="w-4 h-4" />
              <span>Download {{ item.version }}</span>
            </a>

            <button
              type="button"
              class="w-full flex items-center justify-between px-2.5 py-1.5 rounded-md text-[11px] font-mono text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors"
              :title="`Klik untuk menyalin SHA-256: ${item.checksum}`"
              @click="copyChecksum(item)"
            >
              <span class="truncate pr-2">SHA-256: {{ item.checksum.substring(0, 16) }}...</span>
              <span class="text-sky-600 dark:text-sky-400 font-sans font-medium shrink-0">
                {{ copiedChecksumId === item.id ? 'Tersalin!' : 'Salin' }}
              </span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Terminal One-Liner (Untuk Developer yang Suka CLI) -->
    <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-neutral-900 text-white p-6 sm:p-8 space-y-4">
      <div class="flex items-center gap-2 text-sky-400 text-xs font-semibold uppercase tracking-wider">
        <UIcon name="i-lucide-terminal" class="w-4 h-4" />
        <span>One-Click CLI Installer (Linux & macOS)</span>
      </div>

      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h3 class="text-lg font-bold text-white">
            Lebih Suka Menggunakan Terminal?
          </h3>
          <p class="text-sm text-neutral-400">
            Jalankan skrip ini untuk otomatis mengunduh binary, menambahkannya ke $PATH, dan mendaftarkan MCP ke AI Agent:
          </p>
        </div>

        <button
          type="button"
          class="shrink-0 inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-xs font-semibold transition-colors shadow-xs"
          @click="copyCliCommand"
        >
          <UIcon :name="copiedCommand ? 'i-lucide-check' : 'i-lucide-copy'" class="w-4 h-4" />
          <span>{{ copiedCommand ? 'Perintah Tersalin!' : 'Salin Perintah' }}</span>
        </button>
      </div>

      <!-- Code Box -->
      <div class="flex items-center justify-between px-4 py-3 rounded-lg bg-[#010810] border border-white/10 font-mono text-sm text-sky-300 overflow-x-auto">
        <code>{{ cliInstallCommand }}</code>
      </div>
    </div>

    <!-- Section Edukasi: Mengapa Vibecoders Membutuhkan Vibeasik? -->
    <div class="space-y-6 pt-4">
      <div class="text-center max-w-2xl mx-auto space-y-2">
        <h2 class="text-2xl sm:text-3xl font-extrabold text-neutral-900 dark:text-white">
          Mengapa Vibecoders Membutuhkan vibeasik?
        </h2>
        <p class="text-sm text-neutral-500 dark:text-neutral-400">
          Membuat aplikasi dengan AI prompt itu mudah dan cepat. Namun memastikan aplikasi tersebut
          <strong>aman</strong>, <strong>arsitekturnya rapi</strong>, dan <strong>dapat dimaintenance</strong> membutuhkan alat yang tepat.
        </p>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <!-- Feature 1: Security -->
        <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-900 p-6 space-y-3">
          <div class="w-10 h-10 rounded-lg bg-rose-50 dark:bg-rose-950/50 text-rose-600 dark:text-rose-400 flex items-center justify-center">
            <UIcon name="i-lucide-shield-alert" class="w-5 h-5" />
          </div>
          <h3 class="font-bold text-neutral-900 dark:text-white text-base">
            Audit Keamanan Hasil Prompting
          </h3>
          <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Kode dari AI sering kali menyertakan rahasia <code class="text-neutral-800 dark:text-neutral-200">.env</code> yang bocor,
            API key yang terhardcode, rute tanpa otentikasi, atau query SQL injection. vibeasik memindai dan memberikan skor keamanan instan.
          </p>
        </div>

        <!-- Feature 2: Clean Architecture -->
        <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-900 p-6 space-y-3">
          <div class="w-10 h-10 rounded-lg bg-blue-50 dark:bg-blue-950/50 text-blue-600 dark:text-blue-400 flex items-center justify-center">
            <UIcon name="i-lucide-boxes" class="w-5 h-5" />
          </div>
          <h3 class="font-bold text-neutral-900 dark:text-white text-base">
            Mencegah Kode Menjadi Spaghetti
          </h3>
          <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Saat fitur bertambah banyak, prompt AI cenderung menumpuk logic di satu berkas raksasa.
            vibeasik menganalisis topologi komponen, adapter rute, dan menyarankan pemisahan modular agar aplikasi mudah dirawat.
          </p>
        </div>

        <!-- Feature 3: Port Conflict & Background Process -->
        <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-900 p-6 space-y-3">
          <div class="w-10 h-10 rounded-lg bg-amber-50 dark:bg-amber-950/50 text-amber-600 dark:text-amber-400 flex items-center justify-center">
            <UIcon name="i-lucide-network" class="w-5 h-5" />
          </div>
          <h3 class="font-bold text-neutral-900 dark:text-white text-base">
            Monitor Port & Proses Latar Belakang
          </h3>
          <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Seringkali port 3000, 8080, atau 8765 saling berebut saat berganti proyek.
            vibeasik memiliki daemon monitor port live dan fitur 1-klik <em>Force Close</em> untuk membersihkan proses hantu di laptop Anda.
          </p>
        </div>

        <!-- Feature 4: Model Context Protocol (MCP) -->
        <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-900 p-6 space-y-3">
          <div class="w-10 h-10 rounded-lg bg-sky-50 dark:bg-sky-950/50 text-sky-600 dark:text-sky-400 flex items-center justify-center">
            <UIcon name="i-lucide-bot" class="w-5 h-5" />
          </div>
          <h3 class="font-bold text-neutral-900 dark:text-white text-base">
            20 Native MCP Tools untuk AI Agent
          </h3>
          <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Menghubungkan AI favorit Anda (Google Antigravity, Claude, Cursor, Windsurf) ke laptop Anda
            sehingga AI Agent dapat memindai folder, memeriksa database, dan memperbaiki bug secara mandiri.
          </p>
        </div>

        <!-- Feature 5: 3-Tier Planning & Todos -->
        <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-900 p-6 space-y-3">
          <div class="w-10 h-10 rounded-lg bg-purple-50 dark:bg-purple-950/50 text-purple-600 dark:text-purple-400 flex items-center justify-center">
            <UIcon name="i-lucide-list-checks" class="w-5 h-5" />
          </div>
          <h3 class="font-bold text-neutral-900 dark:text-white text-base">
            Perencanaan Terstruktur 3-Tingkat
          </h3>
          <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Kelola roadmap proyek pada level Platform, Bundle, dan Component.
            AI agent dapat mengusulkan Todo terarah yang bisa disetujui atau ditolak secara kolaboratif.
          </p>
        </div>

        <!-- Feature 6: Reverse Engineering (REA) -->
        <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-900 p-6 space-y-3">
          <div class="w-10 h-10 rounded-lg bg-cyan-50 dark:bg-cyan-950/50 text-cyan-600 dark:text-cyan-400 flex items-center justify-center">
            <UIcon name="i-lucide-binary" class="w-5 h-5" />
          </div>
          <h3 class="font-bold text-neutral-900 dark:text-white text-base">
            REA Reverse Engineering Toolkit
          </h3>
          <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Menganalisis bundle JavaScript, berkas ASAR Electron, binary ELF/EXE, dan struktur SQLite tanpa source code
            untuk memahami arsitektur aplikasi referensi secara presisi.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
