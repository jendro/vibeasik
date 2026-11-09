# Paket Layanan Profesional & Pilot Service vibeasik

Dokumen ini mendefinisikan tiga paket layanan profesional (professional services) berbasis platform vibeasik, instrumen validasi pasar, checklist operasional, serta hasil wawancara pilot terhadap calon pelanggan.

---

## 1. Tiga Paket Layanan Awal

Layanan profesional adalah kanal monetisasi tercepat dengan risiko operasional terendah, sekaligus instrumen validasi kebutuhan industri sebelum peluncuran modul berbayar penuh (billing kredit/token). Seluruh paket memaksimalkan kapabilitas platform yang sudah teruji: registrasi proyek, Fast Scan 85+ aturan, Deep Check/Audit, Peta Kode (Code Map), Dual-Transport MCP, dan Kantor Pegawai AI vibeasik Pro.

| Parameter | Paket 1: Audit Cepat (Express Codebase Audit) | Paket 2: Audit + Perbaikan Bergaransi (Remediation Sprint) | Paket 3: Setup Tim AI Coding & Orkestrasi Kantor |
| :--- | :--- | :--- | :--- |
| **Sasaran** | Startup/tim developer yang butuh evaluasi kilat arsitektur, keamanan, dan performa sebelum rilis. | Tim engineering yang memiliki backlog temuan audit dan ingin penuntasan instan bergaransi QA. | Perusahaan yang ingin mengadopsi alur AI Pair Programming otonom berbasis MCP & staf AI kantor. |
| **Cakupan Pekerjaan** | 1 repositori, registrasi proyek, Fast Scan, audit arsitektur, pemetaan topologi & rute kode. | Semua cakupan Paket 1 + Deep Audit, eksekusi perbaikan hingga 15 temuan prioritas, test regression. | Instalasi & konfigurasi vibeasik Pro / Local Agent Server, integrasi 123 tool MCP, setup pegawai AI kantor, workshop 2 jam. |
| **Output / Deliverable** | Executive Diagnostic Report (PDF/MD), Risk Matrix, Rekomendasi tindakan prioritas. | Laporan Audit + Branch/PR perbaikan tervalidasi + Benchmark performa sebelum-sesudah + Test suite hijau. | Blueprint Arsitektur AI Tim, Konfigurasi MCP siap pakai (Cursor/Antigravity/Claude Code), Knowledge Pack awal tim, Rekaman workshop. |
| **Batas Lingkup** | Analisis statis & topologi, tidak mencakup penulisan kode perbaikan. | 1 repositori utama, maksimal 15 temuan Critical/High, tidak mencakup rewrite arsitektur total. | Hingga 5 developer workstation, 1 server/host vibeasik, pendampingan implementasi 1 minggu. |
| **Estimasi Durasi** | 1 hari kerja (turnaround 24 jam) | 3–5 hari kerja | 5–7 hari kerja (termasuk 1 minggu asistensi) |
| **Harga Awal** | **Rp 2.500.000** (~$160 USD) | **Rp 8.500.000** (~$550 USD) | **Rp 17.500.000** (~$1.150 USD) |

---

## 2. Checklist Pekerjaan per Paket

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
