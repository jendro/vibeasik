// Package localauth mengelola token akses lokal engine vibeasik.
//
// Engine menjalankan perintah, agent, dan membaca kode, sehingga setiap pemanggil API/MCP/Studio wajib
// membawa token ini — termasuk dari localhost, karena website mana pun yang dibuka di browser bisa
// mengirim request ke localhost. Token disimpan di ~/.vibeasik/token (izin 0600) dan hanya bisa dibaca
// pengguna laptop ini; proses lokal (CLI, hook, MCP stdio) membacanya dari berkas tersebut.
package localauth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// Header membawa token pada request API/MCP.
	Header = "X-Vibeasik-Token"
	// Cookie membawa token untuk Studio (HttpOnly, SameSite=Strict).
	Cookie = "vibeasik_token"
	// EnvToken memakai token tertentu alih-alih berkas (mis. engine jarak jauh atau CI).
	EnvToken = "VIBEASIK_TOKEN"
	fileName = "token"
)

var (
	mu     sync.Mutex
	cached string
)

// Dir folder data vibeasik per pengguna (~/.vibeasik).
func Dir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".vibeasik")
	}
	return ".vibeasik"
}

// Path lokasi berkas token.
func Path() string { return filepath.Join(Dir(), fileName) }

// Token mengembalikan token engine; dibuat (acak 256 bit) bila belum ada. Dipakai engine saat serve.
func Token() (string, error) {
	mu.Lock()
	defer mu.Unlock()
	if v := strings.TrimSpace(os.Getenv(EnvToken)); v != "" {
		return v, nil
	}
	if cached != "" {
		return cached, nil
	}
	if b, err := os.ReadFile(Path()); err == nil {
		if t := strings.TrimSpace(string(b)); t != "" {
			cached = t
			return t, nil
		}
	}
	t, err := newToken()
	if err != nil {
		return "", err
	}
	if err := write(t); err != nil {
		return "", err
	}
	cached = t
	return t, nil
}

// Rotate membuat token baru; semua sesi Studio & konfigurasi MCP HTTP lama tidak berlaku lagi.
func Rotate() (string, error) {
	if os.Getenv(EnvToken) != "" {
		return "", errors.New(EnvToken + " diset: ganti nilainya langsung")
	}
	t, err := newToken()
	if err != nil {
		return "", err
	}
	mu.Lock()
	defer mu.Unlock()
	if err := write(t); err != nil {
		return "", err
	}
	cached = t
	return t, nil
}

// ClientToken token untuk klien lokal (CLI, hook, MCP stdio); "" bila belum ada. Tidak membuat berkas.
func ClientToken() string {
	if v := strings.TrimSpace(os.Getenv(EnvToken)); v != "" {
		return v
	}
	b, err := os.ReadFile(Path())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Apply memasang header token pada request ke engine lokal.
func Apply(req *http.Request) {
	if t := ClientToken(); t != "" {
		req.Header.Set(Header, t)
	}
}

// Equal membandingkan token dalam waktu konstan.
func Equal(got, want string) bool {
	return got != "" && want != "" && subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func write(t string) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	tmp := Path() + ".tmp"
	if err := os.WriteFile(tmp, []byte(t+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, Path())
}
