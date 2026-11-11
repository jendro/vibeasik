<script setup lang="ts">
import type { SimilarityScanResult } from '~/types'

const open = defineModel<boolean>('open', { default: false })
defineEmits<{
  deprecateRequest: [projectId: number]
  refreshProjects: []
}>()

const router = useRouter()
const { scanSimilarities } = useApi()
const toast = useToast()

const loading = ref(false)
const scanResult = ref<SimilarityScanResult | null>(null)
const activeTab = ref<'matches' | 'clusters'>('matches')

const runScan = async () => {
  loading.value = true
  try {
    scanResult.value = await scanSimilarities()
    if (scanResult.value.matches_found === 0) {
      toast.add({
        title: 'Pemindaian Selesai',
        description: 'Tidak ada proyek dengan folder identik atau kemiripan mencurigakan.',
        icon: 'i-lucide-check-circle',
        color: 'success'
      })
    }
  } catch (err: any) {
    toast.add({
      title: 'Pemindaian Gagal',
      description: err.message || 'Gagal menjalankan similarity scanner',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

watch(open, (isOpen) => {
  if (isOpen && !scanResult.value) {
    runScan()
  }
})

const getScoreColor = (score: number) => {
  if (score >= 80) return 'text-rose-600 dark:text-rose-400 bg-rose-50 dark:bg-rose-950/40 border-rose-500/30'
  if (score >= 65) return 'text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-950/40 border-amber-500/30'
  return 'text-sky-600 dark:text-sky-400 bg-sky-50 dark:bg-sky-950/40 border-sky-500/30'
}

const getSuggestionLabel = (s: string) => {
  switch (s) {
    case 'duplicate_cleanup': return 'Folder Duplikat'
    case 'merge_bundle': return 'Rekomendasi Gabung Bundle'
    case 'mark_deprecated': return 'Kandidat Deprecated'
    default: return 'Perlu Ditinjau'
  }
}

const goToOrganizer = () => {
  open.value = false
  router.push('/organizer')
}
</script>

<template>
  <UModal v-model:open="open" title="Pemindai Duplikat & Kemiripan Proyek (Similarity Scanner)" class="max-w-4xl">
    <template #content>
      <div class="p-6 space-y-6 max-h-[80vh] overflow-y-auto">
        <!-- Header Info & Actions -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-4 border-b border-neutral-100 dark:border-white/[0.08]">
          <div>
            <h3 class="text-base font-bold text-neutral-900 dark:text-white flex items-center gap-2">
              <UIcon name="i-lucide-copy" class="w-5 h-5 text-sky-500" />
              <span>Analisis Duplikasi & Hubungan Antar Proyek</span>
            </h3>
            <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5">
              Mendeteksi apakah ada folder yang sama terdaftar dua kali, atau proyek BE/FE/Mobile yang seharusnya bersatu dalam satu bundle.
            </p>
          </div>

          <div class="flex items-center gap-2 shrink-0">
            <UButton
              icon="i-lucide-refresh-cw"
              color="neutral"
              variant="outline"
              size="xs"
              :loading="loading"
              @click="runScan"
            >
              Scan Ulang
            </UButton>
            <UButton
              icon="i-lucide-folder-tree"
              color="primary"
              variant="solid"
              size="xs"
              @click="goToOrganizer"
            >
              Buka Tata Folder
            </UButton>
          </div>
        </div>

        <!-- Metrics Overview -->
        <div v-if="scanResult" class="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-neutral-50/50 dark:bg-[#04111f]">
            <div class="text-xs text-neutral-500">
              Total Proyek Diteliti
            </div>
            <div class="text-xl font-bold font-mono text-neutral-900 dark:text-white mt-1">
              {{ scanResult.total_projects }}
            </div>
          </div>
          <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-neutral-50/50 dark:bg-[#04111f]">
            <div class="text-xs text-neutral-500">
              Pasangan Kemiripan Terdeteksi
            </div>
            <div class="text-xl font-bold font-mono text-neutral-900 dark:text-white mt-1 flex items-center gap-2">
              <span>{{ scanResult.matches_found }}</span>
              <UBadge v-if="scanResult.high_matches > 0" color="error" variant="subtle" size="xs">
                {{ scanResult.high_matches }} Kemiripan Tinggi
              </UBadge>
            </div>
          </div>
          <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-neutral-50/50 dark:bg-[#04111f]">
            <div class="text-xs text-neutral-500">
              Klaster Proyek Bersaudara
            </div>
            <div class="text-xl font-bold font-mono text-neutral-900 dark:text-white mt-1">
              {{ scanResult.suggested_groups?.length || 0 }}
            </div>
          </div>
        </div>

        <!-- Tabs Navigation -->
        <div class="flex items-center gap-2 border-b border-neutral-200 dark:border-white/[0.08] text-xs">
          <button
            type="button"
            class="pb-2 font-semibold transition-colors flex items-center gap-1.5"
            :class="activeTab === 'matches'
              ? 'text-sky-600 dark:text-sky-400 border-b-2 border-sky-500'
              : 'text-neutral-500 hover:text-neutral-900 dark:hover:text-white'"
            @click="activeTab = 'matches'"
          >
            <UIcon name="i-lucide-git-compare" class="w-4 h-4" />
            <span>Pasangan Terdeteksi ({{ scanResult?.matches?.length || 0 }})</span>
          </button>
          <button
            type="button"
            class="pb-2 font-semibold transition-colors flex items-center gap-1.5"
            :class="activeTab === 'clusters'
              ? 'text-sky-600 dark:text-sky-400 border-b-2 border-sky-500'
              : 'text-neutral-500 hover:text-neutral-900 dark:hover:text-white'"
            @click="activeTab === 'clusters'"
          >
            <UIcon name="i-lucide-layers" class="w-4 h-4" />
            <span>Klaster Produk ({{ scanResult?.suggested_groups?.length || 0 }})</span>
          </button>
        </div>

        <!-- Loading State -->
        <div v-if="loading" class="py-12 text-center space-y-3">
          <UIcon name="i-lucide-loader-2" class="w-8 h-8 mx-auto text-sky-500 animate-spin" />
          <p class="text-xs text-neutral-500 font-mono">
            Membandingkan nama, struktur folder, dan peran komponen...
          </p>
        </div>

        <!-- Tab 1: Matches List -->
        <div v-else-if="activeTab === 'matches'" class="space-y-4">
          <div v-if="!scanResult?.matches?.length" class="text-center py-12 border border-dashed border-neutral-300 dark:border-white/[0.08] rounded-xl">
            <UIcon name="i-lucide-check-circle-2" class="w-8 h-8 mx-auto text-emerald-500 mb-2" />
            <p class="text-sm font-semibold text-neutral-800 dark:text-neutral-200">
              Tidak ada duplikat atau bentrok folder terdeteksi
            </p>
            <p class="text-xs text-neutral-500 mt-1">
              Seluruh proyek memiliki folder dan lingkup yang terpisah rapi.
            </p>
          </div>

          <div
            v-for="(match, idx) in (scanResult?.matches || [])"
            :key="idx"
            class="p-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-neutral-900/60 space-y-3.5"
          >
            <!-- Card Header: Score & Badges -->
            <div class="flex items-start justify-between gap-3">
              <div class="flex flex-wrap items-center gap-2">
                <span
                  class="px-2.5 py-0.5 rounded-full text-xs font-mono font-bold border"
                  :class="getScoreColor(match.overall_score)"
                >
                  Skor Kemiripan: {{ match.overall_score }}%
                </span>
                <UBadge color="neutral" variant="subtle" size="xs">
                  {{ getSuggestionLabel(match.suggestion) }}
                </UBadge>
                <UBadge v-if="match.exact_same_path" color="error" variant="solid" size="xs">
                  ⚠️ Folder Identik
                </UBadge>
                <UBadge v-if="match.component_role_complementary" color="info" variant="subtle" size="xs">
                  🧩 BE + FE / Mobile
                </UBadge>
              </div>

              <div class="flex items-center gap-1.5 shrink-0">
                <UButton
                  color="warning"
                  variant="soft"
                  size="xs"
                  icon="i-lucide-archive"
                  title="Tandai salah satu proyek sebagai deprecated"
                  @click="$emit('deprecateRequest', match.project_b.id)"
                >
                  Deprecate B
                </UButton>
              </div>
            </div>

            <!-- Comparison Columns -->
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
              <!-- Project A -->
              <div class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-200/60 dark:border-neutral-700/50 space-y-1.5">
                <div class="flex items-center justify-between">
                  <span class="font-bold text-neutral-900 dark:text-white">{{ match.project_a.name }}</span>
                  <UBadge size="xs" color="neutral" variant="outline">
                    {{ match.project_a.project_type }}
                  </UBadge>
                </div>
                <div class="text-xs font-mono text-neutral-500 truncate" :title="match.project_a.root_path">
                  📁 {{ match.project_a.root_path || '-' }}
                </div>
                <div class="text-xs text-neutral-600 dark:text-neutral-400">
                  Komponen: {{ match.project_a.components.join(', ') || 'Standalone' }}
                </div>
              </div>

              <!-- Project B -->
              <div class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-200/60 dark:border-neutral-700/50 space-y-1.5">
                <div class="flex items-center justify-between">
                  <span class="font-bold text-neutral-900 dark:text-white">{{ match.project_b.name }}</span>
                  <UBadge size="xs" color="neutral" variant="outline">
                    {{ match.project_b.project_type }}
                  </UBadge>
                </div>
                <div class="text-xs font-mono text-neutral-500 truncate" :title="match.project_b.root_path">
                  📁 {{ match.project_b.root_path || '-' }}
                </div>
                <div class="text-xs text-neutral-600 dark:text-neutral-400">
                  Komponen: {{ match.project_b.components.join(', ') || 'Standalone' }}
                </div>
              </div>
            </div>

            <!-- Analysis Reasons -->
            <div class="p-2.5 rounded-lg bg-sky-50/50 dark:bg-sky-950/20 border border-sky-500/20 text-xs text-neutral-700 dark:text-neutral-300 space-y-1">
              <div class="font-semibold text-sky-800 dark:text-sky-300 text-xs flex items-center gap-1.5">
                <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 text-sky-500" />
                <span>Analisis Temuan:</span>
              </div>
              <ul class="list-disc list-inside space-y-0.5 text-xs">
                <li v-for="(reason, rIdx) in match.reasons" :key="rIdx">
                  {{ reason }}
                </li>
              </ul>
            </div>
          </div>
        </div>

        <!-- Tab 2: Clusters -->
        <div v-else class="space-y-4">
          <div v-if="!scanResult?.suggested_groups?.length" class="text-center py-12 border border-dashed border-neutral-300 dark:border-white/[0.08] rounded-xl">
            <p class="text-sm font-semibold text-neutral-800 dark:text-neutral-200">
              Tidak ada klaster aplikasi terpecah ditemukan
            </p>
          </div>

          <div
            v-for="(cluster, cIdx) in (scanResult?.suggested_groups || [])"
            :key="cIdx"
            class="p-4 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-neutral-900/60 space-y-3"
          >
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-box" class="w-5 h-5 text-sky-500" />
                <span class="font-bold text-sm text-neutral-900 dark:text-white">{{ cluster.suggested_name }}</span>
                <UBadge color="primary" variant="subtle" size="xs">
                  {{ cluster.projects.length }} Bagian Terpisah
                </UBadge>
              </div>
              <UButton
                color="primary"
                variant="soft"
                size="xs"
                icon="i-lucide-folder-input"
                @click="goToOrganizer"
              >
                Tata di Folder Organizer
              </UButton>
            </div>

            <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
              {{ cluster.reason }}
            </p>

            <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2 pt-1">
              <div
                v-for="p in cluster.projects"
                :key="p.id"
                class="p-2 rounded bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-700/40 text-xs font-mono"
              >
                <div class="font-semibold text-neutral-900 dark:text-white truncate">
                  {{ p.name }}
                </div>
                <div class="text-xs text-neutral-400 truncate mt-0.5">
                  {{ p.root_path }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer Dismiss -->
        <div class="flex justify-end pt-3 border-t border-neutral-100 dark:border-white/[0.08]">
          <UButton color="neutral" variant="outline" size="sm" @click="open = false">
            Tutup
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
