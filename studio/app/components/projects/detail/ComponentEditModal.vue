<script setup lang="ts">
import type { ProjectComponent } from '~/types'

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  isEditing: boolean
  initialComponent?: Partial<ProjectComponent> | null
  projectRootPath?: string
  saving?: boolean
}>()

const emit = defineEmits<{
  save: [data: Partial<ProjectComponent>, createFolder: boolean]
}>()

const toast = useToast()

const createFolderOnSave = ref(true)

const form = ref<Partial<ProjectComponent>>({
  name: '',
  role: 'backend',
  relative_path: 'BE',
  absolute_path: '',
  language: 'go',
  framework: 'fiber',
  package_manager: 'go',
  dev_command: 'go run main.go',
  build_command: 'go build -o server main.go',
  port: 8080,
  health_url: 'http://localhost:8080/health',
  env_file: '.env',
  env_summary: ''
})

watch(() => props.initialComponent, (c) => {
  if (c) {
    form.value = { ...c }
    createFolderOnSave.value = false
  }
}, { immediate: true })

const applyRolePreset = (role: string) => {
  form.value.role = role
  const root = props.projectRootPath || '~/Documents'

  if (role === 'backend') {
    if (!props.isEditing || !form.value.name) form.value.name = 'Backend API (BE)'
    if (!props.isEditing || !form.value.relative_path) form.value.relative_path = 'BE'
    form.value.language = 'go'
    form.value.framework = 'fiber'
    form.value.package_manager = 'go'
    form.value.dev_command = 'go run main.go'
    form.value.port = 8080
    form.value.health_url = 'http://localhost:8080/health'
  } else if (role === 'frontend_client') {
    if (!props.isEditing || !form.value.name) form.value.name = 'Frontend Web (FE)'
    if (!props.isEditing || !form.value.relative_path) form.value.relative_path = 'FE'
    form.value.language = 'typescript'
    form.value.framework = 'nuxt'
    form.value.package_manager = 'bun'
    form.value.dev_command = 'bun run dev'
    form.value.port = 3000
    form.value.health_url = 'http://localhost:3000'
  } else if (role === 'frontend_admin') {
    if (!props.isEditing || !form.value.name) form.value.name = 'Admin Dashboard'
    if (!props.isEditing || !form.value.relative_path) form.value.relative_path = 'admin'
    form.value.language = 'typescript'
    form.value.framework = 'nuxt'
    form.value.package_manager = 'bun'
    form.value.dev_command = 'bun run dev'
    form.value.port = 3001
    form.value.health_url = 'http://localhost:3001'
  } else if (role === 'frontend_mobile') {
    if (!props.isEditing || !form.value.name) form.value.name = 'Mobile App'
    if (!props.isEditing || !form.value.relative_path) form.value.relative_path = 'mobile'
    form.value.language = 'dart'
    form.value.framework = 'flutter'
    form.value.package_manager = 'pub'
    form.value.dev_command = 'flutter run'
    form.value.port = 0
    form.value.health_url = ''
  } else {
    if (!props.isEditing || !form.value.name) form.value.name = 'Folder Lain'
    if (!props.isEditing || !form.value.relative_path) form.value.relative_path = 'docs'
    form.value.language = 'auto'
    form.value.framework = ''
    form.value.package_manager = ''
    form.value.dev_command = ''
    form.value.port = 0
    form.value.health_url = ''
  }

  if (!props.isEditing && form.value.relative_path) {
    form.value.absolute_path = `${root}/${form.value.relative_path}`
  }
}

const onRelativePathChange = () => {
  const root = props.projectRootPath || '~/Documents'
  if (form.value.relative_path) {
    form.value.absolute_path = `${root}/${form.value.relative_path}`
  }
}

const handleSubmit = () => {
  if (!form.value.name || !form.value.role) {
    toast.add({
      title: 'Validasi Gagal',
      description: 'Nama folder/komponen dan peran (role) wajib diisi',
      icon: 'i-lucide-alert-circle',
      color: 'warning'
    })
    return
  }
  emit('save', form.value, createFolderOnSave.value)
}
</script>

<template>
  <UModal
    v-model:open="open"
    :title="isEditing ? 'Edit Folder / Komponen Project' : 'Daftarkan Folder Komponen Baru'"
    :description="isEditing ? 'Sesuaikan nama, port, dan path modul ini.' : 'Tambahkan modul Backend, Frontend, atau Mobile baru ke dalam topologi project ini.'"
    :ui="{ content: 'sm:max-w-xl' }"
  >
    <template #body>
      <div class="space-y-4 text-xs">
        <!-- Role Presets (Only when adding) -->
        <div v-if="!isEditing" class="space-y-1.5">
          <label class="block font-medium text-neutral-700 dark:text-neutral-300">Preset Cepat Template Modul:</label>
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
            <button
              type="button"
              class="p-2 rounded-lg border text-left cursor-pointer transition-colors"
              :class="form.role === 'backend' ? 'border-primary-500 bg-primary-50/50 dark:bg-primary-950/40 text-primary-900 dark:text-primary-200 font-bold' : 'border-neutral-200 dark:border-white/[0.08] bg-neutral-50 dark:bg-neutral-800/40 text-neutral-700 dark:text-neutral-300'"
              @click="applyRolePreset('backend')"
            >
              <div class="font-bold">
                Backend (BE)
              </div>
              <div class="text-xs text-neutral-400">
                Go Fiber/Gin :8080
              </div>
            </button>
            <button
              type="button"
              class="p-2 rounded-lg border text-left cursor-pointer transition-colors"
              :class="form.role === 'frontend_client' ? 'border-primary-500 bg-primary-50/50 dark:bg-primary-950/40 text-primary-900 dark:text-primary-200 font-bold' : 'border-neutral-200 dark:border-white/[0.08] bg-neutral-50 dark:bg-neutral-800/40 text-neutral-700 dark:text-neutral-300'"
              @click="applyRolePreset('frontend_client')"
            >
              <div class="font-bold">
                Web Client
              </div>
              <div class="text-xs text-neutral-400">
                Nuxt 4 / Bun :3000
              </div>
            </button>
            <button
              type="button"
              class="p-2 rounded-lg border text-left cursor-pointer transition-colors"
              :class="form.role === 'frontend_admin' ? 'border-primary-500 bg-primary-50/50 dark:bg-primary-950/40 text-primary-900 dark:text-primary-200 font-bold' : 'border-neutral-200 dark:border-white/[0.08] bg-neutral-50 dark:bg-neutral-800/40 text-neutral-700 dark:text-neutral-300'"
              @click="applyRolePreset('frontend_admin')"
            >
              <div class="font-bold">
                Admin Dashboard
              </div>
              <div class="text-xs text-neutral-400">
                Nuxt Admin :3001
              </div>
            </button>
            <button
              type="button"
              class="p-2 rounded-lg border text-left cursor-pointer transition-colors"
              :class="form.role === 'frontend_mobile' ? 'border-primary-500 bg-primary-50/50 dark:bg-primary-950/40 text-primary-900 dark:text-primary-200 font-bold' : 'border-neutral-200 dark:border-white/[0.08] bg-neutral-50 dark:bg-neutral-800/40 text-neutral-700 dark:text-neutral-300'"
              @click="applyRolePreset('frontend_mobile')"
            >
              <div class="font-bold">
                Mobile App
              </div>
              <div class="text-xs text-neutral-400">
                Flutter / Mobile
              </div>
            </button>
          </div>
        </div>

        <div>
          <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">
            Nama Modul / Komponen <span class="text-rose-500">*</span>
          </label>
          <UInput v-model="form.name" placeholder="misal: Core API (BE), Frontend Web (FE)" />
        </div>

        <div>
          <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">Peran dalam Project (Role)</label>
          <select
            v-model="form.role"
            class="w-full px-3 py-2 rounded-lg border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-[#04111f] text-xs"
          >
            <option value="backend">
              Backend API / Microservice (BE)
            </option>
            <option value="frontend_client">
              Client Frontend / Public Web (FE)
            </option>
            <option value="frontend_admin">
              Primary / Admin Frontend (FE Admin)
            </option>
            <option value="frontend_mobile">
              Mobile Application (Flutter / RN / Native)
            </option>
            <option value="docs">
              Documentation / Specs (Docs)
            </option>
            <option value="cli">
              CLI Tool
            </option>
            <option value="worker">
              Background Worker / Job Queue
            </option>
            <option value="other">
              Folder Lain / Custom
            </option>
          </select>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">
              Relative Folder Path <span class="text-rose-500">*</span>
            </label>
            <UInput
              v-model="form.relative_path"
              placeholder="BE, FE, mobile, docs"
              class="font-mono text-xs"
              @input="onRelativePathChange"
            />
          </div>
          <div>
            <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">Port Dev (Jika ada)</label>
            <UInput v-model.number="form.port" type="number" placeholder="8080, 3000, 3005" />
          </div>
        </div>

        <div>
          <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">Absolute Path pada Filesystem</label>
          <UInput
            v-model="form.absolute_path"
            placeholder="/home/username/path/to/folder"
            class="font-mono text-xs"
          />
          <p class="text-xs text-neutral-400 mt-1">
            Otomatis mengikuti root path project, atau sesuaikan jika folder berada di lokasi terpisah.
          </p>
        </div>

        <!-- Create Folder on Disk Checkbox -->
        <label v-if="!isEditing" class="flex items-center gap-2 cursor-pointer pt-1">
          <input
            v-model="createFolderOnSave"
            type="checkbox"
            class="rounded text-sky-600 focus:ring-sky-500"
          >
          <span class="text-neutral-700 dark:text-neutral-300 font-medium">
            Buat folder fisik di filesystem laptop jika belum ada
          </span>
        </label>

        <!-- Technical Details -->
        <div class="pt-2 border-t border-neutral-100 dark:border-white/[0.08] space-y-3">
          <div class="text-xs font-semibold text-neutral-400 uppercase tracking-wider">
            Detail Teknis (Opsional)
          </div>

          <div class="grid grid-cols-3 gap-3">
            <div>
              <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">Bahasa</label>
              <UInput v-model="form.language" placeholder="go, typescript, dart, python" />
            </div>
            <div>
              <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">Framework</label>
              <UInput v-model="form.framework" placeholder="fiber, nuxt, flutter" />
            </div>
            <div>
              <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">Pkg Manager</label>
              <UInput v-model="form.package_manager" placeholder="bun, go, pub, npm" />
            </div>
          </div>

          <div>
            <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">Perintah Dev (Command)</label>
            <UInput v-model="form.dev_command" placeholder="misal: bun run dev, go run main.go" class="font-mono text-xs" />
          </div>

          <div>
            <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">Health Check URL</label>
            <UInput v-model="form.health_url" placeholder="http://localhost:8080/health" class="font-mono text-xs" />
          </div>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-end gap-2 w-full">
        <UButton color="neutral" variant="outline" @click="open = false">
          Batal
        </UButton>
        <UButton color="primary" :loading="saving" @click="handleSubmit">
          Simpan Folder / Komponen
        </UButton>
      </div>
    </template>
  </UModal>
</template>
