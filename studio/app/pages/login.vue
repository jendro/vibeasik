<script setup lang="ts">
definePageMeta({
  public: true
})

const { login, quickDemoLogin } = useAuth()
const route = useRoute()
const router = useRouter()
const toast = useToast()

const email = ref('')
const password = ref('')
const rememberMe = ref(true)
const loading = ref(false)
const errorMessage = ref<string | null>(null)

const redirectTarget = computed(() => {
  const r = route.query.redirect as string
  return r && r.startsWith('/') ? r : '/cockpit'
})

const handleLogin = async () => {
  errorMessage.value = null
  loading.value = true

  try {
    const result = await login(email.value, password.value)
    if (result.success) {
      toast.add({
        title: 'Berhasil Masuk',
        description: 'Selamat datang kembali di vibeasik Cockpit Studio.',
        icon: 'i-lucide-check-circle',
        color: 'success'
      })
      router.push(redirectTarget.value)
    } else {
      errorMessage.value = result.error || 'Email atau password salah.'
    }
  } catch (err: any) {
    errorMessage.value = err.message || 'Terjadi kesalahan sistem saat mencoba masuk.'
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
  router.push(redirectTarget.value)
}

useHead({
  title: 'Masuk ke Studio Cockpit - vibeasik',
  meta: [
    { name: 'description', content: 'Masuk ke akun vibeasik untuk mengelola proyek AI, memantau port, dan menjalankan audit keamanan kode.' }
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
          Masuk ke Studio Cockpit
        </h1>
        <p class="text-xs text-neutral-500 dark:text-neutral-400 max-w-sm mx-auto leading-relaxed">
          Autentikasi diperlukan untuk mengakses katalog proyek lokal, audit keamanan kode AI, dan status port.
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

        <form class="space-y-4" @submit.prevent="handleLogin">
          <!-- Email Input -->
          <div class="space-y-1.5">
            <label for="email" class="block text-xs font-semibold text-neutral-700 dark:text-neutral-300">
              Alamat Email / Username
            </label>
            <input
              id="email"
              v-model="email"
              type="text"
              required
              autocomplete="email"
              placeholder="nama@domain.com"
              class="w-full px-3.5 py-2.5 rounded-lg border border-neutral-300 dark:border-white/10 bg-white dark:bg-[#020c18] text-neutral-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-sky-500 transition-colors"
            >
          </div>

          <!-- Password Input -->
          <div class="space-y-1.5">
            <div class="flex items-center justify-between">
              <label for="password" class="block text-xs font-semibold text-neutral-700 dark:text-neutral-300">
                Password
              </label>
              <a href="#" class="text-[11px] text-sky-600 dark:text-sky-400 hover:underline" @click.prevent="toast.add({ title: 'Info', description: 'Gunakan login cepat demo atau reset via terminal.', color: 'info' })">
                Lupa password?
              </a>
            </div>
            <input
              id="password"
              v-model="password"
              type="password"
              required
              autocomplete="current-password"
              placeholder="••••••••"
              class="w-full px-3.5 py-2.5 rounded-lg border border-neutral-300 dark:border-white/10 bg-white dark:bg-[#020c18] text-neutral-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-sky-500 transition-colors"
            >
          </div>

          <!-- Remember me -->
          <div class="flex items-center">
            <label class="flex items-center gap-2 text-xs text-neutral-600 dark:text-neutral-400 cursor-pointer">
              <input
                v-model="rememberMe"
                type="checkbox"
                class="rounded border-neutral-300 dark:border-neutral-700 text-sky-600 focus:ring-sky-500"
              >
              <span>Ingat sesi saya di komputer ini</span>
            </label>
          </div>

          <!-- Submit Button -->
          <button
            type="submit"
            :disabled="loading"
            class="w-full py-2.5 px-4 rounded-xl font-semibold text-white bg-sky-600 hover:bg-sky-500 active:bg-sky-700 transition-colors shadow-xs text-sm flex items-center justify-center gap-2 disabled:opacity-50 cursor-pointer"
          >
            <UIcon v-if="loading" name="i-lucide-loader-2" class="w-4 h-4 animate-spin" />
            <span>{{ loading ? 'Memverifikasi...' : 'Masuk ke Studio' }}</span>
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
          <span>Masuk Cepat Vibecoder (Mode Lokal Instan)</span>
        </button>

        <!-- Register Link -->
        <p class="text-center text-xs text-neutral-500 dark:text-neutral-400 pt-2">
          Belum memiliki akun?
          <NuxtLink to="/register" class="text-sky-600 dark:text-sky-400 font-semibold hover:underline">
            Daftar Akun Baru →
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
