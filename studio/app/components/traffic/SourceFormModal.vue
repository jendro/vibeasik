<script setup lang="ts">
import type { TrafficMeta, TrafficMode, TrafficSource } from '~/types'

// Hubungkan satu BE live ke Traffic Analyzer. Mode cloud: BE live → vibeasik live → engine ini menarik berkala.
// Mode direct: BE (dev/staging di jaringan yang sama) mengirim langsung ke engine ini.
const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ meta: TrafficMeta | null }>()
const emit = defineEmits<{ (e: 'created', src: TrafficSource): void }>()

const toast = useToast()
const { createSource } = useTraffic()
const { projects, activeProjectId } = useActiveProject()

const mode = ref<TrafficMode>('cloud')
const name = ref('')
const projectId = ref<number | undefined>(undefined)
const componentId = ref<number | undefined>(undefined)
const cloudURL = ref('')
const interval = ref(60)
const saving = ref(false)

const projectItems = computed(() => projects.value.map(p => ({ label: p.name, value: p.id })))
const componentItems = computed(() => {
  const p = projects.value.find(x => x.id === projectId.value)
  const comps = (p?.components || []).filter(c => c.role === 'backend' || c.role === 'other' || c.role?.includes('service'))
  return (comps.length ? comps : p?.components || []).map(c => ({ label: `${c.name} (${c.role})`, value: c.id }))
})
const intervalItems = computed(() => (props.meta?.sync_intervals || [5, 60, 300]).map(v => ({ label: trafficIntervalLabel(v), value: v })))

watch(open, (v) => {
  if (!v) return
  mode.value = 'cloud'
  name.value = ''
  projectId.value = activeProjectId.value || undefined
  componentId.value = undefined
  cloudURL.value = props.meta?.default_cloud_url || ''
  interval.value = 60
})
watch(projectId, () => {
  componentId.value = componentItems.value[0]?.value
})

const isLocalCloud = computed(() => /localhost|127\.0\.0\.1/.test(cloudURL.value))

const submit = async () => {
  saving.value = true
  try {
    const src = await createSource({
      name: name.value.trim() || undefined,
      project_id: projectId.value ?? null,
      component_id: componentId.value ?? null,
      mode: mode.value,
      cloud_url: mode.value === 'cloud' ? cloudURL.value.trim() : undefined,
      sync_interval_sec: interval.value
    })
    toast.add({ title: 'Sumber traffic dibuat', description: 'Pasang SDK di BE live lalu isi env-nya.', icon: 'i-lucide-activity', color: 'success' })
    emit('created', src)
    open.value = false
  } catch (err: any) {
    toast.add({ title: 'Gagal membuat sumber', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" title="Hubungkan Aplikasi Live" description="Statistik request BE live dikirim ke vibeasik, lalu dianalisis di sini." :ui="{ content: 'sm:max-w-2xl' }">
    <template #body>
      <div class="space-y-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
          <button
            v-for="opt in [
              { v: 'cloud', icon: 'i-lucide-cloud', title: 'Lewat vibeasik live', desc: 'Server produksi → vibeasik live → laptop/server dev ini. Untuk aplikasi di internet.' },
              { v: 'direct', icon: 'i-lucide-plug', title: 'Langsung ke engine ini', desc: 'BE dev/staging yang bisa menjangkau engine ini (LAN/localhost).' }
            ]"
            :key="opt.v"
            type="button"
            class="text-left p-3 rounded-xl border transition-colors cursor-pointer"
            :class="mode === opt.v ? 'border-primary-400 bg-primary-50/60 dark:bg-primary-950/30' : 'border-neutral-200 dark:border-white/[0.08] hover:border-neutral-300'"
            @click="mode = opt.v as TrafficMode"
          >
            <div class="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white">
              <UIcon :name="opt.icon" class="w-4 h-4" /> {{ opt.title }}
            </div>
            <p class="text-xs text-neutral-500 mt-1 leading-relaxed">
              {{ opt.desc }}
            </p>
          </button>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <UFormField label="Proyek">
            <USelect v-model="projectId" :items="projectItems" placeholder="Pilih proyek" class="w-full" />
          </UFormField>
          <UFormField label="Komponen BE" help="Menentukan SDK yang disarankan & folder pemasangan.">
            <USelect v-model="componentId" :items="componentItems" placeholder="Pilih komponen" class="w-full" :disabled="!projectId" />
          </UFormField>
        </div>
        <UFormField label="Nama sumber" help="Kosong = nama komponen. Bedakan bila satu BE punya beberapa lingkungan (prod, staging).">
          <UInput v-model="name" placeholder="mis. API produksi" class="w-full" />
        </UFormField>

        <template v-if="mode === 'cloud'">
          <UFormField label="URL vibeasik live" help="Alamat publik cloud-server vibeasik yang bisa dijangkau server produksi.">
            <UInput v-model="cloudURL" placeholder="https://vibeasik.com" class="w-full" icon="i-lucide-globe" />
          </UFormField>
          <p v-if="isLocalCloud" class="text-xs text-amber-700 dark:text-amber-400 bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-900/60 rounded-lg p-2.5 leading-relaxed">
            URL ini localhost: hanya cocok untuk uji coba. Server produksi di internet tidak bisa menjangkau laptop ini — isi dengan URL vibeasik live yang publik.
          </p>
          <UFormField label="Tarik data dari vibeasik live tiap" help="Realtime = tiap 5 detik. Interval kirim SDK diatur terpisah lewat env VIBEASIK_TRAFFIC_FLUSH_SEC.">
            <USelect v-model="interval" :items="intervalItems" class="w-full" />
          </UFormField>
        </template>
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton color="neutral" variant="ghost" @click="open = false">
          Batal
        </UButton>
        <UButton color="primary" icon="i-lucide-link" :loading="saving" :disabled="mode === 'cloud' && !cloudURL" @click="submit">
          Hubungkan
        </UButton>
      </div>
    </template>
  </UModal>
</template>
