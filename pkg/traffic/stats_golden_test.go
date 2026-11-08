package traffic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Mengunci keluaran BuildReport dengan waktu beku dan data sintetis.
func TestBuildReportGolden(t *testing.T) {
	frozen := time.Date(2026, 1, 1, 12, 0, 30, 0, time.UTC)
	timeNow = func() time.Time { return frozen }
	defer func() { timeNow = time.Now }()

	for _, win := range []string{"1h", "7d"} {
		t.Run(win, func(t *testing.T) {
			db := testDB(t)
			src := models.TrafficSource{Name: "g" + win, Mode: models.TrafficModeDirect, IngestKey: "k" + win, Enabled: true}
			db.Create(&src)
			w := ParseWindow(win)
			gran := w.Gran
			step := int64(60)
			if gran == models.TrafficGranHour {
				step = 3600
			}
			base := frozen.Unix()/step*step - 10*step
			hist := func(a, b, c int64) string { return formatHist([]int64{a, b, c, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) }
			routes := []struct {
				m, r string
				n    int64
			}{{"GET", "/api/products/:id", 40}, {"POST", "/api/auth/login", 7}, {"GET", "static:js", 3}, {"HEAD", "/health", 2}, {"POST", "/api/orders", 11}}
			for i := int64(0); i < 8; i++ {
				for k, r := range routes {
					n := r.n * (i + 1)
					db.Create(&models.TrafficRouteStat{SourceID: src.ID, Gran: gran, Bucket: base + i*step, Method: r.m, Route: r.r,
						Count: n, Err4xx: n / 5, Err5xx: n / int64(7+k), SumMS: float64(n) * float64(30+k*17), MaxMS: float64(200 + 90*k),
						SumIn: n * 100, SumOut: n * int64(1000*(k+1)), Hist: hist(n/2, n/3, n-n/2-n/3)})
				}
			}
			dims := []struct {
				dim, val string
				n, e     int64
			}{
				{"status", "200", 500, 0}, {"status", "401", 40, 40}, {"country", "ID", 400, 3}, {"country", "US", 100, 9},
				{"referer", "google.com", 60, 0}, {"ua", "bot", 70, 0}, {"ua", "browser", 400, 12}, {"client", "Chrome", 380, 5},
				{"instance", "a", 300, 2}, {"instance", "b", 200, 4}, {"notfound", "/.env", 25, 25}, {"notfound", "/zzz", 4, 4},
				{"ip", "1.1.1.1", 250, 10}, {"ip", "2.2.2.2", 150, 1}, {"ip", "3.3.3.3", 50, 0},
				{"auth_ip", "1.1.1.1", 30, 0}, {"gpath", "/api/products/:id|/api/products/1", 90, 0}, {"gpath", "/api/products/:id|/api/products/2", 10, 0},
			}
			for _, d := range dims {
				db.Create(&models.TrafficDimStat{SourceID: src.ID, Gran: models.TrafficGranHour, Bucket: base / 3600 * 3600, Dim: d.dim, Value: d.val, Count: d.n, Err: d.e})
			}
			for _, d := range []struct {
				dim, val string
				n        int64
			}{{"ip", "1.1.1.1", 90}, {"ip", "2.2.2.2", 15}, {"ip", "3.3.3.3", 5}, {"auth_ip", "1.1.1.1", 12}} {
				db.Create(&models.TrafficDimStat{SourceID: src.ID, Gran: models.TrafficGranMinute, Bucket: base, Dim: d.dim, Value: d.val, Count: d.n})
			}
			rep, err := BuildReport(db, src, w)
			if err != nil {
				t.Fatal(err)
			}
			rep.Source = models.TrafficSource{} // ID/stempel waktu tidak stabil
			got, _ := json.MarshalIndent(struct {
				*Report
				Minutes   []Point
				IPPeaks   []int64
				ScannerNF int64
			}{rep, rep.minutes, rep.ipPeaks, rep.scannerNF}, "", " ")
			path := filepath.Join("testdata", fmt.Sprintf("report_%s.golden.json", win))
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
				t.Error("output BuildReport berubah")
			}
		})
	}
}
