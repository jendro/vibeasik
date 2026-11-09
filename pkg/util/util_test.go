package util

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	var buf bytes.Buffer
	origWriter := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(origWriter)

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
	// Jalur normal (berkas ada / IsNotExist) tidak boleh mencatat warning
	if buf.Len() > 0 {
		t.Fatalf("expected no warnings for normal files/missing, got: %s", buf.String())
	}

	// Error sistem non-IsNotExist (misal permission denied) harus dicatat [WARN]
	if os.Geteuid() != 0 {
		forbiddenDir := filepath.Join(d, "forbidden")
		if err := os.Mkdir(forbiddenDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(forbiddenDir, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(forbiddenDir, 0o700)
		})

		unreadable := filepath.Join(forbiddenDir, "unreadable.txt")

		// 1. ReadFileString
		buf.Reset()
		if got := ReadFileString(unreadable); got != "" {
			t.Errorf("ReadFileString(%q) = %q, want empty string", unreadable, got)
		}
		if !strings.Contains(buf.String(), "[WARN] ReadFileString: os.ReadFile") {
			t.Errorf("expected log to contain '[WARN] ReadFileString: os.ReadFile', got: %s", buf.String())
		}

		// 2. FileExists
		buf.Reset()
		if FileExists(unreadable) {
			t.Errorf("FileExists(%q) = true, want false on permission error", unreadable)
		}
		if !strings.Contains(buf.String(), "[WARN] FileExists: os.Stat") {
			t.Errorf("expected log to contain '[WARN] FileExists: os.Stat', got: %s", buf.String())
		}

		// 3. PathExists
		buf.Reset()
		if PathExists(unreadable) {
			t.Errorf("PathExists(%q) = true, want false on permission error", unreadable)
		}
		if !strings.Contains(buf.String(), "[WARN] PathExists: os.Stat") {
			t.Errorf("expected log to contain '[WARN] PathExists: os.Stat', got: %s", buf.String())
		}
	}
}

func TestFilesPermissionAndNormalNotExist(t *testing.T) {
	var buf bytes.Buffer
	origWriter := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(origWriter)

	tmp := t.TempDir()
	normalMissing := filepath.Join(tmp, "tidak-ada.txt")

	// 1. Jalur normal: berkas tidak ada tidak boleh menghasilkan log warning
	if PathExists(normalMissing) {
		t.Errorf("PathExists(%q) = true, want false", normalMissing)
	}
	if FileExists(normalMissing) {
		t.Errorf("FileExists(%q) = true, want false", normalMissing)
	}
	if ReadFileString(normalMissing) != "" {
		t.Errorf("ReadFileString(%q) != \"\", want \"\"", normalMissing)
	}
	if buf.Len() > 0 {
		t.Errorf("expected no warning logs for normal missing files, got: %s", buf.String())
	}

	// 2. Jalur abnormal: permission denied (hanya diuji jika bukan root)
	if os.Geteuid() != 0 {
		buf.Reset()
		forbiddenDir := filepath.Join(tmp, "forbidden")
		if err := os.Mkdir(forbiddenDir, 0o700); err != nil {
			t.Fatalf("mkdir forbidden: %v", err)
		}
		if err := os.Chmod(forbiddenDir, 0o000); err != nil {
			t.Fatalf("chmod forbidden: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(forbiddenDir, 0o700)
		})

		unreadable := filepath.Join(forbiddenDir, "file.txt")
		if PathExists(unreadable) {
			t.Errorf("PathExists(%q) = true, want false on permission denied", unreadable)
		}
		if FileExists(unreadable) {
			t.Errorf("FileExists(%q) = true, want false on permission denied", unreadable)
		}
		_ = ReadFileString(unreadable)

		logOut := buf.String()
		if !strings.Contains(logOut, "[WARN] PathExists: os.Stat") {
			t.Errorf("log output missing PathExists warning: %q", logOut)
		}
		if !strings.Contains(logOut, "[WARN] FileExists: os.Stat") {
			t.Errorf("log output missing FileExists warning: %q", logOut)
		}
		if !strings.Contains(logOut, "[WARN] ReadFileString: os.ReadFile") {
			t.Errorf("log output missing ReadFileString warning: %q", logOut)
		}
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

	// Test permission error logging in GenerateFolderTree
	if os.Geteuid() != 0 {
		var buf bytes.Buffer
		origWriter := log.Writer()
		log.SetOutput(&buf)
		defer log.SetOutput(origWriter)

		forbiddenSub := filepath.Join(d, "secret")
		if err := os.Mkdir(forbiddenSub, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(forbiddenSub, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(forbiddenSub, 0o700)
		})

		_ = GenerateFolderTree(d, 2)
		if !strings.Contains(buf.String(), "[WARN] GenerateFolderTree: os.ReadDir") {
			t.Errorf("expected log to contain '[WARN] GenerateFolderTree: os.ReadDir', got: %s", buf.String())
		}
	}
}

