<script setup lang="ts">
import type { ArchitecturePattern } from '~/types'

const props = defineProps<{
  patterns: ArchitecturePattern[]
  selectedPatternId: string
}>()

const activePerspective = ref<'router' | 'controller' | 'functions'>('router')

const currentPattern = computed(() => {
  return props.patterns.find(p => p.id === props.selectedPatternId) || props.patterns[0]
})
</script>

<template>
  <div v-if="currentPattern" class="space-y-6">
    <!-- Perspective Switcher Tabs -->
    <div class="flex items-center gap-2 border-b border-neutral-200 dark:border-neutral-800 pb-3">
      <button
        type="button"
        class="inline-flex items-center gap-2 px-3.5 py-2 text-xs sm:text-sm font-semibold rounded-lg transition-all cursor-pointer"
        :class="activePerspective === 'router'
          ? 'bg-sky-600 text-white shadow-xs'
          : 'bg-white dark:bg-neutral-900 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white border border-neutral-200 dark:border-neutral-800'"
        @click="activePerspective = 'router'"
      >
        <UIcon name="i-lucide-network" class="w-4 h-4" />
        <span>1. Bedah Router</span>
      </button>

      <button
        type="button"
        class="inline-flex items-center gap-2 px-3.5 py-2 text-xs sm:text-sm font-semibold rounded-lg transition-all cursor-pointer"
        :class="activePerspective === 'controller'
          ? 'bg-sky-600 text-white shadow-xs'
          : 'bg-white dark:bg-neutral-900 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white border border-neutral-200 dark:border-neutral-800'"
        @click="activePerspective = 'controller'"
      >
        <UIcon name="i-lucide-cpu" class="w-4 h-4" />
        <span>2. Bedah Controller / Handler</span>
      </button>

      <button
        type="button"
        class="inline-flex items-center gap-2 px-3.5 py-2 text-xs sm:text-sm font-semibold rounded-lg transition-all cursor-pointer"
        :class="activePerspective === 'functions'
          ? 'bg-sky-600 text-white shadow-xs'
          : 'bg-white dark:bg-neutral-900 text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-white border border-neutral-200 dark:border-neutral-800'"
        @click="activePerspective = 'functions'"
      >
        <UIcon name="i-lucide-function-square" class="w-4 h-4" />
        <span>3. Fungsi vs Class / Struct</span>
      </button>
    </div>

    <!-- Perspective 1: Router Analysis -->
    <div v-if="activePerspective === 'router'" class="space-y-5">
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-3">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-search" class="w-5 h-5 text-sky-600 dark:text-sky-400" />
          <h3 class="text-sm font-bold text-neutral-900 dark:text-white">
            Cara Menganalisis Pola dari Router
          </h3>
        </div>
        <p class="text-xs sm:text-sm text-neutral-600 dark:text-neutral-300 leading-relaxed">
          Router adalah pintu gerbang aplikasi. Saat mengaudit codebase backend, periksa berkas pendaftaran rute (`server.go`, `routes.ts`, `api.php`):
        </p>
        <ul class="grid grid-cols-1 md:grid-cols-3 gap-3 text-xs pt-1">
          <li class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-100 dark:border-neutral-800">
            <span class="font-bold text-neutral-800 dark:text-neutral-200 block mb-1">A. Pengelompokan Rute</span>
            Apakah rute dikelompokkan dengan prefix modular (`api.Group("/projects")`) atau ditulis panjang mendatar?
          </li>
          <li class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-100 dark:border-neutral-800">
            <span class="font-bold text-neutral-800 dark:text-neutral-200 block mb-1">B. Logika di Router</span>
            Apakah handler berupa closure anonim panjang (smell) atau murni referensi method controller (`s.getProjects`)?
          </li>
          <li class="p-3 rounded-lg bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-100 dark:border-neutral-800">
            <span class="font-bold text-neutral-800 dark:text-neutral-200 block mb-1">C. Rantai Middleware</span>
            Apakah autentikasi, CORS, dan rate-limiter dipasang deklaratif atau disalin manual di tiap handler?
          </li>
        </ul>
      </div>

      <!-- Code Snippet Box -->
      <div class="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-900 text-neutral-100 overflow-hidden">
        <div class="flex items-center justify-between px-4 py-2.5 bg-neutral-950 border-b border-neutral-800">
          <span class="text-xs font-mono font-bold text-sky-400">
            {{ currentPattern.router_code_sample.title }} (Pola: {{ currentPattern.name }})
          </span>
          <span class="text-[10px] font-mono uppercase bg-neutral-800 px-2 py-0.5 rounded text-neutral-400">
            {{ currentPattern.router_code_sample.language }}
          </span>
        </div>
        <div class="p-4 font-mono text-xs text-neutral-200 leading-relaxed overflow-x-auto">
          <pre>{{ currentPattern.router_code_sample.code }}</pre>
        </div>
        <div class="px-4 py-2 bg-neutral-950/60 border-t border-neutral-800/80 text-xs text-neutral-400">
          <strong>Karakteristik:</strong> {{ currentPattern.router_characteristics }}
        </div>
      </div>
    </div>

    <!-- Perspective 2: Controller Analysis -->
    <div v-else-if="activePerspective === 'controller'" class="space-y-5">
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-3">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-scale" class="w-5 h-5 text-sky-600 dark:text-sky-400" />
          <h3 class="text-sm font-bold text-neutral-900 dark:text-white">
            Cara Menganalisis Pola dari Controller / Handler
          </h3>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
          <div class="p-3.5 rounded-lg border border-red-200 dark:border-red-950 bg-red-50/40 dark:bg-red-950/20">
            <h4 class="font-bold text-red-700 dark:text-red-400 flex items-center gap-1.5 mb-1.5">
              <UIcon name="i-lucide-x-circle" class="w-4 h-4" />
              <span>Ciri Fat Controller (Anti-Pattern)</span>
            </h4>
            <ul class="space-y-1 text-neutral-700 dark:text-neutral-300 list-disc list-inside">
              <li>Mengeksekusi query database (`db.Query`, `db.Find`) langsung di handler.</li>
              <li>Menghitung diskon, harga, atau validasi bisnis rumit di controller.</li>
              <li>File controller melebihi 300 baris kode dan sulit diuji unit test.</li>
            </ul>
          </div>

          <div class="p-3.5 rounded-lg border border-sky-200 dark:border-sky-950 bg-sky-50/40 dark:bg-sky-950/20">
            <h4 class="font-bold text-sky-700 dark:text-sky-400 flex items-center gap-1.5 mb-1.5">
              <UIcon name="i-lucide-check-circle" class="w-4 h-4" />
              <span>Ciri Thin Controller (Scale-Up Best Practice)</span>
            </h4>
            <ul class="space-y-1 text-neutral-700 dark:text-neutral-300 list-disc list-inside">
              <li>Hanya mem-parsing HTTP payload ke DTO dan memvalidasi format.</li>
              <li>Mendelegasikan 100% logika ke Service Layer via Interface.</li>
              <li>Memetakan output dan error ke HTTP status code yang tepat (200, 400, 404, 500).</li>
            </ul>
          </div>
        </div>
      </div>

      <!-- Code Snippet Box -->
      <div class="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-900 text-neutral-100 overflow-hidden">
        <div class="flex items-center justify-between px-4 py-2.5 bg-neutral-950 border-b border-neutral-800">
          <span class="text-xs font-mono font-bold text-sky-400">
            {{ currentPattern.controller_code_sample.title }} (Pola: {{ currentPattern.name }})
          </span>
          <span class="text-[10px] font-mono uppercase bg-neutral-800 px-2 py-0.5 rounded text-neutral-400">
            {{ currentPattern.controller_code_sample.language }}
          </span>
        </div>
        <div class="p-4 font-mono text-xs text-neutral-200 leading-relaxed overflow-x-auto">
          <pre>{{ currentPattern.controller_code_sample.code }}</pre>
        </div>
        <div class="px-4 py-2 bg-neutral-950/60 border-t border-neutral-800/80 text-xs text-neutral-400">
          <strong>Karakteristik:</strong> {{ currentPattern.controller_characteristics }}
        </div>
      </div>
    </div>

    <!-- Perspective 3: Functions vs Class / Struct -->
    <div v-else class="space-y-5">
      <div class="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-3">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-code" class="w-5 h-5 text-sky-600 dark:text-sky-400" />
          <h3 class="text-sm font-bold text-neutral-900 dark:text-white">
            Kaidah Penggunaan: Pure Functions vs Struct / Class dengan Interface
          </h3>
        </div>
        <p class="text-xs sm:text-sm text-neutral-600 dark:text-neutral-300 leading-relaxed">
          Di ekosistem Go, TypeScript, dan Python, memilih antara pure function atau struct method menentukan kemudahan pengujian dan modularitas sistem:
        </p>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs pt-1">
          <div class="p-3.5 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-800/50">
            <h4 class="font-bold text-sky-600 dark:text-sky-400 mb-1">
              Kapan Memakai Pure Function?
            </h4>
            <p class="text-neutral-600 dark:text-neutral-400 mb-2">
              Fungsi murni tanpa side-effect, output hanya tergantung argumen input.
            </p>
            <ul class="space-y-1 text-neutral-700 dark:text-neutral-300 list-disc list-inside">
              <li>Generasi slug: <code>Slugify(name)</code></li>
              <li>Validasi regex email / phone</li>
              <li>Kalkulasi skor diagnostik atau diskon</li>
              <li>Format tanggal / string transformation</li>
            </ul>
          </div>

          <div class="p-3.5 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-800/50">
            <h4 class="font-bold text-blue-600 dark:text-blue-400 mb-1">
              Kapan Memakai Struct / Class + Interface?
            </h4>
            <p class="text-neutral-600 dark:text-neutral-400 mb-2">
              Unit kode yang memiliki dependensi eksternal, koneksi I/O, atau state.
            </p>
            <ul class="space-y-1 text-neutral-700 dark:text-neutral-300 list-disc list-inside">
              <li>Akses database (Repository Struct)</li>
              <li>Komunikasi jaringan (HTTP Client / Cloud SDK)</li>
              <li>Lifecycle server (Inisialisasi & Shutdown)</li>
              <li>Inversion of Control untuk Mocking saat unit test</li>
            </ul>
          </div>
        </div>
      </div>

      <!-- Code Snippet Box -->
      <div class="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-900 text-neutral-100 overflow-hidden">
        <div class="flex items-center justify-between px-4 py-2.5 bg-neutral-950 border-b border-neutral-800">
          <span class="text-xs font-mono font-bold text-sky-400">
            {{ currentPattern.logic_code_sample.title }} (Pola: {{ currentPattern.name }})
          </span>
          <span class="text-[10px] font-mono uppercase bg-neutral-800 px-2 py-0.5 rounded text-neutral-400">
            {{ currentPattern.logic_code_sample.language }}
          </span>
        </div>
        <div class="p-4 font-mono text-xs text-neutral-200 leading-relaxed overflow-x-auto">
          <pre>{{ currentPattern.logic_code_sample.code }}</pre>
        </div>
        <div class="px-4 py-2 bg-neutral-950/60 border-t border-neutral-800/80 text-xs text-neutral-400">
          <strong>Karakteristik:</strong> {{ currentPattern.function_vs_class_philosophy }}
        </div>
      </div>
    </div>
  </div>
</template>
