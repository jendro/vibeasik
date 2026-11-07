<script setup lang="ts">
// Berbagi pengetahuan: persetujuan S&K, mode kirim otomatis, pratinjau, riwayat, dan pencabutan.
// Logika ada di engine (pkg/knowledgesync); halaman ini hanya menampilkan dan memanggil API.
const { apiBase } = useApi()
const toast = useToast()

type Mode = 'auto' | 'ask' | 'off'

interface Sent { local_id: number, hash: string, title: string, sent_at: string, server_id?: number, mode: string, status?: string }
interface Skip { local_id: number, reason: string, at: string }
interface SyncView {
  mode: Mode
  can_control: boolean
  consent: { accepted: boolean, version: string, accepted_version?: string, accepted_at?: string }
  consent_text: string
  logged_in: boolean
  pack: { version?: string, tier?: string, nodes?: number, pulled_at?: string, license?: { id?: string, name?: string, url?: string, notice?: string } }
  limits: { max_per_day: number, min_helpful: number, min_age_days: number }
  ledger: { sent: Sent[], skipped: Skip[], sent_today: number }
}
interface Candidate { id: number, kind: string, domain: string, title: string, content: string }
interface Preview { will_send: Candidate[], skipped: Skip[], max_per_day: number }

const view = ref<SyncView | null>(null)
const preview = ref<Preview | null>(null)
const busy = ref('')
const loadError = ref('')

const errText = (e: any) => e?.data?.error || e?.message || String(e)

const load = async () => {
  try {
    view.value = await $fetch<SyncView>(`${apiBase}/api/knowledge-sync`)
    loadError.value = ''
  } catch (e: any) {
    loadError.value = errText(e)
  }
}

const act = async (key: string, fn: () => Promise<void>, ok?: string) => {
  busy.value = key
  try {
    await fn()
    if (ok) toast.add({ title: ok, color: 'success', icon: 'i-lucide-check' })
  } catch (e: any) {
    toast.add({ title: 'Gagal', description: errText(e), color: 'error' })
  } finally {
    busy.value = ''
  }
}

const accept = () => act('consent', async () => {
  view.value = await $fetch<SyncView>(`${apiBase}/api/knowledge-sync/consent`, {
    method: 'POST', body: { accept: true, version: view.value?.consent.version }
  })
}, 'Persetujuan dicatat')

const setMode = (mode: Mode) => act('mode', async () => {
  view.value = await $fetch<SyncView>(`${apiBase}/api/knowledge-sync/mode`, { method: 'POST', body: { mode } })
}, 'Mode kirim diubah')

const pull = () => act('pull', async () => {
  view.value = await $fetch<SyncView>(`${apiBase}/api/knowledge-sync/pull`, { method: 'POST' })
}, 'Paket pengetahuan diperbarui')

const loadPreview = () => act('preview', async () => {
  preview.value = await $fetch<Preview>(`${apiBase}/api/knowledge-sync/preview`)
})

const syncLedger = () => act('ledger', async () => {
  view.value = await $fetch<SyncView>(`${apiBase}/api/knowledge-sync/ledger/sync`, { method: 'POST' })
}, 'Status diperbarui dari server')

const revoke = (s: Sent) => {
  if (!s.server_id) return
  if (!window.confirm(`Cabut kiriman "${s.title}"? Isinya dihapus dari server (selama belum terbit).`)) return
  act('revoke', async () => {
    view.value = await $fetch<SyncView>(`${apiBase}/api/knowledge-sync/revoke`, { method: 'POST', body: { server_id: s.server_id } })
  }, 'Kiriman dicabut')
}

const canRevoke = (s: Sent) => !!s.server_id && !['approved', 'revoked', 'rejected'].includes(s.status || '')

const fmt = (s?: string) => (s && !s.startsWith('0001')) ? new Date(s).toLocaleString('id-ID') : '-'

const modes: { value: Mode, label: string, desc: string }[] = [
  { value: 'auto', label: 'Otomatis', desc: 'Node yang lolos syarat dikirim sendiri (bawaan)' },
  { value: 'ask', label: 'Tanya dulu', desc: 'Anda menyetujui satu per satu lewat CLI' },
  { value: 'off', label: 'Mati', desc: 'Tidak ada usulan yang dikirim' }
]

const statusColor = (s?: string): 'success' | 'warning' | 'error' | 'neutral' =>
  s === 'approved' ? 'success' : (s === 'rejected' || s === 'revoked') ? 'error' : s === 'screened' ? 'warning' : 'neutral'
const statusLabel = (s?: string) => ({ screened: 'ditinjau', approved: 'terbit', rejected: 'ditolak', revoked: 'dicabut', duplicate: 'duplikat' } as Record<string, string>)[s || ''] || (s || '-')

onMounted(load)
useHead({ title: 'Berbagi Pengetahuan — vibeasik' })
</script>

<template>
  <div class="max-w-3xl mx-auto py-10 px-4 space-y-6">
    <div>
      <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
        Berbagi Pengetahuan
      </h1>
      <p class="text-sm text-neutral-500 dark:text-neutral-400">
        Pusat Pengetahuan bekerja dua arah: Anda menerima paket, dan pokok ilmu Anda yang terbukti berguna ikut memperkaya paket.
        Kode dan data proyek Anda tidak pernah dikirim.
      </p>
    </div>

    <ClientOnly>
      <UAlert v-if="loadError" color="error" variant="subtle" icon="i-lucide-triangle-alert" title="Tidak dapat memuat" :description="loadError" />

      <template v-if="view">
        <!-- Persetujuan -->
        <UCard>
          <div class="space-y-3">
            <div class="flex items-center gap-2">
              <UIcon :name="view.consent.accepted ? 'i-lucide-shield-check' : 'i-lucide-shield-alert'" class="w-6 h-6" :class="view.consent.accepted ? 'text-emerald-500' : 'text-amber-500'" />
              <h2 class="font-semibold text-neutral-900 dark:text-white">
                Syarat &amp; Ketentuan Pusat Pengetahuan
              </h2>
              <UBadge v-if="view.consent.accepted" color="success" variant="subtle">
                disetujui {{ fmt(view.consent.accepted_at) }}
              </UBadge>
            </div>
            <template v-if="!view.consent.accepted">
              <p class="text-sm text-neutral-600 dark:text-neutral-300">
                Pusat Pengetahuan baru aktif setelah Anda membaca dan menyetujui ketentuan berikut.
              </p>
              <pre class="text-xs whitespace-pre-wrap rounded-md bg-neutral-100 dark:bg-neutral-800 p-3 text-neutral-700 dark:text-neutral-200">{{ view.consent_text }}</pre>
              <div class="flex flex-wrap items-center gap-3">
                <UButton :loading="busy === 'consent'" icon="i-lucide-check" @click="accept">
                  Saya setuju
                </UButton>
                <a href="https://vibeasik.com/terms" target="_blank" rel="noopener" class="text-sm text-primary underline">Baca ketentuan lengkap</a>
                <span class="text-xs text-neutral-500">Versi {{ view.consent.version }}</span>
              </div>
            </template>
            <p v-else class="text-sm text-neutral-600 dark:text-neutral-300">
              Versi {{ view.consent.accepted_version }}.
              <a href="https://vibeasik.com/terms" target="_blank" rel="noopener" class="text-primary underline">Baca ketentuan</a>
            </p>
          </div>
        </UCard>

        <!-- Mode -->
        <UCard>
          <div class="space-y-3">
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-send" class="w-6 h-6 text-primary" />
              <h2 class="font-semibold text-neutral-900 dark:text-white">
                Kirim otomatis
              </h2>
              <UBadge variant="subtle" :color="view.mode === 'off' ? 'neutral' : 'primary'">
                {{ view.mode }}
              </UBadge>
            </div>
            <div class="grid gap-2 sm:grid-cols-3">
              <button
                v-for="m in modes"
                :key="m.value"
                type="button"
                class="text-left rounded-lg border p-3 transition disabled:opacity-60"
                :class="view.mode === m.value ? 'border-primary bg-primary/5' : 'border-neutral-200 dark:border-neutral-700'"
                :disabled="!view.can_control || busy === 'mode'"
                @click="view.can_control && view.mode !== m.value && setMode(m.value)"
              >
                <div class="font-medium text-sm">
                  {{ m.label }}
                </div>
                <div class="text-xs text-neutral-500">
                  {{ m.desc }}
                </div>
              </button>
            </div>
            <p v-if="!view.can_control" class="text-xs text-neutral-500 flex items-center gap-1">
              <UIcon name="i-lucide-lock" class="w-3.5 h-3.5" />
              Pada paket Free kirim otomatis adalah bagian dari Pusat Pengetahuan dan tidak dapat dimatikan.
              <NuxtLink to="/upgrade" class="text-primary underline">Paket Pro</NuxtLink> dapat mengubahnya.
            </p>
            <p v-if="!view.logged_in" class="text-xs text-amber-600 flex items-center gap-1">
              <UIcon name="i-lucide-user-x" class="w-3.5 h-3.5" />
              Belum masuk akun: kiriman belum berjalan.
              <NuxtLink to="/account" class="underline">Masuk</NuxtLink>
            </p>
            <p class="text-xs text-neutral-500">
              Syarat kirim: helpful ≥ {{ view.limits.min_helpful }} dan ≥ 3× harmful, umur ≥ {{ view.limits.min_age_days }} hari,
              lolos penyensor dan pemeriksaan privasi. Maksimal {{ view.limits.max_per_day }} per hari
              (hari ini: {{ view.ledger.sent_today }}).
            </p>
          </div>
        </UCard>

        <!-- Paket -->
        <UCard>
          <div class="space-y-2">
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-package" class="w-6 h-6 text-primary" />
              <h2 class="font-semibold text-neutral-900 dark:text-white">
                Paket pengetahuan
              </h2>
            </div>
            <p v-if="view.pack.version" class="text-sm text-neutral-600 dark:text-neutral-300">
              {{ view.pack.version }} ({{ view.pack.tier }}) · {{ view.pack.nodes }} node · ditarik {{ fmt(view.pack.pulled_at) }}
            </p>
            <p v-else class="text-sm text-neutral-500">
              Belum ada paket.
            </p>
            <p v-if="view.pack.license?.id" class="text-xs text-neutral-500">
              Lisensi isi: {{ view.pack.license.name }} ({{ view.pack.license.id }}).
              <a v-if="view.pack.license.url" :href="view.pack.license.url" target="_blank" rel="noopener" class="text-primary underline">Baca</a>
            </p>
            <UButton :loading="busy === 'pull'" :disabled="!view.consent.accepted" icon="i-lucide-download" variant="soft" @click="pull">
              Tarik paket terbaru
            </UButton>
          </div>
        </UCard>

        <!-- Pratinjau -->
        <UCard>
          <div class="space-y-3">
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-eye" class="w-6 h-6 text-primary" />
                <h2 class="font-semibold text-neutral-900 dark:text-white">
                  Pratinjau kiriman
                </h2>
              </div>
              <UButton :loading="busy === 'preview'" variant="soft" icon="i-lucide-refresh-cw" @click="loadPreview">
                Lihat yang akan dikirim
              </UButton>
            </div>
            <p class="text-xs text-neutral-500">
              Daftar ini hanya pratinjau. Tidak ada yang dikirim sampai pengirim otomatis berjalan.
            </p>
            <template v-if="preview">
              <div class="text-sm font-medium">
                Akan dikirim ({{ preview.will_send.length }})
              </div>
              <ul class="space-y-2">
                <li v-for="c in preview.will_send" :key="c.id" class="rounded-md border border-neutral-200 dark:border-neutral-700 p-2">
                  <div class="text-sm font-medium">
                    {{ c.title }}
                  </div>
                  <div class="text-xs text-neutral-500 line-clamp-3">
                    {{ c.content }}
                  </div>
                </li>
                <li v-if="!preview.will_send.length" class="text-xs text-neutral-500">
                  Belum ada node yang memenuhi syarat.
                </li>
              </ul>
              <div v-if="preview.skipped.length" class="text-sm font-medium">
                Dilewati ({{ preview.skipped.length }})
              </div>
              <ul class="text-xs text-neutral-600 dark:text-neutral-400 space-y-1">
                <li v-for="s in preview.skipped" :key="s.local_id + s.reason">
                  #{{ s.local_id }} · {{ s.reason }}
                </li>
              </ul>
            </template>
          </div>
        </UCard>

        <!-- Riwayat -->
        <UCard>
          <div class="space-y-3">
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-history" class="w-6 h-6 text-primary" />
                <h2 class="font-semibold text-neutral-900 dark:text-white">
                  Riwayat kiriman
                </h2>
              </div>
              <UButton :loading="busy === 'ledger'" :disabled="!view.logged_in" variant="soft" icon="i-lucide-refresh-cw" @click="syncLedger">
                Perbarui status
              </UButton>
            </div>
            <div v-if="view.ledger.sent.length" class="overflow-x-auto">
              <table class="w-full text-xs">
                <thead class="text-left text-neutral-500">
                  <tr>
                    <th class="py-1 pr-2">
                      Waktu
                    </th><th class="pr-2">
                      Judul
                    </th><th class="pr-2">
                      Mode
                    </th><th class="pr-2">
                      Status
                    </th><th />
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="s in [...view.ledger.sent].reverse()" :key="s.hash" class="border-t border-neutral-100 dark:border-neutral-800">
                    <td class="py-1 pr-2 whitespace-nowrap">
                      {{ fmt(s.sent_at) }}
                    </td>
                    <td class="pr-2">
                      {{ s.title }}
                    </td>
                    <td class="pr-2">
                      {{ s.mode }}
                    </td>
                    <td class="pr-2">
                      <UBadge size="sm" variant="subtle" :color="statusColor(s.status)">
                        {{ statusLabel(s.status) }}
                      </UBadge>
                    </td>
                    <td class="text-right">
                      <UButton v-if="canRevoke(s)" size="xs" color="error" variant="ghost" :loading="busy === 'revoke'" @click="revoke(s)">
                        Cabut
                      </UButton>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-else class="text-sm text-neutral-500">
              Belum ada kiriman.
            </p>
            <details v-if="view.ledger.skipped.length" class="text-xs text-neutral-600 dark:text-neutral-400">
              <summary class="cursor-pointer">
                Node yang dilewati ({{ view.ledger.skipped.length }})
              </summary>
              <ul class="mt-2 space-y-1">
                <li v-for="s in view.ledger.skipped" :key="s.local_id + s.reason + s.at">
                  {{ fmt(s.at) }} · #{{ s.local_id }} · {{ s.reason }}
                </li>
              </ul>
            </details>
          </div>
        </UCard>
      </template>
    </ClientOnly>
  </div>
</template>
