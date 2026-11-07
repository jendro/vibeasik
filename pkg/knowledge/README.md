# Pusat Pengetahuan vibeasik — Cara Kerja

Pusat Pengetahuan adalah **satu pintu** untuk semua patokan dan pengetahuan **umum** pekerja AI vibeasik
(pegawai, reviewer audit, peserta rapat PM). Pengetahuan khusus satu proyek (fitur yang sudah ada,
keputusan teknis proyek, roadmap) **tidak** disimpan di sini — itu milik dewan eksekutif proyek
(`ExecutiveKnowledge`).

Kode: `local-engine/internal/knowledge/` · Halaman: studio `/knowledge`, `/knowledge-graph` (jaring & simulasi), `/knowledge-docs` · Aturan: `AI_CONTEXT.md` Aturan 33.

---

## 1. Gambaran besar

```
            SUMBER (input)                          PUSAT PENGETAHUAN                         PEMAKAI (output)
 ┌──────────────────────────────┐        ┌──────────────────────────────────┐      ┌─────────────────────────────┐
 │ guidelines/*.json            │──sync─▶│                                  │      │ Prompt pegawai (todo)       │
 │ checklist audit per bahasa   │──sync─▶│   node  ──tautan──  node         │─────▶│ Review audit LLM per berkas │
 │ pengetahuan pegawai          │──sync─▶│     │                 │          │Recall│ Rapat PM + tiap spesialis   │
 │ vonis audit (alasan)         │─Learn─▶│   klaster konsep   bukti ✓/✗     │      │ MCP / studio (uji & cari)   │
 │ agenda belajar celah         │─Learn─▶│                                  │      └──────────────┬──────────────┘
 │ manusia (studio) / agent MCP │─Learn─▶│   celah pengetahuan (gap)        │                     │ hasil kerja
 └──────────────────────────────┘        └──────────────▲───────────────────┘                     │
                                                        └──── Reward / Feedback (nilai QA, vonis) ◀┘
```

Siklusnya: **pakai → dinilai → diperkuat/dilemahkan → dirapikan kurator → dipakai lagi**, dan hal yang
belum dikenal → **dipelajari pegawai → jadi node baru**.

---

## 2. Isi satu node

Tabel `knowledge_nodes` (model `models.KnowledgeNode`):

| Kolom | Arti |
|---|---|
| `kind` | Jenis memori (mengikuti CoALA): `rule` patokan wajib/dilarang · `procedure` cara mengerjakan · `lesson` pelajaran dari pengalaman · `concept` pengetahuan dasar (bahasa, framework, metode) |
| `domain` | Ekosistem (`go`, `php`, `python`, `node`, `web`, `flutter`, `dart`, `android`, `jvm`, `ios`, `rust`, …, atau bahasa baru seperti `gleam`) **atau** area (`security`, `testing`, `performance`, `devops`, `database`, `planning`, `architecture`, `backend`, `mobile`, `ai`, `observability`) |
| `title`, `content`, `tags` | Isi pengetahuan |
| `source` | Asal: `guideline`, `audit_checklist`, `audit_verdict`, `employee`, `promoted`, `gap_research`, `qa_review`, `design_library`, `design_theory`, `design_lesson`, `manual`, `agent` |
| `source_key` | Identitas di sumbernya, mis. `guideline:SEC-001`, `audit:go:2`, `employee:15`, `promoted:40`, `design:ref:7`, `design:profile:purposes:ecommerce` (Galeri Desain + facet, Aturan 36); kosong = node asli pusat |
| `owner_employee_id` | Terisi = keahlian **pribadi** pegawai itu; kosong = **pustaka bersama** kantor |
| `weight` | Kepentingan awal 0–100 |
| `helpful` / `harmful` | Bukti ✓/✗ dari hasil kerja |
| `uses`, `last_used_at` | Berapa kali & kapan terakhir disisipkan ke prompt |
| `cluster_id` | Klaster konsep |
| `needs_review` | Patokan berkas yang berulang kali keliru — berkasnya perlu diperbaiki |
| `status` | `active`, `merged` (dilebur ke node lain), `retired` (dipensiunkan) |

Tabel pendukung: `knowledge_links` (tautan antar node), `knowledge_clusters`, `knowledge_uses`
(node apa dipakai di pekerjaan apa), `knowledge_gaps` (celah pengetahuan).

---

## 3. Algoritma yang dipakai

| # | Algoritma | Untuk apa | Dasar riset | Berkas |
|---|---|---|---|---|
| 1 | **TF-IDF jarang + kemiripan kosinus** | Mengubah teks jadi vektor & mengukur kemiripan | Information retrieval klasik | `vector.go` |
| 2 | **Tokenizer ID↔EN + camelCase** | "validasi" bisa menemukan "validation", `fetchUserData` dipecah | — | `vector.go` |
| 3 | **Posting list (indeks terbalik)** | Mencari tetangga terdekat tanpa membandingkan semua pasangan | IR klasik | `vector.go` `nearest` |
| 4 | **Skor gabungan relevansi + utilitas + kepentingan + kebaruan** | Memilih node paling berguna | Generative Agents (Park dkk., 2023) | `recall.go` |
| 5 | **Utilitas Beta (Bayes)** `(✓+1)/(✓+✗+2)` | Seberapa terbukti node berguna; node baru = 0,5 | ACE (ICLR 2026), ExpeL (2024) | `models.KnowledgeNode.Utility` |
| 6 | **Peluruhan eksponensial** `e^(−hari/45)` | Node yang lama tidak dipakai turun pelan | MemoryBank / kurva lupa | `recall.go` |
| 7 | **Aktivasi menyebar 1 langkah** | Tetangga bertaut dari node teratas ikut dipertimbangkan | HippoRAG, A-MEM | `recall.go` `expand` |
| 8 | **Tautan Zettelkasten (k-tetangga terdekat, k=3)** | Menghubungkan node yang isinya berdekatan | A-MEM (NeurIPS 2025) | `knowledge.go`, `curate.go` `relink` |
| 9 | **Leader clustering satu lintasan** (kosinus ≥ 0,3) | Mengelompokkan node per konsep + label kata pusat | RAPTOR / GraphRAG versi ringan | `curate.go` `recluster` |
| 10 | **Dedup & peleburan** (kosinus ≥ 0,8 saat tulis, ≥ 0,85 saat kurasi) | Memori berkembang, bukan berlipat | ACE (pembaruan delta), ExpeL (upvote) | `knowledge.go`, `curate.go` |
| 11 | **Kurikulum otomatis dari celah** | Hal baru → agenda belajar → node baru | Voyager (2023) | `gaps.go`, `staff/knowledge_gaps.go` |
| 12 | **Penilaian kredit dari hasil** | Nilai QA / vonis audit membagi ✓/✗ ke node yang dipakai | ACE (generator → reflector → curator) | `knowledge.go` `Reward`, `verdict.go` |

Semuanya murni Go (tanpa CGO, tanpa layanan embedding luar) sesuai Aturan 2.

### 3.1 Vektor TF-IDF

Untuk tiap node:

```
frekuensi kata: judul ×2, domain+tag ×2, isi ×1
tf  = 1 + ln(frekuensi)
idf = ln((N + 1) / (df + 0,5)) + 1        N = jumlah node aktif, df = jumlah node yang memuat kata itu
vektor = tf × idf, lalu dinormalisasi (panjang = 1)
kemiripan(a, b) = a · b   (kosinus)
```

- Vektor **kueri** membuang kata yang tidak ada di korpus (pasti tidak cocok).
- Vektor **node baru** tetap menghitung kata barunya — kata baru itulah pembeda, jadi dua catatan yang hanya
  beda topik tidak dianggap duplikat.
- Indeks disimpan di memori dan dibangun ulang hanya bila `COUNT(*)` atau `MAX(updated_at)` tabel berubah
  (ratusan sampai ribuan node ≈ milidetik). Perubahan penghitung ✓/✗/uses memakai `UpdateColumn` sehingga
  tidak memicu bangun ulang.

### 3.2 Domain & penyaringan ekosistem

`DetectDomains(teks)` mengenali domain dari:
1. **Ekstensi berkas** di path (`app/pages/x.vue` → `web`, `internal/api/x.go` → `go`, `lib/x.dart` → `flutter`).
   Ekstensi tak dikenal yang berbentuk path (`src/core/x.gleam`) → ekosistem baru bernama ekstensinya.
2. **Kata kunci** (`laravel` → `php`, `flutter` → `flutter`, `xss/injection` → `security`, …).
3. Untuk rapat PM: **bahasa & framework komponen** proyek (`StackDomain`).

Ekosistem bersifat **eksklusif** (Aturan 16): bila pekerjaan adalah Go, node ekosistem Flutter/PHP/web tidak
ikut. Area (security, testing, …) selalu boleh ikut.

### 3.3 Rumus skor temu-kembali (`Recall`)

```
relevansi_mentah = kosinus(kueri, node) + 0,12 (bila domain node = domain pekerjaan)
relevansi        = relevansi_mentah / relevansi_mentah_tertinggi       (dinormalisasi 0–1)

skor = 0,55 × relevansi
     + 0,20 × utilitas          (✓+1)/(✓+✗+2)
     + 0,15 × bobot/100
     + 0,10 × kebaruan          e^(−hari sejak terakhir dipakai / 45)
```

Langkahnya:
1. Ambil kandidat dengan relevansi mentah ≥ 0,08 (maks 80).
2. Saring: node pribadi pegawai lain, ekosistem lain, jenis/sumber yang dikecualikan, node merugikan
   (✗ ≥ 3 dan utilitas < 0,25).
3. **Perluas lewat tautan**: tetangga node teratas mendapat skor `0,7 × skor_asal × kekuatan_tautan + 0,3 × skor_dirinya`
   dan ditandai "lewat tautan".
4. **Keberagaman**: satu klaster maksimal separuh hasil.
5. Ambil `Limit` teratas.

---

## 4. Input — dari mana pengetahuan masuk

| Sumber | Masuk lewat | Jenis | Kapan |
|---|---|---|---|
| `guidelines/*.json` | Sinkron (`guidelineNotes`) | `rule`, pustaka bersama, bobot dari severity (critical 95 · high 85 · medium 70 · low 55) | Saat kurasi (20 dtk setelah engine start, lalu tiap 15 menit) |
| Checklist review audit per bahasa (`ai/audit_review.go`) | Sinkron (`ai.AuditChecklistNotes`) | `rule`, pustaka bersama | Saat kurasi |
| Pengetahuan pegawai (`employee_knowledge`) | Sinkron (`employeeNotes`) | `procedure`/`lesson`, **pribadi** pegawai | Saat kurasi + tepat sebelum pegawai mengerjakan todo (`SyncEmployee`) |
| Vonis audit (alasan "audit keliru"/"temuan benar") | `LearnFromVerdict` → `Learn` | `lesson` "Bukan masalah: …" / "Masalah nyata: …" | Saat vonis disimpan (studio, cek LLM, MCP `record_audit_feedback`) |
| Hasil agenda belajar celah | `publishGapKnowledge` → `Learn` | `concept`/`procedure`/`lesson`, pustaka bersama, kredit nama pegawai | Saat agenda celah selesai |
| Manusia | Studio `/knowledge` → tombol **Ajarkan** (`POST /api/knowledge/nodes`) | Pilihan sendiri | Kapan saja |
| Agent coding | MCP `vibeasik_knowledge_learn` | Pilihan agent | Kapan saja |

Catatan:
- **Sinkron** = sumber tetap menjadi sumber kebenaran. Butir baru dibuat, butir yang teksnya berubah diperbarui,
  butir yang hilang dari sumber dipensiunkan, butir yang kembali dihidupkan lagi — **bukti ✓/✗ tidak pernah
  ditimpa**. Bila folder guidelines tidak terbaca, tidak ada yang dipensiunkan.
- Pengetahuan pegawai **dewan eksekutif** (CEO/CTO/PO/CMO) tidak disinkron karena isinya fakta proyek.
- Catatan leader tentang anggota timnya (`source = team`) tidak disinkron.

---

## 5. Kapan pengetahuan digunakan

| Pemakai | Titik di kode | Kueri | Jumlah butir | Konteks pemakaian |
|---|---|---|---|---|
| **Pegawai mengerjakan todo** | `staff.promptSection` → `hubKnowledge` | judul + keterangan + aplikasi target + kategori todo; keahlian pribadi pegawai itu + pustaka bersama | 15 | `todo:<id>` |
| **Review audit LLM per berkas** | `ai.reviewFileWithLLM` → `auditKnowledge` | path berkas + bahasa + judul temuan statis; domain = ekosistem grup review; checklist dilewati (sudah di prompt sistem) | 8 | `audit:<proyek>:<path>` |
| **Rapat PM — konteks bersama** | `planner.buildContext` | permintaan rapat + deskripsi proyek; domain = stack komponen | 10 | `plan:<proyek>` |
| **Rapat PM — tiap spesialis** | `planner.specialistPrompt` | nama + fokus peran (mis. Spesialis Keamanan) + deskripsi proyek | 6 | — |
| **Agent luar (MCP)** | `vibeasik_knowledge_recall` | bebas | default 10 | — |
| **Manusia (studio)** | tab **Uji Temu-kembali** / pencarian | bebas | 12 | — (pratinjau, tidak dicatat) |

Di prompt, butir ditulis seperti:

```
- [K233 · patokan · go] Cek Go (backend): goroutine leak, … (✓3 ✗0)
- Celah pengetahuan: kantor belum punya pengetahuan tentang gleam. Baca dokumentasi resmi …
- Butir berlabel K<id> berasal dari Pusat Pengetahuan. Bila ada yang keliru … laporkan lewat MCP vibeasik_knowledge_feedback.
```

Todo yang menyentuh beberapa aplikasi berbeda (backend, frontend, mobile) dipecah dulu per aplikasi
(`staff/split.go`, `AI_CONTEXT.md` Aturan 50): tiap sub todo punya pegawai berkeahlian aplikasinya sendiri, sehingga
temu-kembali di atas berjalan **per pegawai per aplikasi** — keahlian pribadi & domain yang dipakai sesuai bagiannya.

Bila konteks diisi (`todo:…`, `audit:…`, `plan:…`), node yang terpilih **dicatat** di `knowledge_uses`
(konteks yang sama tidak dicatat dua kali selama belum dinilai) dan `uses`/`last_used_at` node naik.
Bila Pusat Pengetahuan kosong untuk pegawai itu, prompt kembali ke cara lama (`relevantKnowledge`).

---

## 6. Kapan belajar (memperbarui bukti)

| Kejadian | Fungsi | Efek |
|---|---|---|
| **Review QA menilai todo** (0–100) | `rewardTodoKnowledge` → `Reward("todo:<id>", nilai)` | Semua node yang dipakai di prompt todo itu: nilai ≥ 75 → ✓+1, < 50 → ✗+1, 50–74 netral |
| **Vonis audit "temuan benar"** | `LearnFromVerdict` | Node yang dipakai review LLM berkas itu dinilai 85 (✓); patokan dasarnya ✓+1; alasan → pelajaran "Masalah nyata: …" |
| **Vonis audit "audit keliru"** | `LearnFromVerdict` | Node yang dipakai dinilai 35 (✗); patokan dasarnya ✗+1; alasan + saran rule → pelajaran "Bukan masalah: …" |
| **Umpan balik langsung** | `Feedback` (studio 👍/👎, MCP `vibeasik_knowledge_feedback`) | ✓+1 atau ✗+1 |
| **Butir yang sama ditulis lagi** | `Learn` (kemiripan ≥ 0,8) | Node lama diperkuat: ✓+1, bobot +3, isinya dilengkapi |

"Patokan dasar" sebuah temuan = node `guideline:<RuleID>` bila ada; untuk temuan review LLM = checklist audit
ekosistem itu yang isinya paling mirip judul & deskripsi temuan (kemiripan ≥ 0,2).

### Kurator (tiap 15 menit, atau tombol **Kurasi sekarang**)

Urutan dalam `Curate`:
1. **Sinkron sumber** (guidelines, checklist audit, pengetahuan pegawai).
2. **Lebur duplikat**: node asli dengan kemiripan ≥ 0,85 di lingkup pemilik yang sama → yang lebih kuat
   (`bobot + 5×(✓−✗) + uses`) dipertahankan, bukti dijumlahkan, yang lain berstatus `merged`.
3. **Promosi**: keahlian pribadi pegawai dengan ✓ ≥ 3 dan utilitas ≥ 0,75 disalin ke pustaka bersama
   (sumber `promoted`, kredit nama pegawai) — pegawai lain bisa langsung memakainya.
4. **Pangkas**: node asli dengan ✗ ≥ 3 dan utilitas < 0,3 → `retired`. Patokan berkas (guideline/checklist)
   dengan ✗ ≥ 3 dan utilitas < 0,4 hanya ditandai **perlu ditinjau** (berkasnya yang harus diperbaiki); tanda
   dilepas lagi bila utilitasnya pulih ≥ 0,5.
5. **Hitung ulang tautan**: tiap node ke 3 tetangga terdekat (kemiripan ≥ 0,25).
6. **Hitung ulang klaster**: leader clustering (node terkuat jadi pusat lebih dulu, gabung bila kemiripan ke
   pusat klaster ≥ 0,3), label = 4 kata terberat, klaster minimal 2 node.
7. Hapus catatan pemakaian lebih dari 30 hari.

### Penilai QA memakai rubrik + patokan kantor

Review QA batch (`staff/review.go`) dan QA gate sebelum merge (`staff/gate.go`) memakai model ≥ Claude Opus 5.5
(Aturan 26) **dengan patokan**, bukan sekadar "pasrah ke LLM pintar":

1. **Rubrik 5 dimensi** (`staff/rubric.go`), tiap dimensi level 0–4 dengan deskripsi tetap:
   kesesuaian & kelengkapan 25 · kebenaran & keamanan 25 · kualitas & kesederhanaan kode 20 ·
   bukti verifikasi 15 · kepatuhan patokan kantor 15. **Nilai akhir dihitung kode** dari bobot
   (`Σ bobot × level/4`), bukan angka bebas penilai; tanpa bukti verifikasi nilai dibatasi 50 (review) / 60 (gate).
2. **Patokan kantor** relevan (≤ 8–10 butir K<id> dari Pusat Pengetahuan) ikut di prompt penilai. Penilai
   memberi status per butir: `followed` → ✓, `wrong` (patokannya keliru untuk kasus ini) → ✗,
   `violated` (hasil kerja melanggar — patokannya tetap benar) dan `n/a` → tanpa perubahan (`knowledge.Judge`).
   Node yang sudah dinilai eksplisit tidak dinilai dua kali oleh `Reward`.
3. **Best practice wajib**: penilai menulis "Seharusnya: …" (cara ideal) — masuk komentar QA dan catatan
   rework agent berikutnya.
4. **Pelajaran umum** dari penilai (maks 3 per review, 2 per putaran gate) masuk **pustaka bersama**
   (sumber `qa_review`) lewat `Learn` — pengetahuan dari LLM luar ikut memperkaya kantor.

---

## 7. Kapan node baru terbentuk

1. **Sinkron** menemukan butir sumber yang belum punya node (guideline baru, checklist baru, pengetahuan
   pegawai baru dari review QA/agenda/obrolan/warisan).
2. **`Learn`** dipanggil (vonis audit beralasan, hasil agenda celah, Ajarkan di studio, MCP) **dan** tidak ada
   node dengan kemiripan ≥ 0,8 di lingkup yang sama. Node baru langsung ditautkan ke ≤ 3 tetangga
   (kemiripan ≥ 0,25) dan menutup celah otomatis domainnya.
3. **Promosi** kurator membuat salinan bersama dari keahlian pribadi yang terbukti.
4. **Penilai QA** menulis best practice umum (`lessons`) saat review/QA gate.

Node **tidak** terbentuk bila isinya hampir sama dengan node yang sudah ada — node lama yang diperkuat.

---

## 8. Belajar mandiri: celah pengetahuan → node baru

Contoh: todo menyentuh `src/core/parser.gleam`, padahal kantor belum punya satu node pun tentang Gleam.

```
1. Pegawai mulai todo → Recall (konteks todo:…) mendeteksi ekosistem "gleam" tanpa node
   (hanya bila pusat sudah berisi ≥ 20 node)
      → celah otomatis dicatat: "Teknologi baru: gleam" (hits +1 tiap muncul lagi)
      → prompt agent diberi peringatan: baca dokumentasi resmi, tulis temuan di PELAJARAN
2. Tiap menit (staff.autoReviewTick → learnGaps), maks 2 celah terbuka:
      → pilih pegawai yang stack/spesialisasinya menyebut domain itu, atau pilihan leader divisi
      → buat agenda pribadi "learn" (pemicu knowledge_gap, kebutuhan model "deep", maks 6 butir)
      → status celah: learning
3. Agenda dikerjakan saat pegawai senggang (LLM pool)
      → hasilnya disimpan ke pengetahuan pribadi pegawai DAN dibagikan ke pustaka bersama
        (sumber gap_research, domain gleam, kredit nama pegawai)
      → status celah: learned
4. Todo Gleam berikutnya (oleh pegawai mana pun) langsung menerima node Gleam, tanpa peringatan celah.
```

Celah juga bisa **dilaporkan** untuk metode baru di ekosistem yang sudah dikenal (mis. "Go 1.26
range-over-func iterator") lewat studio tab **Celah & Belajar** atau MCP `vibeasik_knowledge_gap`.
Agenda yang gagal/dibatalkan/tidak menghasilkan node → celah dibuka lagi; setelah 3 percobaan diabaikan.
Celah yang diabaikan manusia tidak dibuka otomatis lagi.

---

## 9. Pengaturan angka (konstanta)

| Konstanta | Nilai | Berkas |
|---|---|---|
| Dedup saat tulis (`dupMin`) | 0,8 | `knowledge.go` |
| Tautan minimal (`linkMin`) / tautan per node | 0,25 / 3 | `knowledge.go`, `curate.go` |
| Lebur saat kurasi (`mergeMin`) | 0,85 | `curate.go` |
| Klaster (`clusterMin`) | 0,3 | `curate.go` |
| Relevansi minimal (`minRelevance`) | 0,08 | `recall.go` |
| Bonus domain (`domainBonus`) | 0,12 | `recall.go` |
| Peluruhan kebaruan (`recencyDays`) | 45 hari | `recall.go` |
| Minimal node sebelum celah dicatat (`minGapCorpus`) | 20 | `recall.go` |
| Interval kurator | 15 menit (pertama 20 dtk setelah start) | `curate.go` |
| Simpan catatan pemakaian | 30 hari | `curate.go` |
| Percobaan belajar celah | 3 | `gaps.go` |

---

## 10. API & MCP

**REST** (engine `:8765`):

| Metode | Path | Fungsi |
|---|---|---|
| GET | `/api/knowledge` | Ringkasan: jumlah, domain, klaster, celah, patokan perlu ditinjau |
| GET | `/api/knowledge/nodes?q=&domain=&kind=&source=&cluster_id=` | Daftar node (`q` = urut skor Recall) |
| POST | `/api/knowledge/nodes` | Ajarkan pengetahuan ke pustaka bersama |
| GET | `/api/knowledge/nodes/:id` | Detail: tetangga, riwayat pemakaian, peleburan |
| POST | `/api/knowledge/nodes/:id/feedback` | `{helpful: true/false}` |
| DELETE | `/api/knowledge/nodes/:id` | Pensiunkan node |
| POST | `/api/knowledge/recall` | Pratinjau butir untuk sebuah teks pekerjaan (tidak dicatat) |
| POST | `/api/knowledge/preview` | Pratinjau prompt pegawai sebuah pekerjaan: pegawai pilihan leader, prompt final, butir, celah (tidak dicatat) |
| GET | `/api/knowledge/graph` | Node aktif ringkas + tautan + klaster untuk peta Jaring Pengetahuan |
| POST | `/api/knowledge/curate` | Jalankan kurator sekarang |
| GET/POST | `/api/knowledge/gaps` | Daftar / laporkan celah |
| POST | `/api/knowledge/gaps/:gid/dismiss` | Abaikan celah |

**MCP**: `vibeasik_knowledge_recall`, `vibeasik_knowledge_learn`, `vibeasik_knowledge_feedback`,
`vibeasik_knowledge_gap`.

---

## 11. Batasan saat ini

- Vektor berbasis kata (TF-IDF), bukan embedding semantik: sinonim di luar daftar jembatan ID↔EN dan parafrase
  jauh bisa terlewat. Embedding bisa ditambahkan nanti sebagai sinyal tambahan bila ada endpoint embedding.
- Nilai QA tanpa penilaian per butir dibagi rata ke semua node yang dipakai di todo itu; penilaian per butir
  (status `followed`/`wrong` dari rubrik penilai lewat `Judge`, atau laporan K<id> manual/MCP) lebih tepat sasaran.
- Agenda belajar celah memakai pengetahuan model LLM; untuk teknologi yang lebih baru dari model, hasilnya
  sebaiknya dicek atau dilengkapi lewat Ajarkan.
- Klaster dihitung ulang tiap kurasi, jadi nomor klaster bisa berubah.
