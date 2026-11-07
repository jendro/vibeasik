// Fitur berbayar (kunci entitlement) dan halaman Studio milik masing-masing fitur.
// Kunci sama dengan katalog route di engine (pkg/registry/catalog.go).

export interface ProFeatureInfo {
  label: string
  description: string
  icon: string
}

export const PRO_FEATURES: Record<string, ProFeatureInfo> = {
  'audit.deep': { label: 'Deep Audit', description: 'Audit mendalam lintas komponen: kompleksitas, error yang tidak ditangani, rate limit, dan risiko arsitektur.', icon: 'i-lucide-scan-search' },
  'audit.ai_review': { label: 'AI Review & Profil Proyek', description: 'Review temuan dengan LLM, penyelesaian otomatis, profil & peta sinergi proyek, obrolan dengan pengetahuan proyek.', icon: 'i-lucide-sparkles' },
  'audit.learning': { label: 'Pembelajaran Audit', description: 'Vonis temuan keliru/benar dipelajari sehingga audit berikutnya makin tepat.', icon: 'i-lucide-graduation-cap' },
  'runtime_sim': { label: 'Uji Runtime & Visual', description: 'Simulasi pengguna di browser, tangkapan layar, dan audit visual multimodal.', icon: 'i-lucide-monitor-play' },
  'staff': { label: 'Pegawai AI', description: 'Pegawai berkepribadian & berpengetahuan sendiri, dewan eksekutif proyek, agenda, dan review QA.', icon: 'i-lucide-users' },
  'planner': { label: 'Rapat Project Manager', description: 'Rapat perencanaan dengan spesialis yang memutuskan todo berikutnya.', icon: 'i-lucide-calendar-check' },
  'orchestrator': { label: 'Orkestrator & Antrean', description: 'Kirim todo ke agent coding di laptop, antrean job, dan pipeline otomatis.', icon: 'i-lucide-workflow' },
  'llm.router': { label: 'LLM Router', description: 'Uji koneksi, model vision, dan penimbang pemilihan model.', icon: 'i-lucide-route' },
  'design_library': { label: 'Galeri & Pelajaran Desain', description: 'Referensi desain, prinsip, resep, dan pelajaran before/after yang menjadi pengetahuan AI.', icon: 'i-lucide-palette' },
  'office': { label: 'Kantor Agent', description: 'Kantor 3D agent, rekap progres, dan Ruang Klien.', icon: 'i-lucide-building-2' }
}

// Halaman Studio yang membutuhkan fitur berbayar.
export const PRO_PAGES: Record<string, string> = {
  '/agent-office': 'office',
  '/recap': 'office',
  '/client-room': 'office',
  '/ai-hub': 'orchestrator',
  '/orchestrator': 'orchestrator',
  '/queue': 'orchestrator',
  '/staff': 'staff',
  '/executives': 'staff',
  '/audit-learning': 'audit.learning',
  '/runtime-test': 'runtime_sim',
  '/screenshots': 'runtime_sim',
  '/design-library': 'design_library',
  '/design-lessons': 'design_library'
}

export const proFeatureForPath = (path: string): string | undefined => {
  const clean = path.replace(/\/+$/, '') || '/'
  return PRO_PAGES[clean]
}
