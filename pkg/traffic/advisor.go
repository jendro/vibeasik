package traffic

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Recommendation = saran optimasi berbasis data traffic nyata (aturan deterministik, tanpa LLM).
// ID stabil per sumber (rule + rute) agar "Jadikan Todo" tidak menggandakan todo yang sama.
type Recommendation struct {
	ID       string   `json:"id"`
	Rule     string   `json:"rule"`
	Category string   `json:"category"` // rate_limit, cache, load_balancer, split_service, performance, errors, security, payload, capacity, info
	Severity string   `json:"severity"` // critical, warning, info
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
	Action   string   `json:"action"`
	Evidence []string `json:"evidence"`
	Routes   []string `json:"routes,omitempty"`
}

const (
	SevCritical = "critical"
	SevWarning  = "warning"
	SevInfo     = "info"

	minSample = 100 // di bawah ini laporan hanya memberi saran "data belum cukup"
)

func pct(v float64) string { return fmt.Sprintf("%.1f%%", v*100) }

func ms(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.2f dtk", v/1000)
	}
	return fmt.Sprintf("%.0f ms", v)
}

func bytesH(v float64) string {
	switch {
	case v >= 1<<20:
		return fmt.Sprintf("%.1f MB", v/(1<<20))
	case v >= 1<<10:
		return fmt.Sprintf("%.0f KB", v/(1<<10))
	}
	return fmt.Sprintf("%.0f B", v)
}

func quantile(vals []int64, q float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := append([]int64(nil), vals...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	idx := int(math.Ceil(q*float64(len(s)))) - 1
	return float64(s[max(0, min(idx, len(s)-1))])
}

func pearson(xs, ys []float64) float64 {
	n := float64(len(xs))
	if n < 3 {
		return 0
	}
	var sx, sy, sxx, syy, sxy float64
	for i := range xs {
		sx += xs[i]
		sy += ys[i]
		sxx += xs[i] * xs[i]
		syy += ys[i] * ys[i]
		sxy += xs[i] * ys[i]
	}
	den := math.Sqrt((n*sxx - sx*sx) * (n*syy - sy*sy))
	if den == 0 {
		return 0
	}
	return (n*sxy - sx*sy) / den
}

func routeLabel(r RouteRow) string { return r.Method + " " + r.Route }

// Advise menerjemahkan laporan menjadi saran: rate limiting, cache, load balancer, pecah service,
// performa, error, keamanan, ukuran payload & pola beban. Diurutkan dari yang paling mendesak.
func Advise(rep *Report) []Recommendation {
	t := rep.Totals
	if t.Requests < minSample {
		return []Recommendation{adviceSample(t)}
	}
	steps := []func(*Report) []Recommendation{
		adviceRateLimitIP, adviceRateLimitAuth, adviceBurst, adviceBots, adviceScanner,
		adviceCacheRoutes, adviceCacheStatic, advicePerformance, adviceErrors5xx,
		advicePayload, adviceSplitGroups, adviceSaturation, adviceInstances, adviceWriteHeavy,
	}
	recs := []Recommendation{}
	for _, step := range steps {
		recs = append(recs, step(rep)...)
	}
	if !hasProblem(recs) {
		recs = append(recs, adviceHealthy(t))
	}
	rank := map[string]int{SevCritical: 0, SevWarning: 1, SevInfo: 2}
	sort.SliceStable(recs, func(i, j int) bool { return rank[recs[i].Severity] < rank[recs[j].Severity] })
	return recs
}

func hasProblem(recs []Recommendation) bool {
	for _, r := range recs {
		if r.Severity != SevInfo {
			return true
		}
	}
	return false
}

func adviceSample(t Totals) Recommendation {
	return Recommendation{
		ID: "info:sample", Rule: "TRAFFIC-SAMPLE", Category: "info", Severity: SevInfo,
		Title:    "Data traffic belum cukup untuk disimpulkan",
		Detail:   fmt.Sprintf("Baru %d request di jendela ini. Saran rate limit, cache, load balancer & pemecahan service butuh minimal %d request agar tidak menebak.", t.Requests, minSample),
		Action:   "Biarkan SDK berjalan beberapa jam di server live, atau pilih jendela waktu yang lebih panjang.",
		Evidence: []string{fmt.Sprintf("%d request", t.Requests)},
	}
}

func adviceHealthy(t Totals) Recommendation {
	return Recommendation{
		ID: "info:healthy", Rule: "TRAFFIC-HEALTHY", Category: "info", Severity: SevInfo,
		Title:    "Belum perlu load balancer, cache besar, atau pecah service",
		Detail:   fmt.Sprintf("Rata-rata %.2f req/detik, puncak %.0f req/menit, p95 %s, error 5xx %s. Satu instance masih cukup; memecah service sekarang hanya menambah kerumitan.", t.RPSAvg, t.RPMPeak, ms(t.P95), pct(t.ErrRate)),
		Action:   "Fokus ke fitur. Tetap pasang rate limit dasar per IP (mis. 120 req/menit) & batas khusus endpoint login sebagai pengaman sebelum traffic tumbuh.",
		Evidence: []string{fmt.Sprintf("%d request dalam %.0f menit", t.Requests, t.SpanMinutes)},
	}
}

// ── Rate limiting ────────────────────────────────────────────────────────────

func abusiveIPs(rep *Report, threshold float64) []IPRow {
	var abusive []IPRow
	for _, ip := range rep.TopIPs {
		if float64(ip.PeakPerMin) >= threshold && ip.Count >= 200 {
			abusive = append(abusive, ip)
		}
	}
	return abusive
}

func adviceRateLimitIP(rep *Report) []Recommendation {
	if rep.Gran != "m" || len(rep.ipPeaks) == 0 {
		return nil
	}
	median := quantile(rep.ipPeaks, 0.5)
	p95 := quantile(rep.ipPeaks, 0.95)
	suggested := math.Max(60, math.Ceil(p95*3/10)*10)
	abusive := abusiveIPs(rep, math.Max(120, median*10))
	if len(abusive) == 0 {
		return nil
	}
	sev := SevWarning
	ev := []string{fmt.Sprintf("Puncak per menit IP biasa (median) %.0f req, p95 %.0f req", median, p95)}
	for i, ip := range abusive {
		if i >= 5 {
			break
		}
		if ip.PeakPerMin >= 600 {
			sev = SevCritical
		}
		ev = append(ev, fmt.Sprintf("%s: puncak %d req/menit, total %d (%s traffic)", ip.IP, ip.PeakPerMin, ip.Count, pct(ip.Share)))
	}
	return []Recommendation{{
		ID: "rate_limit:ip", Rule: "TRAFFIC-RATE-LIMIT-IP", Category: "rate_limit", Severity: sev,
		Title:    fmt.Sprintf("%d IP mengirim request jauh di atas pengguna normal", len(abusive)),
		Detail:   "Satu IP yang membanjiri server bisa memperlambat semua pengguna lain (scraper, bot, skrip salah, atau serangan). Tanpa rate limit per IP, satu klien bisa menghabiskan kapasitas.",
		Action:   fmt.Sprintf("Pasang rate limiter per IP ±%.0f request/menit (3× p95 pengguna normal), balas 429 + Retry-After. Simpan hitungan di Redis bila instance > 1. Pertimbangkan blokir IP teratas di firewall/CDN.", suggested),
		Evidence: ev,
	}}
}

func adviceRateLimitAuth(rep *Report) []Recommendation {
	var authRoutes []RouteRow
	var authCount, auth4xx int64
	for _, r := range rep.Routes {
		if IsAuthRoute(r.Route) && r.Method != "GET" {
			authRoutes = append(authRoutes, r)
			authCount += r.Count
			auth4xx += r.Err4xx
		}
	}
	if len(authRoutes) == 0 {
		return nil
	}
	var hot []IPRow
	for _, ip := range rep.TopIPs {
		if ip.AuthPeak >= 10 {
			hot = append(hot, ip)
		}
	}
	failRate := ratio(float64(auth4xx), float64(authCount))
	if len(hot) == 0 && !(authCount >= 50 && failRate >= 0.3) {
		return nil
	}
	routes := []string{}
	for _, r := range authRoutes {
		routes = append(routes, routeLabel(r))
	}
	ev := []string{fmt.Sprintf("%d request ke endpoint auth, %s gagal (4xx)", authCount, pct(failRate))}
	for i, ip := range hot {
		if i >= 5 {
			break
		}
		ev = append(ev, fmt.Sprintf("%s: %d percobaan auth/menit (puncak)", ip.IP, ip.AuthPeak))
	}
	return []Recommendation{{
		ID: "rate_limit:auth", Rule: "TRAFFIC-RATE-LIMIT-AUTH", Category: "rate_limit", Severity: SevCritical,
		Title:    "Endpoint login/OTP/daftar diserang percobaan berulang",
		Detail:   "Banyak percobaan gagal atau percobaan cepat dari satu IP ke endpoint autentikasi adalah pola brute force / credential stuffing / spam OTP (biaya SMS/email).",
		Action:   "Rate limit ketat khusus endpoint auth: 5–10 percobaan/menit per IP dan per akun/nomor, jeda bertahap setelah gagal, CAPTCHA setelah 3–5 gagal, dan batas harian untuk kirim OTP.",
		Evidence: ev, Routes: routes,
	}}
}

func adviceBurst(rep *Report) []Recommendation {
	t := rep.Totals
	if !(rep.Gran == "m" && t.RPSAvg*60 >= 5 && t.RPMPeak >= 8*t.RPSAvg*60) {
		return nil
	}
	return []Recommendation{{
		ID: "capacity:burst", Rule: "TRAFFIC-BURST", Category: "capacity", Severity: SevWarning,
		Title:    "Lonjakan traffic tajam (bursty)",
		Detail:   fmt.Sprintf("Menit tersibuk %.0f req/menit, %.0f× rata-rata %.1f req/menit. Server yang disiapkan untuk rata-rata akan tumbang saat lonjakan.", t.RPMPeak, t.RPMPeak/(t.RPSAvg*60), t.RPSAvg*60),
		Action:   "Pasang rate limit global sebagai pengaman (load shedding 429/503 sebelum server mati), pindahkan pekerjaan berat saat lonjakan ke antrean (queue + worker), dan siapkan autoscaling / instance cadangan di belakang load balancer.",
		Evidence: []string{fmt.Sprintf("Puncak %.0f req/menit", t.RPMPeak), fmt.Sprintf("Rata-rata %.2f req/detik", t.RPSAvg)},
	}}
}

// ── Bot & pemindai ───────────────────────────────────────────────────────────

func adviceBots(rep *Report) []Recommendation {
	var out []Recommendation
	for _, ua := range rep.UAClasses {
		if ua.Value == "bot" && ua.Share >= 0.25 && ua.Count >= 200 {
			out = append(out, Recommendation{
				ID: "rate_limit:bot", Rule: "TRAFFIC-BOT-SHARE", Category: "rate_limit", Severity: SevWarning,
				Title:    fmt.Sprintf("%s traffic berasal dari bot/crawler", pct(ua.Share)),
				Detail:   "Bot menghabiskan kapasitas yang seharusnya untuk pengguna. Sebagian berguna (Googlebot), sebagian tidak (scraper, AI crawler).",
				Action:   "Atur robots.txt & Crawl-delay, rate limit lebih ketat untuk UA bot, blokir bot yang tidak berguna di CDN/WAF, dan cache halaman/endpoint yang sering dirayapi.",
				Evidence: []string{fmt.Sprintf("%d request bot", ua.Count)},
			})
		}
	}
	return out
}

func adviceScanner(rep *Report) []Recommendation {
	if rep.scannerNF < 20 {
		return nil
	}
	ev := []string{fmt.Sprintf("%d request ke path pemindai kerentanan", rep.scannerNF)}
	for i, nf := range rep.NotFound {
		if i >= 6 {
			break
		}
		if IsScannerPath(nf.Value) {
			ev = append(ev, fmt.Sprintf("%s (%d×)", nf.Value, nf.Count))
		}
	}
	return []Recommendation{{
		ID: "security:scanner", Rule: "TRAFFIC-SCANNER", Category: "security", Severity: SevWarning,
		Title:    "Server dipindai bot pencari celah (.env, wp-admin, .git, …)",
		Detail:   "Pemindai otomatis mencari berkas rahasia & panel admin. Tidak berbahaya selama berkas itu memang tidak tersaji, tetapi tetap memakan kapasitas dan menandakan server sudah jadi target.",
		Action:   "Pastikan .env/.git tidak pernah tersaji, blokir pola path ini di reverse proxy/WAF (Caddy/Nginx/Cloudflare), dan pasang fail2ban/limit untuk IP yang banyak menghasilkan 404.",
		Evidence: ev,
	}}
}

// ── Cache ────────────────────────────────────────────────────────────────────

func cacheableRoutes(rep *Report) []RouteRow {
	cacheMin := int64(math.Max(100, float64(rep.Totals.Requests)*0.02))
	var cacheable []RouteRow
	for _, r := range rep.Routes {
		if r.Method != "GET" || strings.HasPrefix(r.Route, "static:") || r.Route == RouteNotFound || IsAuthRoute(r.Route) {
			continue
		}
		if r.Count >= cacheMin && r.ErrRate < 0.01 && r.AvgMS >= 30 {
			cacheable = append(cacheable, r)
		}
	}
	sort.Slice(cacheable, func(i, j int) bool { return cacheable[i].sumMS > cacheable[j].sumMS })
	return cacheable
}

func adviceCacheRoutes(rep *Report) []Recommendation {
	var out []Recommendation
	for i, r := range cacheableRoutes(rep) {
		if i >= 5 {
			break
		}
		sev := SevInfo
		hit := "belum diketahui"
		if r.HotShare > 0 {
			hit = pct(r.HotShare)
		}
		if r.HotShare >= 0.5 || r.TimeShare >= 0.15 {
			sev = SevWarning
		}
		out = append(out, Recommendation{
			ID: "cache:" + routeLabel(r), Rule: "TRAFFIC-CACHE-ROUTE", Category: "cache", Severity: sev,
			Title:  "Cache respons " + routeLabel(r),
			Detail: fmt.Sprintf("Endpoint GET ini dipanggil %d kali (%s traffic) dengan rata-rata %s dan memakan %s total waktu server. 10 path teratasnya mencakup %s panggilan — makin tinggi, makin besar rasio hit cache.", r.Count, pct(r.Share), ms(r.AvgMS), pct(r.TimeShare), hit),
			Action: "Pasang cache: data publik → header Cache-Control/ETag + CDN; data per pengguna → cache di Redis/in-memory dengan TTL pendek (30–300 dtk) dan invalidasi saat data berubah. Ukur ulang p95 setelahnya.",
			Evidence: []string{
				fmt.Sprintf("%.1f req/menit, p95 %s", r.RPM, ms(r.P95)),
				fmt.Sprintf("Error 5xx %s", pct(r.ErrRate)),
			},
			Routes: []string{routeLabel(r)},
		})
	}
	return out
}

func adviceCacheStatic(rep *Report) []Recommendation {
	var staticCount int64
	var staticRoutes []string
	for _, r := range rep.Routes {
		if strings.HasPrefix(r.Route, "static:") {
			staticCount += r.Count
			staticRoutes = append(staticRoutes, routeLabel(r))
		}
	}
	reqs := rep.Totals.Requests
	if !(staticCount >= 500 || (staticCount >= 100 && ratio(float64(staticCount), float64(reqs)) >= 0.1)) {
		return nil
	}
	return []Recommendation{{
		ID: "cache:static", Rule: "TRAFFIC-CACHE-STATIC", Category: "cache", Severity: SevWarning,
		Title:    "Berkas statis dilayani oleh backend",
		Detail:   fmt.Sprintf("%d request (%s) adalah berkas statis (js/css/gambar). Backend aplikasi tidak perlu ikut melayani ini.", staticCount, pct(ratio(float64(staticCount), float64(reqs)))),
		Action:   "Sajikan berkas statis lewat CDN (Cloudflare/Bunny) atau langsung dari Caddy/Nginx dengan Cache-Control: public, max-age=31536000, immutable untuk aset ber-hash.",
		Evidence: []string{fmt.Sprintf("%d request statis", staticCount)}, Routes: staticRoutes,
	}}
}

// ── Performa & error per rute ────────────────────────────────────────────────

func adviceLongRunning(r RouteRow) Recommendation {
	return Recommendation{
		ID: "split:job:" + routeLabel(r), Rule: "TRAFFIC-LONG-RUNNING", Category: "split_service", Severity: SevCritical,
		Title:    "Pindahkan " + routeLabel(r) + " ke background job",
		Detail:   fmt.Sprintf("p95 %s, terlama %s. Request selama ini menahan koneksi & worker server, rawan timeout di proxy/CDN (umumnya 30–100 dtk) dan menurunkan kapasitas endpoint lain.", ms(r.P95), ms(r.MaxMS)),
		Action:   "Ubah jadi asinkron: endpoint menerima permintaan → masukkan ke antrean (Redis/DB queue) → balas 202 + id job → worker terpisah memproses → klien polling/SSE/webhook untuk hasil. Worker bisa diskalakan sendiri tanpa menambah server API.",
		Evidence: []string{fmt.Sprintf("%d request, p50 %s, p95 %s, p99 %s", r.Count, ms(r.P50), ms(r.P95), ms(r.P99)), fmt.Sprintf("Memakan %s total waktu server", pct(r.TimeShare))},
		Routes:   []string{routeLabel(r)},
	}
}

func adviceSlowRoute(r RouteRow) Recommendation {
	sev := SevWarning
	if r.P95 >= 3000 {
		sev = SevCritical
	}
	return Recommendation{
		ID: "perf:" + routeLabel(r), Rule: "TRAFFIC-SLOW-ROUTE", Category: "performance", Severity: sev,
		Title:    "Endpoint lambat: " + routeLabel(r),
		Detail:   fmt.Sprintf("p95 %s dari %d request (rata-rata %s). Pengguna merasakan jeda > 1 detik.", ms(r.P95), r.Count, ms(r.AvgMS)),
		Action:   "Profil query endpoint ini: cari N+1 query, tambah index pada kolom WHERE/JOIN/ORDER BY, batasi kolom & baris (paging), dan pindahkan panggilan API luar ke paralel/cache. Bila sudah optimal, cache hasilnya.",
		Evidence: []string{fmt.Sprintf("p50 %s · p95 %s · p99 %s", ms(r.P50), ms(r.P95), ms(r.P99)), fmt.Sprintf("Memakan %s total waktu server", pct(r.TimeShare))},
		Routes:   []string{routeLabel(r)},
	}
}

func advicePerformance(rep *Report) []Recommendation {
	var out []Recommendation
	slowCount := 0
	for _, r := range sortedBy(rep.Routes, func(r RouteRow) float64 { return r.P95 }) {
		if slowCount >= 5 || r.Count < 20 || r.P95 < 1000 {
			continue
		}
		slowCount++
		if r.P95 >= 5000 || r.MaxMS >= 25000 {
			out = append(out, adviceLongRunning(r))
			continue
		}
		out = append(out, adviceSlowRoute(r))
	}
	return out
}

func adviceErrors5xx(rep *Report) []Recommendation {
	var out []Recommendation
	errCount := 0
	for _, r := range sortedBy(rep.Routes, func(r RouteRow) float64 { return float64(r.Err5xx) }) {
		if errCount >= 5 || r.Err5xx < 5 || r.ErrRate < 0.02 {
			continue
		}
		errCount++
		sev := SevWarning
		if r.ErrRate >= 0.1 {
			sev = SevCritical
		}
		out = append(out, Recommendation{
			ID: "errors:" + routeLabel(r), Rule: "TRAFFIC-5XX", Category: "errors", Severity: sev,
			Title:    fmt.Sprintf("%s gagal %s (5xx)", routeLabel(r), pct(r.ErrRate)),
			Detail:   fmt.Sprintf("%d dari %d request berakhir error server. Bila error naik saat traffic ramai, penyebabnya biasanya koneksi DB habis, timeout layanan luar, atau memori.", r.Err5xx, r.Count),
			Action:   "Cek log error endpoint ini pada jam yang sama, tambahkan timeout + retry dengan backoff untuk layanan luar, perbesar/atur pool koneksi DB, dan balas error yang jelas (4xx) untuk input salah alih-alih 500.",
			Evidence: []string{fmt.Sprintf("%d error 5xx", r.Err5xx), fmt.Sprintf("p95 %s", ms(r.P95))},
			Routes:   []string{routeLabel(r)},
		})
	}
	return out
}

// ── Payload ──────────────────────────────────────────────────────────────────

func advicePayload(rep *Report) []Recommendation {
	var out []Recommendation
	for _, r := range rep.Routes {
		if r.Count >= 20 && r.AvgOut >= 512<<10 {
			out = append(out, Recommendation{
				ID: "payload:out:" + routeLabel(r), Rule: "TRAFFIC-PAYLOAD-LARGE", Category: "payload", Severity: SevWarning,
				Title:    "Respons besar: " + routeLabel(r),
				Detail:   fmt.Sprintf("Rata-rata respons %s per request (%d request). Respons besar memperlambat pengguna mobile dan menghabiskan bandwidth server.", bytesH(r.AvgOut), r.Count),
				Action:   "Terapkan paging (limit/cursor), kirim kolom yang dipakai saja, aktifkan kompresi gzip/brotli, dan untuk berkas besar pakai URL object storage/CDN.",
				Evidence: []string{fmt.Sprintf("Rata-rata %s", bytesH(r.AvgOut))}, Routes: []string{routeLabel(r)},
			})
		}
		if r.Count >= 10 && r.AvgIn >= 1<<20 {
			out = append(out, Recommendation{
				ID: "split:upload:" + routeLabel(r), Rule: "TRAFFIC-UPLOAD-HEAVY", Category: "split_service", Severity: SevWarning,
				Title:    "Unggahan besar lewat API: " + routeLabel(r),
				Detail:   fmt.Sprintf("Rata-rata body masuk %s. Unggahan besar menahan worker API selama proses kirim.", bytesH(r.AvgIn)),
				Action:   "Pakai presigned URL agar klien mengunggah langsung ke object storage (S3/R2/MinIO), atau pisahkan service unggah tersendiri dengan batas body & timeout yang berbeda dari API utama.",
				Evidence: []string{fmt.Sprintf("Rata-rata body %s", bytesH(r.AvgIn))}, Routes: []string{routeLabel(r)},
			})
		}
	}
	return out
}

// ── Pecah service ────────────────────────────────────────────────────────────

func isRealGroup(name string) bool {
	return name != RouteNotFound && !strings.HasPrefix(name, "static:")
}

func adviceSplitGroups(rep *Report) []Recommendation {
	realGroups := 0
	for _, g := range rep.Groups {
		if isRealGroup(g.Group) {
			realGroups++
		}
	}
	if realGroups < 3 {
		return nil
	}
	var out []Recommendation
	for _, g := range rep.Groups {
		if !isRealGroup(g.Group) || g.Count < 50 {
			continue
		}
		if g.TimeShare >= 0.4 && g.Share <= 0.25 {
			out = append(out, Recommendation{
				ID: "split:group:" + g.Group, Rule: "TRAFFIC-SPLIT-GROUP", Category: "split_service", Severity: SevWarning,
				Title:    fmt.Sprintf("Kelompok %s layak dipisah jadi service sendiri", g.Group),
				Detail:   fmt.Sprintf("Hanya %s request tetapi memakan %s total waktu server (p95 %s). Saat kelompok ini sibuk, endpoint lain ikut melambat karena berbagi CPU, memori & pool koneksi yang sama.", pct(g.Share), pct(g.TimeShare), ms(g.P95)),
				Action:   "Pisahkan " + g.Group + " ke service/worker sendiri (proses & deploy terpisah, bisa diskalakan sendiri) di belakang reverse proxy yang merutekan per prefix path. Mulai dari memisahkan prosesnya dulu (monorepo, binary kedua) sebelum memecah database.",
				Evidence: []string{fmt.Sprintf("%d request di %d rute", g.Count, g.Routes), fmt.Sprintf("Rata-rata %s, error 5xx %s", ms(g.AvgMS), pct(g.ErrRate))},
				Routes:   []string{g.Group + "/*"},
			})
		}
	}
	return out
}

// ── Kapasitas & load balancer ───────────────────────────────────────────────

func weightedLatency(ps []Point) float64 {
	var s float64
	var n int64
	for _, p := range ps {
		s += p.AvgMS * float64(p.Requests)
		n += p.Requests
	}
	return ratio(s, float64(n))
}

func loadLatencyCorrelation(minutes []Point) float64 {
	xs := make([]float64, 0, len(minutes))
	ys := make([]float64, 0, len(minutes))
	for _, m := range minutes {
		if m.Requests == 0 {
			continue
		}
		xs = append(xs, float64(m.Requests))
		ys = append(ys, m.AvgMS)
	}
	return pearson(xs, ys)
}

func adviceSaturation(rep *Report) []Recommendation {
	if rep.Gran != "m" || len(rep.minutes) < 30 {
		return nil
	}
	t := rep.Totals
	corr := loadLatencyCorrelation(rep.minutes)
	busy := append([]Point(nil), rep.minutes...)
	sort.Slice(busy, func(i, j int) bool { return busy[i].Requests > busy[j].Requests })
	top := busy[:max(1, len(busy)/10)]
	quiet := busy[len(busy)/2:]
	busyLat, quietLat := weightedLatency(top), weightedLatency(quiet)
	if !(corr >= 0.6 && quietLat > 0 && busyLat >= 2*quietLat && busyLat >= 100) {
		return nil
	}
	return []Recommendation{{
		ID: "lb:saturation", Rule: "TRAFFIC-SATURATION", Category: "load_balancer", Severity: SevCritical,
		Title:    "Server melambat saat ramai — kapasitas sudah jenuh",
		Detail:   fmt.Sprintf("Latensi naik seiring jumlah request (korelasi %.2f). Di menit tersibuk rata-rata %s, di menit sepi %s. Ini tanda CPU/DB/pool koneksi sudah penuh, bukan kode yang lambat di semua kondisi.", corr, ms(busyLat), ms(quietLat)),
		Action:   "Tambah instance aplikasi di belakang load balancer (Caddy/Nginx/HAProxy, round-robin + health check). Syaratnya aplikasi stateless: sesi di Redis/JWT, upload di object storage, cron hanya di satu instance. Cek juga DB: bila DB yang jenuh, tambah read replica/cache sebelum menambah instance.",
		Evidence: []string{fmt.Sprintf("Korelasi beban↔latensi %.2f", corr), fmt.Sprintf("Puncak %.0f req/menit, ±%.1f request bersamaan", t.RPMPeak, t.ConcurrencyPeak)},
	}}
}

func adviceInstances(rep *Report) []Recommendation {
	t := rep.Totals
	instances := len(rep.Instances)
	if instances <= 1 {
		if t.RPMPeak >= 1800 || t.ConcurrencyPeak >= 50 {
			return []Recommendation{adviceSingleInstance(t, instances)}
		}
		return nil
	}
	return adviceImbalance(rep)
}

func adviceSingleInstance(t Totals, instances int) Recommendation {
	return Recommendation{
		ID: "lb:single", Rule: "TRAFFIC-SINGLE-INSTANCE", Category: "load_balancer", Severity: SevWarning,
		Title:    "Traffic sudah besar tetapi hanya 1 instance",
		Detail:   fmt.Sprintf("Puncak %.0f req/menit (±%.1f request bersamaan) dilayani satu instance. Restart/deploy atau satu crash = seluruh aplikasi mati.", t.RPMPeak, t.ConcurrencyPeak),
		Action:   "Jalankan minimal 2 instance di belakang load balancer dengan health check & rolling deploy (zero downtime). Pastikan aplikasi stateless dulu.",
		Evidence: []string{fmt.Sprintf("Instance terdeteksi: %d", max(1, instances))},
	}
}

func adviceImbalance(rep *Report) []Recommendation {
	var hi, lo int64 = 0, math.MaxInt64
	for _, in := range rep.Instances {
		hi = max(hi, in.Count)
		lo = min(lo, in.Count)
	}
	if !(lo > 0 && float64(hi)/float64(lo) >= 1.5 && rep.Totals.Requests >= 1000) {
		return nil
	}
	ev := []string{}
	for _, in := range rep.Instances {
		ev = append(ev, fmt.Sprintf("%s: %d request (%s)", in.Value, in.Count, pct(in.Share)))
	}
	return []Recommendation{{
		ID: "lb:imbalance", Rule: "TRAFFIC-LB-IMBALANCE", Category: "load_balancer", Severity: SevWarning,
		Title:    "Beban antar instance tidak seimbang",
		Detail:   fmt.Sprintf("Instance tersibuk menerima %.1f× instance tersepi. Satu instance bisa jenuh sementara yang lain menganggur.", float64(hi)/float64(lo)),
		Action:   "Periksa algoritma load balancer (pakai least_conn/round-robin), matikan sticky session bila tidak perlu, dan pastikan health check mengeluarkan instance yang lambat.",
		Evidence: ev,
	}}
}

func adviceWriteHeavy(rep *Report) []Recommendation {
	t := rep.Totals
	if !(t.Requests >= 1000 && t.ReadShare <= 0.4) {
		return nil
	}
	return []Recommendation{{
		ID: "capacity:write", Rule: "TRAFFIC-WRITE-HEAVY", Category: "capacity", Severity: SevInfo,
		Title:    "Traffic didominasi penulisan (POST/PUT/DELETE)",
		Detail:   fmt.Sprintf("Hanya %s request yang membaca (GET). Cache tidak banyak membantu; yang menentukan kapasitas adalah database tulis.", pct(t.ReadShare)),
		Action:   "Kumpulkan tulis yang bisa ditunda ke antrean & tulis secara batch, pastikan index tidak berlebihan di tabel yang sering ditulis, dan pantau lock/antrean koneksi DB. SQLite: aktifkan WAL; beban tulis tinggi lintas instance → MySQL/Postgres.",
		Evidence: []string{fmt.Sprintf("%s baca, %s tulis", pct(t.ReadShare), pct(1-t.ReadShare))},
	}}
}

func sortedBy(rows []RouteRow, key func(RouteRow) float64) []RouteRow {
	out := append([]RouteRow(nil), rows...)
	sort.SliceStable(out, func(i, j int) bool { return key(out[i]) > key(out[j]) })
	return out
}
