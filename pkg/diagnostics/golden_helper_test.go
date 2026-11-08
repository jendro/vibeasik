package diagnostics

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update", false, "tulis ulang berkas golden karakterisasi")

// assertGolden membandingkan v (dalam JSON) dengan testdata/golden/<name>.json.
func assertGolden(t *testing.T, name string, v any) {
	t.Helper()
	got, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("testdata", "golden", name+".json")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden %s belum ada (jalankan -update): %v", path, err)
	}
	if string(want) != string(got) {
		t.Errorf("golden %s berbeda dari keluaran saat ini", name)
		_ = os.WriteFile(filepath.Join(t.TempDir(), name+".got"), got, 0o644)
	}
}
