package traffic

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/traffic/geo"
)

// fakeGeo = resolver uji berbasis peta; mencatat IP yang dicari.
type fakeGeo struct {
	data  map[string]geo.Info
	calls []string
}

func (f *fakeGeo) Lookup(ip string) geo.Info {
	f.calls = append(f.calls, ip)
	return f.data[ip]
}

func useFakeGeo(t *testing.T, data map[string]geo.Info) *fakeGeo {
	t.Helper()
	f := &fakeGeo{data: data}
	old := SetGeoResolver(f)
	t.Cleanup(func() { SetGeoResolver(old) })
	return f
}

var (
	jakartaHome = geo.Info{Country: "ID", City: "Jakarta", ASN: 7713, Org: "PT Telekomunikasi Indonesia", NetType: geo.NetResidential}
	surabayaMob = geo.Info{Country: "ID", City: "Surabaya", ASN: 23693, Org: "PT Telekomunikasi Selular", NetType: geo.NetMobile}
	ashburnDC   = geo.Info{Country: "US", City: "Ashburn", ASN: 16509, Org: "Amazon.com, Inc.", NetType: geo.NetHosting}
)

func freezeTime(t *testing.T, at time.Time) {
	t.Helper()
	timeNow = func() time.Time { return at }
	t.Cleanup(func() { timeNow = time.Now })
}

// Laporan 1 jam memuat kota, ISP/organisasi, dan jenis jaringan dari event yang masuk lewat Ingest.
func TestReport1hShowsCityOrgNetType(t *testing.T) {
	frozen := time.Date(2026, 1, 1, 12, 0, 30, 0, time.UTC)
	freezeTime(t, frozen)
	useFakeGeo(t, map[string]geo.Info{"203.0.113.9": jakartaHome, "198.51.100.7": surabayaMob, "192.0.2.5": ashburnDC})
	db := testDB(t)
	src := models.TrafficSource{Name: "geo", Mode: models.TrafficModeDirect, IngestKey: "geo", Enabled: true}
	db.Create(&src)

	ms := frozen.UnixMilli()
	ev := func(off int64, ip, country string) Event {
		return Event{TS: ms - off, Method: "GET", Path: "/api/items/1", Status: 200, DurMS: 20, IP: ip, Country: country, UA: "Mozilla/5.0 Chrome/120 Safari/537"}
	}
	var events []Event
	for i := int64(0); i < 6; i++ {
		events = append(events, ev(1000+i, "203.0.113.9", ""))
	}
	for i := int64(0); i < 3; i++ {
		events = append(events, ev(2000+i, "198.51.100.7", "ID"))
	}
	events = append(events, ev(3000, "192.0.2.5", ""), ev(4000, "10.0.0.1", ""), ev(5000, "anon-1f3a", "SG"), ev(5500, "anon-1f3a", "SG"))
	if _, err := Ingest(db, &src, events, "api-1"); err != nil {
		t.Fatal(err)
	}
	rep, err := BuildReport(db, src, ParseWindow("1h"))
	if err != nil {
		t.Fatal(err)
	}
	rep.Source = models.TrafficSource{}
	var recent []models.TrafficRecentEvent
	db.Order("id").Find(&recent)
	for i := range recent {
		recent[i].SourceID, recent[i].ID = 0, 0
	}
	checkGolden(t, "geo_report_1h", map[string]any{
		"countries": rep.Countries, "cities": rep.Cities, "asns": rep.ASNs, "orgs": rep.Orgs, "net_types": rep.NetTypes, "recent": recent,
	})
	if len(rep.Cities) == 0 || rep.Cities[0].Value != "Jakarta, ID" || rep.Cities[0].Count != 6 {
		t.Errorf("kota teratas salah: %+v", rep.Cities)
	}
	if got := kvCount(rep.NetTypes, geo.NetMobile); got != 3 {
		t.Errorf("seluler = %d, want 3", got)
	}
}

func TestIngestGeoRules(t *testing.T) {
	frozen := time.Date(2026, 1, 1, 12, 0, 30, 0, time.UTC)
	freezeTime(t, frozen)
	f := useFakeGeo(t, map[string]geo.Info{"203.0.113.9": jakartaHome, "anon-1f3a": jakartaHome})
	db := testDB(t)
	src := models.TrafficSource{Name: "r", Mode: models.TrafficModeDirect, IngestKey: "r", Enabled: true}
	db.Create(&src)
	base := Event{TS: frozen.UnixMilli() - 1000, Method: "GET", Path: "/x", Status: 200}
	withIP := base
	withIP.IP, withIP.Country = "203.0.113.9", "SG" // header CDN menang atas geo
	masked := base
	masked.IP, masked.Country = "anon-1f3a", "MY"
	if _, err := Ingest(db, &src, []Event{withIP, masked}, "i"); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 1 || f.calls[0] != "203.0.113.9" {
		t.Errorf("IP samaran tidak boleh dicari di geo: %v", f.calls)
	}
	var rows []models.TrafficRecentEvent
	db.Order("id").Find(&rows)
	if rows[0].Country != "SG" || rows[0].City != "Jakarta" || rows[0].NetType != geo.NetResidential {
		t.Errorf("event pertama: %+v", rows[0])
	}
	if rows[1].Country != "MY" || rows[1].City != "" || rows[1].ASN != 0 {
		t.Errorf("IP samaran hanya boleh membawa negara CDN: %+v", rows[1])
	}
}

func TestIngestWithoutResolverLeavesGeoEmpty(t *testing.T) {
	old := SetGeoResolver(nil)
	defer SetGeoResolver(old)
	db := testDB(t)
	src := models.TrafficSource{Name: "n", Mode: models.TrafficModeDirect, IngestKey: "n", Enabled: true}
	db.Create(&src)
	if _, err := Ingest(db, &src, []Event{{Method: "GET", Path: "/", Status: 200, IP: "203.0.113.9"}}, "i"); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&models.TrafficDimStat{}).Where("dim IN ?", []string{"city", "asn", "org", "net_type"}).Count(&n)
	if n != 0 {
		t.Errorf("tanpa resolver tidak boleh ada dimensi geo: %d", n)
	}
	rep, _ := BuildReport(db, src, ParseWindow("1h"))
	b, _ := json.Marshal(rep)
	if strings.Contains(string(b), `"cities"`) || strings.Contains(string(b), `"net_types"`) {
		t.Error("kolom geo kosong harus tidak muncul di JSON laporan")
	}
}

func netTypeReport(requests int64, nt []KV, asns []KV) *Report {
	r := &Report{Gran: "m"}
	r.Totals = Totals{Requests: requests, RPSAvg: 0.1, RPMPeak: 5, ReadShare: 0.9, P95: 80, SpanMinutes: 60}
	r.Routes = []RouteRow{{Method: "GET", Route: "/x", Count: requests}}
	r.NetTypes, r.ASNs = nt, asns
	return r
}

func ruleSet(recs []Recommendation) map[string]bool {
	out := map[string]bool{}
	for _, r := range recs {
		out[r.Rule] = true
	}
	return out
}

func TestAdviseDatacenterAndASN(t *testing.T) {
	positive := netTypeReport(1000,
		[]KV{{Value: geo.NetHosting, Count: 600}, {Value: geo.NetVPN, Count: 100}, {Value: geo.NetResidential, Count: 200}, {Value: geo.NetMobile, Count: 100}},
		[]KV{{Value: "AS14061", Count: 560, Err: 30}, {Value: "AS7713", Count: 200}})
	got := Advise(positive)
	rules := ruleSet(got)
	if !rules["TRAFFIC-DATACENTER-DOMINANT"] || !rules["TRAFFIC-ASN-SPIKE"] {
		t.Fatalf("saran positif tidak muncul: %v", rules)
	}
	if rules["TRAFFIC-HEALTHY"] {
		t.Error("ada masalah, saran 'sehat' tidak boleh muncul")
	}
	checkGolden(t, "geo_advise_positive", got)
}

func TestAdviseGeoNegatives(t *testing.T) {
	cases := map[string]*Report{
		"sehat: mayoritas rumahan & seluler": netTypeReport(1000,
			[]KV{{Value: geo.NetResidential, Count: 600}, {Value: geo.NetMobile, Count: 300}, {Value: geo.NetHosting, Count: 100}},
			[]KV{{Value: "AS7713", Count: 600}, {Value: "AS23693", Count: 300}}),
		"ISP rumahan besar dominan bukan lonjakan": netTypeReport(1000,
			[]KV{{Value: geo.NetResidential, Count: 800}, {Value: geo.NetHosting, Count: 200}},
			[]KV{{Value: "AS64500", Count: 800}}), // tak ada di tabel, tetapi trafik rumahan sebesar itu
		"datacenter di bawah ambang": netTypeReport(1000,
			[]KV{{Value: geo.NetHosting, Count: 450}, {Value: geo.NetResidential, Count: 550}}, nil),
		"sampel geo terlalu sedikit": netTypeReport(1000,
			[]KV{{Value: geo.NetHosting, Count: 90}}, []KV{{Value: "AS14061", Count: 150}}),
		"tanpa data geo": netTypeReport(1000, nil, nil),
	}
	for name, rep := range cases {
		rules := ruleSet(Advise(rep))
		if rules["TRAFFIC-DATACENTER-DOMINANT"] || rules["TRAFFIC-ASN-SPIKE"] {
			t.Errorf("%s: saran geo tidak boleh muncul: %v", name, rules)
		}
		if !rules["TRAFFIC-HEALTHY"] {
			t.Errorf("%s: seharusnya dianggap sehat: %v", name, rules)
		}
	}
}

func TestMaskIP(t *testing.T) {
	cases := map[string]string{
		"203.0.113.77":               "203.0.113.0/24",
		"::ffff:203.0.113.77":        "203.0.113.0/24",
		"2001:db8:abcd:12:34::1":     "2001:db8:abcd::/48",
		"203.0.113.0/24":             "",
		"anon-1f3a":                  "",
		"":                           "",
		"2001:db8:abcd:ffff:1:2:3:4": "2001:db8:abcd::/48",
	}
	for in, want := range cases {
		got, ok := MaskIP(in)
		if want == "" && ok || want != "" && (!ok || got != want) {
			t.Errorf("MaskIP(%q) = %q,%v want %q", in, got, ok, want)
		}
	}
}

func TestIPRetentionEnv(t *testing.T) {
	if IPRetention() != 7*24*time.Hour {
		t.Error("bawaan harus 7 hari")
	}
	for env, want := range map[string]time.Duration{"3": 72 * time.Hour, "0": 7 * 24 * time.Hour, "-2": 7 * 24 * time.Hour, "abc": 7 * 24 * time.Hour} {
		t.Setenv(IPRetentionEnv, env)
		if got := IPRetention(); got != want {
			t.Errorf("env %q: %v want %v", env, got, want)
		}
	}
}

func TestPruneMasksOldIPs(t *testing.T) {
	db := testDB(t)
	src := models.TrafficSource{Name: "p", Mode: models.TrafficModeDirect, IngestKey: "p", Enabled: true}
	db.Create(&src)
	now := time.Now()
	oldH := now.Add(-10*24*time.Hour).Unix() / 3600 * 3600
	newH := now.Add(-2*time.Hour).Unix() / 3600 * 3600
	dim := func(bucket int64, d, v string, n, e int64) {
		db.Create(&models.TrafficDimStat{SourceID: src.ID, Gran: models.TrafficGranHour, Bucket: bucket, Dim: d, Value: v, Count: n, Err: e})
	}
	dim(oldH, "ip", "203.0.113.10", 5, 1)
	dim(oldH, "ip", "203.0.113.200", 7, 2) // bertabrakan dengan baris di atas setelah masking
	dim(oldH, "ip", "198.51.100.4", 3, 0)
	dim(oldH, "ip", "2001:db8:abcd:1::9", 4, 0)
	dim(oldH, "ip", "anon-1f3a", 2, 0) // bukan IP: dibiarkan
	dim(oldH, "auth_ip", "203.0.113.10", 6, 6)
	dim(oldH, "country", "ID", 99, 0)     // dimensi lain tidak disentuh
	dim(newH, "ip", "203.0.113.10", 8, 0) // masih dalam masa simpan
	old := now.Add(-10 * 24 * time.Hour)
	db.Create(&models.TrafficRecentEvent{SourceID: src.ID, At: old, IP: "203.0.113.10"})
	db.Create(&models.TrafficRecentEvent{SourceID: src.ID, At: now, IP: "203.0.113.10"})

	Prune(db)
	Prune(db) // idempoten

	get := func(bucket int64, d, v string) *models.TrafficDimStat {
		var r models.TrafficDimStat
		if db.Where("bucket = ? AND dim = ? AND value = ?", bucket, d, v).Take(&r).Error != nil {
			return nil
		}
		return &r
	}
	if r := get(oldH, "ip", "203.0.113.0/24"); r == nil || r.Count != 12 || r.Err != 3 {
		t.Errorf("baris bertabrakan harus digabung (12/3): %+v", r)
	}
	if get(oldH, "ip", "203.0.113.10") != nil || get(oldH, "ip", "203.0.113.200") != nil {
		t.Error("IP penuh lama harus hilang")
	}
	if r := get(oldH, "ip", "198.51.100.0/24"); r == nil || r.Count != 3 {
		t.Errorf("IPv4 /24: %+v", r)
	}
	if r := get(oldH, "ip", "2001:db8:abcd::/48"); r == nil || r.Count != 4 {
		t.Errorf("IPv6 /48: %+v", r)
	}
	if r := get(oldH, "auth_ip", "203.0.113.0/24"); r == nil || r.Count != 6 || r.Err != 6 {
		t.Errorf("auth_ip: %+v", r)
	}
	if get(oldH, "ip", "anon-1f3a") == nil || get(oldH, "country", "ID") == nil {
		t.Error("nilai non-IP dan dimensi lain harus utuh")
	}
	if get(newH, "ip", "203.0.113.10") == nil {
		t.Error("IP dalam masa simpan harus tetap penuh")
	}
	var sum int64
	db.Model(&models.TrafficDimStat{}).Where("dim = ? AND bucket = ?", "ip", oldH).Select("SUM(count)").Scan(&sum)
	if sum != 5+7+3+4+2 {
		t.Errorf("total hitungan agregat harus tetap: %d", sum)
	}
	var evs []models.TrafficRecentEvent
	db.Order("id").Find(&evs)
	if evs[0].IP != "203.0.113.0/24" || evs[1].IP != "203.0.113.10" {
		t.Errorf("IP ekor event: %q, %q", evs[0].IP, evs[1].IP)
	}
}

func TestPruneRetentionFromEnv(t *testing.T) {
	t.Setenv(IPRetentionEnv, "2")
	db := testDB(t)
	src := models.TrafficSource{Name: "e", Mode: models.TrafficModeDirect, IngestKey: "e", Enabled: true}
	db.Create(&src)
	b := time.Now().Add(-3*24*time.Hour).Unix() / 3600 * 3600
	db.Create(&models.TrafficDimStat{SourceID: src.ID, Gran: models.TrafficGranHour, Bucket: b, Dim: "ip", Value: "203.0.113.10", Count: 1})
	Prune(db)
	var r models.TrafficDimStat
	db.First(&r)
	if r.Value != "203.0.113.0/24" {
		t.Errorf("retensi 2 hari harus menyamarkan baris 3 hari: %q", r.Value)
	}
}
