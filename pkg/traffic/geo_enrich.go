package traffic

import (
	"net/netip"
	"strconv"
	"sync"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/traffic/geo"
)

// GeoResolver mencari lokasi & jaringan satu IP; bisa digantikan di uji.
type GeoResolver interface {
	Lookup(ip string) geo.Info
}

type defaultResolver struct{}

func (defaultResolver) Lookup(ip string) geo.Info { return geo.Default().Lookup(ip) }

var (
	resolverMu sync.RWMutex
	resolver   GeoResolver = defaultResolver{}
)

// SetGeoResolver mengganti resolver (nil = tanpa pengayaan geo) dan mengembalikan yang lama.
func SetGeoResolver(r GeoResolver) GeoResolver {
	resolverMu.Lock()
	defer resolverMu.Unlock()
	old := resolver
	resolver = r
	return old
}

func currentResolver() GeoResolver {
	resolverMu.RLock()
	defer resolverMu.RUnlock()
	return resolver
}

// geoFields = hasil pengayaan geo satu event.
type geoFields struct {
	city, org, netType string
	asn                uint32
}

// applyGeo memperkaya event dari basis data offline. IP yang disamarkan SDK (bukan IP valid) dilewati:
// negara hanya dari header CDN. Negara dari header CDN tidak pernah ditimpa; kosong → diisi dari geo.
func applyGeo(n *normEvent) {
	r := currentResolver()
	if r == nil {
		return
	}
	if _, err := netip.ParseAddr(n.ip); err != nil {
		return
	}
	info := r.Lookup(n.ip)
	if n.country == "" {
		n.country = clip(info.Country, 8)
	}
	n.geo = geoFields{city: clip(info.City, 64), asn: info.ASN, org: clip(info.Org, 128), netType: info.NetType}
}

func asnLabel(asn uint32) string {
	if asn == 0 {
		return ""
	}
	return "AS" + strconv.FormatUint(uint64(asn), 10)
}

// cityLabel = "Kota, ID" agar kota senama di negara berbeda tidak tergabung.
func cityLabel(city, country string) string {
	if city == "" || country == "" {
		return city
	}
	return city + ", " + country
}

func (a *ingestAcc) addGeoDims(n normEvent) {
	g := n.geo
	a.addDim(models.TrafficGranHour, n.hour, "city", cityLabel(g.city, n.country), n.isErr)
	a.addDim(models.TrafficGranHour, n.hour, "asn", asnLabel(g.asn), n.isErr)
	a.addDim(models.TrafficGranHour, n.hour, "org", g.org, n.isErr)
	a.addDim(models.TrafficGranHour, n.hour, "net_type", g.netType, n.isErr)
}

func withGeo(ev models.TrafficRecentEvent, n normEvent) models.TrafficRecentEvent {
	ev.City, ev.ASN, ev.Org, ev.NetType = n.geo.city, n.geo.asn, n.geo.org, n.geo.netType
	return ev
}

func fillGeoLists(rep *Report, dims map[string][]KV) {
	rep.Cities = topKV(dims, "city", 15)
	rep.ASNs = topKV(dims, "asn", 15)
	rep.Orgs = topKV(dims, "org", 15)
	rep.NetTypes = topKV(dims, "net_type", len(geo.NetTypes))
}
