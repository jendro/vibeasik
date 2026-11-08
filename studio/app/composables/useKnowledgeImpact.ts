// Dampak pengetahuan di seluruh pengguna (angka teragregasi dari server): dipakai di berapa pekerjaan besar dan nilainya.
// Ringan: ringkasan kecil diambil sekali (di-cache di Studio, engine, dan server); data per node hanya saat node dibuka.
// Nama pekerjaan/proyek pengguna tidak pernah ada di sini.
export interface ImpactStat { uses: number, helpful: number, harmful: number }
export interface ImpactKind extends ImpactStat { kind: string, label: string, score: number }
export interface ImpactDomain extends ImpactStat { domain: string, score: number }
export interface KnowledgeImpact {
  devices: number
  min_devices: number
  total: ImpactStat
  score: number
  by_kind: ImpactKind[]
  by_domain: ImpactDomain[]
}
export interface NodeImpact {
  available: boolean
  min_devices: number
  node?: ImpactStat & { key: string, score: number, by_kind: ImpactKind[] }
}

export function useKnowledgeImpact() {
  const { apiBase } = useApi()
  const impact = useState<KnowledgeImpact | null>('knowledge-impact', () => null)
  const loading = useState('knowledge-impact-loading', () => false)
  const error = useState('knowledge-impact-error', () => '')
  const nodes = useState<Record<string, NodeImpact>>('knowledge-impact-nodes', () => ({}))

  const load = async (force = false) => {
    if (impact.value && !force) return
    loading.value = true
    try {
      impact.value = await $fetch<KnowledgeImpact>(`${apiBase}/api/knowledge-sync/impact`)
      error.value = ''
    } catch (e: any) {
      error.value = e?.data?.error || e?.message || 'Gagal memuat dampak'
    } finally {
      loading.value = false
    }
  }

  // Kunci node lokal "hub:guideline:SEC-001" ↔ kunci server "guideline:SEC-001".
  const serverKey = (key?: string) => (key || '').replace(/^hub:/, '')

  const loadNode = async (key?: string): Promise<NodeImpact | null> => {
    const k = serverKey(key)
    if (!k) return null
    if (nodes.value[k]) return nodes.value[k]
    const n = await $fetch<NodeImpact>(`${apiBase}/api/knowledge-sync/impact`, { query: { key: k } })
    nodes.value = { ...nodes.value, [k]: n }
    return n
  }

  return { impact, loading, error, load, loadNode, serverKey }
}
