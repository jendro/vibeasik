<script setup lang="ts">
// Penjelasan singkat cara vibeasik menghitung tick, supaya rumus di tabel bisa diperiksa dengan tangan.
const RULES = [
  { code: '1 tick', text: 'satu pernyataan yang dijalankan: assign, panggilan, return, cek syarat, i++' },
  { code: 'if / switch', text: '1 tick cek syarat + cabang termahal (kasus terburuk)' },
  { code: 'loop atas data', text: '(n + 1) cek syarat + n × isi loop — inilah yang menaikkan pangkat n' },
  { code: 'loop batas tetap', text: 'for i < 3 hanya mengalikan angka, tidak menaikkan pangkat' },
  { code: 'fungsi lokal', text: 'rumus fungsi yang dipanggil ikut dijumlahkan; di dalam loop dikalikan n' },
  { code: 'pustaka luar', text: '1 tick; sort = n log n; callback Map/Filter/Walk = n × isi callback' },
  { code: 'query database', text: 'dihitung terpisah sebagai Q(n); query di dalam loop → Q(n) = n + 1' }
]
const EXAMPLES = [
  { formula: '3', bigO: 'O(1)', text: 'ambil satu baris lalu kirim' },
  { formula: '2n + 5', bigO: 'O(n)', text: 'satu loop atas daftar' },
  { formula: '4n + 9', bigO: 'O(n)', text: 'dua loop berurutan (2n → 4n), tetap linear' },
  { formula: '3n² + 2n + 5', bigO: 'O(n²)', text: 'loop di dalam loop' },
  { formula: 'Q(n) = n + 1', bigO: 'N+1', text: '1 query daftar + 1 query per baris' }
]
</script>

<template>
  <section class="rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] p-5 space-y-4">
    <div class="space-y-1">
      <span class="font-mono text-xs uppercase tracking-wider text-sky-600 dark:text-sky-400">Cara menghitung</span>
      <p class="text-sm text-neutral-700 dark:text-neutral-200 max-w-3xl leading-relaxed">
        <strong class="font-semibold">T(n)</strong> = jumlah langkah (tick) yang dijalankan sebuah rute bila jumlah datanya <strong class="font-semibold">n</strong>.
        T(1) adalah biaya untuk 1 data; suku tertinggi rumusnya menentukan kelas pertumbuhan (Big-O).
        Hitungan dibaca langsung dari kode (AST Go), bukan dari pengukuran waktu.
      </p>
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-4">
      <dl class="lg:col-span-7 space-y-1.5">
        <div v-for="r in RULES" :key="r.code" class="flex gap-3 text-sm">
          <dt class="w-32 shrink-0 font-mono text-xs pt-0.5 text-neutral-900 dark:text-white">
            {{ r.code }}
          </dt>
          <dd class="text-neutral-600 dark:text-neutral-300">
            {{ r.text }}
          </dd>
        </div>
      </dl>
      <ul class="lg:col-span-5 rounded-xl bg-neutral-50 dark:bg-white/[0.03] border border-neutral-200/70 dark:border-white/[0.06] p-3 space-y-1.5">
        <li v-for="e in EXAMPLES" :key="e.formula" class="flex items-baseline gap-2 text-sm">
          <code class="font-mono text-xs text-neutral-900 dark:text-white w-28 shrink-0">{{ e.formula }}</code>
          <span class="font-mono text-xs text-sky-700 dark:text-sky-400 w-12 shrink-0">{{ e.bigO }}</span>
          <span class="text-neutral-600 dark:text-neutral-300">{{ e.text }}</span>
        </li>
      </ul>
    </div>
    <p class="text-xs text-neutral-500">
      Batasan: semua koleksi dianggap berukuran n; rekursi ditandai dan rumusnya menjadi batas bawah; biaya di dalam pustaka luar dan mesin database tidak ikut dihitung.
    </p>
  </section>
</template>
