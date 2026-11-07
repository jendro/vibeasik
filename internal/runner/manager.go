package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vibeasik/vibeasik/internal/models"
)

// State is the lifecycle phase of a component process launched by vibeasik.
type State string

const (
	StateStarting State = "starting" // proses jalan, port belum listening
	StateRunning  State = "running"  // port sudah menerima koneksi
	StateStopped  State = "stopped"  // dihentikan oleh pengguna
	StateExited   State = "exited"   // proses keluar sendiri
	StateIdle     State = "idle"     // belum pernah dijalankan
)

const (
	maxLogLines   = 400
	maxLineLength = 2000
	warnAfter     = 45 * time.Second
	stopGrace     = 5 * time.Second
)

var ErrAlreadyRunning = errors.New("komponen sudah berjalan")

// Status is the public snapshot of a component process.
type Status struct {
	ComponentID uint       `json:"component_id"`
	ProjectID   uint       `json:"project_id"`
	Name        string     `json:"name"`
	State       State      `json:"state"`
	PID         int        `json:"pid,omitempty"`
	Port        int        `json:"port"`
	URL         string     `json:"url,omitempty"`
	Command     string     `json:"command,omitempty"`
	Notes       []string   `json:"notes,omitempty"`
	Warning     string     `json:"warning,omitempty"`
	Listening   bool       `json:"listening"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	ExitedAt    *time.Time `json:"exited_at,omitempty"`
	ExitCode    *int       `json:"exit_code,omitempty"`
	Logs        []string   `json:"logs,omitempty"`
}

type process struct {
	mu       sync.Mutex
	status   Status
	cmd      *exec.Cmd
	logs     []string
	partial  string
	done     chan struct{}
	stopping bool
}

// Manager owns every dev server process vibeasik started in this engine instance.
type Manager struct {
	mu    sync.Mutex
	procs map[uint]*process
}

var defaultManager = &Manager{procs: map[uint]*process{}}

// Default returns the process-wide manager shared by the REST API and the MCP server.
func Default() *Manager { return defaultManager }

// Start launches comp with the given spec on port. The caller is responsible for the port being free.
func (m *Manager) Start(comp models.Component, port int, spec LaunchSpec) (*Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.procs[comp.ID]; ok && p.alive() {
		return nil, ErrAlreadyRunning
	}
	if spec.Command == "" {
		return nil, fmt.Errorf("komponen '%s' belum punya dev command", comp.Name)
	}
	if fi, err := os.Stat(comp.AbsolutePath); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("folder komponen tidak ditemukan: %s", comp.AbsolutePath)
	}

	cmd := shellCommand(spec.Command)
	cmd.Dir = comp.AbsolutePath
	cmd.Env = append(os.Environ(), spec.Env...)
	cmd.WaitDelay = 3 * time.Second // jangan menggantung bila cucu proses masih memegang pipe output
	setProcessGroup(cmd)

	now := time.Now()
	p := &process{
		cmd:  cmd,
		done: make(chan struct{}),
		status: Status{
			ComponentID: comp.ID,
			ProjectID:   comp.ProjectID,
			Name:        comp.Name,
			State:       StateStarting,
			Port:        port,
			URL:         fmt.Sprintf("http://localhost:%d", port),
			Command:     spec.Command,
			Notes:       spec.Notes,
			StartedAt:   &now,
		},
	}
	w := &logWriter{p: p}
	cmd.Stdout = w
	cmd.Stderr = w

	p.appendLine(fmt.Sprintf("$ %s   (PORT=%d)", spec.Command, port))
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("gagal menjalankan dev command: %w", err)
	}
	p.status.PID = cmd.Process.Pid
	m.procs[comp.ID] = p

	go p.wait()
	go p.watchPort(comp.Port)

	st := p.snapshot(false)
	return &st, nil
}

// Stop terminates the component's whole process tree (SIGTERM, then SIGKILL after a grace period).
func (m *Manager) Stop(componentID uint) (*Status, error) {
	m.mu.Lock()
	p, ok := m.procs[componentID]
	m.mu.Unlock()
	if !ok || !p.alive() {
		return nil, fmt.Errorf("komponen tidak sedang berjalan")
	}

	p.mu.Lock()
	p.stopping = true
	p.mu.Unlock()

	terminateTree(p.cmd)
	select {
	case <-p.done:
	case <-time.After(stopGrace):
		killTree(p.cmd)
		<-p.done
	}
	st := p.snapshot(false)
	return &st, nil
}

// StopAll stops every running process; called on engine shutdown so no dev server is orphaned.
func (m *Manager) StopAll() {
	m.mu.Lock()
	ids := make([]uint, 0, len(m.procs))
	for id, p := range m.procs {
		if p.alive() {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id uint) {
			defer wg.Done()
			_, _ = m.Stop(id)
		}(id)
	}
	wg.Wait()
}

// Status returns the component's snapshot, or ok=false when vibeasik never launched it.
func (m *Manager) Status(componentID uint, withLogs bool) (Status, bool) {
	m.mu.Lock()
	p, ok := m.procs[componentID]
	m.mu.Unlock()
	if !ok {
		return Status{}, false
	}
	return p.snapshot(withLogs), true
}

// List returns snapshots of every process vibeasik launched (running or finished).
func (m *Manager) List() []Status {
	m.mu.Lock()
	procs := make([]*process, 0, len(m.procs))
	for _, p := range m.procs {
		procs = append(procs, p)
	}
	m.mu.Unlock()

	out := make([]Status, 0, len(procs))
	for _, p := range procs {
		out = append(out, p.snapshot(false))
	}
	return out
}

func (p *process) alive() bool {
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}

func (p *process) wait() {
	err := p.cmd.Wait()
	now := time.Now()
	code := 0
	if p.cmd.ProcessState != nil {
		code = p.cmd.ProcessState.ExitCode()
	}

	p.mu.Lock()
	p.flushPartial()
	p.status.ExitedAt = &now
	p.status.ExitCode = &code
	if p.stopping {
		p.status.State = StateStopped
		p.appendLineLocked("■ Dihentikan dari vibeasik")
	} else {
		p.status.State = StateExited
		msg := fmt.Sprintf("■ Proses keluar (exit code %d)", code)
		if err != nil && code == -1 {
			msg = "■ Proses berhenti: " + err.Error()
		}
		p.appendLineLocked(msg)
	}
	p.mu.Unlock()
	close(p.done)
}

// watchPort flips starting→running once the port accepts connections and warns when the app
// seems to ignore PORT (still silent after warnAfter, or listening on its hardcoded port instead).
func (p *process) watchPort(declaredPort int) {
	port := p.status.Port
	deadline := time.Now().Add(warnAfter)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
		}
		if IsListening(port) {
			p.mu.Lock()
			if p.status.State == StateStarting {
				p.status.State = StateRunning
			}
			p.status.Warning = ""
			p.mu.Unlock()
			return
		}
		if time.Now().After(deadline) {
			warning := fmt.Sprintf("Proses berjalan tapi port :%d belum listening. Pastikan aplikasi membaca env PORT atau sesuaikan dev command.", port)
			if declaredPort > 0 && declaredPort != port && IsListening(declaredPort) {
				warning = fmt.Sprintf("Aplikasi tampaknya mengabaikan PORT dan listen di port hardcoded :%d. Ubah kode agar membaca env PORT supaya tidak bentrok.", declaredPort)
			}
			p.mu.Lock()
			p.status.Warning = warning
			p.mu.Unlock()
			// Tetap pantau dengan interval lebih jarang: dev server besar kadang butuh waktu lama.
			deadline = time.Now().Add(time.Hour)
			ticker.Reset(2 * time.Second)
		}
	}
}

func (p *process) snapshot(withLogs bool) Status {
	p.mu.Lock()
	st := p.status
	st.Notes = append([]string(nil), p.status.Notes...)
	if withLogs {
		st.Logs = append([]string(nil), p.logs...)
	}
	p.mu.Unlock()
	if p.alive() {
		st.Listening = IsListening(st.Port)
	}
	return st
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07]*\x07`)

// logWriter splits process output into lines kept in a bounded ring buffer.
type logWriter struct{ p *process }

func (w *logWriter) Write(b []byte) (int, error) {
	w.p.mu.Lock()
	defer w.p.mu.Unlock()
	text := w.p.partial + ansiRe.ReplaceAllString(string(b), "")
	lines := strings.Split(text, "\n")
	w.p.partial = lines[len(lines)-1]
	for _, line := range lines[:len(lines)-1] {
		w.p.appendLineLocked(line)
	}
	if len(w.p.partial) > maxLineLength {
		w.p.appendLineLocked(w.p.partial)
		w.p.partial = ""
	}
	return len(b), nil
}

func (p *process) appendLine(line string) {
	p.mu.Lock()
	p.appendLineLocked(line)
	p.mu.Unlock()
}

func (p *process) appendLineLocked(line string) {
	// Progress bar memakai \r: simpan hanya segmen terakhir.
	if i := strings.LastIndex(strings.TrimRight(line, "\r"), "\r"); i >= 0 {
		line = line[i+1:]
	}
	line = strings.TrimRight(line, "\r")
	if len(line) > maxLineLength {
		line = line[:maxLineLength] + "…"
	}
	p.logs = append(p.logs, line)
	if len(p.logs) > maxLogLines {
		p.logs = p.logs[len(p.logs)-maxLogLines:]
	}
}

func (p *process) flushPartial() {
	if p.partial != "" {
		p.appendLineLocked(p.partial)
		p.partial = ""
	}
}
