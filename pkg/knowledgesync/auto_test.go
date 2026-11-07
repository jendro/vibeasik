package knowledgesync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
)

type autoEnv struct {
	db     *gorm.DB
	now    time.Time
	posted []map[string]any
}

// newAutoEnv menyiapkan HOME sementara, database, paket (Pro atau Free), dan hook jaringan palsu.
func newAutoEnv(t *testing.T, pro bool) *autoEnv {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	db := testDB(t)
	if err := db.AutoMigrate(&models.Project{}, &models.Component{}, &models.CensoredKeyword{}); err != nil {
		t.Fatal(err)
	}
	registry.SetEntitlement(func(f string) bool { return pro && f == FeatureAutoControl })
	e := &autoEnv{db: db, now: time.Date(2026, 11, 20, 10, 0, 0, 0, time.Local)}

	origPost, origTok, origJitter := postContribution, deviceToken, jitter
	postContribution = func(_ string, body map[string]any, out any) error {
		e.posted = append(e.posted, body)
		if r, ok := out.(*struct {
			ID uint `json:"id"`
		}); ok {
			r.ID = uint(len(e.posted))
		}
		return nil
	}
	deviceToken = func() string { return "vbd_test" }
	jitter = func() {}
	t.Cleanup(func() {
		postContribution, deviceToken, jitter = origPost, origTok, origJitter
		registry.SetEntitlement(func(f string) bool { return f == "" })
	})
	return e
}

func (e *autoEnv) add(title, content, source string, helpful, harmful, ageDays int) uint {
	n := models.KnowledgeNode{
		Kind: "guideline", Domain: "web", Title: title, Content: content, Source: source,
		Helpful: helpful, Harmful: harmful, Status: models.KnowledgeNodeActive,
		CreatedAt: e.now.AddDate(0, 0, -ageDays),
	}
	e.db.Create(&n)
	return n.ID
}

func (e *autoEnv) good(i int) uint {
	return e.add(fmt.Sprintf("Batasi ukuran body request %d", i),
		fmt.Sprintf("Atur batas ukuran body HTTP dan timeout server agar permintaan besar nomor %d tidak menghabiskan memori.", i),
		models.KnowledgeSrcPromoted, 5, 0, 10)
}

func TestAutoEligibleFilters(t *testing.T) {
	e := newAutoEnv(t, false)
	good := e.good(1)
	e.add("Helpful kurang", "Isi aman tentang penggunaan indeks database.", models.KnowledgeSrcPromoted, 2, 0, 10)
	e.add("Rasio buruk", "Isi aman tentang caching respons API.", models.KnowledgeSrcPromoted, 5, 3, 10)
	e.add("Terlalu baru", "Isi aman tentang validasi input form.", models.KnowledgeSrcPromoted, 5, 0, 3)
	e.add("Dari hub", "Isi aman dari paket publik.", models.KnowledgeSrcHub, 9, 0, 30)
	own := e.add("Milik pegawai", "Isi aman tetapi pribadi.", models.KnowledgeSrcPromoted, 9, 0, 30)
	owner := uint(1)
	e.db.Model(&models.KnowledgeNode{}).Where("id = ?", own).Update("owner_employee_id", owner)
	e.add("Memuat path", "Simpan konfigurasi di /home/budi/app/config.yml lalu muat ulang.", models.KnowledgeSrcPromoted, 5, 0, 10)
	secret := "ghp" + "_" + strings.Repeat("a", 36)
	e.add("Memuat token", "Gunakan token "+secret+" untuk akses repositori.", models.KnowledgeSrcPromoted, 5, 0, 10)

	el := AutoEligible(e.db, e.now)
	if len(el.OK) != 1 || el.OK[0].ID != good {
		t.Fatalf("hanya node baik yang lolos, dapat %+v", el.OK)
	}
	reasons := map[string]bool{}
	for _, s := range el.Skipped {
		reasons[s.Reason] = true
	}
	if len(el.Skipped) != 2 {
		t.Fatalf("dua node harus dilewati dengan alasan (path & token), dapat %+v", el.Skipped)
	}
	if !reasons["isi memuat path berkas"] || !reasons["penyensor menemukan data sensitif"] {
		t.Fatalf("alasan tidak sesuai: %v", reasons)
	}
}

func TestAutoEligibleSkipsAlreadySentByHash(t *testing.T) {
	e := newAutoEnv(t, false)
	id := e.good(1)
	var n models.KnowledgeNode
	e.db.First(&n, id)
	if err := recordSent(LedgerSent{LocalID: 999, Hash: contentHash(n.Title, n.Content), SentAt: e.now, Mode: "manual"}); err != nil {
		t.Fatal(err)
	}
	if el := AutoEligible(e.db, e.now); len(el.OK) != 0 {
		t.Fatalf("isi yang hash-nya sudah terkirim tidak boleh dikirim lagi: %+v", el.OK)
	}
}

func TestSendAutoRequiresConsent(t *testing.T) {
	e := newAutoEnv(t, false)
	e.good(1)
	n, err := SendAuto(context.Background(), e.db, e.now)
	if !errors.Is(err, ErrConsentRequired) || n != 0 || len(e.posted) != 0 {
		t.Fatalf("tanpa persetujuan tidak boleh mengirim: n=%d err=%v posted=%d", n, err, len(e.posted))
	}
}

func TestSendAutoRequiresLogin(t *testing.T) {
	e := newAutoEnv(t, false)
	e.good(1)
	_ = Consent()
	deviceToken = func() string { return "" }
	if _, err := SendAuto(context.Background(), e.db, e.now); err == nil || len(e.posted) != 0 {
		t.Fatalf("tanpa login tidak boleh mengirim: err=%v posted=%d", err, len(e.posted))
	}
}

func TestSendAutoDailyCapAndLedger(t *testing.T) {
	e := newAutoEnv(t, false)
	for i := 1; i <= 7; i++ {
		e.good(i)
	}
	if err := Consent(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	n, err := SendAuto(ctx, e.db, e.now)
	if err != nil || n != MaxPerDay {
		t.Fatalf("hari pertama harus mengirim %d, dapat %d (%v)", MaxPerDay, n, err)
	}
	if n, _ = SendAuto(ctx, e.db, e.now.Add(2*time.Hour)); n != 0 {
		t.Fatalf("batas harian tercapai, dapat %d", n)
	}
	if n, _ = SendAuto(ctx, e.db, e.now.AddDate(0, 0, 1)); n != 2 {
		t.Fatalf("hari berikutnya sisa 2 node, dapat %d", n)
	}
	if n, _ = SendAuto(ctx, e.db, e.now.AddDate(0, 0, 2)); n != 0 {
		t.Fatalf("tidak boleh mengirim ulang node yang sama, dapat %d", n)
	}
	l := LoadLedger()
	if len(l.Sent) != 7 {
		t.Fatalf("ledger harus mencatat 7 kiriman, dapat %d", len(l.Sent))
	}
	for _, s := range l.Sent {
		if s.Hash == "" || s.Mode != "auto" || s.ServerID == 0 {
			t.Fatalf("entri ledger tidak lengkap: %+v", s)
		}
	}
	if origin := e.posted[0]["origin"]; origin != "auto" {
		t.Fatalf("kiriman harus bertanda origin=auto, dapat %v", origin)
	}
}

func TestModeLockedForFreeAndChangeableForPro(t *testing.T) {
	free := newAutoEnv(t, false)
	_ = free
	if err := SetMode(ModeOff); !errors.Is(err, ErrLocked) {
		t.Fatalf("Free tidak boleh mengubah mode, err=%v", err)
	}
	// Walau berkas state dipalsukan, Free tetap auto.
	st := LoadState()
	st.AutoMode = string(ModeOff)
	_ = saveState(st)
	if EffectiveMode() != ModeAuto {
		t.Fatal("Free harus selalu auto, apa pun isi state")
	}

	e := newAutoEnv(t, true)
	e.good(1)
	_ = Consent()
	if EffectiveMode() != ModeAuto {
		t.Fatal("bawaan Pro adalah auto")
	}
	if err := SetMode("sembarang"); err == nil {
		t.Fatal("mode tidak dikenal harus ditolak")
	}
	for _, m := range []Mode{ModeOff, ModeAsk} {
		if err := SetMode(m); err != nil {
			t.Fatal(err)
		}
		if n, err := SendAuto(context.Background(), e.db, e.now); n != 0 || err != nil || len(e.posted) != 0 {
			t.Fatalf("mode %s tidak boleh mengirim otomatis: n=%d err=%v", m, n, err)
		}
	}
	if err := SetMode(ModeAuto); err != nil {
		t.Fatal(err)
	}
	if n, _ := SendAuto(context.Background(), e.db, e.now); n != 1 {
		t.Fatalf("mode auto harus mengirim, dapat %d", n)
	}
}

func TestPullRequiresConsent(t *testing.T) {
	newAutoEnv(t, false)
	if _, err := Pull(nil, false); !errors.Is(err, ErrConsentRequired) {
		t.Fatalf("tarik paket tanpa persetujuan harus ditolak, err=%v", err)
	}
}

func TestConsentVersionMustMatch(t *testing.T) {
	newAutoEnv(t, false)
	if HasConsent() {
		t.Fatal("belum ada persetujuan")
	}
	st := LoadState()
	st.ConsentVersion, st.ConsentAt = "versi-lama", time.Now()
	_ = saveState(st)
	if HasConsent() {
		t.Fatal("persetujuan versi lama tidak berlaku")
	}
	_ = Consent()
	if !HasConsent() {
		t.Fatal("persetujuan versi sekarang harus berlaku")
	}
}
