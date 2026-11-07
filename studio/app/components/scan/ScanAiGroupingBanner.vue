<script setup lang="ts">
defineProps<{
  loadingAi: boolean
  llmUsed: boolean
  aiSummary: string
  totalComponents: number
  allSelected: boolean
}>()

const emit = defineEmits<{
  regroup: []
  addGroup: []
  toggleSelectAll: []
}>()
</script>

<template>
  <div class="p-5 rounded-xl border border-sky-200 dark:border-sky-900/60 bg-sky-50/50 dark:bg-sky-950/20 space-y-3.5">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="flex items-center gap-2.5">
        <div class="w-8 h-8 rounded-lg bg-sky-500/10 dark:bg-sky-500/20 text-sky-600 dark:text-sky-400 flex items-center justify-center shrink-0">
          <UIcon name="i-lucide-sparkles" class="w-5 h-5" />
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h2 class="text-sm font-bold text-neutral-900 dark:text-white">
              Pengelompokan Project Cerdas (AI & Sinergi Stack)
            </h2>
            <UBadge
              size="xs"
              :color="llmUsed ? 'primary' : 'neutral'"
              variant="subtle"
            >
              {{ llmUsed ? 'LLM Engine Aktif' : 'Heuristic Rules' }}
            </UBadge>
          </div>
          <p class="text-xs text-neutral-600 dark:text-neutral-300 mt-0.5">
            Komponen wajib dikelompokkan ke dalam project sebelum masuk pemantauan. Periksa & tentukan nama project serta nama komponen secara manual.
          </p>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center gap-2 self-start sm:self-center shrink-0">
        <UButton
          icon="i-lucide-refresh-cw"
          color="neutral"
          variant="outline"
          size="xs"
          :loading="loadingAi"
          title="Jalankan ulang analisis pengelompokan AI"
          @click="emit('regroup')"
        >
          Kelompokkan Ulang AI
        </UButton>
        <UButton
          icon="i-lucide-folder-plus"
          color="neutral"
          variant="soft"
          size="xs"
          title="Buat wadah kelompok project baru secara manual"
          @click="emit('addGroup')"
        >
          + Project Baru
        </UButton>
      </div>
    </div>

    <!-- AI Narrative / Rationale -->
    <div v-if="aiSummary" class="p-3 rounded-lg bg-white/70 dark:bg-slate-900/60 border border-neutral-200/60 dark:border-white/10 text-xs text-neutral-700 dark:text-neutral-300 flex items-start gap-2">
      <UIcon name="i-lucide-bot" class="w-4 h-4 text-sky-500 shrink-0 mt-0.5" />
      <span class="leading-relaxed">{{ aiSummary }}</span>
    </div>

    <!-- Sub bar: Quick Toggle All -->
    <div class="flex items-center justify-between text-xs text-neutral-500 pt-1 border-t border-sky-100 dark:border-sky-900/40">
      <span>Total {{ totalComponents }} komponen terdeteksi pada direktori scan.</span>
      <button
        type="button"
        class="text-sky-600 dark:text-sky-400 hover:underline font-medium cursor-pointer"
        @click="emit('toggleSelectAll')"
      >
        {{ allSelected ? 'Batalkan Pilih Semua' : 'Pilih Semua Komponen' }}
      </button>
    </div>
  </div>
</template>
