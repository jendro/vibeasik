package geo

import (
	"strings"
	"unicode"
)

// Jenis jaringan pengunjung (kunci stabil; dipakai sebagai nilai dimensi dan di API).
const (
	NetMobile      = "mobile"
	NetResidential = "residential"
	NetHosting     = "hosting"
	NetVPN         = "vpn"
	NetUnknown     = "unknown"
)

// NetTypes = urutan tampil yang tetap.
var NetTypes = []string{NetMobile, NetResidential, NetHosting, NetVPN, NetUnknown}

// asnTypes = ASN yang jenisnya sudah diketahui pasti (operator seluler & ISP rumahan Indonesia, cloud besar, VPN).
var asnTypes = map[uint32]string{
	// Seluler Indonesia
	23693: NetMobile, // Telkomsel
	4761:  NetMobile, // Indosat Ooredoo Hutchison
	24203: NetMobile, // XL Axiata
	45727: NetMobile, // Indosat Ooredoo Hutchison (Tri)
	17885: NetMobile, // XL Axiata (lama)
	18004: NetMobile, // Smartfren
	// ISP rumahan Indonesia
	7713:  NetResidential, // Telkom Indonesia (IndiHome)
	23700: NetResidential, // Linknet (First Media)
	9905:  NetResidential, // Linknet
	17670: NetResidential, // MNC Play (Mediacom)
	17451: NetResidential, // Biznet
	// Cloud & hosting besar
	13335:  NetHosting, // Cloudflare
	15169:  NetHosting, // Google
	396982: NetHosting, // Google Cloud
	16509:  NetHosting, // Amazon
	14618:  NetHosting, // Amazon
	8075:   NetHosting, // Microsoft
	14061:  NetHosting, // DigitalOcean
	63949:  NetHosting, // Akamai / Linode
	20473:  NetHosting, // Vultr
	16276:  NetHosting, // OVHcloud
	24940:  NetHosting, // Hetzner
	45102:  NetHosting, // Alibaba Cloud
	132203: NetHosting, // Tencent Cloud
	31898:  NetHosting, // Oracle Cloud
	51167:  NetHosting, // Contabo
	8560:   NetHosting, // IONOS
	// Penyedia VPN / proxy
	9009:   NetVPN, // M247
	212238: NetVPN, // Datacamp
	60068:  NetVPN, // Datacamp
	39351:  NetVPN, // Mullvad
	62371:  NetVPN, // Proton
}

// orgHint = kata kunci nama organisasi (huruf kecil). Kata pendek/ambigu harus cocok sebagai kata utuh
// (whole) agar "mobile" tidak cocok dengan "automobile"; kata khas boleh cocok sebagai potongan ("fastvpn").
type orgHint struct {
	typ, word string
	whole     bool
}

// orgHints dicek berurutan: VPN lebih dulu karena banyak penyedia VPN menyewa server di perusahaan hosting.
var orgHints = []orgHint{
	{NetVPN, "vpn", false}, {NetVPN, "mullvad", false}, {NetVPN, "surfshark", false}, {NetVPN, "proxy", true}, {NetVPN, "tor exit", true},
	{NetMobile, "telkomsel", false}, {NetMobile, "indosat", false}, {NetMobile, "xl axiata", false}, {NetMobile, "smartfren", false},
	{NetMobile, "hutchison", true}, {NetMobile, "mobile", true}, {NetMobile, "cellular", true}, {NetMobile, "telecom selular", true},
	{NetHosting, "hosting", false}, {NetHosting, "datacenter", false}, {NetHosting, "data center", true}, {NetHosting, "cloud", true},
	{NetHosting, "server", true}, {NetHosting, "vps", true}, {NetHosting, "amazon", true}, {NetHosting, "google", true},
	{NetHosting, "microsoft", true}, {NetHosting, "digitalocean", false}, {NetHosting, "hetzner", false}, {NetHosting, "ovh", true},
	{NetResidential, "telkom", true}, {NetResidential, "indihome", false}, {NetResidential, "first media", true}, {NetResidential, "biznet", false},
	{NetResidential, "broadband", true}, {NetResidential, "cable", true}, {NetResidential, "fiber", true}, {NetResidential, "internet service", true},
	{NetResidential, "telecommunication", true},
}

// Classify menentukan jenis jaringan dari ASN (tabel) lalu nama organisasi (kata kunci); tak dikenali → unknown.
func Classify(asn uint32, org string) string {
	if t, ok := asnTypes[asn]; ok {
		return t
	}
	lower, words := strings.ToLower(org), wordsOf(org)
	for _, h := range orgHints {
		if h.whole && strings.Contains(words, " "+h.word+" ") || !h.whole && strings.Contains(lower, h.word) {
			return h.typ
		}
	}
	return NetUnknown
}

// wordsOf = nama organisasi huruf kecil, dipisah spasi tunggal dan diapit spasi, agar kata kunci
// cocok sebagai kata utuh ("mobile" tidak cocok dengan "automobile").
func wordsOf(org string) string {
	fields := strings.FieldsFunc(strings.ToLower(org), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	return " " + strings.Join(fields, " ") + " "
}

// ClassifyASN hanya memakai tabel ASN; kosong bila ASN tidak ada di tabel.
func ClassifyASN(asn uint32) string { return asnTypes[asn] }
