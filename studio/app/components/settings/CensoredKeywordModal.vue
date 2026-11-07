<script setup lang="ts">
import type { CensoredKeyword } from '~/types'

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  isEditing: boolean
  initialKeyword?: Partial<CensoredKeyword> | null
  saving?: boolean
}>()

const emit = defineEmits<{
  save: [data: {
    id: number
    keyword: string
    category: string
    description: string
    is_regex: boolean
    is_active: boolean
  }]
}>()

const toast = useToast()

const form = ref<{
  id: number
  keyword: string
  category: string
  description: string
  is_regex: boolean
  is_active: boolean
}>({
  id: 0,
  keyword: '',
  category: 'API_KEY',
  description: '',
  is_regex: false,
  is_active: true
})

const categoryOptions = [
  'API_KEY',
  'AUTH_TOKEN',
  'DATABASE_CREDENTIAL',
  'ENVIRONMENT_SECRET',
  'OTHER'
]

const keywordPresets = [
  { label: 'DB Password', keyword: 'DB_PASSWORD', category: 'DATABASE_CREDENTIAL', description: 'Kata sandi basis data utama', is_regex: false },
  { label: 'Stripe Secret', keyword: 'sk_live_', category: 'API_KEY', description: 'Prefix live secret key Stripe Gateway', is_regex: false },
  { label: 'Private RSA Key', keyword: 'BEGIN RSA PRIVATE KEY', category: 'AUTH_TOKEN', description: 'Header SSL/TLS Certificate Private Key', is_regex: false },
  { label: 'OAuth Client Secret', keyword: 'CLIENT_SECRET', category: 'API_KEY', description: 'Kredensial OAuth Provider', is_regex: false },
  { label: 'JWT App Secret', keyword: 'JWT_SECRET', category: 'AUTH_TOKEN', description: 'Secret signing HMAC/RSA token', is_regex: false }
]

watch(() => props.initialKeyword, (kw) => {
  if (kw) {
    form.value = {
      id: kw.id || 0,
      keyword: kw.keyword || '',
      category: kw.category || 'API_KEY',
      description: kw.description || '',
      is_regex: kw.is_regex || false,
      is_active: kw.is_active !== undefined ? kw.is_active : true
    }
  }
}, { immediate: true })

const applyPreset = (p: typeof keywordPresets[0]) => {
  form.value.keyword = p.keyword
  form.value.category = p.category
  form.value.description = p.description
  form.value.is_regex = p.is_regex
}

const handleSubmit = () => {
  if (!form.value.keyword.trim()) {
    toast.add({
      title: 'Validasi Gagal',
      description: 'Kata kunci atau pola string rahasia wajib diisi.',
      icon: 'i-lucide-alert-circle',
      color: 'warning'
    })
    return
  }

  emit('save', {
    id: form.value.id,
    keyword: form.value.keyword.trim(),
    category: form.value.category,
    description: form.value.description.trim(),
    is_regex: form.value.is_regex,
    is_active: form.value.is_active
  })
}
</script>

<template>
  <UModal
    v-model:open="open"
    :title="isEditing ? 'Edit Keyword Sensor' : 'Tambah Keyword Sensor Baru'"
    :description="isEditing ? 'Perbarui kata kunci atau pola regex proteksi dalam vault.' : 'Daftarkan kata kunci, variabel env, atau pola regex yang harus selalu disensor sebelum dikirim ke AI.'"
    :ui="{ content: 'sm:max-w-lg' }"
  >
    <template #body>
      <div class="space-y-4 text-xs">
        <!-- Presets (when creating) -->
        <div v-if="!isEditing" class="space-y-1.5">
          <span class="text-[11px] font-medium text-neutral-500 dark:text-neutral-400">Pilih dari Contoh Preset:</span>
          <div class="flex flex-wrap gap-1.5">
            <button
              v-for="p in keywordPresets"
              :key="p.label"
              type="button"
              class="px-2 py-0.5 text-[11px] font-mono rounded bg-neutral-100 hover:bg-neutral-200 dark:bg-neutral-800 dark:hover:bg-neutral-700 text-neutral-700 dark:text-neutral-300 border border-neutral-200 dark:border-neutral-700 transition-colors cursor-pointer"
              @click="applyPreset(p)"
            >
              {{ p.label }}
            </button>
          </div>
        </div>

        <!-- Keyword / Pattern -->
        <div class="space-y-1">
          <label class="block font-medium text-neutral-700 dark:text-neutral-300">
            Kata Kunci / Pola String Rahasia *
          </label>
          <input
            v-model="form.keyword"
            type="text"
            placeholder="Contoh: DB_PASSWORD, sk_live_secret, TOKEN_RAHASIA"
            class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-amber-500"
          >
          <p class="text-[10px] text-neutral-400">
            String ini akan otomatis dicocokkan pada setiap payload/prompt dan diganti dengan alias anonim.
          </p>
        </div>

        <!-- Category -->
        <div class="space-y-1">
          <label class="block font-medium text-neutral-700 dark:text-neutral-300">Kategori Kunci</label>
          <select
            v-model="form.category"
            class="w-full px-3 py-2 text-xs rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-amber-500 cursor-pointer"
          >
            <option v-for="cat in categoryOptions" :key="cat" :value="cat">
              {{ cat }}
            </option>
          </select>
        </div>

        <!-- Description -->
        <div class="space-y-1">
          <label class="block font-medium text-neutral-700 dark:text-neutral-300">Deskripsi / Catatan Tambahan</label>
          <textarea
            v-model="form.description"
            rows="2"
            placeholder="Catatan tujuan atau layanan yang memiliki kunci ini..."
            class="w-full px-3 py-2 text-xs rounded-lg border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 text-neutral-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-amber-500"
          />
        </div>

        <!-- Toggles -->
        <div class="space-y-2 pt-1">
          <div class="flex items-center gap-2">
            <input
              id="is_regex_checkbox_modal"
              v-model="form.is_regex"
              type="checkbox"
              class="w-4 h-4 text-amber-600 rounded border-neutral-300 focus:ring-amber-500 cursor-pointer"
            >
            <label for="is_regex_checkbox_modal" class="text-xs text-neutral-700 dark:text-neutral-300 cursor-pointer select-none">
              Gunakan pencocokan Pola Reguler Expression (Regex Mode)
            </label>
          </div>

          <div class="flex items-center gap-2">
            <input
              id="is_active_kw_checkbox_modal"
              v-model="form.is_active"
              type="checkbox"
              class="w-4 h-4 text-sky-600 rounded border-neutral-300 focus:ring-sky-500 cursor-pointer"
            >
            <label for="is_active_kw_checkbox_modal" class="text-xs text-neutral-700 dark:text-neutral-300 cursor-pointer select-none font-medium">
              Aktifkan sensor kata kunci ini di seluruh aplikasi
            </label>
          </div>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center justify-end gap-2.5 w-full">
        <UButton color="neutral" variant="outline" @click="open = false">
          Batal
        </UButton>
        <UButton color="primary" :loading="saving" @click="handleSubmit">
          {{ isEditing ? 'Simpan Perubahan' : 'Tambahkan ke Vault' }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
