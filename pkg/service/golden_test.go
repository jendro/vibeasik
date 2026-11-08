package service_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// golden membandingkan keluaran JSON dengan berkas testdata/<name>.golden.
// Set UPDATE_GOLDEN=1 untuk menulis ulang. Path sementara diganti <TMP>.
func golden(t *testing.T, name string, v any, tmp string) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if tmp != "" {
		got = strings.ReplaceAll(got, tmp, "<TMP>")
	}
	got += "\n"
	path := filepath.Join("testdata", name+".golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden %s: %v", name, err)
	}
	if string(want) != got {
		t.Fatalf("golden %s berbeda.\n--- want\n%s\n--- got\n%s", name, want, got)
	}
}

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
