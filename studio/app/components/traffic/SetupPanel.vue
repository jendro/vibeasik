<script setup lang="ts">
import type { TrafficSDKInfo, TrafficSource } from '~/types'

// Cara memasang collector di BE live: pilih SDK sesuai stack, tulis berkas ke komponen, isi env (kunci tidak di kode).
const props = defineProps<{ source: TrafficSource }>()
const emit = defineEmits<{ (e: 'rotated'): void }>()

const toast = useToast()
const { getSDK, installSDK, rotateKey } = useTraffic()

const info = ref<TrafficSDKInfo | null>(null)
const kitId = ref('')
const installing = ref(false)
const rotating = ref(false)
const confirmRotate = ref(false)
const showKey = ref(false)

const load = async () => {
  try {
    info.value = await getSDK(props.source.id)
    if (!kitId.value || !info.value.kits.some(k => k.id === kitId.value)) kitId.value = info.value.suggested
  } catch (err: any) {
    toast.add({ title: 'Gagal memuat SDK', description: err?.data?.error || err?.message, color: 'error', icon: 'i-lucide-alert-circle' })
  }
}
watch(() => props.source.id, load, { immediate: true })

const kit = computed(() => info.value?.kits.find(k => k.id === kitId.value) || null)
const kitItems = computed(() => (info.value?.kits || []).map(k => ({ label: k.id === info.value?.suggested ? `${k.label} (disarankan)` : k.label, value: k.id })))
const maskedEnv = computed(() => {
  if (!info.value) return ''
  return showKey.value ? info.value.env : info.value.env.replace(info.value.ingest_key, info.value.ingest_key.slice(0, 8) + '••••••••••••')
})

const copy = async (text: string, label: string) => {
  await navigator.clipboard.writeText(text)
  toast.add({ title: `${label} disalin`, icon: 'i-lucide-clipboard-check', color: 'success' })
}

const install = async (overwrite = false) => {
  if (!kit.value) return
  installing.value = true
  try {
    const res = await installSDK(props.source.id, kit.value.id, overwrite)
    const desc = [
      res.written.length ? `Ditulis: ${res.written.join(', ')}` : '',
      res.skipped.length ? `Sudah ada (tidak ditimpa): ${res.skipped.join(', ')}` : ''
    ].filter(Boolean).join(' · ')
    toast.add({ title: 'SDK dipasang ke komponen', description: desc, icon: 'i-lucide-file-check', color: 'success' })
  } catch (err: any) {
    toast.add({ title: 'Gagal memasang SDK', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    installing.value = false
  }
}

const rotate = async () => {
  rotating.value = true
  try {
    await rotateKey(props.source.id)
    confirmRotate.value = false
    await load()
    emit('rotated')
    toast.add({ title: 'Kunci SDK diganti', description: 'Perbarui VIBEASIK_TRAFFIC_KEY di server live; kunci lama sudah ditolak.', icon: 'i-lucide-key-round', color: 'warning' })
  } catch (err: any) {
    toast.add({ title: 'Gagal mengganti kunci', description: err?.data?.error || err?.message, icon: 'i-lucide-alert-circle', color: 'error' })
  } finally {
    rotating.value = false
  }
}
</script>

<template>
  <div v-if="info" class="space-y-4">
    <!-- Alur data -->
    <div class="flex flex-wrap items-center gap-2 text-xs text-neutral-600 dark:text-neutral-300">
      <span class="px-2 py-1 rounded-lg bg-neutral-100 dark:bg-neutral-800 font-medium">BE live + SDK</span>
      <UIcon name="i-lucide-arrow-right" class="w-3.5 h-3.5 text-neutral-400" />
      <template v-if="source.mode === 'cloud'">
        <span class="px-2 py-1 rounded-lg bg-neutral-100 dark:bg-neutral-800 font-medium">vibeasik live <span class="font-mono font-normal text-neutral-500">{{ source.cloud_url }}</span></span>
        <UIcon name="i-lucide-arrow-right" class="w-3.5 h-3.5 text-neutral-400" />
        <span class="px-2 py-1 rounded-lg bg-neutral-100 dark:bg-neutral-800 font-medium">engine ini (tiap {{ trafficIntervalLabel(source.sync_interval_sec).toLowerCase() }})</span>
      </template>
      <span v-else class="px-2 py-1 rounded-lg bg-neutral-100 dark:bg-neutral-800 font-medium">engine ini (langsung)</span>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-2 gap-4">
      <!-- 1. Berkas SDK -->
      <div class="space-y-2 min-w-0">
        <div class="flex items-center justify-between gap-2">
          <h4 class="text-xs font-bold text-neutral-800 dark:text-neutral-100">
            1 · Pasang SDK di BE
          </h4>
          <USelect v-model="kitId" :items="kitItems" size="xs" class="w-64" />
        </div>
        <template v-if="kit">
          <p class="text-xs text-neutral-500 leading-relaxed">
            {{ kit.notes }}
          </p>
          <div class="flex flex-wrap gap-2">
            <UButton
              size="xs"
              color="primary"
              icon="i-lucide-file-plus-2"
              :loading="installing"
              :disabled="!info.component_dir"
              @click="install(false)"
            >
              Tulis ke {{ info.component_name || 'komponen' }}
            </UButton>
            <UButton v-for="f in kit.files" :key="f.path" size="xs" color="neutral" variant="outline" icon="i-lucide-copy" @click="copy(f.content, f.path)">
              {{ f.path }}
            </UButton>
          </div>
          <p v-if="!info.component_dir" class="text-[11px] text-neutral-500">
            Sumber belum terhubung ke komponen berfolder di laptop ini — salin berkasnya manual.
          </p>
          <div class="text-[11px] font-semibold text-neutral-600 dark:text-neutral-300 pt-1">
            Lalu di entrypoint aplikasi:
          </div>
          <div class="relative">
            <pre class="text-[11px] leading-relaxed font-mono bg-neutral-950 text-neutral-100 rounded-xl p-3 overflow-x-auto">{{ kit.usage }}</pre>
            <UButton size="xs" color="neutral" variant="soft" icon="i-lucide-copy" class="absolute top-2 right-2" aria-label="Salin kode" @click="copy(kit.usage, 'Kode')" />
          </div>
        </template>
      </div>

      <!-- 2. Env -->
      <div class="space-y-2 min-w-0">
        <div class="flex items-center justify-between gap-2">
          <h4 class="text-xs font-bold text-neutral-800 dark:text-neutral-100">
            2 · Isi env di server {{ source.mode === 'cloud' ? 'live' : 'dev' }} <span class="font-mono font-normal text-neutral-500">({{ info.env_file }})</span>
          </h4>
          <div class="flex gap-1">
            <UButton size="xs" color="neutral" variant="ghost" :icon="showKey ? 'i-lucide-eye-off' : 'i-lucide-eye'" :aria-label="showKey ? 'Sembunyikan kunci' : 'Tampilkan kunci'" @click="showKey = !showKey" />
            <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-copy" @click="copy(info.env, 'Env')">
              Salin
            </UButton>
          </div>
        </div>
        <pre class="text-[11px] leading-relaxed font-mono bg-neutral-950 text-neutral-100 rounded-xl p-3 overflow-x-auto whitespace-pre">{{ maskedEnv }}</pre>
        <ul class="text-[11px] text-neutral-500 space-y-1 leading-relaxed list-disc pl-4">
          <li><strong>VIBEASIK_TRAFFIC_FLUSH_SEC</strong> = seberapa sering BE mengirim: 1–2 dtk untuk realtime, 60 untuk per menit. Request pengguna tidak pernah menunggu pengiriman ini.</li>
          <li>Yang dikirim: method, rute, path tanpa query string, status, lama proses, ukuran, IP, negara (header CDN), user agent & host perujuk. Header lain & body tidak pernah dikirim.</li>
          <li>Kunci tidak ditulis ke kode — simpan hanya di env server (ter-ignore git).</li>
        </ul>
        <div class="flex flex-wrap gap-2 pt-1">
          <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-terminal" @click="copy(info.curl_test, 'Perintah uji curl')">
            Salin uji curl
          </UButton>
          <UButton v-if="!confirmRotate" size="xs" color="warning" variant="ghost" icon="i-lucide-key-round" @click="confirmRotate = true">
            Ganti kunci
          </UButton>
          <template v-else>
            <span class="text-[11px] text-amber-700 dark:text-amber-400 self-center">Kunci lama langsung ditolak. Lanjut?</span>
            <UButton size="xs" color="warning" :loading="rotating" @click="rotate">
              Ya, ganti
            </UButton>
            <UButton size="xs" color="neutral" variant="ghost" @click="confirmRotate = false">
              Batal
            </UButton>
          </template>
        </div>
      </div>
    </div>
  </div>
  <div v-else class="text-xs text-neutral-500 flex items-center gap-2">
    <UIcon name="i-lucide-loader-2" class="w-4 h-4 animate-spin" /> Memuat SDK…
  </div>
</template>
