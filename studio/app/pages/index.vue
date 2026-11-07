<script setup lang="ts">
definePageMeta({
  public: true
})

useHead({
  title: 'vibeasik - Platform Observasi, Keamanan & MCP Studio untuk Vibecoders',
  meta: [
    {
      name: 'description',
      content: 'Aplikasi khusus untuk vibecoders: memastikan kode hasil prompting AI aman dari celah keamanan, bebas port bentrok, dan mudah dimaintenance.'
    }
  ]
})
</script>

<template>
  <div class="bg-[#020c18] text-white selection:bg-sky-500/30 selection:text-white">
    <!-- 1. Full Screen Hero Section (100vh / Full 1 Layar) -->
    <section
      class="relative w-full h-screen min-h-[640px] flex items-center overflow-hidden bg-cover bg-center"
      style="background-image: url('/asset/hero-bg.webp');"
    >
      <!-- Top Scrim for transparent header contrast -->
      <div class="absolute inset-x-0 top-0 h-36 bg-gradient-to-b from-[#020c18]/80 via-[#020c18]/30 to-transparent pointer-events-none z-10" />

      <!-- Center-Right Subtle Ambient Scrim (Far Right Edge Only) -->
      <div class="absolute inset-y-0 right-0 w-1/3 max-w-lg bg-gradient-to-l from-[#020c18]/50 to-transparent pointer-events-none z-10" />

      <!-- Bottom Seamless Fade into Page Background -->
      <div class="absolute inset-x-0 bottom-0 h-28 bg-gradient-to-t from-[#020c18] to-transparent pointer-events-none z-10" />

      <!-- Content Container: Text Kecil Tanpa Background di Sisi Kanan Jauh (Fokus ke Es) -->
      <div class="relative z-20 w-full px-6 sm:px-10 lg:px-16 xl:px-24 flex justify-end items-center">
        <div class="w-full max-w-xs sm:max-w-sm text-left space-y-3.5">
          <!-- Minimal Kicker / Tag -->
          <div class="inline-flex items-center gap-1.5 text-[11px] font-mono font-medium text-sky-300">
            <span class="w-1.5 h-1.5 rounded-full bg-sky-400 animate-pulse" />
            <span>vibeasik • Local Cockpit & MCP</span>
          </div>

          <!-- Headline: Ukuran Ringkas & Halus, Fokus ke Gunung Es -->
          <h1 class="text-xl sm:text-2xl lg:text-3xl font-bold tracking-tight text-white leading-snug drop-shadow-sm">
            Bangun Cepat dengan AI.<br>
            <span class="text-sky-300">Kokoh di Balik Layar.</span>
          </h1>

          <!-- Text Ringkas & Bersih -->
          <p class="text-xs sm:text-sm text-slate-200/90 leading-relaxed font-normal drop-shadow-xs">
            Cockpit observasi lokal, port conflict guard, dan MCP hub untuk menjaga kode buatan AI tetap aman, rapi, dan siap rilis.
          </p>

          <!-- Action Buttons Ringkas -->
          <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 pt-1">
            <NuxtLink
              to="/download"
              class="inline-flex items-center justify-center gap-1.5 px-4 py-2 rounded-lg font-medium text-white bg-sky-600 hover:bg-sky-500 active:bg-sky-700 transition-all text-xs shadow-md shadow-sky-950/40"
            >
              <UIcon name="i-lucide-download" class="w-3.5 h-3.5" />
              <span>Unduh Desktop (v1.0.0)</span>
            </NuxtLink>

            <NuxtLink
              to="/cockpit"
              class="inline-flex items-center justify-center gap-1.5 px-3.5 py-2 rounded-lg font-medium text-white/90 bg-white/10 hover:bg-white/15 border border-white/20 transition-all text-xs backdrop-blur-xs"
            >
              <UIcon name="i-lucide-layout-grid" class="w-3.5 h-3.5 text-sky-300" />
              <span>Buka Cockpit</span>
            </NuxtLink>
          </div>

          <!-- Micro Trust Elements Minimalis -->
          <div class="pt-2 flex flex-wrap items-center gap-2.5 text-[11px] text-slate-300/80 font-mono">
            <span class="flex items-center gap-1">
              <UIcon name="i-lucide-shield-check" class="w-3.5 h-3.5 text-sky-400" />
              <span>Local-First</span>
            </span>
            <span>•</span>
            <span class="flex items-center gap-1">
              <UIcon name="i-lucide-cpu" class="w-3.5 h-3.5 text-sky-400" />
              <span>Port Guard</span>
            </span>
            <span>•</span>
            <span class="flex items-center gap-1">
              <UIcon name="i-lucide-bot" class="w-3.5 h-3.5 text-sky-400" />
              <span>20 MCP</span>
            </span>
          </div>
        </div>
      </div>

      <!-- Scroll Indicator -->
      <a
        href="#featured-apps"
        class="absolute bottom-5 left-1/2 -translate-x-1/2 z-20 text-slate-400 hover:text-white transition-colors flex flex-col items-center gap-1 text-xs font-mono"
        aria-label="Scroll ke Featured Vibe Apps"
      >
        <span class="text-[11px] tracking-wider uppercase text-slate-400">Jelajahi Vibe Apps</span>
        <UIcon name="i-lucide-chevron-down" class="w-4 h-4 text-sky-400 animate-bounce" />
      </a>
    </section>

    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-24 sm:space-y-32 py-16 sm:py-24">
      <!-- 2. Featured Vibe Apps: By Tenan (5 Grid) & Popular User Apps (Filtered) -->
      <FeaturedVibeApps />

      <!-- 3. Interactive Mockup Showcase (Transparan Glass) -->
      <section id="showcase" class="scroll-mt-20">
        <div class="rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-md shadow-2xl overflow-hidden">
          <!-- Mockup Top Bar -->
          <div class="h-10 bg-slate-950/70 border-b border-white/10 px-4 flex items-center justify-between">
            <div class="flex items-center gap-2">
              <span class="w-3 h-3 rounded-full bg-rose-500/80 inline-block" />
              <span class="w-3 h-3 rounded-full bg-amber-500/80 inline-block" />
              <span class="w-3 h-3 rounded-full bg-emerald-500/80 inline-block" />
            </div>
            <span class="text-xs font-mono text-slate-400">vibeasik Studio Cockpit — http://localhost:8765</span>
            <div class="w-12" />
          </div>

          <!-- Mockup Content Grid: Transparan Glass Cards -->
          <div class="p-6 sm:p-8 grid grid-cols-1 md:grid-cols-3 gap-6 bg-slate-950/30">
            <!-- Card 1: Diagnostic Health -->
            <div class="rounded-xl border border-white/10 bg-slate-900/50 backdrop-blur-sm p-5 space-y-3">
              <div class="flex items-center justify-between">
                <span class="text-xs font-bold uppercase tracking-wider text-slate-400">Audit Keamanan & Smells</span>
                <span class="px-2 py-0.5 rounded text-[11px] font-bold bg-sky-950/80 text-sky-300 border border-sky-500/30">Skor 94/100</span>
              </div>
              <div class="text-2xl font-bold text-white">
                Aman & Terverifikasi
              </div>
              <p class="text-xs text-slate-300 leading-relaxed">
                Memindai variabel .env tanpa enkripsi, SQL Injection potensial, query berat N+1, dan rute API yang tidak memiliki middleware auth.
              </p>
            </div>

            <!-- Card 2: Port Conflict Guard -->
            <div class="rounded-xl border border-white/10 bg-slate-900/50 backdrop-blur-sm p-5 space-y-3">
              <div class="flex items-center justify-between">
                <span class="text-xs font-bold uppercase tracking-wider text-slate-400">Live Port Conflict Guard</span>
                <span class="flex items-center gap-1.5 text-xs text-sky-400 font-mono">
                  <span class="w-2 h-2 rounded-full bg-sky-400 animate-ping" />
                  Live Monitor
                </span>
              </div>
              <div class="text-sm font-mono space-y-1.5 pt-1">
                <div class="flex items-center justify-between p-2 rounded bg-slate-950/60 border border-white/5 text-slate-200">
                  <span>Port 3000 (Nuxt FE)</span>
                  <span class="text-sky-400 font-bold">AKTIF</span>
                </div>
                <div class="flex items-center justify-between p-2 rounded bg-slate-950/60 border border-white/5 text-slate-200">
                  <span>Port 8765 (Local Engine)</span>
                  <span class="text-sky-400 font-bold">AKTIF</span>
                </div>
              </div>
              <p class="text-[11px] text-slate-400">
                Tidak ada tabrakan proses terdeteksi di 127.0.0.1.
              </p>
            </div>

            <!-- Card 3: MCP Agent Bridge -->
            <div class="rounded-xl border border-white/10 bg-slate-900/50 backdrop-blur-sm p-5 space-y-3">
              <div class="flex items-center justify-between">
                <span class="text-xs font-bold uppercase tracking-wider text-slate-400">Model Context Protocol</span>
                <span class="px-2 py-0.5 rounded text-[11px] font-bold bg-sky-950/80 text-sky-300 border border-sky-500/30">20 Tools</span>
              </div>
              <div class="text-2xl font-bold text-white">
                AI Agent Connected
              </div>
              <p class="text-xs text-slate-300 leading-relaxed">
                Google Antigravity, Claude, Cursor, dan Windsurf dapat langsung menjalankan diagnosa dan membaca modul proyek melalui MCP stdio.
              </p>
            </div>
          </div>
        </div>
      </section>

      <!-- 3. Kenapa Vibecoders Membutuhkan Vibeasik? (Transparan & Rapi) -->
      <section id="features" class="space-y-10 scroll-mt-20">
        <div class="text-center max-w-3xl mx-auto space-y-3">
          <h2 class="text-3xl sm:text-4xl font-extrabold text-white">
            Tantangan Nyata Vibecoding Hari Ini
          </h2>
          <p class="text-sm sm:text-base text-slate-300">
            Ketika Anda membuat aplikasi dengan prompting AI, masalah teknis di bawah permukaan sering kali tidak terlihat sampai aplikasi dirilis.
          </p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-6 sm:gap-8">
          <!-- Pillar 1: Security -->
          <div id="security" class="rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-sm p-7 space-y-4 hover:border-sky-500/30 transition-colors">
            <div class="w-12 h-12 rounded-xl bg-sky-500/10 border border-sky-400/20 text-sky-300 flex items-center justify-center">
              <UIcon name="i-lucide-shield-alert" class="w-6 h-6" />
            </div>
            <h3 class="text-xl font-bold text-white">
              1. Keamanan Hasil Prompting
            </h3>
            <p class="text-sm text-slate-300 leading-relaxed">
              AI sering menghasilkan kode yang bekerja namun rapuh: secret database di-commit tanpa sengaja, endpoint tanpa CORS yang aman, atau SQL injection.
              vibeasik menjalankan <strong class="text-white">audit keamanan komprehensif</strong> dalam 1 klik sebelum kode Anda dideploy.
            </p>
          </div>

          <!-- Pillar 2: Maintainability & Anti-Spaghetti -->
          <div class="rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-sm p-7 space-y-4 hover:border-sky-500/30 transition-colors">
            <div class="w-12 h-12 rounded-xl bg-sky-500/10 border border-sky-400/20 text-sky-300 flex items-center justify-center">
              <UIcon name="i-lucide-boxes" class="w-6 h-6" />
            </div>
            <h3 class="text-xl font-bold text-white">
              2. Kemudahan Maintenance
            </h3>
            <p class="text-sm text-slate-300 leading-relaxed">
              Tanpa disiplin arsitektur, prompt bertubi-tubi membuat file membengkak hingga ribuan baris kode yang membingungkan.
              vibeasik menjaga <strong class="text-white">arsitektur modular bertingkat</strong> (Backend, Frontend, Mobile, DB) dan mengelola TODOs 3-level secara konsisten.
            </p>
          </div>

          <!-- Pillar 3: Ghost Process & Port Collision -->
          <div class="rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-sm p-7 space-y-4 hover:border-sky-500/30 transition-colors">
            <div class="w-12 h-12 rounded-xl bg-sky-500/10 border border-sky-400/20 text-sky-300 flex items-center justify-center">
              <UIcon name="i-lucide-cpu" class="w-6 h-6" />
            </div>
            <h3 class="text-xl font-bold text-white">
              3. Port Guard & Stabilitas
            </h3>
            <p class="text-sm text-slate-300 leading-relaxed">
              Menjalankan berbagai proyek AI sekaligus sering membuat port 3000 atau 8080 macet karena proses zombie yang tidak mati saat terminal ditutup.
              vibeasik memonitor port lokal secara live dan menyediakan tombol <strong class="text-white">1-Klik Force Close</strong> otomatis.
            </p>
          </div>
        </div>
      </section>

      <!-- 4. Model Context Protocol (MCP) Section (Transparan Glass) -->
      <section id="mcp" class="scroll-mt-20">
        <div class="rounded-2xl border border-sky-500/20 bg-slate-900/40 backdrop-blur-md p-8 sm:p-12 relative overflow-hidden">
          <div class="max-w-3xl space-y-4 relative z-10">
            <div class="inline-flex items-center gap-2 text-xs font-mono font-medium uppercase tracking-wider text-sky-300 bg-sky-950/60 border border-sky-500/30 px-3 py-1 rounded-full">
              <UIcon name="i-lucide-bot" class="w-3.5 h-3.5 text-sky-400" />
              <span>AI Superpowers dengan Standar Industri</span>
            </div>

            <h2 class="text-2xl sm:text-3xl lg:text-4xl font-extrabold text-white">
              Hubungkan AI Agent Anda ke Mesin Lokal Melalui 20 MCP Tools
            </h2>

            <p class="text-sm sm:text-base text-slate-300 leading-relaxed">
              Tidak perlu lagi copy-paste struktur folder atau mengulang instruksi panjang.
              Dengan perintah <code class="font-mono bg-slate-950/80 px-2 py-0.5 rounded text-sky-300 border border-white/10">vibeasik mcp install</code>,
              Google Antigravity, Claude Desktop, Cursor, dan Windsurf dapat langsung mengaudit proyek, memindai dependensi, dan mengusulkan perbaikan secara mandiri.
            </p>

            <div class="pt-4 flex flex-wrap gap-3">
              <NuxtLink
                to="/download"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-sky-600 hover:bg-sky-500 text-white font-semibold text-xs transition-colors shadow-sm"
              >
                <UIcon name="i-lucide-download" class="w-4 h-4" />
                <span>Dapatkan vibeasik Desktop</span>
              </NuxtLink>
              <NuxtLink
                to="/cockpit"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl border border-white/20 hover:bg-white/10 text-white font-semibold text-xs transition-colors backdrop-blur-sm"
              >
                <span>Eksplorasi MCP Hub di Studio</span>
                <UIcon name="i-lucide-arrow-right" class="w-3.5 h-3.5 text-sky-300" />
              </NuxtLink>
            </div>
          </div>
        </div>
      </section>

      <!-- 5. Final Call to Action -->
      <section class="max-w-4xl mx-auto text-center space-y-6">
        <h2 class="text-3xl sm:text-4xl font-extrabold text-white">
          Mulai Amankan Aplikasi AI Anda Hari Ini
        </h2>
        <p class="text-sm sm:text-base text-slate-300 max-w-2xl mx-auto">
          Tersedia sebagai aplikasi desktop executable untuk Ubuntu, Debian, dan Windows,
          atau jalankan langsung via CLI lokal.
        </p>

        <div class="flex flex-col sm:flex-row items-center justify-center gap-3 pt-2">
          <NuxtLink
            to="/download"
            class="w-full sm:w-auto px-7 py-3 rounded-xl font-bold text-white bg-sky-600 hover:bg-sky-500 transition-colors shadow-sm text-sm"
          >
            Pasang vibeasik
          </NuxtLink>
          <NuxtLink
            to="/register"
            class="w-full sm:w-auto px-7 py-3 rounded-xl font-semibold border border-white/20 hover:bg-white/10 text-white text-sm transition-colors backdrop-blur-sm"
          >
            Daftar Akun Gratis
          </NuxtLink>
        </div>
      </section>
    </div>
  </div>
</template>
