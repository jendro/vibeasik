package activity

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Watcher membaca jejak kerja asli agent coding di laptop ini (transkrip yang ditulis agent itu sendiri)
// sehingga Kantor Agent hanya menampilkan pekerjaan yang benar-benar terjadi, tanpa perlu agent melapor.

const (
	watchInterval   = 2 * time.Second
	initialTailSize = 4 << 20 // file besar dibaca dari 4 MB terakhir saat pertama ditemukan
)

// Observation adalah keadaan terkini satu sesi agent menurut transkripnya.
type Observation struct {
	Key         string
	Name        string
	ClientName  string
	Transport   string
	Model       string
	Task        string
	Action      string
	Status      string // working | idle | left
	ProjectPath string
	TasksDone   int // jumlah kumulatif giliran yang selesai di bagian log yang sudah dibaca
	LastEvent   time.Time
	TodoID      uint   // todo vibeasik terakhir yang diubah agent lewat update_todo
	TodoStatus  string // status yang dikirim pada update_todo tersebut
	Role        string // peran mode divisi: koordinator (pengambil todo) / anak buah
	ParentKey   string // key koordinator bila sesi ini sub agent
	// OnlyExisting: hanya memperbarui pekerja yang sudah ada dan belum ditutup (transkrip sub agent
	// memperkaya aksi; lahir & selesainya ditentukan transkrip koordinator).
	OnlyExisting bool
}

// transcriptParser memproses satu baris log (urut) dan melaporkan keadaannya.
type transcriptParser interface {
	feed(line []byte)
	observation() Observation
}

// subagentParser: parser yang juga melaporkan sub agent suruhan sesinya (mode divisi).
type subagentParser interface {
	subagents() []Observation
}

// transcriptSource mendeskripsikan satu jenis agent: di mana log-nya dan cara membacanya.
type transcriptSource struct {
	Agent  string
	Dir    string
	files  func() []string
	parser func(path string) transcriptParser
}

type trackedFile struct {
	offset      int64
	size        int64
	mtime       time.Time
	rest        []byte
	parser      transcriptParser
	appliedSig  string
	appliedDone int
	applied     bool
	projectPath string
	todoSig     string
	subSigs     map[string]string // key sub agent → keadaan terakhir yang sudah ditulis
}

// SourceStatus menerangkan sumber yang dipantau (untuk ditampilkan apa adanya di Studio).
type SourceStatus struct {
	Agent          string `json:"agent"`
	Path           string `json:"path"`
	Found          bool   `json:"found"`
	ActiveSessions int    `json:"active_sessions"`
}

var (
	watchMu      sync.Mutex
	sources      []transcriptSource
	tracked      = map[string]*trackedFile{}
	sourceStatus []SourceStatus
)

// StartLocalWatchers memulai pemantauan transkrip agent lokal (dipanggil sekali oleh engine).
func StartLocalWatchers(ctx context.Context) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	cfgDir, _ := os.UserConfigDir()
	watchMu.Lock()
	sources = defaultSources(home, cfgDir)
	watchMu.Unlock()
	livenessEnabled.Store(true)
	go func() {
		t := time.NewTicker(watchInterval)
		defer t.Stop()
		for {
			scanSources(time.Now())
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

func defaultSources(home, cfgDir string) []transcriptSource {
	globber := func(pattern string) func() []string {
		return func() []string { m, _ := filepath.Glob(pattern); return m }
	}
	agy := filepath.Join(home, ".gemini", "antigravity", "brain")
	agyCLI := filepath.Join(home, ".gemini", "antigravity-cli", "brain")
	claude := filepath.Join(home, ".claude", "projects")
	copilot := filepath.Join(home, ".copilot", "session-state")
	vscode := filepath.Join(cfgDir, "Code", "User", "workspaceStorage")
	return []transcriptSource{
		{Agent: "Claude Code", Dir: claude, files: globber(filepath.Join(claude, "*", "*.jsonl")), parser: newClaudeParser},
		{Agent: "Claude Code (sub agent)", Dir: claude, files: globber(filepath.Join(claude, "*", "*", "subagents", "*.jsonl")), parser: newClaudeSubFileParser},
		{Agent: "Antigravity", Dir: agy, files: globber(filepath.Join(agy, "*", ".system_generated", "logs", "transcript.jsonl")),
			parser: func(p string) transcriptParser { return newAntigravityParser(p, "Antigravity", "antigravity") }},
		{Agent: "Antigravity CLI", Dir: agyCLI, files: globber(filepath.Join(agyCLI, "*", ".system_generated", "logs", "transcript.jsonl")),
			parser: func(p string) transcriptParser { return newAntigravityParser(p, "Antigravity CLI", "antigravity-cli") }},
		{Agent: "Copilot CLI", Dir: copilot, files: globber(filepath.Join(copilot, "*", "events.jsonl")), parser: newCopilotCLIParser},
		{Agent: "Copilot Chat (VS Code)", Dir: vscode, files: globber(filepath.Join(vscode, "*", "chatSessions", "*.jsonl")), parser: newCopilotChatParser},
	}
}

// WatchedSources mengembalikan sumber yang dipantau beserta jumlah sesi aktifnya.
func WatchedSources() []SourceStatus {
	watchMu.Lock()
	defer watchMu.Unlock()
	return append([]SourceStatus(nil), sourceStatus...)
}

func scanSources(now time.Time) {
	watchMu.Lock()
	defer watchMu.Unlock()
	status := make([]SourceStatus, 0, len(sources))
	seen := map[string]bool{}
	for _, src := range sources {
		st := SourceStatus{Agent: src.Agent, Path: src.Dir}
		if info, err := os.Stat(src.Dir); err == nil && info.IsDir() {
			st.Found = true
		}
		for _, path := range src.files() {
			info, err := os.Stat(path)
			if err != nil || now.Sub(info.ModTime()) > CodingAgentTTL {
				continue
			}
			seen[path] = true
			st.ActiveSessions++
			tf := tracked[path]
			if tf == nil {
				tf = &trackedFile{parser: src.parser(path)}
				tracked[path] = tf
			}
			if info.Size() == tf.size && info.ModTime().Equal(tf.mtime) {
				continue
			}
			if info.Size() < tf.offset { // file ditulis ulang: mulai lagi dari awal
				*tf = trackedFile{parser: src.parser(path)}
			}
			readNewLines(path, tf, info.Size())
			tf.size, tf.mtime = info.Size(), info.ModTime()
			applyTracked(tf, info.ModTime())
		}
		status = append(status, st)
	}
	for path := range tracked {
		if !seen[path] {
			delete(tracked, path)
		}
	}
	sourceStatus = status
}

// readNewLines membaca byte baru sejak offset terakhir dan meneruskan baris lengkap ke parser.
func readNewLines(path string, tf *trackedFile, size int64) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	start := tf.offset
	skipPartial := false
	if start == 0 && size > initialTailSize {
		start, skipPartial = size-initialTailSize, true
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return
	}
	r := bufio.NewReaderSize(f, 256<<10)
	if skipPartial {
		if skipped, err := r.ReadBytes('\n'); err == nil {
			start += int64(len(skipped))
		}
	}
	tf.offset = start
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 && line[len(line)-1] == '\n' {
			full := append(tf.rest, line...)
			tf.rest = nil
			tf.offset += int64(len(line))
			if trimmed := bytes.TrimSpace(full); len(trimmed) > 0 {
				tf.parser.feed(trimmed)
			}
			continue
		}
		// Baris terakhir belum lengkap (agent masih menulis): simpan, baca lagi nanti.
		tf.rest = append(tf.rest, line...)
		tf.offset += int64(len(line))
		if err != nil {
			return
		}
	}
}

func applyTracked(tf *trackedFile, mtime time.Time) {
	firstRead := !tf.applied
	applySession(tf, mtime)
	applySubagents(tf, firstRead)
}

func applySession(tf *trackedFile, mtime time.Time) {
	o := tf.parser.observation()
	if o.Key == "" || (o.Task == "" && o.Action == "" && o.Status == "") {
		return
	}
	if o.LastEvent.IsZero() {
		o.LastEvent = mtime
	}
	todoSig := fmt.Sprintf("%d:%s", o.TodoID, o.TodoStatus)
	sig := strings.Join([]string{o.Name, o.Model, o.Task, o.Action, o.Status, o.ProjectPath, todoSig, o.LastEvent.Format(time.RFC3339Nano)}, "|")
	if sig == tf.appliedSig && o.TasksDone == tf.appliedDone {
		return
	}
	delta := o.TasksDone - tf.appliedDone
	if delta < 0 {
		delta = 0
	}
	resolveProject := o.ProjectPath != "" && o.ProjectPath != tf.projectPath
	if o.TodoID == 0 || todoSig == tf.todoSig {
		o.TodoID = 0 // todo yang sama tidak diterapkan ulang (agar tugas selesai tidak terhitung ganda)
	}
	if ApplyObservation(o, delta, resolveProject, !tf.applied) {
		tf.appliedSig, tf.appliedDone, tf.projectPath, tf.applied, tf.todoSig = sig, o.TasksDone, o.ProjectPath, true, todoSig
	}
}

// applySubagents menulis anak buah sesi ini; tiap sub agent hanya ditulis ulang saat keadaannya berubah.
// firstRead: log baru dibaca pertama kali (mis. engine baru jalan), jadi sub agent yang sudah selesai
// tidak menambah hitungan tugas lagi.
func applySubagents(tf *trackedFile, firstRead bool) {
	sp, ok := tf.parser.(subagentParser)
	if !ok {
		return
	}
	for _, o := range sp.subagents() {
		sig := strings.Join([]string{o.Status, o.Action, o.Model, o.ProjectPath, o.LastEvent.Format(time.RFC3339Nano)}, "|")
		if tf.subSigs[o.Key] == sig {
			continue
		}
		if ApplyObservation(o, o.TasksDone, tf.subSigs[o.Key] == "", firstRead) {
			if tf.subSigs == nil {
				tf.subSigs = map[string]string{}
			}
			tf.subSigs[o.Key] = sig
		}
	}
}

// ApplyObservation menulis keadaan sesi ke pekerja. LastSeenAt memakai waktu event asli dari log.
// firstRead = log baru pertama kali dibaca (mis. engine baru jalan): bila pekerjanya sudah ada,
// giliran selesai di log itu sudah pernah dihitung sehingga tidak ditambahkan lagi.
func ApplyObservation(o Observation, doneDelta int, resolveProject, firstRead bool) bool {
	d := getDB()
	if d == nil {
		return false
	}
	if shouldSkipObservation(d, o) {
		return true
	}
	base := models.AgentWorker{
		Kind:       models.WorkerKindCodingAgent,
		Name:       o.Name,
		ClientName: o.ClientName,
		Transport:  o.Transport,
	}
	upsertWorker(d, o.Key, base, func(w *models.AgentWorker) {
		applyObservationAttributes(w, o)
		if resolveProject {
			applyObservationProject(w, o.ProjectPath)
		}
		applyObservationTasksAndTodo(w, o, doneDelta, firstRead)
		applyObservationStatus(w, o)
	})
	return true
}

func shouldSkipObservation(d *gorm.DB, o Observation) bool {
	if !o.OnlyExisting {
		return false
	}
	var cur models.AgentWorker
	if d.Select("status", "finished_at").Where("worker_key = ?", o.Key).First(&cur).Error != nil {
		return true
	}
	return cur.FinishedAt != nil || cur.Status == models.WorkerStatusDone || cur.Status == models.WorkerStatusLeft
}

func applyObservationAttributes(w *models.AgentWorker, o Observation) {
	w.Name = o.Name
	if w.ClientName == "" {
		w.ClientName = o.ClientName
	}
	if w.Transport == "" || w.Transport == "hook" {
		w.Transport = o.Transport
	}
	if o.Model != "" {
		w.Model = truncate(o.Model, maxGenericLength)
	}
	if o.Action != "" {
		w.Action = o.Action
	}
	if o.Role != "" {
		w.Role = truncate(o.Role, maxGenericLength)
	}
	if o.ParentKey != "" {
		w.ParentKey = o.ParentKey
	}
}

func applyObservationProject(w *models.AgentWorker, projectPath string) {
	w.ProjectPath = truncate(projectPath, maxPathLength)
	if id := projectIDForPath(projectPath); id > 0 {
		setProject(w, id)
		return
	}
	w.ProjectID = nil
	w.ProjectName = truncate(repoName(projectPath), maxGenericLength)
}

func applyObservationTasksAndTodo(w *models.AgentWorker, o Observation, doneDelta int, firstRead bool) {
	if !(firstRead && w.ID != 0) {
		w.TasksDone += doneDelta
	}
	if o.TodoID > 0 {
		applyTodoUpdate(w, o.TodoID, o.TodoStatus)
	}
	// Selama memegang todo, gelembung menampilkan judul todo; setelah itu kembali ke prompt terakhir.
	// "Selesai: <todo>" hanya tampil pada pembaruan saat todo itu diselesaikan (o.TodoID dikosongkan
	// applyTracked untuk todo yang sama), lalu kembali ke prompt terakhir.
	if o.Task != "" && w.TodoID == nil && !(o.TodoID > 0 && o.TodoStatus == models.TodoStatusCompleted) {
		w.Task = o.Task
	}
}

func applyObservationStatus(w *models.AgentWorker, o Observation) {
	w.LastSeenAt = o.LastEvent
	switch o.Status {
	case models.WorkerStatusLeft:
		at := o.LastEvent
		w.Status, w.FinishedAt = models.WorkerStatusLeft, &at
	case models.WorkerStatusIdle:
		w.Status = models.WorkerStatusIdle
	case models.WorkerStatusDone: // sub agent melapor & ditutup
		at := o.LastEvent
		w.Status, w.FinishedAt = models.WorkerStatusDone, &at
	default:
		w.Status = models.WorkerStatusWorking
	}
}

// repoName memberi nama folder repo (akar git) dari sebuah path file/direktori.
func repoName(path string) string {
	dir := filepath.Clean(path)
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for cur, i := dir, 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return filepath.Base(cur)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return filepath.Base(dir)
}

// mergeDuplicateMCP menyembunyikan baris koneksi MCP bila agent yang sama sudah terpantau dari
// transkripnya (satu agent = satu orang), dan koneksi MCP yang belum pernah memanggil tool.
func mergeDuplicateMCP(workers []models.AgentWorker) []models.AgentWorker {
	observed := map[string]bool{}
	for _, w := range workers {
		if !strings.HasPrefix(w.Key, "mcp-") {
			observed[w.Name] = true
		}
	}
	out := workers[:0]
	for _, w := range workers {
		if strings.HasPrefix(w.Key, "mcp-") && (w.ToolCalls == 0 || observed[w.Name]) {
			continue
		}
		out = append(out, w)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartedAt.Before(out[j].StartedAt) })
	return out
}
