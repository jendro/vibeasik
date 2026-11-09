<script setup lang="ts">
import type { KnowledgeDetail } from '~/types'
import type { NodeImpact } from '~/composables/useKnowledgeImpact'
import { KIND_META, SOURCE_LABEL } from '~/utils/knowledge'

// Detail node: isi lengkap, bukti, tetangga bertaut (gaya Zettelkasten), dan riwayat pemakaian di pekerjaan.
const props = defineProps<{ id: number }>()
const emit = defineEmits<{ (e: 'open', id: number): void }>()
const api = useApi()
const detail = ref<KnowledgeDetail | null>(null)
const error = ref('')

watch(() => props.id, async (id) => {
  detail.value = null
  error.value = ''
  try {
    detail.value = await api.getKnowledgeNode(id)
  } catch (err: any) {
    error.value = err?.data?.error || err?.message || 'Gagal memuat node'
  }
}, { immediate: true })

// Node bersumber Galeri Desain / teori desain → tautan balik ke galeri (referensi, konsep teori, atau kesimpulan).
const galleryLink = computed(() => {
  const key = detail.value?.node.key || ''
  const ref = key.match(/^design:ref:(\d+)$/)
  if (ref) return { label: 'Buka referensi di Galeri Desain', to: { path: '/design-library', query: { ref: ref[1] } } }
  if (key.startsWith('theory:')) return { label: 'Buka di tab Teori Galeri Desain', to: { path: '/design-library', query: { view: 'theory', concept: key.slice(7) } } }
  if (key.startsWith('design:')) return { label: 'Buka Galeri Desain', to: { path: '/design-library', query: { view: key.startsWith('design:conclusion:') ? 'conclusions' : 'principles' } } }
  return null
})

const outcomeLabel: Record<string, string> = { helpful: '✓ membantu', harmful: '✗ merugikan', neutral: '· netral' }
const when = (iso?: string) => iso ? new Date(iso).toLocaleString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) : ''

// Tab: Komunitas (data server, default) · Pemakaian saya (catatan lokal) · Tetangga bertaut.
const tab = ref<'community' | 'mine' | 'links'>('community')
const tabs = computed(() => [
  { label: 'Komunitas', value: 'community', icon: 'i-lucide-users' },
  { label: `Pemakaian saya (${detail.value?.uses?.length || 0})`, value: 'mine', icon: 'i-lucide-user' },
  { label: `Tetangga (${detail.value?.neighbors?.length || 0})`, value: 'links', icon: 'i-lucide-waypoints' }
])
watch(() => props.id, () => { tab.value = 'community' })

// Dampak komunitas hanya ada untuk node milik paket (source hub); node lokal tidak ikut.
const { loadNode } = useKnowledgeImpact()
const community = ref<NodeImpact | null>(null)
const communityError = ref('')
const communityLoading = ref(false)
const isHub = computed(() => detail.value?.node.source === 'hub')
watch(detail, async (d) => {
  community.value = null
  communityError.value = ''
  if (!d || d.node.source !== 'hub') return
  communityLoading.value = true
  try {
    community.value = await loadNode(d.node.key)
  } catch (err: any) {
    communityError.value = err?.data?.error || err?.message || 'Gagal memuat dampak komunitas'
  } finally {
    communityLoading.value = false
  }
})

const scoreColor = (s: number) => s < 0 ? 'bg-neutral-300 dark:bg-neutral-600' : s >= 70 ? 'bg-emerald-500' : s >= 40 ? 'bg-amber-500' : 'bg-rose-500'
const maxKind = computed(() => Math.max(1, ...(community.value?.node?.by_kind || []).map(k => k.uses)))

// Ringkasan pemakaian lokal per jenis pekerjaan (awalan konteks: todo/audit/plan).
const KIND_LABEL: Record<string, string> = { todo: 'Pengerjaan todo', audit: 'Audit kode', plan: 'Perencanaan proyek', lain: 'Lainnya' }
const mineByKind = computed(() => {
  const m: Record<string, { uses: number, helpful: number, harmful: number }> = {}
  for (const u of detail.value?.uses || []) {
    const p = u.context.split(':')[0] || ''
    const k = KIND_LABEL[p] ? p : 'lain'
    const r = (m[k] ||= { uses: 0, helpful: 0, harmful: 0 })
    r.uses++
    if (u.outcome === 'helpful') r.helpful++
    else if (u.outcome === 'harmful') r.harmful++
  }
  return Object.entries(m).map(([k, v]) => ({ kind: k, label: KIND_LABEL[k], ...v })).sort((a, b) => b.uses - a.uses)
})

const stats = computed(() => {
  const n = detail.value?.node
  if (!n || !detail.value) return []
  return [
    { l: 'Utilitas', v: `${Math.round(detail.value.utility * 100)}%`, c: 'text-primary' },
    { l: 'Membantu', v: n.helpful, c: 'text-emerald-600 dark:text-emerald-400' },
    { l: 'Merugikan', v: n.harmful, c: 'text-rose-600 dark:text-rose-400' },
    { l: 'Dipakai', v: `${n.uses}×`, c: 'text-neutral-900 dark:text-white' },
    { l: 'Bobot', v: n.weight, c: 'text-neutral-900 dark:text-white' }
  ]
})
const dateOnly = (iso?: string) => iso ? new Date(iso).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }) : '-'
</script>

<template>
  <div class="space-y-5">
    <p v-if="error" class="text-sm text-rose-600">
      {{ error }}
    </p>
    <template v-if="detail">
      <!-- 1. Judul & kode -->
      <header class="space-y-2">
        <h2 class="text-lg font-bold leading-snug text-neutral-900 dark:text-white">
          {{ detail.node.title }}
        </h2>
        <p class="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs font-mono text-neutral-500">
          <span class="px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300">K{{ detail.node.id }}</span>
          <span v-if="detail.node.key" class="break-all">{{ detail.node.key }}</span>
        </p>
        <div class="flex items-center gap-1.5 flex-wrap">
          <UBadge size="sm" :color="KIND_META[detail.node.kind]?.color || 'neutral'" variant="subtle" :icon="KIND_META[detail.node.kind]?.icon">
            {{ KIND_META[detail.node.kind]?.label || detail.node.kind }}
          </UBadge>
          <UBadge v-if="detail.node.domain" size="sm" color="neutral" variant="outline">
            {{ detail.node.domain }}
          </UBadge>
          <UBadge size="sm" color="neutral" variant="soft">
            {{ SOURCE_LABEL[detail.node.source] || detail.node.source }}
          </UBadge>
          <UBadge v-if="detail.node.status !== 'active'" size="sm" color="warning" variant="soft">
            {{ detail.node.status }}
          </UBadge>
          <UBadge v-if="detail.node.needs_review" size="sm" color="error" variant="soft">
            perlu ditinjau
          </UBadge>
          <UBadge v-for="t in detail.node.tags || []" :key="t" size="sm" color="neutral" variant="subtle">
            #{{ t }}
          </UBadge>
        </div>
        <p v-if="detail.node.status_reason" class="text-xs text-amber-600">
          {{ detail.node.status_reason }}
        </p>
        <div class="flex flex-wrap gap-2">
          <UButton v-if="galleryLink" size="xs" variant="soft" icon="i-lucide-palette" :to="galleryLink.to">
            {{ galleryLink.label }}
          </UButton>
          <UButton v-if="detail.merged_to" size="xs" variant="soft" icon="i-lucide-merge" @click="emit('open', detail.merged_to.id)">
            Dilebur ke K{{ detail.merged_to.id }} {{ detail.merged_to.title }}
          </UButton>
        </div>
      </header>

      <!-- 2. Isi (markdown) -->
      <section class="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-4">
        <p class="text-xs font-semibold uppercase tracking-wide text-neutral-500 mb-2">
          Isi
        </p>
        <MarkdownViewer :content="detail.node.content" />
      </section>

      <!-- 3. Ringkasan angka -->
      <section>
        <p class="text-xs font-semibold uppercase tracking-wide text-neutral-500 mb-2">
          Ringkasan
        </p>
        <div class="grid grid-cols-5 gap-2">
          <div v-for="t in stats" :key="t.l" class="rounded-lg bg-neutral-50 dark:bg-neutral-800/60 p-2.5 text-center">
            <p class="text-base font-bold" :class="t.c">
              {{ t.v }}
            </p>
            <p class="text-xs uppercase font-semibold text-neutral-500">
              {{ t.l }}
            </p>
          </div>
        </div>
        <p class="text-xs text-neutral-400 mt-2">
          Dibuat {{ dateOnly(detail.node.created_at) }} · diperbarui {{ dateOnly(detail.node.updated_at) }}
          <template v-if="detail.node.last_used_at">
            · terakhir dipakai {{ dateOnly(detail.node.last_used_at) }}
          </template>
        </p>
      </section>

      <!-- 4. Pemakaian: komunitas (default) / saya / tetangga -->
      <section class="space-y-3">
        <UTabs v-model="tab" :items="tabs" size="sm" :content="false" class="w-full" />

        <!-- Komunitas -->
        <div v-if="tab === 'community'" class="space-y-3">
          <p v-if="!isHub" class="text-xs text-neutral-500">
            Node ini belum menjadi bagian paket komunitas, jadi belum ada data dari pengguna lain.
            Node buatan Anda yang terbukti berguna dikirim otomatis dan, setelah terbit di paket, datanya muncul di sini.
          </p>
          <USkeleton v-else-if="communityLoading" class="h-24 w-full" />
          <p v-else-if="communityError" class="text-xs text-amber-600">
            {{ communityError }}
          </p>
          <p v-else-if="community && !community.available" class="text-xs text-neutral-500">
            Belum cukup data komunitas. Angka ditampilkan setelah dipakai minimal {{ community.min_devices }} perangkat berbeda.
          </p>
          <template v-else-if="community?.node">
            <p class="text-sm text-neutral-700 dark:text-neutral-300">
              Dipakai <b>{{ community.node.uses }}×</b> oleh pengguna lain ·
              <span class="text-emerald-600 dark:text-emerald-400">✓{{ community.node.helpful }}</span>
              <span class="text-rose-600 dark:text-rose-400"> ✗{{ community.node.harmful }}</span>
              <template v-if="community.node.score >= 0">
                · <b>{{ community.node.score }}%</b> berguna
              </template>
            </p>
            <div v-for="k in community.node.by_kind" :key="k.kind" class="space-y-0.5">
              <div class="flex justify-between text-xs">
                <span class="font-medium text-neutral-700 dark:text-neutral-300">{{ k.label }}</span>
                <span class="text-neutral-500">{{ k.uses }}× · ✓{{ k.helpful }} ✗{{ k.harmful }}</span>
              </div>
              <div class="h-2 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
                <div class="h-full rounded-full" :class="scoreColor(k.score)" :style="{ width: `${Math.max(3, k.uses / maxKind * 100)}%` }" />
              </div>
            </div>
            <p class="text-xs text-neutral-400">
              Hanya angka per jenis pekerjaan; isi pekerjaan pengguna tidak diketahui.
            </p>
          </template>
        </div>

        <!-- Pemakaian saya -->
        <div v-else-if="tab === 'mine'" class="space-y-3">
          <p v-if="!(detail.uses || []).length" class="text-xs text-neutral-400">
            Belum pernah disisipkan ke prompt di komputer ini.
          </p>
          <template v-else>
            <div class="flex flex-wrap gap-2">
              <span v-for="k in mineByKind" :key="k.kind" class="text-xs rounded-md bg-neutral-100 dark:bg-neutral-800 px-2 py-1">
                {{ k.label }} <b>{{ k.uses }}×</b> <span class="text-emerald-600">✓{{ k.helpful }}</span> <span class="text-rose-600">✗{{ k.harmful }}</span>
              </span>
            </div>
            <div class="divide-y divide-neutral-100 dark:divide-neutral-800">
              <div v-for="u in detail.uses || []" :key="u.id" class="flex items-center gap-2 text-xs py-1.5">
                <code class="text-neutral-600 dark:text-neutral-300 truncate flex-1">{{ u.context }}</code>
                <span class="text-neutral-500 whitespace-nowrap">{{ u.outcome ? outcomeLabel[u.outcome] : 'menunggu nilai' }}</span>
                <span class="text-neutral-400 whitespace-nowrap">{{ when(u.created_at) }}</span>
              </div>
            </div>
          </template>
        </div>

        <!-- Tetangga -->
        <div v-else>
          <p v-if="!(detail.neighbors || []).length" class="text-xs text-neutral-400">
            Belum bertaut ke node lain.
          </p>
          <button
            v-for="n in detail.neighbors || []"
            :key="n.id"
            type="button"
            class="w-full text-left flex items-center gap-2 py-1.5 px-2 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800"
            @click="emit('open', n.id)"
          >
            <span class="text-xs font-mono text-neutral-400 w-10">K{{ n.id }}</span>
            <span class="text-sm flex-1 truncate">{{ n.title }}</span>
            <span class="text-xs text-neutral-500">{{ n.domain }}</span>
            <span class="text-xs font-mono text-neutral-400">{{ n.strength.toFixed(2) }}</span>
          </button>
        </div>
      </section>
    </template>
    <USkeleton v-else-if="!error" class="h-40 w-full" />
  </div>
</template>
