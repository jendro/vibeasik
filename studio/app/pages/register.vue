<script setup lang="ts">
definePageMeta({
  public: true
})

const { register, quickDemoLogin } = useAuth()
const router = useRouter()
const toast = useToast()

const name = ref('')
const email = ref('')
const password = ref('')
const toolPreference = ref('Google Antigravity')
const loading = ref(false)
const errorMessage = ref<string | null>(null)

const toolOptions = [
  'Google Antigravity',
  'Claude Code & Desktop',
  'Cursor IDE',
  'Windsurf / Roo Code',
  'Aider & Custom Agents'
]

const handleRegister = async () => {
  errorMessage.value = null
  loading.value = true

  try {
    const result = await register(name.value, email.value, password.value, toolPreference.value)
    if (result.success) {
      toast.add({
        title: 'Registrasi Berhasil',
        description: `Selamat datang, ${name.value}! Akun Anda telah siap.`,
        icon: 'i-lucide-check-circle',
        color: 'success'
      })
      router.push('/cockpit')
    } else {
      errorMessage.value = result.error || 'Gagal membuat akun.'
    }
  } catch (err: any) {
    errorMessage.value = err.message || 'Terjadi kesalahan sistem saat mendaftar.'
  } finally {
    loading.value = false
  }
}

const handleQuickDemo = () => {
  quickDemoLogin()
  toast.add({
    title: 'Login Cepat Berhasil',
    description: 'Anda masuk dalam sesi Pengembang / Vibecoder Lokal.',
    icon: 'i-lucide-zap',
    color: 'success'
  })
  router.push('/cockpit')
}

useHead({
  title: 'Daftar Akun Baru - vibeasik',
  meta: [
    { name: 'description', content: 'Daftar akun vibeasik untuk memastikan aplikasi buatan prompting AI aman dan mudah dimaintenance.' }
  ]
})
</script>

<template>
  <div class="min-h-[calc(100vh-8rem)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
    <div class="w-full max-w-md space-y-8">
      <!-- Header -->
      <div class="text-center space-y-2">
        <NuxtLink to="/" class="inline-flex items-center gap-2.5 mb-2 group">
          <div class="w-10 h-10 rounded-xl bg-sky-600 flex items-center justify-center text-white shadow-xs group-hover:scale-105 transition-transform">
            <UIcon name="i-lucide-sparkles" class="w-6 h-6" />
          </div>
          <span class="font-extrabold text-2xl tracking-tight text-neutral-900 dark:text-white">vibeasik</span>
        </NuxtLink>

        <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white">
          Daftar Akun Vibecoder
        </h1>
        <p class="text-xs text-neutral-500 dark:text-neutral-400 max-w-sm mx-auto leading-relaxed">
          Mulai audit keamanan kode hasil AI, pantau port jaringan, dan hubungkan AI agent lokal Anda.
        </p>
      </div>

      <!-- Card Container -->
      <div class="rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/60 shadow-sm p-6 sm:p-8 space-y-6">
        <!-- Error Alert -->
        <div
          v-if="errorMessage"
          class="p-3 rounded-lg bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-800 text-rose-700 dark:text-rose-400 text-xs flex items-center gap-2"
        >
          <UIcon name="i-lucide-alert-circle" class="w-4 h-4 shrink-0" />
          <span>{{ errorMessage }}</span>
        </div>

        <form class="space-y-4" @submit.prevent="handleRegister">
          <!-- Name Input -->
          <div class="space-y-1.5">
            <label for="name" class="block text-xs font-semibold text-neutral-700 dark:text-neutral-300">
              Nama Lengkap / Alias Vibecoder
            </label>
            <input
              id="name"
              v-model="name"
              type="text"
              required
              autocomplete="name"
              placeholder="Contoh: Alex Vibecoder"
              class="w-full px-3.5 py-2.5 rounded-lg border border-neutral-300 dark:border-white/10 bg-white dark:bg-[#020c18] text-neutral-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-sky-500 transition-colors"
            >
          </div>

          <!-- Email Input -->
          <div class="space-y-1.5">
            <label for="email" class="block text-xs font-semibold text-neutral-700 dark:text-neutral-300">
              Alamat Email
            </label>
            <input
              id="email"
              v-model="email"
              type="email"
              required
              autocomplete="email"
              placeholder="alex@example.com"
              class="w-full px-3.5 py-2.5 rounded-lg border border-neutral-300 dark:border-white/10 bg-white dark:bg-[#020c18] text-neutral-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-sky-500 transition-colors"
            >
          </div>

          <!-- Password Input -->
          <div class="space-y-1.5">
            <label for="password" class="block text-xs font-semibold text-neutral-700 dark:text-neutral-300">
              Password (Minimal 4 Karakter)
            </label>
            <input
              id="password"
              v-model="password"
              type="password"
              required
              autocomplete="new-password"
              placeholder="••••••••"
              class="w-full px-3.5 py-2.5 rounded-lg border border-neutral-300 dark:border-white/10 bg-white dark:bg-[#020c18] text-neutral-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-sky-500 transition-colors"
            >
          </div>

          <!-- AI Tool Preference -->
          <div class="space-y-1.5">
            <label for="tool" class="block text-xs font-semibold text-neutral-700 dark:text-neutral-300">
              AI Agent / Coding Tool Utama Anda
            </label>
            <select
              id="tool"
              v-model="toolPreference"
              class="w-full px-3.5 py-2.5 rounded-lg border border-neutral-300 dark:border-white/10 bg-white dark:bg-[#020c18] text-neutral-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-sky-500 transition-colors"
            >
              <option v-for="t in toolOptions" :key="t" :value="t">
                {{ t }}
              </option>
            </select>
          </div>

          <!-- Submit Button -->
          <button
            type="submit"
            :disabled="loading"
            class="w-full py-2.5 px-4 rounded-xl font-semibold text-white bg-sky-600 hover:bg-sky-500 active:bg-sky-700 transition-colors shadow-xs text-sm flex items-center justify-center gap-2 disabled:opacity-50 cursor-pointer pt-2"
          >
            <UIcon v-if="loading" name="i-lucide-loader-2" class="w-4 h-4 animate-spin" />
            <span>{{ loading ? 'Mendaftarkan Akun...' : 'Daftar & Buka Studio Cockpit' }}</span>
          </button>
        </form>

        <!-- Divider -->
        <div class="relative flex items-center justify-center">
          <div class="border-t border-neutral-200 dark:border-neutral-800 w-full" />
          <span class="bg-white dark:bg-neutral-900 px-3 text-[11px] font-mono uppercase text-neutral-400 absolute">
            atau
          </span>
        </div>

        <!-- 1-Click Developer Demo Access -->
        <button
          type="button"
          class="w-full py-2.5 px-4 rounded-xl border border-sky-500/30 bg-sky-500/10 hover:bg-sky-500/20 text-sky-700 dark:text-sky-300 text-xs font-semibold flex items-center justify-center gap-2 transition-colors cursor-pointer"
          @click="handleQuickDemo"
        >
          <UIcon name="i-lucide-zap" class="w-4 h-4 text-sky-500" />
          <span>Lewati Pendaftaran (Coba Cepat Mode Lokal)</span>
        </button>

        <!-- Login Link -->
        <p class="text-center text-xs text-neutral-500 dark:text-neutral-400 pt-2">
          Sudah memiliki akun?
          <NuxtLink to="/login" class="text-sky-600 dark:text-sky-400 font-semibold hover:underline">
            Masuk ke Akun Anda →
          </NuxtLink>
        </p>
      </div>

      <!-- Back Link -->
      <div class="text-center">
        <NuxtLink to="/" class="inline-flex items-center gap-1.5 text-xs text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200 transition-colors">
          <UIcon name="i-lucide-arrow-left" class="w-3.5 h-3.5" />
          <span>Kembali ke Beranda Utama vibeasik</span>
        </NuxtLink>
      </div>
    </div>
  </div>
</template>
