# vibeasik.com - Sistem Patokan & Rubrik AI (AI Guidelines & Standards)

Direktori ini adalah **Pusat Patokan Tunggal (Single Source of Rubrics & Rules)** yang digunakan oleh AI Coding Agent (Antigravity, Cursor, Claude Code, Cline) dan engine Vibeasik untuk:
1. **Memeriksa & Mengaudit Aplikasi** (apakah arsitektur, port, security, dan rutenya sehat).
2. **Memberikan Saran Pengerjaan Aplikasi (AI Todo Suggestions)** dengan landasan patokan yang terukur.
3. **Mengatur Hierarki Perencanaan 3 Tingkat**:
   - **Tingkat 1: Perencanaan Global (Platform & Architecture Todo)**: Roadmap makro lintas-proyek & standar platform.
   - **Tingkat 2: Perencanaan Proyek (Project Bundle Todo)**: Milestone capaian bundle proyek (`project_id`).
   - **Tingkat 3: Perencanaan Komponen (Component / Service Todo)**: Task teknis operasional spesifik aplikasi/servis (`component_id`).

---

## 📁 Struktur Berkas Patokan

Berkas-berkas dalam folder ini dikelompokkan berdasarkan domain agar mudah dimaintenance dan dipecah kapan saja:

| Berkas | Kategori | Fokus Patokan |
| :--- | :--- | :--- |
| `01_perencanaan_hierarki.json` | `perencanaan_hierarki` | Roadmap 3 tingkat: Global (Platform), Proyek (Project Bundle), Komponen (Service Task) |
| `02_arsitektur_dan_desain.json` | `arsitektur_dan_desain` | Decoupling backend/frontend, unified dev runner, modularitas file |
| `03_backend_dan_api.json` | `backend_dan_api` | Health check probe (/health), zero CGO SQLite, typed error codes |
| `04_frontend_dan_antarmuka.json`| `frontend_dan_antarmuka`| Typed composables, antislop UI, responsivitas mobile |
| `05_keamanan_dan_hygiene.json` | `keamanan_dan_hygiene` | Zero-credential leak, .env sanitization, pre-flight AI censor |
| `06_performa_dan_jaringan.json` | `performa_dan_jaringan` | Zero port collision, LAN binding (0.0.0.0) |
| `07_ai_dan_mcp_integration.json`| `ai_dan_mcp_integration`| Model Context Protocol (MCP), System One / Jev primitives |
| `08_scale_up_architecture_patterns.json` | `scale_up_arsitektur_backend` | Thin Controller, Pure vs Struct, Repository Abstraction, MCP, MVPP |
| `09_mobile_dan_cross_platform.json` | `mobile_dan_cross_platform` | UI Thread, Memory Leak, Offline Handling, Encrypted Storage |
| `10_testing_dan_quality_assurance.json` | `testing_dan_quality_assurance` | Unit test coverage, integration test, test isolation, regression test |
| `11_error_handling_dan_observability.json` | `error_handling_dan_observability` | Structured logging, error wrapping, panic recovery, graceful shutdown |
| `12_devops_dan_deployment.json` | `devops_dan_deployment` | Multi-stage Dockerfile, .dockerignore, semantic versioning, reproducible build |

---

## 🛠️ Format Standar Setiap Butir Patokan

Setiap berkas JSON menggunakan skema terstandarisasi berikut:

```json
{
  "category": "nama_kategori",
  "name": "Judul Kategori",
  "description": "Deskripsi cakupan kategori",
  "guidelines": [
    {
      "id": "KODE-001",
      "level": "global | project | component",
      "title": "Judul Patokan",
      "rule": "Aturan spesifik yang wajib dipatuhi",
      "rationale": "Alasan teknis/bisnis patokan ini",
      "inspection_check": "Apa yang diinspeksi oleh AI/mesin",
      "suggested_action": "Tindakan saran jika patokan belum terpenuhi",
      "severity": "critical | high | medium | low"
    }
  ]
}
```

---

## 🔄 Cara Menambah atau Mengubah Patokan

1. **Menambah Butir Patokan Baru:**
   - Cukup tambahkan objek baru ke array `guidelines` di berkas kategori yang sesuai.
2. **Membuat Kategori Baru:**
   - Buat berkas baru (misal `08_devops_dan_ci_cd.json`). Engine Vibeasik akan secara otomatis memindai seluruh berkas `*.json` di dalam direktori ini tanpa perlu kompilasi ulang.
3. **Mengubah Ambang Batas:**
   - Ubah langsung teks `rule`, `inspection_check`, atau `severity` untuk mengkalibrasi ketatnya saran yang dihasilkan oleh AI agent.

---

## 🧠 Dipakai Lewat Pusat Pengetahuan

Setiap butir di folder ini dicerminkan otomatis ke **Pusat Pengetahuan** (`local-engine/internal/knowledge`, halaman `/knowledge`) sebagai node `rule` berkunci `guideline:<ID>`. Audit, rapat PM, dan pegawai mengambilnya dari sana sesuai relevansi pekerjaan. Hasil kerja (nilai QA, vonis audit) memberi bukti ✓/✗ pada butir; butir yang berulang kali terbukti keliru ditandai **perlu ditinjau** — perbarui berkas JSON-nya di sini (berkas tetap sumber kebenaran). Lihat Aturan 33 di `AI_CONTEXT.md`.
