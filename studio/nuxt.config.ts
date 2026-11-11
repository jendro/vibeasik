// https://nuxt.com/docs/api/configuration/nuxt-config
import { execSync } from 'node:child_process'

// Versi build Studio (tampil di sidebar di bawah judul): STUDIO_BUILD_VERSION dari Makefile, selain itu git describe.
const gitDescribe = () => {
  try {
    return execSync('git describe --tags --match "v[0-9]*" --always --dirty', { stdio: ['ignore', 'pipe', 'ignore'] }).toString().trim()
  } catch {
    return 'unknown'
  }
}
const studioBuildVersion = process.env.STUDIO_BUILD_VERSION || gitDescribe()
const studioBuildTime = new Date().toISOString()

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
      apiToken: process.env.NUXT_PUBLIC_API_TOKEN || '',
      studioBuildVersion,
      studioBuildTime
    }
  },

  devServer: {
    host: process.env.NUXT_HOST || '0.0.0.0',
    // 8760 (bukan 3000) agar tidak bentrok dengan dev server proyek lain; seri 87xx milik vibeasik
    port: Number(process.env.NUXT_PORT) || 8760
  },

  compatibilityDate: '2026-06-30',

  // Halaman fitur Pro (utils/proFeatures.ts PRO_PAGES) tidak di-prerender: di build Community halamannya tidak
  // ada (menu mengarah ke /upgrade), di build Official dirender di browser lewat 200.html.
  nitro: {
    prerender: {
      ignore: ['/agent-office', '/recap', '/client-room', '/ai-hub', '/orchestrator', '/llm-algorithm', '/queue', '/staff',
        '/executives', '/audit-learning', '/runtime-test', '/screenshots', '/design-library', '/design-lessons',
        '/downloads']
    }
  },

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
