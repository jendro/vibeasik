// Data menu navigasi sidebar (statis) + penyusun menu proyek aktif.

export interface NavItem {
  id: string
  to: string
  label: string
  desc: string
  icon: string
  exact?: boolean
  isDetail?: boolean
  badge?: string
  // Kelompok tampilan di sidebar (judul kecil di atas item pertama kelompok).
  group?: string
}

// Kelompok menu proyek aktif: alur kerja auditor — ringkas → audit → kerjakan → jalankan.
const PROJECT_GROUPS: Record<string, string> = {
  'home': 'Ringkasan',
  'project-detail': 'Ringkasan',
  'audit-report': 'Audit & mutu',
  'runtime-test': 'Audit & mutu',
  'security-test': 'Audit & mutu',
  'screenshots': 'Audit & mutu',
  'traffic': 'Audit & mutu',
  'audit-learning': 'Audit & mutu',
  'todos': 'Pengerjaan',
  'client-room': 'Pengerjaan',
  'code-map': 'Pengerjaan',
  'queue': 'Operasional',
  'ports': 'Operasional',
  'delivery': 'Operasional'
}
const PROJECT_ORDER = ['home', 'project-detail', 'audit-report', 'security-test', 'runtime-test', 'screenshots', 'traffic', 'audit-learning',
  'todos', 'client-room', 'code-map', 'queue', 'ports', 'delivery']

export const buildProjectNavItems = (activeProjectId: number | null, activeProject: { name: string } | null): NavItem[] => {
  const detailTo = activeProjectId ? `/projects/${activeProjectId}` : '/cockpit'
  const detailDesc = activeProject ? activeProject.name : 'Pilih proyek aktif'
  const detailBadge = activeProject ? 'Aktif' : undefined

  const items: NavItem[] = [
    {
      id: 'home',
      to: '/',
      label: 'Beranda',
      desc: 'Status & cara memakai',
      icon: 'i-lucide-house',
      exact: true
    },
    {
      id: 'project-detail',
      to: detailTo,
      label: 'Detail Proyek',
      desc: detailDesc,
      icon: 'i-lucide-folder-git-2',
      exact: false,
      isDetail: true,
      badge: detailBadge
    },
    {
      // Satu-satunya ruang obrolan pengguna (klien) dengan dewan eksekutif proyek aktif.
      id: 'client-room',
      to: '/client-room',
      label: 'Ruang Klien',
      desc: 'Diskusi & lapor bug ke dewan',
      icon: 'i-lucide-handshake',
      exact: false
    },
    {
      id: 'todos',
      to: '/todos',
      label: 'Perencanaan & Todo',
      desc: 'Roadmap & task bertingkat',
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
      desc: 'Temuan keamanan, API & code smell',
      icon: 'i-lucide-file-text',
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
      desc: 'API, login, kecepatan & responsif',
      icon: 'i-lucide-laptop',
      exact: false
    },
    {
      // Statistik request aplikasi live (lewat vibeasik live) → saran rate limit, cache, LB & pecah service.
      id: 'traffic',
      to: '/traffic',
      label: 'Traffic Analyzer',
      desc: 'Traffic live, rate limit & skala',
      icon: 'i-lucide-activity',
      badge: 'Live',
      exact: false
    },
    {
      id: 'audit-learning',
      to: '/audit-learning',
      label: 'Pembelajaran Audit',
      desc: 'Vonis audit keliru & benar',
      icon: 'i-lucide-graduation-cap',
      exact: false
    },
    {
      id: 'screenshots',
      to: '/screenshots',
      label: 'Screenshot & Visual',
      desc: 'Galeri snapshot UI proyek',
      icon: 'i-lucide-camera',
      exact: false
    },
    {
      id: 'queue',
      to: '/queue',
      label: 'Antrean Proses',
      desc: 'Monitor antrean & pipeline',
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
    .map(i => ({ ...i, group: PROJECT_GROUPS[i.id] }))
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

export const pageTitleFor = (path: string): { title: string, section: string } => {
  if (path.startsWith('/projects/')) {
    return { title: path.includes('/apps/') ? 'Detail Aplikasi' : 'Detail Proyek', section: 'Proyek aktif' }
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
