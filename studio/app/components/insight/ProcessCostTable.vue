<script setup lang="ts">
import type { FuncCost, ProcessCost, RouteCost } from '~/types'
import { costClassInfo, costStackLabel, fmtTicks, shortFormula } from '~/utils/processCost'

// Tabel rumus biaya per rute atau per fungsi; klik baris untuk melihat pertumbuhan T(n) pada n = 1 … 10.000.
const props = defineProps<{ kind: 'routes' | 'funcs', routes: RouteCost[], funcs: FuncCost[] }>()

type Row = ProcessCost & { key: string, title: string, sub: string, method?: string }
const rows = computed<Row[]>(() => props.kind === 'routes'
  ? props.routes.map(r => ({ ...r, key: `${r.module}|${r.method}|${r.path}`, title: r.path, method: r.method, sub: `${costStackLabel(r)} · ${r.handler}${r.middlewares ? ` · +${r.middlewares} middleware` : ''}` }))
  : props.funcs.map(f => ({ ...f, key: `${f.module}|${f.file}|${f.name}|${f.line}`, title: f.name, sub: `${costStackLabel(f)} · ${f.file}:${f.line}${f.routes ? ` · dipakai ${f.routes} rute` : ''}` })))

const open = ref<string | null>(null)
const toggle = (key: string) => { open.value = open.value === key ? null : key }

const METHOD_TONE: Record<string, string> = {
  GET: 'text-emerald-700 dark:text-emerald-400', POST: 'text-sky-700 dark:text-sky-400', PUT: 'text-amber-700 dark:text-amber-400',
  PATCH: 'text-amber-700 dark:text-amber-400', DELETE: 'text-rose-700 dark:text-rose-400'
}
</script>

<template>
  <div class="overflow-x-auto rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f]">
    <table class="w-full min-w-[760px] text-sm">
      <thead>
        <tr class="text-left text-xs text-neutral-500 border-b border-neutral-200 dark:border-white/[0.08]">
          <th class="font-medium px-5 py-2.5">
            {{ kind === 'routes' ? 'Rute' : 'Fungsi' }}
          </th>
          <th class="font-medium px-3 py-2.5">
            Rumus tick T(n)
          </th>
          <th class="font-medium px-3 py-2.5">
            Pertumbuhan
          </th>
          <th class="font-medium px-3 py-2.5 text-right">
            T(1)
          </th>
          <th class="font-medium px-3 py-2.5 pr-5">
            Query Q(n)
          </th>
        </tr>
      </thead>
      <tbody>
        <template v-for="r in rows" :key="r.key">
          <tr
            class="border-b border-neutral-100 dark:border-white/[0.05] cursor-pointer hover:bg-neutral-50 dark:hover:bg-white/[0.03] focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-sky-500"
            tabindex="0"
            role="button"
            :aria-expanded="open === r.key"
            @click="toggle(r.key)"
            @keydown.enter.prevent="toggle(r.key)"
            @keydown.space.prevent="toggle(r.key)"
          >
            <td class="px-5 py-2.5 max-w-[340px]">
              <div class="flex items-baseline gap-2 min-w-0">
                <span v-if="r.method" class="font-mono text-xs font-semibold w-12 shrink-0" :class="METHOD_TONE[r.method] || 'text-neutral-500'">{{ r.method }}</span>
                <span class="font-mono text-xs text-neutral-900 dark:text-white truncate" :title="r.title">{{ r.title }}</span>
              </div>
              <div class="text-xs text-neutral-500 truncate" :class="r.method ? 'pl-14' : ''" :title="r.sub">
                {{ r.sub }}
              </div>
            </td>
            <td class="px-3 py-2.5">
              <code class="font-mono text-xs text-neutral-900 dark:text-white whitespace-nowrap" :title="r.formula">{{ shortFormula(r.formula) }}</code>
            </td>
            <td class="px-3 py-2.5 whitespace-nowrap">
              <span class="inline-flex items-center gap-1.5">
                <span class="size-2 rounded-full" :class="costClassInfo(r.class).bar" />
                <span class="font-mono text-xs font-semibold" :class="costClassInfo(r.class).text">{{ r.big_o }}</span>
                <span class="text-xs text-neutral-500">{{ costClassInfo(r.class).label }}</span>
              </span>
              <span v-if="r.recursive" class="ml-1.5 text-xs text-amber-700 dark:text-amber-400" title="Ada rekursi: rumus adalah batas bawah">rekursif</span>
              <span v-if="r.unbounded" class="ml-1.5 text-xs text-amber-700 dark:text-amber-400" title="Ada loop tanpa syarat berhenti (for {})">loop tanpa batas</span>
            </td>
            <td class="px-3 py-2.5 text-right font-mono text-xs tabular-nums text-neutral-900 dark:text-white">
              {{ fmtTicks(r.ticks_1) }}
            </td>
            <td class="px-3 py-2.5 pr-5 whitespace-nowrap">
              <code class="font-mono text-xs" :class="r.n_plus_one ? 'text-rose-700 dark:text-rose-400 font-semibold' : 'text-neutral-700 dark:text-neutral-300'">{{ r.query_formula }}</code>
              <span v-if="r.n_plus_one" class="ml-1.5 inline-flex items-center gap-1 rounded-md bg-rose-50 dark:bg-rose-950/40 px-1.5 py-0.5 text-xs font-medium text-rose-700 dark:text-rose-300">
                <UIcon name="i-lucide-triangle-alert" class="size-3" />N+1
              </span>
            </td>
          </tr>
          <tr v-if="open === r.key" class="border-b border-neutral-100 dark:border-white/[0.05] bg-neutral-50/70 dark:bg-white/[0.02]">
            <td colspan="5" class="px-5 py-3">
              <div class="text-xs text-neutral-500">
                Rumus lengkap
              </div>
              <code class="block font-mono text-xs text-neutral-900 dark:text-white break-words mb-3">T(n) = {{ r.formula }}</code>
              <div class="text-xs text-neutral-500 mb-2">
                Bila jumlah data tumbuh — {{ costClassInfo(r.class).meaning }}
              </div>
              <div class="grid grid-cols-5 gap-2 max-w-2xl">
                <div v-for="g in r.growth" :key="g.n" class="rounded-lg border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] px-3 py-2">
                  <div class="text-xs text-neutral-500">
                    n = {{ fmtTicks(g.n) }}
                  </div>
                  <div class="font-mono text-sm font-semibold tabular-nums text-neutral-900 dark:text-white">
                    {{ fmtTicks(g.ticks) }}
                  </div>
                  <div class="text-xs" :class="r.n_plus_one ? 'text-rose-700 dark:text-rose-400' : 'text-neutral-500'">
                    {{ fmtTicks(g.queries) }} query
                  </div>
                </div>
              </div>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
  </div>
</template>
