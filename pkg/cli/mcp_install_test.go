package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readServers(t *testing.T, p string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var d map[string]any
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	s, _ := d["mcpServers"].(map[string]any)
	return s
}

func TestEditMCPConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "mcp.json")
	orig := `{"theme":"dark","mcpServers":{"lain":{"command":"x","args":["y"]}}}`
	if err := os.WriteFile(p, []byte(orig), 0o600); err != nil {
		t.Fatal(err)
	}

	if changed, err := editMCPConfig(p, "/bin/vibeasik", false, true); err != nil || !changed {
		t.Fatalf("dry run harus melaporkan perubahan: %v %v", changed, err)
	}
	if b, _ := os.ReadFile(p); string(b) != orig {
		t.Fatal("dry run tidak boleh mengubah berkas")
	}

	if changed, err := editMCPConfig(p, "/bin/vibeasik", false, false); err != nil || !changed {
		t.Fatalf("install: %v %v", changed, err)
	}
	s := readServers(t, p)
	if s["lain"] == nil || s["vibeasik"] == nil {
		t.Fatalf("entri lain harus utuh: %v", s)
	}
	if b, _ := os.ReadFile(p + ".vibeasik.bak"); string(b) != orig {
		t.Fatal("cadangan harus berisi isi asli")
	}
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 {
		t.Fatalf("izin berkas harus dipertahankan, dapat %v", fi.Mode().Perm())
	}
	if changed, _ := editMCPConfig(p, "/bin/vibeasik", false, false); changed {
		t.Fatal("kedua kali harus no-op")
	}
	if changed, _ := editMCPConfig(p, "/opt/vibeasik", false, false); !changed {
		t.Fatal("path berubah harus memperbarui entri")
	}
	if changed, err := editMCPConfig(p, "", true, false); err != nil || !changed {
		t.Fatalf("uninstall: %v %v", changed, err)
	}
	if s := readServers(t, p); s["vibeasik"] != nil || s["lain"] == nil {
		t.Fatalf("uninstall hanya menghapus vibeasik: %v", s)
	}
	if changed, _ := editMCPConfig(p, "", true, false); changed {
		t.Fatal("uninstall kedua harus no-op")
	}
}

func TestEditMCPConfigInvalidJSONUntouched(t *testing.T) {
	p := filepath.Join(t.TempDir(), "mcp.json")
	os.WriteFile(p, []byte("{bukan json"), 0o644)
	if _, err := editMCPConfig(p, "/bin/vibeasik", false, false); err == nil {
		t.Fatal("JSON rusak harus ditolak")
	}
	if b, _ := os.ReadFile(p); string(b) != "{bukan json" {
		t.Fatal("berkas rusak tidak boleh ditimpa")
	}
}
