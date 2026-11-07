import type { Project } from '~/types'

export const useActiveProject = () => {
  const { getProjects } = useApi()

  // Globally shared state across Nuxt components
  const projects = useState<Project[]>('active_project_list', () => [])
  const activeProjectId = useState<number | null>('active_project_id', () => null)
  const loadingProjects = useState<boolean>('loading_active_projects', () => false)

  const activeProject = computed(() => {
    if (!activeProjectId.value) return null
    return projects.value.find(p => p.id === activeProjectId.value) || null
  })

  const fetchProjects = async (force = false) => {
    if (projects.value.length > 0 && !force) return
    loadingProjects.value = true
    try {
      const data = await getProjects()
      projects.value = data || []

      // If activeProjectId is not set, try restoring from localStorage
      if (import.meta.client && activeProjectId.value === null) {
        const saved = localStorage.getItem('vibeasik_active_project_id')
        if (saved) {
          const parsed = Number(saved)
          if (data.some(p => p.id === parsed)) {
            activeProjectId.value = parsed
          }
        }
      }
    } catch (err) {
      console.error('Failed to fetch projects in useActiveProject:', err)
    } finally {
      loadingProjects.value = false
    }
  }

  const selectActiveProject = (id: number | null) => {
    activeProjectId.value = id
    if (import.meta.client) {
      if (id !== null && id !== undefined && id > 0) {
        localStorage.setItem('vibeasik_active_project_id', String(id))
      } else {
        localStorage.removeItem('vibeasik_active_project_id')
      }
    }
  }

  return {
    projects,
    activeProjectId,
    activeProject,
    loadingProjects,
    fetchProjects,
    selectActiveProject
  }
}
