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
}

export const buildProjectNavItems = (activeProjectId: number | null, activeProject: { name: string } | null): NavItem[] => {
  const detailTo = activeProjectId ? `/projects/${activeProjectId}` : '/cockpit'
  const detailDesc = activeProject ? activeProject.name : 'Pilih proyek aktif'
  const detailBadge = activeProject ? 'Aktif' : undefined

  return [
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
      desc: 'Query, router & smelly code',
      icon: 'i-lucide-file-text',
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
}

export const GLOBAL_NAV_ITEMS: NavItem[] = [
  {
    id: 'catalog',
    to: '/cockpit',
    label: 'Katalog Proyek',
    desc: 'Pantau bundle & topologi',
    icon: 'i-lucide-layout-grid',
    exact: true
  },
  {
    id: 'adapters',
    to: '/adapters',
    label: 'Katalog Adapter & Stack',
    desc: 'Nuxt, Go, Flutter & versi terbaru',
    icon: 'i-lucide-layers',
    exact: false
  },
  {
    id: 'organizer',
    to: '/organizer',
    label: 'Tata Folder',
    desc: 'Workspace product-centric',
    icon: 'i-lucide-folder-tree',
    exact: false
  },
  {
    id: 'scan',
    to: '/scan',
    label: 'Pindai Laptop',
    desc: 'Deteksi otomatis repo lokal',
    icon: 'i-lucide-folder-search',
    exact: false
  },
  {
    id: 'ai-hub',
    to: '/ai-hub',
    label: 'AI Agent Hub',
    desc: 'Tanamkan System One & MCP',
    icon: 'i-lucide-sparkles',
    exact: false
  },
  {
    id: 'mcp',
    to: '/mcp',
    label: 'AI & MCP Hub',
    desc: 'Protokol konteks AI agent',
    icon: 'i-lucide-bot',
    exact: false
  },
  {
    // Peta fungsi yang memanggil LLM luar: yang dikirim, respons yang ditunggu, fungsi lanjutan.
    id: 'llm-algorithm',
    to: '/llm-algorithm',
    label: 'Algoritma LLM',
    desc: 'Cara vibeasik memanggil LLM luar',
    icon: 'i-lucide-workflow',
    exact: false
  },
  {
    id: 'orchestrator',
    to: '/orchestrator',
    label: 'Orkestrator Agent',
    desc: 'Suruh agent di laptop bekerja',
    icon: 'i-lucide-network',
    exact: false
  },
  {
    id: 'agent-office',
    to: '/agent-office',
    label: 'Kantor Agent',
    desc: 'Agent AI bekerja secara live',
    icon: 'i-lucide-gamepad-2',
    badge: 'Live',
    exact: false
  },
  {
    id: 'staff',
    to: '/staff',
    label: 'Pegawai',
    desc: 'Agent berkepribadian & belajar',
    icon: 'i-lucide-id-card',
    exact: false
  },
  {
    // Terpisah dari Pegawai: dewan eksekutif terikat satu proyek, pegawai umum lintas proyek.
    id: 'executives',
    to: '/executives',
    label: 'Pegawai Eksekutif',
    desc: 'Dewan CEO, CTO, PO & CMO per proyek',
    icon: 'i-lucide-crown',
    exact: false
  },
  {
    // Bukan milik satu proyek: satu pintu patokan & pengetahuan umum seluruh pekerja AI.
    id: 'knowledge',
    to: '/knowledge',
    label: 'Pusat Pengetahuan',
    desc: 'Satu pintu patokan & belajar AI',
    icon: 'i-lucide-brain-circuit',
    exact: false
  },
  {
    id: 'knowledge-graph',
    to: '/knowledge-graph',
    label: 'Jaring Pengetahuan',
    desc: 'Peta node & tautan pengetahuan',
    icon: 'i-lucide-waypoints',
    exact: false
  },
  {
    id: 'recap',
    to: '/recap',
    label: 'Rekap Progres',
    desc: 'Commit & todo harian',
    icon: 'i-lucide-trophy',
    exact: false
  },
  {
    // Bukan milik satu proyek: referensi desain dipakai bersama oleh semua proyek & agent AI.
    id: 'design-library',
    to: '/design-library',
    label: 'Galeri Desain',
    desc: 'Referensi UI modern untuk AI',
    icon: 'i-lucide-palette',
    exact: false
  },
  {
    id: 'design-lessons',
    to: '/design-lessons',
    label: 'Pelajaran Desain',
    desc: 'Before / after + prinsip desain',
    icon: 'i-lucide-git-compare-arrows',
    exact: false
  },
  {
    id: 'plans',
    to: '/plans',
    label: 'Free vs Pro',
    desc: 'Perbandingan paket & akun',
    icon: 'i-lucide-scale',
    exact: false
  },
  {
    id: 'settings',
    to: '/settings',
    label: 'Pengaturan',
    desc: 'LLM, folder & berbagi pengetahuan',
    icon: 'i-lucide-sliders-horizontal',
    exact: false
  }
]
