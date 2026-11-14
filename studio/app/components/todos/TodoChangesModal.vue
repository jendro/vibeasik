<script setup lang="ts">
import type { TodoChangeFile, TodoChanges } from '~/types'

/**
 * Perubahan hasil todo: commit yang dikirim (SHA utuh), berkas, fungsi yang berubah, dan diff per bagian — dibaca
 * ulang dari git oleh engine, sehingga pengguna memeriksa kode yang benar-benar masuk, bukan klaim laporan agent.
 */
const props = defineProps<{
  open: boolean
  todoId: number
  todoTitle?: string
}>()
const emit = defineEmits<{ (e: 'update:open', v: boolean): void }>()

const { getTodoChanges } = useApi()
const { copyToClipboard } = useCopyToClipboard()
const { formatDateTime } = useFormatters()

const data = ref<TodoChanges | null>(null)
const loading = ref(false)
const loadError = ref<string | null>(null)
const expanded = ref<Record<string, boolean>>({})

const commit = computed(() => data.value?.commit ?? null)

watch(() => props.open, async (open) => {
  if (!open) return
  loading.value = true
  loadError.value = null
  data.value = null
  try {
    data.value = await getTodoChanges(props.todoId)
    const files = data.value.commit?.files ?? []
    expanded.value = Object.fromEntries(files.map(f => [f.path, files.length <= 4]))
  } catch (err) {
    const e = err as { data?: { error?: string }, message?: string }
    loadError.value = e?.data?.error || e?.message || 'Gagal menghubungi engine'
  } finally {
    loading.value = false
  }
}, { immediate: true })

const statusLabel: Record<TodoChangeFile['status'], { label: string, color: 'success' | 'error' | 'info' | 'neutral' }> = {
  added: { label: 'Baru', color: 'success' },
  modified: { label: 'Diubah', color: 'neutral' },
  deleted: { label: 'Dihapus', color: 'error' },
  renamed: { label: 'Dipindah', color: 'info' }
}

const toggle = (path: string) => {
  expanded.value[path] = !expanded.value[path]
}

const lineClass = (kind: string) => kind === 'add'
  ? 'bg-emerald-50 dark:bg-emerald-950/30 text-emerald-900 dark:text-emerald-200'
  : kind === 'del'
    ? 'bg-rose-50 dark:bg-rose-950/30 text-rose-900 dark:text-rose-200'
    : 'text-neutral-600 dark:text-neutral-400'
</script>

<template>
  <UModal :open="open" :ui="{ content: 'max-w-5xl' }" @update:open="emit('update:open', $event)">
    <template #content>
      <div class="p-5 space-y-4 max-h-[88vh] overflow-y-auto">
        <div class="pb-3 border-b border-neutral-200 dark:border-white/[0.08]">
          <h3 class="text-base font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-git-commit-horizontal" class="size-5 text-sky-500" />
            Perubahan hasil todo #{{ todoId }}
          </h3>
          <p v-if="todoTitle" class="mt-1 text-xs text-neutral-500 truncate">
            {{ todoTitle }}
          </p>
        </div>

        <div v-if="loading" class="py-12 flex flex-col items-center gap-2 text-xs text-neutral-500">
          <UIcon name="i-lucide-loader-circle" class="size-8 animate-spin text-neutral-400" />
          Membaca commit dari repo proyek…
        </div>

        <UAlert v-else-if="loadError" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="loadError" />

        <div v-else-if="!commit" class="p-8 text-center rounded-xl border border-neutral-200 dark:border-white/[0.08] space-y-2">
          <div class="mx-auto size-12 rounded-full bg-sky-50 dark:bg-sky-950/40 text-sky-600 flex items-center justify-center">
            <UIcon name="i-lucide-git-commit-horizontal" class="size-6" />
          </div>
          <p class="text-base font-bold text-neutral-900 dark:text-white">
            Belum ada commit yang bisa diperiksa
          </p>
          <p class="text-xs text-neutral-500 max-w-md mx-auto">
            {{ data?.reason }}
          </p>
        </div>

        <template v-else>
          <div class="rounded-xl border border-neutral-200 dark:border-white/[0.08] px-4 py-3 space-y-2">
            <p class="text-sm font-semibold text-neutral-900 dark:text-white">
              {{ commit.subject }}
            </p>
            <div class="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs text-neutral-500">
              <button
                type="button"
                class="inline-flex items-center gap-1.5 font-mono text-neutral-800 dark:text-neutral-200 hover:text-sky-600"
                :title="`SHA utuh: ${commit.sha} — klik untuk menyalin`"
                @click="copyToClipboard(commit.sha, `SHA ${commit.short} disalin utuh`)"
              >
                <UIcon name="i-lucide-copy" class="size-3.5" />
                {{ commit.sha }}
              </button>
              <span class="inline-flex items-center gap-1"><UIcon name="i-lucide-folder-git-2" class="size-3.5" />{{ commit.repo }}</span>
              <span class="inline-flex items-center gap-1"><UIcon name="i-lucide-user" class="size-3.5" />{{ commit.author }}</span>
              <span class="inline-flex items-center gap-1"><UIcon name="i-lucide-clock" class="size-3.5" />{{ formatDateTime(commit.date) }}</span>
              <span class="tabular-nums">
                {{ commit.files.length }} berkas ·
                <span class="text-emerald-600">+{{ commit.added }}</span>
                <span class="text-rose-600">−{{ commit.deleted }}</span>
              </span>
            </div>
            <p v-if="data?.via_todo_id" class="text-xs text-neutral-500">
              Dikerjakan sekaligus lewat todo gabungan #{{ data.via_todo_id }}.
            </p>
            <div v-if="data?.functions?.length" class="flex flex-wrap items-center gap-1.5 pt-1">
              <span class="text-xs font-medium text-neutral-500">Fungsi yang berubah ({{ data.functions.length }}):</span>
              <span
                v-for="fn in data.functions"
                :key="fn"
                class="font-mono text-xs px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-white/[0.06] border border-neutral-200 dark:border-white/[0.08] text-neutral-800 dark:text-neutral-200"
              >{{ fn }}</span>
            </div>
          </div>

          <UAlert
            v-if="commit.truncated"
            color="warning"
            variant="subtle"
            icon="i-lucide-scissors"
            description="Commit ini besar: sebagian berkas atau baris tidak ditampilkan. Periksa selengkapnya dengan git show memakai SHA di atas."
          />

          <div class="space-y-3">
            <div
              v-for="f in commit.files"
              :key="f.path"
              class="rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] overflow-hidden"
            >
              <button
                type="button"
                class="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-neutral-50 dark:hover:bg-white/[0.03]"
                :title="expanded[f.path] ? 'Klik untuk menutup diff' : 'Klik untuk membuka diff'"
                @click="toggle(f.path)"
              >
                <UIcon :name="expanded[f.path] ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="size-4 text-neutral-400 shrink-0" />
                <UBadge size="xs" variant="subtle" :color="statusLabel[f.status].color">
                  {{ statusLabel[f.status].label }}
                </UBadge>
                <span class="font-mono text-xs text-neutral-900 dark:text-white truncate">
                  <template v-if="f.old_path">{{ f.old_path }} → </template>{{ f.path }}
                </span>
                <span class="ml-auto text-xs tabular-nums shrink-0">
                  <span class="text-emerald-600">+{{ f.added }}</span>
                  <span class="text-rose-600 ml-1">−{{ f.deleted }}</span>
                </span>
              </button>
              <div v-if="f.functions?.length" class="px-3 pb-2 -mt-1 flex flex-wrap gap-1 text-xs text-neutral-500">
                <span>Fungsi:</span>
                <span v-for="fn in f.functions" :key="fn" class="font-mono text-neutral-700 dark:text-neutral-300">{{ fn }}</span>
              </div>

              <div v-if="expanded[f.path]" class="border-t border-neutral-200 dark:border-white/[0.08]">
                <p v-if="f.binary" class="px-3 py-2 text-xs text-neutral-500">
                  Berkas biner — isi tidak ditampilkan.
                </p>
                <div v-for="(h, hi) in f.hunks" :key="hi" class="text-xs">
                  <div class="px-3 py-1 font-mono bg-sky-50/60 dark:bg-sky-950/20 text-sky-800 dark:text-sky-300 border-b border-neutral-100 dark:border-white/[0.05] truncate">
                    <template v-if="h.function">
                      di fungsi <span class="font-semibold">{{ h.function }}</span> ·
                    </template>
                    {{ h.header.replace(/ @@.*$/, ' @@') }}
                  </div>
                  <div class="overflow-x-auto">
                    <table class="font-mono text-xs leading-5 min-w-full">
                      <tbody>
                        <tr v-for="(ln, li) in h.lines" :key="li" :class="lineClass(ln.kind)">
                          <td class="select-none px-2 text-right tabular-nums text-neutral-400 w-10">
                            {{ ln.old || '' }}
                          </td>
                          <td class="select-none px-2 text-right tabular-nums text-neutral-400 w-10">
                            {{ ln.new || '' }}
                          </td>
                          <td class="select-none pl-1 w-4">
                            {{ ln.kind === 'add' ? '+' : ln.kind === 'del' ? '−' : '' }}
                          </td>
                          <td class="pr-3 whitespace-pre" v-text="ln.text" />
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>
                <p v-if="f.truncated" class="px-3 py-1.5 text-xs text-amber-700 dark:text-amber-300 border-t border-neutral-100 dark:border-white/[0.05]">
                  Diff berkas ini dipotong — lihat selengkapnya dengan git show.
                </p>
              </div>
            </div>
          </div>
        </template>

        <div class="flex justify-end pt-2 border-t border-neutral-100 dark:border-white/[0.08]">
          <UButton size="sm" color="neutral" variant="outline" label="Tutup" @click="emit('update:open', false)" />
        </div>
      </div>
    </template>
  </UModal>
</template>
