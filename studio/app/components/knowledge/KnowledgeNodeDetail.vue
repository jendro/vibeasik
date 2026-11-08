<script setup lang="ts">
import type { KnowledgeDetail } from '~/types'
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
</script>

<template>
  <div class="space-y-5">
    <p v-if="error" class="text-sm text-rose-600">
      {{ error }}
    </p>
    <template v-if="detail">
      <div class="space-y-2">
        <div class="flex items-center gap-1.5 flex-wrap">
          <UBadge size="sm" :color="KIND_META[detail.node.kind]?.color || 'neutral'" variant="subtle">
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
        </div>
        <p class="text-sm whitespace-pre-line text-neutral-700 dark:text-neutral-300">
          {{ detail.node.content }}
        </p>
        <p v-if="detail.node.status_reason" class="text-xs text-amber-600">
          {{ detail.node.status_reason }}
        </p>
        <p class="text-xs text-neutral-500">
          Utilitas {{ Math.round(detail.utility * 100) }}% · ✓{{ detail.node.helpful }} ✗{{ detail.node.harmful }} · dipakai {{ detail.node.uses }}× · bobot {{ detail.node.weight }}
          <span v-if="detail.node.key"> · sumber <code>{{ detail.node.key }}</code></span>
        </p>
        <UButton v-if="galleryLink" size="xs" variant="soft" icon="i-lucide-palette" :to="galleryLink.to">
          {{ galleryLink.label }}
        </UButton>
        <UButton v-if="detail.merged_to" size="xs" variant="link" icon="i-lucide-merge" @click="emit('open', detail.merged_to.id)">
          Dilebur ke K{{ detail.merged_to.id }} {{ detail.merged_to.title }}
        </UButton>
      </div>

      <div>
        <p class="text-xs font-semibold uppercase text-neutral-500 mb-1.5">
          Tetangga bertaut
        </p>
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

      <div>
        <p class="text-xs font-semibold uppercase text-neutral-500 mb-1.5">
          Dipakai di pekerjaan
        </p>
        <p v-if="!(detail.uses || []).length" class="text-xs text-neutral-400">
          Belum pernah disisipkan ke prompt.
        </p>
        <div v-for="u in detail.uses || []" :key="u.id" class="flex items-center gap-2 text-xs py-1">
          <code class="text-neutral-600 dark:text-neutral-300 truncate flex-1">{{ u.context }}</code>
          <span class="text-neutral-500">{{ u.outcome ? outcomeLabel[u.outcome] : 'menunggu nilai' }}</span>
          <span class="text-neutral-400">{{ when(u.created_at) }}</span>
        </div>
      </div>
    </template>
    <USkeleton v-else-if="!error" class="h-40 w-full" />
  </div>
</template>
