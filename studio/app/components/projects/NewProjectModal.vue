<script setup lang="ts">
import type { ComponentDraft, BrowseDirectoryResponse, Project } from '~/types'

const open = defineModel<boolean>('open', { default: false })

const emit = defineEmits<{
  created: [project: Project]
}>()

const router = useRouter()
const toast = useToast()
const { createProject, browseDirectories } = useApi()

const creatingProject = ref(false)
const projectName = ref('')
const projectComponents = ref<ComponentDraft[]>([])

// Directory browser state
const activeBrowsingCompIndex = ref<number | null>(null)
const browsingPath = ref('~/Documents')
const currentBrowseData = ref<BrowseDirectoryResponse | null>(null)
const loadingFolders = ref(false)

watch(open, (isOpen) => {
  if (isOpen) {
    projectName.value = ''
    projectComponents.value = []
    activeBrowsingCompIndex.value = null
    browsingPath.value = '~/Documents'
  }
})

const addFolderDraft = (role: 'backend' | 'frontend_client' | 'frontend_mobile' | 'frontend_admin' | 'other' = 'backend') => {
  projectComponents.value.push({
    name: '',
    role,
    absolute_path: '',
    port: 0,
    language: 'auto',
    framework: '',
    dev_command: ''
  })
}

const removeComponentDraft = (index: number) => {
  projectComponents.value.splice(index, 1)
  if (activeBrowsingCompIndex.value === index) {
    activeBrowsingCompIndex.value = null
  }
}

const loadDirectories = async (path?: string, search?: string) => {
  loadingFolders.value = true
  try {
    const res = await browseDirectories(path, search)
    currentBrowseData.value = res
    if (res.current_path && !search) {
      browsingPath.value = res.current_path
    }
  } catch (err) {
    console.error('Failed to browse directories:', err)
  } finally {
    loadingFolders.value = false
  }
}

const startBrowsingForComp = async (index: number) => {
  if (activeBrowsingCompIndex.value === index) {
    activeBrowsingCompIndex.value = null
    return
  }
  activeBrowsingCompIndex.value = index
  const comp = projectComponents.value[index]
  const targetPath = comp?.absolute_path?.trim() || browsingPath.value || '~/Documents'
  await loadDirectories(targetPath)
}

const selectFolderForComp = (path: string, folderName?: string) => {
  if (activeBrowsingCompIndex.value !== null) {
    const comp = projectComponents.value[activeBrowsingCompIndex.value]
    if (comp) {
      comp.absolute_path = path
      const derivedName = folderName || path.split('/').filter(Boolean).pop() || ''
      if (!comp.name.trim()) {
        comp.name = derivedName
      }
      if (!projectName.value.trim()) {
        projectName.value = derivedName
      }
    }
  }
  activeBrowsingCompIndex.value = null
}

const handleNavigateFolder = (path: string) => {
  browsingPath.value = path
  loadDirectories(path)
}

const handleCreateProjectSubmit = async () => {
  if (!projectName.value.trim()) {
    toast.add({
      title: 'Validasi Gagal',
      description: 'Nama project wajib diisi',
      icon: 'i-lucide-alert-circle',
      color: 'warning'
    })
    return
  }

  const validComponents = projectComponents.value
    .filter(c => c.name.trim() || c.absolute_path.trim())
    .map(c => ({
      name: c.name.trim() || (c.absolute_path ? c.absolute_path.split('/').filter(Boolean).pop() || c.role : c.role),
      role: c.role,
      absolute_path: c.absolute_path.trim(),
      relative_path: c.absolute_path ? c.absolute_path.split('/').filter(Boolean).pop() || c.role : c.role,
      port: 0,
      language: 'auto',
      framework: '',
      package_manager: '',
      dev_command: ''
    }))

  creatingProject.value = true
  try {
    const created = await createProject({
      name: projectName.value.trim(),
      description: '', // Kosongkan saja: nanti diisi otomatis oleh LLM setelah scan awal
      root_path: '',
      status: 'active',
      project_type: validComponents.some(c => c.role.includes('mobile')) ? 'bundle' : validComponents.filter(c => c.role.includes('front')).length > 1 ? 'multi_fe' : 'bundle',
      components: validComponents as any
    })

    open.value = false
    emit('created', created)
    router.push(`/projects/${created.id}`)
  } catch (err: any) {
    toast.add({
      title: 'Gagal Membuat Proyek',
      description: `Gagal membuat project: ${err.message || 'Terjadi kesalahan'}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    creatingProject.value = false
  }
}

const { getRoleBadgeColor } = useFormatters()

const { getRoleLabel: roleLabel } = useFormatters()
const getRoleLabel = (role: string) => roleLabel(role, { mobile: 'Mobile App', other: 'Folder Lain' })
</script>

<template>
  <UModal
    v-model:open="open"
    title="Tambah Project Baru"
    description="Cukup masukkan nama dan pilih folder aplikasi. Stack framework, port, dan deskripsi akan dikenali otomatis oleh LLM."
    :ui="{
      content: 'sm:max-w-4xl lg:max-w-5xl bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 shadow-2xl divide-y divide-neutral-200 dark:divide-neutral-800',
      overlay: 'bg-neutral-900/50 backdrop-blur-xs'
    }"
  >
    <template #body>
      <div class="space-y-6 max-h-[78vh] overflow-y-auto pr-1.5 py-1">
        <!-- Step 1: Identitas Project -->
        <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-50/70 dark:bg-neutral-900/60">
          <div class="space-y-1.5">
            <label class="block text-xs font-semibold uppercase tracking-wider text-neutral-700 dark:text-neutral-300">
              Nama Project <span class="text-rose-500">*</span>
            </label>
            <UInput
              v-model="projectName"
              placeholder="Contoh: asetZeka, e-commerce, portal-klinik"
              size="md"
              autofocus
              class="w-full"
              @keyup.enter="handleCreateProjectSubmit"
            />
            <div class="flex items-center gap-1.5 pt-1 text-xs text-sky-600 dark:text-sky-400">
              <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 shrink-0" />
              <span>Deskripsi proyek akan dibuat dan diisi otomatis oleh LLM setelah proses scan awal.</span>
            </div>
          </div>
        </div>

        <!-- Step 2: App Folders (Topology: Cukup Pilih Folder) -->
        <div class="space-y-3">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-2.5 border-b border-neutral-200 dark:border-neutral-800">
            <div>
              <h3 class="text-sm font-bold text-neutral-900 dark:text-white flex items-center gap-2">
                <UIcon name="i-lucide-folder-git-2" class="w-4 h-4 text-sky-500 dark:text-sky-400" />
                Folder Aplikasi
              </h3>
              <p class="text-xs text-neutral-500 dark:text-neutral-400">
                Pilih folder aplikasi (Backend, Frontend, Mobile). Framework, port, dan command akan dideteksi oleh LLM scan.
              </p>
            </div>

            <!-- Single Clean Add Button -->
            <div class="flex items-center gap-1.5">
              <UButton
                icon="i-lucide-folder-plus"
                color="primary"
                variant="solid"
                size="xs"
                @click="addFolderDraft('backend')"
              >
                + Tambah Folder
              </UButton>
            </div>
          </div>

          <!-- Empty Components State -->
          <div
            v-if="projectComponents.length === 0"
            class="p-8 text-center rounded-xl border border-dashed border-neutral-300 dark:border-neutral-700 bg-neutral-50/60 dark:bg-neutral-900/40 space-y-3.5"
          >
            <div class="w-12 h-12 mx-auto rounded-full bg-sky-50 dark:bg-sky-950/50 text-sky-600 dark:text-sky-400 flex items-center justify-center border border-sky-200 dark:border-sky-800/40">
              <UIcon name="i-lucide-folder-plus" class="w-6 h-6" />
            </div>
            <div class="space-y-1">
              <p class="text-sm font-semibold text-neutral-800 dark:text-neutral-100">
                Belum ada folder aplikasi yang dipilih
              </p>
              <p class="text-xs text-neutral-500 dark:text-neutral-400 max-w-md mx-auto">
                Cukup pilih folder aplikasi Anda. Pengenalan stack, port, command, dan pembuatan deskripsi dikerjakan otomatis oleh LLM.
              </p>
            </div>
            <div class="pt-1">
              <UButton
                icon="i-lucide-folder-plus"
                color="primary"
                variant="solid"
                size="sm"
                @click="addFolderDraft('backend')"
              >
                + Tambah Folder Aplikasi
              </UButton>
            </div>
          </div>

          <!-- Components List (Hanya Folder Path & Nama / Peran) -->
          <div v-else class="space-y-3.5">
            <div
              v-for="(comp, idx) in projectComponents"
              :key="idx"
              class="rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800/60 p-4 shadow-xs space-y-3 transition-all hover:border-neutral-300 dark:hover:border-neutral-600"
            >
              <!-- Comp Card Header -->
              <div class="flex items-center justify-between gap-3 pb-2 border-b border-neutral-200 dark:border-neutral-700/80">
                <div class="flex items-center gap-2 flex-1 min-w-0">
                  <UBadge
                    size="sm"
                    :color="getRoleBadgeColor(comp.role)"
                    variant="subtle"
                  >
                    {{ getRoleLabel(comp.role) }}
                  </UBadge>
                  <input
                    v-model="comp.name"
                    type="text"
                    placeholder="Nama layanan (opsional, otomatis dari folder)..."
                    class="text-sm font-bold text-neutral-900 dark:text-neutral-100 bg-transparent border-b border-transparent hover:border-neutral-300 dark:hover:border-neutral-600 focus:border-primary-500 focus:outline-none px-1.5 py-0.5 flex-1 min-w-0"
                  >
                </div>

                <div class="flex items-center gap-2 shrink-0">
                  <select
                    v-model="comp.role"
                    class="px-2.5 py-1 rounded-md border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-900 text-xs text-neutral-800 dark:text-neutral-200 focus:outline-none focus:border-primary-500"
                  >
                    <option value="backend">
                      Backend (BE)
                    </option>
                    <option value="frontend_client">
                      Frontend Web (FE)
                    </option>
                    <option value="frontend_mobile">
                      Mobile App (Flutter / RN)
                    </option>
                    <option value="frontend_admin">
                      FE Admin
                    </option>
                    <option value="docs">
                      Docs
                    </option>
                    <option value="cli">
                      CLI
                    </option>
                    <option value="worker">
                      Worker
                    </option>
                    <option value="other">
                      Folder Lain
                    </option>
                  </select>

                  <UButton
                    icon="i-lucide-trash-2"
                    color="error"
                    variant="ghost"
                    size="xs"
                    title="Hapus folder ini"
                    @click="removeComponentDraft(idx)"
                  />
                </div>
              </div>

              <!-- Folder Path & Browser -->
              <div class="space-y-1.5">
                <div class="flex items-center justify-between text-xs">
                  <label class="font-semibold text-neutral-700 dark:text-neutral-200">
                    Pilih Lokasi Folder di Laptop <span class="text-rose-500">*</span>
                  </label>
                  <span v-if="comp.absolute_path" class="text-sky-600 dark:text-sky-400 font-mono text-xs flex items-center gap-1">
                    <UIcon name="i-lucide-check-circle" class="w-3.5 h-3.5" />
                    Folder terpilih
                  </span>
                </div>

                <div class="flex items-center gap-2">
                  <UInput
                    v-model="comp.absolute_path"
                    placeholder="Pilih atau ketik direktori folder, misal: ~/Documents/go/my-api"
                    size="sm"
                    class="font-mono text-xs flex-1"
                  />
                  <UButton
                    :icon="activeBrowsingCompIndex === idx ? 'i-lucide-x' : 'i-lucide-folder-search'"
                    :color="activeBrowsingCompIndex === idx ? 'primary' : 'neutral'"
                    :variant="activeBrowsingCompIndex === idx ? 'solid' : 'outline'"
                    size="sm"
                    @click="startBrowsingForComp(idx)"
                  >
                    {{ activeBrowsingCompIndex === idx ? 'Tutup Browser' : 'Browse Folder' }}
                  </UButton>
                </div>

                <!-- FolderPicker component embedded -->
                <ProjectsFolderPicker
                  v-if="activeBrowsingCompIndex === idx"
                  class="mt-2.5"
                  :browsing-path="browsingPath"
                  :browse-data="currentBrowseData"
                  :loading="loadingFolders"
                  @navigate="handleNavigateFolder"
                  @select="selectFolderForComp"
                />

                <p class="text-xs text-neutral-400 dark:text-neutral-500 italic">
                  Framework (Go, Nuxt, Flutter, dll.), port dev, dan command akan diidentifikasi secara otomatis saat project di-scan.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-between w-full">
        <div class="text-xs text-neutral-500 dark:text-neutral-400">
          <span v-if="projectComponents.length > 0">
            {{ projectComponents.length }} folder aplikasi disiapkan
          </span>
          <span v-else>
            Folder aplikasi juga dapat ditambahkan nanti
          </span>
        </div>

        <div class="flex items-center gap-2">
          <UButton
            color="neutral"
            variant="outline"
            @click="open = false"
          >
            Batal
          </UButton>
          <UButton
            color="primary"
            :loading="creatingProject"
            :disabled="!projectName.trim()"
            @click="handleCreateProjectSubmit"
          >
            Tambah Project & Mulai Scan
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
