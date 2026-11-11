<script setup lang="ts">
import type { Employee, KnowledgeHit, KnowledgeKind } from '~/types'
import { KIND_META, SOURCE_LABEL } from '~/utils/knowledge'

/**
 * Pusat Pengetahuan: SATU PINTU semua patokan & pengetahuan umum pekerja AI vibeasik — guideline, checklist audit,
 * pelajaran vonis audit, keahlian pegawai, dan hasil belajar teknologi baru. Audit, rapat PM, dan pegawai mengambil
 * butir dari sini lewat temu-kembali berbobot; hasil kerja (nilai QA, vonis audit, umpan balik) menjadi bukti
 * ✓/✗ yang menaikkan/menurunkan butir. Fakta khusus proyek tetap milik dewan eksekutif proyek.
 */
definePageMeta({ fluid: true })

const { overview, nodes, loading, nodesLoading, curating, error, filter, load, loadNodes, teach, feedback, retire, reportGap, dismissGap, curate, recall } = useKnowledge()
const api = useApi()
const employees = ref<Employee[]>([])

const view = ref<'nodes' | 'clusters' | 'gaps' | 'recall' | 'review'>('nodes')
const openGaps = computed(() => (overview.value?.gaps || []).filter(g => g.status === 'open' || g.status === 'learning').length)
const viewTabs = computed(() => [
  { label: 'Butir Pengetahuan', value: 'nodes', icon: 'i-lucide-library' },
  { label: `Klaster Konsep (${overview.value?.clusters?.length || 0})`, value: 'clusters', icon: 'i-lucide-shapes' },
  { label: `Celah & Belajar (${openGaps.value})`, value: 'gaps', icon: 'i-lucide-graduation-cap' },
  { label: 'Uji Temu-kembali', value: 'recall', icon: 'i-lucide-search-check' },
  { label: `Perlu Ditinjau (${overview.value?.needs_review?.length || 0})`, value: 'review', icon: 'i-lucide-triangle-alert' }
])

const kindItems = [{ label: 'Semua jenis', value: 'all' }, ...(Object.keys(KIND_META) as KnowledgeKind[]).map(k => ({ label: KIND_META[k].label, value: k }))]
const sourceItems = computed(() => [{ label: 'Semua asal', value: 'all' }, ...Object.keys(overview.value?.by_source || {}).map(s => ({ label: `${SOURCE_LABEL[s as keyof typeof SOURCE_LABEL] || s} (${overview.value?.by_source[s]})`, value: s }))])
const domainItems = computed(() => [{ label: 'Semua domain', value: 'all' }, ...(overview.value?.domains || []).map(d => ({ label: `${d.domain} (${d.nodes})`, value: d.domain }))])

let searchTimer: ReturnType<typeof setTimeout> | null = null
watch(() => [filter.q, filter.domain, filter.kind, filter.source, filter.cluster_id], () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(loadNodes, 300)
})

// Tautan dari Galeri Desain (tab Teori, detail referensi): ?q=<kata kunci>&source=<asal>.
const route = useRoute()
if (typeof route.query.q === 'string') filter.q = route.query.q
if (typeof route.query.source === 'string') filter.source = route.query.source

onMounted(async () => {
  await Promise.all([load(), loadNodes()])
  try {
    employees.value = (await api.getStaff()).employees
  } catch {
    employees.value = []
  }
})

const showTeach = ref(false)
const form = reactive<{ title: string, content: string, kind: KnowledgeKind, domain: string }>({ title: '', content: '', kind: 'lesson', domain: '' })
const submitTeach = async () => {
  if (!form.title.trim() || !form.content.trim()) return
  if (await teach({ ...form, domain: form.domain || undefined })) {
    Object.assign(form, { title: '', content: '', kind: 'lesson', domain: '' })
    showTeach.value = false
  }
}

const detailId = ref<number | null>(null)
const detailOpen = computed({ get: () => !!detailId.value, set: (v) => { if (!v) detailId.value = null } })
const openNode = (n: KnowledgeHit | number) => { detailId.value = typeof n === 'number' ? n : n.id }
if (Number(route.query.node)) detailId.value = Number(route.query.node)
const showCluster = (id: number) => {
  filter.cluster_id = id
  filter.q = ''
  view.value = 'nodes'
}
const pct = (v: number) => `${Math.round(v * 100)}%`
</script>

<template>
  <div class="w-full max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 pt-6 pb-20 space-y-6">
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4 p-5 sm:p-6 rounded-2xl bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 shadow-xs">
      <div class="flex items-center gap-3.5 min-w-0">
        <div class="size-11 shrink-0 rounded-xl bg-linear-to-br from-indigo-500 to-emerald-500 text-white flex items-center justify-center shadow-xs">
          <UIcon name="i-lucide-brain-circuit" class="size-5.5" />
        </div>
        <div class="min-w-0">
          <h1 class="text-xl font-bold tracking-tight text-neutral-900 dark:text-white">
            Pusat Pengetahuan
          </h1>
          <p class="text-xs text-neutral-500 mt-0.5">
            Satu pintu patokan & pengetahuan pekerja AI: dipakai audit, rapat PM, dan pegawai; belajar dari nilai QA, vonis audit & teknologi baru. Fakta proyek tetap di dewan eksekutif.
          </p>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2 shrink-0">
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-book-marked" to="/knowledge-docs">
          Dokumentasi
        </UButton>
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-waypoints" to="/knowledge-graph">
          Lihat jaring
        </UButton>
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="loading" @click="() => { load(); loadNodes() }">
          Muat ulang
        </UButton>
        <UButton size="sm" color="primary" icon="i-lucide-sparkles" :loading="curating" @click="curate">
          Kurasi sekarang
        </UButton>
      </div>
    </div>

    <p v-if="error" class="text-sm text-rose-600 dark:text-rose-400">
      {{ error }}
    </p>

    <div v-if="overview" class="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3">
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
        <p class="text-xs uppercase font-semibold text-neutral-500">
          Node aktif
        </p>
        <p class="text-2xl font-bold text-neutral-900 dark:text-white">
          {{ overview.total }}
        </p>
      </div>
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
        <p class="text-xs uppercase font-semibold text-neutral-500">
          Pustaka bersama
        </p>
        <p class="text-2xl font-bold text-indigo-600 dark:text-indigo-400">
          {{ overview.shared }}
        </p>
      </div>
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
        <p class="text-xs uppercase font-semibold text-neutral-500">
          Keahlian pribadi
        </p>
        <p class="text-2xl font-bold text-sky-600 dark:text-sky-400">
          {{ overview.personal }}
        </p>
      </div>
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
        <p class="text-xs uppercase font-semibold text-neutral-500">
          Teruji · utilitas
        </p>
        <p class="text-2xl font-bold text-emerald-600 dark:text-emerald-400">
          {{ overview.tested }} <span class="text-sm font-medium text-neutral-500">· {{ overview.tested ? pct(overview.utility) : '—' }}</span>
        </p>
      </div>
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
        <p class="text-xs uppercase font-semibold text-neutral-500">
          Tautan
        </p>
        <p class="text-2xl font-bold text-neutral-900 dark:text-white">
          {{ overview.links }}
        </p>
      </div>
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
        <p class="text-xs uppercase font-semibold text-neutral-500">
          Menunggu nilai
        </p>
        <p class="text-2xl font-bold text-amber-600 dark:text-amber-400">
          {{ overview.uses_pending }}
        </p>
      </div>
    </div>

    <KnowledgeImpact />

    <UTabs v-model="view" :items="viewTabs" size="sm" :content="false" class="w-full overflow-x-auto" />

    <section v-if="view === 'nodes'" class="space-y-3">
      <div class="flex flex-col lg:flex-row gap-2">
        <UInput v-model="filter.q" icon="i-lucide-search" placeholder="Cari (vektor TF-IDF, ID ↔ EN)…" class="flex-1" />
        <USelect v-model="filter.domain" :items="domainItems" class="w-full lg:w-44" />
        <USelect v-model="filter.kind" :items="kindItems" class="w-full lg:w-40" />
        <USelect v-model="filter.source" :items="sourceItems" class="w-full lg:w-52" />
        <UButton color="primary" variant="soft" icon="i-lucide-book-plus" @click="showTeach = !showTeach">
          Ajarkan
        </UButton>
      </div>
      <div v-if="filter.cluster_id" class="flex items-center gap-2 text-xs text-neutral-500">
        <span>Klaster #{{ filter.cluster_id }}</span>
        <UButton size="xs" variant="link" @click="filter.cluster_id = 0">
          hapus filter klaster
        </UButton>
      </div>
      <div v-if="showTeach" class="p-4 rounded-xl border border-indigo-200 dark:border-indigo-900 bg-indigo-50/50 dark:bg-indigo-950/20 space-y-2">
        <p class="text-xs text-neutral-500">
          Untuk pustaka bersama kantor: pengetahuan umum yang bisa dipakai di proyek mana pun (bukan fakta satu aplikasi). Butir yang hampir sama akan memperkuat butir lama.
        </p>
        <UInput v-model="form.title" placeholder="Topik singkat" class="w-full" />
        <UTextarea v-model="form.content" :rows="3" autoresize placeholder="Isi 1–3 kalimat yang konkret" class="w-full" />
        <div class="flex flex-col sm:flex-row gap-2">
          <USelect v-model="form.kind" :items="kindItems.slice(1)" class="w-full sm:w-44" />
          <UInput v-model="form.domain" placeholder="domain (opsional, dideteksi otomatis)" class="flex-1" />
          <UButton icon="i-lucide-check" @click="submitTeach">
            Simpan
          </UButton>
        </div>
      </div>
      <USkeleton v-if="nodesLoading && !nodes.length" class="h-32 w-full" />
      <p v-else-if="!nodes.length" class="text-sm text-neutral-500">
        Tidak ada node yang cocok.
      </p>
      <div class="grid grid-cols-1 xl:grid-cols-2 gap-3">
        <KnowledgeNodeCard
          v-for="n in nodes"
          :key="n.id"
          :node="n"
          :show-score="!!filter.q"
          @open="openNode"
          @feedback="feedback"
          @retire="retire"
        />
      </div>
    </section>

    <section v-else-if="view === 'clusters'" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
      <p v-if="!(overview?.clusters || []).length" class="text-sm text-neutral-500">
        Klaster dihitung kurator (tiap 15 menit, atau tombol Kurasi sekarang).
      </p>
      <button
        v-for="c in overview?.clusters || []"
        :key="c.id"
        type="button"
        class="text-left p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 hover:border-primary-400 transition"
        @click="showCluster(c.id)"
      >
        <div class="flex items-center justify-between gap-2">
          <UBadge size="xs" color="neutral" variant="outline">
            {{ c.domain || 'umum' }}
          </UBadge>
          <span class="text-xs text-neutral-500">{{ c.size }} node · utilitas {{ pct(c.utility) }}</span>
        </div>
        <p class="mt-2 text-sm font-semibold text-neutral-900 dark:text-white">
          {{ c.label }}
        </p>
      </button>
    </section>

    <KnowledgeGapList v-else-if="view === 'gaps'" :gaps="overview?.gaps || []" @report="reportGap" @dismiss="dismissGap" />

    <KnowledgeRecallPanel v-else-if="view === 'recall'" :employees="employees" :recall="recall" @open="openNode" @feedback="feedback" @retire="retire" />

    <section v-else class="space-y-3">
      <p class="text-xs text-neutral-500">
        Patokan bersumber berkas (guideline / checklist audit) yang berulang kali terbukti keliru (✗≥3, utilitas &lt;40%). Perbarui berkas sumbernya — butir di pusat ikut berubah saat sinkron berikutnya.
      </p>
      <p v-if="!(overview?.needs_review || []).length" class="text-sm text-neutral-500">
        Tidak ada patokan yang perlu ditinjau.
      </p>
      <KnowledgeNodeCard
        v-for="n in overview?.needs_review || []"
        :key="n.id"
        :node="{ ...n, score: 0, relevance: 0, utility: (n.helpful + 1) / (n.helpful + n.harmful + 2) }"
        @open="openNode"
        @feedback="feedback"
        @retire="retire"
      />
    </section>

    <USlideover v-model:open="detailOpen" title="Detail node" :ui="{ content: 'max-w-xl' }">
      <template #body>
        <KnowledgeNodeDetail v-if="detailId" :id="detailId" @open="openNode" />
      </template>
    </USlideover>
  </div>
</template>
