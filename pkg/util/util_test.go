package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStrings(t *testing.T) {
	if got := FirstNonEmpty("", "  ", " a ", "b"); got != "a" {
		t.Fatalf("FirstNonEmpty = %q", got)
	}
	if got := FirstNonBlank("", "  ", " a ", "b"); got != " a " {
		t.Fatalf("FirstNonBlank = %q", got)
	}
	if FirstNonEmpty() != "" || FirstNonBlank(" ") != "" {
		t.Fatal("kosong harus \"\"")
	}
	if got := OneLine("a  b\n c", 100); got != "a b c" {
		t.Fatalf("OneLine = %q", got)
	}
	if got := OneLine("abcdef", 3); got != "abc…" {
		t.Fatalf("OneLine clip = %q", got)
	}
	if got := ClipText("  ab\ncd ", 3); got != "ab\n…" {
		t.Fatalf("ClipText = %q", got)
	}
	if got := ClipText(" ab ", 5); got != "ab" {
		t.Fatalf("ClipText short = %q", got)
	}
}

func TestFiles(t *testing.T) {
	d := t.TempDir()
	f := filepath.Join(d, "x.txt")
	if err := os.WriteFile(f, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	if ReadFileString(f) != "hi" || ReadFileString(filepath.Join(d, "none")) != "" {
		t.Fatal("ReadFileString")
	}
	if !FileExists(f) || FileExists(d) || FileExists(filepath.Join(d, "none")) {
		t.Fatal("FileExists")
	}
	if !PathExists(f) || !PathExists(d) || PathExists(filepath.Join(d, "none")) {
		t.Fatal("PathExists")
	}
}
