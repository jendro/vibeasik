<script setup lang="ts">
import type { TrafficGeoStatus, TrafficKV, TrafficReport } from '~/types'

// "Siapa pengunjungnya": kota, ISP/organisasi, dan jenis jaringan (seluler, rumahan, datacenter, VPN).
// Dibaca engine dari basis data offline di mesin ini; jenis jaringan berasal dari ASN, bukan dari perangkat
// pengunjung, jadi merek router atau perangkat tidak pernah bisa diketahui.
const props = defineProps<{ report: TrafficReport }>()

const { getGeoStatus } = useTraffic()
const status = ref<TrafficGeoStatus | null>(null)
const state = ref<'loading' | 'ready' | 'error'>('loading')

onMounted(async () => {
  try {
    status.value = await getGeoStatus()
    state.value = 'ready'
  } catch {
    state.value = 'error'
  }
})

const netLabel: Record<string, { label: string, icon: string, hint: string }> = {
  mobile: { label: 'Seluler', icon: 'i-lucide-smartphone', hint: 'Operator seluler' },
  residential: { label: 'Rumahan', icon: 'i-lucide-house', hint: 'ISP rumah/kantor' },
  hosting: { label: 'Datacenter', icon: 'i-lucide-server', hint: 'Hosting & cloud' },
  vpn: { label: 'VPN / proxy', icon: 'i-lucide-shield', hint: 'Penyedia VPN atau proxy' },
  unknown: { label: 'Tidak dikenal', icon: 'i-lucide-circle-help', hint: 'ASN belum diklasifikasi' }
}

const cities = computed(() => props.report.cities || [])
const orgs = computed(() => props.report.orgs || [])
const netTypes = computed(() => (props.report.net_types || []).map(n => ({ ...n, ...(netLabel[n.value] || netLabel.unknown) })))
const hasData = computed(() => cities.value.length > 0 || orgs.value.length > 0 || netTypes.value.length > 0)
const share = (kv: TrafficKV) => Math.min(100, kv.share * 100)
</script>

<template>
  <section class="rounded-xl border border-neutral-200 dark:border-neutral-800 p-3 space-y-3" aria-label="Lokasi dan jenis jaringan pengunjung">
    <div class="text-xs font-bold text-neutral-800 dark:text-neutral-100 flex items-center gap-1.5">
      <UIcon name="i-lucide-map-pinned" class="w-4 h-4 text-neutral-400" /> Lokasi &amp; jenis jaringan
      <span class="font-normal text-neutral-500">· dibaca offline di mesin ini, IP tidak dikirim ke pihak lain</span>
    </div>

    <div v-if="state === 'loading'" class="grid grid-cols-1 md:grid-cols-3 gap-3" aria-busy="true">
      <USkeleton v-for="i in 3" :key="i" class="h-28 rounded-xl" />
    </div>

    <div v-else-if="state === 'error'" class="flex items-center gap-2 text-xs text-rose-600 dark:text-rose-400" role="alert">
      <UIcon name="i-lucide-circle-x" class="w-4 h-4" />
      Status basis data lokasi tidak bisa dibaca dari engine.
    </div>

    <div v-else-if="!hasData" class="text-xs text-neutral-500 space-y-1">
      <template v-if="status && !status.city_ready && !status.asn_ready">
        <p>Basis data lokasi belum terpasang, jadi kota dan ISP belum bisa ditampilkan.</p>
        <p>
          Pasang dengan <code class="font-mono px-1 rounded bg-neutral-100 dark:bg-neutral-800">{{ status.update_cmd }}</code>
          <span v-if="status.auto_update">(engine juga mengunduhnya sendiri tiap bulan).</span>
          <span v-else>(unduh otomatis dimatikan lewat VIBEASIK_GEO_AUTO).</span>
        </p>
      </template>
      <p v-else>
        Belum ada pengunjung ber-IP publik di jendela ini. Event lama yang masuk sebelum basis data terpasang tidak ikut diperkaya.
      </p>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <div class="rounded-xl border border-neutral-200 dark:border-neutral-800 p-3">
        <div class="text-xs font-bold text-neutral-800 dark:text-neutral-100 flex items-center gap-1.5 mb-2">
          <UIcon name="i-lucide-building-2" class="w-4 h-4 text-neutral-400" /> Kota
        </div>
        <div v-if="cities.length" class="space-y-1.5">
          <div v-for="c in cities.slice(0, 8)" :key="c.value" class="text-xs">
            <div class="flex justify-between gap-2">
              <span class="truncate text-neutral-700 dark:text-neutral-200" :title="c.value">{{ c.value }}</span>
              <span class="tabular-nums text-neutral-500 shrink-0">{{ fmtNum(c.count) }} · {{ fmtPct(c.share) }}</span>
            </div>
            <div class="mt-0.5 h-1 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
              <div class="h-full rounded-full bg-primary-500 dark:bg-primary-400" :style="{ width: share(c) + '%' }" />
            </div>
          </div>
        </div>
        <p v-else class="text-xs text-neutral-500">
          Kota belum tersedia (basis data City belum terpasang).
        </p>
      </div>

      <div class="rounded-xl border border-neutral-200 dark:border-neutral-800 p-3">
        <div class="text-xs font-bold text-neutral-800 dark:text-neutral-100 flex items-center gap-1.5 mb-2">
          <UIcon name="i-lucide-network" class="w-4 h-4 text-neutral-400" /> ISP / organisasi
        </div>
        <div v-if="orgs.length" class="space-y-1.5">
          <div v-for="o in orgs.slice(0, 8)" :key="o.value" class="text-xs">
            <div class="flex justify-between gap-2">
              <span class="truncate text-neutral-700 dark:text-neutral-200" :title="o.value">
                {{ o.value }}
              </span>
              <span class="tabular-nums text-neutral-500 shrink-0">{{ fmtNum(o.count) }} · {{ fmtPct(o.share) }}</span>
            </div>
            <div class="mt-0.5 h-1 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
              <div class="h-full rounded-full bg-primary-500 dark:bg-primary-400" :style="{ width: share(o) + '%' }" />
            </div>
          </div>
        </div>
        <p v-else class="text-xs text-neutral-500">
          ISP belum tersedia (basis data ASN belum terpasang).
        </p>
      </div>

      <div class="rounded-xl border border-neutral-200 dark:border-neutral-800 p-3">
        <div class="text-xs font-bold text-neutral-800 dark:text-neutral-100 flex items-center gap-1.5 mb-2">
          <UIcon name="i-lucide-signal" class="w-4 h-4 text-neutral-400" /> Jenis jaringan
        </div>
        <div v-if="netTypes.length" class="space-y-1.5">
          <div v-for="n in netTypes" :key="n.value" class="text-xs">
            <div class="flex justify-between gap-2">
              <span class="flex items-center gap-1.5 text-neutral-700 dark:text-neutral-200" :title="n.hint">
                <UIcon :name="n.icon" class="w-3.5 h-3.5 text-neutral-400" /> {{ n.label }}
              </span>
              <span class="tabular-nums text-neutral-500 shrink-0">{{ fmtNum(n.count) }} · {{ fmtPct(n.share) }}</span>
            </div>
            <div class="mt-0.5 h-1 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
              <div class="h-full rounded-full bg-primary-500 dark:bg-primary-400" :style="{ width: share(n) + '%' }" />
            </div>
          </div>
        </div>
        <p v-else class="text-xs text-neutral-500">
          Jenis jaringan belum tersedia.
        </p>
        <p class="mt-2 text-xs text-neutral-400">
          Dihitung dari ASN pengunjung, bukan dari perangkat mereka.
        </p>
      </div>
    </div>
  </section>
</template>
