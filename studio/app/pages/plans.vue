<script setup lang="ts">
// Perbandingan paket Free dan Pro, plus status akun saat ini. Daftar fitur Pro dari utils/proFeatures.ts.
const { account, load: loadAccount } = useAuth()
const { load: loadEdition, isLocked, upgradeUrl } = useEdition()
onMounted(() => {
  loadAccount()
  loadEdition()
})

const isPro = computed(() => Boolean(account.value?.logged_in && account.value.plan !== 'free'))

// Fitur yang ada di kedua paket.
const freeFeatures = [
  'Scan proyek & katalog stack',
  '85 rule audit',
  'Port & runner, monitor port bentrok',
  'Peta kode (indeks untuk agent AI)',
  'Todo & perencanaan',
  'Traffic Analyzer & Delivery',
  'MCP: 64 tool',
  'Pusat Pengetahuan: paket dasar'
]

// Baris perbandingan tambahan di luar daftar fitur berbayar.
const rows = [
  { label: 'Akun', free: 'Tidak wajib', pro: 'Wajib (login perangkat)' },
  { label: 'Tool MCP', free: '64', pro: '123' },
  { label: 'Paket pengetahuan', free: 'Dasar (free)', pro: 'Lengkap + tautan antarnode (full)' },
  { label: 'Kirim otomatis pengetahuan', free: 'Bawaan menyala, terkunci', pro: 'Bisa diubah: auto / tanya / mati' }
]

const proList = Object.entries(PRO_FEATURES).map(([key, f]) => ({ key, ...f }))
useHead({ title: 'Free vs Pro — vibeasik' })
</script>

<template>
  <div class="max-w-4xl mx-auto py-10 px-4 space-y-8">
    <div class="space-y-2">
      <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
        Free vs Pro
      </h1>
      <p class="text-sm text-neutral-500 dark:text-neutral-400">
        Semua yang berjalan di laptop Anda tetap Free. Pro membuka modul AI dan jaring pengetahuan lengkap.
      </p>
      <ClientOnly>
        <div class="flex items-center gap-2 text-sm">
          Paket Anda:
          <UBadge :color="isPro ? 'warning' : 'neutral'" variant="subtle">
            {{ isPro ? account?.plan : 'free' }}
          </UBadge>
          <UButton v-if="!account?.logged_in" to="/account" size="xs" variant="soft" icon="i-lucide-log-in">
            Masuk akun
          </UButton>
        </div>
      </ClientOnly>
    </div>

    <div class="grid md:grid-cols-2 gap-4">
      <UCard>
        <template #header>
          <div class="font-semibold">
            Free
          </div>
        </template>
        <ul class="space-y-1.5 text-sm text-neutral-700 dark:text-neutral-300">
          <li v-for="f in freeFeatures" :key="f" class="flex gap-2">
            <UIcon name="i-lucide-check" class="w-4 h-4 mt-0.5 text-emerald-500 shrink-0" /> {{ f }}
          </li>
        </ul>
      </UCard>
      <UCard>
        <template #header>
          <div class="font-semibold flex items-center gap-2">
            Pro <UBadge color="warning" variant="subtle" size="xs">
              semua fitur Free +
            </UBadge>
          </div>
        </template>
        <ul class="space-y-2 text-sm">
          <li v-for="f in proList" :key="f.key" class="flex gap-2">
            <UIcon :name="f.icon" class="w-4 h-4 mt-0.5 text-amber-500 shrink-0" />
            <div class="min-w-0">
              <div class="font-medium text-neutral-800 dark:text-neutral-100 flex items-center gap-1.5">
                {{ f.label }}
                <UIcon v-if="isLocked(f.key)" name="i-lucide-lock" class="w-3 h-3 text-amber-500" />
                <UIcon v-else name="i-lucide-check" class="w-3.5 h-3.5 text-emerald-500" />
              </div>
              <div class="text-xs text-neutral-500 dark:text-neutral-400">
                {{ f.description }}
              </div>
            </div>
          </li>
        </ul>
      </UCard>
    </div>

    <UCard>
      <template #header>
        <div class="font-semibold">
          Ringkasan perbedaan
        </div>
      </template>
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs text-neutral-500">
            <th class="py-1 pr-3" />
            <th class="py-1 pr-3">
              Free
            </th>
            <th class="py-1">
              Pro
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in rows" :key="r.label" class="border-t border-neutral-200 dark:border-white/10">
            <td class="py-2 pr-3 font-medium">
              {{ r.label }}
            </td>
            <td class="py-2 pr-3">
              {{ r.free }}
            </td>
            <td class="py-2">
              {{ r.pro }}
            </td>
          </tr>
        </tbody>
      </table>
    </UCard>

    <div class="flex gap-2">
      <UButton :to="upgradeUrl" target="_blank" color="warning" icon="i-lucide-sparkles">
        Lihat paket di vibeasik.com
      </UButton>
      <UButton to="/account" variant="soft" icon="i-lucide-user-round">
        Akun & masuk
      </UButton>
    </div>
  </div>
</template>
