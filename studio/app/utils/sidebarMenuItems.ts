// Data menu navigasi sidebar (statis) + penyusun menu proyek aktif.

export interface NavItem {
  id: string
  to: string
  label: string
  desc: string
  icon: string
  exact?: boolean
  isDetail?: boolean
  // Tab halaman Detail Proyek yang dibuka item ini (?tab=…); dipakai untuk menandai item aktif.
  tab?: string
  badge?: string
  // Kelompok tampilan di sidebar (judul kecil di atas item pertama kelompok).
  group?: string
}

// Tab halaman Detail Proyek. Tiap tab punya item sendiri di sidebar (tidak lagi tersembunyi di balik
// satu menu), dan bilah tab di halaman hanya menampilkan tab sekelompok.
export interface ProjectTab {
  id: string
  label: string
  desc: string
  icon: string
  group: string
}

export const PROJECT_TABS: ProjectTab[] = [
  { id: 'overview', label: 'Ringkasan Proyek', desc: 'Tujuan, fitur, kematangan & angka utama', icon: 'i-lucide-layout-dashboard', group: 'Ringkasan' },
  { id: 'recap', label: 'Rekap Pemilik', desc: 'Progres, bug ditutup & kesehatan kode', icon: 'i-lucide-chart-no-axes-column', group: 'Ringkasan' },
  { id: 'apps', label: 'Aplikasi', desc: 'Modul & hubungan antar aplikasi', icon: 'i-lucide-boxes', group: 'Struktur proyek' },
  { id: 'technical', label: 'Rute, File & Env', desc: 'Katalog router, struktur & .env', icon: 'i-lucide-book-open-check', group: 'Struktur proyek' },
  { id: 'synergy', label: 'Alur Data', desc: 'Halaman → API → Database', icon: 'i-lucide-network', group: 'Struktur proyek' }
]

export const DEFAULT_PROJECT_TAB = 'overview'

export const projectTabFor = (id: string): ProjectTab | undefined => PROJECT_TABS.find(t => t.id === id)

// Id tab lama tetap bekerja (?tab=topology / ?tab=base_knowledge). Tab "Live Visual" dihapus: tangkapan layar &
// audit visual ada di halaman Screenshot, jadi ?tab=visual_audit jatuh ke Ringkasan.
// Tab yang sudah menjadi halaman sendiri: tautan lama ?tab=… dialihkan ke halamannya.
export const PROJECT_TAB_PAGES: Record<string, string> = { cost: '/process-cost' }

const PROJECT_TAB_ALIASES: Record<string, string> = { topology: 'apps', base_knowledge: 'technical', visual_audit: 'overview' }

export const normalizeProjectTab = (tab: unknown): string => {
  const t = typeof tab === 'string' ? (PROJECT_TAB_ALIASES[tab] || tab) : ''
  return projectTabFor(t) ? t : DEFAULT_PROJECT_TAB
}

// Kelompok menu proyek aktif, mengikuti alur memperbaiki aplikasi:
// ringkas → periksa menyeluruh (performa, data, tampilan, kecepatan, trafik, keamanan) → kerjakan todo → struktur → operasional.
const PROJECT_GROUPS: Record<string, string> = {
  'home': 'Ringkasan',
  'audit-report': 'Periksa & tingkatkan',
  'process-cost': 'Periksa & tingkatkan',
  'runtime-test': 'Periksa & tingkatkan',
  'traffic': 'Periksa & tingkatkan',
  'security-test': 'Periksa & tingkatkan',
  'screenshots': 'Periksa & tingkatkan',
  'todos': 'Kerjakan perbaikan',
  'queue': 'Kerjakan perbaikan',
  'client-room': 'Kerjakan perbaikan',
  'audit-learning': 'Kerjakan perbaikan',
  'code-map': 'Struktur proyek',
  'ports': 'Operasional',
  'delivery': 'Operasional'
}
const PROJECT_ORDER = [
  'home', 'tab:overview', 'tab:recap',
  'audit-report', 'process-cost', 'runtime-test', 'traffic', 'security-test', 'screenshots',
  'todos', 'queue', 'client-room', 'audit-learning',
  'tab:apps', 'tab:technical', 'tab:synergy', 'code-map',
  'ports', 'delivery'
]

export const buildProjectNavItems = (activeProjectId: number | null, activeProject: { name: string } | null): NavItem[] => {
  const detailTo = activeProjectId ? `/projects/${activeProjectId}` : '/cockpit'

  // Tanpa proyek aktif, tab detail tidak punya tujuan: cukup satu item yang membawa ke pemilihan proyek.
  const tabItems: NavItem[] = PROJECT_TABS
    .filter(t => activeProjectId || t.id === DEFAULT_PROJECT_TAB)
    .map(t => ({
      id: `tab:${t.id}`,
      to: !activeProjectId || t.id === DEFAULT_PROJECT_TAB ? detailTo : `${detailTo}?tab=${t.id}`,
      label: t.label,
      desc: t.id === DEFAULT_PROJECT_TAB ? (activeProject ? `${activeProject.name} — ${t.desc}` : 'Pilih proyek aktif') : t.desc,
      icon: t.icon,
      isDetail: true,
      tab: t.id,
      group: t.group
    }))

  const items: NavItem[] = [
    ...tabItems,
    {
      id: 'home',
      to: '/',
      label: 'Beranda',
      desc: 'Status & cara memakai',
      icon: 'i-lucide-house',
      exact: true
    },
    {
      // Satu-satunya ruang obrolan pengguna (klien) dengan dewan eksekutif proyek aktif.
      id: 'client-room',
      to: '/client-room',
      label: 'Ruang Klien',
      desc: 'Minta fitur & lapor bug ke dewan',
      icon: 'i-lucide-handshake',
      exact: false
    },
    {
      id: 'todos',
      to: '/todos',
      label: 'Perencanaan & Todo',
      desc: 'Todo hasil audit & roadmap',
      icon: 'i-lucide-clipboard-check',
      exact: false
    },
    {
      id: 'code-map',
      to: '/code-map',
      label: 'Peta Kode',
      desc: 'Indeks RAG kode untuk AI agent',
      icon: 'i-lucide-map',
      exact: false
    },
    {
      id: 'audit-report',
      to: '/audit-report',
      label: 'Audit & Diagnostik',
      desc: 'Pemeriksaan menyeluruh: keamanan, kode, data & tampilan',
      icon: 'i-lucide-file-text',
      exact: false
    },
    {
      // Rumus tick & Big-O tiap rute/fungsi backend; dulu tab Detail Proyek, kini halaman sendiri.
      id: 'process-cost',
      to: '/process-cost',
      label: 'Biaya Proses',
      desc: 'Performa kode: Big-O & N+1 tiap rute',
      icon: 'i-lucide-sigma',
      exact: false
    },
    {
      // Router + server berjalan: rate limit (DDoS), SQL injection, header, CORS, auth — dibuktikan lewat respons nyata.
      id: 'security-test',
      to: '/security-test',
      label: 'Uji Keamanan',
      desc: 'Router & server live: DDoS, SQLi, header',
      icon: 'i-lucide-shield-check',
      exact: false
    },
    {
      id: 'runtime-test',
      to: '/runtime-test',
      label: 'Uji Runtime',
      desc: 'Kecepatan API, login & responsif',
      icon: 'i-lucide-laptop',
      exact: false
    },
    {
      // Statistik request aplikasi live (lewat vibeasik live) → saran rate limit, cache, LB & pecah service.
      id: 'traffic',
      to: '/traffic',
      label: 'Traffic Analyzer',
      desc: 'Trafik tinggi: rate limit, cache & skala',
      icon: 'i-lucide-activity',
      badge: 'Live',
      exact: false
    },
    {
      id: 'audit-learning',
      to: '/audit-learning',
      label: 'Pembelajaran Audit',
      desc: 'Vonis temuan yang keliru & benar',
      icon: 'i-lucide-graduation-cap',
      exact: false
    },
    {
      id: 'screenshots',
      to: '/screenshots',
      label: 'Screenshot & Visual',
      desc: 'Tampilan: snapshot & audit visual',
      icon: 'i-lucide-camera',
      exact: false
    },
    {
      id: 'queue',
      to: '/queue',
      label: 'Antrean Proses',
      desc: 'Todo & pipeline yang sedang berjalan',
      icon: 'i-lucide-list-todo',
      exact: false
    },
    {
      id: 'ports',
      to: '/ports',
      label: 'Monitor Port',
      desc: 'Pantau port bentrok live',
      icon: 'i-lucide-network',
      exact: false
    },
    {
      id: 'delivery',
      to: '/delivery',
      label: 'Delivery & Domain',
      desc: 'Beli domain & deploy hosting',
      icon: 'i-lucide-rocket',
      exact: false
    }
  ]
  return PROJECT_ORDER
    .map(id => items.find(i => i.id === id))
    .filter((i): i is NavItem => Boolean(i))
    .map(i => ({ ...i, group: i.group || PROJECT_GROUPS[i.id] }))
}

// Item proyek yang aktif untuk rute ini. Halaman Detail Aplikasi termasuk tab "Aplikasi".
export const isProjectNavActive = (item: NavItem, path: string, tab?: unknown): boolean => {
  if (item.isDetail) {
    if (!path.startsWith('/projects/')) return false
    const current = path.includes('/apps/') ? 'apps' : normalizeProjectTab(tab)
    return item.tab === current
  }
  if (item.exact) return path === item.to
  return path === item.to || path.startsWith(item.to + '/')
}

export const GLOBAL_NAV_ITEMS: NavItem[] = [
  {
    id: 'catalog',
    group: 'Proyek',
    to: '/cockpit',
    label: 'Katalog Proyek',
    desc: 'Pantau bundle & topologi',
    icon: 'i-lucide-layout-grid',
    exact: true
  },
  {
    id: 'adapters',
    group: 'Proyek',
    to: '/adapters',
    label: 'Katalog Adapter & Stack',
    desc: 'Nuxt, Go, Flutter & versi terbaru',
    icon: 'i-lucide-layers',
    exact: false
  },
  {
    id: 'organizer',
    group: 'Proyek',
    to: '/organizer',
    label: 'Tata Folder',
    desc: 'Workspace product-centric',
    icon: 'i-lucide-folder-tree',
    exact: false
  },
  {
    id: 'scan',
    group: 'Proyek',
    to: '/scan',
    label: 'Pindai Laptop',
    desc: 'Deteksi otomatis repo lokal',
    icon: 'i-lucide-folder-search',
    exact: false
  },
  {
    id: 'ai-hub',
    group: 'Agent & AI',
    to: '/ai-hub',
    label: 'AI Agent Hub',
    desc: 'Tanamkan System One & MCP',
    icon: 'i-lucide-sparkles',
    exact: false
  },
  {
    id: 'mcp',
    group: 'Agent & AI',
    to: '/mcp',
    label: 'AI & MCP Hub',
    desc: 'Protokol konteks AI agent',
    icon: 'i-lucide-bot',
    exact: false
  },
  {
    // Peta fungsi yang memanggil LLM luar: yang dikirim, respons yang ditunggu, fungsi lanjutan.
    id: 'llm-algorithm',
    group: 'Agent & AI',
    to: '/llm-algorithm',
    label: 'Algoritma LLM',
    desc: 'Cara vibeasik memanggil LLM luar',
    icon: 'i-lucide-workflow',
    exact: false
  },
  {
    // Log setiap request ke LLM luar per model: lama menunggu, gagal/berhasil, model lambat / tidak bisa.
    id: 'llm-log',
    group: 'Agent & AI',
    to: '/llm-log',
    label: 'Log LLM',
    desc: 'Request per model & kesehatannya',
    icon: 'i-lucide-activity',
    exact: false
  },
  {
    // Request LLM yang gagal berulang → dieksekusi dari lokal (agent CLI, MCP, atau ditempel manual).
    id: 'notifications',
    group: 'Agent & AI',
    to: '/notifications',
    label: 'Notifikasi',
    desc: 'Request LLM gagal, eksekusi lokal',
    icon: 'i-lucide-bell',
    exact: false
  },
  {
    id: 'orchestrator',
    group: 'Agent & AI',
    to: '/orchestrator',
    label: 'Orkestrator Agent',
    desc: 'Suruh agent di laptop bekerja',
    icon: 'i-lucide-network',
    exact: false
  },
  {
    id: 'agent-office',
    group: 'Agent & AI',
    to: '/agent-office',
    label: 'Kantor Agent',
    desc: 'Agent AI bekerja secara live',
    icon: 'i-lucide-gamepad-2',
    badge: 'Live',
    exact: false
  },
  {
    id: 'staff',
    group: 'Agent & AI',
    to: '/staff',
    label: 'Pegawai',
    desc: 'Agent berkepribadian & belajar',
    icon: 'i-lucide-id-card',
    exact: false
  },
  {
    // Terpisah dari Pegawai: dewan eksekutif terikat satu proyek, pegawai umum lintas proyek.
    id: 'executives',
    group: 'Agent & AI',
    to: '/executives',
    label: 'Pegawai Eksekutif',
    desc: 'Dewan CEO, CTO, PO & CMO per proyek',
    icon: 'i-lucide-crown',
    exact: false
  },
  {
    // Bukan milik satu proyek: satu pintu patokan & pengetahuan umum seluruh pekerja AI.
    id: 'knowledge',
    group: 'Pengetahuan',
    to: '/knowledge',
    label: 'Pusat Pengetahuan',
    desc: 'Satu pintu patokan & belajar AI',
    icon: 'i-lucide-brain-circuit',
    exact: false
  },
  {
    id: 'knowledge-graph',
    group: 'Pengetahuan',
    to: '/knowledge-graph',
    label: 'Jaring Pengetahuan',
    desc: 'Peta node & tautan pengetahuan',
    icon: 'i-lucide-waypoints',
    exact: false
  },
  {
    id: 'recap',
    group: 'Pengetahuan',
    to: '/recap',
    label: 'Rekap Progres',
    desc: 'Commit & todo harian',
    icon: 'i-lucide-trophy',
    exact: false
  },
  {
    // Bukan milik satu proyek: referensi desain dipakai bersama oleh semua proyek & agent AI.
    id: 'design-library',
    group: 'Pengetahuan',
    to: '/design-library',
    label: 'Galeri Desain',
    desc: 'Referensi UI modern untuk AI',
    icon: 'i-lucide-palette',
    exact: false
  },
  {
    id: 'design-lessons',
    group: 'Pengetahuan',
    to: '/design-lessons',
    label: 'Pelajaran Desain',
    desc: 'Before / after + prinsip desain',
    icon: 'i-lucide-git-compare-arrows',
    exact: false
  },
  {
    id: 'plans',
    group: 'Sistem',
    to: '/plans',
    label: 'Free vs Pro',
    desc: 'Perbandingan paket & akun',
    icon: 'i-lucide-scale',
    exact: false
  },
  {
    id: 'settings',
    group: 'Sistem',
    to: '/settings',
    label: 'Pengaturan',
    desc: 'LLM, folder & berbagi pengetahuan',
    icon: 'i-lucide-sliders-horizontal',
    exact: false
  }
]

// Judul halaman untuk navbar (breadcrumb): diambil dari label menu agar selalu sama dengan sidebar.
const EXTRA_TITLES: Record<string, string> = {
  '/account': 'Akun',
  '/upgrade': 'Upgrade ke Pro',
  '/knowledge-docs': 'Dokumen Pengetahuan',
  '/setup': 'Penyiapan Awal'
}

export const pageTitleFor = (path: string, tab?: unknown): { title: string, section: string } => {
  if (path.startsWith('/projects/')) {
    if (path.includes('/apps/')) return { title: 'Detail Aplikasi', section: 'Proyek aktif' }
    return { title: projectTabFor(normalizeProjectTab(tab))?.label || 'Detail Proyek', section: 'Proyek aktif' }
  }
  const projectItems = buildProjectNavItems(null, null).filter(i => !i.isDetail)
  const all = [...projectItems, ...GLOBAL_NAV_ITEMS]
  const hit = all.find(i => i.to === path) || all.find(i => i.to !== '/' && path.startsWith(i.to + '/'))
  if (hit) {
    const isProject = projectItems.some(i => i.id === hit.id)
    return { title: hit.label, section: isProject ? 'Proyek aktif' : (hit.group || 'Workspace') }
  }
  return { title: EXTRA_TITLES[path] || 'Workspace', section: 'Workspace' }
}
