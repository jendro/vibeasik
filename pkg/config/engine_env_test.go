package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Variabel dari .env engine tidak boleh ikut ke proses aplikasi proyek; variabel dari shell tetap ikut.
func TestChildEnvironDropsEngineEnv(t *testing.T) {
	t.Setenv("SHELL_ONLY_VAR", "shell")
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte("ENGINE_TEST_DB_NAME=engine_db\nSHELL_ONLY_VAR=from_file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("ENGINE_TEST_DB_NAME") })
	loadEngineEnv(p)
	if os.Getenv("ENGINE_TEST_DB_NAME") != "engine_db" || os.Getenv("SHELL_ONLY_VAR") != "shell" {
		t.Fatal("loadEngineEnv must behave like godotenv.Load (no override)")
	}
	env := strings.Join(ChildEnviron(), "\n")
	if strings.Contains(env, "ENGINE_TEST_DB_NAME=") {
		t.Fatal("engine .env variable leaked to child environment")
	}
	if !strings.Contains(env, "SHELL_ONLY_VAR=shell") {
		t.Fatal("shell variable must be kept")
	}
}
