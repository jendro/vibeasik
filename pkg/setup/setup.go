// Package setup menyimpan keadaan penyiapan awal vibeasik di komputer pengguna:
// pilihan akun, persetujuan Syarat & Ketentuan, dan pilihan database. Studio mengarahkan
// pengguna ke wizard sampai penyiapan selesai.
package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vibeasik/vibeasik/pkg/localauth"
)

// State isi ~/.vibeasik/setup.json.
type State struct {
	TermsVersion string    `json:"terms_version,omitempty"`
	TermsAt      time.Time `json:"terms_at,omitempty"`
	// AccountChoice: register | login | skip. Hanya catatan; akun tidak diwajibkan.
	AccountChoice string    `json:"account_choice,omitempty"`
	DatabaseAt    time.Time `json:"database_at,omitempty"`
	CompletedAt   time.Time `json:"completed_at,omitempty"`
}

var mu sync.Mutex

func path() string { return filepath.Join(localauth.Dir(), "setup.json") }

// Load membaca keadaan penyiapan (kosong bila belum pernah).
func Load() State {
	mu.Lock()
	defer mu.Unlock()
	return load()
}

func load() State {
	var s State
	if raw, err := os.ReadFile(path()); err == nil {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

func save(s State) error {
	if err := os.MkdirAll(localauth.Dir(), 0o700); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(path(), raw, 0o600)
}

// Complete true bila penyiapan awal pernah diselesaikan. Penyiapan hanya sekali per komputer:
// versi ketentuan yang lebih baru tidak membuka wizard (dan pilihan database) lagi, dan persetujuan
// awal tetap berlaku untuk Pusat Pengetahuan (lihat knowledgesync.HasConsent).
func Complete() bool {
	return !Load().CompletedAt.IsZero()
}

// TermsAccepted true bila versi ketentuan ini sudah disetujui.
func TermsAccepted(termsVersion string) bool {
	s := Load()
	return s.TermsVersion == termsVersion && !s.TermsAt.IsZero()
}

// AcceptTerms mencatat persetujuan versi ketentuan.
func AcceptTerms(version string) error {
	mu.Lock()
	defer mu.Unlock()
	s := load()
	s.TermsVersion = version
	s.TermsAt = time.Now()
	return save(s)
}

// SetAccountChoice mencatat pilihan akun (register, login, atau skip).
func SetAccountChoice(choice string) error {
	switch choice {
	case "register", "login", "skip":
	default:
		return errors.New("pilihan akun tidak dikenal")
	}
	mu.Lock()
	defer mu.Unlock()
	s := load()
	s.AccountChoice = choice
	return save(s)
}

// MarkDatabase mencatat bahwa pilihan database sudah ditetapkan.
func MarkDatabase() error {
	mu.Lock()
	defer mu.Unlock()
	s := load()
	s.DatabaseAt = time.Now()
	return save(s)
}

// Finish menandai penyiapan selesai. Persetujuan ketentuan dan pilihan database wajib sudah ada.
func Finish(_ string) error {
	mu.Lock()
	defer mu.Unlock()
	s := load()
	// Versi yang disetujui boleh lebih lama (engine diperbarui di tengah wizard); persetujuan tetap berlaku.
	if s.TermsVersion == "" || s.TermsAt.IsZero() {
		return errors.New("Syarat & Ketentuan belum disetujui")
	}
	if s.DatabaseAt.IsZero() {
		return errors.New("database belum dipilih")
	}
	s.CompletedAt = time.Now()
	return save(s)
}

// ---- Database ----

// DB pilihan database dari wizard.
type DB struct {
	Driver   string `json:"driver"` // sqlite | mysql
	Name     string `json:"name"`
	Host     string `json:"host,omitempty"`
	Port     string `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	// CreateIfMissing: MySQL/MariaDB, buat database bila belum ada.
	CreateIfMissing bool `json:"create_if_missing,omitempty"`
}

var nameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{1,39}$`)

// Validate memeriksa dan merapikan isian.
func (d *DB) Validate() error {
	d.Driver = strings.ToLower(strings.TrimSpace(d.Driver))
	d.Name = strings.TrimSpace(d.Name)
	if d.Driver != "sqlite" && d.Driver != "mysql" {
		return errors.New("jenis database harus sqlite atau mysql")
	}
	if !nameRe.MatchString(d.Name) {
		return errors.New("nama database 2-40 karakter: huruf, angka, garis bawah; diawali huruf")
	}
	if d.Driver == "mysql" {
		d.Host = strings.TrimSpace(d.Host)
		d.Port = strings.TrimSpace(d.Port)
		d.User = strings.TrimSpace(d.User)
		if d.Host == "" {
			d.Host = "127.0.0.1"
		}
		if d.Port == "" {
			d.Port = "3306"
		}
		if d.User == "" {
			return errors.New("user MySQL wajib diisi")
		}
		if strings.ContainsAny(d.Host+d.Port+d.User, " \t\r\n'\"`@/") {
			return errors.New("host, port, atau user mengandung karakter yang tidak diizinkan")
		}
		if strings.ContainsAny(d.Password, "'\r\n") {
			return errors.New("kata sandi tidak boleh memuat tanda petik tunggal atau baris baru")
		}
	}
	return nil
}

// SQLitePath lokasi berkas SQLite untuk nama ini.
func SQLitePath(name string) string { return filepath.Join(localauth.Dir(), name+".db") }

// EnvPath berkas konfigurasi yang dibaca engine saat start.
func EnvPath() string { return filepath.Join(localauth.Dir(), ".env") }

// WriteEnv menulis pilihan database ke ~/.vibeasik/.env, menjaga baris lain.
func WriteEnv(d DB) error {
	kv := map[string]string{
		"DB_DRIVER": d.Driver,
		"DB_NAME":   d.Name,
	}
	if d.Driver == "sqlite" {
		kv["DB_PATH"] = SQLitePath(d.Name)
	} else {
		kv["DB_HOST"] = d.Host
		kv["DB_PORT"] = d.Port
		kv["DB_USER"] = d.User
		kv["DB_PASSWORD"] = d.Password
	}
	if err := os.MkdirAll(localauth.Dir(), 0o700); err != nil {
		return err
	}
	var lines []string
	if raw, err := os.ReadFile(EnvPath()); err == nil {
		lines = strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	}
	done := map[string]bool{}
	for i, ln := range lines {
		t := strings.TrimSpace(ln)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		key := strings.TrimSpace(strings.SplitN(strings.TrimPrefix(t, "export "), "=", 2)[0])
		if v, ok := kv[key]; ok {
			lines[i] = fmt.Sprintf("%s='%s'", key, v)
			done[key] = true
		}
	}
	for _, key := range []string{"DB_DRIVER", "DB_NAME", "DB_PATH", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD"} {
		if v, ok := kv[key]; ok && !done[key] {
			lines = append(lines, fmt.Sprintf("%s='%s'", key, v))
		}
	}
	return os.WriteFile(EnvPath(), []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}
