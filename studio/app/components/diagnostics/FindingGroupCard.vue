<script setup lang="ts">
import type { Finding } from '~/types'
import type { FindingGroup } from '~/utils/findingGroups'

// Temuan utama hasil "Satukan Serupa": bagian yang sama di atas, perbedaan tiap temuan sebagai sub temuan.
const props = defineProps<{
  group: FindingGroup
  resolvingKey: string | null
  resolvedKeys: Set<string>
  findingKey: (f: Finding) => string
}>()

const emit = defineEmits<{
  (e: 'resolveGroup', group: FindingGroup): void
  (e: 'resolve' | 'reResolve' | 'viewResolution' | 'ignore' | 'delete', finding: Finding): void
  (e: 'feedback', finding: Finding, mode: 'verify' | 'false_positive' | 'confirmed' | 'clear'): void
}>()

const PREVIEW = 8
const expanded = ref(false)
const showAll = ref(false)
const openSub = ref<string | null>(null)

const first = computed(() => props.group.findings[0])
const isDone = (f: Finding) => f.status === 'resolved' || f.status === 'ignored' || props.resolvedKeys.has(props.findingKey(f))
const openFindings = computed(() => props.group.findings.filter(f => !isDone(f)))
const todoIds = computed(() => [...new Set(props.group.findings.map(f => f.todo_id).filter((id): id is number => !!id))])
const visibleSubs = computed(() => (showAll.value ? props.group.findings : props.group.findings.slice(0, PREVIEW)))

const severityColor = (sev: string) => (sev === 'CRITICAL' ? 'error' : sev === 'WARNING' ? 'warning' : sev === 'INFO' ? 'info' : 'neutral')

const toggleSub = (f: Finding) => {
  const k = props.findingKey(f)
  openSub.value = openSub.value === k ? null : k
}
</script>

<template>
  <div
    class="p-4 rounded-xl border transition-all bg-white dark:bg-[#04111f]"
    :class="group.severity === 'CRITICAL'
      ? 'border-rose-200 dark:border-rose-900/60'
      : group.severity === 'WARNING'
        ? 'border-amber-200 dark:border-amber-900/60'
        : 'border-neutral-200 dark:border-white/[0.08]'"
  >
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="space-y-1.5 flex-1 min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <UIcon name="i-lucide-layers-2" class="w-4 h-4 shrink-0 text-primary-500" />
          <span
            v-if="first?.rule_id"
            class="px-2 py-0.5 rounded text-xs font-mono font-bold bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 border border-neutral-200 dark:border-neutral-700"
          >
            {{ first.rule_id }}
          </span>
          <h3 class="text-sm font-bold truncate text-neutral-900 dark:text-white">
            {{ group.title }}
          </h3>
          <UBadge size="xs" :color="severityColor(group.severity)" variant="subtle">
            {{ group.severity }}
          </UBadge>
          <UBadge size="xs" color="primary" variant="soft" icon="i-lucide-list-tree">
            {{ group.findings.length }} sub temuan · {{ group.fileCount }} berkas
          </UBadge>
          <UBadge v-if="openFindings.length === 0" size="xs" color="success" variant="subtle" icon="i-lucide-check-circle">
            Semua selesai
          </UBadge>
          <span
            v-for="id in todoIds.slice(0, 3)"
            :key="id"
            class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-mono font-bold bg-sky-100 dark:bg-sky-950/60 text-sky-800 dark:text-sky-300 border border-sky-300 dark:border-sky-800"
          >
            <UIcon name="i-lucide-clipboard-check" class="w-3 h-3" />
            TODO #{{ id }}
          </span>
        </div>
        <p class="text-xs text-neutral-600 dark:text-neutral-400 line-clamp-2">
          {{ first?.description }}
        </p>
      </div>

      <div class="flex items-center gap-1.5 shrink-0">
        <UButton
          v-if="openFindings.length > 0"
          size="xs"
          color="primary"
          icon="i-lucide-sparkles"
          :loading="!!resolvingKey && openFindings.some(f => findingKey(f) === resolvingKey)"
          :title="`Selesaikan ${openFindings.length} sub temuan sekaligus — satu panggilan LLM, satu todo yang mencakup semua lokasi`"
          @click="emit('resolveGroup', group)"
        >
          Selesaikan Grup ({{ openFindings.length }})
        </UButton>
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          :icon="expanded ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
          @click="expanded = !expanded"
        >
          {{ expanded ? 'Tutup' : 'Sub temuan' }}
        </UButton>
      </div>
    </div>

    <div v-if="expanded" class="mt-3 pt-3 border-t border-neutral-100 dark:border-white/[0.08] space-y-1.5">
      <template v-for="f in visibleSubs" :key="findingKey(f)">
        <button
          type="button"
          class="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg text-left text-xs hover:bg-neutral-50 dark:hover:bg-neutral-800/60 cursor-pointer"
          @click="toggleSub(f)"
        >
          <UIcon
            :name="isDone(f) ? 'i-lucide-check-circle-2' : 'i-lucide-circle-dot'"
            class="w-3.5 h-3.5 shrink-0"
            :class="isDone(f) ? 'text-emerald-500' : 'text-neutral-400'"
          />
          <span v-if="findingGroupRemainder(group, f)" class="font-medium text-neutral-800 dark:text-neutral-200 truncate">
            {{ findingGroupRemainder(group, f) }}
          </span>
          <span class="font-mono text-xs text-neutral-500 truncate">
            {{ f.target_file || 'global' }}<template v-if="f.line">:{{ f.target_lines || f.line }}</template>
          </span>
          <UBadge v-if="f.todo_id" size="xs" color="info" variant="subtle" class="ml-auto font-mono shrink-0">
            #{{ f.todo_id }}
          </UBadge>
          <UIcon :name="openSub === findingKey(f) ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="w-3.5 h-3.5 shrink-0 text-neutral-400" :class="{ 'ml-auto': !f.todo_id }" />
        </button>
        <DiagnosticsFindingCard
          v-if="openSub === findingKey(f)"
          class="ml-5"
          :finding="f"
          :resolving="resolvingKey === findingKey(f)"
          :has-created-todo="resolvedKeys.has(findingKey(f))"
          @resolve="x => emit('resolve', x)"
          @re-resolve="x => emit('reResolve', x)"
          @view-resolution="x => emit('viewResolution', x)"
          @ignore="x => emit('ignore', x)"
          @delete="x => emit('delete', x)"
          @feedback="(x, mode) => emit('feedback', x, mode)"
        />
      </template>
      <UButton
        v-if="group.findings.length > PREVIEW"
        size="xs"
        color="neutral"
        variant="link"
        @click="showAll = !showAll"
      >
        {{ showAll ? 'Tampilkan lebih sedikit' : `Tampilkan semua (${group.findings.length})` }}
      </UButton>
    </div>
  </div>
</template>
