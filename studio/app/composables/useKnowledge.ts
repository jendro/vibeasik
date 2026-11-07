import type { KnowledgeGap, KnowledgeHit, KnowledgeNoteInput, KnowledgeOverview, KnowledgeRecallResult } from '~/types'

const errMsg = (err: any) => err?.data?.error || err?.message || 'Engine tidak dapat dihubungi'

/**
 * Pusat Pengetahuan (GET /api/knowledge): satu pintu patokan guideline, checklist audit, pelajaran vonis audit,
 * keahlian pegawai, dan hasil belajar teknologi baru — dipakai audit, rapat PM, dan pegawai lewat temu-kembali
 * berbobot (relevansi vektor + bukti helpful/harmful + kepentingan + kebaruan).
 */
export function useKnowledge() {
  const api = useApi()
  const toast = useToast()
  const overview = ref<KnowledgeOverview | null>(null)
  const nodes = ref<KnowledgeHit[]>([])
  const loading = ref(true)
  const nodesLoading = ref(false)
  const curating = ref(false)
  const error = ref<string | null>(null)
  // 'all' = tanpa filter (USelect tidak menerima nilai string kosong)
  const filter = reactive({ q: '', domain: 'all', kind: 'all', source: 'all', cluster_id: 0 })
  const pick = (v: string) => (v && v !== 'all' ? v : undefined)

  const load = async () => {
    try {
      overview.value = await api.getKnowledge()
      error.value = null
    } catch (err: any) {
      error.value = errMsg(err)
    } finally {
      loading.value = false
    }
  }

  const loadNodes = async () => {
    nodesLoading.value = true
    try {
      nodes.value = await api.listKnowledgeNodes({
        q: filter.q || undefined, domain: pick(filter.domain), kind: pick(filter.kind),
        source: pick(filter.source), cluster_id: filter.cluster_id || undefined, limit: 150
      })
    } catch (err: any) {
      toast.add({ title: 'Gagal memuat node', description: errMsg(err), icon: 'i-lucide-alert-circle', color: 'error' })
    } finally {
      nodesLoading.value = false
    }
  }

  const refresh = () => Promise.all([load(), loadNodes()])

  const act = async (fn: () => Promise<unknown>, ok: string, fail: string) => {
    try {
      await fn()
      toast.add({ title: ok, icon: 'i-lucide-check', color: 'success' })
      await refresh()
      return true
    } catch (err: any) {
      toast.add({ title: fail, description: errMsg(err), icon: 'i-lucide-alert-circle', color: 'error' })
      return false
    }
  }

  const teach = async (body: KnowledgeNoteInput) => {
    try {
      const res = await api.teachKnowledge(body)
      toast.add({
        title: res.merged ? 'Butir serupa sudah ada — diperkuat' : 'Pengetahuan ditambahkan ke pustaka kantor',
        description: `K${res.node.id} · ${res.node.title}`, icon: 'i-lucide-book-plus', color: 'success'
      })
      await refresh()
      return true
    } catch (err: any) {
      toast.add({ title: 'Gagal menambah pengetahuan', description: errMsg(err), icon: 'i-lucide-alert-circle', color: 'error' })
      return false
    }
  }

  const feedback = (n: KnowledgeHit, helpful: boolean) =>
    act(() => api.knowledgeFeedback(n.id, helpful), helpful ? `K${n.id} ditandai membantu` : `K${n.id} ditandai keliru`, 'Gagal mencatat umpan balik')
  const retire = (n: KnowledgeHit) => act(() => api.retireKnowledgeNode(n.id, 'Dipensiunkan dari studio'), `K${n.id} dipensiunkan`, 'Gagal memensiunkan node')
  const reportGap = (body: { topic: string, domain?: string }) => act(() => api.reportKnowledgeGap(body), 'Celah dicatat — pegawai yang cocok akan mempelajarinya', 'Gagal mencatat celah')
  const dismissGap = (g: KnowledgeGap) => act(() => api.dismissKnowledgeGap(g.id), 'Celah diabaikan', 'Gagal mengabaikan celah')

  const curate = async () => {
    curating.value = true
    try {
      const r = await api.curateKnowledge()
      toast.add({
        title: `Kurasi selesai (${r.took})`,
        description: `${r.nodes} node · ${r.synced} disinkron · ${r.merged} dilebur · ${r.promoted} dipromosikan · ${r.retired} dipensiunkan · ${r.links} tautan · ${r.clusters} klaster`,
        icon: 'i-lucide-sparkles', color: 'success'
      })
      await refresh()
    } catch (err: any) {
      toast.add({ title: 'Kurasi gagal', description: errMsg(err), icon: 'i-lucide-alert-circle', color: 'error' })
    } finally {
      curating.value = false
    }
  }

  const recall = async (text: string, employeeId?: number): Promise<KnowledgeRecallResult | null> => {
    try {
      return await api.recallKnowledge({ text, employee_id: employeeId || undefined, limit: 12 })
    } catch (err: any) {
      toast.add({ title: 'Uji temu-kembali gagal', description: errMsg(err), icon: 'i-lucide-alert-circle', color: 'error' })
      return null
    }
  }

  return { overview, nodes, loading, nodesLoading, curating, error, filter, load, loadNodes, refresh, teach, feedback, retire, reportGap, dismissGap, curate, recall }
}
