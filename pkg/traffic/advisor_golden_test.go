package traffic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func goldenReports() map[string]*Report {
	busy := &Report{Gran: "m", scannerNF: 30}
	busy.Totals = Totals{Requests: 5000, RPSAvg: 1, RPMPeak: 700, ConcurrencyPeak: 60, ReadShare: 0.3, P95: 900, ErrRate: 0.01, SpanMinutes: 60}
	busy.ipPeaks = []int64{5, 8, 10, 12, 15, 20, 700, 650}
	busy.TopIPs = []IPRow{
		{IP: "1.1.1.1", Count: 900, Share: 0.18, PeakPerMin: 700, AuthPeak: 12},
		{IP: "2.2.2.2", Count: 300, Share: 0.06, PeakPerMin: 300, AuthPeak: 3},
		{IP: "3.3.3.3", Count: 10, Share: 0.002, PeakPerMin: 4, AuthPeak: 11},
	}
	busy.UAClasses = []KV{{Value: "browser", Count: 3000, Share: 0.6}, {Value: "bot", Count: 1500, Share: 0.3}}
	busy.NotFound = []KV{{Value: "/.env", Count: 10}, {Value: "/foo", Count: 9}, {Value: "/wp-admin", Count: 8}}
	busy.Routes = []RouteRow{
		{Method: "POST", Route: "/api/auth/login", Count: 100, Err4xx: 60},
		{Method: "GET", Route: "/api/products", Count: 1500, Share: 0.3, TimeShare: 0.2, AvgMS: 80, ErrRate: 0.001, HotShare: 0.6, RPM: 25, P95: 400, sumMS: 120000},
		{Method: "GET", Route: "/api/cats", Count: 200, Share: 0.04, TimeShare: 0.01, AvgMS: 40, ErrRate: 0.0, sumMS: 8000},
		{Method: "GET", Route: "static:js", Count: 700},
		{Method: "GET", Route: RouteNotFound, Count: 50},
		{Method: "POST", Route: "/api/export", Count: 50, P50: 3000, P95: 6000, P99: 9000, MaxMS: 30000, TimeShare: 0.2},
		{Method: "GET", Route: "/api/slow", Count: 100, P95: 3500, AvgMS: 2000, P50: 1500, P99: 4000, TimeShare: 0.1},
		{Method: "GET", Route: "/api/meh", Count: 100, P95: 1200, AvgMS: 900},
		{Method: "POST", Route: "/api/orders", Count: 200, Err5xx: 30, ErrRate: 0.15, P95: 300},
		{Method: "POST", Route: "/api/items", Count: 200, Err5xx: 6, ErrRate: 0.03, P95: 300},
		{Method: "GET", Route: "/api/big", Count: 30, AvgOut: 600 << 10},
		{Method: "POST", Route: "/api/upload", Count: 20, AvgIn: 2 << 20},
	}
	busy.Groups = []GroupRow{
		{Group: "/api/a", Count: 100, Share: 0.1, TimeShare: 0.5, P95: 800, AvgMS: 300, Routes: 2},
		{Group: "/api/b", Count: 100, Share: 0.5, TimeShare: 0.2},
		{Group: "/api/c", Count: 100, Share: 0.1, TimeShare: 0.1},
		{Group: "static:js", Count: 100, Share: 0.1, TimeShare: 0.9},
	}
	for i := 0; i < 40; i++ {
		busy.minutes = append(busy.minutes, Point{T: int64(i), Requests: int64(10 + i*10), AvgMS: float64(50 + i*20)})
	}
	busy.minutes = append(busy.minutes, Point{T: 99})
	busy.Instances = []KV{{Value: "a", Count: 3000, Share: 0.75}, {Value: "b", Count: 1000, Share: 0.25}}

	single := &Report{Gran: "m"}
	single.Totals = Totals{Requests: 3000, RPSAvg: 10, RPMPeak: 2000, ConcurrencyPeak: 70, ReadShare: 0.9}
	single.Routes = []RouteRow{{Method: "GET", Route: "static:css", Count: 400}}

	healthy := &Report{Gran: "h"}
	healthy.Totals = Totals{Requests: 500, RPSAvg: 0.1, RPMPeak: 3, ReadShare: 0.9, P95: 80, SpanMinutes: 30}
	healthy.Routes = []RouteRow{{Method: "GET", Route: "/x", Count: 500}}
	healthy.Instances = []KV{{Value: "a", Count: 1}, {Value: "b", Count: 1}}

	sample := &Report{}
	sample.Totals = Totals{Requests: 5}
	return map[string]*Report{"busy": busy, "single": single, "healthy": healthy, "sample": sample}
}

func TestAdviseGolden(t *testing.T) {
	for name, rep := range goldenReports() {
		got, _ := json.MarshalIndent(Advise(rep), "", " ")
		path := filepath.Join("testdata", fmt.Sprintf("advise_%s.golden.json", name))
		if os.Getenv("UPDATE_GOLDEN") != "" {
			_ = os.MkdirAll("testdata", 0o755)
			if err := os.WriteFile(path, got, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(want) != string(got) {
			t.Errorf("%s: output Advise berubah", name)
		}
	}
}
