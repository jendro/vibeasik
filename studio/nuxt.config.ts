// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui'
  ],

  devtools: {
    enabled: false
  },

  css: ['~/assets/css/main.css'],

  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8765',
      // Hanya untuk `nuxt dev`: token dari `vibeasik token` (Studio produksi memakai cookie).
      apiToken: process.env.NUXT_PUBLIC_API_TOKEN || ''
    }
  },

  devServer: {
    host: process.env.NUXT_HOST || '0.0.0.0',
    // 8760 (bukan 3000) agar tidak bentrok dengan dev server proyek lain; seri 87xx milik vibeasik
    port: Number(process.env.NUXT_PORT) || 8760
  },

  compatibilityDate: '2026-06-30',

  // Gaya kode proyek: tanpa titik koma, kutip tunggal, tanpa koma di akhir (dipakai `bun run lint` & pre-commit hook)
  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  }
})
