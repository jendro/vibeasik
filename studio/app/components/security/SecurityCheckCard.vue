<script setup lang="ts">
// Satu kontrol keamanan: status mewarnai garis tepi, bukti & lokasi kode, saran perbaikan.
import type { SecurityCheck } from '~/types'
import { SEC_CATEGORY_LABEL, SEC_STATUS_META } from '~/utils/securityTest'

const props = defineProps<{ check: SecurityCheck }>()
const meta = computed(() => SEC_STATUS_META[props.check.status])
const open = ref(props.check.status === 'fail')
</script>

<template>
  <article class="p-4 rounded-xl border bg-white dark:bg-[#04111f]" :class="meta.border">
    <button type="button" class="w-full flex items-start gap-3 text-left" :aria-expanded="open" @click="open = !open">
      <UIcon :name="meta.icon" class="size-4 mt-0.5 shrink-0" :class="meta.text" />
      <div class="min-w-0 flex-1 space-y-1">
        <div class="flex flex-wrap items-center gap-2">
          <span class="font-mono text-xs font-bold px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-white/[0.06] border border-neutral-200 dark:border-white/[0.08] text-neutral-700 dark:text-slate-300">{{ check.id }}</span>
          <h3 class="text-sm font-bold text-neutral-900 dark:text-white truncate">
            {{ check.title }}
          </h3>
          <UBadge size="xs" variant="subtle" :color="meta.badge" :label="meta.label" />
          <UBadge size="xs" variant="outline" color="neutral" :icon="check.source === 'live' ? 'i-lucide-radio-tower' : 'i-lucide-file-code-2'" :label="check.source === 'live' ? 'Server' : 'Kode'" />
          <span class="text-xs text-neutral-500 dark:text-slate-400">{{ SEC_CATEGORY_LABEL[check.category] }}</span>
        </div>
        <p v-if="check.evidence?.length" class="text-xs text-neutral-600 dark:text-slate-400" :class="open ? '' : 'line-clamp-2'">
          {{ check.evidence[0] }}
        </p>
      </div>
      <UIcon name="i-lucide-chevron-down" class="size-4 shrink-0 text-neutral-400 transition-transform" :class="open ? 'rotate-180' : ''" />
    </button>

    <div v-if="open" class="mt-3 pl-7 space-y-3">
      <ul v-if="(check.evidence?.length || 0) > 1" class="space-y-1 text-xs text-neutral-600 dark:text-slate-400 list-disc pl-4">
        <li v-for="(e, i) in check.evidence!.slice(1)" :key="i">
          {{ e }}
        </li>
      </ul>
      <div v-if="check.locations?.length" class="space-y-1">
        <p class="text-xs font-medium text-neutral-500 dark:text-slate-400">
          Lokasi di kode ({{ check.locations.length }})
        </p>
        <div v-for="(l, i) in check.locations" :key="i" class="rounded-lg border border-neutral-200 dark:border-white/[0.08] bg-neutral-50 dark:bg-white/[0.03] px-2.5 py-1.5">
          <p class="font-mono text-xs text-neutral-800 dark:text-slate-200 truncate">
            {{ l.file }}:{{ l.line }}
          </p>
          <p v-if="l.snippet" class="font-mono text-xs text-neutral-500 dark:text-slate-400 truncate">
            {{ l.snippet }}
          </p>
        </div>
      </div>
      <p v-if="check.suggestion" class="flex gap-2 text-xs text-neutral-700 dark:text-slate-300">
        <UIcon name="i-lucide-wrench" class="size-3.5 mt-0.5 shrink-0 text-sky-500" />
        <span>{{ check.suggestion }}<span v-if="check.guideline_id" class="font-mono text-neutral-400"> · {{ check.guideline_id }}</span></span>
      </p>
    </div>
  </article>
</template>
