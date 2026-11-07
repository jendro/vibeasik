package account

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/registry"
)

// State login perangkat yang disimpan di ~/.vibeasik/account.json (izin 0600).
type State struct {
	CloudURL    string    `json:"cloud_url"`
	DeviceID    string    `json:"device_id"`
	DeviceToken string    `json:"device_token"`
	Email       string    `json:"email"`
	Entitlement string    `json:"entitlement"`
	FetchedAt   time.Time `json:"fetched_at"`
}

// Status ringkasan untuk CLI & Studio.
type Status struct {
	LoggedIn     bool       `json:"logged_in"`
	Email        string     `json:"email,omitempty"`
	Plan         string     `json:"plan"`
	Features     []string   `json:"features"`
	DeviceID     string     `json:"device_id,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	UsableUntil  *time.Time `json:"usable_until,omitempty"`
	Offline      bool       `json:"offline"`
	CloudURL     string     `json:"cloud_url"`
	LastRefresh  *time.Time `json:"last_refresh,omitempty"`
	RefreshError string     `json:"refresh_error,omitempty"`
}

var (
	mu         sync.RWMutex
	state      *State
	claims     *Claims
	refreshErr string
	httpClient = &http.Client{Timeout: 15 * time.Second}
)

func statePath() string { return filepath.Join(localauth.Dir(), "account.json") }

// CloudURL alamat server vibeasik (env VIBEASIK_CLOUD_URL).
func CloudURL() string { return strings.TrimRight(config.GetConfig().VibeasikCloudURL, "/") }

// Activate memuat login tersimpan dan memasang pemeriksa fitur ke registry. Aman dipanggil berkali-kali.
func Activate() {
	mu.Lock()
	defer mu.Unlock()
	state, claims = nil, nil
	if raw, err := os.ReadFile(statePath()); err == nil {
		var s State
		if json.Unmarshal(raw, &s) == nil && s.DeviceToken != "" {
			state = &s
			if c, err := Verify(s.Entitlement); err == nil {
				claims = c
			} else {
				log.Printf("[account] entitlement tersimpan ditolak: %v", err)
			}
		}
	}
	registry.SetEntitlement(hasFeature)
}

// DevUnlock membuka semua fitur tanpa akun. Hanya diset build pengembangan Official (ldflags), tidak lewat env.
var DevUnlock bool

func hasFeature(feature string) bool {
	if feature == "" || DevUnlock {
		return true
	}
	mu.RLock()
	c := claims
	mu.RUnlock()
	return c.Has(feature, time.Now())
}

// Current status login sekarang.
func Current() Status {
	mu.RLock()
	defer mu.RUnlock()
	st := Status{Plan: "community", Features: []string{}, CloudURL: CloudURL(), RefreshError: refreshErr}
	if state == nil {
		return st
	}
	st.LoggedIn, st.Email, st.DeviceID = true, state.Email, state.DeviceID
	if !state.FetchedAt.IsZero() {
		t := state.FetchedAt
		st.LastRefresh = &t
	}
	if claims != nil {
		exp, until := time.Unix(claims.ExpiresAt, 0), claims.UsableUntil()
		st.ExpiresAt, st.UsableUntil = &exp, &until
		st.Offline = time.Now().After(exp)
		if claims.Usable(time.Now()) {
			st.Plan, st.Features = claims.Plan, claims.Features
		} else {
			st.Plan = "free"
		}
	}
	return st
}

func save(s *State) error {
	if err := os.MkdirAll(localauth.Dir(), 0o700); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(s, "", "  ")
	tmp := statePath() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, statePath())
}

// DeviceCode jawaban server saat memulai login perangkat.
type DeviceCode struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// StartLogin meminta kode login perangkat ke server.
func StartLogin(deviceName string) (*DeviceCode, error) {
	var dc DeviceCode
	if err := cloudJSON(http.MethodPost, "/api/v1/auth/device/code", "", map[string]string{"device_name": deviceName}, &dc); err != nil {
		return nil, err
	}
	return &dc, nil
}

// ErrPending login belum disetujui di halaman /device.
var ErrPending = errors.New("menunggu persetujuan")

// PollLogin menukar device code dengan token perangkat bila sudah disetujui, lalu menyimpan & mengaktifkan login.
func PollLogin(deviceCode string) (*Status, error) {
	var out struct {
		Error       string `json:"error"`
		DeviceToken string `json:"device_token"`
		DeviceID    string `json:"device_id"`
		Email       string `json:"email"`
		Entitlement string `json:"entitlement"`
	}
	err := cloudJSON(http.MethodPost, "/api/v1/auth/device/token", "", map[string]string{"device_code": deviceCode}, &out)
	if out.Error == "authorization_pending" {
		return nil, ErrPending
	}
	if out.Error != "" {
		return nil, errors.New("login kedaluwarsa atau ditolak; mulai ulang")
	}
	if err != nil {
		return nil, err
	}
	if _, err := Verify(out.Entitlement); err != nil {
		return nil, fmt.Errorf("server %s bukan server vibeasik resmi untuk build ini: %w", CloudURL(), err)
	}
	s := &State{CloudURL: CloudURL(), DeviceID: out.DeviceID, DeviceToken: out.DeviceToken, Email: out.Email, Entitlement: out.Entitlement, FetchedAt: time.Now()}
	if err := save(s); err != nil {
		return nil, err
	}
	Activate()
	st := Current()
	return &st, nil
}

// Refresh mengambil entitlement terbaru (paket bisa berubah). Gagal jaringan tidak menghapus entitlement lama.
func Refresh() error {
	mu.RLock()
	s := state
	mu.RUnlock()
	if s == nil {
		return errors.New("belum login")
	}
	var out struct {
		Entitlement string `json:"entitlement"`
		Error       string `json:"error"`
	}
	err := cloudJSON(http.MethodGet, "/api/v1/entitlement", s.DeviceToken, nil, &out)
	if out.Error == "unauthorized" {
		// Perangkat dicabut dari akun: hapus login lokal.
		_ = os.Remove(statePath())
		Activate()
		return errors.New("perangkat ini sudah dicabut dari akun; login ulang")
	}
	if err != nil {
		setRefreshErr(err.Error())
		return err
	}
	if _, err := Verify(out.Entitlement); err != nil {
		setRefreshErr(err.Error())
		return err
	}
	ns := *s
	ns.Entitlement, ns.FetchedAt = out.Entitlement, time.Now()
	if err := save(&ns); err != nil {
		return err
	}
	setRefreshErr("")
	Activate()
	return nil
}

func setRefreshErr(e string) {
	mu.Lock()
	refreshErr = e
	mu.Unlock()
}

// Logout mencabut perangkat di server (bila bisa) dan menghapus login lokal.
func Logout() error {
	mu.RLock()
	s := state
	mu.RUnlock()
	if s != nil {
		_ = cloudJSON(http.MethodPost, "/api/v1/auth/device/revoke", s.DeviceToken, nil, nil)
	}
	if err := os.Remove(statePath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	Activate()
	return nil
}

// StartRefresher memperbarui entitlement saat start lalu tiap 6 jam selama engine berjalan.
func StartRefresher(ctx context.Context) {
	go func() {
		t := time.NewTicker(6 * time.Hour)
		defer t.Stop()
		for {
			if Current().LoggedIn {
				if err := Refresh(); err != nil {
					log.Printf("[account] gagal memperbarui entitlement (dipakai yang tersimpan): %v", err)
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

// DeviceToken token perangkat untuk layanan cloud lain (mis. Pusat Pengetahuan); "" bila belum login.
func DeviceToken() string {
	mu.RLock()
	defer mu.RUnlock()
	if state == nil {
		return ""
	}
	return state.DeviceToken
}

func cloudJSON(method, path, bearer string, body, out any) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, CloudURL()+path, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("server vibeasik (%s) tidak bisa dihubungi: %w", CloudURL(), err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if out != nil {
		_ = json.Unmarshal(raw, out)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server vibeasik menjawab %d", resp.StatusCode)
	}
	return nil
}
