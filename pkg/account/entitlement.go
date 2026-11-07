// Package account menyambungkan engine ke akun vibeasik: login perangkat, entitlement bertanda tangan,
// dan pembukaan fitur sesuai paket (registry.SetEntitlement).
//
// Entitlement diterbitkan server vibeasik dan ditandatangani Ed25519. Engine hanya memercayai entitlement
// yang lolos verifikasi dengan PublicKey di bawah — mengubah berkas account.json tidak membuka fitur apa pun.
package account

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
)

// PublicKey kunci publik penandatangan entitlement (base64). Bisa diganti saat build:
// -ldflags "-X github.com/vibeasik/vibeasik/pkg/account.PublicKey=<base64>" — tidak lewat env.
var PublicKey = "7U4G01XmA3HtIXaE5HOEoPNLCp4/8l59i/2FRysDAKk="

// Claims isi entitlement (format sama dengan server vibeasik-cloud).
type Claims struct {
	V         int      `json:"v"`
	Sub       string   `json:"sub"`
	Email     string   `json:"email"`
	Device    string   `json:"device"`
	Plan      string   `json:"plan"`
	Features  []string `json:"features"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
	GraceDays int      `json:"grace_days"`
}

// ErrBadSignature entitlement tidak ditandatangani server vibeasik.
var ErrBadSignature = errors.New("tanda tangan entitlement tidak valid")

// Verify memeriksa tanda tangan token "payload.signature" dan mengurai klaimnya.
func Verify(token string) (*Claims, error) {
	pub, err := base64.StdEncoding.DecodeString(PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("kunci publik entitlement di build ini rusak")
	}
	parts := strings.SplitN(strings.TrimSpace(token), ".", 2)
	if len(parts) != 2 {
		return nil, ErrBadSignature
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !ed25519.Verify(ed25519.PublicKey(pub), []byte(parts[0]), sig) {
		return nil, ErrBadSignature
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrBadSignature
	}
	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, ErrBadSignature
	}
	return &c, nil
}

// UsableUntil batas akhir entitlement dipakai: kedaluwarsa + masa tenggang offline.
func (c *Claims) UsableUntil() time.Time {
	return time.Unix(c.ExpiresAt, 0).Add(time.Duration(c.GraceDays) * 24 * time.Hour)
}

// Usable melaporkan entitlement masih boleh dipakai pada waktu now (termasuk masa tenggang).
// Jam sistem yang mundur sebelum waktu terbit dianggap tidak sah.
func (c *Claims) Usable(now time.Time) bool {
	if c == nil {
		return false
	}
	if now.Before(time.Unix(c.IssuedAt, 0).Add(-10 * time.Minute)) {
		return false
	}
	return now.Before(c.UsableUntil())
}

// Has melaporkan fitur ada di entitlement yang masih berlaku.
func (c *Claims) Has(feature string, now time.Time) bool {
	return c.Usable(now) && slices.Contains(c.Features, feature)
}
