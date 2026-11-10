package geo

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	downloadBase = "https://download.db-ip.com/free/"
	// Batas ukuran hasil unduhan setelah dibuka (City Lite ±130 MB, ASN Lite ±10 MB).
	maxCityBytes = 512 << 20
	maxASNBytes  = 64 << 20
	httpTimeout  = 30 * time.Minute // satu unduhan penuh; koneksi macet tetap terputus oleh konteks
	// AutoEnv = "0"/"false"/"off" mematikan unduhan otomatis bulanan dari `serve`.
	AutoEnv = "VIBEASIK_GEO_AUTO"
	// StaleAfter = umur berkas sebelum dianggap perlu diperbarui.
	StaleAfter = 30 * 24 * time.Hour
)

// Source = satu basis data yang diunduh.
type Source struct {
	File     string // nama berkas tujuan
	Prefix   string // awalan nama berkas DB-IP, mis. dbip-city-lite
	MaxBytes int64
}

// Sources = daftar basis data; ASN lebih dulu karena kecil dan paling berguna.
var Sources = []Source{
	{File: ASNFile, Prefix: "dbip-asn-lite", MaxBytes: maxASNBytes},
	{File: CityFile, Prefix: "dbip-city-lite", MaxBytes: maxCityBytes},
}

// Updater mengunduh basis data. Field kosong memakai nilai bawaan; BaseURL/Now dapat diganti untuk uji.
type Updater struct {
	Dir     string
	BaseURL string
	Client  *http.Client
	Now     func() time.Time
	Log     func(format string, args ...any)
}

func (u *Updater) client() *http.Client {
	if u.Client != nil {
		return u.Client
	}
	return &http.Client{Timeout: httpTimeout}
}

func (u *Updater) now() time.Time {
	if u.Now != nil {
		return u.Now()
	}
	return time.Now()
}

func (u *Updater) logf(format string, args ...any) {
	if u.Log != nil {
		u.Log(format, args...)
	}
}

func (u *Updater) base() string {
	if u.BaseURL != "" {
		return strings.TrimRight(u.BaseURL, "/") + "/"
	}
	return downloadBase
}

// months = bulan berjalan lalu bulan sebelumnya (DB-IP baru terbit beberapa hari setelah awal bulan).
func months(now time.Time) []string {
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return []string{first.Format("2006-01"), first.AddDate(0, -1, 0).Format("2006-01")}
}

var errNotFound = errors.New("belum terbit")

// Update mengunduh semua sumber (ASN, lalu City). Kegagalan satu sumber tidak menghentikan yang lain;
// galat digabung. Setelah berhasil, onSwap dipanggil sebelum berkas diganti (lepas pemetaan lama).
func (u *Updater) Update(ctx context.Context, only string, onSwap func()) error {
	if err := os.MkdirAll(u.Dir, 0o755); err != nil {
		return err
	}
	var errs []error
	for _, src := range Sources {
		if only != "" && only != "all" && !strings.HasPrefix(src.Prefix, "dbip-"+only) {
			continue
		}
		if err := u.fetch(ctx, src, onSwap); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", src.File, err))
		}
	}
	return errors.Join(errs...)
}

func (u *Updater) fetch(ctx context.Context, src Source, onSwap func()) error {
	var lastErr error
	for _, month := range months(u.now()) {
		url := fmt.Sprintf("%s%s-%s.mmdb.gz", u.base(), src.Prefix, month)
		u.logf("mengunduh %s", url)
		tmp, err := u.download(ctx, url, src)
		if errors.Is(err, errNotFound) {
			lastErr = fmt.Errorf("%s: %w", month, err)
			continue
		}
		if err != nil {
			return err
		}
		if onSwap != nil {
			onSwap()
		}
		dst := filepath.Join(u.Dir, src.File)
		if err := os.Rename(tmp, dst); err != nil {
			_ = os.Remove(tmp)
			return err
		}
		u.logf("terpasang %s", dst)
		return nil
	}
	return lastErr
}

// download menulis hasil gunzip ke berkas sementara di folder tujuan (agar rename atomik) lalu mengembalikan path-nya.
func (u *Updater) download(ctx context.Context, url string, src Source) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := u.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	zr, err := gzip.NewReader(io.LimitReader(resp.Body, src.MaxBytes))
	if err != nil {
		return "", err
	}
	defer zr.Close()
	f, err := os.CreateTemp(u.Dir, src.File+".*.tmp")
	if err != nil {
		return "", err
	}
	n, err := io.Copy(f, io.LimitReader(zr, src.MaxBytes+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > src.MaxBytes {
		err = fmt.Errorf("ukuran melebihi batas %d MB", src.MaxBytes>>20)
	}
	if err == nil && n == 0 {
		err = errors.New("berkas kosong")
	}
	if err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// Stale true bila salah satu berkas tidak ada atau lebih tua dari StaleAfter.
func Stale(dir string, now time.Time) bool {
	for _, s := range Sources {
		fi, err := os.Stat(filepath.Join(dir, s.File))
		if err != nil || now.Sub(fi.ModTime()) > StaleAfter {
			return true
		}
	}
	return false
}

// AutoEnabled false bila VIBEASIK_GEO_AUTO dimatikan.
func AutoEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(AutoEnv))) {
	case "0", "false", "off", "no":
		return false
	}
	return true
}

// StartAutoUpdate memeriksa tiap hari dan mengunduh bila berkas belum ada atau berumur >30 hari.
// Hanya dipanggil dari `serve`; tidak pernah dari uji. Gagal jaringan hanya dicatat di log.
func StartAutoUpdate(ctx context.Context, db *DB) {
	if !AutoEnabled() {
		return
	}
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Minute): // jangan berebut dengan startup
		}
		for {
			if Stale(db.dir, time.Now()) {
				u := &Updater{Dir: db.dir, Log: func(f string, a ...any) { log.Printf("[geo] "+f, a...) }}
				if err := u.Update(ctx, "all", db.Release); err != nil {
					log.Printf("[geo] pembaruan basis data lokasi gagal: %v", err)
				}
				db.Reload()
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(24 * time.Hour):
			}
		}
	}()
}
