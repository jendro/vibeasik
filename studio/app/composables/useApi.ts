import { useApiBase } from './api/base'
import { useProjectsApi } from './api/projects'
import { useDiagnosticsApi } from './api/diagnostics'
import { useKnowledgeApi } from './api/knowledge'
import { useLlmApi } from './api/llm'
import { useStaffApi } from './api/staff'
import { useAgentsApi } from './api/agents'
import { useQueueApi } from './api/queue'
import { useTodosApi } from './api/todos'
import { useSystemApi } from './api/system'
import { useInsightApi } from './api/insight'

export * from '../types'

// Fasad: objek kembaliannya identik dengan useApi lama (apiBase + semua fungsi),
// tetapi implementasinya dipecah per domain di composables/api/.
// Pemanggil baru dianjurkan memakai composable domain langsung agar impor lebih selektif.
export const useApi = () => {
  const apiBase = useApiBase()
  return {
    apiBase,
    ...useProjectsApi(),
    ...useDiagnosticsApi(),
    ...useKnowledgeApi(),
    ...useLlmApi(),
    ...useStaffApi(),
    ...useAgentsApi(),
    ...useQueueApi(),
    ...useTodosApi(),
    ...useSystemApi(),
    ...useInsightApi()
  }
}
