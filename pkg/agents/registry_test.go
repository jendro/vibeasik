package agents

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
)

// shAdapter menjalankan skrip sh nyata; prompt adalah isi skrip.
type shAdapter struct{ path string }

func (a shAdapter) Key() string                    { return "test-sh" }
func (a shAdapter) Name() string                   { return "test sh" }
func (a shAdapter) Command() string                { return "sh" }
func (a shAdapter) Description() string            { return "" }
func (a shAdapter) Capabilities() []string         { return nil }
func (a shAdapter) Detect() (bool, string, string) { return true, a.path, "" }
func (a shAdapter) BuildArgs(p, _ string) []string { return []string{"-c", p} }

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh tidak tersedia")
	}
	old := registry.H
	t.Cleanup(func() { registry.H = old })
	registry.H.AgentCommand = nil
	// cwd proses dipindah ke folder lain agar terbukti tidak dipakai.
	t.Chdir(t.TempDir())
	r := &Registry{adapters: map[string]AgentAdapter{}}
	r.Register(shAdapter{path: sh})
	return r
}

func run(r *Registry, dir, script string) (*models.AgentExecutionResult, error) {
	return r.Execute(context.Background(), models.AgentExecutionRequest{AgentKey: "test-sh", Prompt: script, ProjectPath: dir, TimeoutSec: 10}, nil)
}

func TestExecuteRunsInProjectPathWithoutHook(t *testing.T) {
	r := newTestRegistry(t)
	proj := t.TempDir()
	res, err := run(r, proj, "pwd -P")
	if err != nil || !res.Success {
		t.Fatalf("eksekusi gagal: %v %+v", err, res)
	}
	want, _ := filepath.EvalSymlinks(proj)
	if got := strings.TrimSpace(res.Output); got != want {
		t.Fatalf("cwd agent = %q, mau %q", got, want)
	}
}

func TestExecuteUsesHookResult(t *testing.T) {
	r := newTestRegistry(t)
	proj := t.TempDir()
	var gotDir string
	registry.H.AgentCommand = func(path string, args []string, dir string, env []string) (string, []string, []string, error) {
		gotDir = dir
		return path, []string{"-c", "echo wrapped-$WRAP_MARK"}, append(env, "WRAP_MARK=ok"), nil
	}
	res, err := run(r, proj, "echo asli")
	if err != nil || !res.Success {
		t.Fatalf("eksekusi gagal: %v %+v", err, res)
	}
	if strings.TrimSpace(res.Output) != "wrapped-ok" {
		t.Fatalf("output = %q, hook tidak dipakai", res.Output)
	}
	if !filepath.IsAbs(gotDir) || gotDir != proj {
		t.Fatalf("hook menerima dir %q, mau absolut %q", gotDir, proj)
	}
}

func TestExecuteHookErrorRunsNothing(t *testing.T) {
	r := newTestRegistry(t)
	proj := t.TempDir()
	registry.H.AgentCommand = func(string, []string, string, []string) (string, []string, []string, error) {
		return "", nil, nil, errors.New("ditolak")
	}
	if _, err := run(r, proj, "touch marker"); err == nil {
		t.Fatal("mau error dari hook")
	}
	if _, err := os.Stat(filepath.Join(proj, "marker")); !os.IsNotExist(err) {
		t.Fatal("proses tetap berjalan walau hook error")
	}
}

func TestExecuteRejectsInvalidProjectPath(t *testing.T) {
	r := newTestRegistry(t)
	cwd, _ := os.Getwd()
	file := filepath.Join(t.TempDir(), "berkas")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for name, dir := range map[string]string{"kosong": "", "spasi": "  ", "tidak ada": filepath.Join(cwd, "nope"), "berkas": file} {
		if _, err := run(r, dir, "touch marker"); err == nil {
			t.Errorf("%s: mau error", name)
		}
	}
	if _, err := os.Stat(filepath.Join(cwd, "marker")); !os.IsNotExist(err) {
		t.Fatal("agent berjalan di cwd proses")
	}
}
