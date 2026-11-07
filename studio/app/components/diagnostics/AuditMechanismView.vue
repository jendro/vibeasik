<script setup lang="ts">
// Penjelasan statis cara kerja mesin diagnostik (4 layer pemeriksaan).
</script>

<template>
  <div class="space-y-6 text-neutral-800 dark:text-neutral-200">
    <!-- 1. Executive Summary: Jawaban Langsung -->
    <div class="p-4 rounded-xl border border-violet-200 dark:border-violet-800/60 bg-violet-50/50 dark:bg-violet-950/20 space-y-3">
      <div class="flex items-center gap-2">
        <span class="p-1.5 rounded-lg bg-violet-600 text-white">
          <UIcon name="i-lucide-help-circle" class="w-4 h-4" />
        </span>
        <h3 class="text-sm font-bold text-violet-950 dark:text-violet-200">
          Apakah Audit Mengecek Seluruh File Satu per Satu?
        </h3>
      </div>
      <p class="text-xs leading-relaxed text-neutral-700 dark:text-neutral-300">
        <strong>Tidak secara membabi buta.</strong> Mesin diagnostik Vibeasik menerapkan pendekatan <strong>Selective Targeted Traversal</strong> (Penelusuran Selektif Berbasis Whitelist &amp; Pruning Direktori).
      </p>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3 pt-1 text-xs">
        <div class="p-3 rounded-lg bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 space-y-1.5">
          <div class="flex items-center gap-1.5 font-bold text-rose-700 dark:text-rose-400">
            <UIcon name="i-lucide-folder-x" class="w-4 h-4" />
            <span>Dilewati Otomatis (Pruning):</span>
          </div>
          <p class="text-[11px] text-neutral-600 dark:text-neutral-400 leading-normal">
            Folder non-sumber langsung diabaikan tanpa dibuka: <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">node_modules/</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">vendor/</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.git/</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.nuxt/</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">dist/</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">build/</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">bin/</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">tmp/</code>. File tes (<code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">*_test.go</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.test.ts</code>), mock, dan minified bundle (<code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.min.js</code>) juga dilewati.
          </p>
        </div>
        <div class="p-3 rounded-lg bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 space-y-1.5">
          <div class="flex items-center gap-1.5 font-bold text-emerald-700 dark:text-emerald-400">
            <UIcon name="i-lucide-file-code" class="w-4 h-4" />
            <span>Dipindai Mendalam (Line-by-Line):</span>
          </div>
          <p class="text-[11px] text-neutral-600 dark:text-neutral-400 leading-normal">
            Hanya berkas kode aktif yang masuk whitelist: <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.go</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.ts</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.js</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.vue</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.dart</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.php</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.py</code>, <code class="font-mono bg-neutral-100 dark:bg-neutral-800 px-1 py-0.5 rounded">.java</code>. Berkas-berkas ini diperiksa <strong>baris demi baris</strong> dengan pelacak konteks loop dan blok pasangan resource.
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2 p-2.5 rounded-lg bg-neutral-100/80 dark:bg-neutral-800/60 border border-neutral-200/80 dark:border-neutral-700/60 text-[11px]">
        <UIcon name="i-lucide-gauge" class="w-4 h-4 text-violet-600 dark:text-violet-400 shrink-0" />
        <span>
          <strong>Safety Cap:</strong> Dibatasi maksimal <code class="font-mono font-bold">150 berkas</code> per komponen untuk menjamin latensi diagnosa instan (&lt;1 detik) pada lingkungan lokal.
        </span>
      </div>
    </div>

    <!-- 2. Arsitektur 4-Layer Inspection Engine -->
    <div class="space-y-3">
      <div class="flex items-center gap-2">
        <span class="p-1.5 rounded-lg bg-neutral-900 text-white dark:bg-white dark:text-neutral-900">
          <UIcon name="i-lucide-layers" class="w-4 h-4" />
        </span>
        <h3 class="text-sm font-bold text-neutral-900 dark:text-white">
          Arsitektur 4-Layer Pemeriksaan Diagnostik
        </h3>
      </div>

      <div class="space-y-3 text-xs">
        <!-- Layer 1 -->
        <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-2">
          <div class="flex items-center justify-between">
            <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-2">
              <span class="w-5 h-5 rounded-full bg-violet-100 dark:bg-violet-950 text-violet-700 dark:text-violet-300 font-mono text-[11px] flex items-center justify-center font-bold">1</span>
              Layer 1: Pre-flight &amp; Lingkungan Root (Struktur Proyek)
            </span>
            <span class="px-2 py-0.5 rounded text-[10px] font-mono font-medium bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-300">Fast Scan (~5ms)</span>
          </div>
          <p class="text-neutral-600 dark:text-neutral-400 text-[11px] leading-relaxed">
            Memvalidasi integritas fisik proyek sebelum membaca kode:
          </p>
          <ul class="list-disc list-inside space-y-1 text-[11px] text-neutral-700 dark:text-neutral-300 pl-1">
            <li><strong>Validasi Path Fisik:</strong> Memastikan direktori root dan komponen benar-benar ada di disk (<code class="font-mono">os.Stat</code>).</li>
            <li><strong>Deteksi Bentrokan Port:</strong> Menangkap tabrakan port internal antar-komponen (Backend vs Frontend pada port sama) serta shared dev port antar-proyek.</li>
            <li><strong>Higiene Berkas Wajib:</strong> Memeriksa keberadaan <code class="font-mono">.gitignore</code> (memastikan <code class="font-mono">.env</code> tidak terekspos), <code class="font-mono">README.md</code>, <code class="font-mono">.env.example</code>, dan suite pengujian otomatis.</li>
            <li><strong>Git Hygiene:</strong> Mendeteksi uncommitted working tree sebelum refactor besar dilakukan.</li>
          </ul>
        </div>

        <!-- Layer 2 -->
        <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-2">
          <div class="flex items-center justify-between">
            <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-2">
              <span class="w-5 h-5 rounded-full bg-violet-100 dark:bg-violet-950 text-violet-700 dark:text-violet-300 font-mono text-[11px] flex items-center justify-center font-bold">2</span>
              Layer 2: Traversal &amp; Seleksi Berkas Terarah (Smart Filter)
            </span>
            <span class="px-2 py-0.5 rounded text-[10px] font-mono font-medium bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-300">Whitelist Filter</span>
          </div>
          <p class="text-neutral-600 dark:text-neutral-400 text-[11px] leading-relaxed">
            Menyusuri pohon folder dengan aturan eliminasi ketat:
          </p>
          <ul class="list-disc list-inside space-y-1 text-[11px] text-neutral-700 dark:text-neutral-300 pl-1">
            <li><strong>Pruning Blacklist:</strong> Folder non-sumber langsung di-skip (<code class="font-mono">filepath.SkipDir</code>).</li>
            <li><strong>Ekstensi Whitelist:</strong> Hanya berkas kode sumber (.go, .ts/.tsx, .js/.jsx, .vue, .svelte, .dart, .kt, .java, .php, .py) serta layout Android <code class="font-mono">res/layout*/*.xml</code> yang masuk ke proses inspeksi.</li>
            <li><strong>Aset Media:</strong> Gambar &amp; video (.png, .jpg, .gif, .webp, .svg, .mp4, ...) hanya diukur ukurannya tanpa dibaca isinya.</li>
            <li><strong>Isolasi Komponen:</strong> Backend, Frontend Client, Frontend Admin, dan Mobile dipindai sesuai peran masing-masing.</li>
          </ul>
        </div>

        <!-- Layer 3 -->
        <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-2">
          <div class="flex items-center justify-between">
            <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-2">
              <span class="w-5 h-5 rounded-full bg-violet-100 dark:bg-violet-950 text-violet-700 dark:text-violet-300 font-mono text-[11px] flex items-center justify-center font-bold">3</span>
              Layer 3: Inspeksi Kontekstual &amp; AST Heuristics (Deep Inspection)
            </span>
            <span class="px-2 py-0.5 rounded text-[10px] font-mono font-medium bg-rose-100 dark:bg-rose-950/60 text-rose-700 dark:text-rose-300">Inti Analisis</span>
          </div>
          <p class="text-neutral-600 dark:text-neutral-400 text-[11px] leading-relaxed">
            Membaca berkas kode baris demi baris menggunakan parser kontekstual:
          </p>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[11px] pt-1">
            <div class="p-2.5 rounded bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200/60 dark:border-neutral-700/60 space-y-1">
              <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1">
                <UIcon name="i-lucide-repeat" class="w-3.5 h-3.5 text-amber-500" />
                State Machine Loop (N+1 Query)
              </span>
              <p class="text-neutral-600 dark:text-neutral-400">
                Melacak kedalaman kurung kurawal <code class="font-mono">{ }</code> pada blok <code class="font-mono">for</code>, <code class="font-mono">while</code>, atau <code class="font-mono">.map()</code> untuk mendeteksi kueri database berulang di dalam perulangan.
              </p>
            </div>
            <div class="p-2.5 rounded bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200/60 dark:border-neutral-700/60 space-y-1">
              <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1">
                <UIcon name="i-lucide-shield-alert" class="w-3.5 h-3.5 text-rose-500" />
                Celah OWASP &amp; Taint Signatures
              </span>
              <p class="text-neutral-600 dark:text-neutral-400">
                Mendeteksi SQL Injection (string concat), Command Injection / RCE (shell exec), Path Traversal, XSS (raw HTML v-html), dan Hardcoded Secrets (&gt;12 karakter).
              </p>
            </div>
            <div class="p-2.5 rounded bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200/60 dark:border-neutral-700/60 space-y-1">
              <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1">
                <UIcon name="i-lucide-plug-zap" class="w-3.5 h-3.5 text-sky-500" />
                Pasangan Resource Lifecycle
              </span>
              <p class="text-neutral-600 dark:text-neutral-400">
                Mengecek pasangan pemanggilan <code class="font-mono">db.Query</code> dengan <code class="font-mono">defer rows.Close()</code>, <code class="font-mono">db.Begin</code> dengan <code class="font-mono">defer tx.Rollback()</code>, serta lifecycle <code class="font-mono">dispose()</code> pada Flutter controller.
              </p>
            </div>
            <div class="p-2.5 rounded bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200/60 dark:border-neutral-700/60 space-y-1">
              <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1">
                <UIcon name="i-lucide-activity" class="w-3.5 h-3.5 text-violet-500" />
                Resiliensi &amp; Arsitektur API
              </span>
              <p class="text-neutral-600 dark:text-neutral-400">
                Mendeteksi rute mutasi tanpa auth guard, wildcard CORS (*), God Controller (&gt;400 baris), error yang diabaikan (<code class="font-mono">_ = err</code>), blocking sync I/O di NodeJS, dan dialog browser native (<code class="font-mono">alert()</code>).
              </p>
            </div>
            <div class="p-2.5 rounded bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200/60 dark:border-neutral-700/60 space-y-1">
              <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1">
                <UIcon name="i-lucide-layout-template" class="w-3.5 h-3.5 text-emerald-500" />
                Tampilan &amp; Performa Frontend
              </span>
              <p class="text-neutral-600 dark:text-neutral-400">
                Ukuran halaman/komponen (&gt;600 baris), <code class="font-mono">v-for</code> tanpa key, gambar tanpa dimensi/alt/lazy, elemen klik non-aksesibel, font &lt;11px, lebar tetap non-responsif, import library berat, listener bocor, serta warna/inline style/z-index di luar design token.
              </p>
            </div>
            <div class="p-2.5 rounded bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200/60 dark:border-neutral-700/60 space-y-1">
              <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1">
                <UIcon name="i-lucide-smartphone" class="w-3.5 h-3.5 text-sky-500" />
                Android &amp; Flutter
              </span>
              <p class="text-neutral-600 dark:text-neutral-400">
                God Activity, blocking main thread (ANR), GlobalScope, Context statis, <code class="font-mono">notifyDataSetChanged()</code>, state Compose tanpa <code class="font-mono">remember</code>, layout XML terlalu dalam, textSize dp, area sentuh &lt;48dp, serta ListView non-builder &amp; <code class="font-mono">shrinkWrap</code> di Flutter.
              </p>
            </div>
            <div class="p-2.5 rounded bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200/60 dark:border-neutral-700/60 space-y-1 sm:col-span-2">
              <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1">
                <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 text-violet-500" />
                Aturan Sesuai Bahasa &amp; Review LLM via Antrean
              </span>
              <p class="text-neutral-600 dark:text-neutral-400">
                Setiap berkas diklasifikasi dulu (server, UI web, Android, Flutter) sehingga aturan rute/SQL tidak dijalankan di komponen UI, aturan Android tidak dijalankan di Spring/Ktor, dan kompleksitas fungsi dihitung dengan sintaks Go, TS/JS, Kotlin, Java, Dart, PHP, atau Python. Tombol <strong>Review LLM</strong> menambah job <code class="font-mono">llm_audit</code> ke antrean (progres tampil di bar Antrean) dan memakai checklist khusus bahasa tiap berkas.
              </p>
            </div>
          </div>
        </div>

        <!-- Layer 4 -->
        <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 space-y-2">
          <div class="flex items-center justify-between">
            <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-2">
              <span class="w-5 h-5 rounded-full bg-violet-100 dark:bg-violet-950 text-violet-700 dark:text-violet-300 font-mono text-[11px] flex items-center justify-center font-bold">4</span>
              Layer 4: Skor Kesehatan, Deduplikasi &amp; Stateful Action
            </span>
            <span class="px-2 py-0.5 rounded text-[10px] font-mono font-medium bg-emerald-100 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300">Stateful Scoring</span>
          </div>
          <p class="text-neutral-600 dark:text-neutral-400 text-[11px] leading-relaxed">
            Menghasilkan skor 0–100 dan mengingat keputusan penanganan pengguna:
          </p>
          <ul class="list-disc list-inside space-y-1 text-[11px] text-neutral-700 dark:text-neutral-300 pl-1">
            <li><strong>Deduplikasi Akurat:</strong> Temuan digabungkan berdasarkan signature unik <code class="font-mono">RuleID + TargetFile + Line</code>.</li>
            <li><strong>Deductive Penalty Formula:</strong> Skor dimulai dari 100, dikurangi bobot penalti temuan aktif (Critical: -8 s/d -30, Warning: -4 s/d -15, Info: -1 s/d -4).</li>
            <li><strong>Carry-Over Status:</strong> Temuan yang telah ditandai <code class="font-mono">Resolved</code> atau <code class="font-mono">Ignored</code> disimpan di database lokal sehingga tidak memotong skor kesehatan pada pemindaian berikutnya.</li>
          </ul>
        </div>
      </div>
    </div>

    <!-- 3. Tabel Komparasi Whitelist vs Blacklist -->
    <div class="space-y-3">
      <div class="flex items-center gap-2">
        <span class="p-1.5 rounded-lg bg-neutral-900 text-white dark:bg-white dark:text-neutral-900">
          <UIcon name="i-lucide-list-filter" class="w-4 h-4" />
        </span>
        <h3 class="text-sm font-bold text-neutral-900 dark:text-white">
          Tabel Penapisan Berkas &amp; Direktori
        </h3>
      </div>

      <div class="border border-neutral-200 dark:border-neutral-800 rounded-xl overflow-hidden bg-white dark:bg-neutral-900">
        <table class="w-full text-[11px] text-left">
          <thead class="bg-neutral-100 dark:bg-neutral-800/80 text-neutral-700 dark:text-neutral-300 font-bold border-b border-neutral-200 dark:border-neutral-800">
            <tr>
              <th class="p-2.5">
                Target / Pola Berkas
              </th>
              <th class="p-2.5">
                Aksi Traversal
              </th>
              <th class="p-2.5">
                Rasional &amp; Alasan
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-neutral-100 dark:divide-neutral-800 font-sans">
            <tr>
              <td class="p-2.5 font-mono text-neutral-900 dark:text-white">
                node_modules/, vendor/
              </td>
              <td class="p-2.5 text-rose-600 dark:text-rose-400 font-semibold">
                ❌ Skip Directory
              </td>
              <td class="p-2.5 text-neutral-600 dark:text-neutral-400">
                Library eksternal; memindainya memicu ribuan false alarm dan membebani RAM.
              </td>
            </tr>
            <tr>
              <td class="p-2.5 font-mono text-neutral-900 dark:text-white">
                .git/, .vscode/, .idea/
              </td>
              <td class="p-2.5 text-rose-600 dark:text-rose-400 font-semibold">
                ❌ Skip Directory
              </td>
              <td class="p-2.5 text-neutral-600 dark:text-neutral-400">
                Metadata VCS dan konfigurasi editor, bukan logika aplikasi.
              </td>
            </tr>
            <tr>
              <td class="p-2.5 font-mono text-neutral-900 dark:text-white">
                dist/, .nuxt/, build/, bin/
              </td>
              <td class="p-2.5 text-rose-600 dark:text-rose-400 font-semibold">
                ❌ Skip Directory
              </td>
              <td class="p-2.5 text-neutral-600 dark:text-neutral-400">
                Hasil build / output bundle sementara yang bukan source code asli.
              </td>
            </tr>
            <tr>
              <td class="p-2.5 font-mono text-neutral-900 dark:text-white">
                *_test.go, *.test.ts, mock_*
              </td>
              <td class="p-2.5 text-amber-600 dark:text-amber-400 font-semibold">
                ⚡ Skip File
              </td>
              <td class="p-2.5 text-neutral-600 dark:text-neutral-400">
                Berkas tes dan mock diabaikan agar mock query tidak disalahpahami sebagai kode produksi.
              </td>
            </tr>
            <tr>
              <td class="p-2.5 font-mono text-neutral-900 dark:text-white">
                *.min.js
              </td>
              <td class="p-2.5 text-amber-600 dark:text-amber-400 font-semibold">
                ⚡ Skip File
              </td>
              <td class="p-2.5 text-neutral-600 dark:text-neutral-400">
                Kode minified yang dipadatkan sulit di-parse secara kontekstual tanpa AST penuh.
              </td>
            </tr>
            <tr>
              <td class="p-2.5 font-mono text-neutral-900 dark:text-white">
                .go, .ts, .js, .vue, .dart, .php, .py
              </td>
              <td class="p-2.5 text-emerald-600 dark:text-emerald-400 font-semibold">
                ✅ Full Line Inspection
              </td>
              <td class="p-2.5 text-neutral-600 dark:text-neutral-400">
                Kode sumber aktif aplikasi; dipindai mendalam baris demi baris hingga batas 150 berkas.
              </td>
            </tr>
            <tr>
              <td class="p-2.5 font-mono text-neutral-900 dark:text-white">
                .gitignore, .env.example, README.md
              </td>
              <td class="p-2.5 text-emerald-600 dark:text-emerald-400 font-semibold">
                ✅ Root &amp; Component Check
              </td>
              <td class="p-2.5 text-neutral-600 dark:text-neutral-400">
                Konfigurasi esensial untuk mencegah kebocoran secret dan menjamin dokumentasi.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 4. Panduan Mendapatkan Hasil Seakurat Mungkin -->
    <div class="p-4 rounded-xl border border-emerald-200 dark:border-emerald-800/60 bg-emerald-50/50 dark:bg-emerald-950/20 space-y-3">
      <div class="flex items-center gap-2">
        <span class="p-1.5 rounded-lg bg-emerald-600 text-white">
          <UIcon name="i-lucide-sparkles" class="w-4 h-4" />
        </span>
        <h3 class="text-sm font-bold text-emerald-950 dark:text-emerald-200">
          Panduan Strategis Memperoleh Hasil Audit Seakurat Mungkin
        </h3>
      </div>
      <p class="text-xs text-neutral-700 dark:text-neutral-300 leading-relaxed">
        Untuk memastikan tidak ada berkas penting yang terlewat dan hasil diagnosis mencapai tingkat akurasi 100%:
      </p>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
        <div class="p-3 rounded-lg bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 space-y-1">
          <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1.5">
            <span class="w-4 h-4 rounded-full bg-emerald-100 dark:bg-emerald-950 text-emerald-700 dark:text-emerald-300 font-mono text-[10px] flex items-center justify-center font-bold">1</span>
            Partisi Komponen Modular (Hindari Monolith Raksasa)
          </span>
          <p class="text-[11px] text-neutral-600 dark:text-neutral-400 leading-normal">
            Karena limit pemindaian adalah <strong>150 file per komponen</strong>, daftarkan sub-service atau modul besar sebagai komponen terpisah di Vibeasik. Setiap komponen mendapat kuota 150 file independen sehingga seluruh berkas terinspeksi tanpa batas.
          </p>
        </div>

        <div class="p-3 rounded-lg bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 space-y-1">
          <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1.5">
            <span class="w-4 h-4 rounded-full bg-emerald-100 dark:bg-emerald-950 text-emerald-700 dark:text-emerald-300 font-mono text-[10px] flex items-center justify-center font-bold">2</span>
            Sinergi Multi-Engine (Static + Fast Scan AI + REA)
          </span>
          <p class="text-[11px] text-neutral-600 dark:text-neutral-400 leading-normal">
            Aturan statis memberikan kepastian 100% pada signature sintaks (SQLi, Secret, Port). Padukan dengan <strong>Fast Scan AI</strong> atau <strong>AI Review</strong> untuk memvalidasi logika semantik antar-layanan (misal keselarasan variabel API endpoint).
          </p>
        </div>

        <div class="p-3 rounded-lg bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 space-y-1">
          <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1.5">
            <span class="w-4 h-4 rounded-full bg-emerald-100 dark:bg-emerald-950 text-emerald-700 dark:text-emerald-300 font-mono text-[10px] flex items-center justify-center font-bold">3</span>
            Manfaatkan Kurasi Stateful (Resolve &amp; Ignore)
          </span>
          <p class="text-[11px] text-neutral-600 dark:text-neutral-400 leading-normal">
            Pada kasus yang memang sengaja didesain terbuka (misalnya webhook publik Stripe tanpa JWT), tandai temuan dengan tombol <strong>"Abaikan (Ignore)"</strong>. Sistem akan mengingat keputusan ini sehingga diagnosis berikutnya 100% bebas dari noise.
          </p>
        </div>

        <div class="p-3 rounded-lg bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 space-y-1">
          <span class="font-bold text-neutral-900 dark:text-white flex items-center gap-1.5">
            <span class="w-4 h-4 rounded-full bg-emerald-100 dark:bg-emerald-950 text-emerald-700 dark:text-emerald-300 font-mono text-[10px] flex items-center justify-center font-bold">4</span>
            Higiene Path &amp; Konfigurasi Berkas
          </span>
          <p class="text-[11px] text-neutral-600 dark:text-neutral-400 leading-normal">
            Pastikan direktori absolut komponen valid, sediakan <code class="font-mono">.env.example</code>, dan sertakan aturan <code class="font-mono">.env*</code> di <code class="font-mono">.gitignore</code> untuk menghindari penalti arsitektur dan kebocoran kredensial.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
