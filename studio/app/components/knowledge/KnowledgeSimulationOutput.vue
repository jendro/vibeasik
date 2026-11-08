<script setup lang="ts">
import type { KnowledgePromptPreview } from '~/types'

/**
 * Hasil simulasi pekerjaan: bukan hanya prompt. Satu pekerjaan sungguhan menghasilkan (1) prompt final pegawai untuk
 * agent coding, (2) penugasan pegawai, (3) catatan pemakaian butir yang menunggu nilai, (4) celah pengetahuan bila
 * ekosistemnya belum dikenal, lalu (5) bukti ✓/✗ & pelajaran baru setelah dinilai QA. Ini pratinjau: tidak ada yang dicatat.
 */
const props = defineProps<{ preview: KnowledgePromptPreview, task: string }>()
const emit = defineEmits<{ (e: 'open', id: number): void }>()
const toast = useToast()

// Pekerjaan lintas aplikasi: satu sub todo per aplikasi, masing-masing dengan pegawai & prompt sendiri.
const sel = ref(0)
watch(() => props.preview, () => { sel.value = 0 })
const split = computed(() => props.preview.parts || [])
const current = computed(() => {
  const p = split.value[sel.value]
  if (p) return { employee: p.employee, reason: p.reason, prompt: p.prompt, knowledge_section: p.knowledge_section, hits: p.hits || [], header: false }
  const v = props.preview
  return { employee: v.employee, reason: v.reason, prompt: v.prompt, knowledge_section: v.knowledge_section, hits: v.hits || [], header: true }
})
const CLASS_LABEL: Record<string, string> = { backend: 'Backend', frontend: 'Frontend', mobile: 'Mobile' }

// Prompt dipecah: sebelum · bagian Pusat Pengetahuan (disorot) · sesudah.
const parts = computed(() => {
  const p = current.value.prompt || ''
  const k = current.value.knowledge_section || ''
  const i = k ? p.indexOf(k) : -1
  if (i < 0) return { before: p, knowledge: '', after: '' }
  return { before: p.slice(0, i), knowledge: k, after: p.slice(i + k.length) }
})
const tokens = (s: string) => Math.round(s.length / 4)
const hits = computed(() => current.value.hits)
const gaps = computed(() => props.preview.gaps || [])

const copy = async () => {
  try {
    await navigator.clipboard.writeText(current.value.prompt)
    toast.add({ title: 'Prompt disalin', color: 'success' })
  } catch {
    toast.add({ title: 'Gagal menyalin prompt', color: 'error' })
  }
}
</script>

<template>
  <section class="rounded-2xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-4 sm:p-5 space-y-4">
    <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-2">
      <div>
        <h2 class="text-base font-bold text-neutral-900 dark:text-white flex items-center gap-2">
          <UIcon name="i-lucide-package-open" class="size-4.5 text-primary-500" /> Yang dihasilkan pekerjaan ini
        </h2>
        <p class="text-xs text-neutral-500 mt-0.5">
          “{{ task }}” — hasilnya bukan hanya prompt: ada penugasan, catatan pemakaian, celah, lalu bukti belajar. Ini pratinjau; pada todo sungguhan semuanya benar-benar dicatat.
        </p>
      </div>
      <UBadge color="warning" variant="soft" size="sm" icon="i-lucide-flask-conical">
        Pratinjau · tidak ada yang dicatat
      </UBadge>
    </div>

    <div v-if="split.length" class="space-y-2">
      <p class="text-xs text-neutral-600 dark:text-neutral-400">
        Pekerjaan ini menyentuh <b>{{ split.length }} aplikasi berbeda</b>, jadi dipecah menjadi {{ split.length }} sub todo — tiap pegawai menerima prompt final sesuai keahliannya, mengerjakan aplikasinya saja, dan dinilai QA sendiri. Induknya selesai otomatis saat semua bagian selesai.
      </p>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="(p, i) in split"
          :key="p.title"
          type="button"
          class="text-left px-3 py-2 rounded-xl border transition"
          :class="sel === i ? 'border-primary-400 bg-primary-50 dark:bg-primary-950/30' : 'border-neutral-200 dark:border-neutral-800 hover:border-primary-300'"
          @click="sel = i"
        >
          <span class="block text-xs uppercase font-semibold text-neutral-500">{{ CLASS_LABEL[p.class] || p.class }} · {{ p.apps.join(', ') }}</span>
          <span class="block text-sm font-semibold text-neutral-900 dark:text-white">{{ p.employee?.name || 'Belum ada pegawai' }}</span>
          <span class="block text-xs text-neutral-500">{{ p.employee?.role || '—' }} · {{ (p.hits || []).length }} butir</span>
        </button>
      </div>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-[1fr_360px] gap-4">
      <div class="space-y-2 min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-sm font-semibold text-neutral-900 dark:text-white">1. Prompt final untuk agent coding<template v-if="split.length"> — {{ current.employee?.name || CLASS_LABEL[split[sel]?.class || ''] }}</template></span>
          <span class="text-xs text-neutral-500">≈{{ tokens(current.prompt) }} token · bagian Pusat Pengetahuan ≈{{ tokens(parts.knowledge) }} token</span>
          <UButton class="ml-auto" size="xs" color="neutral" variant="outline" icon="i-lucide-copy" @click="copy">
            Salin
          </UButton>
        </div>
        <p class="text-xs text-neutral-500">
          Disisipkan di bawah isi todo saat agent mulai mengerjakan (update_todo in_progress / vibeasik_staff_brief). Bagian yang <mark class="px-1 rounded bg-amber-200/70 dark:bg-amber-500/25 text-inherit">disorot</mark> berasal dari jaring ini.
        </p>
        <pre class="max-h-[520px] overflow-auto rounded-xl bg-neutral-50 dark:bg-neutral-950 border border-neutral-200 dark:border-neutral-800 p-3 text-xs leading-relaxed whitespace-pre-wrap break-words font-mono text-neutral-700 dark:text-neutral-300"><span v-if="current.header" class="text-neutral-400"># Todo: {{ task }}
</span>{{ parts.before }}<mark class="rounded bg-amber-200/60 dark:bg-amber-500/20 text-inherit">{{ parts.knowledge }}</mark>{{ parts.after }}</pre>
      </div>

      <div class="space-y-3">
        <div class="p-3 rounded-xl border border-neutral-200 dark:border-neutral-800">
          <p class="text-sm font-semibold text-neutral-900 dark:text-white">
            2. Pegawai yang mengerjakan
          </p>
          <p v-if="current.employee" class="mt-1 text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            <b class="text-neutral-900 dark:text-white">{{ current.employee.name }}</b> ({{ current.employee.role }}). {{ current.reason }}
          </p>
          <p v-else class="mt-1 text-xs text-neutral-500">
            Fitur pegawai mati / belum ada pegawai: agent menerima bagian pengetahuan saja.
          </p>
        </div>

        <div class="p-3 rounded-xl border border-neutral-200 dark:border-neutral-800">
          <p class="text-sm font-semibold text-neutral-900 dark:text-white">
            3. Catatan pemakaian ({{ hits.length }} baris{{ split.length ? ' untuk bagian ini' : '' }})
          </p>
          <p class="mt-1 text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Tiap butir dicatat dengan kunci <code class="text-xs">{{ preview.context }}</code> berstatus <b>menunggu nilai</b>; hitungan pakai & “terakhir dipakai” butirnya naik (skor kebaruan ikut naik).
          </p>
          <div class="mt-2 flex flex-wrap gap-1">
            <button
              v-for="h in hits"
              :key="h.id"
              type="button"
              class="text-xs font-mono px-1.5 py-0.5 rounded border border-neutral-200 dark:border-neutral-700 hover:border-primary-400"
              :title="h.title"
              @click="emit('open', h.id)"
            >
              K{{ h.id }}
            </button>
          </div>
        </div>

        <div class="p-3 rounded-xl border border-neutral-200 dark:border-neutral-800">
          <p class="text-sm font-semibold text-neutral-900 dark:text-white">
            4. Celah pengetahuan
          </p>
          <p v-if="gaps.length" class="mt-1 text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Ekosistem <b>{{ gaps.join(', ') }}</b> belum punya butir sama sekali → dicatat sebagai celah otomatis, agent diberi peringatan untuk membaca dokumentasi resmi, dan pegawai yang cocok ditugaskan mempelajarinya hingga menjadi node baru.
          </p>
          <p v-else class="mt-1 text-xs text-neutral-500 leading-relaxed">
            Tidak ada — semua ekosistem yang dikenali ({{ (preview.domains || []).join(', ') || 'umum' }}) sudah punya pengetahuan.
          </p>
        </div>

        <div class="p-3 rounded-xl border border-emerald-200 dark:border-emerald-900 bg-emerald-50/50 dark:bg-emerald-950/20">
          <p class="text-sm font-semibold text-neutral-900 dark:text-white">
            5. Setelah dinilai QA
          </p>
          <ul class="mt-1 text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed space-y-0.5 list-disc pl-4">
            <li>Nilai ≥75 → {{ hits.length }} butir di atas dapat <b>✓ membantu</b> (node membesar, makin sering terambil).</li>
            <li>Nilai 50–74 → netral; nilai &lt;50 → <b>✗ keliru</b> (✗≥3 & utilitas rendah → ditinjau / tidak dipakai).</li>
            <li>K&lt;id&gt; yang dilaporkan keliru oleh agent → ✗ langsung untuk butir itu.</li>
            <li>Bagian <b>PELAJARAN:</b> di catatan penyelesaian → keahlian baru pegawai → disinkronkan sebagai node baru.</li>
          </ul>
        </div>
      </div>
    </div>
  </section>
</template>
