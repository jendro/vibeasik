<script setup lang="ts">
import type { PortRegistryEntry } from '~/types'

const { getPortRegistry, assignPort } = useRunner()
const toast = useToast()

const entries = ref<PortRegistryEntry[]>([])
const range = ref<{ start: number, end: number } | null>(null)
const loading = ref(false)
const allocating = ref<number | null>(null)

const load = async () => {
  loading.value = true
  try {
    const res = await getPortRegistry()
    entries.value = res.entries || []
    range.value = res.range
  } catch {
    entries.value = []
  } finally {
    loading.value = false
  }
}

// Alokasikan port unik tanpa menjalankan komponen (berguna untuk mengisi .env / CORS lebih dulu).
const allocate = async (entry: PortRegistryEntry, reassign: boolean) => {
  allocating.value = entry.component_id
  try {
    const res = await assignPort(entry.component_id, { reassign })
    toast.add({ title: `${entry.component_name} → :${res.run_port}`, icon: 'i-lucide-check', color: 'success' })
    await load()
  } catch (err: any) {
    toast.add({ title: 'Gagal mengalokasikan port', description: err?.data?.error || err?.message, color: 'error' })
  } finally {
    allocating.value = null
  }
}

const stateClass = (e: PortRegistryEntry) => {
  if (e.state === 'running') return 'text-emerald-600 dark:text-emerald-400'
  if (e.state === 'starting') return 'text-amber-600 dark:text-amber-400'
  if (e.listening) return 'text-sky-600 dark:text-sky-400'
  return 'text-neutral-400'
}

const stateLabel = (e: PortRegistryEntry) => {
  if (e.state === 'running' || e.state === 'starting') return e.state.toUpperCase()
  if (e.listening) return 'LISTENING (luar vibeasik)'
  return e.run_port ? 'SIAP' : 'BELUM DIALOKASIKAN'
}

onMounted(load)
defineExpose({ load })
</script>

<template>
  <div class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs overflow-hidden">
    <div class="px-5 py-4 border-b border-neutral-100 dark:border-white/10 flex flex-wrap items-center justify-between gap-2">
      <div>
        <span class="text-xs font-semibold uppercase tracking-wider text-neutral-500">
          Port Registry vibeasik ({{ entries.length }} komponen)
        </span>
        <p class="text-xs text-neutral-400 mt-0.5">
          Port bawaan proyek dipakai bila unik; bila bentrok, vibeasik memberi port tetap dari rentang
          <span class="font-mono">{{ range?.start }}–{{ range?.end }}</span> dan menyuntikkannya sebagai env <span class="font-mono">PORT</span> saat Run.
        </p>
      </div>
      <UButton icon="i-lucide-refresh-cw" color="neutral" variant="outline" size="xs" :loading="loading" @click="load">
        Muat Ulang
      </UButton>
    </div>

    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs">
        <thead>
          <tr class="border-b border-neutral-200 dark:border-white/10 bg-neutral-50/70 dark:bg-slate-800/40 text-neutral-500">
            <th class="py-3 px-4 font-semibold">
              Komponen
            </th>
            <th class="py-3 px-4 font-semibold">
              Port Proyek
            </th>
            <th class="py-3 px-4 font-semibold">
              Port Run (vibeasik)
            </th>
            <th class="py-3 px-4 font-semibold">
              Status
            </th>
            <th class="py-3 px-4 font-semibold text-right">
              Aksi
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800">
          <tr v-for="e in entries" :key="e.component_id" class="hover:bg-neutral-50/50 dark:hover:bg-slate-800/30">
            <td class="py-3 px-4">
              <NuxtLink :to="`/projects/${e.project_id}`" class="font-medium text-neutral-700 dark:text-neutral-300 hover:text-sky-600">
                {{ e.project_name }}
              </NuxtLink>
              <span class="text-neutral-400 font-mono text-xs"> → {{ e.component_name }}</span>
            </td>
            <td class="py-3 px-4 font-mono text-neutral-500">
              {{ e.declared_port ? `:${e.declared_port}` : '—' }}
            </td>
            <td class="py-3 px-4 font-mono font-bold text-neutral-900 dark:text-white">
              {{ e.run_port ? `:${e.run_port}` : '—' }}
              <span v-if="e.run_port && e.declared_port && e.run_port !== e.declared_port" class="ml-1 font-sans font-normal text-xs text-amber-600">
                dipindah
              </span>
            </td>
            <td class="py-3 px-4 font-mono text-xs font-semibold" :class="stateClass(e)">
              {{ stateLabel(e) }}
            </td>
            <td class="py-3 px-4 text-right">
              <UButton
                v-if="!e.run_port"
                size="xs"
                variant="soft"
                color="primary"
                :loading="allocating === e.component_id"
                @click="allocate(e, false)"
              >
                Alokasikan
              </UButton>
              <UButton
                v-else-if="e.state !== 'running' && e.state !== 'starting'"
                size="xs"
                variant="ghost"
                color="neutral"
                icon="i-lucide-shuffle"
                title="Pindahkan ke port bebas lain"
                :loading="allocating === e.component_id"
                @click="allocate(e, true)"
              />
            </td>
          </tr>
          <tr v-if="!entries.length && !loading">
            <td colspan="5" class="py-6 px-4 text-center text-neutral-400">
              Belum ada komponen dengan port.
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
