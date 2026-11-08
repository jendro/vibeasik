<script setup lang="ts">
// Dampak pengetahuan di seluruh pengguna: berguna di jenis pekerjaan besar mana dan di area mana.
// Hanya angka teragregasi dari server (tanpa nama pekerjaan/proyek). Angka baru muncul bila dari >= beberapa perangkat.
const { impact, loading, error, load } = useKnowledgeImpact()
onMounted(() => load())

const max = computed(() => Math.max(1, ...(impact.value?.by_kind || []).map(k => k.uses)))
const maxDomain = computed(() => Math.max(1, ...(impact.value?.by_domain || []).map(d => d.uses)))
const scoreColor = (s: number) => s < 0 ? 'bg-neutral-300 dark:bg-neutral-600' : s >= 70 ? 'bg-emerald-500' : s >= 40 ? 'bg-amber-500' : 'bg-rose-500'
const scoreText = (s: number) => s < 0 ? 'belum dinilai' : `${s}% berguna`
</script>

<template>
  <section class="p-5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-4">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h2 class="text-sm font-semibold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-chart-no-axes-column" class="size-4 text-primary" />
          Dampak di komunitas
        </h2>
        <p class="text-xs text-neutral-500 mt-0.5">
          Pengetahuan ini sudah berjasa di berapa pekerjaan dan seberapa berguna, dirangkum per jenis pekerjaan besar.
          Hanya angka: nama pekerjaan dan proyek pengguna tidak pernah dikirim.
        </p>
      </div>
      <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-refresh-cw" :loading="loading" @click="load(true)" />
    </div>

    <p v-if="error" class="text-xs text-amber-600">
      {{ error }}
    </p>

    <template v-if="impact">
      <p v-if="!impact.by_kind.length" class="text-xs text-neutral-500">
        Belum ada data. Angka baru ditampilkan setelah dipakai oleh minimal {{ impact.min_devices }} perangkat berbeda.
      </p>
      <div v-else class="grid gap-6 lg:grid-cols-2">
        <div class="space-y-2">
          <p class="text-xs font-semibold uppercase text-neutral-500">
            Per jenis pekerjaan · {{ impact.total.uses }} pemakaian · {{ scoreText(impact.score) }}
          </p>
          <div v-for="k in impact.by_kind" :key="k.kind" class="space-y-0.5">
            <div class="flex justify-between text-xs">
              <span class="font-medium text-neutral-700 dark:text-neutral-300">{{ k.label }}</span>
              <span class="text-neutral-500">{{ k.uses }}× · ✓{{ k.helpful }} ✗{{ k.harmful }} · {{ scoreText(k.score) }}</span>
            </div>
            <div class="h-2 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
              <div class="h-full rounded-full" :class="scoreColor(k.score)" :style="{ width: `${Math.max(3, k.uses / max * 100)}%` }" />
            </div>
          </div>
        </div>
        <div class="space-y-2">
          <p class="text-xs font-semibold uppercase text-neutral-500">
            Berdampak pada area
          </p>
          <p v-if="!impact.by_domain.length" class="text-xs text-neutral-400">
            Belum ada.
          </p>
          <div v-for="d in impact.by_domain.slice(0, 8)" :key="d.domain" class="space-y-0.5">
            <div class="flex justify-between text-xs">
              <span class="font-medium text-neutral-700 dark:text-neutral-300">{{ d.domain }}</span>
              <span class="text-neutral-500">{{ d.uses }}× · {{ scoreText(d.score) }}</span>
            </div>
            <div class="h-2 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
              <div class="h-full rounded-full" :class="scoreColor(d.score)" :style="{ width: `${Math.max(3, d.uses / maxDomain * 100)}%` }" />
            </div>
          </div>
        </div>
      </div>
      <p class="text-xs text-neutral-400">
        Dari {{ impact.devices }} perangkat.
      </p>
    </template>
  </section>
</template>
