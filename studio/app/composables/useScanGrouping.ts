import type { ScanResult, Project, ManualProjectGroup, ManualComponentItem } from '~/types'

export const useScanGrouping = () => {
  const router = useRouter()
  const toast = useToast()
  const { groupScanComponents, createProject, addComponent, getProjects } = useApi()

  const loadingAi = ref(false)
  const llmUsed = ref(false)
  const aiSummary = ref('')
  const manualGroups = ref<ManualProjectGroup[]>([])
  const existingProjects = ref<Project[]>([])
  const importing = ref(false)

  const loadExistingProjects = async () => {
    try {
      existingProjects.value = await getProjects()
    } catch (err) {
      console.error('Failed to load existing projects:', err)
    }
  }

  const runAiGrouping = async (result: ScanResult) => {
    loadingAi.value = true
    try {
      const groupingRes = await groupScanComponents({
        root_path: result.root_path,
        components: result.components as any
      })

      llmUsed.value = groupingRes.llm_used
      aiSummary.value = groupingRes.summary

      const newGroups: ManualProjectGroup[] = []
      for (const g of groupingRes.groups) {
        const isExisting = (g.matched_existing_project_id || 0) > 0
        const matchedProj = isExisting ? existingProjects.value.find(p => p.id === g.matched_existing_project_id) : null

        const comps: ManualComponentItem[] = []
        for (const idx of g.component_indices) {
          const rawComp = result.components[idx]
          if (rawComp) {
            comps.push({
              original_index: idx,
              selected: true,
              name: rawComp.name,
              role: rawComp.role,
              relative_path: rawComp.relative_path,
              absolute_path: rawComp.absolute_path,
              language: rawComp.language,
              framework: rawComp.framework,
              package_manager: rawComp.package_manager,
              dev_command: rawComp.dev_command,
              build_command: rawComp.build_command,
              port: rawComp.port,
              health_url: rawComp.health_url,
              env_file: rawComp.env_file,
              env_summary: rawComp.env_summary
            })
          }
        }

        newGroups.push({
          id: g.id || `group-${newGroups.length + 1}`,
          is_existing: isExisting,
          existing_project_id: isExisting ? g.matched_existing_project_id : (existingProjects.value[0]?.id || undefined),
          name: isExisting && matchedProj ? matchedProj.name : g.name,
          project_type: g.project_type || 'bundle',
          reason: g.reason,
          components: comps
        })
      }

      manualGroups.value = newGroups
    } catch (err: any) {
      console.error('AI grouping error:', err)
      manualGroups.value = [
        {
          id: 'group-fallback',
          is_existing: false,
          name: result.suggested_name,
          project_type: result.project_type,
          reason: 'Pengelompokan default',
          components: result.components.map((c, idx) => ({
            original_index: idx,
            selected: true,
            ...c
          }))
        }
      ]
    } finally {
      loadingAi.value = false
    }
  }

  const handleAddGroup = () => {
    manualGroups.value.push({
      id: `group-custom-${Date.now()}`,
      is_existing: false,
      name: 'Project Baru',
      project_type: 'bundle',
      reason: 'Dibuat manual oleh pengguna',
      components: []
    })
  }

  const handleRemoveGroup = (groupIdx: number) => {
    const removed = manualGroups.value.splice(groupIdx, 1)[0]
    if (removed && removed.components.length > 0 && manualGroups.value.length > 0) {
      manualGroups.value[0]!.components.push(...removed.components)
    }
  }

  const handleMoveComponent = (fromGroupIdx: number, compIdx: number, toGroupIdx: number) => {
    const fromGroup = manualGroups.value[fromGroupIdx]
    const toGroup = manualGroups.value[toGroupIdx]
    if (!fromGroup || !toGroup) return
    const [comp] = fromGroup.components.splice(compIdx, 1)
    if (comp) {
      toGroup.components.push(comp)
    }
  }

  const handleCreateNewGroupWithComponent = (fromGroupIdx: number, compIdx: number) => {
    const fromGroup = manualGroups.value[fromGroupIdx]
    if (!fromGroup) return
    const [comp] = fromGroup.components.splice(compIdx, 1)
    if (comp) {
      manualGroups.value.push({
        id: `group-custom-${Date.now()}`,
        is_existing: false,
        name: comp.name || 'Project Baru',
        project_type: 'bundle',
        reason: 'Dipisahkan ke project baru',
        components: [comp]
      })
    }
  }

  const allSelected = computed(() => {
    const allComps = manualGroups.value.flatMap(g => g.components)
    return allComps.length > 0 && allComps.every(c => c.selected)
  })

  const handleToggleSelectAll = () => {
    const target = !allSelected.value
    manualGroups.value.forEach((g) => {
      g.components.forEach((c) => {
        c.selected = target
      })
    })
  }

  const totalComponents = computed(() => {
    return manualGroups.value.reduce((acc, g) => acc + g.components.length, 0)
  })

  const totalSelected = computed(() => {
    return manualGroups.value.reduce((acc, g) => acc + g.components.filter(c => c.selected).length, 0)
  })

  const validationError = computed(() => {
    if (totalSelected.value === 0) {
      return 'Pilih minimal 1 komponen untuk didaftarkan ke pemantauan.'
    }
    for (const g of manualGroups.value) {
      const selectedInGroup = g.components.filter(c => c.selected)
      if (selectedInGroup.length === 0) continue

      if (!g.is_existing && !g.name.trim()) {
        return 'Nama project baru tidak boleh kosong.'
      }
      if (g.is_existing && !g.existing_project_id) {
        return 'Silakan pilih project tujuan eksisting.'
      }
      for (const c of selectedInGroup) {
        if (!c.name.trim()) {
          return 'Nama setiap komponen terpilih wajib diisi.'
        }
      }
    }
    return ''
  })

  const handleConfirmImport = async (scanResult: ScanResult) => {
    if (validationError.value) {
      toast.add({
        title: 'Validasi Belum Lengkap',
        description: validationError.value,
        color: 'warning',
        icon: 'i-lucide-alert-triangle'
      })
      return
    }

    importing.value = true
    let successCount = 0
    let lastProjectId: number | null = null

    try {
      for (const g of manualGroups.value) {
        const selectedComps = g.components.filter(c => c.selected)
        if (selectedComps.length === 0) continue

        if (!g.is_existing) {
          const created = await createProject({
            name: g.name.trim(),
            description: g.reason || `Discovered on local laptop (${g.project_type}).`,
            root_path: scanResult.root_path || '',
            project_type: g.project_type as any,
            status: 'active',
            git_branch: scanResult.git_branch || '',
            git_remote: scanResult.git_remote || '',
            components: selectedComps.map(c => ({
              name: c.name.trim(),
              role: c.role,
              relative_path: c.relative_path,
              absolute_path: c.absolute_path,
              language: c.language,
              framework: c.framework,
              package_manager: c.package_manager,
              dev_command: c.dev_command,
              build_command: c.build_command,
              port: c.port,
              health_url: c.health_url,
              env_file: c.env_file,
              env_summary: c.env_summary
            })) as any
          })
          successCount++
          lastProjectId = created.id
        } else if (g.existing_project_id) {
          for (const c of selectedComps) {
            await addComponent(g.existing_project_id, {
              name: c.name.trim(),
              role: c.role,
              relative_path: c.relative_path,
              absolute_path: c.absolute_path,
              language: c.language,
              framework: c.framework,
              package_manager: c.package_manager,
              dev_command: c.dev_command,
              build_command: c.build_command,
              port: c.port,
              health_url: c.health_url,
              env_file: c.env_file,
              env_summary: c.env_summary
            })
          }
          successCount++
          lastProjectId = g.existing_project_id
        }
      }

      toast.add({
        title: 'Berhasil Didaftarkan ke Pemantauan',
        description: `${totalSelected.value} komponen berhasil didaftarkan ke dalam ${successCount} project.`,
        color: 'success',
        icon: 'i-lucide-check-circle'
      })

      if (lastProjectId) {
        router.push(`/projects/${lastProjectId}`)
      } else {
        router.push('/cockpit')
      }
    } catch (err: any) {
      toast.add({
        title: 'Pendaftaran Gagal',
        description: err.data?.error || err.message || 'Gagal mendaftarkan ke pemantauan',
        color: 'error',
        icon: 'i-lucide-alert-circle'
      })
    } finally {
      importing.value = false
    }
  }

  return {
    loadingAi,
    llmUsed,
    aiSummary,
    manualGroups,
    existingProjects,
    importing,
    loadExistingProjects,
    runAiGrouping,
    handleAddGroup,
    handleRemoveGroup,
    handleMoveComponent,
    handleCreateNewGroupWithComponent,
    handleToggleSelectAll,
    allSelected,
    totalComponents,
    totalSelected,
    validationError,
    handleConfirmImport
  }
}
