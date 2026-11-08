<script setup lang="ts">
import type { Project, TodoLevel, TodoPriority } from '~/types'

const props = defineProps<{
  open: boolean
  projects: Project[]
  activeProjectId?: number
}>()

const emit = defineEmits<{
  (e: 'update:open', val: boolean): void
  (e: 'created'): void
}>()

const { createTodo } = useApi()

const form = ref({
  level: 'project' as TodoLevel,
  category: 'feature' as string,
  target_apps: '',
  project_id: props.activeProjectId as number | undefined,
  component_id: undefined as number | undefined,
  title: '',
  description: '',
  priority: 'medium' as TodoPriority,
  creator_name: 'Developer'
})

// Update project_id when activeProjectId or modal open changes
watch(() => props.open, (isOpen) => {
  if (isOpen && props.activeProjectId) {
    form.value.project_id = props.activeProjectId
  }
})

const submitting = ref(false)
const activeDescTab = ref<'write' | 'preview'>('write')

const selectedProject = computed(() => {
  return props.projects.find(p => p.id === form.value.project_id)
})

const availableComponents = computed(() => {
  return selectedProject.value?.components || []
})

// Auto-fill target_apps when project or component changes
watch(() => form.value.component_id, (compId) => {
  if (compId) {
    const comp = availableComponents.value.find(c => c.id === compId)
    if (comp) {
      form.value.target_apps = `${comp.name} [1 App]`
    }
  } else if (selectedProject.value) {
    const count = selectedProject.value.components?.length || 0
    form.value.target_apps = count > 0
      ? `Semua Komponen (${count} Apps: ${selectedProject.value.components?.map(c => c.name).join(', ')})`
      : `${selectedProject.value.name} [Semua App]`
  }
})

watch(() => form.value.project_id, () => {
  if (form.value.level === 'project' && selectedProject.value) {
    const count = selectedProject.value.components?.length || 0
    form.value.target_apps = count > 0
      ? `Semua Komponen (${count} Apps: ${selectedProject.value.components?.map(c => c.name).join(', ')})`
      : `${selectedProject.value.name} [Semua App]`
  }
})

const handleSubmit = async () => {
  if (!form.value.title.trim()) return

  submitting.value = true
  try {
    const payload = {
      level: form.value.level,
      category: form.value.category,
      target_apps: form.value.target_apps.trim() || undefined,
      title: form.value.title.trim(),
      description: form.value.description.trim(),
      priority: form.value.priority,
      creator_type: 'human' as const,
      creator_name: form.value.creator_name || 'Developer',
      is_ai_suggestion: false,
      source: 'manual',
      project_id: form.value.level !== 'global' ? form.value.project_id : undefined,
      component_id: form.value.level === 'component' ? form.value.component_id : undefined
    }

    await createTodo(payload)
    emit('created')
    emit('update:open', false)

    // Reset form
    form.value.title = ''
    form.value.description = ''
    form.value.target_apps = ''
  } catch (err) {
    console.error('Failed to create todo:', err)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs"
  >
    <div class="w-full max-w-lg rounded-2xl bg-white dark:bg-slate-900/40 border border-neutral-200 dark:border-white/10 shadow-xl overflow-hidden duration-150">
      <div class="px-6 py-4 border-b border-neutral-100 dark:border-white/10 flex items-center justify-between">
        <h3 class="text-base font-bold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-plus-circle" class="w-5 h-5 text-sky-500" />
          Tambah Agenda Perencanaan Baru
        </h3>
        <button
          type="button"
          class="p-1 rounded-lg text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200"
          @click="emit('update:open', false)"
        >
          <UIcon name="i-lucide-x" class="w-5 h-5" />
        </button>
      </div>

      <form class="p-6 space-y-4 text-xs" @submit.prevent="handleSubmit">
        <!-- Level Picker -->
        <div>
          <label class="block font-semibold text-neutral-700 dark:text-neutral-300 mb-1.5">
            Tingkat Perencanaan (Hierarki)
          </label>
          <div class="grid grid-cols-3 gap-2">
            <button
              type="button"
              class="p-2.5 rounded-lg border text-left transition-colors"
              :class="form.level === 'global' ? 'border-violet-500 bg-violet-50 dark:bg-violet-950/40 text-violet-900 dark:text-violet-200 font-bold' : 'border-neutral-200 dark:border-white/10 text-neutral-600 dark:text-neutral-400'"
              @click="form.level = 'global'"
            >
              <div class="flex items-center gap-1.5">
                <UIcon name="i-lucide-globe" class="w-3.5 h-3.5" />
                <span>Global</span>
              </div>
              <div class="text-xs text-neutral-500 mt-0.5">
                Platform & Standar
              </div>
            </button>

            <button
              type="button"
              class="p-2.5 rounded-lg border text-left transition-colors"
              :class="form.level === 'project' ? 'border-blue-500 bg-blue-50 dark:bg-blue-950/40 text-blue-900 dark:text-blue-200 font-bold' : 'border-neutral-200 dark:border-white/10 text-neutral-600 dark:text-neutral-400'"
              @click="form.level = 'project'"
            >
              <div class="flex items-center gap-1.5">
                <UIcon name="i-lucide-folder-kanban" class="w-3.5 h-3.5" />
                <span>Proyek</span>
              </div>
              <div class="text-xs text-neutral-500 mt-0.5">
                Bundle & Milestone
              </div>
            </button>

            <button
              type="button"
              class="p-2.5 rounded-lg border text-left transition-colors"
              :class="form.level === 'component' ? 'border-sky-500 bg-sky-50 dark:bg-sky-950/40 text-sky-900 dark:text-sky-200 font-bold' : 'border-neutral-200 dark:border-white/10 text-neutral-600 dark:text-neutral-400'"
              @click="form.level = 'component'"
            >
              <div class="flex items-center gap-1.5">
                <UIcon name="i-lucide-layers" class="w-3.5 h-3.5" />
                <span>Komponen</span>
              </div>
              <div class="text-xs text-neutral-500 mt-0.5">
                Modul & Layanan
              </div>
            </button>
          </div>
        </div>

        <!-- Project & Component selectors (if not global) -->
        <div v-if="form.level !== 'global'" class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <label class="block font-semibold text-neutral-700 dark:text-neutral-300 mb-1">
              Pilih Proyek
            </label>
            <select
              v-model="form.project_id"
              class="w-full px-3 py-2 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800 text-neutral-900 dark:text-white"
              required
            >
              <option :value="undefined" disabled>
                -- Pilih Proyek --
              </option>
              <option v-for="p in projects" :key="p.id" :value="p.id">
                {{ p.name }}
              </option>
            </select>
          </div>

          <div v-if="form.level === 'component'">
            <label class="block font-semibold text-neutral-700 dark:text-neutral-300 mb-1">
              Pilih Komponen / Modul
            </label>
            <select
              v-model="form.component_id"
              class="w-full px-3 py-2 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50 dark:bg-neutral-800 text-neutral-900 dark:text-white"
              :disabled="!availableComponents.length"
            >
              <option :value="undefined">
                -- Opsional / Umum --
              </option>
              <option v-for="c in availableComponents" :key="c.id" :value="c.id">
                {{ c.name }} ({{ c.role }})
              </option>
            </select>
          </div>
        </div>

        <!-- Title -->
        <div>
          <label class="block font-semibold text-neutral-700 dark:text-neutral-300 mb-1">
            Judul Agenda / Tugas
          </label>
          <input
            v-model="form.title"
            type="text"
            placeholder="Contoh: Implementasi standarisasi endpoint /health dan port assignment"
            class="w-full px-3 py-2 rounded-lg border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-sky-500"
            required
          >
        </div>

        <!-- Description with Markdown Support and Live Preview -->
        <div>
          <div class="flex items-center justify-between mb-1.5">
            <label class="block font-semibold text-neutral-700 dark:text-neutral-300">
              Deskripsi & Rincian Instruksi
            </label>
            <div class="flex items-center gap-1 bg-neutral-100 dark:bg-neutral-800 p-0.5 rounded-lg text-xs">
              <button
                type="button"
                class="px-2 py-0.5 rounded-md font-medium transition-colors"
                :class="activeDescTab === 'write' ? 'bg-white dark:bg-neutral-700 text-neutral-900 dark:text-white shadow-2xs' : 'text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-200'"
                @click="activeDescTab = 'write'"
              >
                Tulis
              </button>
              <button
                type="button"
                class="px-2 py-0.5 rounded-md font-medium transition-colors flex items-center gap-1"
                :class="activeDescTab === 'preview' ? 'bg-white dark:bg-neutral-700 text-neutral-900 dark:text-white shadow-2xs' : 'text-neutral-500 hover:text-neutral-800 dark:hover:text-neutral-200'"
                @click="activeDescTab = 'preview'"
              >
                <UIcon name="i-lucide-eye" class="w-3 h-3" />
                <span>Pratinjau MD</span>
              </button>
            </div>
          </div>

          <div v-show="activeDescTab === 'write'">
            <textarea
              v-model="form.description"
              rows="4"
              placeholder="Jelaskan kebutuhan teknis, target implementasi, atau kriteria penyelesaian (mendukung Markdown)..."
              class="w-full px-3 py-2 rounded-lg border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-sky-500 font-mono text-xs"
            />
            <div class="mt-1 flex items-center justify-between text-xs text-neutral-400">
              <span>Mendukung sintaks Markdown: <strong>**tebal**</strong>, <code>`kode`</code>, <code>- daftar</code></span>
              <span class="font-mono">{{ form.description.length }} karakter</span>
            </div>
          </div>

          <div
            v-show="activeDescTab === 'preview'"
            class="min-h-[100px] max-h-[180px] overflow-y-auto px-3 py-2.5 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50/70 dark:bg-neutral-800/40 text-xs"
          >
            <MarkdownViewer
              :content="form.description"
              compact
              empty-text="Belum ada teks keterangan. Tulis sesuatu di tab 'Tulis' untuk melihat hasil tampilan Markdown."
            />
          </div>
        </div>

        <!-- Category & Target Application -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <label class="block font-semibold text-neutral-700 dark:text-neutral-300 mb-1">
              Kategori Agenda
            </label>
            <select
              v-model="form.category"
              class="w-full px-3 py-2 rounded-lg border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white"
            >
              <option value="feature">
                Fitur (Feature)
              </option>
              <option value="security">
                Keamanan (Security)
              </option>
              <option value="ui_ux">
                UI / UX & Tampilan
              </option>
              <option value="architecture">
                Arsitektur & Standar
              </option>
              <option value="performance">
                Performa & Port
              </option>
              <option value="backend">
                Backend & API
              </option>
              <option value="devops">
                DevOps & CI/CD
              </option>
              <option value="general">
                Umum (General)
              </option>
            </select>
          </div>

          <div>
            <label class="block font-semibold text-neutral-700 dark:text-neutral-300 mb-1">
              Untuk Aplikasi / Komponen
            </label>
            <input
              v-model="form.target_apps"
              type="text"
              placeholder="Contoh: Backend API [1 App], atau Semua Komponen"
              class="w-full px-3 py-2 rounded-lg border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-sky-500"
            >
          </div>
        </div>

        <!-- Priority & Creator Name -->
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block font-semibold text-neutral-700 dark:text-neutral-300 mb-1">
              Prioritas
            </label>
            <select
              v-model="form.priority"
              class="w-full px-3 py-2 rounded-lg border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white"
            >
              <option value="low">
                Rendah (Low)
              </option>
              <option value="medium">
                Sedang (Medium)
              </option>
              <option value="high">
                Tinggi (High)
              </option>
              <option value="critical">
                Kritis (Critical)
              </option>
            </select>
          </div>

          <div>
            <label class="block font-semibold text-neutral-700 dark:text-neutral-300 mb-1">
              Nama Pembuat (Human)
            </label>
            <input
              v-model="form.creator_name"
              type="text"
              class="w-full px-3 py-2 rounded-lg border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white"
            >
          </div>
        </div>

        <div class="pt-3 border-t border-neutral-100 dark:border-white/10 flex justify-end gap-2">
          <UButton
            type="button"
            color="neutral"
            variant="ghost"
            @click="emit('update:open', false)"
          >
            Batal
          </UButton>
          <UButton
            type="submit"
            color="primary"
            :loading="submitting"
            icon="i-lucide-check"
          >
            Simpan Agenda
          </UButton>
        </div>
      </form>
    </div>
  </div>
</template>
