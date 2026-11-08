<script setup lang="ts">
/**
 * Kalkulator skor temu-kembali (rumus Recall di local-engine/internal/knowledge/recall.go): geser tiap faktor untuk
 * melihat pengaruhnya ke skor sebuah butir dan apakah ia lolos ke prompt.
 */
const rel = ref(70) // relevansi ternormalisasi 0–100 (kosinus + bonus domain, dibagi relevansi tertinggi)
const helpful = ref(4)
const harmful = ref(1)
const weight = ref(70)
const days = ref(10)
const authoritative = ref(false)

const utility = computed(() => authoritative.value ? 1 : (helpful.value + 1) / (helpful.value + harmful.value + 2))
const recency = computed(() => Math.exp(-days.value / 45))
const blocked = computed(() => !authoritative.value && harmful.value >= 3 && utility.value < 0.25)
const parts = computed(() => [
  { label: 'Relevansi', w: 0.55, v: rel.value / 100, color: '#6366f1' },
  { label: 'Utilitas', w: 0.2, v: utility.value, color: '#10b981' },
  { label: 'Bobot', w: 0.15, v: weight.value / 100, color: '#f59e0b' },
  { label: 'Kebaruan', w: 0.1, v: recency.value, color: '#0ea5e9' }
])
const score = computed(() => parts.value.reduce((a, p) => a + p.w * p.v, 0))
const pct = (v: number) => `${Math.round(v * 100)}%`
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
    <div class="space-y-4">
      <label class="block">
        <span class="flex justify-between text-xs font-medium text-neutral-700 dark:text-neutral-300">Relevansi ke pekerjaan <span class="font-mono">{{ rel }}%</span></span>
        <USlider v-model="rel" :min="0" :max="100" size="sm" class="mt-1.5" />
        <span class="text-xs text-neutral-500">Kemiripan kosinus TF-IDF (+0,12 bila domain sama), dibagi relevansi tertinggi di kandidat.</span>
      </label>
      <div class="grid grid-cols-2 gap-3">
        <label class="block">
          <span class="flex justify-between text-xs font-medium text-neutral-700 dark:text-neutral-300">Bukti ✓ membantu <span class="font-mono">{{ helpful }}</span></span>
          <USlider v-model="helpful" :min="0" :max="30" size="sm" class="mt-1.5" />
        </label>
        <label class="block">
          <span class="flex justify-between text-xs font-medium text-neutral-700 dark:text-neutral-300">Bukti ✗ keliru <span class="font-mono">{{ harmful }}</span></span>
          <USlider v-model="harmful" :min="0" :max="30" size="sm" color="error" class="mt-1.5" />
        </label>
      </div>
      <label class="block">
        <span class="flex justify-between text-xs font-medium text-neutral-700 dark:text-neutral-300">Bobot awal <span class="font-mono">{{ weight }}</span></span>
        <USlider v-model="weight" :min="0" :max="100" size="sm" class="mt-1.5" />
        <span class="text-xs text-neutral-500">Dari severity guideline (critical 95 · high 85 · medium 70 · low 55) atau penulisnya; +3 tiap kali butir yang sama diajarkan lagi.</span>
      </label>
      <label class="block">
        <span class="flex justify-between text-xs font-medium text-neutral-700 dark:text-neutral-300">Hari sejak terakhir dipakai <span class="font-mono">{{ days }}</span></span>
        <USlider v-model="days" :min="0" :max="365" size="sm" class="mt-1.5" />
      </label>
      <USwitch v-model="authoritative" size="sm" label="Sumber berkedudukan tinggi (Galeri Desain) — utilitas selalu 1" />
    </div>

    <div class="space-y-3">
      <div class="flex items-end gap-3">
        <span class="text-4xl font-bold tabular-nums" :class="blocked ? 'text-rose-500 line-through' : 'text-neutral-900 dark:text-white'">{{ pct(score) }}</span>
        <span class="text-xs text-neutral-500 pb-1.5">skor akhir</span>
      </div>
      <div class="flex h-3 rounded-full overflow-hidden bg-neutral-100 dark:bg-neutral-800">
        <span v-for="p in parts" :key="p.label" :style="{ width: pct(p.w * p.v), background: p.color }" :title="p.label" />
      </div>
      <table class="w-full text-xs">
        <tbody>
          <tr v-for="p in parts" :key="p.label" class="border-b border-neutral-100 dark:border-neutral-800">
            <td class="py-1.5">
              <span class="inline-block size-2 rounded-full mr-1.5" :style="{ background: p.color }" />{{ p.label }}
            </td>
            <td class="py-1.5 text-right font-mono text-neutral-500">
              {{ p.w.toFixed(2) }} × {{ p.v.toFixed(2) }}
            </td>
            <td class="py-1.5 text-right font-mono font-semibold w-16">
              {{ (p.w * p.v).toFixed(3) }}
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="blocked" class="text-xs text-rose-600 dark:text-rose-400">
        Dilewati: ✗ ≥ 3 dan utilitas {{ pct(utility) }} &lt; 25% — butir yang terbukti merugikan tidak pernah dibawa ke prompt (kurator juga memensiunkannya bila utilitas &lt; 30%).
      </p>
      <p v-else class="text-xs text-neutral-500 leading-relaxed">
        Relevansi menentukan lebih dari separuh skor, jadi butir yang tidak nyambung tidak akan menang walau terbukti. Di antara butir yang sama-sama relevan, bukti ✓/✗ dan bobot yang memutuskan urutan. Butir yang lolos lalu disaring keberagamannya (satu klaster maksimal separuh hasil) dan diambil teratas (todo 15 · audit 8 · rapat 10 · spesialis 6).
      </p>
    </div>
  </div>
</template>
