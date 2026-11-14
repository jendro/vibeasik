package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/runner"
	"github.com/vibeasik/vibeasik/pkg/secaudit"
)

// Sesi Uji Keamanan: berjalan di engine (bukan di browser) seperti Uji Runtime — server komponen backend
// dinyalakan bila belum berjalan, kode dibaca, server diuji lewat request nyata, lalu server dimatikan lagi.
// Studio mem-poll snapshot tiap detik untuk menampilkan langkah, aktivitas, dan log server.

const (
	secTestDefaultBoot = 120 * time.Second
	secTestMaxBoot     = 300 * time.Second
	secTestMaxEvents   = 400
	secTestMaxLogs     = 300
)

// ErrSecTestBusy = uji keamanan proyek ini masih berjalan.
var ErrSecTestBusy = errors.New("uji keamanan proyek ini masih berjalan")

// SecTestStep = satu langkah uji (pending, running, done, failed, skipped).
type SecTestStep struct {
	Key        string     `json:"key"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	Detail     string     `json:"detail,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	DurationMs int64      `json:"duration_ms,omitempty"`
}

// SecTestEvent = satu baris aktivitas uji.
type SecTestEvent struct {
	At      time.Time `json:"at"`
	Level   string    `json:"level"` // info, ok, warn, error
	Message string    `json:"message"`
}

// SecTestServer = server yang diuji. Bentuknya sejalan dengan komponen Uji Runtime agar terminal log di Studio
// bisa dipakai bersama.
type SecTestServer struct {
	ComponentID uint     `json:"component_id"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	State       string   `json:"state"`  // starting, running, ready, stopped, failed, external
	Source      string   `json:"source"` // launched (dinyalakan uji), reused (sudah berjalan), external (URL live)
	Command     string   `json:"command,omitempty"`
	Port        int      `json:"port,omitempty"`
	URL         string   `json:"url,omitempty"`
	PID         int      `json:"pid,omitempty"`
	BootMs      int64    `json:"boot_ms,omitempty"`
	KeptRunning bool     `json:"kept_running,omitempty"`
	ErrorLines  int      `json:"error_lines"`
	RoutesFound int      `json:"routes_found"`
	Note        string   `json:"note,omitempty"`
	Logs        []string `json:"logs,omitempty"`
}

// SecTestSession = snapshot satu uji keamanan.
type SecTestSession struct {
	ID          string           `json:"id"`
	ProjectID   uint             `json:"project_id"`
	ProjectName string           `json:"project_name"`
	Status      string           `json:"status"` // running, completed, failed, cancelled
	Steps       []SecTestStep    `json:"steps"`
	Events      []SecTestEvent   `json:"events"`
	Server      *SecTestServer   `json:"server,omitempty"`
	Report      *secaudit.Report `json:"report,omitempty"`
	Summary     string           `json:"summary,omitempty"`
	Error       string           `json:"error,omitempty"`
	StartedAt   time.Time        `json:"started_at"`
	FinishedAt  *time.Time       `json:"finished_at,omitempty"`

	mu     sync.Mutex
	cancel context.CancelFunc
}

var (
	secSessionsMu sync.Mutex
	secSessions   = map[uint]*SecTestSession{}
)

func newSecTestSession(p *models.Project, req SecurityTestRequest) *SecTestSession {
	now := time.Now()
	sess := &SecTestSession{
		ID:          fmt.Sprintf("sec-%d-%d", p.ID, now.UnixNano()),
		ProjectID:   p.ID,
		ProjectName: p.Name,
		Status:      "running",
		StartedAt:   now,
	}
	serverTitle := "Nyalakan server"
	if strings.TrimSpace(req.BaseURL) != "" {
		serverTitle = "Hubungi server live"
	}
	sess.Steps = []SecTestStep{
		{Key: "server", Title: serverTitle, Status: "pending"},
		{Key: "static", Title: "Baca router & kode backend", Status: "pending"},
		{Key: "live", Title: "Periksa header, CORS & HTTPS", Status: "pending"},
		{Key: "attack", Title: "Serang endpoint: SQLi, XSS, traversal, JWT, …", Status: "pending"},
		{Key: "burst", Title: "Uji rate limit (DDoS & brute-force)", Status: "pending"},
		{Key: "stop", Title: "Matikan server", Status: "pending"},
	}
	return sess
}

// snapshot menyalin sesi agar bisa diserialisasi tanpa balapan dengan goroutine uji.
func (s *SecTestSession) snapshot() *SecTestSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := &SecTestSession{
		ID: s.ID, ProjectID: s.ProjectID, ProjectName: s.ProjectName, Status: s.Status, Report: s.Report,
		Summary: s.Summary, Error: s.Error, StartedAt: s.StartedAt, FinishedAt: s.FinishedAt,
		Steps:  append([]SecTestStep{}, s.Steps...),
		Events: append([]SecTestEvent{}, s.Events...), // [] bukan null di JSON
	}
	if s.Server != nil {
		srv := *s.Server
		srv.Logs = append([]string(nil), s.Server.Logs...)
		out.Server = &srv
	}
	return out
}

func (s *SecTestSession) event(level, format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, SecTestEvent{At: time.Now(), Level: level, Message: fmt.Sprintf(format, args...)})
	if n := len(s.Events); n > secTestMaxEvents {
		s.Events = append([]SecTestEvent(nil), s.Events[n-secTestMaxEvents:]...)
	}
}

func (s *SecTestSession) step(key, status, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Steps {
		st := &s.Steps[i]
		if st.Key != key {
			continue
		}
		if st.Status == status && st.FinishedAt != nil {
			if detail != "" {
				st.Detail = detail // langkah sudah selesai: hanya perbarui keterangan, bukan durasinya
			}
			continue
		}
		now := time.Now()
		if status == "running" {
			st.StartedAt = &now
		} else if status != "pending" {
			st.FinishedAt = &now
			if st.StartedAt != nil {
				st.DurationMs = now.Sub(*st.StartedAt).Milliseconds()
			}
		}
		st.Status = status
		if detail != "" {
			st.Detail = detail
		}
	}
}

func (s *SecTestSession) stepStatus(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, st := range s.Steps {
		if st.Key == key {
			return st.Status
		}
	}
	return ""
}

func (s *SecTestSession) setServer(fn func(srv *SecTestServer)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Server == nil {
		s.Server = &SecTestServer{}
	}
	fn(s.Server)
}

var secErrLineRe = regexp.MustCompile(`(?i)(panic:|fatal|uncaught|unhandled|traceback|\berror\b|exception)`)

// pullLogs menyalin log proses server dari runner ke sesi.
func (s *securityTestService) pullLogs(sess *SecTestSession, componentID uint) {
	if s.runner == nil {
		return
	}
	st, err := s.runner.Status(componentID, true)
	if err != nil {
		return
	}
	logs := st.Logs
	if len(logs) > secTestMaxLogs {
		logs = logs[len(logs)-secTestMaxLogs:]
	}
	errs := 0
	for _, l := range logs {
		if secErrLineRe.MatchString(l) {
			errs++
		}
	}
	sess.setServer(func(srv *SecTestServer) {
		srv.Logs = append([]string(nil), logs...)
		srv.ErrorLines = errs
		if st.PID > 0 {
			srv.PID = st.PID
		}
	})
}

// Start memulai uji di latar; satu uji berjalan per proyek.
func (s *securityTestService) Start(req SecurityTestRequest) (*SecTestSession, error) {
	plan, err := s.prepare(context.Background(), req)
	if err != nil {
		return nil, err
	}
	secSessionsMu.Lock()
	if cur, ok := secSessions[plan.project.ID]; ok && cur.snapshot().Status == "running" {
		secSessionsMu.Unlock()
		return cur.snapshot(), ErrSecTestBusy
	}
	sess := newSecTestSession(plan.project, req)
	ctx, cancel := context.WithCancel(context.Background())
	sess.cancel = cancel
	secSessions[plan.project.ID] = sess
	secSessionsMu.Unlock()

	go func() {
		defer cancel()
		s.execute(ctx, sess, plan, req)
	}()
	return sess.snapshot(), nil
}

func (s *securityTestService) Session(projectID uint) (*SecTestSession, bool) {
	secSessionsMu.Lock()
	sess, ok := secSessions[projectID]
	secSessionsMu.Unlock()
	if !ok {
		return nil, false
	}
	return sess.snapshot(), true
}

func (s *securityTestService) Cancel(projectID uint) error {
	secSessionsMu.Lock()
	sess, ok := secSessions[projectID]
	secSessionsMu.Unlock()
	if !ok || sess.snapshot().Status != "running" {
		return fmt.Errorf("tidak ada uji keamanan yang sedang berjalan")
	}
	sess.event("warn", "Pembatalan diminta…")
	if sess.cancel != nil {
		sess.cancel()
	}
	return nil
}

// execute menjalankan seluruh langkah uji pada sesi.
func (s *securityTestService) execute(ctx context.Context, sess *SecTestSession, plan *secTestPlan, req SecurityTestRequest) {
	in := plan.in
	sess.event("info", "— Uji keamanan %s dimulai", in.Target)

	// 1. Server: URL live, server yang sudah berjalan, atau dinyalakan oleh uji ini.
	sess.step("server", "running", "")
	base, launched := s.prepareServer(ctx, sess, plan, req)

	// 2–3. Baca kode lalu uji server (secaudit memberi tahu pergantian tahap).
	wantLive := req.Live == nil || *req.Live
	if wantLive && base != "" && ctx.Err() == nil {
		in.Live = &secaudit.LiveOptions{BaseURL: base, Burst: req.Burst, SkipBurst: req.SkipBurst, Timeout: 8 * time.Second,
			SkipAttacks: req.SkipAttacks, AuthHeader: req.AuthHeader,
			OnLog: func(level, msg string) { sess.event(level, "%s", msg) },
			OnPhase: func(phase string) {
				switch phase {
				case "attack":
					sess.step("live", "done", "")
					sess.step("attack", "running", "")
					sess.event("info", "— Menyerang endpoint: SQL/NoSQL injection, XSS, path traversal, command injection, SSTI, JWT palsu, akses tanpa login, berkas sensitif")
				case "burst":
					if sess.stepStatus("attack") == "running" {
						sess.step("attack", "done", "")
					} else {
						sess.step("live", "done", "")
					}
					sess.step("burst", "running", "")
					sess.event("info", "— Uji rate limit")
				}
			},
			OnRequest: func(method, path string, status int, latency time.Duration, err error) {
				path = shortenProbePath(path)
				if err != nil {
					sess.event("error", "%s %s → %v", method, path, err)
					return
				}
				level := "ok"
				switch {
				case status >= 500:
					level = "error"
				case status == 429:
					level = "warn"
				}
				sess.event(level, "%s %s → %d · %d ms", method, path, status, latency.Milliseconds())
			}}
	}
	in.OnStage = func(stage string) {
		switch stage {
		case "static":
			sess.step("static", "running", "")
			sess.event("info", "— Membaca router & kode dari %d folder backend", len(in.Dirs))
		case "live":
			sess.step("static", "done", "")
			if in.Live != nil {
				sess.step("live", "running", "")
				sess.event("info", "— Menguji server %s", base)
			}
		}
	}
	rep := secaudit.Run(ctx, in)
	if len(in.Dirs) > 0 {
		sess.step("static", "done", fmt.Sprintf("%d berkas · %d rute", rep.FilesRead, rep.RoutesSeen))
	} else {
		sess.step("static", "skipped", "Tidak ada folder backend")
	}
	switch {
	case in.Live == nil && !wantLive:
		sess.step("live", "skipped", "Probe server dimatikan")
		sess.step("attack", "skipped", "")
		sess.step("burst", "skipped", "")
	case in.Live == nil:
		sess.step("live", "skipped", "Tidak ada server untuk diuji")
		sess.step("attack", "skipped", "Tidak ada server untuk diserang")
		sess.step("burst", "skipped", "")
		rep.Notes = append(rep.Notes, "Tidak ada server berjalan maupun URL live untuk diuji; nyalakan server otomatis atau isi URL live untuk membuktikan kontrol di server.")
	case rep.Live != nil && rep.Live.Reachable:
		sess.step("live", "done", fmt.Sprintf("%d rute dasar", len(rep.Live.Paths)))
		if req.SkipAttacks {
			sess.step("attack", "skipped", "Dimatikan di checklist")
		} else {
			vuln, routes := 0, map[string]bool{}
			for _, at := range rep.Live.Attacks {
				routes[at.Method+" "+at.Route] = true
				if at.Verdict == secaudit.VerdictVulnerable {
					vuln++
				}
			}
			sess.step("attack", "done", fmt.Sprintf("%d serangan ke %d sasaran · %d tembus", len(rep.Live.Attacks), len(routes), vuln))
			level := "ok"
			if vuln > 0 {
				level = "error"
			}
			sess.event(level, "%d serangan dikirim, %d tembus", len(rep.Live.Attacks), vuln)
		}
		if req.SkipBurst {
			sess.step("burst", "skipped", "Dimatikan di checklist")
		} else {
			sess.step("burst", "done", "")
		}
		sess.event("ok", "Server diuji dengan %d request", rep.Live.RequestsRun)
	default:
		msg := "Server tidak terjangkau"
		if rep.Live != nil && rep.Live.Error != "" {
			msg += ": " + rep.Live.Error
		}
		sess.step("live", "failed", msg)
		sess.event("error", "%s", msg)
	}
	if plan.liveComp != nil && launched {
		s.pullLogs(sess, plan.liveComp.ID)
	}

	// 4. Matikan server yang dinyalakan uji ini (kecuali diminta tetap menyala).
	s.finishServer(sess, plan, req, launched)

	cancelled := ctx.Err() != nil
	now := time.Now()
	lastSecTestMu.Lock()
	if !cancelled {
		lastSecTest[plan.project.ID] = rep
	}
	lastSecTestMu.Unlock()

	sess.mu.Lock()
	sess.FinishedAt = &now
	if cancelled {
		sess.Status = "cancelled"
		sess.Summary = "Uji keamanan dibatalkan; hasil uji sebelumnya tetap dipakai."
	} else {
		sess.Status = "completed"
		sess.Report = rep
		sm := rep.Summary
		sess.Summary = fmt.Sprintf("Skor %d (%s) — %d gagal, %d lemah, %d lulus, %d hanya di kode.", rep.Score, rep.Grade, sm.Fail, sm.Warn, sm.Pass, sm.Declared)
	}
	for i := range sess.Steps {
		if sess.Steps[i].Status == "pending" || sess.Steps[i].Status == "running" {
			sess.Steps[i].Status = "skipped"
		}
	}
	sess.mu.Unlock()
	if cancelled {
		sess.event("warn", "— Uji dibatalkan")
	} else {
		sess.event("ok", "— Selesai: skor %d/100 dalam %s", rep.Score, time.Since(sess.StartedAt).Round(100*time.Millisecond))
	}
}

// prepareServer menentukan server yang diuji dan menyalakannya bila perlu. launched = dinyalakan uji ini.
func (s *securityTestService) prepareServer(ctx context.Context, sess *SecTestSession, plan *secTestPlan, req SecurityTestRequest) (base string, launched bool) {
	if u := strings.TrimSpace(req.BaseURL); u != "" {
		sess.setServer(func(srv *SecTestServer) {
			srv.Name, srv.Role, srv.State, srv.Source, srv.URL = "server live", "backend", "external", "external", u
		})
		sess.step("server", "done", u)
		sess.event("info", "Server live: %s", u)
		return u, false
	}
	c := plan.liveComp
	if c == nil {
		sess.step("server", "skipped", "Tidak ada komponen backend")
		sess.event("warn", "Proyek tidak punya komponen backend untuk dinyalakan")
		return "", false
	}
	sess.setServer(func(srv *SecTestServer) {
		srv.ComponentID, srv.Name, srv.Role, srv.Command = c.ID, c.Name, c.Role, c.DevCommand
	})
	if u, ok := localBaseURL(c.RunPort, c.Port); ok {
		sess.setServer(func(srv *SecTestServer) {
			srv.State, srv.Source, srv.URL = "ready", "reused", u
			srv.Note = "Sudah berjalan sebelum uji; tidak dimatikan."
		})
		sess.step("server", "done", "Sudah berjalan di "+u)
		sess.event("ok", "%s sudah berjalan di %s", c.Name, u)
		if s.runner != nil {
			if st, ok := runner.Default().Status(c.ID, false); ok && (st.State == runner.StateRunning || st.State == runner.StateStarting) {
				s.pullLogs(sess, c.ID)
			}
		}
		return u, false
	}
	if !req.StartServer {
		base, _ = localBaseURL(c.RunPort, c.Port)
		sess.step("server", "skipped", "Server mati; nyalakan otomatis tidak dipilih")
		sess.event("warn", "%s tidak berjalan dan tidak dinyalakan otomatis", c.Name)
		return base, false
	}
	if s.runner == nil {
		sess.step("server", "failed", "Runner tidak tersedia")
		return "", false
	}
	if strings.TrimSpace(c.DevCommand) == "" {
		sess.setServer(func(srv *SecTestServer) { srv.State, srv.Note = "failed", "Belum punya dev command" })
		sess.step("server", "failed", "Komponen belum punya dev command — isi di detail komponen")
		sess.event("error", "%s belum punya dev command; server tidak bisa dinyalakan", c.Name)
		return "", false
	}

	st, err := s.runner.Run(c.ID, true)
	if err != nil {
		sess.setServer(func(srv *SecTestServer) { srv.State, srv.Note = "failed", err.Error() })
		sess.step("server", "failed", err.Error())
		sess.event("error", "Gagal menyalakan %s: %v", c.Name, err)
		return "", false
	}
	sess.setServer(func(srv *SecTestServer) {
		srv.State, srv.Source, srv.Port, srv.URL, srv.PID, srv.Command = "starting", "launched", st.Port, st.URL, st.PID, st.Command
	})
	sess.event("info", "$ %s   (PORT=%d)", st.Command, st.Port)

	boot := secTestDefaultBoot
	if req.BootTimeout > 0 {
		boot = min(time.Duration(req.BootTimeout)*time.Second, secTestMaxBoot)
	}
	started := time.Now()
	deadline := started.Add(boot)
	nextNote := started.Add(10 * time.Second)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			sess.step("server", "failed", "Dibatalkan saat menunggu server")
			return "", true
		case <-tick.C:
		}
		s.pullLogs(sess, c.ID)
		cur, _ := s.runner.Status(c.ID, false)
		if cur.State == runner.StateExited || cur.State == runner.StateStopped {
			code := ""
			if cur.ExitCode != nil {
				code = fmt.Sprintf(" (exit %d)", *cur.ExitCode)
			}
			sess.setServer(func(srv *SecTestServer) { srv.State, srv.Note = "failed", "Proses keluar sebelum siap"+code })
			sess.step("server", "failed", "Proses keluar sebelum siap"+code+" — lihat log server")
			sess.event("error", "%s berhenti sebelum siap%s", c.Name, code)
			return "", false
		}
		if u, ok := localBaseURL(cur.Port, c.Port); ok {
			ms := time.Since(started).Milliseconds()
			sess.setServer(func(srv *SecTestServer) { srv.State, srv.URL, srv.BootMs = "ready", u, ms })
			sess.step("server", "done", fmt.Sprintf("Siap di %s setelah %.1f dtk", u, float64(ms)/1000))
			sess.event("ok", "%s siap di %s (%.1f dtk)", c.Name, u, float64(ms)/1000)
			return u, true
		}
		if time.Now().After(deadline) {
			sess.setServer(func(srv *SecTestServer) { srv.State, srv.Note = "failed", "Port tidak listening" })
			sess.step("server", "failed", fmt.Sprintf("Port :%d belum listening setelah %s", cur.Port, boot))
			sess.event("error", "%s belum listening di :%d setelah %s — pastikan aplikasi membaca env PORT", c.Name, cur.Port, boot)
			return "", true
		}
		if time.Now().After(nextNote) {
			sess.event("info", "Menunggu %s siap… %d dtk", c.Name, int(time.Since(started).Seconds()))
			nextNote = nextNote.Add(10 * time.Second)
		}
	}
}

// finishServer menutup langkah "Matikan server".
func (s *securityTestService) finishServer(sess *SecTestSession, plan *secTestPlan, req SecurityTestRequest, launched bool) {
	switch {
	case !launched || plan.liveComp == nil:
		detail := "Server bukan dinyalakan uji ini"
		if strings.TrimSpace(req.BaseURL) != "" {
			detail = "Server live tidak disentuh"
		}
		sess.step("stop", "skipped", detail)
	case req.KeepRunning:
		sess.setServer(func(srv *SecTestServer) { srv.KeptRunning = true; srv.Note = "Dibiarkan menyala sesuai pilihan." })
		sess.step("stop", "skipped", "Dibiarkan menyala")
		sess.event("info", "%s dibiarkan menyala", plan.liveComp.Name)
	default:
		sess.step("stop", "running", "")
		if _, err := s.runner.Stop(plan.liveComp.ID); err != nil {
			sess.step("stop", "done", err.Error())
		} else {
			sess.step("stop", "done", "Server dimatikan")
			sess.event("ok", "%s dimatikan", plan.liveComp.Name)
		}
		s.pullLogs(sess, plan.liveComp.ID)
		sess.setServer(func(srv *SecTestServer) {
			if srv.State != "failed" {
				srv.State = "stopped"
			}
		})
	}
}

// shortenProbePath memendekkan path request untuk log aktivitas: query di-decode dan dipotong agar payload
// serangan tetap terbaca tanpa membanjiri layar.
func shortenProbePath(p string) string {
	if dec, err := url.QueryUnescape(p); err == nil {
		p = dec
	}
	if r := []rune(p); len(r) > 110 {
		p = string(r[:110]) + "…"
	}
	return p
}
