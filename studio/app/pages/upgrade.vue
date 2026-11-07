<script setup lang="ts">
// Penjelasan fitur berbayar yang belum terbuka di build/akun ini.
const route = useRoute()
const { load, upgradeUrl, edition, isInstalled } = useEdition()
await load()

const feature = computed(() => String(route.query.feature || ''))
const info = computed(() => PRO_FEATURES[feature.value])
const others = computed(() => Object.entries(PRO_FEATURES).filter(([k]) => k !== feature.value))
const needsOfficial = computed(() => feature.value && !isInstalled(feature.value))

useHead({ title: `${info.value?.label || 'Fitur Pro'} — vibeasik` })
</script>

<template>
  <div class="max-w-3xl mx-auto py-10 px-4 space-y-8">
    <div class="space-y-3">
      <UBadge color="warning" variant="subtle" icon="i-lucide-lock">
        Fitur Pro
      </UBadge>
      <h1 class="text-2xl font-bold text-neutral-900 dark:text-white flex items-center gap-2">
        <UIcon v-if="info" :name="info.icon" class="w-7 h-7 text-amber-500" />
        {{ info?.label || 'Fitur ini butuh paket Pro' }}
      </h1>
      <p class="text-neutral-600 dark:text-neutral-400">
        {{ info?.description || 'Fitur ini tersedia di paket berbayar vibeasik.' }}
      </p>
      <p class="text-sm text-neutral-500 dark:text-neutral-400">
        <template v-if="needsOfficial">
          Engine ini adalah edisi <b>{{ edition || 'Community' }}</b>. Pasang build Official vibeasik, masuk dengan akun, lalu aktifkan paket untuk membuka fitur ini.
        </template>
        <template v-else>
          Masuk dengan akun vibeasik dan aktifkan paket Pro untuk membuka fitur ini.
        </template>
      </p>
      <div class="flex gap-2">
        <UButton :to="upgradeUrl" target="_blank" color="warning" icon="i-lucide-sparkles">
          Lihat paket di vibeasik.com
        </UButton>
        <UButton v-if="route.query.from" variant="ghost" color="neutral" to="/cockpit" icon="i-lucide-arrow-left">
          Kembali
        </UButton>
      </div>
    </div>

    <div class="space-y-3">
      <h2 class="text-sm font-semibold text-neutral-700 dark:text-neutral-300">
        Fitur Pro lainnya
      </h2>
      <div class="grid sm:grid-cols-2 gap-3">
        <NuxtLink
          v-for="[key, f] in others"
          :key="key"
          :to="{ path: '/upgrade', query: { feature: key } }"
          class="rounded-lg border border-neutral-200 dark:border-white/10 p-3 hover:bg-neutral-50 dark:hover:bg-neutral-800/50 flex gap-3"
        >
          <UIcon :name="f.icon" class="w-5 h-5 shrink-0 text-amber-500 mt-0.5" />
          <div class="min-w-0">
            <div class="text-sm font-medium text-neutral-800 dark:text-neutral-100">{{ f.label }}</div>
            <div class="text-xs text-neutral-500 dark:text-neutral-400 line-clamp-2">{{ f.description }}</div>
          </div>
        </NuxtLink>
      </div>
    </div>
  </div>
</template>
