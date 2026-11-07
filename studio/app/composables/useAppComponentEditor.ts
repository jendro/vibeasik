import type { ComputedRef, Ref } from 'vue'
import type { Project, ProjectComponent } from '~/types'

// Dialog ubah/tambah aplikasi, hapus aplikasi, dan buat folder di disk.
export function useAppComponentEditor(opts: {
  comp: ComputedRef<ProjectComponent | null>
  project: Ref<Project | null>
  projectId: ComputedRef<number>
  checkActivePort: () => Promise<void>
}) {
  const { comp, project, projectId, checkActivePort } = opts
  const toast = useToast()
  const router = useRouter()
  const { updateComponent, deleteComponent, createFolderOnDisk, addComponent } = useApi()

  // Component Edit Modal state
  const componentModalOpen = ref(false)
  const isEditingComp = ref(false)
  const savingComp = ref(false)
  const initialEditComponent = ref<Partial<ProjectComponent> | null>(null)

  const openEditComponentModal = () => {
    if (!comp.value) return
    isEditingComp.value = true
    initialEditComponent.value = { ...comp.value }
    componentModalOpen.value = true
  }

  const openAddComponentModal = () => {
    isEditingComp.value = false
    const root = project.value?.root_path || '~/Documents'
    initialEditComponent.value = {
      name: 'Aplikasi Baru',
      role: 'backend',
      relative_path: 'backend',
      absolute_path: `${root}/backend`,
      language: 'go',
      port: 8080
    }
    componentModalOpen.value = true
  }

  const handleSaveComponent = async (data: Partial<ProjectComponent>, createFolder: boolean) => {
    savingComp.value = true
    try {
      if (isEditingComp.value && data.id) {
        const updated = await updateComponent(data.id, data)
        if (project.value?.components) {
          const idx = project.value.components.findIndex(item => item.id === updated.id)
          if (idx !== -1) {
            project.value.components[idx] = updated
          }
        }
      } else {
        const created = await addComponent(projectId.value, {
          ...data,
          create_folder: createFolder
        })
        if (project.value?.components) {
          project.value.components.push(created)
        }
      }
      componentModalOpen.value = false
      await checkActivePort()
    } catch (err: any) {
      toast.add({
        title: 'Gagal Menyimpan Komponen',
        description: `Gagal menyimpan komponen: ${err.message}`,
        icon: 'i-lucide-alert-circle',
        color: 'error'
      })
    } finally {
      savingComp.value = false
    }
  }

  const deleteConfirmOpen = ref(false)
  const deletingComp = ref(false)

  const handleDeleteComponent = async () => {
    if (!comp.value) return
    deletingComp.value = true
    try {
      await deleteComponent(comp.value.id)
      deleteConfirmOpen.value = false
      router.push(`/projects/${projectId.value}`)
    } catch (err: any) {
      toast.add({
        title: 'Gagal Menghapus Aplikasi',
        description: err.message || 'Terjadi kesalahan',
        icon: 'i-lucide-alert-circle',
        color: 'error'
      })
    } finally {
      deletingComp.value = false
    }
  }

  const handleCreateDirOnDisk = async () => {
    if (!comp.value) return
    try {
      const res = await createFolderOnDisk(comp.value.id)
      toast.add({
        title: 'Folder Berhasil Dibuat',
        description: `Folder berhasil dibuat di filesystem:\n${res.path}`,
        icon: 'i-lucide-check-circle',
        color: 'success'
      })
    } catch (err: any) {
      toast.add({
        title: 'Gagal Membuat Folder',
        description: `Gagal membuat folder: ${err.message}`,
        icon: 'i-lucide-alert-circle',
        color: 'error'
      })
    }
  }

  return {
    componentModalOpen, isEditingComp, savingComp, initialEditComponent, deleteConfirmOpen, deletingComp,
    openEditComponentModal, openAddComponentModal, handleSaveComponent, handleDeleteComponent, handleCreateDirOnDisk
  }
}
