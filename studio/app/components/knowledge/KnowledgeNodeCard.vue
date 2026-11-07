<script setup lang="ts">
import type { KnowledgeHit } from '~/types'
import { KIND_META, SOURCE_LABEL } from '~/utils/knowledge'

// Satu node Pusat Pengetahuan: jenis, domain, asal, bukti (✓/✗ → utilitas), dan aksi umpan balik.
const props = defineProps<{ node: KnowledgeHit, showScore?: boolean }>()
const emit = defineEmits<{
  (e: 'open', n: KnowledgeHit): void
  (e: 'feedback', n: KnowledgeHit, helpful: boolean): void
  (e: 'retire', n: KnowledgeHit): void
}>()

const kind = computed(() => KIND_META[props.node.kind] ?? KIND_META.lesson)
const utilityPct = computed(() => Math.round((props.node.utility || (props.node.helpful + 1) / (props.node.helpful + props.node.harmful + 2)) * 100))
const tested = computed(() => props.node.helpful + props.node.harmful > 0)
const barColor = computed(() => utilityPct.value >= 70 ? 'bg-emerald-500' : utilityPct.value >= 45 ? 'bg-amber-500' : 'bg-rose-500')
</script>

<template>
  <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-2">
    <div class="flex items-start justify-between gap-3">
      <button type="button" class="min-w-0 text-left" @click="emit('open', node)">
        <div class="flex items-center gap-1.5 flex-wrap">
          <span class="text-[11px] font-mono text-neutral-400">K{{ node.id }}</span>
          <UBadge size="xs" :color="kind.color" variant="subtle" :icon="kind.icon">
            {{ kind.label }}
          </UBadge>
          <UBadge v-if="node.domain" size="xs" color="neutral" variant="outline">
            {{ node.domain }}
          </UBadge>
          <UBadge size="xs" color="neutral" variant="soft">
            {{ SOURCE_LABEL[node.source] || node.source }}
          </UBadge>
          <UBadge v-if="node.owner_employee_id" size="xs" color="info" variant="soft" icon="i-lucide-user">
            {{ node.owner_name || 'Pribadi' }}
          </UBadge>
          <UBadge v-else-if="node.owner_name" size="xs" color="success" variant="soft" icon="i-lucide-share-2">
            dari {{ node.owner_name }}
          </UBadge>
          <UBadge v-if="node.via" size="xs" color="secondary" variant="soft" icon="i-lucide-link">
            lewat tautan
          </UBadge>
          <UBadge v-if="node.needs_review" size="xs" color="warning" variant="soft" icon="i-lucide-triangle-alert">
            Perlu ditinjau
          </UBadge>
        </div>
        <p class="mt-1 text-sm font-semibold text-neutral-900 dark:text-white">
          {{ node.title }}
        </p>
        <p class="text-xs text-neutral-600 dark:text-neutral-400 line-clamp-3 mt-0.5">
          {{ node.content }}
        </p>
      </button>
      <div class="flex shrink-0 gap-1">
        <UButton size="xs" color="success" variant="ghost" icon="i-lucide-thumbs-up" aria-label="Tandai membantu" @click="emit('feedback', node, true)" />
        <UButton size="xs" color="error" variant="ghost" icon="i-lucide-thumbs-down" aria-label="Tandai keliru" @click="emit('feedback', node, false)" />
        <UButton v-if="node.status === 'active'" size="xs" color="neutral" variant="ghost" icon="i-lucide-archive" aria-label="Pensiunkan" @click="emit('retire', node)" />
      </div>
    </div>
    <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-neutral-500">
      <div class="flex items-center gap-1.5 w-32" :title="tested ? 'Utilitas dari bukti ✓/✗' : 'Belum teruji (utilitas awal 50%)'">
        <div class="h-1.5 flex-1 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
          <div class="h-full rounded-full" :class="tested ? barColor : 'bg-neutral-300 dark:bg-neutral-600'" :style="{ width: `${utilityPct}%` }" />
        </div>
        <span>{{ utilityPct }}%</span>
      </div>
      <span class="whitespace-nowrap">✓{{ node.helpful }} ✗{{ node.harmful }}</span>
      <span class="whitespace-nowrap">dipakai {{ node.uses }}×</span>
      <span class="whitespace-nowrap">bobot {{ node.weight }}</span>
      <span v-if="showScore" class="sm:ml-auto font-mono">skor {{ node.score.toFixed(3) }} · relevansi {{ node.relevance.toFixed(3) }}</span>
    </div>
  </div>
</template>
