package activity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Kantor Agent hanya menampilkan pekerja yang prosesnya benar-benar masih hidup di laptop ini.
// Daftar proses dibaca dari /proc (Linux); di OS lain (atau bila /proc tak terbaca) pemeriksaan
// ini dilewati dan aturan waktu (IdleAfter/CodingAgentTTL) tetap berlaku seperti biasa.

const (
	procCacheTTL = 2 * time.Second
	// Sesi Claude Code yang tidak terdaftar di ~/.claude/sessions (mis. `claude -p`) dianggap hidup
	// hanya selama transkripnya masih bergerak dan masih ada proses claude yang berjalan.
	unregisteredClaudeGrace = 2 * time.Minute
)

type procSnapshot struct {
	ok             bool
	bootTime       time.Time
	starts         map[int]uint64 // pid -> waktu mulai (clock tick sejak boot)
	names          map[string]bool
	claudeSessions map[string]bool // sessionId (slug) dari proses Claude Code yang hidup
	claudeRegistry bool
}

var (
	procMu     sync.Mutex
	procCache  *procSnapshot
	procCached time.Time
	procRoot   = "/proc"
	claudeDir  = func() string { h, _ := os.UserHomeDir(); return filepath.Join(h, ".claude", "sessions") }
	clockTicks = 100.0 // USER_HZ pada Linux
)

func currentProcs(now time.Time) *procSnapshot {
	procMu.Lock()
	defer procMu.Unlock()
	if procCache != nil && now.Sub(procCached) < procCacheTTL {
		return procCache
	}
	procCache, procCached = readProcs(), now
	return procCache
}

func readProcs() *procSnapshot {
	s := &procSnapshot{starts: map[int]uint64{}, names: map[string]bool{}, claudeSessions: map[string]bool{}}
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return s
	}
	if b, err := os.ReadFile(filepath.Join(procRoot, "stat")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if f := strings.Fields(line); len(f) == 2 && f[0] == "btime" {
				if sec, err := strconv.ParseInt(f[1], 10, 64); err == nil {
					s.bootTime = time.Unix(sec, 0)
				}
			}
		}
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		start, ok := procStart(pid)
		if !ok {
			continue
		}
		s.starts[pid] = start
		if cmd, err := os.ReadFile(filepath.Join(procRoot, e.Name(), "cmdline")); err == nil && len(cmd) > 0 {
			argv0 := string(cmd)
			if i := strings.IndexByte(argv0, 0); i >= 0 {
				argv0 = argv0[:i]
			}
			s.names[strings.ToLower(filepath.Base(argv0))] = true
		}
		if comm, err := os.ReadFile(filepath.Join(procRoot, e.Name(), "comm")); err == nil {
			s.names[strings.ToLower(strings.TrimSpace(string(comm)))] = true
		}
	}
	s.ok = len(s.starts) > 0
	s.readClaudeRegistry()
	return s
}

// procStart membaca kolom ke-22 /proc/<pid>/stat (starttime). Nama proses bisa berisi spasi/kurung,
// jadi kolom dihitung setelah ')' terakhir.
func procStart(pid int) (uint64, bool) {
	b, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, false
	}
	i := strings.LastIndexByte(string(b), ')')
	if i < 0 {
		return 0, false
	}
	f := strings.Fields(string(b)[i+1:])
	if len(f) < 20 {
		return 0, false
	}
	v, err := strconv.ParseUint(f[19], 10, 64)
	return v, err == nil
}

// readClaudeRegistry membaca ~/.claude/sessions/<pid>.json yang ditulis setiap proses Claude Code;
// sesi dihitung hidup bila pid-nya masih berjalan dengan waktu mulai yang sama (pid tidak dipakai ulang).
func (s *procSnapshot) readClaudeRegistry() {
	dir := claudeDir()
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return
	}
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		s.claudeRegistry = true
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var reg struct {
			PID       int    `json:"pid"`
			SessionID string `json:"sessionId"`
			ProcStart string `json:"procStart"`
		}
		if json.Unmarshal(b, &reg) != nil || reg.PID == 0 || reg.SessionID == "" {
			continue
		}
		start, alive := s.starts[reg.PID]
		if !alive || (reg.ProcStart != "" && reg.ProcStart != strconv.FormatUint(start, 10)) {
			continue
		}
		s.claudeSessions[slug(reg.SessionID)] = true
	}
}

// pidAlive: proses masih ada dan sudah berjalan sebelum `since` (bukan pid lama yang dipakai ulang).
func (s *procSnapshot) pidAlive(pid int, since time.Time) bool {
	start, ok := s.starts[pid]
	if !ok {
		return false
	}
	if s.bootTime.IsZero() || since.IsZero() {
		return true
	}
	began := s.bootTime.Add(time.Duration(float64(start) / clockTicks * float64(time.Second)))
	return !began.After(since.Add(2 * time.Second))
}

func (s *procSnapshot) running(names ...string) bool {
	for _, n := range names {
		if s.names[n] {
			return true
		}
	}
	return false
}

// keyPID mengambil pid pada key "<prefix>:<pid>[:...]".
func keyPID(key string) (int, bool) {
	parts := strings.SplitN(key, ":", 3)
	if len(parts) < 2 {
		return 0, false
	}
	pid, err := strconv.Atoi(parts[1])
	return pid, err == nil && pid > 0
}

// processAlive menjawab apakah proses di balik pekerja masih berjalan. known=false bila tidak bisa
// dipastikan (OS tanpa /proc, atau sumber tanpa proses lokal seperti MCP HTTP / laporan REST).
func processAlive(w models.AgentWorker, now time.Time, s *procSnapshot) (alive, known bool) {
	if s == nil || !s.ok {
		return true, false
	}
	key := w.Key
	switch {
	case strings.HasPrefix(key, "llm:"), strings.HasPrefix(key, "local:"), strings.HasPrefix(key, "planner:"), strings.HasPrefix(key, "mcp-stdio:"):
		pid, ok := keyPID(key)
		if !ok {
			return true, false
		}
		return s.pidAlive(pid, w.StartedAt), true
	case strings.HasPrefix(key, "report:claude-code:"):
		if !s.running("claude") {
			return false, true
		}
		if !s.claudeRegistry {
			return true, false
		}
		session, _, _ := strings.Cut(strings.TrimPrefix(key, "report:claude-code:"), ":sub:") // sub agent hidup selama sesi koordinatornya
		if s.claudeSessions[session] {
			return true, true
		}
		return now.Sub(w.LastSeenAt) <= unregisteredClaudeGrace, true
	case strings.HasPrefix(key, "antigravity-cli:"):
		return s.running("agy", "antigravity-cli"), true
	case strings.HasPrefix(key, "antigravity:"):
		return s.running("antigravity"), true
	case strings.HasPrefix(key, "copilot-cli:"):
		return s.running("copilot"), true
	case strings.HasPrefix(key, "copilot-vscode:"):
		return s.running("code", "code-insiders", "codium"), true
	}
	return true, false
}
