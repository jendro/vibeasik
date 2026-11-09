# Spesifikasi Paket Pro, Team/Business, dan AI Credits vibeasik

Dokumen ini mendefinisikan rancangan komersial dan arsitektur teknis paket **Community**, **Pro**, dan **Team/Business**, serta spesifikasi sistem **AI Credits** berbasis temuan aktual dari pelaksanaan pilot service berbayar (*professional services*).

---

## 1. Filosofi Nilai: Menjual Hasil & Workflow, Bukan Token Mentah

Sesuai keputusan Dewan Eksekutif vibeasik (*technical decision*), **vibeasik tidak menjual token AI mentah sebagai komoditas utama**.

```
┌────────────────────────────────────────────────────────────────────────┐
│                        PARADIGMA MONETISASI                            │
├───────────────────────────────────┬────────────────────────────────────┤
│   JUAL TOKEN MENTAH (DIHINDARI)   │     AI CREDITS VIBEASIK (DIPILIH)   │
├───────────────────────────────────┼────────────────────────────────────┤
│ • Harga mudah dibanding-bandingkan│ • Menjual hasil akhir bernilai     │
│ • Risiko margin negatif fluktuasi │ • Biaya terprediksi per tugas      │
│ • User pusing hitung prompt/output│ • 1 AI Review = 2 Credits          │
│ • Beban mental saat iterasi dev   │ • 1 Deep Check = 10 Credits        │
│ • Vendor lock-in yang kaku        │ • Bebas BYOK tanpa sandera fitur   │
└───────────────────────────────────┴────────────────────────────────────┘
```

### Kenapa Menolak Menjual Token Mentah?
1. **Fluktuasi Model & Volatilitas Biaya**: Harga token per provider (OpenAI, Anthropic, Google) berubah setiap beberapa bulan. Menjual token langsung menciptakan risiko margin negatif saat model baru yang lebih cerdas (dan mahal) dirilis.
2. **Kecemasan Pengembang (Developer Friction)**: Pengembang tidak ingin menghitung berapa token yang dihabiskan untuk satu kali audit AST atau multi-turn PR review. Mereka ingin kepastian: *"Berapa kali review yang bisa saya jalankan bulan ini?"*
3. **Penyelarasan Nilai**: Nilai vibeasik terletak pada **orkestrasi**, **Pusat Pengetahuan (Knowledge Mesh)**, **85+ aturan audit**, **peta kode (Code Map)**, dan **pegawai AI yang patuh standar**. AI hanyalah mesin inferensi di balik alur kerja terstruktur tersebut.

---

## 2. Temuan Pilot Service: Fitur Paling Sering Digunakan & Kebutuhan Nyata

Berdasarkan data wawancara dan eksekusi pilot pada 4 calon pelanggan (`docs/SERVICES.md`):

| Segmen Pelanggan | Kebutuhan Inti | Fitur yang Paling Sering Digunakan | Frekuensi Rata-Rata | Keberatan Terbesar (*Objection*) |
| :--- | :--- | :--- | :--- | :--- |
| **FinTech Scale-up** (18 devs, Go) | Keamanan pre-ISO & mitigasi latensi DB | • Fast Scan (85 rule)<br>• Deep Check (N+1, deadlocks)<br>• QA Gate | 60–100 run audit / bulan | **Privasi Kode**: Menolak keras jika source code diunggah ke cloud luar. |
| **B2B SaaS HRIS** (8 devs, Nuxt/Node) | Standarisasi kualitas tim junior | • Pegawai AI (Staff Office)<br>• AI PR Review<br>• Galeri Desain UI | 150–220 PR review & task / bulan | **Kualitas**: Khawatir AI menghasilkan kode halusinasi/*AI slop*. |
| **Software Agensi** (12 devs, 15+ repo) | Onboarding audit intake proyek baru | • Fast Scan<br>• Topologi Code Map<br>• Ekspor Health Score | 20–35 audit repo / bulan | **Kecepatan**: Turnaround harus di bawah 24 jam. |
| **E-Commerce Scale-up** (25 devs, Monorepo) | Performa database event promo besar | • Deep Check DB query<br>• Remediation Sprint | 50–80 deep runs / bulan | **Stabilitas**: Takut refactoring merusak logika transaksi stabil. |

### Urutan Prioritas Fitur Paling Bernilai:
1. **Fast Scan & Code Map** (0 Biaya AI Cloud — 100% engine Go lokal).
2. **Deep Check & Remediation Analysis** (Memerlukan LLM reasoning tinggi untuk menganalisis call graph).
3. **AI Code Review & QA Gate** (Evaluasi kepatuhan patokan arsitektur, diff analysis).
4. **Staff Office & Orchestrator** (Multi-turn execution, pengawalan todo, konsolidasi pengetahuan).
5. **Design Guidance & Visual Audit** (Vision multimodal LLM).

---

## 3. Struktur Biaya Provider AI (Unit Economics) & Margin

Untuk menjaga kesehatan bisnis, paket cloud dirancang dengan target **Gross Margin 75%–85%**.

### Estimasi Konsumsi Token & Biaya Aktual per Aksi Fitur

| Fitur | Estimasi Input Token | Estimasi Output Token | Biaya Provider (Gemini 1.5 Pro / Claude 3.5 Sonnet) | Alokasi AI Credits | Nilai Jual Setara Credit (~$0.01/credit) | Gross Margin Est. |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Fast Scan & Code Map** | 0 (Lokal) | 0 (Lokal) | $0.000 | **0 Credit (Gratis)** | $0.00 | 100% |
| **AI Code Review (per PR / Diff)** | ~4.000 | ~1.000 | $0.015 – $0.022 | **2 Credits** | $0.020 | **~75%** (bila pakai mix Flash/Sonnet) |
| **Deep Check (Arsitektur/DB)** | ~15.000 | ~3.000 | $0.065 – $0.095 | **10 Credits** | $0.100 | **~72% – 80%** |
| **Staff Run (1 Todo Execution)** | ~28.000 (multi-turn) | ~6.000 | $0.140 – $0.210 | **20 Credits** | $0.200 | **~70% – 76%** |
| **Design Library / Vision Audit** | 1 Screenshot + ~2.500 | ~800 | $0.012 – $0.018 | **2 Credits** | $0.020 | **~75%** |
| **Chat Konsultasi Planner (per turn)** | ~3.000 | ~800 | $0.008 – $0.012 | **1 Credit** | $0.010 | **~78%** |

> **Prinsip Konversi**: `1 AI Credit ≈ $0.01 USD` (atau setara ~Rp 160 IDR).

---

## 4. Definisi Tiga Paket: Community, Pro, dan Team/Business

Sesuai kaidah pemisahan kapabilitas (K5535), setiap paket memetakan batasan operasional dan komersial yang eksplisit.

```
┌───────────────────────────────────────────────────────────────────────────────────┐
│                           MATRIKS TIER PRODUK VIBEASIK                            │
├──────────────────────┬──────────────────────┬─────────────────────────────────────┤
│   COMMUNITY (FREE)   │       PRO TIER       │        TEAM / BUSINESS TIER         │
│  "Local-First Power" │  "Autonomous Craft"  │   "Collaborative Org Intelligence"  │
├──────────────────────┼──────────────────────┼─────────────────────────────────────┤
│ • $0 / Gratis        │ • $19 / bulan        │ • $49 / seat / bulan (min. 3 seats) │
│ • 64 MCP Tools       │ • 123 MCP Tools      │ • 123 MCP Tools + Org Admin Tools   │
│ • 0 Cloud Credits    │ • 1.000 Credits/bln  │ • 3.000 Credits / seat / bulan      │
│ • Full BYOK Support  │ • Cloud LLM + BYOK   │ • Pooled Team Credits + Org BYOK    │
│ • Local Knowledge    │ • Knowledge Mesh L1  │ • Knowledge Mesh L2 (Shared Mesh)   │
│ • 1 User Standalone  │ • 1 Dev Seat (3 dev) │ • Multi-Seat, Org Dashboard, Audit  │
└──────────────────────┴──────────────────────┴─────────────────────────────────────┘
```

### Rincian Lengkap Entitlement per Paket

| Parameter | Community (Free) | Pro (Individual Power Dev) | Team / Business (Scale-up & Ent) |
| :--- | :--- | :--- | :--- |
| **Harga Resmi** | **Rp 0** (Gratis selamanya) | **Rp 299.000 / bulan**<br>(~$19 USD / bulan) | **Rp 750.000 / seat / bulan**<br>(~$49 USD / seat / bulan, min. 3 seat) |
| **Bentuk Distribusi** | Build Community (Apache-2.0, open-source) | Build Official (Binary terkompilasi, auto-updater) | Build Official + Org Cloud Coordinator |
| **Target Pengguna** | Mahasiswa, solo dev, open-source contributor | Senior engineer, konsultan, freelance dev | Startup tim (5–30 devs), software house, divisi enterprise |
| **Jumlah Device per Lisensi** | Tidak terbatas (tanpa akun) | Maksimal 3 perangkat aktif per akun | Hingga 5 perangkat per seat anggota tim |
| **Katalog MCP Tools** | **64 Tools** (Scan, Peta Kode, Port, Runner, Todo) | **123 Tools** (Semua modul Pro terbuka) | **123 Tools** + Enterprise Org API |
| **AI Credits Bulanan** | 0 (Mendukung BYOK & Local LLM tanpa batas) | **1.000 Credits / bulan** (~500 reviews atau 100 deep checks) | **3.000 Credits / seat / bulan** (Pooled pool tim; 5 dev = 15.000 credits) |
| **Top-up Add-on Credits** | Tidak tersedia (gunakan BYOK) | **Rp 150.000 per 1.000 Credits** | **Rp 1.200.000 per 10.000 Credits** (Faktur perusahaan) |
| **Model AI Bawaan** | BYOK (OpenAI, Anthropic, Gemini) atau Ollama/Llama lokal | Cloud Managed Router (Zero config, potong credit) | Cloud Managed Router dengan prioritas latensi enterprise |
| **Pusat Pengetahuan** | Paket dasar tersemat (read-only lokal) | Jaring Lengkap (L1): kurasi lokal + auto-sync global | Jaring Lengkap (L2): repositori patokan internal perusahaan |
| **Kantor Pegawai AI (Staff Office)** | Terkunci (`upgrade_required`) | 1 Kantor pribadi, 5 staf default | Kantor tim terdistribusi, agenda sprint bersama, custom persona |
| **Penyensoran Rahasia (SEC-003)** | Ya (GlobalSanitizer bawaan) | Ya (GlobalSanitizer pre-flight mandatory) | Ya + Kustomisasi regex rahasia internal tim |
| **SLA & Support** | GitHub Discussions / Community Discord | Email prioritas (respon < 24 jam) | Dedicated Slack/Discord, respon < 4 jam, 2 jam onboarding |

---

## 5. Arsitektur Teknis Backend (Go · Fiber · GORM)

Sebagai arsitektur backend yang kokoh, sistem mematuhi prinsip **pemisahan aturan harga dari kode kalkulasi (K4979)** dan **penegakan batas pada perimeter API (K5662)**.

### A. Diagram Alur Verifikasi Entitlement & Penanganan Kuota

```
  Pengembang / Agent (MCP / Studio)
                 │
                 ▼
 ┌──────────────────────────────────────────────────────────────┐
 │ 1. Fiber Gateway (Perimeter API)                             │
 │    - Cek rute: apakah butuh fitur Pro?                       │
 │    - Cek registry.Has(feature) via pkg/account               │
 └──────────────────────────────┬───────────────────────────────┘
                                │ Lolos Entitlement
                                ▼
 ┌──────────────────────────────────────────────────────────────┐
 │ 2. Inference Mode Resolver                                   │
 │    - Apakah pengguna memakai BYOK (API Key sendiri / Ollama)?│
 └──────────────┬───────────────────────────────┬───────────────┘
                │ BYOK Aktif                    │ Pakai Cloud Managed Credits
                │ (Bypass Kuota)                ▼
                │             ┌─────────────────────────────────────────────────┐
                │             │ 3. Credit Quota Enforcement                     │
                │             │    - Cek saldo akun: (balance >= cost)?         │
                │             │    - IF saldo cukup: Hold/Reserve credit        │
                │             │    - IF saldo habis: Kembalikan HTTP 402        │
                │             │      ("quota_exceeded" + saran fallback BYOK)   │
                │             └────────────────────────┬────────────────────────┘
                │                                      │ Saldo Cukup
                ▼                                      ▼
 ┌──────────────────────────────────────────────────────────────┐
 │ 4. SEC-003 Pre-flight Sanitizer                              │
 │    - Masking ADMIN_PASSWORD, JWT_SECRET, PRIVATE_KEY         │
 └──────────────────────────────┬───────────────────────────────┘
                                │
                                ▼
 ┌──────────────────────────────────────────────────────────────┐
 │ 5. LLM Provider Execution (Anthropic / Gemini / OpenAI)       │
 └──────────────────────────────┬───────────────────────────────┘
                                │ Response Diterima
                                ▼
 ┌──────────────────────────────────────────────────────────────┐
 │ 6. Ledger Settlement (Append-Only Transaction)               │
 │    - Commit pemotongan kredit aktual di CreditLedger         │
 │    - Kirim respon hasil ke klien                             │
 └──────────────────────────────────────────────────────────────┘
```

### B. Desain Skema Database (GORM Models untuk vibeasik-cloud)

Pemisahan tabel aturan harga, saldo akun, dan buku besar pemakaian (*immutable append-only ledger*):

```go
package billing

import (
	"time"
	"gorm.io/gorm"
)

// AccountBilling profil kredit satu akun.
type AccountBilling struct {
	AccountID           uint      `gorm:"primaryKey" json:"account_id"`
	Plan                string    `gorm:"size:32;not null;default:'community'" json:"plan"`
	CreditsBalance      int64     `gorm:"not null;default:0" json:"credits_balance"`
	MonthlyQuota        int64     `gorm:"not null;default:0" json:"monthly_quota"`
	QuotaResetAt        time.Time `json:"quota_reset_at"`
	AllowBYOK           bool      `gorm:"not null;default:true" json:"allow_byok"`
	IsSuspended         bool      `gorm:"not null;default:false" json:"is_suspended"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// CreditLedger buku besar mutasi kredit (immutable audit trail).
// Tidak pernah di-UPDATE atau di-DELETE; hanya INSERT.
type CreditLedger struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	AccountID      uint      `gorm:"index;not null" json:"account_id"`
	DeviceID       string    `gorm:"size:40;index" json:"device_id"`
	Amount         int64     `gorm:"not null" json:"amount"` // Negatif = pemakaian, Positif = top-up/kuota bulanan
	BalanceAfter   int64     `gorm:"not null" json:"balance_after"`
	Feature        string    `gorm:"size:64;index;not null" json:"feature"` // mis: "audit.deep", "audit.ai_review"
	ReferenceID    string    `gorm:"size:64" json:"reference_id"`           // mis: run_id, todo_id, atau order_id
	TokensIn       int       `json:"tokens_in"`
	TokensOut      int       `json:"tokens_out"`
	Provider       string    `gorm:"size:32" json:"provider"` // mis: "anthropic", "gemini"
	ModelName      string    `gorm:"size:64" json:"model_name"`
	CreatedAt      time.Time `gorm:"index" json:"created_at"`
}

// FeatureCostVersion aturan biaya per fitur yang ber-versi (K4979).
type FeatureCostVersion struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	FeatureKey  string    `gorm:"size:64;index;not null" json:"feature_key"`
	CreditCost  int64     `gorm:"not null" json:"credit_cost"`
	EffectiveAt time.Time `gorm:"not null" json:"effective_at"`
	RetiredAt   *time.Time `json:"retired_at,omitempty"`
}
```

### C. Format Payload Entitlement Token (Ed25519) Versi 2

Token entitlement yang diterbitkan server cloud dan diverifikasi lokal oleh engine melalui `pkg/account/entitlement.go`:

```json
{
  "v": 2,
  "sub": "usr_948194a8f",
  "email": "lead@fintech.co.id",
  "device": "dev_macbook_m3_office",
  "plan": "pro",
  "features": [
    "audit.deep",
    "audit.ai_review",
    "audit.learning",
    "runtime_sim",
    "staff",
    "planner",
    "orchestrator",
    "llm.router",
    "design_library",
    "office",
    "knowledge.full",
    "knowledge.auto_control"
  ],
  "credits_balance": 840,
  "credits_reset_at": 1796083200,
  "byok_allowed": true,
  "iat": 1793491200,
  "exp": 1794096000,
  "grace_days": 14
}
```

---

## 6. Pembatasan Kuota & Perilaku Ketika Kuota Habis (Graceful Fallback)

### Prinsip Utama: Tidak Ada Fitur Lokal yang Disandera!
vibeasik adalah engine *local-first*. Ketika AI Credits bulanan habis:

1. **Fitur Lokal Tetap 100% Aktif**: Fast Scan, Code Map, Port Registry, Runner, dan 64 MCP tools Community tetap berjalan tanpa batasan apa pun.
2. **BYOK (Bring Your Own Key) Tidak Pernah Diblokir**: Jika pengguna memasukkan API key pribadi (Anthropic/OpenAI/Gemini) atau menghubungkan model lokal (Ollama / System One), seluruh fitur Pro (Deep Check, AI Review, Pegawai AI) **tetap dapat dijalankan secara gratis tanpa memotong kredit**.
3. **Format Respons API Ketika Kuota Habis (HTTP 402)**:

```json
{
  "error": "quota_exceeded",
  "message": "AI Credits bulanan Anda telah habis (tersisa 0 credits). Fitur lokal tetap aktif.",
  "feature": "audit.deep",
  "required_credits": 10,
  "current_balance": 0,
  "quota_reset_at": "2026-12-01T00:00:00Z",
  "fallback_options": {
    "byok": {
      "enabled": true,
      "instructions": "Masukkan API Key Anda di Settings > LLM Provider untuk melanjutkan tanpa batas kredit vibeasik.",
      "settings_url": "/settings/llm"
    },
    "local_llm": {
      "supported": ["ollama", "system-one", "llama.cpp"],
      "instructions": "Arahkan endpoint LLM ke server inferensi lokal."
    },
    "top_up_url": "https://vibeasik.com/account/credits"
  }
}
```

4. **Indikator Proaktif di UI (Studio)**:
   - Persentase sisa kredit ditampilkan di header Studio.
   - Peringatan transisi warna:
     - Hijau: > 20% sisa kuota.
     - Oranye: 5% – 20% sisa kuota.
     - Merah: < 5% sisa kuota, dengan prompt menawarkan aktivasi BYOK.

---

## 7. Transparansi Privasi Kode & Keamanan Enterprise

Dari hasil wawancara pilot, **privasi source code adalah penentu utama pembelian enterprise**. Oleh karena itu, vibeasik menerapkan standar keamanan ketat:

### 1. SEC-003 Pre-flight AI Censor Sanitization (K34)
Setiap prompt, potongan diff, atau ringkasan arsitektur yang akan dikirim ke model AI cloud **wajib disaring oleh `GlobalSanitizer` lokal sebelum payload keluar dari mesin**:
- Token rahasia (`PRIVATE KEY`, `JWT_SECRET`, token OAuth) digantikan dengan placeholder `[REDACTED_SECRET]`.
- Kata sandi konfigurasi database (`ADMIN_PASSWORD`, `DB_PASS`) disamarkan.
- Kredensial cloud (`AKIA...`, service account keys) otomatis di-mask.

### 2. Kebijakan Zero Data Retention (ZDR)
- Server proxy vibeasik **tidak pernah menyimpan, meng-cache, atau menulis kode sumber pelanggan ke disk**.
- Kontrak penyedia AI tingkat enterprise menggunakan flag `no-store` dan kebijakan **tidak menggunakan data pelanggan untuk training model**.

### 3. Sovereign Air-Gapped Mode
Untuk lingkungan perbankan atau fasilitas keamanan tinggi, seluruh suite vibeasik dapat dikonfigurasi berjalan dalam isolasi total:
- Inferensi AI dialihkan 100% ke cluster GPU lokal atau endpoint intranet (vLLM / Ollama / System One).
- Validasi lisensi menggunakan token offline berdurasi tahunan tanpa koneksi internet sama sekali.

---

## 8. Rencana Implementasi Bertahap (Iterasi Kecil)

Mengikuti prinsip *"Commit kecil, hati tenang"*, implementasi sistem komersial dilakukan bertahap sesuai validasi pasar:

```
    FASE 1 (Saat Ini)                FASE 2                        FASE 3
┌───────────────────────┐   ┌───────────────────────┐   ┌───────────────────────┐
│ Definisi Produk &     │   │ Telemetri Lokal &     │   │ Billing Gateway &     │
│ Spesifikasi Kontrak   │──▶│ Metering Pasif        │──▶│ Self-Serve Top-Up     │
│ (Docs, Matriks Tier,  │   │ (Menghitung konsumsi  │   │ (Stripe/Midtrans      │
│ Entitlement v2 Spec)  │   │ nyata tanpa tagihan)  │   │ integrasi penuh)      │
└───────────────────────┘   └───────────────────────┘   └───────────────────────┘
```

1. **Fase 1 (Selesai pada Todo ini)**: Dokumentasi lengkap edisi, penataan paket Community, Pro, Team, definisi AI Credits, dan kontrak arsitektur data.
2. **Fase 2 (Observabilitas Penggunaan)**: Implementasi penghitung konsumsi credit lokal di Studio agar pengembang dapat melihat berapa estimasi biaya yang mereka hemat dengan vibeasik.
3. **Fase 3 (Billing Otomatis)**: Peluncuran modul pembayaran online mandiri setelah transaksi pilot service awal mencapai target validasi pasar.
