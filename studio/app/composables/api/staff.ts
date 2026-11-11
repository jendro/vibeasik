// Domain API "staff" — dipecah dari useApi.ts tanpa mengubah kontrak HTTP.
import { useApiBase } from './base'

export const useStaffApi = () => {
  const apiBase = useApiBase()

  // Pegawai vibeasik: agent berkepribadian & berpengetahuan sendiri, review QA, obrolan
  const getStaff = async (): Promise<StaffOverview> => await $fetch<StaffOverview>(`${apiBase}/api/staff`)

  const getEmployee = async (id: number): Promise<EmployeeDetail> => await $fetch<EmployeeDetail>(`${apiBase}/api/staff/${id}`)

  const getEmployeePlaybook = async (id: number): Promise<EmployeePlaybook> => await $fetch<EmployeePlaybook>(`${apiBase}/api/staff/${id}/playbook`)

  const exportEmployeePackage = async (id: number): Promise<EmployeePackageExport> =>
    await $fetch<EmployeePackageExport>(`${apiBase}/api/staff/${id}/package`, { method: 'POST' })

  const hireEmployee = async (role: string): Promise<Employee> => await $fetch<Employee>(`${apiBase}/api/staff`, { method: 'POST', body: { role } })

  const updateEmployee = async (id: number, body: Partial<Employee>): Promise<Employee> => await $fetch<Employee>(`${apiBase}/api/staff/${id}`, { method: 'PUT', body })

  const resignEmployee = async (id: number) => await $fetch(`${apiBase}/api/staff/${id}`, { method: 'DELETE' })

  const teachEmployee = async (id: number, body: { topic: string, content: string, kind?: string, weight?: number }): Promise<EmployeeKnowledge> =>
    await $fetch<EmployeeKnowledge>(`${apiBase}/api/staff/${id}/knowledge`, { method: 'POST', body })

  const consolidateKnowledge = async (id: number): Promise<{ merged_count: number }> =>
    await $fetch<{ merged_count: number }>(`${apiBase}/api/staff/${id}/knowledge/consolidate`, { method: 'POST' })

  const forgetKnowledge = async (id: number, kid: number, reason?: string) =>
    await $fetch(`${apiBase}/api/staff/${id}/knowledge/${kid}`, { method: 'DELETE', query: { reason } })

  const getEmployeeChat = async (id: number): Promise<EmployeeChatMessage[]> => await $fetch<EmployeeChatMessage[]>(`${apiBase}/api/staff/${id}/chat`)

  const sendEmployeeChat = async (id: number, message: string): Promise<{ message: EmployeeChatMessage, job_id: string, warning?: string }> =>
    await $fetch(`${apiBase}/api/staff/${id}/chat`, { method: 'POST', body: { message } })

  const clearEmployeeChat = async (id: number) => await $fetch(`${apiBase}/api/staff/${id}/chat`, { method: 'DELETE' })

  const requestEmployeeReview = async (id: number): Promise<{ job_id: string }> => await $fetch(`${apiBase}/api/staff/${id}/review`, { method: 'POST' })

  const getStaffTeams = async (): Promise<StaffTeam[]> => await $fetch<StaffTeam[]>(`${apiBase}/api/staff/teams`)

  const setStaffLeader = async (id: number, lead: boolean): Promise<StaffTeam[]> =>
    await $fetch<StaffTeam[]>(`${apiBase}/api/staff/${id}/leader`, { method: 'PUT', body: { lead } })

  const updateStaffSettings = async (body: StaffSettings): Promise<StaffSettings> => await $fetch<StaffSettings>(`${apiBase}/api/staff/settings`, { method: 'PUT', body })

  // Eksekutif proyek (CEO, CTO, Product Owner)
  const getExecutives = async (query: { project_id?: number } = {}): Promise<{ executives: ProjectExecutive[], roles: ExecRole[] }> =>
    await $fetch<{ executives: ProjectExecutive[], roles: ExecRole[] }>(`${apiBase}/api/staff/executives`, { query })

  const getExecutive = async (id: number): Promise<ExecutiveDetail> =>
    await $fetch<ExecutiveDetail>(`${apiBase}/api/staff/executives/${id}`)

  const provisionExecutives = async (projectId: number): Promise<{ project_id: number, project_name: string, executives: ProjectExecutive[] }> =>
    await $fetch<{ project_id: number, project_name: string, executives: ProjectExecutive[] }>(`${apiBase}/api/staff/executives/provision/${projectId}`, { method: 'POST' })

  // Agenda pribadi pegawai (terpisah dari todo proyek) & pool LLM
  const getAgendas = async (query: { employee_id?: number, status?: string, limit?: number } = {}): Promise<AgendaOverview> =>
    await $fetch<AgendaOverview>(`${apiBase}/api/staff/agenda`, { query })

  const addAgenda = async (employeeId: number, body: AgendaInput): Promise<EmployeeAgenda> =>
    await $fetch<EmployeeAgenda>(`${apiBase}/api/staff/${employeeId}/agenda`, { method: 'POST', body })

  const startAgenda = async (id: number): Promise<EmployeeAgenda> => await $fetch<EmployeeAgenda>(`${apiBase}/api/staff/agenda/${id}/start`, { method: 'POST' })

  const dropAgenda = async (id: number) => await $fetch(`${apiBase}/api/staff/agenda/${id}`, { method: 'DELETE' })

  // Obrolan santai antar pegawai (model obrolan ringan)
  const getHR = async (query: { employee_id?: number, kind?: string, status?: string, limit?: number } = {}): Promise<HROverview> =>
    await $fetch<HROverview>(`${apiBase}/api/staff/hr`, { query })

  const evaluateHR = async (): Promise<{ created: EmployeeHRAction[] }> => await $fetch(`${apiBase}/api/staff/hr/evaluate`, { method: 'POST' })

  const decideHR = async (id: number, approve: boolean, note = ''): Promise<EmployeeHRAction> =>
    await $fetch<EmployeeHRAction>(`${apiBase}/api/staff/hr/${id}/${approve ? 'approve' : 'reject'}`, { method: 'POST', body: { note } })

  const getTurnovers = async (query: { limit?: number } = {}): Promise<{ turnovers: EmployeeTurnover[] }> =>
    await $fetch<{ turnovers: EmployeeTurnover[] }>(`${apiBase}/api/staff/turnovers`, { query })

  // Antrean perbaikan tertunda (todo selesai bernilai kurang, dibuka lagi setelah skill naik)
  const getRevisits = async (query: { status?: string, cause?: string, employee_id?: number, limit?: number } = {}): Promise<RevisitOverview> =>
    await $fetch<RevisitOverview>(`${apiBase}/api/staff/revisits`, { query })

  const processRevisits = async (): Promise<RevisitReport> => await $fetch<RevisitReport>(`${apiBase}/api/staff/revisits/process`, { method: 'POST' })

  const revisitAction = async (id: number, action: 'reopen' | 'dismiss', note = ''): Promise<EmployeeRevisit> =>
    await $fetch<EmployeeRevisit>(`${apiBase}/api/staff/revisits/${id}/${action}`, { method: 'POST', body: { note } })

  // Konsultasi agent coding dengan pegawai pendamping todo, beserta ringkasan mutu arahannya
  const getConsults = async (query: { todo_id?: number, employee_id?: number, phase?: string, state?: string, limit?: number } = {}): Promise<ConsultOverview> =>
    await $fetch<ConsultOverview>(`${apiBase}/api/staff/consults`, { query })

  const rateConsult = async (id: number, rating: -1 | 0 | 1, note = ''): Promise<EmployeeConsult> =>
    await $fetch<EmployeeConsult>(`${apiBase}/api/staff/consults/${id}/rate`, { method: 'POST', body: { rating, note } })

  const getConversations = async (query: { employee_id?: number, limit?: number } = {}): Promise<ConversationOverview> =>
    await $fetch<ConversationOverview>(`${apiBase}/api/staff/conversations`, { query })

  const startConversation = async (aId: number, bId: number): Promise<EmployeeConversation> =>
    await $fetch<EmployeeConversation>(`${apiBase}/api/staff/conversations`, { method: 'POST', body: { a_id: aId, b_id: bId } })

  const assignTodoEmployee = async (todoId: number, employeeId?: number | null) =>
    await $fetch(`${apiBase}/api/todos/${todoId}/employee`, { method: 'PUT', body: employeeId === undefined ? {} : { employee_id: employeeId ?? 0 } })

  const chatProjectExecutives = async (projectId: number, payload: ExecutiveChatRequestDTO): Promise<ExecutiveChatResponseDTO> => {
    return await $fetch<ExecutiveChatResponseDTO>(`${apiBase}/api/projects/${projectId}/executives/chat`, {
      method: 'POST',
      body: payload
    })
  }

  const getProjectExecutivesChatHistory = async (projectId: number): Promise<ExecutiveChatMessageDTO[]> => {
    return await $fetch<ExecutiveChatMessageDTO[]>(`${apiBase}/api/projects/${projectId}/executives/chat/history`)
  }

  const clearProjectExecutivesChatHistory = async (projectId: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/projects/${projectId}/executives/chat/history`, {
      method: 'DELETE'
    })
  }

  const getProjectExecutiveKnowledge = async (projectId: number): Promise<ExecutiveKnowledgeItem[]> => {
    return await $fetch<ExecutiveKnowledgeItem[]>(`${apiBase}/api/projects/${projectId}/executives/knowledge`)
  }

  const createProjectExecutiveKnowledge = async (projectId: number, payload: Partial<ExecutiveKnowledgeItem>): Promise<ExecutiveKnowledgeItem> => {
    return await $fetch<ExecutiveKnowledgeItem>(`${apiBase}/api/projects/${projectId}/executives/knowledge`, {
      method: 'POST',
      body: payload
    })
  }

  const deleteProjectExecutiveKnowledge = async (projectId: number, knowledgeId: number): Promise<{ message: string }> => {
    return await $fetch<{ message: string }>(`${apiBase}/api/projects/${projectId}/executives/knowledge/${knowledgeId}`, {
      method: 'DELETE'
    })
  }

  const analyzeProjectTodosByExecutives = async (projectId: number): Promise<ExecutiveTodoAnalysisResult> => {
    return await $fetch<ExecutiveTodoAnalysisResult>(`${apiBase}/api/projects/${projectId}/executives/todos/analyze`, {
      method: 'POST'
    })
  }

  const batchActionProjectTodosByExecutives = async (
    projectId: number,
    payload: { action: string, todo_ids: number[], new_priority?: string, reason?: string, executive_role?: string }
  ): Promise<{ message: string, action: string, affected: number, todo_ids: number[] }> => {
    return await $fetch<{ message: string, action: string, affected: number, todo_ids: number[] }>(`${apiBase}/api/projects/${projectId}/executives/todos/batch-action`, {
      method: 'POST',
      body: payload
    })
  }

  const triggerExecutiveAutonomousCycle = async (projectId: number, payload?: { invited_employee_ids?: number[], topic?: string }): Promise<any> => {
    return await $fetch<any>(`${apiBase}/api/projects/${projectId}/executives/autonomous/cycle`, {
      method: 'POST',
      body: payload
    })
  }

  const getExecutiveAutonomousStatus = async (projectId: number): Promise<ExecutiveAutonomousStatus> => {
    return await $fetch<ExecutiveAutonomousStatus>(`${apiBase}/api/projects/${projectId}/executives/autonomous/status`)
  }

  const getCurrentExecutiveMeeting = async (projectId: number): Promise<{ active_meeting: ActiveExecutiveMeeting | null }> => {
    return await $fetch<{ active_meeting: ActiveExecutiveMeeting | null }>(`${apiBase}/api/projects/${projectId}/executives/meeting/current`)
  }

  const getEligibleMeetingStaff = async (projectId: number): Promise<OfficeEmployee[]> => {
    return await $fetch<OfficeEmployee[]>(`${apiBase}/api/projects/${projectId}/executives/meeting/eligible-staff`)
  }

  return {
    getStaff,
    getEmployee,
    getEmployeePlaybook,
    exportEmployeePackage,
    hireEmployee,
    updateEmployee,
    resignEmployee,
    teachEmployee,
    consolidateKnowledge,
    forgetKnowledge,
    getEmployeeChat,
    sendEmployeeChat,
    clearEmployeeChat,
    requestEmployeeReview,
    getStaffTeams,
    setStaffLeader,
    updateStaffSettings,
    getExecutives,
    getExecutive,
    provisionExecutives,
    getAgendas,
    addAgenda,
    startAgenda,
    dropAgenda,
    getHR,
    evaluateHR,
    decideHR,
    getTurnovers,
    getRevisits,
    processRevisits,
    revisitAction,
    getConsults,
    rateConsult,
    getConversations,
    startConversation,
    assignTodoEmployee,
    chatProjectExecutives,
    getProjectExecutivesChatHistory,
    clearProjectExecutivesChatHistory,
    getProjectExecutiveKnowledge,
    createProjectExecutiveKnowledge,
    deleteProjectExecutiveKnowledge,
    analyzeProjectTodosByExecutives,
    batchActionProjectTodosByExecutives,
    triggerExecutiveAutonomousCycle,
    getExecutiveAutonomousStatus,
    getCurrentExecutiveMeeting,
    getEligibleMeetingStaff
  }
}
