package traffic

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.TrafficSource{}, &models.TrafficRouteStat{}, &models.TrafficDimStat{}, &models.TrafficRecentEvent{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestNormalizeRoute(t *testing.T) {
	cases := map[string]string{
		"/api/users/42": "/api/users/:id",
		"/api/orders/550e8400-e29b-41d4-a716-446655440000/items": "/api/orders/:uuid/items",
		"/files/abcdef0123456789abcdef":                          "/files/:hash",
		"/assets/app.3f2a.js":                                    "static:*.js",
		"/health":                                                "/health",
	}
	for in, want := range cases {
		if got := NormalizeRoute("", in, 200); got != want {
			t.Errorf("NormalizeRoute(%q) = %q, want %q", in, got, want)
		}
	}
	if got := NormalizeRoute("", "/.env", 404); got != RouteNotFound {
		t.Errorf("404 tanpa pola rute harus digabung, dapat %q", got)
	}
	if got := NormalizeRoute("/users/:id", "/users/7", 200); got != "/users/:id" {
		t.Errorf("pola rute SDK harus diutamakan, dapat %q", got)
	}
	if g := RouteGroup("/api/v1/reports/:id/export"); g != "/reports" {
		t.Errorf("RouteGroup = %q", g)
	}
}

func TestPercentile(t *testing.T) {
	h := make([]int64, len(LatencyBounds)+1)
	h[histIndex(8)] = 90   // 5–10 ms
	h[histIndex(800)] = 10 // 500–1000 ms
	if p50 := Percentile(h, 0.5, 900); p50 < 5 || p50 > 10 {
		t.Errorf("p50 = %v, mau di ember 5–10 ms", p50)
	}
	if p95 := Percentile(h, 0.95, 900); p95 < 500 || p95 > 900 {
		t.Errorf("p95 = %v, mau di 500–900 ms", p95)
	}
}

func TestIngestReportAdvise(t *testing.T) {
	db := testDB(t)
	src := models.TrafficSource{Name: "uji", Mode: models.TrafficModeDirect, IngestKey: "k", Enabled: true}
	db.Create(&src)

	now := time.Now().Add(-10 * time.Minute).UnixMilli()
	var events []Event
	for i := 0; i < 600; i++ {
		ts := now + int64(i)*500
		events = append(events,
			Event{TS: ts, Method: "GET", Path: fmt.Sprintf("/api/products/%d", i%5), Status: 200, DurMS: 120, IP: fmt.Sprintf("10.0.0.%d", i%40), UA: "Mozilla/5.0 Chrome/120"},
			Event{TS: ts, Method: "GET", Path: "/api/reports/export", Status: 200, DurMS: 7000, IP: "10.0.1.1", UA: "Mozilla/5.0 Chrome/120"},
		)
		if i%3 == 0 {
			events = append(events, Event{TS: ts, Method: "POST", Path: "/api/auth/login", Status: 401, DurMS: 40, IP: "6.6.6.6", UA: "python-requests/2.31"})
		}
	}
	for i := 0; i < len(events); i += 1000 {
		end := min(i+1000, len(events))
		if _, err := Ingest(db, &src, events[i:end], "api-1"); err != nil {
			t.Fatal(err)
		}
	}

	rep, err := BuildReport(db, src, ParseWindow("1h"))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Totals.Requests != int64(len(events)) {
		t.Fatalf("requests = %d, want %d", rep.Totals.Requests, len(events))
	}
	var products *RouteRow
	for i := range rep.Routes {
		if rep.Routes[i].Route == "/api/products/:id" {
			products = &rep.Routes[i]
		}
	}
	if products == nil || products.Count != 600 || products.HotShare < 0.99 {
		t.Fatalf("rute produk salah: %+v", products)
	}

	rules := map[string]bool{}
	for _, r := range rep.Recommendations {
		rules[r.Rule] = true
	}
	for _, want := range []string{"TRAFFIC-RATE-LIMIT-AUTH", "TRAFFIC-CACHE-ROUTE", "TRAFFIC-LONG-RUNNING"} {
		if !rules[want] {
			t.Errorf("saran %s tidak muncul; dapat %v", want, rules)
		}
	}
	if rules["TRAFFIC-HEALTHY"] {
		t.Error("ada masalah nyata, saran 'sehat' tidak boleh muncul")
	}

	// Ingest kedua di menit yang sama harus menggabung baris, bukan menggandakan.
	before := rep.Totals.Requests
	if _, err := Ingest(db, &src, events[:10], "api-1"); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&models.TrafficRouteStat{}).Where("source_id = ? AND gran = 'h' AND route = ?", src.ID, "/api/products/:id").Count(&n)
	if n > 2 {
		t.Errorf("baris per jam tidak digabung: %d", n)
	}
	rep2, _ := BuildReport(db, src, ParseWindow("1h"))
	if rep2.Totals.Requests != before+10 {
		t.Errorf("total setelah ingest kedua = %d, want %d", rep2.Totals.Requests, before+10)
	}
}

func TestSDKKitsEmbedded(t *testing.T) {
	for _, k := range Kits("example.com/app") {
		if len(k.Files) == 0 {
			t.Fatalf("kit %s tanpa berkas", k.ID)
		}
		for _, f := range k.Files {
			if !strings.Contains(f.Content, "VIBEASIK_TRAFFIC") && !strings.Contains(f.Content, "Enabled()") {
				t.Errorf("berkas %s di kit %s tidak terbaca dari embed", f.Path, k.ID)
			}
		}
	}
	if SuggestKit("go", "fiber") != "go-fiber" || SuggestKit("typescript", "nuxt") != "nitro" || SuggestKit("php", "laravel") != "laravel" {
		t.Error("SuggestKit salah")
	}
}

func TestPrune(t *testing.T) {
	db := testDB(t)
	src := models.TrafficSource{Name: "prune-test", Mode: models.TrafficModeDirect, IngestKey: "pk", Enabled: true}
	if err := db.Create(&src).Error; err != nil {
		t.Fatalf("gagal membuat source: %v", err)
	}

	now := time.Now()
	oldMinBucket := now.Add(-MinuteRetention - time.Hour).Unix()
	newMinBucket := now.Add(-time.Hour).Unix()
	oldHourBucket := now.Add(-HourRetention - 24*time.Hour).Unix()
	newHourBucket := now.Add(-24 * time.Hour).Unix()

	routes := []models.TrafficRouteStat{
		{SourceID: src.ID, Gran: models.TrafficGranMinute, Bucket: oldMinBucket, Method: "GET", Route: "/old/min", Count: 10},
		{SourceID: src.ID, Gran: models.TrafficGranMinute, Bucket: newMinBucket, Method: "GET", Route: "/new/min", Count: 20},
		{SourceID: src.ID, Gran: models.TrafficGranHour, Bucket: oldHourBucket, Method: "GET", Route: "/old/hour", Count: 30},
		{SourceID: src.ID, Gran: models.TrafficGranHour, Bucket: newHourBucket, Method: "GET", Route: "/new/hour", Count: 40},
	}
	for i := range routes {
		if err := db.Create(&routes[i]).Error; err != nil {
			t.Fatalf("gagal membuat TrafficRouteStat: %v", err)
		}
	}

	dims := []models.TrafficDimStat{
		{SourceID: src.ID, Gran: models.TrafficGranMinute, Bucket: oldMinBucket, Dim: "status", Value: "200", Count: 10},
		{SourceID: src.ID, Gran: models.TrafficGranMinute, Bucket: newMinBucket, Dim: "status", Value: "200", Count: 20},
		{SourceID: src.ID, Gran: models.TrafficGranHour, Bucket: oldHourBucket, Dim: "status", Value: "200", Count: 30},
		{SourceID: src.ID, Gran: models.TrafficGranHour, Bucket: newHourBucket, Dim: "status", Value: "200", Count: 40},
	}
	for i := range dims {
		if err := db.Create(&dims[i]).Error; err != nil {
			t.Fatalf("gagal membuat TrafficDimStat: %v", err)
		}
	}

	Prune(db)

	var remainingRoutes []models.TrafficRouteStat
	if err := db.Find(&remainingRoutes).Error; err != nil {
		t.Fatalf("gagal query TrafficRouteStat: %v", err)
	}
	if len(remainingRoutes) != 2 {
		t.Fatalf("sisa TrafficRouteStat = %d, ingin 2", len(remainingRoutes))
	}
	for _, r := range remainingRoutes {
		if r.Route == "/old/min" || r.Route == "/old/hour" {
			t.Errorf("rute usang %s masih ada setelah Prune", r.Route)
		}
	}

	var remainingDims []models.TrafficDimStat
	if err := db.Find(&remainingDims).Error; err != nil {
		t.Fatalf("gagal query TrafficDimStat: %v", err)
	}
	if len(remainingDims) != 2 {
		t.Fatalf("sisa TrafficDimStat = %d, ingin 2", len(remainingDims))
	}
	for _, d := range remainingDims {
		if d.Bucket == oldMinBucket || d.Bucket == oldHourBucket {
			t.Errorf("dimensi usang dengan bucket %d masih ada setelah Prune", d.Bucket)
		}
	}
}

