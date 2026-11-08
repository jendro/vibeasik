// Dampak pengetahuan di seluruh pengguna (angka teragregasi dari server): dipakai di berapa pekerjaan besar dan nilainya.
// Server hanya menampilkan angka yang berasal dari cukup banyak perangkat; nama pekerjaan tidak pernah ada di sini.
export interface ImpactStat { uses: number, helpful: number, harmful: number }
export interface ImpactKind extends ImpactStat { kind: string, label: string, score: number }
export interface ImpactDomain extends ImpactStat { domain: string, score: number }
export interface ImpactNode extends ImpactStat { key: string, score: number, by_kind: ImpactKind[] }
export interface KnowledgeImpact {
  devices: number
  min_devices: number
  total: ImpactStat
  score: number
  by_kind: ImpactKind[]
  by_domain: ImpactDomain[]
  nodes: ImpactNode[]
}

export function useKnowledgeImpact() {
  const { apiBase } = useApi()
  const impact = useState<KnowledgeImpact | null>('knowledge-impact', () => null)
  const loading = useState('knowledge-impact-loading', () => false)
  const error = useState('knowledge-impact-error', () => '')

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
  const forNode = (key?: string) => {
    const k = (key || '').replace(/^hub:/, '')
    return impact.value?.nodes.find(n => n.key === k) || null
  }

  return { impact, loading, error, load, forNode }
}
