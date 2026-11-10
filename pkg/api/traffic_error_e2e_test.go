package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/traffic"
)

// Uji ujung-ke-ujung event error & heartbeat: aplikasi contoh yang sengaja error (testdata/errapps) dijalankan sebagai
// proses nyata dengan SDK-nya, mengirim ke endpoint ingest engine yang sungguhan (listener lokal), lalu grup error,
// status proses, dan hitungan request diperiksa di database. Stack tanpa runtime-nya di mesin akan di-skip dengan pesan jelas.

const (
	e2eRelease = "rel-test-1"
	e2eRequest = 5 // 3x /boom + 2x /ok per aplikasi contoh
)

var e2eSecrets = []string{"SECRETQ", "SECRETH", "SECRETC", "SECRETB", "SECRETA", "hunter2"}

type e2eEngine struct {
	s   *Server
	url string
	key string
	src models.TrafficSource
}

func startE2EEngine(t *testing.T) *e2eEngine {
	t.Helper()
	s := newGuardTestServer(t)
	if err := s.db.AutoMigrate(&models.TrafficSource{}, &models.TrafficRouteStat{}, &models.TrafficDimStat{}, &models.TrafficRecentEvent{},
		&models.TrafficErrorGroup{}, &models.TrafficProcess{}); err != nil {
		t.Fatal(err)
	}
	key := "vtd_e2e_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	src := models.TrafficSource{Name: "contoh-error", Mode: models.TrafficModeDirect, IngestKey: key, Enabled: true}
	if err := s.db.Create(&src).Error; err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.app.Listener(ln) }()
	t.Cleanup(func() { _ = s.app.Shutdown() })
	return &e2eEngine{s: s, url: "http://" + ln.Addr().String() + "/api/traffic/ingest", key: key, src: src}
}

func (e *e2eEngine) env(extra ...string) []string {
	sdk, _ := filepath.Abs("../traffic/sdk")
	return append(os.Environ(), append([]string{
		"VIBEASIK_TRAFFIC_URL=" + e.url, "VIBEASIK_TRAFFIC_KEY=" + e.key, "VIBEASIK_TRAFFIC_FLUSH_SEC=1",
		"VIBEASIK_RELEASE=" + e2eRelease, "VIBEASIK_SDK_DIR=" + sdk,
	}, extra...)...)
}

func runApp(t *testing.T, e *e2eEngine, wantFail bool, name string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = e.env()
	out, err := cmd.CombinedOutput()
	if (err != nil) != wantFail {
		t.Fatalf("%s %v: err=%v (wantFail=%v)\n%s", name, args, err, wantFail, out)
	}
}

func needTool(t *testing.T, tool string) {
	t.Helper()
	if _, err := exec.LookPath(tool); err != nil {
		t.Skipf("SKIP: runtime %q tidak ada di mesin ini; stack ini tidak bisa dibuktikan ujung-ke-ujung", tool)
	}
}

// verifyOneGroup: tepat satu grup error dengan tipe & frame teratas benar, tanpa data sensitif di database.
func (e *e2eEngine) verifyOneGroup(t *testing.T, wantType, wantTop string, wantErrors int64, wantRequests int64) {
	t.Helper()
	db := e.s.db
	groups, more, err := traffic.ErrorGroups(db, e.src.ID, "", 50, 0)
	if err != nil || more || len(groups) != 1 {
		t.Fatalf("harus tepat 1 grup error, dapat %d (err=%v)", len(groups), err)
	}
	g := groups[0]
	if g.Count != wantErrors || g.ErrType != wantType || !strings.Contains(g.TopFrame, wantTop) || g.Side != "server" || g.LastRelease != e2eRelease {
		t.Errorf("grup salah: count=%d type=%q top=%q side=%q release=%q; want count=%d type=%q top~%q", g.Count, g.ErrType, g.TopFrame, g.Side, g.LastRelease, wantErrors, wantType, wantTop)
	}
	t.Logf("grup: type=%s count=%d top=%s route=%s", g.ErrType, g.Count, g.TopFrame, g.Route)
	assertNoSecrets(t, db)
	if wantRequests >= 0 {
		assertRequestCount(t, db, e.src.ID, wantRequests)
	}
}

func assertRequestCount(t *testing.T, db *gorm.DB, sourceID uint, want int64) {
	t.Helper()
	var sum int64
	db.Model(&models.TrafficRouteStat{}).Where("source_id = ? AND gran = ?", sourceID, models.TrafficGranHour).Select("COALESCE(SUM(count),0)").Scan(&sum)
	var src models.TrafficSource
	db.First(&src, sourceID)
	if sum != want || src.TotalEvents != want {
		t.Errorf("hitungan request: statistik rute=%d total_events=%d, want %d (error/heartbeat tidak boleh dihitung)", sum, src.TotalEvents, want)
	}
}

func assertNoSecrets(t *testing.T, db *gorm.DB) {
	t.Helper()
	var dump strings.Builder
	for _, rows := range []any{new([]models.TrafficErrorGroup), new([]models.TrafficProcess), new([]models.TrafficRecentEvent), new([]models.TrafficDimStat), new([]models.TrafficRouteStat)} {
		if err := db.Find(rows).Error; err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(rows)
		dump.Write(b)
	}
	for _, s := range e2eSecrets {
		if strings.Contains(dump.String(), s) {
			t.Errorf("data sensitif %q bocor ke database", s)
		}
	}
	var g models.TrafficErrorGroup
	db.First(&g)
	if g.Stack == "" || !strings.Contains(g.Stack, "\n") {
		t.Errorf("stack contoh kosong atau bukan banyak frame: %q", g.Stack)
	}
}

func (e *e2eEngine) verifyHeartbeat(t *testing.T) {
	t.Helper()
	procs, err := traffic.Processes(e.s.db, e.src.ID)
	if err != nil || len(procs) == 0 {
		t.Fatalf("heartbeat tidak tercatat (err=%v)", err)
	}
	p := procs[0]
	if p.Release != e2eRelease || p.MemBytes <= 0 || p.UptimeS < 0 || !p.Alive {
		t.Errorf("heartbeat salah: %+v", p)
	}
	t.Logf("heartbeat: instance=%s release=%s mem=%d uptime=%ds", p.Instance, p.Release, p.MemBytes, p.UptimeS)
}

func TestE2EErrorGoNetHTTP(t *testing.T) {
	needTool(t, "go")
	e := startE2EEngine(t)
	runApp(t, e, false, "go", "run", "./testdata/errapps/gohttp")
	e.verifyOneGroup(t, "*main.DBError", "main.loadUser", 3, e2eRequest)
	e.verifyHeartbeat(t)
}

func TestE2EErrorGoFiber(t *testing.T) {
	needTool(t, "go")
	e := startE2EEngine(t)
	runApp(t, e, false, "go", "run", "./testdata/errapps/gofiber")
	e.verifyOneGroup(t, "*main.DBError", "main.loadUser", 3, e2eRequest)
	e.verifyHeartbeat(t)
}

func TestE2EErrorExpress(t *testing.T) {
	needTool(t, "node")
	e := startE2EEngine(t)
	runApp(t, e, false, "node", "testdata/errapps/express.cjs")
	e.verifyOneGroup(t, "DbError", "loadUser", 3, e2eRequest)
	e.verifyHeartbeat(t)
}

// Proses mati karena uncaughtException: event error harus tetap terkirim (perilaku crash bawaan Node tidak diubah).
func TestE2EErrorExpressUncaughtCrash(t *testing.T) {
	needTool(t, "node")
	e := startE2EEngine(t)
	runApp(t, e, true, "node", "testdata/errapps/express-crash.cjs")
	e.verifyOneGroup(t, "CrashError", "crashNow", 1, -1)
}

func TestE2EErrorNitro(t *testing.T) {
	needTool(t, "node")
	if out, err := exec.Command("node", "-p", "process.versions.node.split('.').map(Number).reduce((a,b,i)=>a+(i<2?b/(i?1000:1):0),0)>=23.006").Output(); err != nil || strings.TrimSpace(string(out)) != "true" {
		t.Skip("SKIP: Node < 23.6 tidak bisa menjalankan berkas .ts (type stripping); plugin Nitro tidak bisa dibuktikan ujung-ke-ujung")
	}
	e := startE2EEngine(t)
	dir := t.TempDir()
	plugin, err := os.ReadFile("../traffic/sdk/node/vibeasik-traffic.nitro.ts")
	if err != nil {
		t.Fatal(err)
	}
	stub, _ := os.ReadFile("testdata/errapps/nitro-h3-stub.cjs")
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(dir, "node_modules/h3"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "node_modules/h3/index.js"), stub, 0o644))
	must(os.WriteFile(filepath.Join(dir, "node_modules/h3/package.json"), []byte(`{"name":"h3","main":"index.js"}`), 0o644))
	must(os.WriteFile(filepath.Join(dir, "plugin.ts"), plugin, 0o644))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "testdata/errapps/nitro/harness.mjs")
	cmd.Env = e.env("VIBEASIK_PLUGIN=" + filepath.Join(dir, "plugin.ts"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("nitro harness: %v\n%s", err, out)
	}
	e.verifyOneGroup(t, "DbError", "loadUser", 3, e2eRequest)
	e.verifyHeartbeat(t)
}

func TestE2EErrorPythonASGI(t *testing.T) {
	needTool(t, "python3")
	e := startE2EEngine(t)
	runApp(t, e, false, "python3", "-I", "testdata/errapps/asgi_app.py")
	e.verifyOneGroup(t, "__main__.DbError", "load_user", 3, e2eRequest)
	e.verifyHeartbeat(t)
}

func TestE2EErrorPythonWSGI(t *testing.T) {
	needTool(t, "python3")
	e := startE2EEngine(t)
	runApp(t, e, false, "python3", "-I", "testdata/errapps/wsgi_app.py")
	e.verifyOneGroup(t, "__main__.DbError", "load_user", 3, e2eRequest)
	e.verifyHeartbeat(t)
}

func TestE2EErrorLaravelPHP(t *testing.T) {
	needTool(t, "php")
	e := startE2EEngine(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	router, _ := filepath.Abs("testdata/errapps/laravel/router.php")
	srv := exec.CommandContext(ctx, "php", "-d", "zend.exception_ignore_args=0", "-S", addr, router)
	srv.Env = e.env()
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cancel(); _ = srv.Wait() }()
	waitHTTP(t, "http://"+addr+"/ok")
	// antrean PHP dikirim paling cepat tiap FLUSH_SEC (1 dtk): beri jeda lalu satu request penutup memicu pengiriman.
	for _, p := range []string{"/boom/11", "/boom/22", "/boom/33", "/ok"} {
		phpHit(t, addr, p)
	}
	time.Sleep(1200 * time.Millisecond)
	phpHit(t, addr, "/ok")
	// request pemanasan + 1 /ok sebelum boom + 1 penutup: 3 boom + 3 ok = 6 request
	e.verifyOneGroup(t, "DbException", "loadUser", 3, 6)
	e.verifyHeartbeat(t)
}

func waitHTTP(t *testing.T, url string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		if resp, err := http.Get(url); err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server %s tidak naik", url)
}

func phpHit(t *testing.T, addr, p string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, "http://"+addr+p+"?token=SECRETQ", strings.NewReader(`{"password":"SECRETB"}`))
	req.Header.Set("Authorization", "Bearer SECRETH")
	req.Header.Set("Cookie", "sid=SECRETC")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}

// Snippet browser: kunci di badan (text/plain, tanpa preflight), hanya event error sisi browser yang diterima.
func TestE2EErrorBrowserSnippet(t *testing.T) {
	needTool(t, "node")
	e := startE2EEngine(t)
	runApp(t, e, false, "node", "testdata/errapps/browser-harness.cjs")
	db := e.s.db
	var g models.TrafficErrorGroup
	if err := db.Where("source_id = ? AND err_type = ?", e.src.ID, "CartError").First(&g).Error; err != nil {
		t.Fatalf("grup CartError tidak ada: %v", err)
	}
	if g.Count != 3 || g.Side != "browser" || g.Route != "/checkout" || !strings.Contains(g.TopFrame, "renderCart") || g.LastRelease != e2eRelease {
		t.Errorf("grup browser salah: %+v", g)
	}
	var total int64
	db.Model(&models.TrafficErrorGroup{}).Select("COALESCE(SUM(count),0)").Where("source_id = ?", e.src.ID).Scan(&total)
	// halaman 1-2: 1 event; halaman 3: batas 20 event per halaman (CartError + TypeError + RangeError sampai penuh)
	if total != 1+1+20 {
		t.Errorf("total event error = %d, want 22 (dedup + maksimum 20 per halaman)", total)
	}
	assertNoSecrets(t, db)
	assertRequestCount(t, db, e.src.ID, 0)
}

func TestIngestBodyKeyOnlyAcceptsBrowserErrors(t *testing.T) {
	e := startE2EEngine(t)
	body := `{"key":"` + e.key + `","events":[{"kind":"heartbeat","release":"x","mem_bytes":5},{"path":"/pwn","status":200},` +
		`{"kind":"error","side":"server","err_type":"E","err_msg":"m","stack":"f (a.js:1)","route":"/p","path":"/p"}]}`
	resp, err := http.Post(e.url, "text/plain", strings.NewReader(body))
	if err != nil || resp.StatusCode != 202 {
		t.Fatalf("ingest: %v %v", err, resp)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("ingest harus memberi Access-Control-Allow-Origin agar snippet browser bisa mengirim")
	}
	_ = resp.Body.Close()
	var g models.TrafficErrorGroup
	if e.s.db.First(&g).Error != nil || g.Side != "browser" {
		t.Errorf("event error harus dipaksa sisi browser: %+v", g)
	}
	var procs int64
	e.s.db.Model(&models.TrafficProcess{}).Count(&procs)
	assertRequestCount(t, e.s.db, e.src.ID, 0)
	if procs != 0 {
		t.Error("heartbeat via kunci di badan harus ditolak (kunci tampil di JS publik)")
	}
	bad, _ := http.Post(e.url, "text/plain", strings.NewReader(`{"key":"salah","events":[]}`))
	if bad.StatusCode != 401 {
		t.Errorf("kunci salah harus 401, dapat %d", bad.StatusCode)
	}
	_ = bad.Body.Close()
}

func TestTrafficErrorReadEndpoints(t *testing.T) {
	e := startE2EEngine(t)
	now := time.Now().UnixMilli()
	post := func(ev string) {
		req, _ := http.NewRequest(http.MethodPost, e.url, strings.NewReader(`{"events":[`+ev+`]}`))
		req.Header.Set("X-Vibeasik-Key", e.key)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}
	ts := strconv.FormatInt(now, 10)
	post(`{"kind":"error","ts":` + ts + `,"err_type":"E1","stack":"f (a.go:1)\ng (b.go:2)"},{"kind":"heartbeat","ts":` + ts + `,"instance":"i1","release":"r1","mem_bytes":10,"uptime_s":3}`)
	get := func(path string) map[string]any {
		req, _ := http.NewRequest(http.MethodGet, strings.TrimSuffix(e.url, "/traffic/ingest")+path, nil)
		req.Header.Set(localauth.Header, testLocalToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		out["_status"] = resp.StatusCode
		return out
	}
	id := strconv.Itoa(int(e.src.ID))
	list := get("/traffic/sources/" + id + "/errors")
	groups, _ := list["groups"].([]any)
	if len(groups) != 1 {
		t.Fatalf("daftar grup: %v", list)
	}
	first := groups[0].(map[string]any)
	if _, has := first["stack"]; has {
		t.Error("daftar grup tidak boleh membawa stack (kolom berat)")
	}
	detail := get("/traffic/sources/" + id + "/errors/" + strconv.Itoa(int(first["id"].(float64))))
	if !strings.Contains(detail["stack"].(string), "g (b.go") {
		t.Errorf("detail tanpa stack: %v", detail)
	}
	procs, _ := get("/traffic/sources/" + id + "/processes")["processes"].([]any)
	if len(procs) != 1 || procs[0].(map[string]any)["alive"] != true {
		t.Errorf("status proses salah: %v", procs)
	}
	if st := get("/traffic/sources/999/errors")["_status"]; st != float64(404) && st != 404 {
		t.Errorf("sumber tak ada harus 404, dapat %v", st)
	}
}
