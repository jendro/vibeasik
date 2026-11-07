package localauth

import (
	"os"
	"testing"
)

func TestTokenCreatedOnceWithPrivatePerms(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(EnvToken, "")
	cached = ""
	a, err := Token()
	if err != nil || len(a) != 64 {
		t.Fatalf("token: %q %v", a, err)
	}
	fi, err := os.Stat(Path())
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("token file must be 0600, got %v", fi.Mode().Perm())
	}
	cached = ""
	b, _ := Token()
	if a != b || ClientToken() != a {
		t.Fatalf("token must persist: %q vs %q", a, b)
	}
	r, err := Rotate()
	if err != nil || r == a || ClientToken() != r {
		t.Fatalf("rotate failed: %v", err)
	}
	if !Equal(r, r) || Equal("", "") || Equal(a, r) {
		t.Fatal("Equal wrong")
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(EnvToken, "fixed")
	if tok, _ := Token(); tok != "fixed" || ClientToken() != "fixed" {
		t.Fatalf("env override ignored: %q", tok)
	}
}
