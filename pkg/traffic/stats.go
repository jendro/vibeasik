package traffic

import (
	"math"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Window = rentang laporan. Jendela ≤24 jam memakai rangkuman per menit, lebih panjang per jam.
type Window struct {
	Key      string
	Duration time.Duration
	Gran     string
	Step     time.Duration // lebar satu titik grafik
}

var Windows = map[string]Window{
	"15m": {"15m", 15 * time.Minute, models.TrafficGranMinute, time.Minute},
	"1h":  {"1h", time.Hour, models.TrafficGranMinute, time.Minute},
	"6h":  {"6h", 6 * time.Hour, models.TrafficGranMinute, 5 * time.Minute},
	"24h": {"24h", 24 * time.Hour, models.TrafficGranMinute, 15 * time.Minute},
	"7d":  {"7d", 7 * 24 * time.Hour, models.TrafficGranHour, time.Hour},
	"30d": {"30d", 30 * 24 * time.Hour, models.TrafficGranHour, 6 * time.Hour},
}

func ParseWindow(key string) Window {
	if w, ok := Windows[key]; ok {
		return w
	}
	return Windows["24h"]
}

type Totals struct {
	Requests        int64   `json:"requests"`
	Err4xx          int64   `json:"err_4xx"`
	Err5xx          int64   `json:"err_5xx"`
	ErrRate         float64 `json:"err_rate"` // 5xx / total
	ClientErrRate   float64 `json:"client_err_rate"`
	RPSAvg          float64 `json:"rps_avg"`
	RPMPeak         float64 `json:"rpm_peak"`         // request di menit tersibuk (jendela per jam: rata-rata menit di jam tersibuk)
	PeakAt          int64   `json:"peak_at"`          // unix detik
	ConcurrencyAvg  float64 `json:"concurrency_avg"`  // Little's law: RPS × latensi rata-rata
	ConcurrencyPeak float64 `json:"concurrency_peak"` // di ember tersibuk
	AvgMS           float64 `json:"avg_ms"`
	P50             float64 `json:"p50"`
	P95             float64 `json:"p95"`
	P99             float64 `json:"p99"`
	MaxMS           float64 `json:"max_ms"`
	AvgOut          float64 `json:"avg_out"`
	AvgIn           float64 `json:"avg_in"`
	UniqueIPs       int     `json:"unique_ips"`
	Instances       int     `json:"instances"`
	ReadShare       float64 `json:"read_share"` // GET/HEAD
	SpanMinutes     float64 `json:"span_minutes"`
}

type Point struct {
	T        int64   `json:"t"`
	Requests int64   `json:"requests"`
	Err4xx   int64   `json:"err_4xx"`
	Err5xx   int64   `json:"err_5xx"`
	AvgMS    float64 `json:"avg_ms"`
	P95      float64 `json:"p95"`
}

type RouteRow struct {
	Method    string  `json:"method"`
	Route     string  `json:"route"`
	Group     string  `json:"group"`
	Count     int64   `json:"count"`
	Share     float64 `json:"share"`
	TimeShare float64 `json:"time_share"` // porsi total waktu server yang dihabiskan rute ini
	RPM       float64 `json:"rpm"`
	Err4xx    int64   `json:"err_4xx"`
	Err5xx    int64   `json:"err_5xx"`
	ErrRate   float64 `json:"err_rate"`
	AvgMS     float64 `json:"avg_ms"`
	P50       float64 `json:"p50"`
	P95       float64 `json:"p95"`
	P99       float64 `json:"p99"`
	MaxMS     float64 `json:"max_ms"`
	AvgOut    float64 `json:"avg_out"`
	AvgIn     float64 `json:"avg_in"`
	HotShare  float64 `json:"hot_share"` // porsi 10 path GET teratas dari rute ini (petunjuk rasio hit cache)
	hist      []int64
	sumMS     float64
	sumIn     int64
	sumOut    int64
}

type GroupRow struct {
	Group     string  `json:"group"`
	Routes    int     `json:"routes"`
	Count     int64   `json:"count"`
	Share     float64 `json:"share"`
	TimeShare float64 `json:"time_share"`
	P95       float64 `json:"p95"`
	ErrRate   float64 `json:"err_rate"`
	AvgMS     float64 `json:"avg_ms"`
}

type KV struct {
	Value string  `json:"value"`
	Count int64   `json:"count"`
	Err   int64   `json:"err"`
	Share float64 `json:"share"`
}

type IPRow struct {
	IP         string  `json:"ip"`
	Count      int64   `json:"count"`
	Share      float64 `json:"share"`
	PeakPerMin int64   `json:"peak_per_min"`
	ErrRate    float64 `json:"err_rate"`
	AuthHits   int64   `json:"auth_hits"`
	AuthPeak   int64   `json:"auth_peak_per_min"`
}

type Report struct {
	Source          models.TrafficSource `json:"source"`
	Window          string               `json:"window"`
	Gran            string               `json:"gran"`
	From            time.Time            `json:"from"`
	To              time.Time            `json:"to"`
	StepSec         int64                `json:"step_sec"`
	Totals          Totals               `json:"totals"`
	Series          []Point              `json:"series"`
	Routes          []RouteRow           `json:"routes"`
	Groups          []GroupRow           `json:"groups"`
	Status          []KV                 `json:"status"`
	Countries       []KV                 `json:"countries"`
	Cities          []KV                 `json:"cities,omitempty"`
	ASNs            []KV                 `json:"asns,omitempty"`
	Orgs            []KV                 `json:"orgs,omitempty"`
	NetTypes        []KV                 `json:"net_types,omitempty"`
	Referers        []KV                 `json:"referers"`
	UAClasses       []KV                 `json:"ua_classes"`
	Clients         []KV                 `json:"clients"`
	Instances       []KV                 `json:"instances"`
	NotFound        []KV                 `json:"not_found"`
	TopIPs          []IPRow              `json:"top_ips"`
	Recommendations []Recommendation     `json:"recommendations"`

	minutes   []Point // per menit (hanya jendela gran menit) untuk deteksi kejenuhan
	ipPeaks   []int64
	hotByKey  map[string]float64
	scannerNF int64
}

// Percentile memperkirakan kuantil dari histogram ember (interpolasi linear di dalam ember).
func Percentile(h []int64, q float64, maxMS float64) float64 {
	var total int64
	for _, v := range h {
		total += v
	}
	if total == 0 {
		return 0
	}
	target := q * float64(total)
	var cum int64
	for i, v := range h {
		if v == 0 {
			continue
		}
		if float64(cum+v) >= target {
			lo := 0.0
			if i > 0 {
				lo = LatencyBounds[i-1]
			}
			// maxMS (bila diketahui) mempersempit ember teratas yang terisi; ember terakhir tak berbatas.
			hi := math.Max(maxMS, lo)
			if i < len(LatencyBounds) {
				hi = LatencyBounds[i]
				if maxMS > 0 {
					hi = math.Min(hi, math.Max(maxMS, lo))
				}
			}
			frac := (target - float64(cum)) / float64(v)
			return round1(lo + (hi-lo)*frac)
		}
		cum += v
	}
	return round1(maxMS)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func ratio(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// timeNow dipisah agar uji karakterisasi bisa membekukan waktu.
var timeNow = time.Now

type bucketAcc struct {
	count, e4, e5 int64
	sum           float64
	hist          []int64
}

func newHist() []int64 { return make([]int64, len(LatencyBounds)+1) }

func addHist(dst, src []int64) {
	for i := range src {
		dst[i] += src[i]
	}
}

// rowAgg = hasil agregasi baris statistik mentah menjadi total, per rute, dan per ember waktu.
type rowAgg struct {
	totalHist   []int64
	byRoute     map[string]*RouteRow
	byBucket    map[int64]*bucketAcc
	sumMS       float64
	sumIn       int64
	sumOut      int64
	reads       int64
	firstBucket int64
}

func (a *rowAgg) addTotals(t *Totals, r models.TrafficRouteStat, h []int64) {
	t.Requests += r.Count
	t.Err4xx += r.Err4xx
	t.Err5xx += r.Err5xx
	t.MaxMS = math.Max(t.MaxMS, r.MaxMS)
	a.sumMS += r.SumMS
	a.sumIn += r.SumIn
	a.sumOut += r.SumOut
	if r.Method == "GET" || r.Method == "HEAD" {
		a.reads += r.Count
	}
	if r.Bucket < a.firstBucket {
		a.firstBucket = r.Bucket
	}
	addHist(a.totalHist, h)
}

func (a *rowAgg) addRoute(r models.TrafficRouteStat, h []int64) {
	key := r.Method + " " + r.Route
	rr := a.byRoute[key]
	if rr == nil {
		rr = &RouteRow{Method: r.Method, Route: r.Route, Group: RouteGroup(r.Route), hist: newHist()}
		a.byRoute[key] = rr
	}
	rr.Count += r.Count
	rr.Err4xx += r.Err4xx
	rr.Err5xx += r.Err5xx
	rr.sumMS += r.SumMS
	rr.sumIn += r.SumIn
	rr.sumOut += r.SumOut
	rr.MaxMS = math.Max(rr.MaxMS, r.MaxMS)
	addHist(rr.hist, h)
}

func (a *rowAgg) addBucket(r models.TrafficRouteStat, h []int64) {
	b := a.byBucket[r.Bucket]
	if b == nil {
		b = &bucketAcc{hist: newHist()}
		a.byBucket[r.Bucket] = b
	}
	b.count += r.Count
	b.e4 += r.Err4xx
	b.e5 += r.Err5xx
	b.sum += r.SumMS
	addHist(b.hist, h)
}

func aggregateRows(t *Totals, rows []models.TrafficRouteStat) *rowAgg {
	a := &rowAgg{totalHist: newHist(), byRoute: map[string]*RouteRow{}, byBucket: map[int64]*bucketAcc{}, firstBucket: math.MaxInt64}
	for _, r := range rows {
		h := parseHist(r.Hist)
		a.addTotals(t, r, h)
		a.addRoute(r, h)
		a.addBucket(r, h)
	}
	return a
}

// fillTotals mengisi statistik turunan total; mengembalikan rentang data (menit) yang dipakai.
func fillTotals(rep *Report, a *rowAgg, now time.Time) float64 {
	t := &rep.Totals
	// Rentang data sebenarnya: sumber yang baru tersambung tidak dihitung seolah sepi sepanjang jendela.
	span := now.Sub(rep.From).Minutes()
	if a.firstBucket != math.MaxInt64 {
		span = math.Min(span, now.Sub(time.Unix(a.firstBucket, 0)).Minutes())
	}
	span = math.Max(span, 1)
	t.SpanMinutes = round1(span)

	if t.Requests > 0 {
		n := float64(t.Requests)
		t.ErrRate = ratio(float64(t.Err5xx), n)
		t.ClientErrRate = ratio(float64(t.Err4xx), n)
		t.AvgMS = round1(a.sumMS / n)
		t.AvgIn = math.Round(float64(a.sumIn) / n)
		t.AvgOut = math.Round(float64(a.sumOut) / n)
		t.ReadShare = ratio(float64(a.reads), n)
		t.P50 = Percentile(a.totalHist, 0.50, t.MaxMS)
		t.P95 = Percentile(a.totalHist, 0.95, t.MaxMS)
		t.P99 = Percentile(a.totalHist, 0.99, t.MaxMS)
		t.RPSAvg = n / (span * 60)
		t.ConcurrencyAvg = a.sumMS / (span * 60000)
	}
	return span
}

// fillPeaks menghitung puncak beban dan deret per menit (jendela gran menit).
func fillPeaks(rep *Report, w Window, byBucket map[int64]*bucketAcc) {
	t := &rep.Totals
	bucketSec := int64(60)
	if w.Gran == models.TrafficGranHour {
		bucketSec = 3600
	}
	for bk, b := range byBucket {
		perMin := float64(b.count) / (float64(bucketSec) / 60)
		if perMin > t.RPMPeak {
			t.RPMPeak = perMin
			t.PeakAt = bk
			t.ConcurrencyPeak = b.sum / (float64(bucketSec) * 1000)
		}
		if w.Gran == models.TrafficGranMinute {
			rep.minutes = append(rep.minutes, Point{T: bk, Requests: b.count, Err4xx: b.e4, Err5xx: b.e5, AvgMS: ratio(b.sum, float64(b.count)), P95: Percentile(b.hist, 0.95, 0)})
		}
	}
	sort.Slice(rep.minutes, func(i, j int) bool { return rep.minutes[i].T < rep.minutes[j].T })
	t.RPSAvg = math.Round(t.RPSAvg*100) / 100
	t.RPMPeak = round1(t.RPMPeak)
	t.ConcurrencyAvg = math.Round(t.ConcurrencyAvg*100) / 100
	t.ConcurrencyPeak = math.Round(t.ConcurrencyPeak*100) / 100
}

// fillSeries membuat seri grafik: ember digabung per Step, termasuk titik kosong agar grafik tidak menipu.
func fillSeries(rep *Report, w Window, byBucket map[int64]*bucketAcc, now time.Time) {
	step := int64(w.Step.Seconds())
	series := map[int64]*bucketAcc{}
	for bk, b := range byBucket {
		k := bk / step * step
		s := series[k]
		if s == nil {
			s = &bucketAcc{hist: newHist()}
			series[k] = s
		}
		s.count += b.count
		s.e4 += b.e4
		s.e5 += b.e5
		s.sum += b.sum
		addHist(s.hist, b.hist)
	}
	for k := rep.From.Unix() / step * step; k <= now.Unix(); k += step {
		p := Point{T: k}
		if s := series[k]; s != nil {
			p.Requests, p.Err4xx, p.Err5xx = s.count, s.e4, s.e5
			p.AvgMS = round1(ratio(s.sum, float64(s.count)))
			p.P95 = Percentile(s.hist, 0.95, 0)
		}
		rep.Series = append(rep.Series, p)
	}
}

// dimData = hasil pembacaan dimensi (ip, status, negara, dst.) untuk satu jendela.
type dimData struct {
	dims         map[string][]KV
	ipAuth       map[string]int64
	gpathByRoute map[string][]int64
}

func loadDims(db *gorm.DB, rep *Report, srcID uint, fromUnix int64) (*dimData, error) {
	var dimRows []struct {
		Dim   string
		Value string
		Count int64
		Err   int64
	}
	if err := db.Model(&models.TrafficDimStat{}).Select("dim, value, SUM(count) AS count, SUM(err) AS err").
		Where("source_id = ? AND gran = ? AND bucket >= ?", srcID, models.TrafficGranHour, fromUnix/3600*3600).
		Group("dim, value").Scan(&dimRows).Error; err != nil {
		return nil, err
	}
	d := &dimData{dims: map[string][]KV{}, ipAuth: map[string]int64{}, gpathByRoute: map[string][]int64{}}
	for _, row := range dimRows {
		switch row.Dim {
		case "auth_ip":
			d.ipAuth[row.Value] = row.Count
			continue
		case "gpath":
			if i := strings.LastIndexByte(row.Value, '|'); i > 0 {
				d.gpathByRoute[row.Value[:i]] = append(d.gpathByRoute[row.Value[:i]], row.Count)
			}
			continue
		case "notfound":
			if IsScannerPath(row.Value) {
				rep.scannerNF += row.Count
			}
		}
		d.dims[row.Dim] = append(d.dims[row.Dim], KV{Value: row.Value, Count: row.Count, Err: row.Err, Share: ratio(float64(row.Count), float64(rep.Totals.Requests))})
	}
	return d, nil
}

func fillHotShare(rep *Report, gpathByRoute map[string][]int64) {
	for route, counts := range gpathByRoute {
		sort.Slice(counts, func(i, j int) bool { return counts[i] > counts[j] })
		var top, all int64
		for i, c := range counts {
			all += c
			if i < 10 {
				top += c
			}
		}
		rep.hotByKey[route] = ratio(float64(top), float64(all))
	}
}

func topKV(dims map[string][]KV, name string, n int) []KV {
	list := dims[name]
	sort.Slice(list, func(i, j int) bool { return list[i].Count > list[j].Count })
	if len(list) > n {
		list = list[:n]
	}
	if list == nil {
		list = []KV{}
	}
	return list
}

func fillDimLists(rep *Report, dims map[string][]KV) {
	rep.Status = topKV(dims, "status", 20)
	rep.Countries = topKV(dims, "country", 15)
	fillGeoLists(rep, dims)
	rep.Referers = topKV(dims, "referer", 15)
	rep.UAClasses = topKV(dims, "ua", 10)
	rep.Clients = topKV(dims, "client", 15)
	rep.Instances = topKV(dims, "instance", 20)
	rep.NotFound = topKV(dims, "notfound", 15)
	rep.Totals.Instances = len(dims["instance"])
	rep.Totals.UniqueIPs = len(dims["ip"])
}

// loadIPPeaks membaca puncak per menit per IP (hanya ada di jendela ≤48 jam: dimensi per menit).
func loadIPPeaks(db *gorm.DB, rep *Report, srcID uint, gran string, fromUnix int64) (peaks, authPeaks map[string]int64) {
	peaks = map[string]int64{}
	authPeaks = map[string]int64{}
	if gran != models.TrafficGranMinute {
		return
	}
	var pk []struct {
		Dim   string
		Value string
		Peak  int64
	}
	db.Model(&models.TrafficDimStat{}).Select("dim, value, MAX(count) AS peak").
		Where("source_id = ? AND gran = ? AND bucket >= ? AND dim IN ?", srcID, models.TrafficGranMinute, fromUnix, []string{"ip", "auth_ip"}).
		Group("dim, value").Scan(&pk)
	for _, p := range pk {
		if p.Dim == "ip" {
			peaks[p.Value] = p.Peak
			rep.ipPeaks = append(rep.ipPeaks, p.Peak)
		} else {
			authPeaks[p.Value] = p.Peak
		}
	}
	return
}

func fillTopIPs(rep *Report, ips []KV, peaks, ipAuth, authPeaks map[string]int64) {
	sort.Slice(ips, func(i, j int) bool { return ips[i].Count > ips[j].Count })
	for i, ip := range ips {
		if i >= 25 {
			break
		}
		rep.TopIPs = append(rep.TopIPs, IPRow{IP: ip.Value, Count: ip.Count, Share: ip.Share, PeakPerMin: peaks[ip.Value],
			ErrRate: ratio(float64(ip.Err), float64(ip.Count)), AuthHits: ipAuth[ip.Value], AuthPeak: authPeaks[ip.Value]})
	}
	if rep.TopIPs == nil {
		rep.TopIPs = []IPRow{}
	}
}

// fillRoutesAndGroups menurunkan baris rute dan kelompok rute dari agregat.
func fillRoutesAndGroups(rep *Report, a *rowAgg, span float64) {
	t := rep.Totals
	groups := map[string]*GroupRow{}
	groupHist := map[string][]int64{}
	groupSum := map[string]float64{}
	for _, rr := range a.byRoute {
		finishRoute(rep, rr, a.sumMS, span)
		rep.Routes = append(rep.Routes, *rr)

		g := groups[rr.Group]
		if g == nil {
			g = &GroupRow{Group: rr.Group}
			groups[rr.Group] = g
			groupHist[rr.Group] = newHist()
		}
		g.Routes++
		g.Count += rr.Count
		g.ErrRate += float64(rr.Err5xx)
		groupSum[rr.Group] += rr.sumMS
		addHist(groupHist[rr.Group], rr.hist)
		g.P95 = math.Max(g.P95, rr.MaxMS) // sementara: dipakai sebagai max untuk Percentile di bawah
	}
	sort.Slice(rep.Routes, func(i, j int) bool { return rep.Routes[i].Count > rep.Routes[j].Count })
	if rep.Routes == nil {
		rep.Routes = []RouteRow{}
	}
	for name, g := range groups {
		n := float64(g.Count)
		g.Share = ratio(n, float64(t.Requests))
		g.TimeShare = ratio(groupSum[name], a.sumMS)
		g.ErrRate = ratio(g.ErrRate, n)
		g.AvgMS = round1(ratio(groupSum[name], n))
		g.P95 = Percentile(groupHist[name], 0.95, g.P95)
		rep.Groups = append(rep.Groups, *g)
	}
	sort.Slice(rep.Groups, func(i, j int) bool { return rep.Groups[i].TimeShare > rep.Groups[j].TimeShare })
	if rep.Groups == nil {
		rep.Groups = []GroupRow{}
	}
}

func finishRoute(rep *Report, rr *RouteRow, sumMS, span float64) {
	n := float64(rr.Count)
	rr.Share = ratio(n, float64(rep.Totals.Requests))
	rr.TimeShare = ratio(rr.sumMS, sumMS)
	rr.RPM = round1(n / span)
	rr.ErrRate = ratio(float64(rr.Err5xx), n)
	rr.AvgMS = round1(ratio(rr.sumMS, n))
	rr.P50 = Percentile(rr.hist, 0.50, rr.MaxMS)
	rr.P95 = Percentile(rr.hist, 0.95, rr.MaxMS)
	rr.P99 = Percentile(rr.hist, 0.99, rr.MaxMS)
	rr.AvgOut = math.Round(ratio(float64(rr.sumOut), n))
	rr.AvgIn = math.Round(ratio(float64(rr.sumIn), n))
	if rr.Method == "GET" {
		rr.HotShare = rep.hotByKey[rr.Route]
	}
}

// BuildReport menghitung statistik + rekomendasi untuk satu sumber pada jendela tertentu.
func BuildReport(db *gorm.DB, src models.TrafficSource, w Window) (*Report, error) {
	now := timeNow()
	from := now.Add(-w.Duration)
	fromUnix := from.Unix()
	if w.Gran == models.TrafficGranHour {
		fromUnix = fromUnix / 3600 * 3600
	} else {
		fromUnix = fromUnix / 60 * 60
	}
	rep := &Report{Source: src, Window: w.Key, Gran: w.Gran, From: time.Unix(fromUnix, 0), To: now, StepSec: int64(w.Step.Seconds()), hotByKey: map[string]float64{}}

	var rows []models.TrafficRouteStat
	if err := db.Where("source_id = ? AND gran = ? AND bucket >= ?", src.ID, w.Gran, fromUnix).Find(&rows).Error; err != nil {
		return nil, err
	}
	agg := aggregateRows(&rep.Totals, rows)
	span := fillTotals(rep, agg, now)
	fillPeaks(rep, w, agg.byBucket)
	fillSeries(rep, w, agg.byBucket, now)

	d, err := loadDims(db, rep, src.ID, fromUnix)
	if err != nil {
		return nil, err
	}
	fillHotShare(rep, d.gpathByRoute)
	fillDimLists(rep, d.dims)
	peaks, authPeaks := loadIPPeaks(db, rep, src.ID, w.Gran, fromUnix)
	fillTopIPs(rep, d.dims["ip"], peaks, d.ipAuth, authPeaks)
	fillRoutesAndGroups(rep, agg, span)

	rep.Recommendations = Advise(rep)
	return rep, nil
}
