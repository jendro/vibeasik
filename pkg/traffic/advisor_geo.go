package traffic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/traffic/geo"
)

// Ambang saran jaringan: hanya memicu bila cakupan data geo cukup dan pola jelas menyimpang.
const (
	geoMinClassified    = 100 // request ber-jenis jaringan minimal agar persentase bermakna
	datacenterShareMin  = 0.5 // porsi hosting+VPN dari request terklasifikasi
	asnConcentrationMin = 0.5 // porsi satu ASN dari seluruh request
	asnMinRequests      = 200
)

func kvCount(list []KV, value string) int64 {
	for _, kv := range list {
		if kv.Value == value {
			return kv.Count
		}
	}
	return 0
}

func totalCount(list []KV) int64 {
	var n int64
	for _, kv := range list {
		n += kv.Count
	}
	return n
}

// adviceDatacenterTraffic: pengunjung manusia jarang datang dari datacenter/VPN; dominasinya biasanya bot,
// scraper, atau panggilan antar-server.
func adviceDatacenterTraffic(rep *Report) []Recommendation {
	classified := totalCount(rep.NetTypes)
	if classified < geoMinClassified {
		return nil
	}
	dc := kvCount(rep.NetTypes, geo.NetHosting) + kvCount(rep.NetTypes, geo.NetVPN)
	share := ratio(float64(dc), float64(classified))
	if share < datacenterShareMin {
		return nil
	}
	return []Recommendation{{
		ID: "security:datacenter", Rule: "TRAFFIC-DATACENTER-DOMINANT", Category: "security", Severity: SevWarning,
		Title:  "Trafik dari datacenter/VPN mendominasi",
		Detail: fmt.Sprintf("%s request berasal dari jaringan hosting/VPN, bukan dari jaringan seluler atau rumahan. Itu pola khas bot, scraper, atau panggilan antar-server; bila aplikasi ini memang dipakai server lain (API publik), abaikan saran ini.", pct(share)),
		Action: "Cek apakah rute teratas memang dipakai integrasi server. Bila tidak, pasang rate limit lebih ketat atau tantangan bot untuk jaringan hosting/VPN dan jangan menghitungnya sebagai pengguna sungguhan di metrik produk.",
		Evidence: []string{
			fmt.Sprintf("hosting %d, VPN %d dari %d request terklasifikasi", kvCount(rep.NetTypes, geo.NetHosting), kvCount(rep.NetTypes, geo.NetVPN), classified),
			fmt.Sprintf("seluler %d, rumahan %d", kvCount(rep.NetTypes, geo.NetMobile), kvCount(rep.NetTypes, geo.NetResidential)),
		},
	}}
}

// adviceASNConcentration: satu ASN menguasai sebagian besar request. ISP seluler/rumahan besar yang wajar
// dominan di pasar lokal tidak dihitung; yang dicurigai adalah hosting/VPN/tak dikenal.
func adviceASNConcentration(rep *Report) []Recommendation {
	if len(rep.ASNs) == 0 {
		return nil
	}
	top := rep.ASNs[0]
	asn, _ := strconv.ParseUint(strings.TrimPrefix(top.Value, "AS"), 10, 32)
	switch geo.ClassifyASN(uint32(asn)) {
	case geo.NetMobile, geo.NetResidential:
		return nil
	}
	// Jenis jaringan ASN yang tak ada di tabel ikut ditimbang: bila trafik seluler+rumahan sebesar ASN ini,
	// ASN itu hampir pasti ISP biasa yang dominan di pasar lokal.
	if kvCount(rep.NetTypes, geo.NetMobile)+kvCount(rep.NetTypes, geo.NetResidential) >= top.Count {
		return nil
	}
	share := ratio(float64(top.Count), float64(rep.Totals.Requests))
	if top.Count < asnMinRequests || share < asnConcentrationMin {
		return nil
	}
	return []Recommendation{{
		ID: "security:asn:" + top.Value, Rule: "TRAFFIC-ASN-SPIKE", Category: "security", Severity: SevWarning,
		Title:  fmt.Sprintf("Lonjakan dari satu jaringan (%s)", top.Value),
		Detail: fmt.Sprintf("%s dari seluruh request datang dari satu ASN (%s). Pengguna sungguhan tersebar di banyak ISP; konsentrasi setinggi ini biasanya satu pelaku (bot, scraper, atau serangan) atau satu integrasi.", pct(share), top.Value),
		Action: "Lihat IP teratas dari jaringan ini. Bila bukan integrasi yang dikenal, batasi lewat rate limit per ASN/rentang IP di CDN atau firewall.",
		Evidence: []string{
			fmt.Sprintf("%s: %d request (%s)", top.Value, top.Count, pct(share)),
			fmt.Sprintf("kesalahan dari ASN ini: %d", top.Err),
		},
	}}
}
