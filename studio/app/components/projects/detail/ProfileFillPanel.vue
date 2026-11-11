<script setup lang="ts">
import type { Project, ProjectProfile } from '~/types'

const { modelName } = useModelNames()

const props = defineProps<{
  project: Project
  profile: ProjectProfile
  generating?: boolean
  generatingLabel?: string
}>()

const emit = defineEmits<{
  generate: []
  updated: [profile: ProjectProfile]
}>()

const toast = useToast()
const { getProjectProfile, getProjectProfilePrompt, saveProjectProfile } = useApi()

const sourceInfo = computed(() => {
  switch (props.profile.source) {
    case 'agent': return { label: 'Diisi AI Agent', icon: 'i-lucide-bot', tone: 'sky' }
    case 'llm': return { label: 'Dianalisis LLM bawaan', icon: 'i-lucide-sparkles', tone: 'sky' }
    default: return { label: 'Perkiraan otomatis — belum diisi AI', icon: 'i-lucide-info', tone: 'amber' }
  }
})
const generatedLabel = computed(() => props.profile.generated_at
  ? new Date(props.profile.generated_at).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' })
  : '')

const showGuide = ref(props.profile.source === 'heuristic')

// --- Agent route: copy prompt, then watch for the agent's save_project_profile call ---
const copyingPrompt = ref(false)
const waitingForAgent = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null
let pollDeadline = 0

const stopWaiting = () => {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = null
  waitingForAgent.value = false
}

const startWaiting = () => {
  stopWaiting()
  const since = props.profile.generated_at
  waitingForAgent.value = true
  pollDeadline = Date.now() + 15 * 60 * 1000
  pollTimer = setInterval(async () => {
    if (Date.now() > pollDeadline) return stopWaiting()
    try {
      const fresh = await getProjectProfile(props.project.id)
      if (fresh.source === 'agent' && fresh.generated_at !== since) {
        stopWaiting()
        emit('updated', fresh)
        toast.add({ title: 'Profil Diperbarui oleh Agent', description: modelName(fresh.model) || 'AI agent via MCP', icon: 'i-lucide-bot', color: 'success' })
      }
    } catch {
      // engine may be busy; try again on the next tick
    }
  }, 5000)
}

onBeforeUnmount(stopWaiting)

const copyText = async (text: string, title: string, description: string) => {
  await navigator.clipboard.writeText(text)
  toast.add({ title, description, icon: 'i-lucide-clipboard-check', color: 'success' })
}

const copyQuickPrompt = async (q: { label: string, text: string }) => {
  try {
    await copyText(q.text, 'Perintah Disalin', `Tempel ke agent untuk memperbarui ${q.label.toLowerCase()}.`)
    startWaiting()
  } catch {
    toast.add({ title: 'Gagal Menyalin', description: 'Clipboard tidak dapat diakses.', icon: 'i-lucide-alert-circle', color: 'error' })
  }
}

const copyAgentPrompt = async () => {
  copyingPrompt.value = true
  try {
    const res = await getProjectProfilePrompt(props.project.id)
    await copyText(res.agent_prompt, 'Prompt Agent Disalin', 'Tempel ke Claude Code / agy di folder proyek. Halaman ini akan memperbarui otomatis saat agent menyimpan.')
    startWaiting()
  } catch (err: any) {
    toast.add({ title: 'Gagal Menyalin Prompt', description: err.message || 'Terjadi kesalahan', icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    copyingPrompt.value = false
  }
}

// Short instructions for updating one part of the profile; the agent fetches context itself via MCP.
const quickPrompts = computed(() => {
  const id = props.project.id
  const base = `Gunakan MCP vibeasik. Panggil get_project_profile (project_id: ${id}, with_prompt: true) untuk konteks, baca kode proyek "${props.project.name}" untuk verifikasi, lalu simpan dengan save_project_profile (project_id: ${id}) dan HANYA kirim field`
  return [
    { label: 'Arah pengembangan', icon: 'i-lucide-compass', text: `${base} "roadmap", "risks", dan "open_questions". Turunkan dari TODO terbuka, fitur setengah jadi, dan commit terbaru.` },
    { label: 'Daftar fitur', icon: 'i-lucide-puzzle', text: `${base} "features". Gabungkan fitur yang sama lintas aplikasi dan sertakan evidence konkret.` },
    { label: 'Tujuan & kematangan', icon: 'i-lucide-target', text: `${base} "short_description", "full_description", "purpose", dan "maturity".` }
  ]
})

// --- Manual fallback: paste the JSON an agent produced without MCP ---
const pasteOpen = ref(false)
const pasteText = ref('')
const pasteAgent = ref('')
const savingPaste = ref(false)

const savePasted = async () => {
  let parsed: unknown
  try {
    const raw = pasteText.value.trim().replace(/^```(?:json)?\s*/i, '').replace(/```$/, '')
    parsed = JSON.parse(raw)
  } catch {
    toast.add({ title: 'JSON Tidak Valid', description: 'Pastikan yang ditempel adalah satu objek JSON profil.', icon: 'i-lucide-alert-circle', color: 'warning' })
    return
  }
  savingPaste.value = true
  try {
    const saved = await saveProjectProfile(props.project.id, parsed as Partial<ProjectProfile>, pasteAgent.value.trim() || 'Tempel manual')
    emit('updated', saved)
    pasteOpen.value = false
    pasteText.value = ''
    toast.add({ title: 'Profil Disimpan', description: `${saved.features.length} fitur, ${saved.roadmap.length} butir roadmap.`, icon: 'i-lucide-check-circle', color: 'success' })
  } catch (err: any) {
    toast.add({ title: 'Gagal Menyimpan Profil', description: err.data?.error || err.message, icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    savingPaste.value = false
  }
}
</script>

<template>
  <div
    class="rounded-xl border text-xs"
    :class="sourceInfo.tone === 'sky'
      ? 'border-sky-200 dark:border-sky-900/60 bg-sky-50/60 dark:bg-sky-950/20'
      : 'border-amber-300 dark:border-amber-800/60 bg-amber-50/60 dark:bg-amber-950/20'"
  >
    <!-- Status row -->
    <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
      <div class="flex items-start gap-2.5 min-w-0 flex-1 basis-64">
        <UIcon :name="sourceInfo.icon" class="w-4 h-4 shrink-0 mt-0.5" :class="sourceInfo.tone === 'sky' ? 'text-sky-500' : 'text-amber-500'" />
        <div class="min-w-0">
          <div class="font-semibold text-neutral-800 dark:text-neutral-100">
            {{ sourceInfo.label }}<span v-if="profile.model && profile.source !== 'heuristic'" class="font-mono font-normal text-neutral-500"> · {{ modelName(profile.model) }}</span>
          </div>
          <div class="text-neutral-500 mt-0.5">
            <template v-if="profile.source !== 'heuristic'">
              {{ generatedLabel }} · keyakinan {{ Math.round(profile.confidence * 100) }}%
            </template>
            <template v-else>
              Dikelompokkan dari rute & TODO saja. Isi lewat LLM bawaan atau AI agent Anda untuk hasil akurat.
            </template>
            <span v-if="profile.llm_error" class="block text-rose-600 dark:text-rose-400 truncate" :title="profile.llm_error">LLM gagal: {{ profile.llm_error }}</span>
          </div>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <span v-if="waitingForAgent" class="inline-flex items-center gap-1.5 text-sky-600 dark:text-sky-400">
          <UIcon name="i-lucide-loader-circle" class="w-3.5 h-3.5 animate-spin" />Menunggu agent…
          <button type="button" class="underline cursor-pointer text-neutral-500" @click="stopWaiting">batal</button>
        </span>
        <UButton size="xs" color="neutral" variant="outline" :trailing-icon="showGuide ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" @click="showGuide = !showGuide">
          Isi / Perbarui Profil
        </UButton>
      </div>
    </div>

    <!-- Two ways to fill the profile -->
    <div v-if="showGuide" class="grid grid-cols-1 xl:grid-cols-2 gap-3 px-4 pb-4">
      <section class="rounded-lg border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] p-3.5 space-y-2.5">
        <div class="flex items-center gap-2 font-bold text-neutral-900 dark:text-white">
          <span class="w-5 h-5 rounded-full bg-sky-100 dark:bg-sky-950 text-sky-600 dark:text-sky-400 flex items-center justify-center text-xs">1</span>
          LLM bawaan vibeasik
        </div>
        <p class="text-neutral-600 dark:text-neutral-300 leading-relaxed">
          vibeasik mengumpulkan bukti (README, rute, tabel, TODO, commit) lalu mengirimnya ke LLM yang aktif di <NuxtLink to="/settings" class="text-sky-600 dark:text-sky-400 hover:underline">Pengaturan</NuxtLink>. Semua bagian profil ditulis ulang. ±1–2 menit, tanpa agent.
        </p>
        <UButton icon="i-lucide-sparkles" size="xs" :loading="generating" @click="emit('generate')">
          {{ generating ? (generatingLabel || 'Menganalisis…') : profile.source === 'heuristic' ? 'Analisis dengan LLM' : 'Analisis Ulang dengan LLM' }}
        </UButton>
      </section>

      <section class="rounded-lg border border-neutral-200 dark:border-white/[0.08] bg-white dark:bg-[#04111f] p-3.5 space-y-2.5">
        <div class="flex items-center gap-2 font-bold text-neutral-900 dark:text-white">
          <span class="w-5 h-5 rounded-full bg-sky-100 dark:bg-sky-950 text-sky-600 dark:text-sky-400 flex items-center justify-center text-xs">2</span>
          AI agent Anda via MCP
        </div>
        <p class="text-neutral-600 dark:text-neutral-300 leading-relaxed">
          Claude Code, agy, atau Cursor membaca kode langsung lalu menyimpan lewat tool <code class="font-mono text-sky-600 dark:text-sky-400">save_project_profile</code>. Lebih akurat untuk proyek besar, dan bisa memperbarui sebagian saja.
        </p>
        <ol class="list-decimal list-inside text-neutral-500 space-y-0.5">
          <li>Salin prompt, tempel ke agent yang dibuka di folder proyek.</li>
          <li>Agent menyimpan hasil — halaman ini memperbarui otomatis.</li>
        </ol>
        <div class="flex flex-wrap gap-2">
          <UButton icon="i-lucide-clipboard-copy" size="xs" :loading="copyingPrompt" @click="copyAgentPrompt">
            Salin Prompt Lengkap
          </UButton>
          <UButton icon="i-lucide-clipboard-paste" size="xs" color="neutral" variant="outline" title="Untuk agent tanpa MCP: tempel JSON hasilnya di sini" @click="pasteOpen = true">
            Tempel JSON
          </UButton>
        </div>
        <div class="pt-1 space-y-1">
          <div class="text-xs uppercase font-semibold text-neutral-400">
            Perbarui sebagian (salin perintah singkat)
          </div>
          <div class="flex flex-wrap gap-1.5">
            <button
              v-for="q in quickPrompts"
              :key="q.label"
              type="button"
              class="inline-flex items-center gap-1 px-2 py-1 rounded-md border border-neutral-200 dark:border-neutral-700 hover:border-sky-400 cursor-pointer"
              :title="q.text"
              @click="copyQuickPrompt(q)"
            >
              <UIcon :name="q.icon" class="w-3 h-3 text-neutral-400" />{{ q.label }}
            </button>
          </div>
          <p class="text-xs text-neutral-400">
            MCP belum terpasang di agent? Jalankan <code class="font-mono">vibeasik mcp install</code>.
          </p>
        </div>
      </section>
    </div>

    <UModal v-model:open="pasteOpen" title="Tempel JSON Profil" description="Hasil dari agent yang tidak terhubung MCP. Boleh sebagian, mis. hanya { &quot;roadmap&quot;: [...] }." :ui="{ content: 'sm:max-w-2xl' }">
      <template #body>
        <div class="space-y-3">
          <UInput v-model="pasteAgent" size="sm" placeholder="Nama agent (opsional), mis. Claude Code" class="w-full" />
          <UTextarea v-model="pasteText" :rows="14" placeholder="{ &quot;short_description&quot;: &quot;...&quot;, &quot;features&quot;: [...] }" class="w-full font-mono text-xs" />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2.5 w-full">
          <UButton color="neutral" variant="outline" @click="pasteOpen = false">
            Batal
          </UButton>
          <UButton icon="i-lucide-save" :loading="savingPaste" :disabled="!pasteText.trim()" @click="savePasted">
            Simpan Profil
          </UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
