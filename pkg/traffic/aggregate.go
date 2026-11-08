package traffic

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// LatencyBounds = batas atas (ms) tiap ember histogram latensi; ember terakhir = di atas 30 detik.
var LatencyBounds = []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000}

const (
	RecentKeep         = 1000                // ekor event live per sumber
	MinuteRetention    = 48 * time.Hour      // rangkuman per menit
	HourRetention      = 90 * 24 * time.Hour // rangkuman per jam
	MaxEventsPerIngest = 5000
)

var (
	locksMu sync.Mutex
	locks   = map[uint]*sync.Mutex{}
)

func sourceLock(id uint) *sync.Mutex {
	locksMu.Lock()
	defer locksMu.Unlock()
	if locks[id] == nil {
		locks[id] = &sync.Mutex{}
	}
	return locks[id]
}

func histIndex(ms float64) int {
	for i, b := range LatencyBounds {
		if ms <= b {
			return i
		}
	}
	return len(LatencyBounds)
}

func parseHist(s string) []int64 {
	out := make([]int64, len(LatencyBounds)+1)
	for i, p := range strings.Split(s, ",") {
		if i >= len(out) {
			break
		}
		out[i], _ = strconv.ParseInt(p, 10, 64)
	}
	return out
}

func formatHist(h []int64) string {
	parts := make([]string, len(h))
	for i, v := range h {
		parts[i] = strconv.FormatInt(v, 10)
	}
	return strings.Join(parts, ",")
}

type routeKey struct {
	gran   string
	bucket int64
	method string
	route  string
}

type dimKey struct {
	gran   string
	bucket int64
	dim    string
	value  string
}

type routeAcc struct {
	count, e4, e5, in, out int64
	sum, max               float64
	hist                   []int64
}

// ingestAcc = akumulator batch: statistik rute dan hitungan dimensi per ember waktu.
type ingestAcc struct {
	routes map[routeKey]*routeAcc
	dims   map[dimKey][2]int64
}

func (a *ingestAcc) addDim(gran string, bucket int64, dim, value string, isErr bool) {
	if value == "" {
		return
	}
	k := dimKey{gran, bucket, dim, clip(value, 255)}
	v := a.dims[k]
	v[0]++
	if isErr {
		v[1]++
	}
	a.dims[k] = v
}

// normEvent = event yang sudah dibersihkan dan diberi waktu/ember.
type normEvent struct {
	ev                        Event
	at                        time.Time
	method, path, route       string
	instance, uaClass, client string
	ref, country, ip          string
	dur                       float64
	isErr                     bool
	minute, hour              int64
}

func normalizeEvent(ev Event, now time.Time, defaultInstance string) normEvent {
	at := time.UnixMilli(ev.TS)
	if ev.TS <= 0 || at.After(now.Add(5*time.Minute)) || at.Before(now.Add(-HourRetention)) {
		at = now
	}
	method := strings.ToUpper(clip(strings.TrimSpace(ev.Method), 10))
	if method == "" {
		method = "GET"
	}
	p := CleanPath(ev.Path)
	instance := clip(strings.TrimSpace(ev.Instance), 128)
	if instance == "" {
		instance = defaultInstance
	}
	uaClass, client := ClassifyUA(ev.UA)
	dur := ev.DurMS
	if dur < 0 {
		dur = 0
	}
	return normEvent{
		ev: ev, at: at, method: method, path: p, route: NormalizeRoute(ev.Route, p, ev.Status),
		instance: instance, uaClass: uaClass, client: client, ref: RefererHost(ev.Referer),
		country: strings.ToUpper(clip(strings.TrimSpace(ev.Country), 8)),
		ip:      clip(strings.TrimSpace(ev.IP), 64), dur: dur, isErr: ev.Status >= 400,
		minute: at.Unix() / 60 * 60, hour: at.Unix() / 3600 * 3600,
	}
}

func (a *ingestAcc) addRoute(n normEvent) {
	for _, g := range []struct {
		gran   string
		bucket int64
	}{{models.TrafficGranMinute, n.minute}, {models.TrafficGranHour, n.hour}} {
		k := routeKey{g.gran, g.bucket, n.method, n.route}
		r := a.routes[k]
		if r == nil {
			r = &routeAcc{hist: make([]int64, len(LatencyBounds)+1)}
			a.routes[k] = r
		}
		r.count++
		if n.ev.Status >= 500 {
			r.e5++
		} else if n.ev.Status >= 400 {
			r.e4++
		}
		r.sum += n.dur
		if n.dur > r.max {
			r.max = n.dur
		}
		r.in += max(n.ev.BytesIn, 0)
		r.out += max(n.ev.BytesOut, 0)
		r.hist[histIndex(n.dur)]++
	}
}

func (a *ingestAcc) addDims(n normEvent) {
	// ip, auth_ip & instance juga per menit: puncak per menit dipakai saran rate limit & load balancer.
	a.addDim(models.TrafficGranMinute, n.minute, "ip", n.ip, n.isErr)
	a.addDim(models.TrafficGranHour, n.hour, "ip", n.ip, n.isErr)
	a.addDim(models.TrafficGranMinute, n.minute, "instance", n.instance, n.isErr)
	a.addDim(models.TrafficGranHour, n.hour, "instance", n.instance, n.isErr)
	if IsAuthRoute(n.route) {
		a.addDim(models.TrafficGranMinute, n.minute, "auth_ip", n.ip, n.isErr)
		a.addDim(models.TrafficGranHour, n.hour, "auth_ip", n.ip, n.isErr)
	}
	a.addDim(models.TrafficGranHour, n.hour, "country", n.country, n.isErr)
	a.addDim(models.TrafficGranHour, n.hour, "ua", n.uaClass, n.isErr)
	a.addDim(models.TrafficGranHour, n.hour, "client", n.client, n.isErr)
	a.addDim(models.TrafficGranHour, n.hour, "referer", n.ref, n.isErr)
	a.addDim(models.TrafficGranHour, n.hour, "status", strconv.Itoa(n.ev.Status), n.isErr)
	if n.route == RouteNotFound {
		a.addDim(models.TrafficGranHour, n.hour, "notfound", n.path, true)
	} else if n.method == "GET" && !n.isErr {
		// Path GET persis per rute → seberapa terkonsentrasi permintaan (petunjuk rasio hit cache).
		a.addDim(models.TrafficGranHour, n.hour, "gpath", n.route+"|"+n.path, false)
	}
}

func recentEventOf(srcID uint, n normEvent) models.TrafficRecentEvent {
	return models.TrafficRecentEvent{
		SourceID: srcID, At: n.at, Method: n.method, Route: n.route, Path: n.path, Status: n.ev.Status, DurMS: n.dur,
		BytesOut: n.ev.BytesOut, IP: n.ip, Country: n.country, UAClass: n.uaClass, Client: clip(n.client, 32), Referer: n.ref, Instance: n.instance,
	}
}

// persistIngest menyimpan hasil agregasi batch dalam satu transaksi.
func persistIngest(db *gorm.DB, src *models.TrafficSource, acc *ingestAcc, recent []models.TrafficRecentEvent, total int, lastAt time.Time, lastInstance string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := mergeRoutes(tx, src.ID, acc.routes); err != nil {
			return err
		}
		if err := mergeDims(tx, src.ID, acc.dims); err != nil {
			return err
		}
		if err := tx.CreateInBatches(recent, 200).Error; err != nil {
			return err
		}
		updates := map[string]any{"total_events": gorm.Expr("total_events + ?", total), "last_instance": lastInstance}
		if src.LastEventAt == nil || lastAt.After(*src.LastEventAt) {
			updates["last_event_at"] = lastAt
		}
		return tx.Model(&models.TrafficSource{}).Where("id = ?", src.ID).Updates(updates).Error
	})
}

// Ingest merangkum batch event ke tabel per menit/jam & ekor live. Aman dipanggil paralel (kunci per sumber).
func Ingest(db *gorm.DB, src *models.TrafficSource, events []Event, defaultInstance string) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	if len(events) > MaxEventsPerIngest {
		events = events[:MaxEventsPerIngest]
	}
	lk := sourceLock(src.ID)
	lk.Lock()
	defer lk.Unlock()

	acc := &ingestAcc{routes: map[routeKey]*routeAcc{}, dims: map[dimKey][2]int64{}}
	now := timeNow()
	recent := make([]models.TrafficRecentEvent, 0, 200)
	var lastAt time.Time
	lastInstance := ""
	for i, ev := range events {
		n := normalizeEvent(ev, now, defaultInstance)
		acc.addRoute(n)
		acc.addDims(n)
		if i >= len(events)-200 {
			recent = append(recent, recentEventOf(src.ID, n))
		}
		if n.at.After(lastAt) {
			lastAt = n.at
			lastInstance = n.instance
		}
	}
	if err := persistIngest(db, src, acc, recent, len(events), lastAt, lastInstance); err != nil {
		return 0, err
	}
	trimRecent(db, src.ID)
	return len(events), nil
}

func bucketsOf(set map[int64]bool) []int64 {
	out := make([]int64, 0, len(set))
	for b := range set {
		out = append(out, b)
	}
	return out
}

func mergeRoutes(tx *gorm.DB, sourceID uint, acc map[routeKey]*routeAcc) error {
	byGran := map[string]map[int64]bool{}
	for k := range acc {
		if byGran[k.gran] == nil {
			byGran[k.gran] = map[int64]bool{}
		}
		byGran[k.gran][k.bucket] = true
	}
	existing := map[routeKey]*models.TrafficRouteStat{}
	for gran, set := range byGran {
		var rows []models.TrafficRouteStat
		if err := tx.Where("source_id = ? AND gran = ? AND bucket IN ?", sourceID, gran, bucketsOf(set)).Find(&rows).Error; err != nil {
			return err
		}
		for i := range rows {
			r := &rows[i]
			existing[routeKey{r.Gran, r.Bucket, r.Method, r.Route}] = r
		}
	}
	var fresh []models.TrafficRouteStat
	for k, a := range acc {
		if r := existing[k]; r != nil {
			h := parseHist(r.Hist)
			for i := range h {
				h[i] += a.hist[i]
			}
			r.Count += a.count
			r.Err4xx += a.e4
			r.Err5xx += a.e5
			r.SumMS += a.sum
			r.MaxMS = max(r.MaxMS, a.max)
			r.SumIn += a.in
			r.SumOut += a.out
			r.Hist = formatHist(h)
			if err := tx.Save(r).Error; err != nil {
				return err
			}
			continue
		}
		fresh = append(fresh, models.TrafficRouteStat{
			SourceID: sourceID, Gran: k.gran, Bucket: k.bucket, Method: k.method, Route: k.route,
			Count: a.count, Err4xx: a.e4, Err5xx: a.e5, SumMS: a.sum, MaxMS: a.max, SumIn: a.in, SumOut: a.out, Hist: formatHist(a.hist),
		})
	}
	if len(fresh) == 0 {
		return nil
	}
	return tx.CreateInBatches(fresh, 200).Error
}

func mergeDims(tx *gorm.DB, sourceID uint, acc map[dimKey][2]int64) error {
	byGran := map[string]map[int64]bool{}
	for k := range acc {
		if byGran[k.gran] == nil {
			byGran[k.gran] = map[int64]bool{}
		}
		byGran[k.gran][k.bucket] = true
	}
	existing := map[dimKey]*models.TrafficDimStat{}
	for gran, set := range byGran {
		var rows []models.TrafficDimStat
		if err := tx.Where("source_id = ? AND gran = ? AND bucket IN ?", sourceID, gran, bucketsOf(set)).Find(&rows).Error; err != nil {
			return err
		}
		for i := range rows {
			r := &rows[i]
			existing[dimKey{r.Gran, r.Bucket, r.Dim, r.Value}] = r
		}
	}
	var fresh []models.TrafficDimStat
	for k, v := range acc {
		if r := existing[k]; r != nil {
			if err := tx.Model(r).Updates(map[string]any{"count": r.Count + v[0], "err": r.Err + v[1]}).Error; err != nil {
				return err
			}
			continue
		}
		fresh = append(fresh, models.TrafficDimStat{SourceID: sourceID, Gran: k.gran, Bucket: k.bucket, Dim: k.dim, Value: k.value, Count: v[0], Err: v[1]})
	}
	if len(fresh) == 0 {
		return nil
	}
	return tx.CreateInBatches(fresh, 500).Error
}

func trimRecent(db *gorm.DB, sourceID uint) {
	var cut models.TrafficRecentEvent
	if err := db.Select("id").Where("source_id = ?", sourceID).Order("id DESC").Offset(RecentKeep).Limit(1).Take(&cut).Error; err == nil {
		db.Where("source_id = ? AND id <= ?", sourceID, cut.ID).Delete(&models.TrafficRecentEvent{})
	}
}

// Prune membuang rangkuman di luar masa simpan (dipanggil berkala oleh syncer).
func Prune(db *gorm.DB) {
	now := time.Now()
	minCut := now.Add(-MinuteRetention).Unix()
	hourCut := now.Add(-HourRetention).Unix()
	for _, m := range []any{&models.TrafficRouteStat{}, &models.TrafficDimStat{}} {
		db.Where("gran = ? AND bucket < ?", models.TrafficGranMinute, minCut).Delete(m)
		db.Where("gran = ? AND bucket < ?", models.TrafficGranHour, hourCut).Delete(m)
	}
}
