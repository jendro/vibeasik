package knowledgesync

import (
	"errors"
	"fmt"
)

// Hook untuk pengujian.
var (
	revokeRequest = func(tok string, serverID uint) error {
		return post(fmt.Sprintf("/api/v1/knowledge/contributions/%d/revoke", serverID), tok, map[string]any{}, nil)
	}
	listRequest = func(out any) error { return get("/api/v1/knowledge/contributions", out) }
)

// updateLedgerSent mengubah entri terkirim yang cocok dengan serverID.
func updateLedgerSent(serverID uint, status string) bool {
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	l := LoadLedger()
	found := false
	for i := range l.Sent {
		if l.Sent[i].ServerID == serverID {
			l.Sent[i].Status = status
			found = true
		}
	}
	if found {
		_ = l.save()
	}
	return found
}

// Revoke meminta server menghapus kiriman yang belum terbit. serverID adalah angka "server#N" di ledger.
func Revoke(serverID uint) error {
	tok := deviceToken()
	if tok == "" {
		return errors.New("belum login: pencabutan hanya dari perangkat yang masuk akun")
	}
	if err := revokeRequest(tok, serverID); err != nil {
		return err
	}
	updateLedgerSent(serverID, "revoked")
	return nil
}

// SyncLedger memperbarui status kiriman di ledger dari server. Mengembalikan jumlah entri yang diperbarui.
func SyncLedger() (int, error) {
	if deviceToken() == "" {
		return 0, errors.New("belum login: sinkron riwayat hanya dari perangkat yang masuk akun")
	}
	var resp struct {
		Contributions []struct {
			ID     uint   `json:"id"`
			Status string `json:"status"`
		} `json:"contributions"`
	}
	if err := listRequest(&resp); err != nil {
		return 0, err
	}
	n := 0
	for _, c := range resp.Contributions {
		if updateLedgerSent(c.ID, c.Status) {
			n++
		}
	}
	return n, nil
}
