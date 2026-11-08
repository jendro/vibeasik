package traffic

import (
	"time"

	"encoding/json"
	"github.com/vibeasik/vibeasik/pkg/models"
	"os"
	"path/filepath"
	"testing"
)

func checkGolden(t *testing.T, name string, v any) {
	t.Helper()
	got, _ := json.MarshalIndent(v, "", " ")
	path := filepath.Join("testdata", name+".golden.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(got) {
		t.Errorf("%s: keluaran berubah", name)
	}
}

func TestClassifyUAGolden(t *testing.T) {
	uas := []string{"", "Googlebot/2.1", "Mozilla/5.0 (compatible; bingbot/2.0)", "SomeCrawler", "Slurp", "facebookexternalhit/1.1", "HeadlessChrome/120", "Lighthouse", "UptimeRobot", "mystery spider",
		"curl/8.0", "Wget/1.2", "python-requests/2.31", "python-urllib3", "aiohttp/3", "Go-http-client/1.1", "okhttp/4", "axios/1.6", "node-fetch/2", "undici", "Dart/3", "PostmanRuntime/7", "insomnia/8", "Java/17", "GuzzleHttp/7", "MyApp CFNetwork/1", "Alamofire/5", "k6/0.4", "ApacheBench/2.3",
		"Mozilla/5.0 (Windows NT 10.0) Chrome/120 Edg/120", "Mozilla/5.0 OPR/100 Chrome/120", "Mozilla/5.0 Opera", "Mozilla/5.0 SamsungBrowser/20 Chrome/110 Mobile", "Mozilla/5.0 Firefox/120", "Mozilla/5.0 CriOS/120 iPhone", "Mozilla/5.0 Chrome/120 Safari/537", "Mozilla/5.0 Version/17 Safari/605", "Mozilla/5.0 (X11)", "Mozilla/5.0 Android", "Mozilla/5.0 iPad", "something/1.0 extra", "x"}
	type row struct{ UA, Class, Family string }
	var out []row
	for _, u := range uas {
		c, f := ClassifyUA(u)
		out = append(out, row{u, c, f})
	}
	checkGolden(t, "classify_ua", out)
}

func TestNormalizeRouteGolden(t *testing.T) {
	type in struct {
		Route, Path string
		Status      int
	}
	ins := []in{{"/api/x/:id", "/api/x/1", 200}, {" * ", "/a", 200}, {"/*", "/a/1", 200}, {"", "/zzz", 404}, {"", "/app.JS", 200}, {"", "/img/logo.png", 200},
		{"", "/api/users/123/orders/550e8400-e29b-41d4-a716-446655440000", 200}, {"", "/files/deadbeefdeadbeef0123", 200}, {"", "/t/abc123def456ghi789jkl012mno345pq", 200},
		{"", "/u/" + string(make([]byte, 0)) + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 200}, {"", "/a/b/c/d/e/f/g/h/i/j/k", 200}, {"", "/", 200}, {"", "//a//b", 200}, {"", "/hello-world", 200}}
	type row struct {
		In  in
		Out string
	}
	var out []row
	for _, i := range ins {
		out = append(out, row{i, NormalizeRoute(i.Route, i.Path, i.Status)})
	}
	checkGolden(t, "normalize_route", out)
}

func TestSuggestKitGolden(t *testing.T) {
	type row struct{ L, F, Out string }
	var out []row
	for _, l := range []string{"", "go", "Go", "python", "php", "typescript", "javascript", "rust"} {
		for _, f := range []string{"", "fiber", "Nuxt 3", "nitro", "fastapi", "Flask", "django", "laravel", "express", "nestjs", "gin"} {
			out = append(out, row{l, f, SuggestKit(l, f)})
		}
	}
	checkGolden(t, "suggest_kit", out)
}

func TestIngestGolden(t *testing.T) {
	frozen := time.Date(2026, 1, 1, 12, 0, 30, 0, time.UTC)
	timeNow = func() time.Time { return frozen }
	defer func() { timeNow = time.Now }()
	db := testDB(t)
	src := models.TrafficSource{Name: "ig", Mode: models.TrafficModeDirect, IngestKey: "ig", Enabled: true}
	db.Create(&src)
	ms := frozen.UnixMilli()
	evs := []Event{
		{TS: ms - 5000, Method: "get", Path: "/api/items/12?x=1", Status: 200, DurMS: 12, IP: "9.9.9.9", UA: "Mozilla/5.0 Chrome/120 Safari/537", Referer: "https://Example.com/p?q=1", Country: "id", BytesIn: 10, BytesOut: 200, Instance: "web-1"},
		{TS: ms - 4000, Method: "GET", Path: "/api/items/13", Status: 200, DurMS: 300, IP: "9.9.9.9", UA: "curl/8", BytesOut: -5},
		{TS: ms - 3000, Method: "POST", Path: "/api/auth/login", Status: 401, DurMS: 40, IP: "8.8.8.8", UA: "python-requests/2"},
		{TS: ms - 2000, Method: "", Path: "/nothing", Status: 404, DurMS: -1, IP: "8.8.8.8", UA: "Googlebot"},
		{TS: ms - 1000, Method: "GET", Path: "/boom", Route: "/boom", Status: 503, DurMS: 6000, IP: "7.7.7.7"},
		{TS: 0, Method: "GET", Path: "/zero", Status: 200, DurMS: 1},
		{TS: ms + int64(time.Hour/time.Millisecond), Method: "GET", Path: "/future", Status: 200, DurMS: 1},
		{TS: 1000, Method: "GET", Path: "/ancient.css", Status: 200, DurMS: 1},
	}
	if _, err := Ingest(db, &src, evs, "default-inst"); err != nil {
		t.Fatal(err)
	}
	if _, err := Ingest(db, &src, evs[:3], "default-inst"); err != nil {
		t.Fatal(err)
	}
	var routes []models.TrafficRouteStat
	db.Order("gran, bucket, method, route").Find(&routes)
	var dims []models.TrafficDimStat
	db.Order("gran, bucket, dim, value").Find(&dims)
	var recent []models.TrafficRecentEvent
	db.Order("id").Find(&recent)
	for i := range recent {
		recent[i].SourceID = 0
	}
	var s2 models.TrafficSource
	db.First(&s2, src.ID)
	checkGolden(t, "ingest", map[string]any{"routes": routes, "dims": dims, "recent": recent, "total": s2.TotalEvents, "inst": s2.LastInstance, "last": s2.LastEventAt})
}
