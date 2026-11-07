<script setup lang="ts">
import type { Tutorial } from '~/types'

const { fetchTutorials } = useVibeasik()

const tutorials = ref<Tutorial[]>([])
const loading = ref(true)
const selectedCategory = ref('all')
const searchQuery = ref('')
const activeTutorial = ref<Tutorial | null>(null)
const isModalOpen = ref(false)

const categories = [
  { id: 'all', label: 'Semua Materi' },
  { id: 'dasar-pemrograman', label: 'Dasar Pemrograman' },
  { id: 'database', label: 'Database & ORM' },
  { id: 'security', label: 'Keamanan Web' },
  { id: 'deployment', label: 'Deploy & Hosting' },
  { id: 'ai-agent', label: 'AI Agent & System One' }
]

const loadData = async () => {
  loading.value = true
  try {
    const cat = selectedCategory.value === 'all' ? '' : selectedCategory.value
    tutorials.value = await fetchTutorials(cat)
  } catch (err) {
    console.error('Gagal memuat tutorial:', err)
  } finally {
    loading.value = false
  }
}

watch(selectedCategory, () => {
  loadData()
})

onMounted(() => {
  loadData()
})

const filteredTutorials = computed(() => {
  if (!searchQuery.value.trim()) return tutorials.value
  const q = searchQuery.value.toLowerCase()
  return tutorials.value.filter(
    t => t.title.toLowerCase().includes(q) || t.summary.toLowerCase().includes(q) || t.tags.toLowerCase().includes(q)
  )
})

const openTutorial = (t: Tutorial) => {
  activeTutorial.value = t
  isModalOpen.value = true
}

const parseSnippets = (jsonStr: string): string[] => {
  try {
    return JSON.parse(jsonStr) || []
  } catch {
    return []
  }
}

const copiedSnippetIdx = ref<number | null>(null)
const copySnippet = (text: string, idx: number) => {
  navigator.clipboard.writeText(text)
  copiedSnippetIdx.value = idx
  setTimeout(() => {
    copiedSnippetIdx.value = null
  }, 2000)
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header Banner -->
    <div class="p-6 rounded-2xl bg-white dark:bg-slate-900/60 border border-neutral-200 dark:border-white/10 shadow-xs flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
      <div class="space-y-1">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-graduation-cap" class="w-7 h-7 text-sky-500" />
          <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white">
            Vibe Academy
          </h1>
          <UBadge color="primary" variant="subtle" size="sm">
            Edukasi Vibecoder
          </UBadge>
        </div>
        <p class="text-sm text-neutral-600 dark:text-slate-300 max-w-2xl">
          Kuasai konsep inti arsitektur web, database, keamanan kode, dan cara menanamkan AI Agent ke sistem Anda tanpa pusing teori rumit.
        </p>
      </div>

      <div class="w-full md:w-72">
        <UInput
          v-model="searchQuery"
          icon="i-lucide-search"
          placeholder="Cari topik atau keyword..."
          class="w-full"
        />
      </div>
    </div>

    <!-- Category Pills -->
    <div class="flex items-center gap-2 overflow-x-auto pb-2 scrollbar-none">
      <button
        v-for="cat in categories"
        :key="cat.id"
        type="button"
        class="px-3.5 py-1.5 rounded-full text-xs font-medium whitespace-nowrap transition-all shrink-0 cursor-pointer"
        :class="[
          selectedCategory === cat.id
            ? 'bg-sky-600 text-white shadow-xs'
            : 'bg-white dark:bg-slate-900/40 border border-neutral-200 dark:border-white/10 text-neutral-600 dark:text-slate-300 hover:border-sky-500/50'
        ]"
        @click="selectedCategory = cat.id"
      >
        {{ cat.label }}
      </button>
    </div>

    <!-- Content Grid -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="i in 6" :key="i" class="h-44 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-5 animate-pulse space-y-3">
        <div class="h-4 bg-neutral-200 dark:bg-neutral-800 rounded-md w-1/3" />
        <div class="h-5 bg-neutral-200 dark:bg-neutral-800 rounded-md w-3/4" />
        <div class="h-14 bg-neutral-100 dark:bg-neutral-800/50 rounded-md w-full" />
      </div>
    </div>

    <div v-else-if="filteredTutorials.length === 0" class="p-12 text-center rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-3">
      <UIcon name="i-lucide-book-open" class="w-10 h-10 mx-auto text-neutral-400" />
      <h3 class="font-medium text-neutral-800 dark:text-neutral-200">
        Tidak ada materi ditemukan
      </h3>
      <p class="text-sm text-neutral-500 max-w-md mx-auto">
        Coba ganti filter kategori atau kata kunci pencarian Anda.
      </p>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="t in filteredTutorials"
        :key="t.id"
        class="group p-5 rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 hover:border-sky-500/50 dark:hover:border-sky-500/50 transition-all shadow-xs hover:shadow-md flex flex-col justify-between cursor-pointer"
        @click="openTutorial(t)"
      >
        <div class="space-y-2.5">
          <div class="flex items-center justify-between gap-2">
            <UBadge size="xs" color="primary" variant="subtle" class="capitalize">
              {{ t.category.replace('-', ' ') }}
            </UBadge>
            <span class="text-xs text-neutral-500 dark:text-neutral-400 flex items-center gap-1 font-mono">
              <UIcon name="i-lucide-clock" class="w-3.5 h-3.5" />
              {{ t.estimated_minutes }} mnt
            </span>
          </div>

          <h3 class="font-semibold text-base text-neutral-900 dark:text-white group-hover:text-sky-600 dark:group-hover:text-sky-400 transition-colors line-clamp-2">
            {{ t.title }}
          </h3>

          <p class="text-xs text-neutral-600 dark:text-slate-300 line-clamp-3 leading-relaxed">
            {{ t.summary }}
          </p>
        </div>

        <div class="pt-4 mt-3 border-t border-neutral-100 dark:border-white/10 flex items-center justify-between">
          <span class="text-xs font-medium text-sky-600 dark:text-sky-400 flex items-center gap-1 group-hover:translate-x-1 transition-transform">
            Baca Materi
            <UIcon name="i-lucide-arrow-right" class="w-3.5 h-3.5" />
          </span>
          <div class="flex items-center gap-2">
            <span v-if="t.source_url" class="text-[10px] text-sky-600 dark:text-sky-400 font-mono flex items-center gap-0.5 bg-sky-50 dark:bg-sky-950/50 px-1.5 py-0.5 rounded border border-sky-200 dark:border-sky-800/50">
              <UIcon name="i-lucide-book-open" class="w-2.5 h-2.5" />
              Ada Sumber
            </span>
            <span class="text-[11px] font-mono text-neutral-400 capitalize">{{ t.difficulty }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Reading Modal -->
    <UModal v-model:open="isModalOpen" :ui="{ content: 'max-w-3xl' }">
      <template v-if="activeTutorial" #content>
        <div class="p-6 space-y-5 max-h-[85vh] overflow-y-auto">
          <!-- Modal Header -->
          <div class="space-y-2 border-b border-neutral-200 dark:border-white/10 pb-4">
            <div class="flex items-center gap-2 flex-wrap">
              <UBadge color="primary" variant="subtle" size="sm" class="capitalize">
                {{ activeTutorial.category.replace('-', ' ') }}
              </UBadge>
              <UBadge color="neutral" variant="outline" size="sm" class="font-mono capitalize">
                {{ activeTutorial.difficulty }}
              </UBadge>
              <a
                v-if="activeTutorial.source_url"
                :href="activeTutorial.source_url"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-1 text-xs text-sky-600 dark:text-sky-400 hover:underline font-mono bg-sky-50 dark:bg-sky-950/40 px-2 py-0.5 rounded-full border border-sky-200 dark:border-sky-800/60 transition-colors"
                @click.stop
              >
                <UIcon name="i-lucide-external-link" class="w-3 h-3" />
                <span>Dokumentasi Resmi</span>
              </a>
              <span class="text-xs text-neutral-500 font-mono ml-auto flex items-center gap-1">
                <UIcon name="i-lucide-clock" class="w-3.5 h-3.5" />
                {{ activeTutorial.estimated_minutes }} menit baca
              </span>
            </div>
            <h2 class="text-xl font-bold text-neutral-900 dark:text-white leading-snug">
              {{ activeTutorial.title }}
            </h2>
          </div>

          <!-- Markdown Content -->
          <MarkdownViewer :content="activeTutorial.content_markdown" />

          <!-- Code Snippets Box -->
          <div v-if="parseSnippets(activeTutorial.code_snippets).length > 0" class="space-y-3 pt-3">
            <h4 class="text-xs font-bold tracking-wider uppercase text-neutral-500 font-mono">
              Kode Contoh / Snippet Siap Pakai:
            </h4>
            <div
              v-for="(snip, idx) in parseSnippets(activeTutorial.code_snippets)"
              :key="idx"
              class="relative rounded-xl overflow-hidden border border-neutral-800 bg-neutral-950 font-mono text-xs text-neutral-200 p-4"
            >
              <button
                type="button"
                class="absolute right-3 top-3 p-1.5 rounded-md bg-neutral-800 hover:bg-neutral-700 text-neutral-300 transition-colors flex items-center gap-1 text-[11px]"
                @click="copySnippet(snip, idx)"
              >
                <UIcon :name="copiedSnippetIdx === idx ? 'i-lucide-check' : 'i-lucide-copy'" class="w-3.5 h-3.5" />
                {{ copiedSnippetIdx === idx ? 'Tersalin' : 'Salin' }}
              </button>
              <pre class="overflow-x-auto whitespace-pre pr-16">{{ snip }}</pre>
            </div>
          </div>

          <!-- Modal Footer -->
          <div class="pt-4 border-t border-neutral-200 dark:border-neutral-800 flex justify-end">
            <UButton color="neutral" variant="subtle" @click="isModalOpen = false">
              Tutup
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
