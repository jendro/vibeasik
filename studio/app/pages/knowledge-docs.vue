<script setup lang="ts">
/**
 * Dokumentasi Pusat Pengetahuan: algoritma, pemrosesan input pekerjaan, pengaruh tiap faktor ke hasil, mekanisme
 * pengetahuan masuk, dipakai, dan belajar. Sumber kebenaran teknis: local-engine/internal/knowledge/README.md.
 */
definePageMeta({ fluid: true })

const TOC = [
  { id: 'ringkas', label: 'Ringkasnya' },
  { id: 'node', label: 'Isi satu butir' },
  { id: 'input', label: 'Pemrosesan input' },
  { id: 'skor', label: 'Rumus skor & pengaruhnya' },
  { id: 'algoritma', label: 'Algoritma yang dipakai' },
  { id: 'masuk', label: 'Pengetahuan baru masuk' },
  { id: 'dipakai', label: 'Pengetahuan digunakan' },
  { id: 'belajar', label: 'Mekanisme belajar' },
  { id: 'kurator', label: 'Kurator berkala' },
  { id: 'celah', label: 'Belajar mandiri dari celah' },
  { id: 'angka', label: 'Angka pengaturan' },
  { id: 'batas', label: 'Batasan' }
]

const PIPELINE = [
  {
    icon: 'i-lucide-text-cursor-input', title: 'Teks pekerjaan dirakit',
    what: 'Todo: judul + keterangan + aplikasi target + kategori. Audit: path berkas + bahasa + judul temuan. Rapat PM: permintaan + deskripsi proyek.',
    effect: 'Makin spesifik teksnya (nama berkas, framework, kata kunci masalah), makin tepat butir yang terambil.'
  },
  {
    icon: 'i-lucide-split', title: 'Tokenisasi',
    what: 'Huruf kecil, camelCase/snake_case dipecah (fetchUserData → fetch user data), kata umum dibuang, jembatan Indonesia ↔ Inggris (validasi ↔ validation).',
    effect: 'Pekerjaan berbahasa Indonesia tetap menemukan patokan berbahasa Inggris dan sebaliknya.'
  },
  {
    icon: 'i-lucide-scan-search', title: 'Deteksi domain',
    what: 'Dari ekstensi berkas (.go → go, .vue → web, .dart → flutter; ekstensi asing → ekosistem baru), kata kunci (laravel → php, xss → security), atau stack komponen proyek.',
    effect: 'Ekosistem bersifat eksklusif: pekerjaan Go tidak menerima patokan Flutter/PHP. Area (security, testing, …) tetap boleh ikut. Ekosistem tanpa butir sama sekali → celah.'
  },
  {
    icon: 'i-lucide-sigma', title: 'Vektor TF-IDF kueri',
    what: 'Tiap kata diberi bobot tf × idf; kata yang tidak ada di korpus dibuang; vektor dinormalisasi.',
    effect: 'Kata langka (mis. "jwt", "prisma") jauh lebih menentukan daripada kata umum (mis. "buat", "halaman").'
  },
  {
    icon: 'i-lucide-filter', title: 'Kandidat & penyaringan',
    what: 'Kemiripan kosinus + 0,12 bila domain sama; ambil yang ≥ 0,08 (maks 80). Buang: keahlian pribadi pegawai lain, ekosistem lain, jenis/sumber yang dikecualikan, butir merugikan (✗ ≥ 3 & utilitas < 25%).',
    effect: 'Butir yang terbukti keliru tidak pernah sampai ke prompt; keahlian pribadi hanya dipakai pemiliknya.'
  },
  {
    icon: 'i-lucide-calculator', title: 'Skor gabungan',
    what: '0,55 relevansi + 0,20 utilitas + 0,15 bobot + 0,10 kebaruan (lihat kalkulator di bawah).',
    effect: 'Urutan ditentukan terutama oleh kecocokan, lalu oleh bukti & kepentingan.'
  },
  {
    icon: 'i-lucide-waypoints', title: 'Perluasan lewat tautan',
    what: 'Tetangga bertaut dari butir teratas ikut dinilai: 0,7 × skor asal × kekuatan tautan + 0,3 × skor dirinya.',
    effect: 'Pelajaran terkait yang kata-katanya berbeda tetap bisa ikut (aktivasi menyebar satu langkah).'
  },
  {
    icon: 'i-lucide-shapes', title: 'Keberagaman & batas jumlah',
    what: 'Satu klaster konsep maksimal separuh hasil; diambil teratas: todo 15 · audit 8 · rapat 10 · spesialis 6.',
    effect: 'Prompt tidak dipenuhi 10 butir yang membahas hal sama; total tetap ringkas (±1–2 ribu token).'
  },
  {
    icon: 'i-lucide-file-text', title: 'Ditulis ke prompt',
    what: 'Format "- [K<id> · jenis · domain] Judul: isi (✓n ✗n)" + peringatan celah + ajakan melapor K<id> yang keliru.',
    effect: 'Agent tahu asal & tingkat bukti tiap butir, dan bisa melaporkan butir yang menyesatkan.'
  },
  {
    icon: 'i-lucide-notebook-pen', title: 'Dicatat untuk dinilai',
    what: 'Bila ada konteks (todo:<id>, audit:<proyek>:<path>, plan:<proyek>): tiap butir dicatat "menunggu nilai", hitungan pakai & waktu terakhir dipakai naik, celah dicatat.',
    effect: 'Inilah yang memungkinkan hasil kerja kembali menilai butir yang dipakai.'
  }
]

const ALGOS = [
  ['TF-IDF jarang + kosinus', 'Mengubah teks jadi vektor & mengukur kemiripan', 'Information retrieval klasik'],
  ['Tokenizer ID ↔ EN + camelCase', '"validasi" menemukan "validation"; fetchUserData dipecah', '—'],
  ['Posting list (indeks terbalik)', 'Mencari tetangga terdekat tanpa membandingkan semua pasangan', 'IR klasik'],
  ['Skor relevansi + utilitas + bobot + kebaruan', 'Memilih butir paling berguna', 'Generative Agents (Park dkk., 2023)'],
  ['Utilitas Beta (Bayes) (✓+1)/(✓+✗+2)', 'Seberapa terbukti berguna; butir baru = 0,5', 'ACE, ExpeL'],
  ['Peluruhan eksponensial e^(−hari/45)', 'Butir yang lama tak dipakai turun pelan', 'MemoryBank / kurva lupa'],
  ['Aktivasi menyebar 1 langkah', 'Tetangga bertaut ikut dipertimbangkan', 'HippoRAG, A-MEM'],
  ['Tautan Zettelkasten (k = 3)', 'Menghubungkan butir yang isinya berdekatan (≥ 0,25)', 'A-MEM'],
  ['Leader clustering satu lintasan', 'Klaster konsep (kosinus ke pusat ≥ 0,3) + label kata pusat', 'RAPTOR / GraphRAG versi ringan'],
  ['Dedup & peleburan', 'Tulis ≥ 0,8 diperkuat; kurasi ≥ 0,85 dilebur — memori berkembang, tidak berlipat', 'ACE, ExpeL'],
  ['Kurikulum otomatis dari celah', 'Hal baru → agenda belajar pegawai → butir baru', 'Voyager (2023)'],
  ['Penilaian kredit dari hasil', 'Nilai QA / vonis audit membagi ✓/✗ ke butir yang dipakai', 'ACE (generator → reflector → curator)'],
  ['PageRank berbobot (peta jaring)', 'Ukuran node di Jaring Pengetahuan = pengaruh', 'PageRank (Brin & Page)']
]

const SOURCES = [
  ['guidelines/*.json', 'Sinkron', 'Patokan · pustaka bersama · bobot dari severity', 'Tiap kurasi (20 dtk setelah engine menyala, lalu tiap 15 menit)'],
  ['Checklist review audit per bahasa', 'Sinkron', 'Patokan · pustaka bersama', 'Tiap kurasi'],
  ['Pengetahuan pegawai', 'Sinkron', 'Prosedur/pelajaran · pribadi pegawai', 'Tiap kurasi + tepat sebelum pegawai mengerjakan todo'],
  ['Vonis audit beralasan', 'Learn', 'Pelajaran "Masalah nyata: …" / "Bukan masalah: …"', 'Saat vonis disimpan (studio, cek LLM, MCP)'],
  ['Penilai QA', 'Learn', 'Best practice umum (sumber qa_review)', 'Tiap review / QA gate (maks 3 / 2 butir)'],
  ['Agenda belajar celah', 'Learn', 'Konsep/prosedur/pelajaran · pustaka bersama · kredit pegawai', 'Saat agenda celah selesai'],
  ['Galeri Desain & teori desain', 'Sinkron', 'Konsep/patokan · sumber berkedudukan tinggi', 'Tiap kurasi'],
  ['Manusia', 'Learn', 'Tombol "Ajarkan" di Pusat Pengetahuan', 'Kapan saja'],
  ['Agent coding', 'Learn', 'MCP vibeasik_knowledge_learn', 'Kapan saja']
]

const USERS = [
  ['Pegawai mengerjakan todo', 'judul + keterangan + aplikasi + kategori todo; keahlian pribadinya + pustaka bersama', '15', 'todo:<id>'],
  ['Review audit LLM per berkas', 'path + bahasa + judul temuan statis; checklist dilewati (sudah di prompt sistem)', '8', 'audit:<proyek>:<path>'],
  ['Rapat PM — konteks bersama', 'permintaan rapat + deskripsi proyek; domain = stack komponen', '10', 'plan:<proyek>'],
  ['Rapat PM — tiap spesialis', 'nama + fokus peran + deskripsi proyek', '6', '—'],
  ['Penilai QA (rubrik)', 'patokan relevan untuk hasil kerja yang dinilai', '8–10', 'dinilai per butir'],
  ['Agent luar (MCP)', 'vibeasik_knowledge_recall — teks bebas', '10', '—'],
  ['Manusia (studio)', 'Uji Temu-kembali, pencarian, simulasi Jaring', '12–15', '— (pratinjau)']
]

const LEARN = [
  ['Review QA menilai todo (0–100)', 'Semua butir yang dipakai todo itu: nilai ≥ 75 → ✓+1, < 50 → ✗+1, 50–74 netral (Reward).'],
  ['Penilai QA menilai per butir', 'Status per K<id>: followed → ✓, wrong (patokannya keliru) → ✗, violated / n/a → tetap. Butir yang dinilai eksplisit tidak dinilai dua kali (Judge).'],
  ['Vonis audit "temuan benar"', 'Butir yang dipakai review berkas itu dinilai 85 (✓); patokan dasarnya ✓+1; alasan menjadi pelajaran "Masalah nyata: …".'],
  ['Vonis audit "audit keliru"', 'Butir yang dipakai dinilai 35 (✗); patokan dasarnya ✗+1; alasan + saran menjadi pelajaran "Bukan masalah: …".'],
  ['Umpan balik langsung', '👍/👎 di studio atau MCP vibeasik_knowledge_feedback → ✓+1 / ✗+1.'],
  ['Butir yang sama diajarkan lagi', 'Kemiripan ≥ 0,8 → butir lama diperkuat: ✓+1, bobot +3, isinya dilengkapi (bukan digandakan).'],
  ['PELAJARAN: di catatan penyelesaian', 'Dinilai QA lalu menjadi keahlian pribadi pegawai → disinkronkan sebagai butir baru; yang terbukti dipromosikan ke pustaka bersama.']
]

const CURATOR = [
  ['Sinkron sumber', 'guideline, checklist audit, pengetahuan pegawai, Galeri Desain. Bukti ✓/✗ tidak pernah ditimpa; butir yang hilang dari sumber dipensiunkan.'],
  ['Lebur duplikat', 'Kemiripan ≥ 0,85 di lingkup pemilik yang sama → yang terkuat (bobot + 5×(✓−✗) + pakai) dipertahankan, bukti dijumlahkan.'],
  ['Promosi', 'Keahlian pribadi dengan ✓ ≥ 3 dan utilitas ≥ 75% disalin ke pustaka bersama (kredit nama pegawai).'],
  ['Pangkas', 'Butir asli ✗ ≥ 3 & utilitas < 30% → dipensiunkan. Patokan berkas ✗ ≥ 3 & < 40% → ditandai "perlu ditinjau" (berkasnya yang diperbaiki); dilepas bila pulih ≥ 50%.'],
  ['Tautan ulang', 'Tiap butir ke 3 tetangga terdekat (kemiripan ≥ 0,25); keahlian pribadi hanya bertaut ke pemilik yang sama / pustaka bersama.'],
  ['Klaster ulang', 'Butir terkuat jadi pusat lebih dulu; gabung bila kemiripan ke pusat ≥ 0,3; label = 4 kata terberat; minimal 2 butir.'],
  ['Bersih-bersih', 'Catatan pemakaian lebih dari 30 hari dihapus.']
]

const CONSTS = [
  ['Dedup saat tulis', '0,8'], ['Tautan minimal / per butir', '0,25 / 3'], ['Lebur saat kurasi', '0,85'], ['Klaster', '0,3'],
  ['Relevansi minimal', '0,08'], ['Bonus domain', '0,12'], ['Peluruhan kebaruan', '45 hari'], ['Butir minimal sebelum celah dicatat', '20'],
  ['Interval kurator', '15 menit'], ['Simpan catatan pemakaian', '30 hari'], ['Percobaan belajar celah', '3']
]

const active = ref('ringkas')
const go = (id: string) => {
  active.value = id
  document.getElementById(id)?.scrollIntoView({ block: 'start' })
}
</script>

<template>
  <div class="w-full px-4 sm:px-6 lg:px-8 pt-6 pb-20 space-y-6">
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4 p-5 sm:p-6 rounded-2xl bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 shadow-xs">
      <div class="flex items-center gap-3.5 min-w-0">
        <div class="size-11 shrink-0 rounded-xl bg-linear-to-br from-indigo-500 to-emerald-500 text-white flex items-center justify-center shadow-xs">
          <UIcon name="i-lucide-book-marked" class="size-5.5" />
        </div>
        <div class="min-w-0">
          <h1 class="text-xl font-bold tracking-tight text-neutral-900 dark:text-white">
            Dokumentasi Pusat Pengetahuan
          </h1>
          <p class="text-xs text-neutral-500 mt-0.5">
            Algoritma, cara input pekerjaan diproses, pengaruh tiap faktor ke hasil, dan bagaimana pengetahuan masuk, dipakai, serta belajar.
          </p>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2 shrink-0">
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-library" to="/knowledge">
          Daftar butir
        </UButton>
        <UButton size="sm" color="primary" variant="soft" icon="i-lucide-waypoints" to="/knowledge-graph">
          Jaring & simulasi
        </UButton>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-[220px_1fr] gap-6">
      <nav class="hidden lg:block">
        <div class="sticky top-4 space-y-0.5">
          <p class="text-xs uppercase font-semibold text-neutral-500 px-2 pb-1">
            Isi
          </p>
          <button
            v-for="t in TOC"
            :key="t.id"
            type="button"
            class="w-full text-left text-sm px-2 py-1.5 rounded-md transition"
            :class="active === t.id ? 'bg-primary-50 dark:bg-primary-950/40 text-primary-700 dark:text-primary-300 font-medium' : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800'"
            @click="go(t.id)"
          >
            {{ t.label }}
          </button>
        </div>
      </nav>

      <article class="min-w-0 max-w-5xl space-y-10 text-sm text-neutral-700 dark:text-neutral-300 leading-relaxed">
        <!-- 1 -->
        <section id="ringkas" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Ringkasnya
          </h2>
          <p>
            Pusat Pengetahuan adalah <b>satu pintu</b> semua patokan dan pengetahuan <b>umum</b> pekerja AI vibeasik — pegawai, reviewer audit, dan peserta rapat PM. Fakta khusus satu proyek (fitur yang ada, keputusan teknis, roadmap) <b>tidak</b> di sini; itu milik dewan eksekutif proyek.
          </p>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
            <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
              <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
                <UIcon name="i-lucide-circle-alert" class="text-rose-500" /> Masalah
              </p>
              <p>Tiap agent AI mulai dari nol: lupa standar keamanan, mengulang bug & temuan audit yang sama, atau prompt dijejali ratusan guideline sekaligus.</p>
            </div>
            <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
              <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
                <UIcon name="i-lucide-lightbulb" class="text-amber-500" /> Solusi
              </p>
              <p>Pengetahuan disimpan sebagai butir bertaut. Tiap pekerjaan hanya membawa ±10–15 butir paling relevan & terbukti ke prompt-nya.</p>
            </div>
            <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
              <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
                <UIcon name="i-lucide-refresh-cw" class="text-emerald-500" /> Siklus
              </p>
              <p><b>pakai → dinilai → diperkuat/dilemahkan → dirapikan kurator → dipakai lagi</b>; yang belum dikenal → dipelajari pegawai → butir baru.</p>
            </div>
          </div>
          <pre class="overflow-x-auto rounded-xl bg-neutral-50 dark:bg-neutral-950 border border-neutral-200 dark:border-neutral-800 p-3 text-xs leading-relaxed font-mono text-neutral-700 dark:text-neutral-300">  SUMBER                         PUSAT PENGETAHUAN                     PEMAKAI
  guideline · checklist audit ─┐  ┌──────────────────────────────┐    ┌─ prompt pegawai (todo)
  keahlian pegawai ───────────┼─▶│ butir ── tautan ── butir     │───▶├─ review audit LLM
  vonis audit · penilai QA ───┤  │   klaster · bukti ✓/✗ · celah│    ├─ rapat PM & spesialis
  belajar celah · manusia ────┘  └──────────────▲───────────────┘    └─ MCP / studio
                                                 └── nilai QA · vonis audit · umpan balik ◀── hasil kerja</pre>
        </section>

        <!-- 2 -->
        <section id="node" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Isi satu butir (node)
          </h2>
          <div class="overflow-x-auto rounded-xl border border-neutral-200 dark:border-neutral-800">
            <table class="w-full text-sm [&_th]:text-left [&_th]:text-xs [&_th]:uppercase [&_th]:font-semibold [&_th]:text-neutral-500 [&_th]:bg-neutral-50 dark:[&_th]:bg-neutral-900 [&_th]:px-3 [&_th]:py-2 [&_td]:px-3 [&_td]:py-2 [&_td]:align-top [&_td]:border-t [&_td]:border-neutral-100 dark:[&_td]:border-neutral-800">
              <tbody>
                <tr><td>Jenis</td><td><b>Patokan</b> (wajib/dilarang) · <b>Prosedur</b> (cara mengerjakan) · <b>Pelajaran</b> (dari pengalaman) · <b>Konsep</b> (pengetahuan dasar & angka acuan)</td></tr>
                <tr><td>Domain</td><td>Ekosistem (go, web, flutter, php, …, atau ekosistem baru) <i>atau</i> area (security, testing, performance, database, …)</td></tr>
                <tr><td>Pemilik</td><td>Kosong = <b>pustaka bersama</b>; terisi = <b>keahlian pribadi</b> satu pegawai</td></tr>
                <tr><td>Bobot</td><td>Kepentingan awal 0–100</td></tr>
                <tr><td>Bukti ✓ / ✗</td><td>Dari hasil kerja (nilai QA, vonis audit, umpan balik) → <b>utilitas</b> = (✓+1)/(✓+✗+2)</td></tr>
                <tr><td>Pakai</td><td>Berapa kali & kapan terakhir disisipkan ke prompt</td></tr>
                <tr><td>Tautan & klaster</td><td>3 tetangga terdekat & kelompok konsep, dihitung ulang kurator</td></tr>
                <tr><td>Status</td><td>Aktif · dilebur ke butir lain · dipensiunkan · perlu ditinjau (patokan berkas yang sering keliru)</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- 3 -->
        <section id="input" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Pemrosesan input pekerjaan
          </h2>
          <p>Apa yang terjadi pada teks pekerjaan (todo, berkas audit, permintaan rapat) sampai menjadi bagian prompt — dan pengaruh tiap langkah ke hasil.</p>
          <ol class="space-y-2">
            <li v-for="(p, i) in PIPELINE" :key="p.title" class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 flex gap-3">
              <span class="size-8 rounded-lg bg-primary-50 dark:bg-primary-950/40 text-primary-600 dark:text-primary-400 flex items-center justify-center shrink-0">
                <UIcon :name="p.icon" class="size-4" />
              </span>
              <div class="min-w-0 grid grid-cols-1 md:grid-cols-2 gap-x-5 gap-y-1 flex-1">
                <p class="md:col-span-2 font-semibold text-neutral-900 dark:text-white">
                  {{ i + 1 }}. {{ p.title }}
                </p>
                <p><span class="block text-xs uppercase font-semibold tracking-wide text-neutral-400">Diapakan</span>{{ p.what }}</p>
                <p><span class="block text-xs uppercase font-semibold tracking-wide text-emerald-600 dark:text-emerald-400">Pengaruh</span>{{ p.effect }}</p>
              </div>
            </li>
          </ol>
          <p class="text-xs text-neutral-500">
            Ingin melihatnya pada data asli? Jalankan simulasi di <NuxtLink to="/knowledge-graph" class="text-primary-600 dark:text-primary-400 underline">Jaring Pengetahuan</NuxtLink> — termasuk prompt final yang dihasilkan.
          </p>
        </section>

        <!-- 4 -->
        <section id="skor" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Rumus skor & pengaruhnya ke hasil
          </h2>
          <pre class="overflow-x-auto rounded-xl bg-neutral-50 dark:bg-neutral-950 border border-neutral-200 dark:border-neutral-800 p-3 text-xs leading-relaxed font-mono text-neutral-700 dark:text-neutral-300">relevansi = (kosinus(kueri, butir) + 0,12 bila domain sama) / relevansi tertinggi di kandidat
skor      = 0,55 × relevansi + 0,20 × utilitas + 0,15 × bobot/100 + 0,10 × e^(−hari sejak dipakai / 45)
utilitas  = (✓ + 1) / (✓ + ✗ + 2)          — butir baru 0,5; Galeri Desain selalu 1
tetangga  = 0,7 × skor asal × kekuatan tautan + 0,3 × skor dirinya</pre>
          <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
            <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
              <UIcon name="i-lucide-sliders-horizontal" class="text-primary-500" /> Coba: geser faktornya
            </p>
            <KnowledgeScoreCalculator />
          </div>
        </section>

        <!-- 5 -->
        <section id="algoritma" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Algoritma yang dipakai
          </h2>
          <p>Semuanya murni Go di engine lokal — tanpa CGO dan tanpa layanan embedding luar, jadi cepat (milidetik untuk ribuan butir) dan data tidak keluar mesin.</p>
          <div class="overflow-x-auto rounded-xl border border-neutral-200 dark:border-neutral-800">
            <table class="w-full text-sm [&_th]:text-left [&_th]:text-xs [&_th]:uppercase [&_th]:font-semibold [&_th]:text-neutral-500 [&_th]:bg-neutral-50 dark:[&_th]:bg-neutral-900 [&_th]:px-3 [&_th]:py-2 [&_td]:px-3 [&_td]:py-2 [&_td]:align-top [&_td]:border-t [&_td]:border-neutral-100 dark:[&_td]:border-neutral-800">
              <thead>
                <tr><th>Algoritma</th><th>Untuk apa</th><th>Dasar riset</th></tr>
              </thead>
              <tbody>
                <tr v-for="a in ALGOS" :key="a[0]">
                  <td class="font-medium text-neutral-900 dark:text-white">
                    {{ a[0] }}
                  </td><td>{{ a[1] }}</td><td class="text-neutral-500">
                    {{ a[2] }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
            <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
              <UIcon name="i-lucide-sigma" class="text-indigo-500" /> Vektor TF-IDF
            </p>
            <pre class="overflow-x-auto rounded-xl bg-neutral-50 dark:bg-neutral-950 border border-neutral-200 dark:border-neutral-800 p-3 text-xs leading-relaxed font-mono text-neutral-700 dark:text-neutral-300 mt-0">frekuensi kata: judul ×2, domain + tag ×2, isi ×1
tf  = 1 + ln(frekuensi)        idf = ln((N + 1) / (df + 0,5)) + 1
vektor = tf × idf, dinormalisasi;  kemiripan(a, b) = a · b (kosinus)</pre>
            <p class="text-xs text-neutral-500 mt-2">
              Indeks disimpan di memori dan dibangun ulang hanya bila jumlah butir atau waktu ubah terakhir berubah. Perubahan ✓/✗/pakai tidak memicu bangun ulang.
            </p>
          </div>
        </section>

        <!-- 6 -->
        <section id="masuk" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Bagaimana pengetahuan baru masuk
          </h2>
          <div class="overflow-x-auto rounded-xl border border-neutral-200 dark:border-neutral-800">
            <table class="w-full text-sm [&_th]:text-left [&_th]:text-xs [&_th]:uppercase [&_th]:font-semibold [&_th]:text-neutral-500 [&_th]:bg-neutral-50 dark:[&_th]:bg-neutral-900 [&_th]:px-3 [&_th]:py-2 [&_td]:px-3 [&_td]:py-2 [&_td]:align-top [&_td]:border-t [&_td]:border-neutral-100 dark:[&_td]:border-neutral-800">
              <thead>
                <tr><th>Sumber</th><th>Lewat</th><th>Menjadi</th><th>Kapan</th></tr>
              </thead>
              <tbody>
                <tr v-for="s in SOURCES" :key="s[0]">
                  <td class="font-medium text-neutral-900 dark:text-white">
                    {{ s[0] }}
                  </td><td>
                    <UBadge size="xs" :color="s[1] === 'Sinkron' ? 'info' : 'success'" variant="soft">
                      {{ s[1] }}
                    </UBadge>
                  </td><td>{{ s[2] }}</td><td class="text-neutral-500">
                    {{ s[3] }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
              <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
                <UIcon name="i-lucide-refresh-ccw-dot" class="text-sky-500" /> Sinkron
              </p>
              <p>Berkas/tabel sumber tetap jadi sumber kebenaran: butir baru dibuat, teks yang berubah diperbarui, yang hilang dipensiunkan, yang kembali dihidupkan. <b>Bukti ✓/✗ tidak pernah ditimpa.</b> Pengetahuan dewan eksekutif & catatan leader tentang timnya tidak disinkron (isinya fakta proyek).</p>
            </div>
            <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
              <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
                <UIcon name="i-lucide-graduation-cap" class="text-emerald-500" /> Learn
              </p>
              <p>Dicari butir dengan kemiripan ≥ 0,8 di lingkup yang sama. Ada → butir lama <b>diperkuat</b> (✓+1, bobot +3, isinya dilengkapi). Tidak ada → <b>butir baru</b>, langsung ditautkan ke ≤ 3 tetangga dan menutup celah otomatis domainnya.</p>
            </div>
          </div>
        </section>

        <!-- 7 -->
        <section id="dipakai" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Bagaimana pengetahuan digunakan
          </h2>
          <div class="overflow-x-auto rounded-xl border border-neutral-200 dark:border-neutral-800">
            <table class="w-full text-sm [&_th]:text-left [&_th]:text-xs [&_th]:uppercase [&_th]:font-semibold [&_th]:text-neutral-500 [&_th]:bg-neutral-50 dark:[&_th]:bg-neutral-900 [&_th]:px-3 [&_th]:py-2 [&_td]:px-3 [&_td]:py-2 [&_td]:align-top [&_td]:border-t [&_td]:border-neutral-100 dark:[&_td]:border-neutral-800">
              <thead>
                <tr><th>Pemakai</th><th>Teks kueri</th><th>Butir</th><th>Konteks pemakaian</th></tr>
              </thead>
              <tbody>
                <tr v-for="u in USERS" :key="u[0]">
                  <td class="font-medium text-neutral-900 dark:text-white">
                    {{ u[0] }}
                  </td><td>{{ u[1] }}</td><td class="font-mono">
                    {{ u[2] }}
                  </td><td class="font-mono text-xs text-neutral-500">
                    {{ u[3] }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <p>Di prompt, butir ditulis seperti ini:</p>
          <pre class="overflow-x-auto rounded-xl bg-neutral-50 dark:bg-neutral-950 border border-neutral-200 dark:border-neutral-800 p-3 text-xs leading-relaxed font-mono text-neutral-700 dark:text-neutral-300">Pengetahuan yang dikuasai & patokan kantor dari Pusat Pengetahuan (ikuti bila relevan dengan todo ini):
- [K38 · patokan · security] SEC-007 Rate Limiting & Proteksi Anti Brute-Force pada Endpoint Publik: … (✓9 ✗3)
- [K1654 · patokan · go] Cek Go (backend): query N+1, transaksi tanpa rollback, rows tidak di-Close … (✓9 ✗1)
- Celah pengetahuan: kantor belum punya pengetahuan tentang gleam. Baca dokumentasi resmi …
- Butir berlabel K&lt;id&gt; berasal dari Pusat Pengetahuan. Bila ada yang keliru … laporkan lewat MCP vibeasik_knowledge_feedback.</pre>
          <p>
            Untuk todo, bagian ini menyatu dengan persona pegawai, pedoman desain & paging, konteks dewan proyek, peringatan kinerja, dan instruksi penilaian QA (wajib menulis <b>PELAJARAN:</b>). Bila konteks diisi, tiap butir dicatat <b>menunggu nilai</b> dan hitungan pakainya naik. Bila pusat belum punya apa pun untuk pegawai itu, prompt kembali ke daftar keahlian pegawai saja.
          </p>
        </section>

        <section class="-mt-6 space-y-3">
          <div class="p-3.5 rounded-xl border border-indigo-200 dark:border-indigo-900 bg-indigo-50/40 dark:bg-indigo-950/20">
            <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
              <UIcon name="i-lucide-split" class="text-indigo-500" /> Todo lintas aplikasi dipecah per pegawai
            </p>
            <p>
              Bila todo menyentuh beberapa aplikasi berbeda di proyek (backend, frontend, mobile), todo dipecah menjadi <b>sub todo per aplikasi</b> — seperti tiket per engineer. Tiap sub todo ditugaskan ke pegawai dengan keahlian aplikasinya (backend engineer, frontend engineer, mobile engineer), sehingga temu-kembali dan prompt final berjalan <b>per pegawai per aplikasi</b>, dan tiap pegawai dinilai QA atas bagiannya sendiri. Induknya berstatus "dipecah per aplikasi" dan selesai otomatis (nilai rata-rata) saat semua bagian selesai.
            </p>
            <p class="mt-1.5 text-xs text-neutral-500">
              Aplikasi sasaran: nama/folder komponen yang disebut todo (satu sub todo per komponen), atau — untuk todo "seluruh aplikasi" — kelas yang punya sinyal kata kunci (endpoint/query → backend, halaman/tampilan → frontend, flutter/mobile → mobile). Todo satu aplikasi tidak pernah dipecah. Coba di simulasi Jaring Pengetahuan dengan memilih proyek.
            </p>
          </div>
        </section>

        <!-- 8 -->
        <section id="belajar" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Mekanisme belajar
          </h2>
          <p>Belajar = memperbarui bukti ✓/✗ (dan dari situ utilitas, skor, ukuran node) setiap kali hasil kerja diketahui.</p>
          <div class="overflow-x-auto rounded-xl border border-neutral-200 dark:border-neutral-800">
            <table class="w-full text-sm [&_th]:text-left [&_th]:text-xs [&_th]:uppercase [&_th]:font-semibold [&_th]:text-neutral-500 [&_th]:bg-neutral-50 dark:[&_th]:bg-neutral-900 [&_th]:px-3 [&_th]:py-2 [&_td]:px-3 [&_td]:py-2 [&_td]:align-top [&_td]:border-t [&_td]:border-neutral-100 dark:[&_td]:border-neutral-800">
              <thead>
                <tr><th>Kejadian</th><th>Efek</th></tr>
              </thead>
              <tbody>
                <tr v-for="l in LEARN" :key="l[0]">
                  <td class="font-medium text-neutral-900 dark:text-white whitespace-nowrap">
                    {{ l[0] }}
                  </td><td>{{ l[1] }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
            <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
              <UIcon name="i-lucide-clipboard-check" class="text-emerald-500" /> Penilai QA memakai rubrik, bukan angka bebas
            </p>
            <p>Rubrik 5 dimensi (kesesuaian 25 · kebenaran & keamanan 25 · kualitas kode 20 · bukti verifikasi 15 · kepatuhan patokan 15), tiap dimensi level 0–4; <b>nilai akhir dihitung kode</b> (Σ bobot × level/4). Tanpa bukti verifikasi nilai dibatasi 50 (review) / 60 (gate). Patokan relevan ikut di prompt penilai dan dinilai per butir; penilai juga menulis "Seharusnya: …" dan maksimal 3 pelajaran umum ke pustaka bersama.</p>
          </div>
          <p class="text-xs text-neutral-500">
            Butir dari Galeri Desain berkedudukan di atas pengetahuan umum LLM & pegawai: hasilnya tetap tercatat, tetapi ✓/✗-nya tidak diubah oleh hasil kerja pegawai.
          </p>
        </section>

        <!-- 9 -->
        <section id="kurator" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Kurator berkala
          </h2>
          <p>Berjalan 20 detik setelah engine menyala, lalu tiap 15 menit (atau tombol <b>Kurasi sekarang</b>), berurutan:</p>
          <ol class="space-y-1.5">
            <li v-for="(c, i) in CURATOR" :key="c[0]" class="flex gap-3">
              <span class="size-6 rounded-full bg-neutral-100 dark:bg-neutral-800 text-xs font-semibold flex items-center justify-center shrink-0">{{ i + 1 }}</span>
              <p><b class="text-neutral-900 dark:text-white">{{ c[0] }}</b> — {{ c[1] }}</p>
            </li>
          </ol>
        </section>

        <!-- 10 -->
        <section id="celah" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Belajar mandiri: celah → butir baru
          </h2>
          <p>Contoh: todo menyentuh <code>src/core/parser.gleam</code>, padahal kantor belum punya satu butir pun tentang Gleam.</p>
          <div class="grid grid-cols-1 md:grid-cols-4 gap-2">
            <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
              <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
                1 · Terdeteksi
              </p>
              <p>Temu-kembali (konteks todo) menemukan ekosistem tanpa butir → celah otomatis "Teknologi baru: gleam"; prompt agent diberi peringatan membaca dokumentasi resmi & menulis PELAJARAN.</p>
            </div>
            <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
              <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
                2 · Ditugaskan
              </p>
              <p>Tiap menit, maks 2 celah terbuka: pegawai yang stack-nya cocok (atau pilihan leader) mendapat agenda belajar (model "deep", maks 6 butir). Status: sedang dipelajari.</p>
            </div>
            <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
              <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
                3 · Dipelajari
              </p>
              <p>Hasil agenda disimpan ke keahlian pegawai <b>dan</b> pustaka bersama (sumber "belajar teknologi baru", kredit nama pegawai). Status: sudah dipelajari.</p>
            </div>
            <div class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
              <p class="flex items-center gap-1.5 font-semibold text-neutral-900 dark:text-white mb-1">
                4 · Dipakai
              </p>
              <p>Todo Gleam berikutnya — oleh pegawai mana pun — langsung menerima butir Gleam, tanpa peringatan celah.</p>
            </div>
          </div>
          <p class="text-xs text-neutral-500">
            Celah juga bisa dilaporkan (metode baru di ekosistem yang sudah dikenal) lewat tab Celah & Belajar atau MCP vibeasik_knowledge_gap. Gagal 3× → diabaikan; celah yang diabaikan manusia tidak dibuka otomatis lagi.
          </p>
        </section>

        <!-- 11 -->
        <section id="angka" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Angka pengaturan
          </h2>
          <div class="grid grid-cols-2 md:grid-cols-4 gap-2">
            <div v-for="c in CONSTS" :key="c[0]" class="p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900">
              <p class="text-xs text-neutral-500">
                {{ c[0] }}
              </p>
              <p class="text-lg font-bold font-mono text-neutral-900 dark:text-white">
                {{ c[1] }}
              </p>
            </div>
          </div>
        </section>

        <!-- 12 -->
        <section id="batas" class="scroll-mt-4 space-y-3">
          <h2 class="text-lg font-bold tracking-tight text-neutral-900 dark:text-white">
            Batasan saat ini
          </h2>
          <ul class="list-disc pl-5 space-y-1">
            <li>Vektor berbasis kata (TF-IDF), bukan embedding semantik: sinonim di luar jembatan ID ↔ EN dan parafrase jauh bisa terlewat.</li>
            <li>Nilai QA tanpa penilaian per butir dibagi rata ke semua butir yang dipakai todo itu; penilaian per butir (rubrik penilai, laporan K&lt;id&gt;) lebih tepat sasaran.</li>
            <li>Agenda belajar celah memakai pengetahuan model LLM; untuk teknologi yang lebih baru dari model, hasilnya sebaiknya dicek atau dilengkapi lewat Ajarkan.</li>
            <li>Tautan hanya 3 tetangga per butir, jadi jumlah tautan antar butir tidak terlalu berbeda; nomor klaster bisa berubah tiap kurasi.</li>
          </ul>
          <p class="text-xs text-neutral-500">
            Rincian teknis: <code>local-engine/internal/knowledge/README.md</code>.
          </p>
        </section>
      </article>
    </div>
  </div>
</template>
