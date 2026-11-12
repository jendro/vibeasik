<script setup lang="ts">
// Bar paging baku Studio untuk daftar panjang (pasangan useListPaging).
// Tampil hanya bila daftar melebihi ukuran halaman terkecil; menggulir daftar ke atas saat halaman berganti.
const props = withDefaults(defineProps<{
  total: number
  perPageOptions: number[]
  rangeStart: number
  rangeEnd: number
  /** Kata benda isi daftar, mis. "halaman", "alur", "rute". */
  noun?: string
  /** id elemen daftar yang digulir ke atas saat halaman berganti. */
  scrollTarget?: string
  /** Letak bar: atas (ringkas) atau bawah (dengan pemilih per halaman). */
  position?: 'top' | 'bottom'
}>(), { noun: 'baris', scrollTarget: '', position: 'bottom' })

const page = defineModel<number>('page', { required: true })
const perPage = defineModel<number>('perPage', { required: true })

const uid = useId()
const enabled = computed(() => props.total > (props.perPageOptions[0] ?? 0))

const setPage = (p: number) => {
  page.value = p
  if (!import.meta.client || !props.scrollTarget) return
  nextTick(() => document.getElementById(props.scrollTarget)?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}
</script>

<template>
  <div
    v-if="enabled"
    class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs text-neutral-500 dark:text-slate-400"
    :class="position === 'bottom' ? 'pt-3 border-t border-neutral-200 dark:border-white/[0.07]' : ''"
  >
    <span class="tabular-nums">
      Menampilkan
      <strong class="font-semibold text-neutral-900 dark:text-white">{{ rangeStart }}–{{ rangeEnd }}</strong>
      dari
      <strong class="font-semibold text-neutral-900 dark:text-white">{{ total }}</strong>
      {{ noun }}
    </span>

    <div class="flex items-center gap-2.5 flex-wrap self-end sm:self-auto">
      <label v-if="position === 'bottom' && perPageOptions.length > 1" class="flex items-center gap-1.5" :for="`${uid}-per-page`">
        <span class="font-medium">Per hal</span>
        <select
          :id="`${uid}-per-page`"
          v-model.number="perPage"
          class="h-8 pl-2 pr-6 text-xs tabular-nums rounded-lg border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] text-neutral-800 dark:text-slate-200 focus:outline-none focus:ring-1 focus:ring-sky-500 cursor-pointer"
          title="Jumlah item per halaman"
        >
          <option v-for="opt in perPageOptions" :key="opt" :value="opt">
            {{ opt }}
          </option>
        </select>
      </label>
      <UPagination
        v-if="total > perPage"
        :page="page"
        :total="total"
        :items-per-page="perPage"
        size="sm"
        @update:page="setPage"
      />
    </div>
  </div>
</template>
