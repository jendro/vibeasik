package traffic

import (
	"gorm.io/gorm"
	"strings"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func errTestDB(t *testing.T) (*gorm.DB, *models.TrafficSource, func(...Event)) {
	t.Helper()
	db := testDB(t)
	if err := db.AutoMigrate(&models.TrafficErrorGroup{}, &models.TrafficProcess{}); err != nil {
		t.Fatal(err)
	}
	src := models.TrafficSource{Name: "uji-error", Mode: models.TrafficModeDirect, IngestKey: "k", Enabled: true}
	db.Create(&src)
	return db, &src, func(evs ...Event) {
		if _, err := Ingest(db, &src, evs, "inst-a"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNormalizeFrame(t *testing.T) {
	cases := map[string]string{
		"main.loadUser (/srv/x/proj/cmd/main.go:42)":                       "main.loadUser (cmd/main.go)",
		"main.loadUser (/srv/other/cmd/main.go:99)":                         "main.loadUser (cmd/main.go)",
		"at handler (https://app.example.com/_nuxt/entry.3f2a1b9c.js:10:5)": "handler (_nuxt/entry.js)",
		"at fn (/app/node_modules/express/lib/router.js:284:15)":            "fn (node_modules/express/lib/router.js)",
		"File \"/app/svc/views.py\", line 12, in get":                       "File \"svc/views.py\", in get",
		"at loadUser (file:///srv/app/dist/server.mjs:7:9)":                 "loadUser (dist/server.mjs)",
		"runtime.call (0xc000123456)":                                       "runtime.call (0x?)",
	}
	for in, want := range cases {
		if got := NormalizeFrame(in); got != want {
			t.Errorf("NormalizeFrame(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFingerprintIgnoresLineAndMessage(t *testing.T) {
	a := Fingerprint("*main.DBError", "vibeasiktraffic.x (a.go:1)\nmain.loadUser (/a/b/main.go:10)", "id=1")
	b := Fingerprint("*main.DBError", "main.loadUser (/z/b/main.go:77)", "id=2")
	if a != b {
		t.Error("sidik jari harus sama walau nomor baris, path absolut, dan pesan berbeda")
	}
	if c := Fingerprint("*main.Other", "main.loadUser (/z/b/main.go:77)", ""); c == a {
		t.Error("tipe berbeda harus grup berbeda")
	}
	if Fingerprint("E", "", "user 12 gagal") != Fingerprint("E", "", "user 99 gagal") {
		t.Error("tanpa stack, angka di pesan diseragamkan")
	}
}

func TestIngestErrorGroupsAndHeartbeat(t *testing.T) {
	db, src, ingest := errTestDB(t)
	now := time.Now().UnixMilli()
	ingest(Event{TS: now, Method: "GET", Path: "/ok", Status: 200, DurMS: 3, IP: "1.1.1.1"})
	for i := 0; i < 3; i++ {
		ingest(Event{Kind: KindError, TS: now + int64(i), Method: "get", Route: "/boom", Path: "/boom", Status: 500,
			ErrType: "*main.DBError", ErrMsg: "db down", Release: "v" + string(rune('1'+i)), Side: "server",
			Stack: "main.loadUser (/x/main.go:" + string(rune('1'+i)) + ")\nmain.handler (/x/main.go:20)"})
	}
	ingest(Event{Kind: KindHeartbeat, TS: now, Release: "v1", MemBytes: 1234, UptimeS: 7, Instance: "inst-a"})
	ingest(Event{Kind: KindHeartbeat, TS: now + 5, Release: "v2", MemBytes: 2000, UptimeS: 12, Instance: "inst-a"})

	groups, more, err := ErrorGroups(db, src.ID, "", 50, 0)
	if err != nil || more || len(groups) != 1 {
		t.Fatalf("grup=%d more=%v err=%v", len(groups), more, err)
	}
	g := groups[0]
	if g.Count != 3 || g.ErrType != "*main.DBError" || !strings.HasPrefix(g.TopFrame, "main.loadUser") || g.FirstRelease != "v1" || g.LastRelease != "v3" {
		t.Errorf("grup salah: %+v", g)
	}
	procs, _ := Processes(db, src.ID)
	if len(procs) != 1 || procs[0].Release != "v2" || procs[0].MemBytes != 2000 || procs[0].UptimeS != 12 || !procs[0].Alive {
		t.Errorf("proses salah: %+v", procs)
	}
	rep, _ := BuildReport(db, *src, ParseWindow("1h"))
	if rep.Totals.Requests != 1 {
		t.Errorf("event error/heartbeat tidak boleh dihitung sebagai request: %d", rep.Totals.Requests)
	}
}

func TestBrowserEventsOnly(t *testing.T) {
	out := BrowserEventsOnly([]Event{{Kind: KindHeartbeat}, {Path: "/x"}, {Kind: KindError, Side: "server", Instance: "x"}})
	if len(out) != 1 || out[0].Side != SideBrowser || out[0].Instance != "" {
		t.Errorf("hasil salah: %+v", out)
	}
}
