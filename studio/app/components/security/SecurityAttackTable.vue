<script setup lang="ts">
import type { SecurityAttack } from '~/types'
import { ATTACK_CATEGORY_LABEL, ATTACK_VERDICT_META, filterAttacks, type AttackFilter } from '~/utils/securityTest'

// Serangan yang dikirim Uji Keamanan (bentuk sama dengan tabel endpoint Uji Runtime): payload, rute, HTTP,
// waktu, dan vonis. Bawaan menampilkan yang tembus & mencurigakan lebih dulu.
const props = defineProps<{ attacks: SecurityAttack[] }>()

const filter = reactive<AttackFilter>({ verdict: 'problems', category: 'all', q: '' })

const counts = computed(() => {
  const out: Record<string, number> = { vulnerable: 0, suspect: 0, safe: 0, blocked: 0, error: 0 }
  for (const a of props.attacks) out[a.verdict] = (out[a.verdict] || 0) + 1
  return out
})
const routes = computed(() => new Set(props.attacks.map(a => `${a.method} ${a.route}`)).size)
const list = computed(() => filterAttacks(props.attacks, filter))
const categoryOptions = computed(() => {
  const per: Record<string, number> = {}
  for (const a of props.attacks) per[a.category] = (per[a.category] || 0) + 1
  return [
    { label: `Semua serangan (${props.attacks.length})`, value: 'all' },
    ...Object.entries(per).map(([k, n]) => ({ label: `${ATTACK_CATEGORY_LABEL[k] || k} (${n})`, value: k }))
  ]
})
const filters = computed(() => [
  { key: 'problems', label: `Tembus & curiga (${counts.value.vulnerable + counts.value.suspect})` },
  { key: 'all', label: `Semua (${props.attacks.length})` },
  { key: 'blocked', label: `Ditahan (${counts.value.blocked})` }
] as const)

const methodCls = (m: string) => ({
  GET: 'text-sky-600 dark:text-sky-400',
  POST: 'text-emerald-600 dark:text-emerald-400',
  TRACE: 'text-violet-600 dark:text-violet-400'
}[m] || 'text-neutral-500')
const statusColor = (code?: number) => {
  if (!code) return 'text-neutral-400'
  if (code >= 500) return 'text-rose-600 dark:text-rose-400'
  if (code >= 400) return 'text-amber-600 dark:text-amber-400'
  if (code >= 300) return 'text-sky-600 dark:text-sky-400'
  return 'text-emerald-600 dark:text-emerald-400'
}
</script>

<template>
  <section class="rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] overflow-hidden">
    <div class="px-4 py-3 flex flex-col xl:flex-row xl:items-center justify-between gap-2 border-b border-neutral-200 dark:border-white/[0.08]">
      <div class="flex items-center gap-2 text-xs flex-wrap min-w-0">
        <UIcon name="i-lucide-swords" class="size-4 text-rose-500" />
        <span class="text-sm font-semibold text-neutral-900 dark:text-white">Serangan yang dicoba</span>
        <span class="text-neutral-500 tabular-nums">{{ attacks.length }} request ke {{ routes }} sasaran ·
          <b :class="counts.vulnerable ? 'text-rose-600 dark:text-rose-400' : 'text-emerald-600 dark:text-emerald-400'">{{ counts.vulnerable }} tembus</b>
          · {{ counts.suspect }} mencurigakan · {{ counts.blocked }} ditahan rate limit</span>
      </div>
      <div class="flex items-center gap-1.5 flex-wrap">
        <button
          v-for="f in filters"
          :key="f.key"
          type="button"
          class="px-3 py-1.5 rounded-lg text-xs font-medium whitespace-nowrap transition-colors cursor-pointer"
          :class="filter.verdict === f.key ? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900' : 'text-neutral-600 dark:text-slate-400 hover:bg-neutral-100 dark:hover:bg-white/[0.05]'"
          @click="filter.verdict = f.key"
        >
          {{ f.label }}
        </button>
        <USelect v-model="filter.category" :items="categoryOptions" size="sm" class="w-48" />
        <UInput v-model="filter.q" size="sm" icon="i-lucide-search" placeholder="Cari path / payload…" class="w-44" />
      </div>
    </div>

    <div v-if="list.length === 0" class="p-6 text-center text-xs text-neutral-500">
      {{ attacks.length === 0 ? 'Serangan muncul di sini setelah server diuji.' : filter.verdict === 'problems' ? 'Tidak ada serangan yang tembus atau mencurigakan.' : 'Tidak ada serangan yang cocok dengan filter.' }}
    </div>
    <div v-else class="max-h-[28rem] overflow-auto">
      <table class="w-full text-xs min-w-[760px]">
        <thead class="sticky top-0 bg-neutral-50 dark:bg-[#04111f] text-neutral-500">
          <tr>
            <th class="text-left font-semibold px-4 py-2">
              Jenis
            </th>
            <th class="text-left font-semibold px-2 py-2">
              Sasaran
            </th>
            <th class="text-left font-semibold px-2 py-2">
              Payload
            </th>
            <th class="text-right font-semibold px-2 py-2">
              HTTP
            </th>
            <th class="text-right font-semibold px-2 py-2">
              Waktu
            </th>
            <th class="text-right font-semibold px-4 py-2">
              Hasil
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-neutral-100 dark:divide-white/[0.06]">
          <tr v-for="(a, idx) in list.slice(0, 400)" :key="idx" class="hover:bg-neutral-50 dark:hover:bg-white/[0.03] align-top">
            <td class="px-4 py-1.5 whitespace-nowrap text-neutral-700 dark:text-slate-300">
              {{ ATTACK_CATEGORY_LABEL[a.category] || a.category }}
            </td>
            <td class="px-2 py-1.5 font-mono text-neutral-900 dark:text-white max-w-[260px]">
              <span class="font-bold mr-1" :class="methodCls(a.method)">{{ a.method }}</span>
              <span class="break-all" :title="a.path">{{ a.route }}</span>
            </td>
            <td class="px-2 py-1.5 font-mono text-neutral-600 dark:text-slate-400 max-w-[240px] break-all">
              {{ a.payload }}
              <p v-if="a.note" class="font-sans text-neutral-500 dark:text-slate-400">
                {{ a.note }}
              </p>
            </td>
            <td class="px-2 py-1.5 text-right font-mono font-bold" :class="statusColor(a.status)">
              {{ a.status || '—' }}
            </td>
            <td class="px-2 py-1.5 text-right font-mono text-neutral-500 tabular-nums">
              {{ a.latency_ms }}ms
            </td>
            <td class="px-4 py-1.5 text-right">
              <span class="px-1.5 py-0.5 rounded text-xs font-semibold whitespace-nowrap" :class="ATTACK_VERDICT_META[a.verdict]?.cls">{{ ATTACK_VERDICT_META[a.verdict]?.label || a.verdict }}</span>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="list.length > 400" class="px-4 py-2 text-xs text-neutral-500">
        Menampilkan 400 dari {{ list.length }} — persempit dengan filter.
      </p>
    </div>
  </section>
</template>
