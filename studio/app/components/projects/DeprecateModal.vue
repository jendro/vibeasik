<script setup lang="ts">
import type { Project } from '~/types'

const props = defineProps<{
  project: Project | null
}>()

const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{
  updated: [project: Project]
}>()

const { deprecateProject } = useApi()
const toast = useToast()

const reason = ref('')
const loading = ref(false)

watch(() => props.project, (p) => {
  if (p) {
    reason.value = p.deprecated_reason || ''
  }
}, { immediate: true })

const handleSubmit = async () => {
  if (!props.project?.id) return
  loading.value = true
  try {
    const res = await deprecateProject(props.project.id, reason.value)
    toast.add({
      title: 'Proyek Ditandai Deprecated',
      description: `Proyek '${props.project.name}' kini berstatus deprecated dan disembunyikan dari daftar aktif default.`,
      icon: 'i-lucide-archive',
      color: 'warning'
    })
    emit('updated', res.project || { ...props.project, status: 'deprecated', deprecated_reason: reason.value } as Project)
    open.value = false
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menandai Deprecated',
      description: err.message || 'Terjadi kesalahan saat memperbarui status proyek',
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" :title="`Tandai Deprecated: ${project?.name || ''}`">
    <template #content>
      <div class="p-6 space-y-4">
        <div class="flex items-start gap-3 p-3.5 rounded-xl border border-amber-500/30 bg-amber-50/50 dark:bg-amber-950/20 text-amber-900 dark:text-amber-200 text-xs leading-relaxed">
          <UIcon name="i-lucide-alert-triangle" class="w-5 h-5 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
          <div>
            <p class="font-semibold text-amber-800 dark:text-amber-300">
              Konfirmasi Status Deprecated
            </p>
            <p class="mt-0.5 text-neutral-600 dark:text-neutral-300">
              Menandai proyek sebagai deprecated akan meredupkan tampilannya di katalog dan menyembunyikannya dari filter aktif default. File asli di komputer Anda tidak akan dihapus.
            </p>
          </div>
        </div>

        <div class="space-y-1.5">
          <label class="block text-xs font-semibold text-neutral-700 dark:text-neutral-300">
            Alasan Deprecated (Opsional)
          </label>
          <UTextarea
            v-model="reason"
            placeholder="Contoh: Digantikan oleh proyek versi baru (bejamERP2), eksperimen selesai, atau kode tidak lagi dipelihara."
            :rows="3"
            class="w-full text-xs"
          />
        </div>

        <div class="flex items-center justify-end gap-2 pt-2 border-t border-neutral-100 dark:border-neutral-800">
          <UButton
            color="neutral"
            variant="outline"
            size="sm"
            :disabled="loading"
            @click="open = false"
          >
            Batal
          </UButton>
          <UButton
            color="warning"
            variant="solid"
            size="sm"
            icon="i-lucide-archive"
            :loading="loading"
            @click="handleSubmit"
          >
            Tandai Sebagai Deprecated
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
