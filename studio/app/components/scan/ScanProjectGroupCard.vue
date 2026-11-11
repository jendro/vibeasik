<script setup lang="ts">
import type { ManualProjectGroup, Project } from '~/types'

const props = defineProps<{
  group: ManualProjectGroup
  groupIndex: number
  existingProjects: Project[]
  allGroups: ManualProjectGroup[]
}>()

const emit = defineEmits<{
  removeGroup: [groupIndex: number]
  moveComponent: [fromGroupIndex: number, compIndex: number, toGroupIndex: number]
  createNewGroupWithComponent: [fromGroupIndex: number, compIndex: number]
}>()

const handleMoveChange = (compIndex: number, event: Event) => {
  const target = event.target as HTMLSelectElement
  const val = target.value
  if (val === 'new') {
    emit('createNewGroupWithComponent', props.groupIndex, compIndex)
  } else {
    const toGroupIdx = parseInt(val, 10)
    if (!isNaN(toGroupIdx) && toGroupIdx !== props.groupIndex) {
      emit('moveComponent', props.groupIndex, compIndex, toGroupIdx)
    }
  }
  // reset select to default
  target.value = ''
}
</script>

<template>
  <div class="p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] space-y-4">
    <!-- Group Configuration Header -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-3 pb-3 border-b border-neutral-100 dark:border-white/10">
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-xs font-bold uppercase tracking-wider text-neutral-400">Project:</span>

        <!-- Mode Toggle: Project Baru vs Existing -->
        <div class="inline-flex rounded-lg p-0.5 bg-neutral-100 dark:bg-neutral-800 text-xs">
          <button
            type="button"
            class="px-2.5 py-1 rounded-md font-medium transition-colors cursor-pointer"
            :class="!group.is_existing ? 'bg-white dark:bg-neutral-700 text-sky-600 dark:text-sky-400' : 'text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300'"
            @click="group.is_existing = false"
          >
            Project Baru
          </button>
          <button
            type="button"
            class="px-2.5 py-1 rounded-md font-medium transition-colors cursor-pointer"
            :class="group.is_existing ? 'bg-white dark:bg-neutral-700 text-sky-600 dark:text-sky-400' : 'text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300'"
            @click="group.is_existing = true; if (!group.existing_project_id && existingProjects.length > 0) group.existing_project_id = existingProjects[0]?.id"
          >
            Project yang Sudah Ada
          </button>
        </div>

        <UBadge size="xs" color="neutral" variant="subtle">
          {{ group.components.filter(c => c.selected).length }} / {{ group.components.length }} Komponen Terpilih
        </UBadge>
      </div>

      <!-- Delete Group Button -->
      <UButton
        v-if="allGroups.length > 1"
        icon="i-lucide-trash-2"
        color="error"
        variant="ghost"
        size="xs"
        title="Hapus wadah project ini (komponen akan dikembalikan ke project pertama)"
        @click="emit('removeGroup', groupIndex)"
      >
        Hapus Project Ini
      </UButton>
    </div>

    <!-- Project Identity Form -->
    <div class="grid grid-cols-1 md:grid-cols-12 gap-3 items-center">
      <!-- If New Project: Name + Type -->
      <template v-if="!group.is_existing">
        <div class="md:col-span-8 space-y-1">
          <label class="block text-xs font-semibold text-neutral-500 uppercase tracking-wider">
            Nama Project Baru <span class="text-rose-500">*</span>
          </label>
          <UInput
            v-model="group.name"
            placeholder="Masukkan nama project (wajib diisi)"
            size="sm"
            class="w-full font-medium"
            :color="!group.name.trim() && group.components.some(c => c.selected) ? 'error' : undefined"
          />
        </div>

        <div class="md:col-span-4 space-y-1">
          <label class="block text-xs font-semibold text-neutral-500 uppercase tracking-wider">
            Tipe Arsitektur
          </label>
          <select
            v-model="group.project_type"
            class="w-full px-2.5 py-1.5 rounded-md border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-[#04111f] text-xs text-neutral-800 dark:text-neutral-200 focus:outline-none focus:border-primary-500"
          >
            <option value="bundle">
              Bundle (BE + FE)
            </option>
            <option value="multi_fe">
              Multi-FE (BE + Multi Frontends)
            </option>
            <option value="single_app">
              Single App (Mandiri)
            </option>
            <option value="monorepo">
              Monorepo
            </option>
            <option value="microservice">
              Microservices
            </option>
          </select>
        </div>
      </template>

      <!-- If Existing Project: Dropdown -->
      <template v-else>
        <div class="md:col-span-12 space-y-1">
          <label class="block text-xs font-semibold text-neutral-500 uppercase tracking-wider">
            Pilih Project Tujuan <span class="text-rose-500">*</span>
          </label>
          <select
            v-model="group.existing_project_id"
            class="w-full px-2.5 py-1.5 rounded-md border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-[#04111f] text-xs text-neutral-800 dark:text-neutral-200 focus:outline-none focus:border-primary-500 font-medium"
          >
            <option v-for="ep in existingProjects" :key="ep.id" :value="ep.id">
              {{ ep.name }} (ID: {{ ep.id }} • {{ ep.project_type }} • {{ ep.components?.length || 0 }} komponen)
            </option>
          </select>
        </div>
      </template>
    </div>

    <!-- AI Reasoning pill -->
    <div v-if="group.reason" class="text-xs text-neutral-500 dark:text-neutral-400 italic flex items-center gap-1.5 pt-0.5">
      <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 text-sky-500 shrink-0" />
      <span>Saran AI: {{ group.reason }}</span>
    </div>

    <!-- Components Table / Cards -->
    <div class="space-y-2 pt-1">
      <div class="text-xs font-semibold text-neutral-500 uppercase tracking-wider">
        Komponen yang Dimasukkan ke Project Ini ({{ group.components.length }})
      </div>

      <div v-if="group.components.length === 0" class="p-4 rounded-lg border border-dashed border-neutral-200 dark:border-white/[0.08] text-center text-xs text-neutral-400">
        Belum ada komponen di project ini. Pindahkan komponen dari kelompok lain.
      </div>

      <div
        v-for="(comp, cIdx) in group.components"
        :key="comp.original_index"
        class="p-3.5 rounded-lg border transition-all space-y-2.5 text-xs"
        :class="comp.selected
          ? 'border-neutral-200 dark:border-white/10 bg-neutral-50/60 dark:bg-slate-800/40'
          : 'border-neutral-200/50 dark:border-neutral-800/40 bg-neutral-100/30 dark:bg-slate-900/20 opacity-60'"
      >
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
          <!-- Selection Checkbox + Editable Name -->
          <div class="flex items-center gap-2.5 flex-1 min-w-0">
            <input
              v-model="comp.selected"
              type="checkbox"
              class="w-4 h-4 rounded text-sky-600 focus:ring-sky-500 cursor-pointer shrink-0"
              title="Centang untuk mendaftarkan komponen ini ke pemantauan"
            >
            <div class="flex-1 min-w-0">
              <input
                v-model="comp.name"
                type="text"
                placeholder="Nama Komponen (wajib)"
                class="w-full text-xs font-bold text-neutral-900 dark:text-neutral-100 bg-transparent border-b px-1 py-0.5 focus:outline-none transition-colors"
                :class="!comp.name.trim() && comp.selected
                  ? 'border-rose-500 focus:border-rose-600'
                  : 'border-transparent hover:border-neutral-300 dark:hover:border-neutral-600 focus:border-primary-500'"
              >
            </div>
          </div>

          <!-- Role Select + Move Menu -->
          <div class="flex items-center gap-2 shrink-0">
            <select
              v-model="comp.role"
              class="px-2 py-1 rounded-md border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-[#04111f] text-xs text-neutral-800 dark:text-neutral-200 focus:outline-none focus:border-primary-500"
            >
              <option value="backend">
                Backend (BE)
              </option>
              <option value="frontend_client">
                Frontend Web (FE)
              </option>
              <option value="frontend_mobile">
                Mobile App
              </option>
              <option value="frontend_admin">
                FE Admin
              </option>
              <option value="worker">
                Worker
              </option>
              <option value="cli">
                CLI
              </option>
              <option value="docs">
                Docs
              </option>
              <option value="other">
                Lainnya
              </option>
            </select>

            <!-- Move to other project -->
            <select
              class="px-2 py-1 rounded-md border border-neutral-300 dark:border-neutral-700 bg-neutral-50 dark:bg-neutral-800 text-xs text-neutral-600 dark:text-neutral-300 focus:outline-none focus:border-primary-500 cursor-pointer"
              title="Pindahkan komponen ini ke project lain"
              @change="handleMoveChange(cIdx, $event)"
            >
              <option value="" disabled selected>
                Pindahkan ke...
              </option>
              <option
                v-for="(otherG, oIdx) in allGroups"
                :key="otherG.id"
                :value="oIdx"
                :disabled="oIdx === groupIndex"
              >
                {{ otherG.name || `Project ${oIdx + 1}` }} {{ oIdx === groupIndex ? '(Aktif)' : '' }}
              </option>
              <option value="new">
                + Project Baru
              </option>
            </select>
          </div>
        </div>

        <!-- Tech badges & directory path -->
        <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-neutral-500 font-mono">
          <div class="flex flex-wrap items-center gap-1.5">
            <span class="text-neutral-400">📁 {{ comp.relative_path || comp.absolute_path }}</span>
            <span class="text-neutral-300 dark:text-neutral-700">•</span>
            <TechBadge :tech="comp.language" size="xs" />
            <TechBadge
              v-if="comp.framework && comp.framework.toLowerCase() !== comp.language.toLowerCase()"
              :tech="comp.framework"
              size="xs"
            />
          </div>

          <div class="flex items-center gap-2">
            <span v-if="comp.dev_command" class="px-2 py-0.5 rounded bg-neutral-200 dark:bg-neutral-700 text-xs font-bold">
              {{ comp.dev_command }}
            </span>
            <span v-if="comp.port > 0" class="px-2 py-0.5 rounded bg-sky-100 dark:bg-sky-950 text-sky-700 dark:text-sky-300 text-xs font-bold">
              :{{ comp.port }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
