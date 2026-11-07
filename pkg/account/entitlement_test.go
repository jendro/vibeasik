package account

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func signWith(t *testing.T, priv ed25519.PrivateKey, c Claims) string {
	t.Helper()
	raw, _ := json.Marshal(c)
	p := base64.RawURLEncoding.EncodeToString(raw)
	return p + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, []byte(p)))
}

func TestVerifyAndUsable(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	old := PublicKey
	PublicKey = base64.StdEncoding.EncodeToString(pub)
	defer func() { PublicKey = old }()

	now := time.Now()
	c := Claims{V: 1, Plan: "pro", Features: []string{"staff"}, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix(), GraceDays: 14}
	tok := signWith(t, priv, c)

	got, err := Verify(tok)
	if err != nil || !got.Has("staff", now) || got.Has("orchestrator", now) {
		t.Fatalf("verify: %v %+v", err, got)
	}
	// Masa tenggang offline: lewat exp tapi < 14 hari tetap berlaku; lewat itu tidak.
	if !got.Usable(now.Add(10*24*time.Hour)) || got.Usable(now.Add(16*24*time.Hour)) {
		t.Fatal("grace period wrong")
	}
	// Jam dimundurkan jauh sebelum waktu terbit → tidak sah.
	if got.Usable(now.Add(-48 * time.Hour)) {
		t.Fatal("clock rollback must not be usable")
	}

	// Payload diubah (plan team) dengan tanda tangan lama → ditolak.
	c.Plan, c.Features = "team", []string{"staff", "orchestrator"}
	forged := signWith(t, priv, c)
	forged = forged[:len(forged)-10] + tok[len(tok)-10:]
	if _, err := Verify(forged); err == nil {
		t.Fatal("forged entitlement accepted")
	}
	// Ditandatangani kunci lain → ditolak.
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := Verify(signWith(t, other, c)); err == nil {
		t.Fatal("foreign key accepted")
	}
	var nilClaims *Claims
	if nilClaims.Has("staff", now) {
		t.Fatal("nil claims must not grant features")
	}
}
