package knowledgesync

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/account"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/security"
	"github.com/vibeasik/vibeasik/pkg/setup"
)

// Kirim otomatis pengetahuan berguna. Rancangan: vibeasik-docs/docs/KNOWLEDGE_AUTO_CONTRIBUTE.md.
//
// Ringkas: bawaan menyala (mode auto). Pengguna paket Pro boleh mengubahnya; pengguna Free tidak. Apa pun modenya,
// hanya node yang lolos syarat ketat di bawah yang dikirim, dan hanya setelah pengguna menyetujui Syarat & Ketentuan.

// Mode pengiriman usulan pengetahuan.
type Mode string

const (
	ModeAuto Mode = "auto" // node lolos syarat dikirim otomatis (bawaan)
	ModeAsk  Mode = "ask"  // pengguna menyetujui satu per satu (`knowledge contribute --send`)
	ModeOff  Mode = "off"  // tidak ada usulan yang dikirim
)

// FeatureAutoControl fitur entitlement yang mengizinkan pengguna mengubah mode (Pro).
const FeatureAutoControl = "knowledge.auto_control"

// ConsentVersion versi Syarat & Ketentuan Pusat Pengetahuan yang harus disetujui. Naikkan bila pasal 4 berubah.
const ConsentVersion = "2026-11-draf-4"

// Syarat kelayakan node (lihat rancangan §3).
const (
	MinHelpful    = 3 // helpful minimal
	HelpfulRatio  = 3 // helpful >= HelpfulRatio * harmful
	MaxPerDay     = 5 // per perangkat; server membatasi 50 per akun
	maxContentLen = 8000
	maxTitleLen   = 191
)

// ErrLocked mode tidak dapat diubah pada paket ini.
var ErrLocked = errors.New("pengaturan ini terkunci pada paket Free: kirim otomatis adalah bagian dari Pusat Pengetahuan (Pro dapat mengubahnya)")

// ErrConsentRequired Syarat & Ketentuan belum disetujui.
var ErrConsentRequired = errors.New("Pusat Pengetahuan belum aktif: setujui Syarat & Ketentuan dengan `vibeasik knowledge consent`")

// CanControl true bila paket pengguna boleh mengubah mode.
func CanControl() bool { return registry.Has(FeatureAutoControl) }

// EffectiveMode mode yang berlaku: Free selalu auto; Pro memakai pilihannya (bawaan auto).
func EffectiveMode() Mode {
	if !CanControl() {
		return ModeAuto
	}
	switch m := Mode(LoadState().AutoMode); m {
	case ModeAsk, ModeOff, ModeAuto:
		return m
	}
	return ModeAuto
}

// SetMode mengubah mode (hanya bila paket mengizinkan).
func SetMode(m Mode) error {
	if m != ModeAuto && m != ModeAsk && m != ModeOff {
		return fmt.Errorf("mode tidak dikenal %q (auto, ask, off)", m)
	}
	if !CanControl() {
		return ErrLocked
	}
	st := LoadState()
	st.AutoMode = string(m)
	return saveState(st)
}

// HasConsent true bila pengguna pernah menyetujui S&K, di wizard penyiapan awal atau di Setting.
// Persetujuan awal tetap berlaku saat ConsentVersion naik: pembaruan ketentuan tidak meminta persetujuan
// ulang (keputusan pemilik 2026-11-08). Versi yang disetujui tetap tercatat apa adanya untuk riwayat.
func HasConsent() bool {
	st := LoadState()
	if !st.ConsentAt.IsZero() {
		return true
	}
	return !setup.Load().TermsAt.IsZero()
}

// Consent mencatat persetujuan S&K versi saat ini.
func Consent() error {
	st := LoadState()
	st.ConsentVersion = ConsentVersion
	st.ConsentAt = time.Now()
	return saveState(st)
}

// AutoCandidate node yang memenuhi semua syarat kirim otomatis.
type AutoCandidate struct {
	Candidate
	Hash string `json:"hash"`
}

// Eligibility hasil penyaringan: yang lolos dan yang dilewati beserta alasannya.
type Eligibility struct {
	OK      []AutoCandidate
	Skipped []LedgerSkip
}

// AutoEligible menyaring node lokal sesuai syarat rancangan §3 (tanpa mengirim apa pun).
func AutoEligible(db *gorm.DB, now time.Time) Eligibility {
	var nodes []models.KnowledgeNode
	db.Where("owner_employee_id IS NULL AND status = ? AND source IN ?", models.KnowledgeNodeActive,
		[]string{models.KnowledgeSrcPromoted, models.KnowledgeSrcReview, models.KnowledgeSrcGap}).
		Where("helpful >= ? AND helpful >= ? * harmful", MinHelpful, HelpfulRatio).
		Order("helpful DESC, id").Limit(200).Find(&nodes)

	ledger := LoadLedger()
	st := LoadState()
	sentLocal := map[uint]bool{}
	for _, id := range st.Contributed {
		sentLocal[id] = true
	}
	pc := NewPrivacyChecker(db)

	var out Eligibility
	for _, n := range nodes {
		if sentLocal[n.ID] {
			continue
		}
		h := contentHash(n.Title, n.Content)
		if ledger.HasHash(h) || ledger.HasServerRejected(h) {
			continue
		}
		skip := func(reason string) {
			out.Skipped = append(out.Skipped, LedgerSkip{LocalID: n.ID, Hash: h, Reason: reason, At: now})
		}
		if len(n.Title) > maxTitleLen || len(n.Content) > maxContentLen || n.Title == "" || n.Content == "" {
			skip("panjang tidak sesuai")
			continue
		}
		// Sensor tidak boleh mengubah apa pun: bila perlu disunting, kemungkinan ada rahasia, jadi tidak dikirim.
		if security.SanitizeText(n.Title) != n.Title || security.SanitizeText(n.Content) != n.Content {
			skip("penyensor menemukan data sensitif")
			continue
		}
		if r := pc.Check(n.Title); r != "" {
			skip("judul " + r)
			continue
		}
		if r := pc.Check(n.Content); r != "" {
			skip("isi " + r)
			continue
		}
		out.OK = append(out.OK, AutoCandidate{
			Candidate: Candidate{ID: n.ID, Kind: n.Kind, Domain: n.Domain, Title: n.Title, Content: n.Content, Source: n.Source, Key: n.Key},
			Hash:      h,
		})
	}
	return out
}

// Hook untuk pengujian.
var (
	postContribution = func(tok string, body map[string]any, out any) error {
		return post("/api/v1/knowledge/contributions", tok, body, out)
	}
	deviceToken = account.DeviceToken
	jitter      = func() { time.Sleep(time.Duration(2+rand.IntN(18)) * time.Second) }
)

// SendAuto mengirim node yang lolos syarat, paling banyak MaxPerDay per hari. Mengembalikan jumlah terkirim.
// Tidak mengirim bila mode bukan auto, S&K belum disetujui, atau perangkat belum login.
func SendAuto(ctx context.Context, db *gorm.DB, now time.Time) (int, error) {
	if EffectiveMode() != ModeAuto {
		return 0, nil
	}
	if !HasConsent() {
		return 0, ErrConsentRequired
	}
	tok := deviceToken()
	if tok == "" {
		return 0, errors.New("belum login: kiriman hanya dari perangkat yang masuk akun")
	}
	el := AutoEligible(db, now)
	_ = recordSkips(el.Skipped)
	remaining := MaxPerDay - LoadLedger().SentOn(now)
	sent := 0
	for _, c := range el.OK {
		if remaining <= 0 {
			break
		}
		if sent > 0 {
			select {
			case <-ctx.Done():
				return sent, ctx.Err()
			default:
				jitter()
			}
		}
		var resp Outcome
		err := postContribution(tok, map[string]any{
			"kind": c.Kind, "domain": c.Domain, "title": c.Title, "content": c.Content, "source_key": c.Key, "origin": "auto",
		}, &resp)
		if err != nil {
			return sent, err
		}
		if resp.Status == "rejected" {
			// Server menolak (skrining ulang): jangan dikirim lagi, catat alasannya agar terlihat di ledger.
			_ = recordSkips([]LedgerSkip{{LocalID: c.ID, Hash: c.Hash, Reason: serverRejectedPrefix + resp.Note, At: now}})
			st := LoadState()
			st.Contributed = append(st.Contributed, c.ID)
			_ = saveState(st)
			continue
		}
		status := resp.Status
		if status == "" {
			status = "screened"
		}
		if err := recordSent(LedgerSent{LocalID: c.ID, Hash: c.Hash, Title: c.Title, SentAt: now, ServerID: resp.ID, Mode: "auto", Status: status}); err != nil {
			return sent, err
		}
		st := LoadState()
		st.Contributed = append(st.Contributed, c.ID)
		_ = saveState(st)
		sent++
		remaining--
	}
	if sent > 0 {
		log.Printf("[knowledge] %d usulan terkirim otomatis (riwayat: vibeasik knowledge ledger)", sent)
	}
	return sent, nil
}

// ConsentText ringkasan Syarat & Ketentuan Pusat Pengetahuan yang ditampilkan sebelum persetujuan.
// Teks lengkap: pasal 4 Syarat & Ketentuan di vibeasik.com/terms.
const ConsentText = `Prinsip utama:
  • Data koding Anda tetap aman di laptop atau server Anda: kode, data proyek, dan kunci tidak diunggah.
  • vibeasik adalah auditor lokal dan aplikasi manajemen proyek yang membantu memastikan aplikasi Anda aman.
  • Yang dikumpulkan hanya "pokok ilmu" (node pengetahuan yang lolos penyaringan), bukan kode atau data kerja Anda.
  • Hanya pokok ilmu itu yang dipakai meningkatkan pengetahuan vibeasik, termasuk melatih dan mengevaluasi model dan
    AI vibeasik, agar hasilnya lebih maksimal.

Pusat Pengetahuan bekerja dua arah.

  • Anda mendapat paket panduan dan checklist; pencarian pengetahuan berjalan di komputer Anda.
  • Node pengetahuan di komputer Anda yang terbukti berguna dikirim otomatis ke server vibeasik setelah disensor
    dan disaring, untuk dipelajari dan dikurasi. Pada paket Free ini bawaan dan tidak dapat dimatikan; pengguna Pro
    dapat mengubah atau mematikannya.

Yang dikirim : judul dan isi node pengetahuan yang lolos penyaringan, serta angka helpful/harmful (anonim).
Tidak dikirim: kode proyek, nama proyek/komponen/repo, path berkas, hostname, kunci & kredensial, pengetahuan
               pribadi pegawai AI Anda. Node yang perlu disunting penyensor tidak dikirim sama sekali.

Dampak pengetahuan (angka saja):
  • Untuk node dari paket, aplikasi mengirim berapa kali node itu dipakai, berapa kali terbukti berguna atau merugikan,
    dan rinciannya per jenis pekerjaan besar (pengerjaan todo, audit kode, perencanaan proyek) beserta area node-nya.
  • Server menampilkannya sebagai grafik "dampak di komunitas" dan hanya bila berasal dari beberapa perangkat berbeda.
  • Tidak dikirim: nama atau isi pekerjaan, nama proyek, path, dan kode. Hanya jenis pekerjaan dari daftar baku.

Data pemakaian (terpisah dari pengetahuan):
  • Aplikasi mengirim ping ringan ke server vibeasik saat dijalankan lalu tiap 24 jam, dengan atau tanpa akun:
    ID instalasi acak, sistem operasi dan versinya, arsitektur, versi dan edisi vibeasik, bahasa dan zona waktu,
    jenis database, jumlah proyek (angka saja), dan apakah Anda masuk akun.
  • Server juga melihat alamat IP dan perkiraan negara/kota dari setiap permintaan, dan menyimpannya bersama ping.
    Saat Anda mendaftar atau masuk, IP dan user agent dicatat pada akun Anda.
  • Dipakai untuk menghitung jumlah pengguna, mendukung versi yang dipakai, mencegah penyalahgunaan, dan arah produk.
    Tidak dijual dan tidak dibagikan ke pihak ketiga untuk iklan.
  • Tidak dikirim: kode, nama proyek, path, hostname, nama pengguna komputer, isi berkas, kunci.
  • Matikan dengan variabel lingkungan VIBEASIK_NO_TELEMETRY=1 sebelum menjalankan vibeasik.

Kiriman masuk karantina dan ditinjau sebelum dipakai pengguna lain. Anda dapat melihat pratinjau
(vibeasik knowledge auto preview) dan riwayat (vibeasik knowledge ledger).

Lisensi:
  • Isi yang Anda kirim: Anda menyatakan isinya bukan rahasia pihak mana pun dan memberi vibeasik lisensi di seluruh
    dunia, selamanya, tidak dapat ditarik untuk bagian yang sudah terbit, boleh disublisensikan dan diubah, untuk
    memakai dan menerbitkannya dalam paket pengetahuan (paket gratis maupun paket penuh). Hak cipta tetap pada Anda.
  • Paket gratis: CC BY 4.0 (boleh dipakai dan diterbitkan ulang dengan menyebut vibeasik).
  • Paket penuh: milik vibeasik, penggunaan terbatas (Lisensi Pengetahuan vibeasik: tidak boleh diterbitkan ulang,
    dijual, disalin massal, atau dipakai melatih model AI yang bersaing).
  • Kode vibeasik: Apache-2.0.`

// Outcome jawaban server atas satu kiriman.
type Outcome struct {
	ID     uint   `json:"id"`
	Status string `json:"status"` // screened | rejected | duplicate
	Note   string `json:"note,omitempty"`
}
