<script setup lang="ts">
// Dampak pengetahuan di seluruh pengguna, dirangkum: membantu dan mengganggu di bidang apa (pie + persentase),
// serta pemakaian per jenis pekerjaan besar. Hanya angka teragregasi dari server; ringan (di-cache berlapis).
const { impact, loading, error, load } = useKnowledgeImpact()
onMounted(() => load())

const dim = ref<'domain' | 'kind'>('domain')
const dimItems = [{ label: 'Per area', value: 'domain' }, { label: 'Per jenis pekerjaan', value: 'kind' }]

const rows = computed(() => dim.value === 'domain'
  ? (impact.value?.by_domain || []).map(d => ({ label: d.domain, helpful: d.helpful, harmful: d.harmful }))
  : (impact.value?.by_kind || []).map(k => ({ label: k.label, helpful: k.helpful, harmful: k.harmful })))
const helpfulItems = computed(() => rows.value.map(r => ({ label: r.label, value: r.helpful })))
const harmfulItems = computed(() => rows.value.map(r => ({ label: r.label, value: r.harmful })))

const GOOD = ['#10b981', '#14b8a6', '#06b6d4', '#3b82f6', '#6366f1', '#8b5cf6', '#94a3b8']
const BAD = ['#f43f5e', '#f97316', '#f59e0b', '#ec4899', '#ef4444', '#a855f7', '#94a3b8']

const maxUses = computed(() => Math.max(1, ...(impact.value?.by_kind || []).map(k => k.uses)))
const scoreColor = (s: number) => s < 0 ? 'bg-neutral-300 dark:bg-neutral-600' : s >= 70 ? 'bg-emerald-500' : s >= 40 ? 'bg-amber-500' : 'bg-rose-500'
const scoreText = (s: number) => s < 0 ? 'belum dinilai' : `${s}% berguna`
</script>

<template>
  <section class="p-5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-5">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h2 class="text-sm font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-chart-pie" class="size-4 text-primary" />
          Dampak di komunitas
        </h2>
        <p class="text-xs text-neutral-500 mt-0.5 max-w-2xl">
          Ringkasan seluruh pengguna: pengetahuan ini membantu dan mengganggu di bidang apa. Hanya angka teragregasi,
          nama pekerjaan dan proyek pengguna tidak pernah dikirim.
        </p>
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <USelect v-model="dim" :items="dimItems" size="xs" class="w-40" />
        <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-refresh-cw" :loading="loading" aria-label="Muat ulang" @click="load(true)" />
      </div>
    </div>

    <p v-if="error" class="text-xs text-amber-600">
      {{ error }}
    </p>
    <USkeleton v-else-if="loading && !impact" class="h-40 w-full" />

    <template v-if="impact">
      <p v-if="!impact.by_kind.length" class="text-xs text-neutral-500">
        Belum ada data. Angka ditampilkan setelah dipakai oleh minimal {{ impact.min_devices }} perangkat berbeda.
      </p>
      <template v-else>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <div
            v-for="t in [
              { l: 'Total pemakaian', v: impact.total.uses, c: 'text-neutral-900 dark:text-white' },
              { l: 'Membantu', v: impact.total.helpful, c: 'text-emerald-600 dark:text-emerald-400' },
              { l: 'Mengganggu', v: impact.total.harmful, c: 'text-rose-600 dark:text-rose-400' },
              { l: 'Nilai keberhasilan', v: impact.score < 0 ? '-' : `${impact.score}%`, c: 'text-primary' }
            ]" :key="t.l" class="rounded-lg bg-neutral-50 dark:bg-neutral-800/60 p-3"
          >
            <p class="text-xs uppercase font-semibold text-neutral-500">
              {{ t.l }}
            </p>
            <p class="text-xl font-bold" :class="t.c">
              {{ t.v }}
            </p>
          </div>
        </div>

        <div class="grid gap-6 lg:grid-cols-2">
          <div class="space-y-2">
            <p class="text-xs font-semibold text-emerald-700 dark:text-emerald-400 flex items-center gap-1.5">
              <UIcon name="i-lucide-thumbs-up" class="size-3.5" /> Membantu di bidang
            </p>
            <KnowledgeDonut :items="helpfulItems" :palette="GOOD" empty-text="Belum ada penilaian membantu." />
          </div>
          <div class="space-y-2">
            <p class="text-xs font-semibold text-rose-700 dark:text-rose-400 flex items-center gap-1.5">
              <UIcon name="i-lucide-thumbs-down" class="size-3.5" /> Mengganggu di bidang
            </p>
            <KnowledgeDonut :items="harmfulItems" :palette="BAD" empty-text="Belum ada yang mengganggu. Bagus." />
          </div>
        </div>

        <div class="space-y-2">
          <p class="text-xs font-semibold uppercase text-neutral-500">
            Pemakaian per jenis pekerjaan
          </p>
          <div v-for="k in impact.by_kind" :key="k.kind" class="space-y-0.5">
            <div class="flex justify-between text-xs">
              <span class="font-medium text-neutral-700 dark:text-neutral-300">{{ k.label }}</span>
              <span class="text-neutral-500">{{ k.uses }}× · ✓{{ k.helpful }} ✗{{ k.harmful }} · {{ scoreText(k.score) }}</span>
            </div>
            <div class="h-2 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
              <div class="h-full rounded-full" :class="scoreColor(k.score)" :style="{ width: `${Math.max(3, k.uses / maxUses * 100)}%` }" />
            </div>
          </div>
        </div>
        <p class="text-xs text-neutral-400">
          Dari {{ impact.devices }} perangkat · diperbarui berkala.
        </p>
      </template>
    </template>
  </section>
</template>
