<script setup lang="ts">
import type { TodoMergeGroup } from '~/types'

/** "Lebur Todo Serupa" (Aturan 43): pratinjau kelompok todo serupa, lalu lebur menjadi todo gabungan berisi sub todo. */
const props = defineProps<{
  open: boolean
  projectId?: number
}>()
const emit = defineEmits<{
  (e: 'update:open', v: boolean): void
  (e: 'merged'): void
}>()

const { mergeTodos } = useApi()
const toast = useToast()

const groups = ref<TodoMergeGroup[]>([])
const loading = ref(false)
const merging = ref(false)
const loadError = ref<string | null>(null)

const errorText = (err: unknown) => {
  const e = err as { data?: { error?: string }, message?: string }
  return e?.data?.error || e?.message || 'Gagal menghubungi engine'
}

watch(() => props.open, async (open) => {
  if (!open || !props.projectId) return
  loading.value = true
  loadError.value = null
  groups.value = []
  try {
    groups.value = (await mergeTodos({ project_id: props.projectId, dry_run: true })).groups
  } catch (err) {
    loadError.value = errorText(err)
  } finally {
    loading.value = false
  }
})

const submit = async () => {
  if (!props.projectId) return
  merging.value = true
  try {
    const res = await mergeTodos({ project_id: props.projectId })
    toast.add({
      title: `${res.groups.length} todo gabungan dibentuk`,
      description: `${res.merged} todo serupa kini dikerjakan sekaligus sebagai sub todo.`,
      icon: 'i-lucide-layers',
      color: 'success'
    })
    emit('merged')
    emit('update:open', false)
  } catch (err) {
    toast.add({ title: 'Gagal melebur todo', description: errorText(err), icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    merging.value = false
  }
}
</script>

<template>
  <UModal :open="open" :ui="{ content: 'max-w-xl' }" @update:open="emit('update:open', $event)">
    <template #content>
      <div class="p-5 space-y-4 max-h-[85vh] overflow-y-auto">
        <div>
          <h3 class="text-base font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-layers" class="size-5 text-violet-500" />
            Lebur Todo Serupa
          </h3>
          <p class="mt-1 text-xs text-neutral-500">
            Todo pending/saran yang konteksnya dekat (rule sama, berkas sama, atau isi & judul mirip) di repositori dan kategori yang sama dijadikan satu todo gabungan. Satu agent mengerjakan semua sub todonya sekaligus: satu branch, satu commit, satu review QA.
          </p>
        </div>

        <UAlert v-if="loadError" color="warning" variant="subtle" icon="i-lucide-info" :description="loadError" />

        <div v-if="loading" class="flex items-center gap-2 text-xs text-neutral-500">
          <UIcon name="i-lucide-loader-circle" class="size-4 animate-spin" />
          Mencari todo serupa…
        </div>

        <p v-else-if="!loadError && !groups.length" class="text-xs text-neutral-500">
          Tidak ada todo serupa yang layak dilebur. Todo berat (complex), yang pernah gagal QA, dan yang sedang dikerjakan agent selalu dikerjakan sendiri.
        </p>

        <div v-else class="space-y-2">
          <div v-for="(g, i) in groups" :key="i" class="p-2.5 rounded-lg border border-neutral-200 dark:border-neutral-800 text-xs space-y-1.5">
            <div class="flex items-start justify-between gap-2">
              <span class="font-semibold text-neutral-900 dark:text-white">{{ g.title }}</span>
              <UBadge v-if="g.joined" size="xs" color="info" variant="subtle">
                Gabung ke #{{ g.parent_id }}
              </UBadge>
            </div>
            <p class="text-xs text-violet-700 dark:text-violet-300">
              Dilebur karena {{ g.reason }}
            </p>
            <ul class="space-y-0.5 text-neutral-600 dark:text-neutral-300">
              <li v-for="(t, j) in g.titles" :key="j" class="flex gap-1.5">
                <span class="font-mono text-neutral-400">#{{ g.todo_ids[j] }}</span>
                <span>{{ t }}</span>
              </li>
            </ul>
          </div>
        </div>

        <div class="flex justify-end gap-2 pt-2 border-t border-neutral-100 dark:border-neutral-800">
          <UButton color="neutral" variant="ghost" label="Batal" @click="emit('update:open', false)" />
          <UButton
            color="primary"
            icon="i-lucide-layers"
            :label="`Lebur ${groups.length} kelompok`"
            :loading="merging"
            :disabled="loading || !groups.length"
            @click="submit"
          />
        </div>
      </div>
    </template>
  </UModal>
</template>
