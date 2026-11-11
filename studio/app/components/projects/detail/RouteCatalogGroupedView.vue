<script setup lang="ts">
import type { ComponentRoute } from '~/types'

defineProps<{
  groupedRoutes: { domain: string, routes: ComponentRoute[] }[]
  isFrontend: boolean
  expandedRequests: Record<number, boolean>
  copiedPath: string | null
}>()
const emit = defineEmits<{
  toggleExpand: [routeId: number]
  copyPath: [path: string]
  edit: [route: ComponentRoute]
  delete: [route: ComponentRoute]
}>()
const { getMethodBadgeColor } = useFormatters()
const toggleExpandRequests = (routeId: number) => emit('toggleExpand', routeId)
const copyRoutePath = (path: string) => emit('copyPath', path)
const openEditModal = (route: ComponentRoute) => emit('edit', route)
const handleDelete = (route: ComponentRoute) => emit('delete', route)
</script>

<template>
  <div class="space-y-4">
    <div
      v-for="group in groupedRoutes"
      :key="group.domain"
      class="rounded-xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-[#04111f] overflow-hidden"
    >
      <!-- Group Header -->
      <div class="p-3.5 bg-neutral-50/80 dark:bg-neutral-800 border-b border-neutral-200 dark:border-white/[0.08] flex items-center justify-between">
        <div class="flex items-center gap-2">
          <UIcon :name="isFrontend ? 'i-lucide-panels-top-left' : 'i-lucide-folder-git-2'" class="w-4 h-4 text-primary-500" />
          <span class="font-bold text-xs text-neutral-900 dark:text-white uppercase tracking-wider">
            {{ group.domain }}
          </span>
          <UBadge size="xs" color="neutral" variant="subtle" class="font-semibold">
            {{ group.routes.length }} {{ isFrontend ? 'halaman' : 'endpoint' }}
          </UBadge>
        </div>
      </div>

      <!-- Table within Group -->
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-neutral-50/40 dark:bg-neutral-800/30 text-neutral-400 uppercase text-xs tracking-wider border-b border-neutral-100 dark:border-neutral-800/80">
            <tr v-if="isFrontend">
              <th class="py-2.5 px-4 w-20">
                Tipe
              </th>
              <th class="py-2.5 px-4">
                Rute Halaman
              </th>
              <th class="py-2.5 px-4 min-w-[240px]">
                Request API yang Dipanggil
              </th>
              <th class="py-2.5 px-4 w-36">
                Sub-Fitur
              </th>
              <th class="py-2.5 px-4">
                Keterangan Halaman
              </th>
              <th class="py-2.5 px-4 text-right w-44">
                File Sumber
              </th>
              <th class="py-2.5 px-4 text-center w-20">
                Aksi
              </th>
            </tr>
            <tr v-else>
              <th class="py-2.5 px-4 w-24">
                Method
              </th>
              <th class="py-2.5 px-4">
                Endpoint Path & Middleware
              </th>
              <th class="py-2.5 px-4 w-44">
                Kelompok Fungsi
              </th>
              <th class="py-2.5 px-4">
                Keterangan / Penjelasan
              </th>
              <th class="py-2.5 px-4 text-right w-44">
                Handler
              </th>
              <th class="py-2.5 px-4 text-center w-20">
                Aksi
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800/80">
            <tr
              v-for="r in group.routes"
              :key="r.id"
              class="hover:bg-neutral-50/60 dark:hover:bg-neutral-800/40 transition-colors"
            >
              <!-- Method / Type Badge -->
              <td class="py-3 px-4 whitespace-nowrap align-top">
                <UBadge
                  size="xs"
                  :color="getMethodBadgeColor(r.method)"
                  variant="solid"
                  class="font-mono font-bold shrink-0 tracking-wider"
                >
                  {{ r.method }}
                </UBadge>
              </td>

              <!-- Path & Middleware -->
              <td class="py-3 px-4 align-top">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-mono font-semibold text-neutral-900 dark:text-neutral-100 select-all">
                    {{ r.path }}
                  </span>
                  <button
                    type="button"
                    class="text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 cursor-pointer transition-colors"
                    :title="isFrontend ? 'Salin rute halaman' : 'Salin path endpoint'"
                    @click="copyRoutePath(r.path)"
                  >
                    <UIcon
                      :name="copiedPath === r.path ? 'i-lucide-check' : 'i-lucide-copy'"
                      class="w-3.5 h-3.5"
                      :class="copiedPath === r.path ? 'text-sky-400' : ''"
                    />
                  </button>
                  <span
                    v-if="r.middleware"
                    class="text-xs px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 font-mono text-neutral-500 shrink-0"
                  >
                    {{ r.middleware }}
                  </span>
                </div>
                <div v-if="r.tags" class="text-xs text-neutral-400 mt-0.5">
                  Tags: {{ r.tags }}
                </div>
              </td>

              <!-- Target Requests (Only Frontend) -->
              <td v-if="isFrontend" class="py-3 px-4 align-top">
                <div v-if="parseTargetRequests(r.target_requests).length > 0" class="space-y-1.5 max-w-sm">
                  <div class="flex flex-wrap gap-1">
                    <span
                      v-for="(req, idx) in (expandedRequests[r.id] ? parseTargetRequests(r.target_requests) : parseTargetRequests(r.target_requests).slice(0, 3))"
                      :key="idx"
                      class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-mono border"
                      :class="[getRequestMethodColor(req).bg, getRequestMethodColor(req).text, getRequestMethodColor(req).border]"
                    >
                      <span class="font-bold shrink-0">{{ req.split(' ')[0] }}</span>
                      <span class="truncate max-w-[200px]" :title="req">{{ req.substring(req.indexOf(' ') + 1) || req }}</span>
                    </span>
                  </div>
                  <button
                    v-if="parseTargetRequests(r.target_requests).length > 3"
                    type="button"
                    class="text-xs font-semibold text-primary-500 hover:text-primary-600 dark:hover:text-primary-400 cursor-pointer block mt-1"
                    @click="toggleExpandRequests(r.id)"
                  >
                    {{ expandedRequests[r.id] ? 'Tutup' : `+${parseTargetRequests(r.target_requests).length - 3} request lainnya` }}
                  </button>
                </div>
                <span v-else class="text-neutral-400 italic text-xs">
                  -
                </span>
              </td>

              <!-- Kelompok Fungsi / Sub-Fitur -->
              <td class="py-3 px-4 align-top">
                <UBadge size="xs" color="neutral" variant="outline" class="font-medium text-xs">
                  {{ r.function_group || (isFrontend ? 'Halaman' : 'Umum') }}
                </UBadge>
              </td>

              <!-- Keterangan / Deskripsi -->
              <td class="py-3 px-4 align-top text-neutral-700 dark:text-neutral-300 leading-relaxed text-xs">
                <p v-if="r.description">
                  {{ r.description }}
                </p>
                <p v-else class="text-neutral-400 italic text-xs">
                  {{ isFrontend ? 'Belum ada keterangan. Klik "Kirim ke LLM" untuk membuat keterangan halaman otomatis.' : 'Belum ada keterangan. Klik "Kirim ke LLM" untuk membuat keterangan otomatis.' }}
                </p>
              </td>

              <!-- Handler / File Sumber -->
              <td class="py-3 px-4 text-right align-top font-mono text-xs text-neutral-500 truncate max-w-xs">
                <span :title="r.handler">{{ r.handler || '-' }}</span>
              </td>

              <!-- Actions -->
              <td class="py-3 px-4 text-center align-top whitespace-nowrap">
                <div class="flex items-center justify-center gap-1">
                  <button
                    type="button"
                    class="p-1 rounded text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 cursor-pointer"
                    :title="isFrontend ? 'Edit keterangan dan modul halaman' : 'Edit keterangan dan domain'"
                    @click="openEditModal(r)"
                  >
                    <UIcon name="i-lucide-pencil" class="w-3.5 h-3.5" />
                  </button>
                  <button
                    type="button"
                    class="p-1 rounded text-neutral-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/40 cursor-pointer"
                    :title="isFrontend ? 'Hapus halaman dari database' : 'Hapus rute dari database'"
                    @click="handleDelete(r)"
                  >
                    <UIcon name="i-lucide-trash-2" class="w-3.5 h-3.5" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
