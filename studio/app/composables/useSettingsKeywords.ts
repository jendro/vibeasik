import type { CensoredKeyword } from '~/types'

type Notify = (msg: string, type?: 'success' | 'info' | 'error') => void

// Dialog tambah/ubah/hapus keyword sensor (vault kunci tersensor).
export function useSettingsKeywords(notify: Notify, reload: () => Promise<void>) {
  const { createCensoredKeyword, updateCensoredKeyword, deleteCensoredKeyword } = useApi()

  // Modal Add/Edit Keyword
  const isKeywordModalOpen = ref(false)
  const isEditingKeyword = ref(false)
  const keywordModalSaving = ref(false)
  const editingKeyword = ref<Partial<CensoredKeyword> | null>(null)

  // Modal Delete Keyword
  const isDeleteKeywordModalOpen = ref(false)
  const keywordToDelete = ref<CensoredKeyword | null>(null)
  const deletingKeyword = ref(false)

  // Keyword Modal Handlers
  const openAddKeywordModal = (preset?: any) => {
    isEditingKeyword.value = false
    editingKeyword.value = preset
      ? {
          id: 0,
          keyword: preset.keyword,
          category: preset.category,
          description: preset.description,
          is_regex: preset.is_regex,
          is_active: true
        }
      : {
          id: 0,
          keyword: '',
          category: 'API_KEY',
          description: '',
          is_regex: false,
          is_active: true
        }
    isKeywordModalOpen.value = true
  }

  const openEditKeywordModal = (kw: CensoredKeyword) => {
    isEditingKeyword.value = true
    editingKeyword.value = { ...kw }
    isKeywordModalOpen.value = true
  }

  const handleSaveKeyword = async (formData: any) => {
    keywordModalSaving.value = true
    try {
      if (isEditingKeyword.value && formData.id) {
        await updateCensoredKeyword(formData.id, {
          keyword: formData.keyword,
          category: formData.category,
          description: formData.description,
          is_regex: formData.is_regex,
          is_active: formData.is_active
        })
        notify(`Keyword '${formData.keyword}' berhasil diperbarui.`)
      } else {
        await createCensoredKeyword({
          keyword: formData.keyword,
          category: formData.category,
          description: formData.description,
          is_regex: formData.is_regex,
          is_active: formData.is_active
        })
        notify(`Keyword '${formData.keyword}' berhasil ditambahkan ke vault.`)
      }
      isKeywordModalOpen.value = false
      await reload()
    } catch (err: any) {
      notify(`Gagal menyimpan keyword sensor: ${err.message || err}`, 'error')
    } finally {
      keywordModalSaving.value = false
    }
  }

  const handleToggleKeywordStatus = async (kw: CensoredKeyword) => {
    try {
      await updateCensoredKeyword(kw.id, {
        ...kw,
        is_active: !kw.is_active
      })
      kw.is_active = !kw.is_active
      notify(`Status keyword '${kw.keyword}' diubah menjadi: ${kw.is_active ? 'Aktif' : 'Nonaktif'}`)
    } catch (err: any) {
      notify(`Gagal mengubah status: ${err.message || err}`, 'error')
    }
  }

  const confirmDeleteKeyword = (kw: CensoredKeyword) => {
    keywordToDelete.value = kw
    isDeleteKeywordModalOpen.value = true
  }

  const handleDeleteKeyword = async () => {
    if (!keywordToDelete.value) return
    deletingKeyword.value = true
    try {
      await deleteCensoredKeyword(keywordToDelete.value.id)
      notify(`Keyword '${keywordToDelete.value.keyword}' telah dihapus dari vault.`)
      isDeleteKeywordModalOpen.value = false
      keywordToDelete.value = null
      await reload()
    } catch (err: any) {
      notify(`Gagal menghapus keyword: ${err.message || err}`, 'error')
    } finally {
      deletingKeyword.value = false
    }
  }

  return {
    isKeywordModalOpen, isEditingKeyword, keywordModalSaving, editingKeyword,
    isDeleteKeywordModalOpen, keywordToDelete, deletingKeyword,
    openAddKeywordModal, openEditKeywordModal, handleSaveKeyword, handleToggleKeywordStatus, confirmDeleteKeyword, handleDeleteKeyword
  }
}
