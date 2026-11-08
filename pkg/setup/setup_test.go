package setup

import (
	"os"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		d  DB
		ok bool
	}{
		{DB{Driver: "sqlite", Name: "vibeasik"}, true},
		{DB{Driver: "sqlite", Name: "1abc"}, false},
		{DB{Driver: "sqlite", Name: "a/../b"}, false},
		{DB{Driver: "postgres", Name: "vibeasik"}, false},
		{DB{Driver: "mysql", Name: "vibeasik", User: "root"}, true},
		{DB{Driver: "mysql", Name: "vibeasik"}, false},
		{DB{Driver: "mysql", Name: "vibeasik", User: "root", Password: "a'b"}, false},
	}
	for _, c := range cases {
		d := c.d
		if err := d.Validate(); (err == nil) != c.ok {
			t.Errorf("%+v: err=%v, ok diharapkan %v", c.d, err, c.ok)
		}
	}
}

func TestFlowAndEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if Complete() {
		t.Fatal("belum boleh selesai")
	}
	if err := Finish("v1"); err == nil {
		t.Fatal("Finish tanpa persetujuan harus gagal")
	}
	if err := AcceptTerms("v1"); err != nil {
		t.Fatal(err)
	}
	if err := Finish("v1"); err == nil {
		t.Fatal("Finish tanpa database harus gagal")
	}
	_ = os.MkdirAll(EnvPath()[:strings.LastIndex(EnvPath(), "/")], 0o700)
	_ = os.WriteFile(EnvPath(), []byte("# catatan\nFOO=bar\nDB_NAME=lama\n"), 0o600)
	if err := WriteEnv(DB{Driver: "sqlite", Name: "baru"}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(EnvPath())
	s := string(raw)
	for _, want := range []string{"# catatan", "FOO=bar", "DB_NAME='baru'", "DB_DRIVER='sqlite'", "baru.db"} {
		if !strings.Contains(s, want) {
			t.Errorf(".env kehilangan %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "lama") {
		t.Errorf("nilai lama masih ada:\n%s", s)
	}
	if err := MarkDatabase(); err != nil {
		t.Fatal(err)
	}
	if err := Finish("v1"); err != nil {
		t.Fatal(err)
	}
	if !Complete() {
		t.Fatal("penyiapan harus tercatat selesai")
	}
	// Versi ketentuan baru tidak membuka wizard lagi.
	if err := AcceptTerms("v2"); err != nil || !Complete() {
		t.Fatal("penyiapan cukup sekali; versi ketentuan baru tidak mengulanginya")
	}
}

func TestSQLiteReport(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if r := Test(DB{Driver: "sqlite", Name: "uji"}); !r.OK {
		t.Fatalf("sqlite harus siap: %+v", r)
	}
	if r := Test(DB{Driver: "sqlite", Name: "!"}); r.OK {
		t.Fatal("nama tak valid harus ditolak")
	}
}
