package knowledgesync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vibeasik/vibeasik/pkg/localauth"
)

// Ledger riwayat kiriman pengetahuan di ~/.vibeasik/contributions.json. Hanya menyimpan hash isi dan metadata,
// tidak menyalin teks node. Pengguna dapat melihatnya (`vibeasik knowledge ledger`) dan alasan sebuah node dilewati.
type Ledger struct {
	Sent    []LedgerSent `json:"sent"`
	Skipped []LedgerSkip `json:"skipped"`
}

// LedgerSent satu node yang sudah dikirim.
type LedgerSent struct {
	LocalID  uint      `json:"local_id"`
	Hash     string    `json:"hash"`
	Title    string    `json:"title"` // judul yang sudah disensor, agar riwayat mudah dibaca (bukan isi)
	SentAt   time.Time `json:"sent_at"`
	ServerID uint      `json:"server_id,omitempty"`
	Mode     string    `json:"mode"` // auto | manual
}

// LedgerSkip node yang memenuhi syarat dasar tetapi ditolak penyaring, beserta alasannya.
type LedgerSkip struct {
	LocalID uint      `json:"local_id"`
	Hash    string    `json:"hash"`
	Reason  string    `json:"reason"`
	At      time.Time `json:"at"`
}

const maxSkipped = 300

var ledgerMu sync.Mutex

func ledgerPath() string { return filepath.Join(localauth.Dir(), "contributions.json") }

// LoadLedger membaca ledger (kosong bila belum ada).
func LoadLedger() Ledger {
	var l Ledger
	if b, err := os.ReadFile(ledgerPath()); err == nil {
		_ = json.Unmarshal(b, &l)
	}
	return l
}

func (l Ledger) save() error {
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ledgerPath()), 0o700); err != nil {
		return err
	}
	tmp := ledgerPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ledgerPath())
}

// HasHash true bila isi dengan hash ini sudah pernah dikirim.
func (l Ledger) HasHash(h string) bool {
	for _, s := range l.Sent {
		if s.Hash == h {
			return true
		}
	}
	return false
}

// SentOn jumlah kiriman pada hari kalender lokal dari t.
func (l Ledger) SentOn(t time.Time) int {
	y, m, d := t.Date()
	n := 0
	for _, s := range l.Sent {
		sy, sm, sd := s.SentAt.In(t.Location()).Date()
		if sy == y && sm == m && sd == d {
			n++
		}
	}
	return n
}

// recordSent menambah satu entri terkirim (aman dipanggil bersamaan).
func recordSent(e LedgerSent) error {
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	l := LoadLedger()
	l.Sent = append(l.Sent, e)
	return l.save()
}

// recordSkips mencatat node yang dilewati; entri dengan hash dan alasan yang sama tidak digandakan.
func recordSkips(skips []LedgerSkip) error {
	if len(skips) == 0 {
		return nil
	}
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	l := LoadLedger()
	have := map[string]bool{}
	for _, s := range l.Skipped {
		have[s.Hash+"|"+s.Reason] = true
	}
	changed := false
	for _, s := range skips {
		if have[s.Hash+"|"+s.Reason] {
			continue
		}
		l.Skipped = append(l.Skipped, s)
		changed = true
	}
	if !changed {
		return nil
	}
	if len(l.Skipped) > maxSkipped {
		l.Skipped = l.Skipped[len(l.Skipped)-maxSkipped:]
	}
	return l.save()
}

// contentHash hash isi yang dinormalisasi (huruf kecil, spasi dirapatkan) untuk mendeteksi duplikat.
func contentHash(title, content string) string {
	n := strings.ToLower(strings.Join(strings.Fields(title+"\n"+content), " "))
	sum := sha256.Sum256([]byte(n))
	return hex.EncodeToString(sum[:])
}
