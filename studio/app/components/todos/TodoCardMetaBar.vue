<script setup lang="ts">
import type { Todo } from '~/types'

const props = defineProps<{ todo: Todo }>()

const levelMeta = computed(() => todoLevelMeta(props.todo))
const priorityMeta = computed(() => todoPriorityMeta(props.todo))
const statusMeta = computed(() => todoStatusMeta(props.todo))
const categoriesMeta = computed(() => todoCategoriesMeta(props.todo))
const targetAppsDisplay = computed(() => todoTargetAppsDisplay(props.todo))
const recommendedModelMeta = computed(() => todoRecommendedModelMeta(props.todo))
const originMeta = computed(() => todoOriginMeta(props.todo))
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-2">
    <div class="flex flex-wrap items-center gap-1.5 text-xs">
      <!-- Level Badge -->
      <span
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-medium text-xs border"
        :class="levelMeta.badgeClass"
      >
        <UIcon :name="levelMeta.icon" class="w-3.5 h-3.5" />
        {{ levelMeta.label }}
      </span>

      <!-- Context Pill (Project / Component) -->
      <span
        v-if="todo.project_name"
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-mono bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 border border-neutral-200 dark:border-neutral-700"
      >
        {{ todo.project_name }}
        <template v-if="todo.component_name">
          <span class="text-neutral-400">/</span>
          <span class="font-semibold">{{ todo.component_name }}</span>
        </template>
      </span>

      <!-- Category Badge -->
      <span
        v-for="cat in categoriesMeta"
        :key="cat.key"
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-medium text-xs border"
        :class="cat.class"
      >
        <UIcon :name="cat.icon" class="w-3.5 h-3.5" />
        {{ cat.label }}
      </span>

      <TodosTodoTagChips :todo="todo" />
      <TodosComplexityBadge :todo="todo" />
      <TodosImpactBadge :todo="todo" />

      <!-- Recommended Model Badge -->
      <span
        v-if="recommendedModelMeta"
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-medium text-xs border"
        :class="recommendedModelMeta.badgeClass"
        :title="recommendedModelMeta.title"
      >
        <UIcon :name="recommendedModelMeta.icon" class="w-3.5 h-3.5" />
        <span>{{ recommendedModelMeta.label }}</span>
      </span>
      <span
        v-if="todo.split_from_id"
        class="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-md border border-indigo-200 dark:border-indigo-800 text-indigo-700 dark:text-indigo-300"
        title="Sub todo pecahan per aplikasi: hanya aplikasi ini, dikerjakan pegawai dengan keahliannya; induk selesai saat semua bagian selesai"
      >
        <UIcon name="i-lucide-split" class="w-3 h-3" />
        Bagian dari #{{ todo.split_from_id }}
      </span>
      <NuxtLink
        v-if="todo.employee_id && todo.employee_name"
        :to="`/staff?id=${todo.employee_id}`"
        class="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-md border border-sky-200 dark:border-sky-800 text-sky-700 dark:text-sky-300 hover:bg-sky-50 dark:hover:bg-sky-950/40"
        :title="todo.employee_reason || 'Pegawai vibeasik yang ditugaskan: agent coding mengerjakan todo ini dengan persona & pengetahuannya'"
      >
        <UIcon name="i-lucide-id-card" class="w-3 h-3" />
        {{ todo.employee_name }}
      </NuxtLink>
      <span
        v-if="todo.rework_count"
        class="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-md border"
        :class="(todo.employee_reason?.includes('di-hold') || todo.rework_note?.includes('di-hold'))
          ? 'bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-300 border-rose-300 dark:border-rose-800 font-semibold'
          : 'border-orange-200 dark:border-orange-800 text-orange-700 dark:text-orange-300'"
        :title="todo.employee_reason?.includes('di-hold') ? todo.employee_reason : (todo.rework_note || 'Dikerjakan ulang karena nilai QA rendah')"
      >
        <UIcon :name="(todo.employee_reason?.includes('di-hold') || todo.rework_note?.includes('di-hold')) ? 'i-lucide-alert-octagon' : 'i-lucide-rotate-ccw'" class="w-3 h-3" />
        {{ (todo.employee_reason?.includes('di-hold') || todo.rework_note?.includes('di-hold')) ? `Ulang ×${todo.rework_count} · Di-hold` : `Ulang ×${todo.rework_count}` }}
      </span>

      <!-- Target Application Badge -->
      <span
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium bg-neutral-100 dark:bg-neutral-800 text-neutral-800 dark:text-neutral-200 border border-neutral-200 dark:border-neutral-700"
        :title="'Aplikasi target pengerjaan: ' + targetAppsDisplay"
      >
        <UIcon name="i-lucide-layout-grid" class="w-3.5 h-3.5 text-neutral-500" />
        <span>{{ targetAppsDisplay }}</span>
      </span>

      <!-- Origin / Source Tag -->
      <span
        class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-xs border"
        :class="originMeta.badgeClass"
        :title="originMeta.title"
      >
        <UIcon :name="originMeta.icon" class="w-3.5 h-3.5" />
        <span>{{ originMeta.label }}</span>
        <span v-if="originMeta.sublabel && originMeta.sublabel !== originMeta.label" class="opacity-75 text-xs">
          ({{ originMeta.sublabel }})
        </span>
      </span>
    </div>

    <div class="flex items-center gap-1.5 text-xs">
      <!-- Priority Badge -->
      <span
        class="px-2 py-0.5 rounded-md text-xs font-medium border"
        :class="priorityMeta.class"
      >
        Prioritas: {{ priorityMeta.label }}
      </span>

      <!-- Status Badge -->
      <span
        class="px-2 py-0.5 rounded-md text-xs font-medium border"
        :class="statusMeta.class"
      >
        {{ statusMeta.label }}
      </span>
    </div>
  </div>
</template>
