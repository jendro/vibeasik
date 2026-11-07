<script setup lang="ts">
interface TenanApp {
  id: string
  name: string
  tagline: string
  description: string
  category: string
  badge: string
  icon: string
  techStack: string[]
  statsLabel: string
  version: string
  command: string
  demoUrl?: string
}

interface UserApp {
  id: string
  title: string
  tagline: string
  description: string
  author: {
    name: string
    handle: string
    avatarInitials: string
    avatarBg: string
    isVerified?: boolean
  }
  category: string
  tags: string[]
  likes: number
  weeklyScore: number
  monthlyScore: number
  allTimeScore: number
  forks: number
  dateAgo: string
  command: string
  demoUrl: string
  previewType: 'code' | 'dashboard' | 'mobile' | 'api' | 'chat' | 'iot'
}

const toast = useToast()

// 1. By Tenan: 5 Official Core Apps (1 baris 5 grid)
const tenanApps: TenanApp[] = [
  {
    id: 'vibeport-guard',
    name: 'VibePort Guard Pro',
    tagline: 'Zombie Port Killer & Socket Monitor',
    description: 'Mendeteksi port 3000, 8080 & 8765 yang macet akibat server AI zombi lalu membunuhnya seketika dalam 1-klik tanpa terminal.',
    category: 'System & Socket',
    badge: 'Official Core',
    icon: 'i-lucide-activity',
    techStack: ['Go Socket', 'Syscall', 'Local First'],
    statsLabel: 'Zero Latency',
    version: 'v1.4.2',
    command: 'vibeasik ports --guard'
  },
  {
    id: 'mcp-agent-box',
    name: 'MCP Agent Box',
    tagline: 'Universal 20-Tools Agent Bridge',
    description: 'Jembatan Model Context Protocol lokal menghubungkan Claude Desktop, Antigravity, dan Cursor ke 20 tools diagnosa mesin.',
    category: 'AI Orchestration',
    badge: 'Official Core',
    icon: 'i-lucide-bot',
    techStack: ['Go', 'JSON-RPC', 'Stdio/SSE'],
    statsLabel: '20 Active Tools',
    version: 'v2.1.0',
    command: 'vibeasik mcp install'
  },
  {
    id: 'system-one-reasoner',
    name: 'System One Reasoner',
    tagline: 'TypeSafe LLM Judgment Engine',
    description: 'Mesin inferensi semantik berkecepatan tinggi untuk evaluasi probabilitas, scoring terstruktur, dan routing intent cerdas.',
    category: 'AI Reasoning',
    badge: 'Official Core',
    icon: 'i-lucide-brain-circuit',
    techStack: ['Jev Arch', 'Laya Engine', 'TypeSafe'],
    statsLabel: 'Sub-45ms P95',
    version: 'v1.2.0',
    command: 'vibeasik ai inject system-one'
  },
  {
    id: 'promptshield-audit',
    name: 'PromptShield Audit',
    tagline: 'AST Scanner & .env Leak Stopper',
    description: 'Scanner statis tingkat lanjut yang menyisir sintaks hasil prompting AI dari celah SQL injection, hardcoded secrets, dan endpoint rentan.',
    category: 'Security Scanner',
    badge: 'Official Core',
    icon: 'i-lucide-shield-check',
    techStack: ['AST Parser', 'SecRules', 'Pre-Deploy'],
    statsLabel: '99.4% Accuracy',
    version: 'v1.0.8',
    command: 'vibeasik scan --security'
  },
  {
    id: 'vibedeploy-auto-ssl',
    name: 'VibeDeploy Auto-SSL',
    tagline: 'Blueprint Cloud VPS & Auto-HTTPS',
    description: 'Generator konfigurasi deployment sekali-klik untuk Docker Compose dan Caddy reverse proxy dengan sertifikat SSL gratis instan.',
    category: 'DevOps & Cloud',
    badge: 'Official Core',
    icon: 'i-lucide-rocket',
    techStack: ['Caddy Auto-SSL', 'Docker', 'DNS API'],
    statsLabel: 'Zero-Config SSL',
    version: 'v1.1.5',
    command: 'vibeasik deploy --plan'
  }
]

// 2. User Community Apps (Populer & Ukuran Lebih Besar)
const userApps = ref<UserApp[]>([
  {
    id: 'kopiprompt-ui',
    title: 'KopiPrompt UI - Generator Prompt & UI Layout',
    tagline: 'Visual Builder Komponen Tailwind untuk AI Agents',
    description: 'Penyusun layout halaman web interaktif yang otomatis menghasilkan prompt arsitektur dan potongan Tailwind CSS bersih siap dimasukkan ke Claude Code atau Antigravity.',
    author: {
      name: 'Aditya Pratama',
      handle: '@aditya_vibecoder',
      avatarInitials: 'AP',
      avatarBg: 'bg-amber-600',
      isVerified: true
    },
    category: 'AI & Prompt',
    tags: ['Nuxt 4', 'Tailwind v4', 'VueUse', 'Design System'],
    likes: 1420,
    weeklyScore: 980,
    monthlyScore: 3240,
    allTimeScore: 12850,
    forks: 342,
    dateAgo: '2 hari lalu',
    command: 'vibeasik pull @aditya/kopiprompt-ui',
    demoUrl: 'https://kopiprompt.vibeasik.dev',
    previewType: 'code'
  },
  {
    id: 'kasirkilat-pos',
    title: 'KasirKilat POS - Kasir Offline-First Toko Retail',
    tagline: 'Point of Sale Cepat Tanpa Ketergantungan Cloud',
    description: 'Sistem kasir retail mini market berbasis Flutter Web & SQLite lokal. Mendukung cetak struk via printer thermal Bluetooth ESC/POS serta auto-sync saat koneksi internet pulih.',
    author: {
      name: 'Siti Nurhaliza',
      handle: '@siti_dev',
      avatarInitials: 'SN',
      avatarBg: 'bg-emerald-600',
      isVerified: true
    },
    category: 'Fullstack',
    tags: ['Flutter Web', 'SQLite', 'Bluetooth ESC/POS', 'Offline-First'],
    likes: 1890,
    weeklyScore: 1120,
    monthlyScore: 4110,
    allTimeScore: 15920,
    forks: 489,
    dateAgo: '4 hari lalu',
    command: 'vibeasik pull @siti/kasirkilat-pos',
    demoUrl: 'https://kasirkilat.vibeasik.dev',
    previewType: 'dashboard'
  },
  {
    id: 'autoinvoice-wa',
    title: 'AutoInvoice WA - Notifikasi & Tagihan Otomatis',
    tagline: 'Kirim Invoice WhatsApp & Pantau Pembayaran Midtrans',
    description: 'Sistem penagihan otomatis yang mengirimkan pesan pengingat tagihan invoice resmi via WhatsApp API, lengkap dengan link checkout Midtrans Snap dan pencatatan audit.',
    author: {
      name: 'Fajar Nugraha',
      handle: '@fajar_vibes',
      avatarInitials: 'FN',
      avatarBg: 'bg-violet-600',
      isVerified: true
    },
    category: 'Fullstack',
    tags: ['Nuxt 4', 'Midtrans Snap', 'WhatsApp API', 'PostgreSQL'],
    likes: 2130,
    weeklyScore: 1350,
    monthlyScore: 5420,
    allTimeScore: 18200,
    forks: 612,
    dateAgo: '1 minggu lalu',
    command: 'vibeasik pull @fajar/autoinvoice-wa',
    demoUrl: 'https://autoinvoice.vibeasik.dev',
    previewType: 'mobile'
  },
  {
    id: 'docuvibe-ai',
    title: 'DocuVibe AI - Generator API Docs & Diagram Mermaid',
    tagline: 'Reverse-Engineering Kode Menjadi OpenAPI 3.1 & Mermaid Flow',
    description: 'Memindai handler Go Gin, FastAPI, dan rute Nuxt lokal lalu otomatis memvisualisasikan arsitektur sistem menjadi diagram interaktif Mermaid serta spesifikasi Swagger siap ekspor.',
    author: {
      name: 'Budi Santoso',
      handle: '@budi_santoso',
      avatarInitials: 'BS',
      avatarBg: 'bg-sky-600'
    },
    category: 'Developer Tools',
    tags: ['Go Gin', 'OpenAPI 3.1', 'Mermaid.js', 'FastAPI'],
    likes: 954,
    weeklyScore: 870,
    monthlyScore: 2890,
    allTimeScore: 9640,
    forks: 215,
    dateAgo: '6 hari lalu',
    command: 'vibeasik pull @budi/docuvibe-ai',
    demoUrl: 'https://docuvibe.vibeasik.dev',
    previewType: 'api'
  },
  {
    id: 'vibechat-multiagent',
    title: 'VibeChat Support - Live Chat AI Multi-Agent',
    tagline: 'Chatbot Cerdas dengan Eskalasi Otomatis ke Manusia',
    description: 'Widget obrolan pelanggan cerdas yang memecahkan pertanyaan pelanggan berbasis System One Reasoner dan langsung mengalihkan sesi percakapan ke operator manusia via bot Telegram.',
    author: {
      name: 'Reza Pahlevi',
      handle: '@reza_ai',
      avatarInitials: 'RP',
      avatarBg: 'bg-rose-600',
      isVerified: true
    },
    category: 'AI & Prompt',
    tags: ['System One', 'Nuxt 4', 'Telegram Bot', 'SSE Stream'],
    likes: 1670,
    weeklyScore: 1040,
    monthlyScore: 3980,
    allTimeScore: 14100,
    forks: 420,
    dateAgo: '3 hari lalu',
    command: 'vibeasik pull @reza/vibechat-multiagent',
    demoUrl: 'https://vibechat.vibeasik.dev',
    previewType: 'chat'
  },
  {
    id: 'agrisensor-iot',
    title: 'AgriSensor IoT - Telemetri Tanah & Cuaca Realtime',
    tagline: 'Dashboard Monitoring Sensor Pertanian Presisi',
    description: 'Telemetri langsung untuk memonitor kelembapan tanah, suhu ruang, pH, dan kadar nutrisi NPK via protokol MQTT dengan visualisasi grafik canvas 60fps tanpa lagging.',
    author: {
      name: 'Dewi Kusuma',
      handle: '@dewi_tech',
      avatarInitials: 'DK',
      avatarBg: 'bg-teal-600'
    },
    category: 'Mobile & IoT',
    tags: ['Go Engine', 'MQTT Protocol', 'Chart.js', 'ESP32'],
    likes: 780,
    weeklyScore: 640,
    monthlyScore: 2150,
    allTimeScore: 7430,
    forks: 180,
    dateAgo: '2 minggu lalu',
    command: 'vibeasik pull @dewi/agrisensor-iot',
    demoUrl: 'https://agrisensor.vibeasik.dev',
    previewType: 'iot'
  }
])

// Filter & Sort State
type FilterMode = 'like-terbanyak' | 'this-week' | 'this-month' | 'all-time'
const activeFilter = ref<FilterMode>('this-week')
const activeCategory = ref<string>('all')
const searchQuery = ref<string>('')

// Interactive Like Tracking
const likedAppIds = ref<Set<string>>(new Set())

const toggleLike = (app: UserApp) => {
  if (likedAppIds.value.has(app.id)) {
    likedAppIds.value.delete(app.id)
    app.likes--
    toast.add({
      title: 'Like Dibatalkan',
      description: `Batal menyukai ${app.title}`,
      color: 'neutral'
    })
  } else {
    likedAppIds.value.add(app.id)
    app.likes++
    toast.add({
      title: 'Aplikasi Disukai! ✨',
      description: `Terima kasih telah mengapresiasi ${app.title}`,
      color: 'success'
    })
  }
}

// Filtered & Sorted Apps Computed
const categories = ['all', 'AI & Prompt', 'Fullstack', 'Developer Tools', 'Mobile & IoT']

const filteredUserApps = computed(() => {
  let list = [...userApps.value]

  // Category Filter
  if (activeCategory.value !== 'all') {
    list = list.filter(app => app.category === activeCategory.value)
  }

  // Search Filter
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase().trim()
    list = list.filter(app =>
      app.title.toLowerCase().includes(q)
      || app.tagline.toLowerCase().includes(q)
      || app.description.toLowerCase().includes(q)
      || app.author.name.toLowerCase().includes(q)
      || app.author.handle.toLowerCase().includes(q)
      || app.tags.some(t => t.toLowerCase().includes(q))
    )
  }

  // Sort Filter
  if (activeFilter.value === 'like-terbanyak') {
    list.sort((a, b) => b.likes - a.likes)
  } else if (activeFilter.value === 'this-week') {
    list.sort((a, b) => b.weeklyScore - a.weeklyScore)
  } else if (activeFilter.value === 'this-month') {
    list.sort((a, b) => b.monthlyScore - a.monthlyScore)
  } else if (activeFilter.value === 'all-time') {
    list.sort((a, b) => b.allTimeScore - a.allTimeScore)
  }

  return list
})

// Modal State
const selectedApp = ref<UserApp | TenanApp | null>(null)
const isModalOpen = ref(false)
const copiedCommand = ref<string | null>(null)

const openAppDetail = (app: UserApp | TenanApp) => {
  selectedApp.value = app
  isModalOpen.value = true
}

const copyCommand = async (cmd: string) => {
  try {
    await navigator.clipboard.writeText(cmd)
    copiedCommand.value = cmd
    toast.add({
      title: 'Perintah Disalin!',
      description: 'Tempel di terminal Anda untuk menjalankan aplikasi ini.',
      color: 'success'
    })
    setTimeout(() => {
      if (copiedCommand.value === cmd) {
        copiedCommand.value = null
      }
    }, 2000)
  } catch (err) {
    console.error('Failed to copy command:', err)
  }
}

const isUserApp = (app: UserApp | TenanApp): app is UserApp => {
  return 'author' in app
}
</script>

<template>
  <section id="featured-apps" class="space-y-16 sm:space-y-24 scroll-mt-24">
    <!-- Section Header Utama -->
    <div class="text-center max-w-3xl mx-auto space-y-3 px-4">
      <div class="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-mono font-medium text-sky-300 bg-sky-950/60 border border-sky-500/30">
        <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 text-sky-400" />
        <span>Featured Vibe Apps Ecosystem</span>
      </div>
      <h2 class="text-2xl sm:text-3xl lg:text-4xl font-extrabold text-white tracking-tight">
        Katalog Aplikasi Pilihan Vibecoders
      </h2>
      <p class="text-xs sm:text-sm text-slate-300 leading-relaxed">
        Solusi siap pakai dari pemilik vibeasik serta ratusan proyek inspiratif buatan komunitas vibecoding.
      </p>
    </div>

    <!-- ======================================================== -->
    <!-- BAGIAN 1: BY TENAN (Aplikasi Buatan Pemilik Vibeasik)     -->
    <!-- SPESIFIKASI: 1 BARIS BERISI 5 GRID                       -->
    <!-- ======================================================== -->
    <div class="space-y-6">
      <!-- Header By Tenan -->
      <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-4 border-b border-white/10 pb-4">
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-md text-[11px] font-mono font-semibold bg-sky-500/20 text-sky-300 border border-sky-400/30">
              <UIcon name="i-lucide-badge-check" class="w-3.5 h-3.5 text-sky-400" />
              <span>BY TENAN</span>
            </span>
            <span class="text-xs font-mono text-slate-400">• Official Founder Edition</span>
          </div>
          <h3 class="text-lg sm:text-xl font-bold text-white flex items-center gap-2">
            Aplikasi Buatan Pemilik vibeasik
          </h3>
          <p class="text-xs text-slate-400 max-w-2xl">
            5 modul inti yang saya rancang dan bangun sendiri untuk menyelesaikan masalah port macet, keamanan prompt, dan orkestrasi AI lokal.
          </p>
        </div>

        <div class="flex items-center gap-2 text-xs font-mono text-sky-300 bg-slate-900/60 border border-white/10 px-3 py-1.5 rounded-lg w-fit">
          <UIcon name="i-lucide-check-circle" class="w-3.5 h-3.5 text-sky-400" />
          <span>5 Core Modules Active</span>
        </div>
      </div>

      <!-- 1 Baris Berisi 5 Grid (1 row of 5 columns on desktop, responsive wrap) -->
      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3.5 sm:gap-4">
        <article
          v-for="app in tenanApps"
          :key="app.id"
          class="group rounded-xl border border-white/10 bg-slate-900/50 backdrop-blur-xs p-4 flex flex-col justify-between hover:border-sky-500/40 hover:bg-slate-900/75 transition-all duration-200 shadow-sm hover:shadow-md hover:shadow-sky-950/30"
        >
          <!-- Top Card Meta -->
          <div class="space-y-3">
            <div class="flex items-start justify-between gap-2">
              <div class="w-10 h-10 rounded-lg bg-sky-500/10 border border-sky-400/20 text-sky-300 flex items-center justify-center group-hover:scale-105 group-hover:bg-sky-500/20 transition-all">
                <UIcon :name="app.icon" class="w-5 h-5 text-sky-400" />
              </div>
              <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-slate-950/80 text-sky-300 border border-white/10 font-medium">
                {{ app.badge }}
              </span>
            </div>

            <div class="space-y-1">
              <h4 class="text-sm font-bold text-white group-hover:text-sky-300 transition-colors line-clamp-1">
                {{ app.name }}
              </h4>
              <p class="text-[11px] font-mono text-slate-400 line-clamp-1">
                {{ app.tagline }}
              </p>
            </div>

            <p class="text-xs text-slate-300/90 leading-relaxed line-clamp-3">
              {{ app.description }}
            </p>
          </div>

          <!-- Bottom Card Meta -->
          <div class="pt-3 mt-3 border-t border-white/5 space-y-2.5">
            <div class="flex flex-wrap gap-1">
              <span
                v-for="tech in app.techStack"
                :key="tech"
                class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-950/60 text-slate-300 border border-white/5"
              >
                {{ tech }}
              </span>
            </div>

            <div class="flex items-center justify-between pt-1">
              <span class="text-[10px] font-mono text-slate-400">
                {{ app.statsLabel }}
              </span>
              <button
                type="button"
                class="inline-flex items-center gap-1 text-xs font-semibold text-sky-400 hover:text-sky-300 transition-colors cursor-pointer py-1 px-1.5 -mr-1.5 rounded hover:bg-sky-950/50"
                @click="openAppDetail(app)"
              >
                <span>Detail</span>
                <UIcon name="i-lucide-arrow-up-right" class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </article>
      </div>
    </div>

    <!-- ======================================================== -->
    <!-- BAGIAN 2: POPULAR APPS MILIK USER                        -->
    <!-- SPESIFIKASI: FILTER (LIKE, WEEK, MONTH, ALL TIME)        -->
    <!-- UKURAN KARTU LEBIH BESAR DARI KARTU TENAN                -->
    <!-- ======================================================== -->
    <div class="space-y-8 pt-6">
      <!-- Header Popular Apps User -->
      <div class="flex flex-col lg:flex-row lg:items-end justify-between gap-6 border-b border-white/10 pb-6">
        <div class="space-y-1.5">
          <div class="flex items-center gap-2">
            <span class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-md text-[11px] font-mono font-semibold bg-emerald-500/10 text-emerald-300 border border-emerald-400/20">
              <UIcon name="i-lucide-users" class="w-3.5 h-3.5 text-emerald-400" />
              <span>KOMUNITAS VIBECODERS</span>
            </span>
            <span class="text-xs font-mono text-slate-400">• Apps Milik Pengguna</span>
          </div>
          <h3 class="text-xl sm:text-2xl font-bold text-white">
            Popular Vibe Apps Komunitas
          </h3>
          <p class="text-xs sm:text-sm text-slate-300 max-w-2xl">
            Aplikasi inovatif buatan sesama vibecoders yang dapat Anda jalankan, pelajari polanya, atau fork langsung ke proyek Anda.
          </p>
        </div>

        <!-- Filter Buttons: Like Terbanyak, This Week, This Month, All Time -->
        <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2.5">
          <div class="inline-flex p-1 rounded-xl bg-slate-950/80 border border-white/10 backdrop-blur-xs">
            <button
              type="button"
              class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5"
              :class="activeFilter === 'like-terbanyak'
                ? 'bg-sky-600 text-white shadow-xs font-semibold'
                : 'text-slate-400 hover:text-white hover:bg-white/5'"
              @click="activeFilter = 'like-terbanyak'"
            >
              <UIcon name="i-lucide-heart" class="w-3.5 h-3.5" :class="activeFilter === 'like-terbanyak' ? 'text-rose-300' : ''" />
              <span>Like Terbanyak</span>
            </button>

            <button
              type="button"
              class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5"
              :class="activeFilter === 'this-week'
                ? 'bg-sky-600 text-white shadow-xs font-semibold'
                : 'text-slate-400 hover:text-white hover:bg-white/5'"
              @click="activeFilter = 'this-week'"
            >
              <UIcon name="i-lucide-flame" class="w-3.5 h-3.5" :class="activeFilter === 'this-week' ? 'text-amber-300' : ''" />
              <span>This Week</span>
            </button>

            <button
              type="button"
              class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5"
              :class="activeFilter === 'this-month'
                ? 'bg-sky-600 text-white shadow-xs font-semibold'
                : 'text-slate-400 hover:text-white hover:bg-white/5'"
              @click="activeFilter = 'this-month'"
            >
              <UIcon name="i-lucide-calendar" class="w-3.5 h-3.5" />
              <span>This Month</span>
            </button>

            <button
              type="button"
              class="px-3 py-1.5 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5"
              :class="activeFilter === 'all-time'
                ? 'bg-sky-600 text-white shadow-xs font-semibold'
                : 'text-slate-400 hover:text-white hover:bg-white/5'"
              @click="activeFilter = 'all-time'"
            >
              <UIcon name="i-lucide-trophy" class="w-3.5 h-3.5" :class="activeFilter === 'all-time' ? 'text-yellow-400' : ''" />
              <span>All Time</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Sub Filter: Category Chips + Search Bar -->
      <div class="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
        <!-- Categories Chips -->
        <div class="flex items-center gap-1.5 overflow-x-auto pb-1 sm:pb-0 scrollbar-none">
          <button
            v-for="cat in categories"
            :key="cat"
            type="button"
            class="px-3 py-1 rounded-lg text-xs font-medium transition-colors shrink-0 cursor-pointer"
            :class="activeCategory === cat
              ? 'bg-sky-500/20 text-sky-300 border border-sky-400/40 font-semibold'
              : 'bg-slate-900/60 text-slate-400 border border-white/5 hover:text-slate-200 hover:bg-slate-800/60'"
            @click="activeCategory = cat"
          >
            {{ cat === 'all' ? 'Semua Kategori' : cat }}
          </button>
        </div>

        <!-- Search Bar Ringkas -->
        <div class="relative w-full sm:w-64">
          <UIcon name="i-lucide-search" class="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Cari vibe apps atau tech..."
            class="w-full pl-9 pr-3 py-1.5 text-xs rounded-lg bg-slate-950/80 border border-white/10 text-white placeholder-slate-500 focus:outline-none focus:border-sky-500/50 focus:ring-1 focus:ring-sky-500/30 transition-all"
          >
          <button
            v-if="searchQuery"
            type="button"
            class="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white"
            @click="searchQuery = ''"
          >
            <UIcon name="i-lucide-x" class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      <!-- Grid Aplikasi Milik User: UKURAN LEBIH BESAR DARI KARTU TENAN -->
      <!-- 3 Kolom di Layar Besar (Memberi ruang visual tinggi, banner, dan meta detail) -->
      <div v-if="filteredUserApps.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 sm:gap-7">
        <article
          v-for="app in filteredUserApps"
          :key="app.id"
          class="group rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-xs flex flex-col justify-between overflow-hidden hover:border-sky-500/40 hover:bg-slate-900/70 transition-all duration-300 shadow-lg hover:shadow-xl hover:shadow-sky-950/30"
        >
          <!-- 1. Top Visual Banner (Ukuran Besar & Informatif) -->
          <div class="relative h-36 bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 border-b border-white/10 p-4 flex flex-col justify-between overflow-hidden">
            <!-- Subtle Radial Gradient -->
            <div class="absolute inset-0 bg-radial from-sky-500/5 to-transparent pointer-events-none" />

            <!-- Banner Header: Category & Rank -->
            <div class="relative z-10 flex items-center justify-between">
              <span class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-mono font-medium bg-slate-950/80 text-sky-300 border border-sky-400/20 backdrop-blur-xs">
                <UIcon name="i-lucide-tag" class="w-3 h-3 text-sky-400" />
                <span>{{ app.category }}</span>
              </span>

              <div class="flex items-center gap-1.5 text-xs font-mono text-slate-300 bg-slate-950/70 px-2 py-0.5 rounded-md border border-white/5">
                <UIcon name="i-lucide-git-fork" class="w-3 h-3 text-slate-400" />
                <span>{{ app.forks }} forks</span>
              </div>
            </div>

            <!-- Banner Mockup Graphics -->
            <div class="relative z-10 font-mono text-[11px] text-slate-400 bg-slate-950/80 rounded-lg p-2.5 border border-white/10 flex items-center justify-between">
              <div class="flex items-center gap-2 truncate">
                <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                <span class="text-slate-300 font-medium truncate">$ {{ app.command }}</span>
              </div>
              <button
                type="button"
                title="Salin perintah"
                class="text-slate-400 hover:text-sky-300 cursor-pointer p-1 rounded hover:bg-white/5 transition-colors"
                @click.stop="copyCommand(app.command)"
              >
                <UIcon :name="copiedCommand === app.command ? 'i-lucide-check' : 'i-lucide-copy'" class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>

          <!-- 2. Middle Body: Author & Content (Ukuran Lebih Besar) -->
          <div class="p-6 space-y-4 flex-1 flex flex-col justify-between">
            <div class="space-y-3">
              <!-- Author Profile -->
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2.5">
                  <div
                    class="w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold text-white shadow-xs"
                    :class="app.author.avatarBg"
                  >
                    {{ app.author.avatarInitials }}
                  </div>
                  <div>
                    <div class="flex items-center gap-1 text-xs font-semibold text-white">
                      <span>{{ app.author.name }}</span>
                      <UIcon
                        v-if="app.author.isVerified"
                        name="i-lucide-badge-check"
                        class="w-3.5 h-3.5 text-sky-400"
                        title="Verified Vibecoder"
                      />
                    </div>
                    <span class="text-[11px] font-mono text-slate-400">{{ app.author.handle }}</span>
                  </div>
                </div>

                <span class="text-[11px] font-mono text-slate-500">{{ app.dateAgo }}</span>
              </div>

              <!-- Title & Tagline -->
              <div class="space-y-1">
                <h4 class="text-base font-bold text-white group-hover:text-sky-300 transition-colors line-clamp-1">
                  {{ app.title }}
                </h4>
                <p class="text-xs text-sky-400/90 font-medium line-clamp-1">
                  {{ app.tagline }}
                </p>
              </div>

              <!-- Longer Descriptive Text -->
              <p class="text-xs text-slate-300 leading-relaxed line-clamp-3">
                {{ app.description }}
              </p>
            </div>

            <!-- Tech Badges -->
            <div class="pt-2">
              <div class="flex flex-wrap gap-1.5">
                <span
                  v-for="tag in app.tags"
                  :key="tag"
                  class="text-[10px] font-mono px-2 py-0.5 rounded-md bg-slate-950/70 text-slate-300 border border-white/10"
                >
                  {{ tag }}
                </span>
              </div>
            </div>
          </div>

          <!-- 3. Bottom Action Bar: Likes + Detail Button -->
          <div class="px-6 py-4 bg-slate-950/50 border-t border-white/10 flex items-center justify-between">
            <!-- Interactive Like Button with Animated Feedback -->
            <button
              type="button"
              class="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer border"
              :class="likedAppIds.has(app.id)
                ? 'bg-rose-500/10 text-rose-400 border-rose-500/30'
                : 'bg-slate-900/60 text-slate-300 border-white/10 hover:border-white/20 hover:text-white'"
              @click.stop="toggleLike(app)"
            >
              <UIcon
                name="i-lucide-heart"
                class="w-4 h-4 transition-transform active:scale-125"
                :class="likedAppIds.has(app.id) ? 'fill-rose-500 text-rose-500' : 'text-slate-400'"
              />
              <span class="font-mono">{{ app.likes.toLocaleString('id-ID') }}</span>
            </button>

            <!-- Action Links -->
            <div class="flex items-center gap-2">
              <button
                type="button"
                class="px-3 py-1.5 rounded-lg text-xs font-medium text-slate-300 hover:text-white bg-white/5 hover:bg-white/10 border border-white/10 transition-colors cursor-pointer"
                @click="openAppDetail(app)"
              >
                Detail
              </button>

              <NuxtLink
                :to="app.demoUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold text-white bg-sky-600 hover:bg-sky-500 transition-colors shadow-xs"
              >
                <span>Live Demo</span>
                <UIcon name="i-lucide-external-link" class="w-3.5 h-3.5" />
              </NuxtLink>
            </div>
          </div>
        </article>
      </div>

      <!-- State Kosong jika Filter Tidak Menemukan Hasil -->
      <div
        v-else
        class="text-center py-12 px-4 rounded-2xl border border-dashed border-white/15 bg-slate-900/20 space-y-3"
      >
        <UIcon name="i-lucide-search-x" class="w-10 h-10 text-slate-500 mx-auto" />
        <h4 class="text-sm font-semibold text-white">
          Tidak ada aplikasi yang cocok
        </h4>
        <p class="text-xs text-slate-400 max-w-sm mx-auto">
          Coba ubah kata kunci pencarian atau ganti kategori filter untuk melihat aplikasi lainnya.
        </p>
        <button
          type="button"
          class="text-xs text-sky-400 hover:underline pt-1 cursor-pointer font-medium"
          @click="searchQuery = ''; activeCategory = 'all'"
        >
          Reset Semua Filter
        </button>
      </div>

      <!-- Banner Submit Vibe App Sendiri -->
      <div class="rounded-2xl border border-sky-500/20 bg-slate-900/50 backdrop-blur-xs p-6 sm:p-8 flex flex-col sm:flex-row items-center justify-between gap-6">
        <div class="space-y-1.5 text-center sm:text-left">
          <div class="inline-flex items-center gap-1.5 text-xs font-mono text-sky-300">
            <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 text-sky-400" />
            <span>Punya Aplikasi Buatan Sendiri?</span>
          </div>
          <h4 class="text-lg sm:text-xl font-bold text-white">
            Pamerkan Vibe App Anda ke Ribuan Vibecoders
          </h4>
          <p class="text-xs sm:text-sm text-slate-300 max-w-xl">
            Jadikan aplikasi buatan Anda dikenal luas, dapatkan feedback kode, dan raih like terbanyak di leaderboard mingguan vibeasik.
          </p>
        </div>

        <NuxtLink
          to="/cockpit"
          class="shrink-0 inline-flex items-center gap-2 px-5 py-2.5 rounded-xl font-semibold text-white bg-sky-600 hover:bg-sky-500 transition-all text-xs shadow-md shadow-sky-950/40"
        >
          <UIcon name="i-lucide-plus-circle" class="w-4 h-4" />
          <span>Daftarkan Aplikasi Anda</span>
        </NuxtLink>
      </div>
    </div>

    <!-- ======================================================== -->
    <!-- MODAL DETAIL APLIKASI (Untuk Tenan & User Apps)          -->
    <!-- ======================================================== -->
    <UModal
      :open="isModalOpen"
      :title="selectedApp ? ('name' in selectedApp ? selectedApp.name : selectedApp.title) : 'Detail Aplikasi'"
      :description="selectedApp?.tagline || 'Informasi lengkap modul dan arsitektur aplikasi.'"
      @update:open="val => isModalOpen = val"
    >
      <template #body>
        <div v-if="selectedApp" class="space-y-5 text-sm">
          <!-- Meta Header -->
          <div class="flex items-center justify-between p-3 rounded-xl bg-slate-950/70 border border-white/10">
            <div class="flex items-center gap-2.5">
              <span
                class="px-2.5 py-1 rounded-md text-xs font-mono font-semibold"
                :class="'badge' in selectedApp
                  ? 'bg-sky-500/20 text-sky-300 border border-sky-400/30'
                  : 'bg-emerald-500/20 text-emerald-300 border border-emerald-400/30'"
              >
                {{ 'badge' in selectedApp ? selectedApp.badge : selectedApp.category }}
              </span>
              <span class="text-xs font-mono text-slate-400">
                {{ 'version' in selectedApp ? selectedApp.version : ('Dibuat oleh ' + selectedApp.author.name) }}
              </span>
            </div>

            <div v-if="isUserApp(selectedApp)" class="flex items-center gap-1.5 text-xs font-mono text-rose-400">
              <UIcon name="i-lucide-heart" class="w-3.5 h-3.5 fill-rose-500" />
              <span>{{ selectedApp.likes.toLocaleString('id-ID') }} Likes</span>
            </div>
          </div>

          <!-- Description -->
          <div class="space-y-2">
            <h5 class="text-xs font-bold uppercase tracking-wider text-slate-400">
              Deskripsi Lengkap
            </h5>
            <p class="text-xs sm:text-sm text-slate-200 leading-relaxed">
              {{ selectedApp.description }}
            </p>
          </div>

          <!-- Tech Stack -->
          <div class="space-y-2">
            <h5 class="text-xs font-bold uppercase tracking-wider text-slate-400">
              Teknologi & Dependensi
            </h5>
            <div class="flex flex-wrap gap-2">
              <span
                v-for="t in ('techStack' in selectedApp ? selectedApp.techStack : selectedApp.tags)"
                :key="t"
                class="px-2.5 py-1 rounded-lg bg-slate-950/80 border border-white/10 text-xs font-mono text-sky-300"
              >
                {{ t }}
              </span>
            </div>
          </div>

          <!-- Run Command -->
          <div class="space-y-2">
            <h5 class="text-xs font-bold uppercase tracking-wider text-slate-400">
              Jalankan di Terminal Lokal
            </h5>
            <div class="flex items-center justify-between p-3 rounded-xl bg-slate-950 border border-white/10 font-mono text-xs">
              <span class="text-sky-300">$ {{ selectedApp.command }}</span>
              <button
                type="button"
                class="inline-flex items-center gap-1 text-slate-400 hover:text-white px-2 py-1 rounded bg-white/5 hover:bg-white/10 transition-colors cursor-pointer"
                @click="copyCommand(selectedApp.command)"
              >
                <UIcon :name="copiedCommand === selectedApp.command ? 'i-lucide-check' : 'i-lucide-copy'" class="w-3.5 h-3.5" />
                <span>{{ copiedCommand === selectedApp.command ? 'Tersalin' : 'Salin' }}</span>
              </button>
            </div>
          </div>
        </div>
      </template>

      <template #footer>
        <div class="flex items-center justify-end gap-2 w-full">
          <UButton
            color="neutral"
            variant="ghost"
            label="Tutup"
            @click="isModalOpen = false"
          />

          <NuxtLink
            v-if="selectedApp && 'demoUrl' in selectedApp && selectedApp.demoUrl"
            :to="selectedApp.demoUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-semibold text-white bg-sky-600 hover:bg-sky-500 transition-colors shadow-xs"
          >
            <span>Buka Demo</span>
            <UIcon name="i-lucide-external-link" class="w-3.5 h-3.5" />
          </NuxtLink>
        </div>
      </template>
    </UModal>
  </section>
</template>
