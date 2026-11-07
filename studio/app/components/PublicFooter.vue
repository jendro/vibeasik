<template>
  <footer class="border-t border-neutral-200 dark:border-white/10 bg-white dark:bg-[#020c18] text-neutral-600 dark:text-slate-400 text-sm transition-colors">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-8">
        <!-- Col 1: Brand & Narrative -->
        <div class="md:col-span-2 space-y-4">
          <div class="flex items-center gap-2.5">
            <div class="w-8 h-8 rounded-lg bg-sky-500/20 border border-sky-400/30 flex items-center justify-center text-sky-300">
              <UIcon name="i-lucide-sparkles" class="w-4 h-4" />
            </div>
            <span class="font-extrabold text-lg text-neutral-900 dark:text-white">vibeasik</span>
          </div>

          <p class="text-xs text-neutral-500 dark:text-slate-300 max-w-md leading-relaxed">
            Platform observasi, audit keamanan, dan delivery khusus untuk para <strong>Vibecoders</strong>.
            Menjembatani kode hasil prompting AI agar terstruktur rapi, bebas celah keamanan, dan siap dirilis ke dunia nyata.
          </p>

          <div class="flex items-center gap-2 pt-2 text-xs font-mono text-neutral-400 dark:text-slate-400">
            <span>Powered by Electron • Go Fiber • Nuxt 4 • MCP Hub</span>
          </div>
        </div>

        <!-- Col 2: Navigasi Cepat -->
        <div class="space-y-3">
          <h4 class="text-xs font-bold uppercase tracking-wider text-neutral-900 dark:text-white">
            Produk & Ekosistem
          </h4>
          <ul class="space-y-2 text-xs">
            <li>
              <NuxtLink to="/" class="hover:text-sky-500 dark:hover:text-sky-300 transition-colors">Beranda Publik</NuxtLink>
            </li>
            <li>
              <NuxtLink to="/download" class="hover:text-sky-500 dark:hover:text-sky-300 transition-colors">Pasang vibeasik</NuxtLink>
            </li>
            <li>
              <NuxtLink to="/cockpit" class="hover:text-sky-500 dark:hover:text-sky-300 transition-colors">Cockpit Proyek (Admin)</NuxtLink>
            </li>
            <li>
              <NuxtLink to="/mcp" class="hover:text-sky-500 dark:hover:text-sky-300 transition-colors">AI & MCP Hub (20 Tools)</NuxtLink>
            </li>
            <li>
              <NuxtLink to="/academy" class="hover:text-sky-500 dark:hover:text-sky-300 transition-colors">Vibe Academy</NuxtLink>
            </li>
          </ul>
        </div>

        <!-- Col 3: AI Agents & Komunitas -->
        <div class="space-y-3">
          <h4 class="text-xs font-bold uppercase tracking-wider text-neutral-900 dark:text-white">
            Dukungan AI Agent
          </h4>
          <ul class="space-y-2 text-xs text-neutral-500 dark:text-slate-400">
            <li class="flex items-center gap-1.5">
              <UIcon name="i-lucide-bot" class="w-3.5 h-3.5 text-sky-400" />
              <span>Google Antigravity</span>
            </li>
            <li class="flex items-center gap-1.5">
              <UIcon name="i-lucide-bot" class="w-3.5 h-3.5 text-sky-400" />
              <span>Claude Desktop & Code</span>
            </li>
            <li class="flex items-center gap-1.5">
              <UIcon name="i-lucide-bot" class="w-3.5 h-3.5 text-sky-400" />
              <span>Cursor IDE</span>
            </li>
            <li class="flex items-center gap-1.5">
              <UIcon name="i-lucide-bot" class="w-3.5 h-3.5 text-sky-400" />
              <span>Windsurf & Roo Code</span>
            </li>
          </ul>
        </div>
      </div>

      <div class="pt-8 mt-8 border-t border-neutral-100 dark:border-white/10 flex flex-col sm:flex-row items-center justify-between gap-4 text-xs text-neutral-400 dark:text-slate-400">
        <p>© 2026 vibeasik.com • Dibuat dengan penuh dedikasi untuk Vibecoders.</p>
        <div class="flex items-center gap-4">
          <NuxtLink to="/download" class="hover:text-neutral-600 dark:hover:text-white transition-colors">Pasang</NuxtLink>
          <NuxtLink to="/login" class="hover:text-neutral-600 dark:hover:text-white transition-colors">Portal Masuk</NuxtLink>
        </div>
      </div>
    </div>
  </footer>
</template>
