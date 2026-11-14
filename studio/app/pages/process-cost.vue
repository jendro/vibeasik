<script setup lang="ts">
/**
 * Biaya Proses: rumus tick T(n), Big-O, dan jumlah query tiap rute/fungsi backend proyek aktif.
 * Halaman sendiri (bukan tab Detail Proyek) agar navigasinya cukup lewat sidebar.
 */
definePageMeta({ fluid: true })

const { activeProjectId, activeProject, fetchProjects } = useActiveProject()

onMounted(fetchProjects)
</script>

<template>
  <div class="w-full max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 pt-6 pb-16 space-y-6">
    <AppPageHeader
      title="Biaya Proses"
      description="Performa kode tiap rute dan fungsi backend: rumus tick T(n), kelas pertumbuhan Big-O, dan query di dalam perulangan (N+1). Rute yang berat bisa langsung dijadikan todo."
    />

    <UAlert
      v-if="!activeProjectId"
      color="neutral"
      variant="subtle"
      icon="i-lucide-folder-search"
      title="Pilih proyek aktif"
      description="Pilih proyek lewat pemilih proyek di navbar atas untuk menghitung biaya prosesnya."
    />

    <div v-else-if="!activeProject" class="h-72 rounded-2xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] animate-pulse" />

    <InsightProcessCostTab v-else :project="activeProject" />
  </div>
</template>
