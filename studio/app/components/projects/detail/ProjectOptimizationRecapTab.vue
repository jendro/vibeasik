<script setup lang="ts">
import type { Project } from '~/types'
import { buildOptimizationRecap, type RecapTodo } from '~/utils/optimizationRecap'

const props = defineProps<{ project: Project }>()

const { getTodos } = useApi()
const todos = ref<RecapTodo[]>([])
const loading = ref(true)
const failed = ref('')

// Satu panggilan ke todo project (selesai + masih terbuka); engine Community melayaninya tanpa Pro.
const load = async () => {
  loading.value = true
  failed.value = ''
  try {
    todos.value = (await getTodos({ project_id: props.project.id, limit: 1000 })) as RecapTodo[]
  } catch (err: any) {
    failed.value = err?.message || 'Gagal memuat todo project'
  } finally {
    loading.value = false
  }
}
onMounted(load)
watch(() => props.project.id, load)

const recap = computed(() => buildOptimizationRecap(todos.value))
</script>

<template>
  <div class="space-y-5">
    <div v-if="loading" class="h-48 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] animate-pulse" />

    <div v-else-if="failed" class="rounded-xl border border-rose-300 dark:border-rose-800 bg-rose-50 dark:bg-rose-950/30 p-4 text-sm text-rose-800 dark:text-rose-200 flex items-center justify-between gap-3">
      <span>{{ failed }}</span>
      <UButton size="xs" color="neutral" variant="outline" @click="load">
        Coba lagi
      </UButton>
    </div>

    <div v-else-if="recap.total === 0" class="rounded-xl border border-dashed border-neutral-300 dark:border-neutral-700 p-8 text-center text-sm text-neutral-500">
      Belum ada todo di project ini, jadi belum ada optimasi yang bisa direkap.
    </div>

    <template v-else>
      <section class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] p-4 sm:p-5">
        <div class="flex flex-wrap items-end justify-between gap-4">
          <div>
            <div class="text-xs font-semibold text-neutral-400">
              Optimasi selesai
            </div>
            <div class="text-4xl font-bold font-mono text-neutral-900 dark:text-white leading-none mt-1">
              {{ recap.done }}<span class="text-lg font-normal text-neutral-400"> / {{ recap.total }}</span>
            </div>
          </div>
          <div class="text-sm text-neutral-600 dark:text-neutral-300">
            {{ recap.percent }}% selesai · {{ recap.open }} masih terbuka · {{ recap.categories.length }} kategori
          </div>
        </div>
        <div class="mt-4 h-2 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden" role="progressbar" :aria-valuenow="recap.percent" aria-valuemin="0" aria-valuemax="100">
          <div class="h-full bg-sky-500" :style="{ width: `${recap.percent}%` }" />
        </div>
      </section>

      <section class="grid grid-cols-1 xl:grid-cols-2 gap-4">
        <article
          v-for="c in recap.categories"
          :key="c.category"
          class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] p-4 space-y-3"
        >
          <header class="flex items-center justify-between gap-3">
            <h3 class="text-sm font-bold text-neutral-900 dark:text-white">
              {{ c.label }}
            </h3>
            <UBadge size="xs" color="neutral" variant="subtle" class="font-mono">
              {{ c.done }} selesai · {{ c.open }} terbuka
            </UBadge>
          </header>
          <div class="h-1.5 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
            <div class="h-full bg-sky-500" :style="{ width: `${c.percent}%` }" />
          </div>
          <ul v-if="c.latest.length" class="space-y-1.5">
            <li v-for="t in c.latest" :key="t.id" class="text-xs text-neutral-700 dark:text-neutral-300 flex gap-2">
              <UIcon name="i-lucide-check" class="w-3.5 h-3.5 shrink-0 mt-0.5 text-sky-500" />
              <span class="min-w-0 truncate" :title="t.title">{{ t.title }}</span>
            </li>
          </ul>
          <p v-else class="text-xs text-neutral-400">
            Belum ada yang selesai di kategori ini.
          </p>
        </article>
      </section>
    </template>
  </div>
</template>
