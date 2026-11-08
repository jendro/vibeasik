package cli

import (
	"strings"
	"testing"
)

func TestLanTarget(t *testing.T) {
	if got, err := lanTarget("192.168.1.3", ""); err != nil || got != "192.168.1.3" {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := lanTarget("pc.local", "budi"); err != nil || got != "budi@pc.local" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"", "-oProxyCommand=x", "a b", "h;rm"} {
		if _, err := lanTarget(bad, ""); err == nil {
			t.Errorf("host %q harus ditolak", bad)
		}
	}
	if _, err := lanTarget("h", "-x"); err == nil {
		t.Error("user berawalan - harus ditolak")
	}
}

func TestLanSSHArgs(t *testing.T) {
	a := lanSSHArgs("u@h", 2222, 8765, 8765, "http://localhost:8765/auth?t=abc")
	j := strings.Join(a, " ")
	for _, want := range []string{"-R 127.0.0.1:8765:127.0.0.1:8765", "-p 2222", "ExitOnForwardFailure=yes", "-- u@h", "xdg-open 'http://localhost:8765/auth?t=abc'"} {
		if !strings.Contains(j, want) {
			t.Errorf("argumen ssh tidak memuat %q: %s", want, j)
		}
	}
}
