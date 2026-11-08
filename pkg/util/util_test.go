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

func TestContainsAny(t *testing.T) {
	if !ContainsAny("hello world", "foo", "world") {
		t.Fatal("ContainsAny should return true for matching substring")
	}
	if ContainsAny("hello world", "foo", "bar") {
		t.Fatal("ContainsAny should return false when no substrings match")
	}
	if ContainsAny("hello world") {
		t.Fatal("ContainsAny should return false with no substrings provided")
	}
	if !ContainsAny("", "") {
		t.Fatal("ContainsAny should return true for empty target and empty query")
	}
	if ContainsAny("", "a", "b") {
		t.Fatal("ContainsAny should return false for empty string against non-empty queries")
	}
}

func TestUniq(t *testing.T) {
	if got := Uniq(nil); got != nil {
		t.Fatalf("Uniq(nil) = %v, want nil", got)
	}
	in := []string{"a", "b", "a", "", "c", "b", "d", ""}
	want := []string{"a", "b", "c", "d"}
	got := Uniq(in)
	if len(got) != len(want) {
		t.Fatalf("Uniq(%v) len = %d, want %d", in, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Uniq[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFolderTree(t *testing.T) {
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "src", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(d, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(d, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "src", "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, ".git", "config"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	tree := GenerateFolderTree(d, 2)
	if !ContainsAny(tree, "src/") || !ContainsAny(tree, "main.go") {
		t.Fatalf("tree should contain src/ and main.go, got:\n%s", tree)
	}
	if ContainsAny(tree, ".git") || ContainsAny(tree, "node_modules") {
		t.Fatalf("tree should skip hidden dirs and node_modules, got:\n%s", tree)
	}
}

