<script setup lang="ts">
// Deep dive component for MVPP and MCP architecture
</script>

<template>
  <div class="space-y-8">
    <!-- Section 1: MVPP Architecture Deep Dive -->
    <div class="p-6 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 shadow-2xs space-y-5">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-xl bg-purple-50 text-purple-600 dark:bg-purple-950/60 dark:text-purple-400 flex items-center justify-center border border-purple-500/20">
          <UIcon name="i-lucide-brain" class="w-5 h-5" />
        </div>
        <div>
          <h2 class="text-base sm:text-lg font-bold text-neutral-900 dark:text-white">
            Paradigma MVPP (Model - View - Presenter - Planner)
          </h2>
          <p class="text-xs sm:text-sm text-neutral-500 dark:text-neutral-400">
            Arsitektur masa depan untuk Backend bertenaga AI (AI-Augmented Systems).
          </p>
        </div>
      </div>

      <p class="text-xs sm:text-sm text-neutral-700 dark:text-neutral-300 leading-relaxed">
        Pada era sebelum AI, pola MVC (Model-View-Controller) atau MVP (Model-View-Presenter) sudah cukup karena semua alur data bersifat <strong>100% deterministik</strong>. Namun saat backend mulai mengintegrasikan LLM dan AI Agent, mencampur <em>prompt engineering</em> atau pemanggilan model probabilistik di dalam Controller menimbulkan kerapuhan sistem (*fragility*), latency tak terprediksi, dan kegagalan unit test.
      </p>

      <!-- 4 Pillars of MVPP Grid -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <!-- M: Model -->
        <div class="p-4 rounded-xl border border-blue-200 dark:border-blue-900/60 bg-blue-50/40 dark:bg-blue-950/20">
          <div class="flex items-center gap-2 mb-2">
            <span class="w-6 h-6 rounded-md bg-blue-600 text-white flex items-center justify-center text-xs font-bold">M</span>
            <h4 class="font-bold text-sm text-blue-950 dark:text-blue-300">
              Model
            </h4>
          </div>
          <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Sumber kebenaran data (*Single Source of Truth*), skema GORM/SQL, invariant entitas, dan persistensi database.
          </p>
        </div>

        <!-- V: View / Serializer -->
        <div class="p-4 rounded-xl border border-cyan-200 dark:border-cyan-900/60 bg-cyan-50/40 dark:bg-cyan-950/20">
          <div class="flex items-center gap-2 mb-2">
            <span class="w-6 h-6 rounded-md bg-cyan-600 text-white flex items-center justify-center text-xs font-bold">V</span>
            <h4 class="font-bold text-sm text-cyan-950 dark:text-cyan-300">
              View / Serializer
            </h4>
          </div>
          <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Format serialisasi output: JSON API DTO, format SSE event stream, atau schema Tool Call JSON-RPC untuk AI Agent.
          </p>
        </div>

        <!-- P: Presenter -->
        <div class="p-4 rounded-xl border border-sky-200 dark:border-sky-900/60 bg-sky-50/40 dark:bg-sky-950/20">
          <div class="flex items-center gap-2 mb-2">
            <span class="w-6 h-6 rounded-md bg-sky-600 text-white flex items-center justify-center text-xs font-bold">P</span>
            <h4 class="font-bold text-sm text-sky-950 dark:text-sky-300">
              Presenter / Controller
            </h4>
          </div>
          <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Konduktor alur kontrol deterministik: Memvalidasi DTO, memanggil Service bisnis, dan mengorkestrasi kapan meminta pertimbangan ke Planner.
          </p>
        </div>

        <!-- P: Planner -->
        <div class="p-4 rounded-xl border border-purple-200 dark:border-purple-900/60 bg-purple-50/40 dark:bg-purple-950/20">
          <div class="flex items-center gap-2 mb-2">
            <span class="w-6 h-6 rounded-md bg-purple-600 text-white flex items-center justify-center text-xs font-bold">P</span>
            <h4 class="font-bold text-sm text-purple-950 dark:text-purple-300">
              Planner (AI Layer)
            </h4>
          </div>
          <p class="text-xs text-neutral-600 dark:text-neutral-400 leading-relaxed">
            Lapisan kognitif probabilistik: Menjalankan primitif System One (choice, noul, score), prompt template, dan penalaran AI terisolasi.
          </p>
        </div>
      </div>
    </div>

    <!-- Section 2: MCP (Model Context Protocol) First Architecture -->
    <div class="p-6 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 shadow-2xs space-y-5">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-xl bg-sky-50 text-sky-600 dark:bg-sky-950/60 dark:text-sky-400 flex items-center justify-center border border-sky-500/20">
          <UIcon name="i-lucide-bot" class="w-5 h-5" />
        </div>
        <div>
          <h2 class="text-base sm:text-lg font-bold text-neutral-900 dark:text-white">
            Pola Arsitektur MCP-Native (Dual Transport)
          </h2>
          <p class="text-xs sm:text-sm text-neutral-500 dark:text-neutral-400">
            Satu backend melayani Pengguna Manusia dan AI Coding Agent secara setara.
          </p>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-5 text-xs">
        <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-50/60 dark:bg-neutral-800/40 space-y-3">
          <h3 class="font-bold text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-monitor" class="w-4 h-4 text-blue-500" />
            <span>Pintu 1: REST API (Transport Web Browser)</span>
          </h3>
          <p class="text-neutral-600 dark:text-neutral-300 leading-relaxed">
            Menyediakan endpoint HTTP standar (seperti <code>GET /api/projects</code>, <code>POST /api/scan</code>). Digunakan oleh browser, Vue/Nuxt dashboard, atau aplikasi mobile Flutter.
          </p>
          <div class="p-2.5 rounded bg-neutral-900 text-neutral-200 font-mono text-[11px]">
            Frontend Nuxt ──▶ HTTP GET /api/projects ──▶ ProjectHandler ──▶ ProjectService
          </div>
        </div>

        <div class="p-4 rounded-xl border border-sky-500/20 bg-sky-50/30 dark:bg-sky-950/20 space-y-3">
          <h3 class="font-bold text-sky-900 dark:text-sky-300 flex items-center gap-2">
            <UIcon name="i-lucide-terminal" class="w-4 h-4 text-sky-400" />
            <span>Pintu 2: MCP Server (Transport AI Coding Agent)</span>
          </h3>
          <p class="text-neutral-700 dark:text-neutral-300 leading-relaxed">
            Menyediakan protokol JSON-RPC 2.0 (stdio CLI atau HTTP SSE <code>/mcp/sse</code>). AI Agent seperti Antigravity, Cursor, dan Claude Code dapat memanggil Tools secara terstruktur tanpa hallucination rute.
          </p>
          <div class="p-2.5 rounded bg-[#010810] text-sky-300 font-mono text-[11px] border border-white/10">
            AI Agent ──▶ tool: vibeasik_list_projects ──▶ McpExecutor ──▶ ProjectService
          </div>
        </div>
      </div>

      <!-- Key Takeaway Alert -->
      <div class="p-4 rounded-xl border border-purple-500/20 bg-purple-50/50 dark:bg-purple-950/30 flex items-start gap-3">
        <UIcon name="i-lucide-lightbulb" class="w-5 h-5 text-purple-600 dark:text-purple-400 shrink-0 mt-0.5" />
        <div class="text-xs text-purple-900 dark:text-purple-200 leading-relaxed">
          <strong>Inti Keberhasilan Scale Up:</strong> Kedua pintu masuk (HTTP REST dan MCP JSON-RPC) mengeksekusi <strong>Domain Service yang sama persis</strong> di <code>internal/service/</code>. Tidak ada duplikasi logika bisnis!
        </div>
      </div>
    </div>
  </div>
</template>
