import type { LLMConfig } from '~/types'

// Nama tampilan model LLM untuk seluruh Studio. Request ke LLM luar tetap memakai id model asli; di layar
// pengguna cukup melihat nama modelnya (alias / display_name), bukan id penyedia seperti "vendor/model-x".
// Peta alias diambil sekali dari semua konfigurasi LLM dan dipakai bersama semua komponen.

let loading: Promise<void> | null = null

export const useModelNames = () => {
  const aliasMap = useState<Record<string, string>>('llm_model_alias_map', () => ({}))
  const loaded = useState<boolean>('llm_model_alias_loaded', () => false)
  const { getLLMConfigs } = useApi()

  const setFromConfigs = (configs: LLMConfig[]) => {
    const map: Record<string, string> = {}
    for (const cfg of configs) {
      for (const [model, alias] of Object.entries(cfg.aliases || {})) {
        if (!alias) continue
        // Konfigurasi aktif menang bila model yang sama diberi alias berbeda.
        if (!map[model] || cfg.is_active) map[model] = alias
        map[`${cfg.id}:${model}`] = alias
      }
    }
    aliasMap.value = map
    loaded.value = true
  }

  const refreshModelNames = async () => {
    if (import.meta.server) return
    if (!loading) {
      loading = getLLMConfigs()
        .then(setFromConfigs)
        .catch(() => { /* engine offline: tampilkan id apa adanya */ })
        .finally(() => { loading = null })
    }
    await loading
  }

  if (import.meta.client && !loaded.value && !loading) void refreshModelNames()

  /**
   * Nama tampilan sebuah model. Menerima id model ("vendor/model-x"), ref pool ("12:vendor/model-x"),
   * atau label "model · konfigurasi"; tanpa alias, teks aslinya dikembalikan.
   */
  const modelName = (model?: string | null): string => {
    if (!model) return ''
    const raw = String(model).trim()
    const map = aliasMap.value
    if (map[raw]) return map[raw]
    const ref = raw.match(/^\d+:(.+)$/)
    if (ref?.[1] && map[ref[1]]) return map[ref[1]]
    const sep = raw.indexOf(' · ')
    if (sep > 0) {
      const head = raw.slice(0, sep)
      if (map[head]) return map[head] + raw.slice(sep)
    }
    return raw
  }

  return { modelName, refreshModelNames, setModelNamesFromConfigs: setFromConfigs }
}
