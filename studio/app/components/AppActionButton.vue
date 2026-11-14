<script setup lang="ts">
// Tombol aksi berjenis (lihat utils/actionKinds): ikon jenis yang sama di semua halaman, penanda "LLM" bila aksinya
// memanggil LLM lewat API, dan tooltip yang menjelaskan akibat klik. `primary` = aksi utama halaman (solid hitam),
// selain itu outline netral — sesuai patokan gaya admin Studio (paling banyak satu aksi utama per bagian).
import { ACTION_KINDS, actionTitle, type ActionKind } from '~/utils/actionKinds'

const props = withDefaults(defineProps<{
  kind: ActionKind
  label: string
  /** penjelasan singkat akibat klik (tooltip) */
  desc?: string
  /** aksi proses yang memanggil LLM (jenis llm selalu dianggap memanggil LLM) */
  llm?: boolean
  primary?: boolean
  size?: 'xs' | 'sm' | 'md'
  loading?: boolean
  disabled?: boolean
  to?: string
  /** ikon pengganti; bawaan = ikon jenis */
  icon?: string
}>(), { llm: false, primary: false, size: 'sm', loading: false, disabled: false, desc: undefined, to: undefined, icon: undefined })

defineEmits<{ click: [e: MouseEvent] }>()

const usesLLM = computed(() => props.kind === 'llm' || props.llm)
const title = computed(() => actionTitle(props.kind, usesLLM.value, props.desc))
</script>

<template>
  <UButton
    :size="size"
    color="neutral"
    :variant="primary ? 'solid' : 'outline'"
    :icon="icon || ACTION_KINDS[kind].icon"
    :loading="loading"
    :disabled="disabled"
    :to="to"
    :title="title"
    :data-action-kind="kind"
    :data-action-llm="usesLLM ? '1' : undefined"
    @click="$emit('click', $event)"
  >
    {{ label }}
    <span
      v-if="usesLLM"
      class="ml-0.5 inline-flex items-center rounded px-1 text-xs font-semibold leading-4"
      :class="primary ? 'bg-white/20 text-white dark:bg-neutral-900/15 dark:text-neutral-900' : 'bg-violet-500/10 text-violet-700 dark:text-violet-300'"
    >LLM</span>
  </UButton>
</template>
