<script setup lang="ts">
import type { CostCallee, CostEvidence, CostSourceWindow } from '~/types'
import { EDITOR_CONTEXT, editorSegments, evidenceMark, tokenizeLine, type TokenKind } from '~/utils/codeEditor'

// Bukti rumus biaya dalam tampilan editor: potongan berkas asli (nomor baris, indentasi, pewarnaan sintaks) di
// sekitar baris loop/query/pemanggilan yang membentuk T(n) dan Q(n). Baris query yang ikut tumbuh bersama data
// (N+1, Q(n)) ditandai merah; biaya yang datang dari fungsi lain bisa diklik untuk membuka fungsi itu.
const props = defineProps<{ title: string, file?: string, evidence?: CostEvidence[], emptyText?: string, projectId?: number }>()
const emit = defineEmits<{ openFunc: [callee: CostCallee] }>()

const { getProcessCostSource } = useApi()
const rows = computed(() => props.evidence || [])
const evidenceAt = computed(() => {
  const m = new Map<number, CostEvidence[]>()
  for (const e of rows.value) m.set(e.line, [...(m.get(e.line) || []), e])
  return m
})

// Potongan berkas dimuat saat bukti dibuka; bila gagal, baris bukti saja yang tampil (tanpa konteks).
const win = shallowRef<CostSourceWindow | null>(null)
const load = async () => {
  win.value = null
  const lines = rows.value.map(e => e.line)
  if (!props.projectId || !props.file || !lines.length) return
  try {
    win.value = await getProcessCostSource(props.projectId, props.file, Math.max(1, Math.min(...lines) - EDITOR_CONTEXT), Math.max(...lines) + EDITOR_CONTEXT)
  } catch {
    win.value = null
  }
}
watch(() => [props.projectId, props.file, rows.value.map(e => e.line).join(',')], load, { immediate: true })

interface Line { n: number, code: string, marks: CostEvidence[] }
type Block = { fold: number, key: string } | { lines: Line[], key: string }

const textAt = (n: number): string | null => {
  const w = win.value
  if (w && n >= w.from && n < w.from + w.lines.length) return w.lines[n - w.from] ?? ''
  return evidenceAt.value.get(n)?.[0]?.code ?? null
}
const blocks = computed<Block[]>(() => {
  const lines = rows.value.map(e => e.line)
  const segs = win.value ? editorSegments(lines, EDITOR_CONTEXT, win.value.total) : editorSegments(lines, 0).flatMap(s => lines.filter(n => n >= s.from && n <= s.to).map(n => ({ from: n, to: n })))
  const out: Block[] = []
  segs.forEach((s, i) => {
    const prev = segs[i - 1]
    if (prev && s.from - prev.to > 1) out.push({ fold: s.from - prev.to - 1, key: `f${s.from}` })
    const ls: Line[] = []
    for (let n = s.from; n <= s.to; n++) {
      const code = textAt(n)
      if (code !== null) ls.push({ n, code, marks: evidenceAt.value.get(n) || [] })
    }
    out.push({ lines: ls, key: `s${s.from}` })
  })
  return out
})

const TOKEN_CLASS: Record<TokenKind, string> = {
  plain: '', comment: 'text-[#8b949e] italic', string: 'text-[#a5d6ff]', keyword: 'text-[#ff7b72]',
  number: 'text-[#79c0ff]', func: 'text-[#d2a8ff]', var: 'text-[#ffa657]'
}
const LINE_TONE = {
  heavy: 'bg-rose-500/[0.16] border-rose-400',
  query: 'bg-white/[0.05] border-slate-400',
  loop: 'bg-amber-400/[0.10] border-amber-400',
  call: 'bg-sky-400/[0.10] border-sky-400'
} as const
const BADGE_TONE = {
  heavy: 'bg-rose-500/25 text-rose-200',
  query: 'bg-white/10 text-slate-200',
  loop: 'bg-amber-400/20 text-amber-200',
  call: 'bg-sky-400/20 text-sky-200'
} as const
const lineTone = (l: Line) => {
  if (!l.marks.length) return 'border-transparent'
  const order = ['heavy', 'query', 'loop', 'call'] as const
  const tones = l.marks.map(e => evidenceMark(e).tone)
  return LINE_TONE[order.find(t => tones.includes(t)) || 'call']
}
const heavyCount = computed(() => rows.value.filter(e => evidenceMark(e).tone === 'heavy').length)
</script>

<template>
  <div class="rounded-lg border border-neutral-200 dark:border-white/[0.08] overflow-hidden">
    <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 px-3 py-2 border-b border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] text-xs">
      <span class="font-semibold text-neutral-900 dark:text-white">
        {{ title }} ({{ rows.length }})
        <span v-if="heavyCount" class="ml-1.5 inline-flex items-center gap-1 rounded-md bg-rose-50 dark:bg-rose-950/40 px-1.5 py-0.5 font-medium text-rose-700 dark:text-rose-300">
          <UIcon name="i-lucide-database" class="size-3" />{{ heavyCount }} query berat
        </span>
      </span>
      <span v-if="file" class="font-mono text-neutral-500 truncate max-w-full" :title="file">{{ file }}</span>
    </div>
    <p v-if="!rows.length" class="px-3 py-2.5 text-xs text-neutral-500 bg-white dark:bg-[#04111f]">
      {{ emptyText || 'Tidak ada baris yang menaikkan biaya.' }}
    </p>
    <div v-else class="bg-[#0d1117] text-[#e6edf3] font-mono text-xs leading-5 overflow-x-auto py-1" style="tab-size: 4">
      <template v-for="b in blocks" :key="b.key">
        <div v-if="'fold' in b" class="flex items-center gap-2 pl-14 py-0.5 text-[#8b949e] bg-white/[0.03] select-none font-sans">
          <UIcon name="i-lucide-fold-vertical" class="size-3" /> {{ b.fold }} baris dilipat
        </div>
        <template v-else>
          <template v-for="l in b.lines" :key="l.n">
            <div class="flex min-w-max border-l-2" :class="lineTone(l)">
              <span class="w-12 shrink-0 pr-3 text-right tabular-nums select-none" :class="l.marks.length ? 'text-[#e6edf3]' : 'text-[#6e7681]'">{{ l.n }}</span>
              <code class="whitespace-pre pr-6"><span v-for="(t, j) in tokenizeLine(l.code)" :key="j" :class="TOKEN_CLASS[t.kind]">{{ t.text }}</span></code>
            </div>
            <div
              v-for="e in l.marks"
              :key="`${e.kind}-${e.line}`"
              class="flex flex-wrap items-center gap-x-2 gap-y-1 border-l-2 pl-14 pr-4 py-1 font-sans text-xs"
              :class="LINE_TONE[evidenceMark(e).tone]"
            >
              <span class="rounded px-1.5 py-px font-mono font-semibold" :class="BADGE_TONE[evidenceMark(e).tone]">{{ evidenceMark(e).label }}</span>
              <span class="text-[#c9d1d9]">{{ e.note }}</span>
              <button
                v-if="e.callee"
                type="button"
                class="inline-flex items-center gap-1 rounded px-1.5 py-px font-mono text-sky-300 hover:text-sky-100 hover:bg-sky-400/15 underline decoration-dotted underline-offset-2 focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-400"
                :title="`Buka fungsi ${e.callee.name} (${e.callee.file}:${e.callee.line}) di daftar Fungsi`"
                @click="emit('openFunc', e.callee)"
              >
                {{ e.callee.name }}()
                <UIcon name="i-lucide-arrow-right" class="size-3" />
              </button>
            </div>
          </template>
        </template>
      </template>
    </div>
  </div>
</template>
