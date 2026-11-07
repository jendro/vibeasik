<script setup lang="ts">
import type { ReorgPlan } from '~/types'

definePageMeta({
  title: 'Workspace Folder Organizer'
})

const { generateReorgPlan, executeReorgMoves, getWorkspaceSettings } = useApi()
const toast = useToast()

const targetRoot = ref('')
const strategy = ref('product_centric')
const loading = ref(false)
const executing = ref(false)
const plan = ref<ReorgPlan | null>(null)
const selectedMoveIds = ref<Set<string>>(new Set())
const updateDb = ref(true)
const confirmModalOpen = ref(false)
const scriptModalOpen = ref(false)

const loadPlan = async () => {
  loading.value = true
  try {
    const ws = await getWorkspaceSettings().catch(() => null)
    if (ws?.default_projects_root) {
      targetRoot.value = ws.default_projects_root
    }
    const res = await generateReorgPlan({
      target_directory: targetRoot.value,
      strategy: strategy.value
    })
    plan.value = res
    // Select all non-conflicting moves by default
    selectedMoveIds.value = new Set(
      res.planned_moves
        .filter(m => m.status !== 'conflict')
        .map(m => m.id)
    )
    toast.add({
      title: 'Rencana Penataan Dibuat',
      description: `Ditemukan ${res.total_items} folder yang siap ditata ke ${res.target_directory}`,
      icon: 'i-lucide-folder-check',
      color: 'success'
    })
  } catch (err: any) {
    toast.add({
      title: 'Gagal Membuat Rencana',
      description: err.message || 'Terjadi kesalahan saat memindai folder',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadPlan()
})

const { formatBytes } = useFormatters()

const toggleSelectAll = () => {
  if (!plan.value) return
  if (selectedMoveIds.value.size === validMoves.value.length) {
    selectedMoveIds.value.clear()
  } else {
    selectedMoveIds.value = new Set(validMoves.value.map(m => m.id))
  }
}

const toggleItem = (id: string) => {
  if (selectedMoveIds.value.has(id)) {
    selectedMoveIds.value.delete(id)
  } else {
    selectedMoveIds.value.add(id)
  }
}

const validMoves = computed(() => {
  return plan.value?.planned_moves.filter(m => m.status !== 'conflict') || []
})

const selectedMovesList = computed(() => {
  return plan.value?.planned_moves.filter(m => selectedMoveIds.value.has(m.id)) || []
})

const handleExecuteMoves = async () => {
  if (selectedMovesList.value.length === 0) return
  confirmModalOpen.value = false
  executing.value = true

  try {
    const res = await executeReorgMoves({
      moves: selectedMovesList.value,
      update_db: updateDb.value
    })

    toast.add({
      title: 'Eksekusi Pemindahan Selesai',
      description: res.message,
      icon: 'i-lucide-check-circle',
      color: res.failed_count === 0 ? 'success' : 'warning'
    })

    // Refresh plan to reflect new filesystem state
    await loadPlan()
  } catch (err: any) {
    toast.add({
      title: 'Eksekusi Gagal',
      description: err.message || 'Terjadi kesalahan saat memindahkan folder',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    executing.value = false
  }
}

const copyScript = async () => {
  if (!plan.value?.generated_script) return
  try {
    await navigator.clipboard.writeText(plan.value.generated_script)
    toast.add({
      title: 'Script Disalin',
      description: 'Script bash siap dijalankan di terminal Anda.',
      icon: 'i-lucide-clipboard-check',
      color: 'success'
    })
  } catch {
    toast.add({
      title: 'Gagal Menyalin',
      description: 'Silakan salin teks manual dari modal.',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  }
}

const downloadScript = () => {
  if (!plan.value?.generated_script) return
  const blob = new Blob([plan.value.generated_script], { type: 'text/x-shellscript' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'organize_projects.sh'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)

  toast.add({
    title: 'Script Diunduh',
    description: 'File organize_projects.sh berhasil diunduh.',
    icon: 'i-lucide-download',
    color: 'success'
  })
}
</script>

<template>
  <div class="space-y-8">
    <!-- Header Section -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-2 border-b border-neutral-200 dark:border-neutral-800">
      <div>
        <div class="flex items-center gap-2">
          <NuxtLink to="/cockpit" class="text-xs text-sky-600 dark:text-sky-400 hover:underline flex items-center gap-1">
            <UIcon name="i-lucide-arrow-left" class="w-3.5 h-3.5" />
            <span>Kembali ke Cockpit</span>
          </NuxtLink>
        </div>
        <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white flex items-center gap-2.5 mt-1">
          <UIcon name="i-lucide-folder-tree" class="w-7 h-7 text-sky-500" />
          <span>Workspace Folder Organizer</span>
          <UBadge color="primary" variant="subtle" size="sm">
            Product-Centric
          </UBadge>
        </h1>
        <p class="text-sm text-neutral-500 dark:text-neutral-400 mt-1">
          Menata ulang folder repositori proyek dari yang awalnya terpisah berdasar bahasa (<code class="text-xs font-mono">go/</code>, <code class="text-xs font-mono">Nuxt/</code>, <code class="text-xs font-mono">Flutter/</code>, <code class="text-xs font-mono">PHP/</code>) menjadi satu payung folder produk yang rapi.
        </p>
      </div>

      <div class="flex items-center gap-2">
        <UButton
          icon="i-lucide-refresh-cw"
          color="neutral"
          variant="outline"
          size="sm"
          :loading="loading"
          @click="loadPlan"
        >
          Pindai Ulang
        </UButton>
      </div>
    </div>

    <!-- Strategy & Target Configuration Card -->
    <div class="p-5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 shadow-2xs space-y-4">
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <!-- Target Directory -->
        <div class="md:col-span-2 space-y-1.5">
          <label class="block text-xs font-semibold text-neutral-700 dark:text-neutral-300">
            Lokasi Folder Induk Tujuan (Target Root)
          </label>
          <div class="flex items-center gap-2">
            <UInput
              v-model="targetRoot"
              placeholder="/home/username/Documents/Projects"
              icon="i-lucide-folder"
              class="w-full font-mono text-xs"
            />
            <UButton
              color="primary"
              variant="solid"
              size="sm"
              icon="i-lucide-sparkles"
              :loading="loading"
              @click="loadPlan"
            >
              Terapkan
            </UButton>
          </div>
          <p class="text-[11px] text-neutral-400">
            Semua komponen proyek (Backend, Frontend, Mobile) akan dikelompokkan ke dalam subfolder produk di folder ini.
          </p>
        </div>

        <!-- Strategy Option -->
        <div class="space-y-1.5">
          <label class="block text-xs font-semibold text-neutral-700 dark:text-neutral-300">
            Strategi Pengelompokan
          </label>
          <div class="p-2.5 rounded-lg border border-sky-500/30 bg-sky-50/50 dark:bg-sky-950/20 text-xs text-sky-900 dark:text-sky-200 font-medium flex items-center gap-2">
            <UIcon name="i-lucide-box" class="w-4 h-4 text-sky-500 shrink-0" />
            <span>Product-Centric (Satu Produk, Satu Folder)</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Before / After Architecture Comparison Banner -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs font-mono">
      <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-900/50 space-y-2">
        <div class="font-bold text-neutral-700 dark:text-neutral-300 flex items-center gap-1.5 font-sans">
          <span class="w-2 h-2 rounded-full bg-rose-500" />
          <span>Struktur Saat Ini (Terpencar Berdasarkan Bahasa):</span>
        </div>
        <div class="text-[11px] text-neutral-500 space-y-1 pl-3.5 border-l border-neutral-300 dark:border-neutral-700">
          <div>Documents/go/bejamBE</div>
          <div>Documents/Nuxt/bejamERP2</div>
          <div>Documents/Flutter/bejamMobile</div>
          <div>Documents/go/dongengKu</div>
          <div>Documents/PHP/dongengKu</div>
        </div>
      </div>

      <div class="p-3.5 rounded-xl border border-sky-500/30 bg-sky-50/30 dark:bg-sky-950/20 space-y-2">
        <div class="font-bold text-sky-900 dark:text-sky-200 flex items-center gap-1.5 font-sans">
          <span class="w-2 h-2 rounded-full bg-emerald-500" />
          <span>Target Reorganisasi (Product-Centric Rapi):</span>
        </div>
        <div class="text-[11px] text-neutral-600 dark:text-neutral-300 space-y-1 pl-3.5 border-l border-sky-400">
          <div>Projects/bejam/backend/</div>
          <div>Projects/bejam/frontend/</div>
          <div>Projects/bejam/mobile/</div>
          <div>Projects/dongengKu/backend-go/</div>
          <div>Projects/dongengKu/backend-php/</div>
        </div>
      </div>
    </div>

    <!-- Action Bar for Plan -->
    <div v-if="plan" class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
      <div class="flex items-center gap-3">
        <div class="text-xs font-medium text-neutral-600 dark:text-neutral-300">
          Terpilih: <strong class="text-neutral-900 dark:text-white">{{ selectedMoveIds.size }}</strong> dari {{ plan.total_items }} folder
          ({{ formatBytes(plan.total_size) }})
        </div>
        <UCheckbox
          v-model="updateDb"
          label="Otomatis update path di database vibeasik"
          class="text-xs"
        />
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <UButton
          icon="i-lucide-code"
          color="neutral"
          variant="outline"
          size="xs"
          @click="scriptModalOpen = true"
        >
          Lihat Script Bash
        </UButton>
        <UButton
          icon="i-lucide-download"
          color="neutral"
          variant="outline"
          size="xs"
          @click="downloadScript"
        >
          Unduh .sh
        </UButton>
        <UButton
          icon="i-lucide-play"
          color="primary"
          variant="solid"
          size="xs"
          :disabled="selectedMoveIds.size === 0 || executing"
          :loading="executing"
          @click="confirmModalOpen = true"
        >
          Eksekusi Pemindahan ({{ selectedMoveIds.size }})
        </UButton>
      </div>
    </div>

    <!-- Planned Moves List Table -->
    <div v-if="loading" class="py-16 text-center space-y-3">
      <UIcon name="i-lucide-loader-2" class="w-8 h-8 mx-auto text-sky-500 animate-spin" />
      <p class="text-xs text-neutral-500 font-mono">
        Memindai komponen proyek dan menghitung ukuran direktori...
      </p>
    </div>

    <div v-else-if="plan && plan.planned_moves.length === 0" class="text-center py-16 border border-dashed border-neutral-300 dark:border-neutral-800 rounded-xl bg-white dark:bg-neutral-900/40">
      <UIcon name="i-lucide-check-circle-2" class="w-8 h-8 mx-auto text-emerald-500 mb-2" />
      <h3 class="text-sm font-semibold text-neutral-800 dark:text-neutral-200">
        Semua folder sudah terorganisir di target root!
      </h3>
      <p class="text-xs text-neutral-500 mt-1">
        Tidak ada pemindahan folder yang tertunda.
      </p>
    </div>

    <div v-else-if="plan" class="border border-neutral-200 dark:border-neutral-800 rounded-xl overflow-hidden bg-white dark:bg-neutral-900 shadow-2xs">
      <div class="p-3 bg-neutral-50 dark:bg-neutral-800/60 border-b border-neutral-200 dark:border-neutral-800 flex items-center justify-between text-xs font-semibold">
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="text-xs text-sky-600 dark:text-sky-400 hover:underline cursor-pointer"
            @click="toggleSelectAll"
          >
            {{ selectedMoveIds.size === validMoves.length ? 'Batal Pilih Semua' : 'Pilih Semua yang Aman' }}
          </button>
        </div>
        <div class="text-neutral-500 font-normal">
          {{ plan.planned_moves.length }} Rencana Pemindahan
        </div>
      </div>

      <div class="divide-y divide-neutral-100 dark:divide-neutral-800 text-xs">
        <div
          v-for="item in plan.planned_moves"
          :key="item.id"
          class="p-4 flex flex-col md:flex-row md:items-center justify-between gap-4 transition-colors"
          :class="[
            item.status === 'conflict' ? 'bg-rose-50/30 dark:bg-rose-950/10' : '',
            item.status === 'success' ? 'bg-emerald-50/30 dark:bg-emerald-950/10' : '',
            selectedMoveIds.has(item.id) ? 'bg-sky-50/20 dark:bg-sky-950/10' : ''
          ]"
        >
          <!-- Checkbox & Paths -->
          <div class="flex items-start gap-3 flex-1 min-w-0">
            <input
              type="checkbox"
              :checked="selectedMoveIds.has(item.id)"
              :disabled="item.status === 'conflict' || item.status === 'success'"
              class="mt-1 rounded text-sky-600 focus:ring-sky-500 cursor-pointer"
              @change="toggleItem(item.id)"
            >

            <div class="space-y-1.5 flex-1 min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-bold text-neutral-900 dark:text-white">{{ item.project_name }}</span>
                <UBadge v-if="item.role" size="xs" color="info" variant="subtle">
                  {{ item.role }}
                </UBadge>
                <span class="text-[11px] font-mono text-neutral-400">({{ formatBytes(item.size_bytes) }})</span>
                <UBadge v-if="item.is_git_repo" size="xs" color="neutral" variant="outline">
                  Git Repo
                </UBadge>
                <UBadge v-if="item.has_uncommitted_changes" size="xs" color="warning" variant="subtle">
                  Uncommitted
                </UBadge>
                <UBadge v-if="item.status === 'success'" size="xs" color="success" variant="solid">
                  Sudah Dipindah
                </UBadge>
                <UBadge v-if="item.status === 'conflict'" size="xs" color="error" variant="subtle">
                  Konflik
                </UBadge>
              </div>

              <!-- From / To display -->
              <div class="grid grid-cols-1 lg:grid-cols-2 gap-2 text-[11px] font-mono">
                <div class="p-1.5 rounded bg-neutral-100/70 dark:bg-neutral-800/70 text-neutral-600 dark:text-neutral-400 truncate" :title="item.current_path">
                  <span class="text-neutral-400 font-sans mr-1">Dari:</span>{{ item.current_path }}
                </div>
                <div class="p-1.5 rounded bg-sky-50 dark:bg-sky-950/40 text-sky-700 dark:text-sky-300 truncate" :title="item.target_path">
                  <span class="text-sky-500 font-sans mr-1">Ke:</span>{{ item.target_path }}
                </div>
              </div>

              <!-- Warning or Error -->
              <div v-if="item.warning" class="text-[11px] text-amber-600 dark:text-amber-400 flex items-center gap-1 font-sans">
                <UIcon name="i-lucide-alert-triangle" class="w-3.5 h-3.5 shrink-0" />
                <span>{{ item.warning }}</span>
              </div>
              <div v-if="item.error" class="text-[11px] text-rose-600 dark:text-rose-400 flex items-center gap-1 font-sans">
                <UIcon name="i-lucide-x-circle" class="w-3.5 h-3.5 shrink-0" />
                <span>{{ item.error }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Confirm Modal -->
    <UModal v-model:open="confirmModalOpen" title="Konfirmasi Pemindahan Folder">
      <template #content>
        <div class="p-6 space-y-4">
          <div class="p-3.5 rounded-xl border border-sky-500/30 bg-sky-50/50 dark:bg-sky-950/20 text-xs text-neutral-700 dark:text-neutral-300 space-y-1.5">
            <div class="font-bold text-sky-900 dark:text-sky-300 flex items-center gap-1.5">
              <UIcon name="i-lucide-shield-alert" class="w-4 h-4 text-sky-500" />
              <span>Keamanan Pemindahan Folder</span>
            </div>
            <p>
              Anda akan memindahkan <strong class="text-neutral-900 dark:text-white">{{ selectedMoveIds.size }} folder</strong> ke lokasi baru. Operasi ini menggunakan fungsi rename sistem operasi yang aman.
            </p>
            <p v-if="updateDb">
              Path absolut proyek di database vibeasik akan langsung diperbarui ke lokasi baru secara otomatis.
            </p>
          </div>

          <div class="flex justify-end gap-2 pt-2 border-t border-neutral-100 dark:border-neutral-800">
            <UButton color="neutral" variant="outline" size="sm" @click="confirmModalOpen = false">
              Batal
            </UButton>
            <UButton
              color="primary"
              variant="solid"
              size="sm"
              icon="i-lucide-play"
              :loading="executing"
              @click="handleExecuteMoves"
            >
              Ya, Pindahkan Sekarang
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Script Preview Modal -->
    <UModal v-model:open="scriptModalOpen" title="Script Bash Reorganisasi" class="max-w-3xl">
      <template #content>
        <div class="p-6 space-y-4">
          <div class="flex items-center justify-between text-xs text-neutral-500">
            <span>Anda dapat menyalin script ini dan menjalankannya secara mandiri di terminal:</span>
            <div class="flex items-center gap-2">
              <UButton size="xs" variant="outline" icon="i-lucide-copy" @click="copyScript">
                Salin Script
              </UButton>
              <UButton size="xs" variant="solid" color="primary" icon="i-lucide-download" @click="downloadScript">
                Unduh .sh
              </UButton>
            </div>
          </div>

          <pre class="p-4 rounded-xl bg-slate-950 text-slate-200 font-mono text-xs overflow-x-auto max-h-[50vh] border border-white/10 leading-relaxed">{{ plan?.generated_script }}</pre>

          <div class="flex justify-end pt-2 border-t border-neutral-100 dark:border-neutral-800">
            <UButton color="neutral" variant="outline" size="sm" @click="scriptModalOpen = false">
              Tutup
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
