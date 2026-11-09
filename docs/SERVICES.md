# Paket Layanan Profesional & Pilot Service vibeasik

Dokumen ini mendefinisikan tiga paket layanan profesional (professional services) berbasis platform vibeasik, jaminan privasi repositori privat, instrumen validasi pasar, checklist operasional, alur pembayaran manual, serta hasil verifikasi pilot terhadap pelanggan berbayar.

---

```
+-----------------------------------------------------------------------------------------+
|                  ALUR OPERASIONAL AUDIT REPOSITORI PRIVAT vibeasik                      |
|                                                                                         |
|  [Klien: Registrasi Minat / Order Manual]                                               |
|         │ (Pilih Paket: Express / Remediation / Team Setup)                             |
|         ▼                                                                               |
|  [Pembayaran Manual & NDA] ──► Transfer BCA / Mandiri / QRIS / PO Invoice               |
|         │                                                                               |
|         ▼ (Eksekusi 100% On-Premise / Workstation Lokal Terisolasi)                     |
|  [vibeasik Local Engine] ──► Fast Scan (85+ Rule) & Code Map Topologi                   |
|         │                                                                               |
|         ├──► Deep Audit (Cyclomatic Complexity, Bottleneck Latensi, BOLA, N+1 Query)    |
|         └──► Remediation Sprint (Isolasi Git Worktree, PR Perbaikan, Unit Test)         |
|         ▼                                                                               |
|  [QA Gatekeeper vibeasik] ──► Nilai Kelulusan Skor > 70/100                             |
|         │                                                                               |
|         ▼                                                                               |
|  [Serah Terima Deliverable] ──► Executive Report + PR Siap Merge + Benchmark Sebelum-   |
|                                 Sesudah (ZERO CODE LEAK KE CLOUD PIHAK KETIGA)          |
+-----------------------------------------------------------------------------------------+
```

---

## 1. Jaminan Privasi Repositori Privat (Zero-Leak Security Guarantee)

Bagi perusahaan fintech, logistik, software house, dan SaaS yang mengelola repositori privat berkode hak milik (proprietary code):
1. **100% Local-First Execution**:
   - Engine audit vibeasik berjalan sepenuhnya di komputer lokal klien atau mesin server internal (on-premise / private VPC).
   - Seluruh analisis statis, pemetaan AST, dan pengujian berjalan tanpa mengunggah berkas kode sumber ke cloud publik atau LLM pihak ketiga.
2. **Penyensoran Rahasia Bawaan (SEC-003)**:
   - Modul `GlobalSanitizer` otomatis menyamarkan token API, string koneksi basis data, dan kunci privat sebelum laporan dicetak.
3. **Isolasi Git Worktree**:
   - Perbaikan kode dilakukan di branch kerja terisolasi (`git worktree`) tanpa menyentuh branch produksi secara langsung sebelum diverifikasi oleh tim klien.

---

## 2. Tiga Paket Layanan Awal & Tingkat Harga Uji

Layanan profesional adalah kanal monetisasi langsung yang memberikan nilai riil bagi tim rekayasa perangkat lunak tanpa risiko operasional.

| Parameter | Paket 1: Audit Cepat (Express Codebase Audit) | Paket 2: Audit + Perbaikan Bergaransi (Remediation Sprint) | Paket 3: Setup Tim AI Coding & Orkestrasi Kantor |
| :--- | :--- | :--- | :--- |
| **Sasaran Pengguna** | Startup/tim developer yang butuh evaluasi kilat arsitektur, keamanan, dan performa sebelum rilis. | Tim engineering yang memiliki backlog temuan audit dan ingin penuntasan instan bergaransi QA. | Perusahaan/agensi yang ingin mengadopsi alur AI Pair Programming otonom berbasis MCP & staf AI kantor. |
| **Cakupan Pekerjaan** | 1 repositori privat, registrasi proyek, Fast Scan (85+ rule), audit arsitektur, pemetaan topologi & rute kode. | Semua cakupan Paket 1 + Deep Audit, eksekusi refactoring hingga 15 temuan prioritas, test regression. | Instalasi & konfigurasi vibeasik Pro / Local Agent Server di 5 workstation, integrasi 123 tool MCP, setup pegawai AI kantor, workshop 2 jam. |
| **Output / Deliverable** | Executive Diagnostic Report (PDF/MD), Risk Matrix, Rekomendasi tindakan prioritas. | Laporan Audit + Branch/PR perbaikan tervalidasi + Benchmark performa sebelum-sesudah + Test suite hijau. | Blueprint Arsitektur AI Tim, Konfigurasi MCP siap pakai (Cursor/Antigravity/Claude Code), Knowledge Pack tim, Rekaman workshop. |
| **Batas Lingkup (Scope)** | Analisis statis & topologi, tidak mencakup penulisan kode perbaikan. | 1 repositori utama, maksimal 15 temuan Critical/High, tidak mencakup rewrite arsitektur total. | Hingga 5 developer workstation, 1 server/host vibeasik, pendampingan implementasi 1 minggu. |
| **Estimasi Durasi** | 1 hari kerja (turnaround 24 jam) | 3–5 hari kerja | 5–7 hari kerja (termasuk 1 minggu asistensi) |
| **Tingkat Harga Uji** | **Rp 2.500.000** (~$160 USD) | **Rp 8.500.000** (~$550 USD) | **Rp 17.500.000** (~$1.150 USD) |

*Contoh nyata hasil laporan audit yang telah dianonimkan dapat dipelajari pada [Contoh Laporan Audit Anonim](SAMPLE_AUDIT_REPORT_ANONYMIZED.md).*

---

## 3. Checklist Pekerjaan per Paket

### A. Paket 1: Audit Cepat (Express Codebase Audit)
1. **Pre-flight & Registrasi Proyek**:
   - Registrasi repositori via CLI (`vibeasik project add`) atau Studio UI.
   - Deteksi otomatis stack teknologi, framework, dan subkomponen.
2. **Pindaian Cepat (Fast Scan)**:
   - Eksekusi 85+ aturan diagnostik (Keamanan CWE/OWASP, N+1 Query, Deadlock/Race Condition, Unbounded Viewport, Aksesibilitas).
   - Verifikasi false-positive awal melalui call graph inspection.
3. **Peta Kode & Topologi Arsitektur**:
   - Pembangunan Code Map: pemetaan layer (controller, service, repository, model).
   - Inventaris rute HTTP dan kontrak payload.
4. **Penyusunan & Pengiriman Laporan**:
   - Ekspor skor kesehatan kode (0–100) dan klasifikasi severity.
   - Pengiriman dokumen rekomendasi teknis siap eksekusi.

### B. Paket 2: Audit + Perbaikan Bergaransi (Remediation Sprint)
1. **Deep Check & Audit Mendalam**:
   - Analisis mendalam Cyclomatic Complexity (CC > 15), latency simulator rute HTTP kritis (> 500ms), dan data leak.
   - Prioritisasi 10–15 temuan paling berdampak pada stabilitas dan keamanan produksi.
2. **Eksekusi Refactoring Terisolasi**:
   - Pembagian tugas ke branch perbaikan terisolasi (git worktree).
   - Penerapan pola solusi kanonik: Transaction Script, Batch IN Query, Deferred Rollback (`defer tx.Rollback()`), Sanitasi input DOMPurify, dan Boundary Validation DTO.
3. **Pengujian & Validasi QA Gate**:
   - Penulisan unit test layer service (cakupan target > 60%).
   - Pembuatan integration test rute API dengan SQLite in-memory mandiri.
   - Eksekusi benchmark latensi sebelum-sesudah (`ns/op` dan memory allocs/op).
   - Validasi kelulusan lewat QA Gate vibeasik (skor minimal 70/100).
4. **Serah Terima Kode**:
   - Pull Request bersih dengan Conventional Commits dan dokumentasi diff lengkap.
   - Panduan review dan merge untuk tim klien.

### C. Paket 3: Setup Tim AI Coding & Orkestrasi Kantor
1. **Penyediaan Environment & Tools**:
   - Instalasi engine vibeasik Pro dan registrasi local auth token.
   - Konfigurasi dual-transport MCP (stdio & SSE) untuk editor AI (Antigravity, Cursor, Claude Code, Cline).
2. **Personalisasi Tim Pegawai AI (Staff Office)**:
   - Konfigurasi persona pegawai AI internal (Tech Lead, Senior Backend, Frontend UI, QA Gatekeeper, Security Specialist).
   - Pengaturan batasan model AI per peran (mis. model hemat untuk drafting, model tingkat lanjut untuk QA Gate).
3. **Inisialisasi Pusat Pengetahuan (Knowledge Mesh)**:
   - Kurasi aturan kustom arsitektur proyek ke dalam Pusat Pengetahuan tim.
   - Sinkronisasi patokan kantor untuk memandu AI agent mematuhi standar internal.
4. **Workshop & Pendampingan Hands-on (2 Jam)**:
   - Sesi interaktif teknik prompt, delegation, dan evaluasi hasil kerja AI agent.
   - Simulasi penyelesaian 1 fitur riil menggunakan alur orkestrator vibeasik.

---

## 4. Alur Pendaftaran Minat & Prosedur Pembayaran Manual

Untuk mengakomodasi transaksi B2B tanpa hambatan gerbang pembayaran otomatis, vibeasik menyediakan alur pendaftaran minat dan pembayaran manual yang transparan:

```
[Isi Formulir Minat / Order di Web] ──► [Terbitkan Invoice / SPK Manual]
                                                    │
                                                    ▼
[Konfirmasi via WhatsApp / Email] ◄─── [Transfer Bank / QRIS Manual / PO]
          │
          ▼
[Kickoff Audit & Eksekusi Terjadwal]
```

1. **Pengisian Formulir Pendaftaran**:
   - Klien memilih paket yang diinginkan melalui halaman [Audit & Layanan](https://vibeasik.com/audit).
   - Menyertakan data: Nama kontak, organisasi/perusahaan, segmen bisnis, URL repositori (atau rincian stack teknologi), dan paket yang dipilih.
2. **Penerbitan Invoice & Instruksi Pembayaran**:
   - Tim finance vibeasik menerbitkan Faktur Penjualan resmi (Invoice) atas nama badan hukum penyelenggara: **PT Pancen Apik Tenan**.
   - Opsi pembayaran manual yang didukung:
     - **Transfer Bank BCA**: `Rekening: 037-XXXX-XXX a.n. PT Pancen Apik Tenan`
     - **Transfer Bank Mandiri**: `Rekening: 137-00-XXXX-XXX a.n. PT Pancen Apik Tenan`
     - **QRIS Dinamis/Statis**: Tersedia untuk verifikasi instan.
     - **Purchase Order (PO) / Termin Korporat**: Termin 50% Down Payment dan 50% setelah serah terima deliverable (khusus Paket 2 & 3).
3. **Verifikasi & Kickoff Teknis**:
   - Bukti pembayaran dikonfirmasikan melalui WhatsApp Hotline (`+62 812-XXXX-XXXX`) atau email finance (`jendro@tenan.id`).
   - Tim engineer mengonfirmasi repositori, menandatangani NDA kerahasiaan (bila diperlukan), dan memulai sprint pindaian sesuai SLA durasi.
