<script setup lang="ts">
// Isi navigasi sidebar (dipakai sidebar desktop & laci mobile): menu proyek aktif berkelompok
// menurut alur kerja auditor, lalu menu workspace yang kelompoknya bisa dilipat.
import { useActiveProject } from '~/composables/useActiveProject'
import { GLOBAL_NAV_ITEMS, buildProjectNavItems, isProjectNavActive, type NavItem } from '~/utils/sidebarMenuItems'

const props = withDefaults(defineProps<{ collapsed?: boolean, mobile?: boolean }>(), { collapsed: false, mobile: false })
const emit = defineEmits<{ (e: 'navigate'): void }>()

const route = useRoute()
const { activeProjectId, activeProject } = useActiveProject()
// Nama proyek aktif tampil di breadcrumb navbar; di sidebar cukup di tooltip item Ringkasan Proyek.
const { isLocked } = useEdition()
const lockedItem = (item: NavItem) => isLocked(proFeatureForPath(item.to))

const isItemActive = (item: NavItem) => isProjectNavActive(item, route.path, route.query.tab)

const groupItems = (items: NavItem[]) => {
  const out: { name: string, items: NavItem[] }[] = []
  for (const it of items) {
    const name = it.group || ''
    const last = out[out.length - 1]
    if (last && last.name === name) last.items.push(it)
    else out.push({ name, items: [it] })
  }
  return out
}

const projectGroups = computed(() => groupItems(buildProjectNavItems(activeProjectId.value, activeProject.value)))
const globalGroups = computed(() => groupItems(GLOBAL_NAV_ITEMS))

// Kelompok workspace yang dilipat disimpan per peramban; kelompok berisi halaman aktif selalu terbuka.
const folded = ref<Record<string, boolean>>({})
onMounted(() => {
  try {
    folded.value = JSON.parse(localStorage.getItem('vibeasik_sidebar_folded') || '{}')
  } catch {
    folded.value = {}
  }
})
const isOpen = (g: { name: string, items: NavItem[] }) => g.items.some(isItemActive) || !folded.value[g.name]
const toggleGroup = (name: string) => {
  folded.value = { ...folded.value, [name]: !folded.value[name] }
  try {
    localStorage.setItem('vibeasik_sidebar_folded', JSON.stringify(folded.value))
  } catch {
    // penyimpanan peramban tidak tersedia: lipatan hanya berlaku sesi ini
  }
}

const itemClass = (item: NavItem) => [
  props.collapsed ? 'justify-center h-10 w-10 mx-auto' : 'h-9 px-2.5',
  props.mobile ? 'min-h-11' : '',
  isItemActive(item)
    ? 'bg-neutral-900/[0.06] text-neutral-950 dark:bg-white/[0.07] dark:text-white font-semibold'
    : 'text-neutral-600 dark:text-slate-400 hover:bg-neutral-900/[0.04] dark:hover:bg-white/[0.04] hover:text-neutral-900 dark:hover:text-slate-100'
]
</script>

<template>
  <nav class="space-y-4" aria-label="Navigasi utama">
    <!-- Menu proyek aktif -->
    <div v-for="g in projectGroups" :key="'p-' + g.name" class="space-y-0.5">
      <p v-if="!collapsed && g.name" class="px-2.5 pb-1 text-xs font-medium text-neutral-400 dark:text-slate-500">
        {{ g.name }}
      </p>
      <NuxtLink
        v-for="item in g.items"
        :key="item.id"
        :to="item.to"
        class="relative flex items-center gap-2.5 rounded-lg text-sm transition-colors group focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
        :class="itemClass(item)"
        :title="item.desc ? `${item.label} — ${item.desc}` : item.label"
        :aria-current="isItemActive(item) ? 'page' : undefined"
        @click="emit('navigate')"
      >
        <span v-if="isItemActive(item)" class="absolute -left-2 top-2 bottom-2 w-0.5 rounded-full bg-sky-500" aria-hidden="true" />
        <UIcon
          :name="item.icon"
          class="size-[18px] shrink-0"
          :class="isItemActive(item) ? 'text-sky-600 dark:text-sky-400' : 'text-neutral-400 dark:text-slate-500 group-hover:text-neutral-600 dark:group-hover:text-slate-300'"
        />
        <template v-if="!collapsed">
          <span class="truncate">{{ item.label }}</span>
          <UIcon v-if="lockedItem(item)" name="i-lucide-lock" class="ml-auto size-3 shrink-0 text-amber-500" />
          <span
            v-else-if="item.badge && !item.isDetail"
            class="ml-auto inline-flex items-center gap-1 rounded-full px-1.5 text-xs font-medium text-emerald-700 dark:text-emerald-400"
          >
            <span class="size-1.5 rounded-full bg-emerald-500" />{{ item.badge }}
          </span>
        </template>
      </NuxtLink>
    </div>

    <div class="mx-2.5 border-t border-neutral-200/80 dark:border-white/[0.06]" />

    <!-- Menu workspace (lintas proyek) -->
    <div v-for="g in globalGroups" :key="'g-' + g.name" class="space-y-0.5">
      <button
        v-if="!collapsed"
        type="button"
        class="w-full flex items-center justify-between px-2.5 pb-1 text-xs font-medium text-neutral-400 dark:text-slate-500 hover:text-neutral-600 dark:hover:text-slate-300 focus:outline-none focus-visible:text-sky-600"
        :aria-expanded="isOpen(g)"
        @click="toggleGroup(g.name)"
      >
        {{ g.name }}
        <UIcon name="i-lucide-chevron-down" class="size-3.5 transition-transform" :class="isOpen(g) ? '' : '-rotate-90'" />
      </button>
      <template v-if="collapsed || isOpen(g)">
        <NuxtLink
          v-for="item in g.items"
          :key="item.id"
          :to="item.to"
          class="relative flex items-center gap-2.5 rounded-lg text-sm transition-colors group focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
          :class="itemClass(item)"
          :title="item.desc ? `${item.label} — ${item.desc}` : item.label"
          :aria-current="isItemActive(item) ? 'page' : undefined"
          @click="emit('navigate')"
        >
          <span v-if="isItemActive(item)" class="absolute -left-2 top-2 bottom-2 w-0.5 rounded-full bg-sky-500" aria-hidden="true" />
          <UIcon
            :name="item.icon"
            class="size-[18px] shrink-0"
            :class="isItemActive(item) ? 'text-sky-600 dark:text-sky-400' : 'text-neutral-400 dark:text-slate-500 group-hover:text-neutral-600 dark:group-hover:text-slate-300'"
          />
          <template v-if="!collapsed">
            <span class="truncate">{{ item.label }}</span>
            <UIcon v-if="lockedItem(item)" name="i-lucide-lock" class="ml-auto size-3 shrink-0 text-amber-500" />
            <span
              v-else-if="item.badge"
              class="ml-auto inline-flex items-center gap-1 rounded-full px-1.5 text-xs font-medium text-emerald-700 dark:text-emerald-400"
            >
              <span class="size-1.5 rounded-full bg-emerald-500" />{{ item.badge }}
            </span>
          </template>
        </NuxtLink>
      </template>
    </div>
  </nav>
</template>
