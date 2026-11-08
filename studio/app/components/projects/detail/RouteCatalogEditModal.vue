<script setup lang="ts">
import type { ComponentRoute } from '~/types'

// Dialog edit keterangan/kelompok satu rute atau halaman.
const props = defineProps<{ componentId: number | undefined, isFrontend: boolean, loading: boolean }>()
const emit = defineEmits<{ saved: [route: ComponentRoute] }>()

const toast = useToast()
const { updateComponentRoute } = useApi()
const { getMethodBadgeColor } = useFormatters()

const editModalOpen = ref(false)
const editingRoute = ref<ComponentRoute | null>(null)
const editForm = ref({
  domain: '',
  function_group: '',
  description: '',
  tags: '',
  target_requests: ''
})
const savingEdit = ref(false)

const open = (r: ComponentRoute) => {
  editingRoute.value = r
  editForm.value = {
    domain: r.domain || 'Umum',
    function_group: r.function_group || 'Lainnya',
    description: r.description || '',
    tags: r.tags || '',
    target_requests: r.target_requests ? parseTargetRequests(r.target_requests).join('\n') : ''
  }
  editModalOpen.value = true
}
defineExpose({ open })

const handleSaveEdit = async () => {
  if (!editingRoute.value || !props.componentId) return
  savingEdit.value = true
  try {
    const reqsArray = editForm.value.target_requests
      .split('\n')
      .map(s => s.trim())
      .filter(Boolean)
    const payload = {
      domain: editForm.value.domain,
      function_group: editForm.value.function_group,
      description: editForm.value.description,
      tags: editForm.value.tags,
      target_requests: JSON.stringify(reqsArray)
    }
    const updated = await updateComponentRoute(props.componentId, editingRoute.value.id, payload)
    emit('saved', updated)
    editModalOpen.value = false
  } catch (err: any) {
    toast.add({
      title: 'Gagal Menyimpan Rute',
      description: `Gagal menyimpan perubahan: ${err.message}`,
      icon: 'i-lucide-alert-circle',
      color: 'error'
    })
  } finally {
    savingEdit.value = false
  }
}
</script>

<template>
  <UModal
    v-model:open="editModalOpen"
    :title="isFrontend ? 'Edit Keterangan Halaman & Request API' : 'Edit Keterangan & Kelompok Router'"
    :description="isFrontend ? 'Sesuaikan modul fitur, sub-fitur, request API, atau keterangan untuk halaman ini.' : 'Sesuaikan domain bisnis, kelompok fungsi, atau keterangan untuk endpoint ini.'"
    :ui="{ content: 'sm:max-w-lg' }"
  >
    <template #body>
      <div v-if="editingRoute" class="space-y-4 text-xs">
        <!-- Endpoint Identifier Badge -->
        <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 flex items-center gap-2">
          <UBadge size="xs" :color="getMethodBadgeColor(editingRoute.method)" variant="solid" class="font-mono font-bold">
            {{ editingRoute.method }}
          </UBadge>
          <span class="font-mono font-semibold text-neutral-900 dark:text-neutral-100">{{ editingRoute.path }}</span>
        </div>

        <!-- Domain / Modul Fitur -->
        <div>
          <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">
            {{ isFrontend ? 'Modul Fitur' : 'Domain Bisnis' }} <span class="text-rose-500">*</span>
          </label>
          <UInput
            v-model="editForm.domain"
            :placeholder="isFrontend ? 'misal: Dashboard, Manajemen Proyek, Pengaturan' : 'misal: Autentikasi & Keamanan, Manajemen Proyek, Aset'"
          />
        </div>

        <!-- Kelompok Fungsi / Sub-Fitur -->
        <div>
          <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">
            {{ isFrontend ? 'Sub-Fitur / Jenis Halaman' : 'Kelompok Fungsi' }} <span class="text-rose-500">*</span>
          </label>
          <UInput
            v-model="editForm.function_group"
            :placeholder="isFrontend ? 'misal: Detail Aplikasi, Form Buat, Ringkasan' : 'misal: Login & Sesi, Pengambilan Data (Read), Mutasi Data'"
          />
        </div>

        <!-- Target Requests (Only Frontend) -->
        <div v-if="isFrontend">
          <div class="flex items-center justify-between mb-1">
            <label class="block font-medium text-neutral-700 dark:text-neutral-300">
              Request API yang Dipanggil (1 baris per request)
            </label>
            <span class="text-xs text-neutral-400">Contoh: GET /api/v1/projects</span>
          </div>
          <UTextarea
            v-model="editForm.target_requests"
            :rows="4"
            placeholder="GET /api/v1/projects&#10;POST /api/v1/projects&#10;GET /api/v1/auth/me"
            class="font-mono text-xs"
          />
          <p class="text-xs text-neutral-400 mt-1">
            Daftar endpoint backend yang dipanggil oleh halaman ini.
          </p>
        </div>

        <!-- Keterangan / Deskripsi -->
        <div>
          <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">
            {{ isFrontend ? 'Keterangan / Tujuan Halaman' : 'Keterangan / Penjelasan Fungsi' }} <span class="text-rose-500">*</span>
          </label>
          <UTextarea
            v-model="editForm.description"
            :rows="3"
            :placeholder="isFrontend ? 'Jelaskan tujuan antarmuka halaman ini, interaksi yang disediakan, dan data yang dimuat...' : 'Jelaskan peran endpoint ini, parameter yang diterima, dan respon yang dihasilkan...'"
          />
        </div>

        <!-- Tags -->
        <div>
          <label class="block font-medium text-neutral-700 dark:text-neutral-300 mb-1">Kata Kunci (Tags)</label>
          <UInput v-model="editForm.tags" placeholder="misal: auth, login, jwt, session (pisahkan dengan koma)" />
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-end gap-2">
        <UButton color="neutral" variant="ghost" size="xs" @click="editModalOpen = false">
          Batal
        </UButton>
        <UButton color="primary" size="xs" :loading="savingEdit" @click="handleSaveEdit">
          Simpan Perubahan
        </UButton>
      </div>
    </template>
  </UModal>
</template>
