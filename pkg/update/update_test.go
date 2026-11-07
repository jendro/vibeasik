package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVersionCompare(t *testing.T) {
	cases := []struct {
		latest, cur string
		want        bool
	}{
		{"v0.3.2", "v0.3.1", true}, {"v0.3.1", "v0.3.1", false}, {"v0.10.0", "v0.9.9", true},
		{"v0.3.0", "v0.3.1", false}, {"v1.0.0", "dev", false}, {"abc", "v0.1.0", false},
	}
	for _, c := range cases {
		if got := newer(c.latest, c.cur); got != c.want {
			t.Errorf("newer(%s,%s)=%v", c.latest, c.cur, got)
		}
	}
}

func fakeRelease(t *testing.T, version string, body []byte, badSum bool) *httptest.Server {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "vibeasik", Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
	_, _ = tw.Write(body)
	tw.Close()
	gz.Close()
	sum := sha256.Sum256(buf.Bytes())
	hexsum := hex.EncodeToString(sum[:])
	if badSum {
		hexsum = hex.EncodeToString(make([]byte, 32))
	}
	name := assetName(version)
	mux := http.NewServeMux()
	mux.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, version) })
	mux.HandleFunc("/"+version+"/sha256sums.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  %s\n", hexsum, name)
	})
	mux.HandleFunc("/"+version+"/"+name, func(w http.ResponseWriter, r *http.Request) { w.Write(buf.Bytes()) })
	return httptest.NewServer(mux)
}

func TestApplyReplacesBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	body := bytes.Repeat([]byte("N"), 2<<20)
	srv := fakeRelease(t, "v0.9.0", body, false)
	defer srv.Close()
	exe := filepath.Join(t.TempDir(), "vibeasik")
	_ = os.WriteFile(exe, []byte("lama"), 0o755)

	u := New("v0.8.0", "Official")
	u.Base, u.ExePath = srv.URL, exe
	info := u.Check(context.Background())
	if !info.Available || !info.CanApply || info.Latest != "v0.9.0" {
		t.Fatalf("info: %+v", info)
	}
	v, err := u.Apply(context.Background())
	if err != nil || v != "v0.9.0" {
		t.Fatalf("apply: %v %v", v, err)
	}
	got, _ := os.ReadFile(exe)
	if !bytes.Equal(got, body) {
		t.Fatal("biner tidak diganti")
	}
	if st, _ := os.Stat(exe); st.Mode()&0o111 == 0 {
		t.Fatal("biner baru tidak executable")
	}
	if u.Last().Staged != "v0.9.0" {
		t.Fatal("Staged harus tercatat")
	}
}

func TestApplyRejectsBadChecksum(t *testing.T) {
	srv := fakeRelease(t, "v0.9.0", bytes.Repeat([]byte("N"), 2<<20), true)
	defer srv.Close()
	exe := filepath.Join(t.TempDir(), "vibeasik")
	_ = os.WriteFile(exe, []byte("lama"), 0o755)
	u := New("v0.8.0", "Official")
	u.Base, u.ExePath = srv.URL, exe
	if _, err := u.Apply(context.Background()); err == nil {
		t.Fatal("checksum salah harus ditolak")
	}
	if got, _ := os.ReadFile(exe); string(got) != "lama" {
		t.Fatal("biner lama tidak boleh berubah")
	}
}

func TestCommunityAndDevNotApplied(t *testing.T) {
	srv := fakeRelease(t, "v0.9.0", []byte("x"), false)
	defer srv.Close()
	c := New("v0.8.0", "Community")
	c.Base = srv.URL
	if i := c.Check(context.Background()); !i.Available || i.CanApply {
		t.Fatalf("Community hanya notifikasi: %+v", i)
	}
	d := New("dev", "Official")
	d.Base = srv.URL
	if i := d.Check(context.Background()); i.Available || i.Reason == "" {
		t.Fatalf("build dev tidak diperbarui: %+v", i)
	}
	e := New("v0.3.0-dev", "Official")
	e.Base = srv.URL
	if i := e.Check(context.Background()); i.Available {
		t.Fatalf("build -dev tidak diperbarui: %+v", i)
	}
}
