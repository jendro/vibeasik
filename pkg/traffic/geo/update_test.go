package geo

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func gz(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(b)
	_ = w.Close()
	return buf.Bytes()
}

// fakeMirror menyajikan basis data uji hanya untuk bulan `month`; bulan lain 404.
func fakeMirror(t *testing.T, month string) (*httptest.Server, *[]string) {
	t.Helper()
	src := writeFixtureDB(t)
	var hits []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		for _, s := range Sources {
			if r.URL.Path == "/"+s.Prefix+"-"+month+".mmdb.gz" {
				b, _ := os.ReadFile(filepath.Join(src, s.File))
				_, _ = w.Write(gz(t, b))
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestUpdateFallsBackToPreviousMonth(t *testing.T) {
	srv, hits := fakeMirror(t, "2026-10")
	dir := t.TempDir()
	u := &Updater{Dir: dir, BaseURL: srv.URL, Client: srv.Client(), Now: func() time.Time { return time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC) }}
	swapped := 0
	if err := u.Update(context.Background(), "all", func() { swapped++ }); err != nil {
		t.Fatal(err)
	}
	if swapped != 2 {
		t.Errorf("onSwap dipanggil %d kali, want 2", swapped)
	}
	if !strings.Contains(strings.Join(*hits, " "), "2026-11") {
		t.Errorf("harus mencoba bulan berjalan dulu: %v", *hits)
	}
	db := New(dir)
	defer db.Release()
	if got := db.Lookup("203.0.113.9"); got.City != "Jakarta" || got.NetType != NetResidential {
		t.Errorf("hasil unduhan tidak terbaca: %+v", got)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("berkas sementara tertinggal: %s", e.Name())
		}
	}
}

func TestUpdateOnlyASN(t *testing.T) {
	srv, _ := fakeMirror(t, "2026-11")
	dir := t.TempDir()
	u := &Updater{Dir: dir, BaseURL: srv.URL, Client: srv.Client(), Now: func() time.Time { return time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC) }}
	if err := u.Update(context.Background(), "asn", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, CityFile)); err == nil {
		t.Error("City tidak boleh diunduh bila hanya asn diminta")
	}
	if _, err := os.Stat(filepath.Join(dir, ASNFile)); err != nil {
		t.Error("ASN harus terpasang")
	}
}

func TestUpdateFailureKeepsOldFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "rusak", 500) }))
	defer srv.Close()
	dir := t.TempDir()
	old := filepath.Join(dir, ASNFile)
	if err := os.WriteFile(old, []byte("lama"), 0o600); err != nil {
		t.Fatal(err)
	}
	u := &Updater{Dir: dir, BaseURL: srv.URL, Client: srv.Client()}
	err := u.Update(context.Background(), "asn", nil)
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("galat harus menyebut HTTP 500: %v", err)
	}
	if b, _ := os.ReadFile(old); string(b) != "lama" {
		t.Error("berkas lama harus tetap utuh saat unduhan gagal")
	}
}

func TestUpdateBothMonthsMissing(t *testing.T) {
	srv, _ := fakeMirror(t, "1999-01")
	u := &Updater{Dir: t.TempDir(), BaseURL: srv.URL, Client: srv.Client()}
	if err := u.Update(context.Background(), "asn", nil); err == nil || !strings.Contains(err.Error(), "belum terbit") {
		t.Errorf("harus galat 'belum terbit': %v", err)
	}
}

func TestDownloadRespectsSizeLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(gz(t, bytes.Repeat([]byte{1}, 4096)))
	}))
	defer srv.Close()
	dir := t.TempDir()
	u := &Updater{Dir: dir, Client: srv.Client()}
	_, err := u.download(context.Background(), srv.URL+"/x.mmdb.gz", Source{File: "x.mmdb", MaxBytes: 1024})
	if err == nil || !strings.Contains(err.Error(), "melebihi batas") {
		t.Errorf("harus menolak berkas melebihi batas: %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("berkas sementara harus dibersihkan: %d", len(entries))
	}
}

func TestStaleAndAutoEnabled(t *testing.T) {
	dir := writeFixtureDB(t)
	now := time.Now()
	if Stale(dir, now) {
		t.Error("berkas baru tidak boleh dianggap usang")
	}
	if !Stale(dir, now.Add(StaleAfter+time.Hour)) {
		t.Error("berkas lama harus dianggap usang")
	}
	if !Stale(t.TempDir(), now) {
		t.Error("berkas yang belum ada harus dianggap usang")
	}
	t.Setenv(AutoEnv, "0")
	if AutoEnabled() {
		t.Error("VIBEASIK_GEO_AUTO=0 harus mematikan")
	}
	t.Setenv(AutoEnv, "")
	if !AutoEnabled() {
		t.Error("bawaan harus hidup")
	}
}
