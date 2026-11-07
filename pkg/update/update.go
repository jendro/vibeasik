// Package update memeriksa versi terbaru vibeasik dan memasangnya di tempat: unduh, verifikasi
// sha256, lalu ganti biner yang sedang berjalan. Hanya build Official yang diperbarui otomatis
// (sumbernya vibeasik.com/dl, sama dengan installer); build Community hanya diberi tahu.
package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultBase sumber unduhan Official; dapat ditimpa VIBEASIK_UPDATE_URL.
const DefaultBase = "https://vibeasik.com/dl"

// ReleasesURL halaman rilis untuk build Community.
const ReleasesURL = "https://github.com/vibeasik/vibeasik/releases/latest"

const maxDownload = 200 << 20

// Info hasil pemeriksaan.
type Info struct {
	Current   string    `json:"current"`
	Latest    string    `json:"latest"`
	Available bool      `json:"available"`
	CanApply  bool      `json:"can_apply"` // bisa dipasang otomatis dari aplikasi
	Reason    string    `json:"reason,omitempty"`
	URL       string    `json:"url,omitempty"` // halaman unduh manual
	Staged    string    `json:"staged,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
	Error     string    `json:"error,omitempty"`
}

// Updater memeriksa dan memasang pembaruan. Nol-nilai tidak dipakai: gunakan New.
type Updater struct {
	Current  string // versi berjalan, mis. v0.3.2
	Edition  string // Official | Community
	Base     string
	Client   *http.Client
	ExePath  string // biner yang diganti; bawaan os.Executable()
	mu       sync.Mutex
	last     Info
	staged   string
	checking bool
}

// New membuat updater untuk versi dan edisi berjalan.
func New(current, edition string) *Updater {
	base := strings.TrimRight(os.Getenv("VIBEASIK_UPDATE_URL"), "/")
	if base == "" {
		base = DefaultBase
	}
	return &Updater{Current: current, Edition: edition, Base: base, Client: &http.Client{Timeout: 60 * time.Second}}
}

// Disabled alasan pembaruan tidak berlaku (kosong bila berlaku).
func (u *Updater) disabledReason() string {
	switch {
	case os.Getenv("VIBEASIK_NO_UPDATE") != "":
		return "dimatikan lewat VIBEASIK_NO_UPDATE"
	case !validVersion(u.Current):
		return "build pengembangan"
	case strings.Contains(u.Current, "-"):
		return "build pengembangan"
	}
	return ""
}

// Check mengambil versi terbaru dari server (hasil disimpan sebagai cache).
func (u *Updater) Check(ctx context.Context) Info {
	info := Info{Current: u.Current, URL: u.downloadPage(), CheckedAt: time.Now()}
	if r := u.disabledReason(); r != "" {
		info.Reason = r
		return u.store(info)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.Base+"/latest", nil)
	resp, err := u.Client.Do(req)
	if err != nil {
		info.Error = "server pembaruan tidak terjangkau"
		return u.store(info)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	latest := strings.TrimSpace(string(raw))
	if resp.StatusCode != 200 || !validVersion(latest) {
		info.Error = "versi terbaru tidak terbaca"
		return u.store(info)
	}
	info.Latest = latest
	info.Available = newer(latest, u.Current)
	switch {
	case !info.Available:
	case u.Edition != "Official":
		info.Reason = "build Community: unduh versi baru dari halaman rilis"
	default:
		info.CanApply = u.writable()
	}
	if info.Available && info.Reason == "" && !info.CanApply {
		info.Reason = "biner tidak bisa ditulis; jalankan ulang installer"
	}
	return u.store(info)
}

func (u *Updater) store(i Info) Info {
	u.mu.Lock()
	defer u.mu.Unlock()
	i.Staged = u.staged
	u.last = i
	return i
}

// Last hasil pemeriksaan terakhir (kosong bila belum pernah).
func (u *Updater) Last() Info {
	u.mu.Lock()
	defer u.mu.Unlock()
	i := u.last
	i.Staged = u.staged
	if i.Current == "" {
		i.Current = u.Current
	}
	return i
}

func (u *Updater) downloadPage() string {
	if u.Edition == "Official" {
		return "https://vibeasik.com/#unduh"
	}
	return ReleasesURL
}

func (u *Updater) exe() (string, error) {
	if u.ExePath != "" {
		return u.ExePath, nil
	}
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	p = strings.TrimSuffix(p, " (deleted)") // biner sudah diganti oleh pembaruan sebelumnya
	return filepath.EvalSymlinks(p)
}

func (u *Updater) writable() bool {
	p, err := u.exe()
	if err != nil {
		return false
	}
	f, err := os.CreateTemp(filepath.Dir(p), ".vibeasik-uji-*")
	if err != nil {
		return false
	}
	f.Close()
	_ = os.Remove(f.Name())
	return true
}

// Apply mengunduh versi terbaru, memverifikasinya, dan mengganti biner. Mengembalikan versi baru.
// Proses yang berjalan tidak berubah sampai dimulai ulang.
func (u *Updater) Apply(ctx context.Context) (string, error) {
	u.mu.Lock()
	if u.checking {
		u.mu.Unlock()
		return "", errors.New("pembaruan sedang berjalan")
	}
	u.checking = true
	u.mu.Unlock()
	defer func() { u.mu.Lock(); u.checking = false; u.mu.Unlock() }()

	if u.Edition != "Official" {
		return "", errors.New("pembaruan otomatis hanya untuk build Official; unduh dari " + ReleasesURL)
	}
	info := u.Check(ctx)
	if info.Error != "" {
		return "", errors.New(info.Error)
	}
	if !info.Available {
		return u.Current, errors.New("sudah versi terbaru")
	}
	if !info.CanApply {
		return "", errors.New(info.Reason)
	}
	exe, err := u.exe()
	if err != nil {
		return "", err
	}
	bin, err := u.fetchBinary(ctx, info.Latest)
	if err != nil {
		return "", err
	}
	if err := replaceExe(exe, bin); err != nil {
		return "", err
	}
	u.mu.Lock()
	u.staged = info.Latest
	u.mu.Unlock()
	u.store(Info{Current: u.Current, Latest: info.Latest, Available: true, CheckedAt: time.Now(), URL: u.downloadPage()})
	return info.Latest, nil
}

func assetName(version string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("vibeasik-%s-windows-%s.zip", version, runtime.GOARCH)
	}
	return fmt.Sprintf("vibeasik-%s-%s-%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
}

func (u *Updater) get(ctx context.Context, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unduhan gagal: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unduhan gagal: HTTP %d (%s)", resp.StatusCode, url)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxDownload {
		return nil, errors.New("berkas unduhan terlalu besar")
	}
	return b, nil
}

// fetchBinary mengunduh arsip versi ini, memeriksa sha256-nya, dan mengembalikan isi biner.
func (u *Updater) fetchBinary(ctx context.Context, version string) ([]byte, error) {
	name := assetName(version)
	dir := u.Base + "/" + version
	sums, err := u.get(ctx, dir+"/sha256sums.txt")
	if err != nil {
		return nil, err
	}
	want := ""
	for _, ln := range strings.Split(string(sums), "\n") {
		f := strings.Fields(ln)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			want = f[0]
		}
	}
	if want == "" {
		return nil, fmt.Errorf("checksum %s tidak ada di sha256sums.txt", name)
	}
	archive, err := u.get(ctx, dir+"/"+name)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(archive)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), want) {
		return nil, errors.New("checksum tidak cocok; pembaruan dibatalkan")
	}
	return extract(archive, name)
}

func extract(archive []byte, name string) ([]byte, error) {
	exeName := "vibeasik"
	if strings.HasSuffix(name, ".zip") {
		exeName += ".exe"
		zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if f.Name == exeName {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(io.LimitReader(rc, maxDownload))
			}
		}
		return nil, errors.New("biner tidak ada di arsip")
	}
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err != nil {
			return nil, errors.New("biner tidak ada di arsip")
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == exeName {
			return io.ReadAll(io.LimitReader(tr, maxDownload))
		}
	}
}

// replaceExe menulis biner baru di folder yang sama lalu menggantikan yang lama secara atomik.
// Di Windows biner yang berjalan tidak bisa ditimpa, jadi dipindah dulu ke .old.
func replaceExe(exe string, bin []byte) error {
	if len(bin) < 1<<20 {
		return errors.New("biner hasil unduhan tidak masuk akal (terlalu kecil)")
	}
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".vibeasik-baru-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if runtime.GOOS == "windows" {
		old := exe + ".old"
		_ = os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			_ = os.Remove(tmpName)
			return err
		}
		if err := os.Rename(tmpName, exe); err != nil {
			_ = os.Rename(old, exe)
			_ = os.Remove(tmpName)
			return err
		}
		return nil
	}
	if err := os.Rename(tmpName, exe); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// ---- versi ----

func validVersion(v string) bool {
	_, ok := parse(v)
	return ok
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func newer(latest, current string) bool {
	a, ok1 := parse(latest)
	b, ok2 := parse(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// ---- pengaturan ----

// Background memeriksa saat start lalu tiap interval, dan (bila auto) memasang pembaruan di latar.
// Biner baru dipakai setelah mulai ulang; engine yang berjalan tidak diganggu.
func (u *Updater) Background(ctx context.Context, auto func() bool, interval time.Duration) {
	go func() {
		t := time.NewTimer(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
			info := u.Check(ctx)
			if info.CanApply && info.Available && u.Last().Staged != info.Latest && auto() {
				_, _ = u.Apply(ctx)
			}
			t.Reset(interval)
		}
	}()
}
