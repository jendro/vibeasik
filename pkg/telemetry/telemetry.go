// Package telemetry mengirim ping pemakaian ringan ke server vibeasik (enjoy.vibeasik.com): ID instalasi acak,
// OS, arsitektur, versi, edisi, zona waktu, dan status akun. TIDAK mengirim kode, nama proyek, path, hostname,
// nama pengguna, atau isi apa pun dari proyek. Server juga melihat IP dan negara pada setiap permintaan.
// Matikan dengan VIBEASIK_NO_TELEMETRY=1 (atau DO_NOT_TRACK=1). Rincian: vibeasik.com/terms.
package telemetry

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vibeasik/vibeasik/pkg/localauth"
)

// Info bagian yang diisi pemanggil (hanya data yang tidak mengandung isi proyek).
type Info struct {
	Version       string
	Edition       string
	DBDriver      string
	LoggedIn      bool
	AccountChoice string
	Projects      int
}

// Disabled true bila pemakai mematikan pengiriman.
func Disabled() bool {
	for _, k := range []string{"VIBEASIK_NO_TELEMETRY", "DO_NOT_TRACK"} {
		if v := strings.ToLower(strings.TrimSpace(os.Getenv(k))); v == "1" || v == "true" || v == "yes" {
			return true
		}
	}
	return false
}

func idPath() string { return filepath.Join(localauth.Dir(), "install_id") }

// InstallID mengembalikan ID instalasi acak (dibuat sekali, tidak terkait identitas apa pun).
func InstallID() string {
	if raw, err := os.ReadFile(idPath()); err == nil {
		if id := strings.TrimSpace(string(raw)); len(id) == 36 {
			return id
		}
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	id := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	_ = os.MkdirAll(localauth.Dir(), 0o700)
	_ = os.WriteFile(idPath(), []byte(id+"\n"), 0o600)
	return id
}

type payload struct {
	InstallID     string `json:"install_id"`
	Event         string `json:"event"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	OSVersion     string `json:"os_version"`
	Version       string `json:"version"`
	Edition       string `json:"edition"`
	Locale        string `json:"locale"`
	Timezone      string `json:"timezone"`
	DBDriver      string `json:"db_driver"`
	LoggedIn      bool   `json:"logged_in"`
	AccountChoice string `json:"account_choice"`
	Projects      int    `json:"projects"`
}

func osVersion() string {
	if runtime.GOOS == "linux" {
		if raw, err := os.ReadFile("/etc/os-release"); err == nil {
			for _, ln := range strings.Split(string(raw), "\n") {
				if strings.HasPrefix(ln, "PRETTY_NAME=") {
					return strings.Trim(strings.TrimPrefix(ln, "PRETTY_NAME="), `"`)
				}
			}
		}
	}
	return ""
}

func timezone() string {
	if tz := os.Getenv("TZ"); tz != "" {
		return strings.TrimPrefix(tz, ":")
	}
	if raw, err := os.ReadFile("/etc/timezone"); err == nil {
		return strings.TrimSpace(string(raw))
	}
	if p, err := os.Readlink("/etc/localtime"); err == nil {
		if i := strings.Index(p, "zoneinfo/"); i >= 0 {
			return p[i+len("zoneinfo/"):]
		}
	}
	return time.Now().Location().String()
}

func locale() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			if i := strings.IndexAny(v, ".@"); i > 0 {
				v = v[:i]
			}
			return v
		}
	}
	return ""
}

// Send mengirim satu ping; gagal jaringan diabaikan.
func Send(cloudURL, event string, in Info) {
	if Disabled() || cloudURL == "" {
		return
	}
	p := payload{
		InstallID: InstallID(), Event: event, OS: runtime.GOOS, Arch: runtime.GOARCH, OSVersion: osVersion(),
		Version: in.Version, Edition: in.Edition, Locale: locale(), Timezone: timezone(), DBDriver: in.DBDriver,
		LoggedIn: in.LoggedIn, AccountChoice: in.AccountChoice, Projects: in.Projects,
	}
	raw, _ := json.Marshal(p)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(cloudURL, "/")+"/api/v1/telemetry/ping", bytes.NewReader(raw))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "vibeasik/"+in.Version+" ("+runtime.GOOS+"/"+runtime.GOARCH+")")
	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

// HeartbeatEvery jeda heartbeat; panel admin menganggap komputer online bila terlihat dalam 15 menit.
const HeartbeatEvery = 5 * time.Minute

// Start mengirim "start" sebentar setelah engine berjalan, lalu "heartbeat" tiap 5 menit.
func Start(ctx context.Context, cloudURL string, info func() Info) {
	if Disabled() {
		return
	}
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
		Send(cloudURL, "start", info())
		t := time.NewTicker(HeartbeatEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				Send(cloudURL, "heartbeat", info())
			}
		}
	}()
}
