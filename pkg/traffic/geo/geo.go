// Package geo membaca basis data lokasi & jaringan IP offline (DB-IP Lite, format MMDB) milik pengguna.
// Tidak ada IP pengunjung yang dikirim ke pihak lain: pencarian terjadi di mesin ini.
//
// IP Geolocation by DB-IP (https://db-ip.com), lisensi CC BY 4.0.
package geo

import (
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oschwald/maxminddb-golang/v2"

	"github.com/vibeasik/vibeasik/pkg/localauth"
)

// Nama berkas basis data di folder geo.
const (
	CityFile = "dbip-city-lite.mmdb"
	ASNFile  = "dbip-asn-lite.mmdb"
)

// Info = hasil pencarian satu IP. Semua field boleh kosong (basis data belum terpasang atau IP tidak dikenal).
type Info struct {
	Country string `json:"country,omitempty"` // kode ISO 3166-1 alpha-2
	City    string `json:"city,omitempty"`
	ASN     uint32 `json:"asn,omitempty"`
	Org     string `json:"org,omitempty"`
	NetType string `json:"net_type,omitempty"` // mobile | residential | hosting | vpn | unknown; kosong bila ASN tidak diketahui
}

// Empty true bila tidak ada satu pun keterangan.
func (i Info) Empty() bool { return i == Info{} }

// Dir = folder basis data geo di direktori data engine.
func Dir() string { return filepath.Join(localauth.Dir(), "geo") }

type cityRecord struct {
	City struct {
		Names struct {
			En string `maxminddb:"en"`
		} `maxminddb:"names"`
	} `maxminddb:"city"`
	Country struct {
		ISO string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

type asnRecord struct {
	Number uint32 `maxminddb:"autonomous_system_number"`
	Org    string `maxminddb:"autonomous_system_organization"`
}

// DB menampung pembaca MMDB; aman dipakai paralel. Berkas yang tidak ada diperlakukan sebagai kosong.
type DB struct {
	dir  string
	mu   sync.RWMutex
	city *maxminddb.Reader
	asn  *maxminddb.Reader

	stamp     string       // "mtime|mtime" berkas saat terakhir dimuat
	lastCheck atomic.Int64 // unix nano pemeriksaan perubahan berkas terakhir
}

// recheckEvery = jarak minimum antar pemeriksaan perubahan berkas (mis. setelah `geo update` dari proses lain).
const recheckEvery = 30 * time.Second

func (d *DB) fileStamp() string {
	var b strings.Builder
	for _, name := range []string{CityFile, ASNFile} {
		if fi, err := os.Stat(filepath.Join(d.dir, name)); err == nil {
			b.WriteString(fi.ModTime().UTC().Format(time.RFC3339Nano))
		}
		b.WriteByte('|')
	}
	return b.String()
}

// refreshIfChanged memuat ulang bila berkas diganti proses lain; murah (dua stat per 30 detik).
func (d *DB) refreshIfChanged() {
	now := time.Now().UnixNano()
	last := d.lastCheck.Load()
	if now-last < int64(recheckEvery) || !d.lastCheck.CompareAndSwap(last, now) {
		return
	}
	d.mu.RLock()
	same := d.stamp == d.fileStamp()
	d.mu.RUnlock()
	if !same {
		d.Reload()
	}
}

// New membuat DB untuk folder tertentu dan langsung memuat berkas yang ada.
func New(dir string) *DB {
	d := &DB{dir: dir}
	d.Reload()
	return d
}

var (
	defaultMu sync.Mutex
	defaultDB *DB
)

// Default = DB pada Dir() (dimuat sekali, malas).
func Default() *DB {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultDB == nil {
		defaultDB = New(Dir())
	}
	return defaultDB
}

// Reload membuka ulang kedua berkas (dipanggil setelah `geo update`). Berkas rusak/tidak ada → pembaca itu kosong.
func (d *DB) Reload() {
	stamp := d.fileStamp()
	city := openOrNil(filepath.Join(d.dir, CityFile))
	asn := openOrNil(filepath.Join(d.dir, ASNFile))
	d.mu.Lock()
	oldCity, oldASN := d.city, d.asn
	d.city, d.asn, d.stamp = city, asn, stamp
	d.mu.Unlock()
	closeReader(oldCity)
	closeReader(oldASN)
}

// Release menutup pembaca (melepas pemetaan berkas) tanpa membukanya lagi; dipakai sebelum berkas diganti.
func (d *DB) Release() {
	d.mu.Lock()
	oldCity, oldASN := d.city, d.asn
	d.city, d.asn = nil, nil
	d.mu.Unlock()
	closeReader(oldCity)
	closeReader(oldASN)
}

func openOrNil(path string) *maxminddb.Reader {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	r, err := maxminddb.Open(path)
	if err != nil {
		return nil
	}
	return r
}

func closeReader(r *maxminddb.Reader) {
	if r != nil {
		_ = r.Close()
	}
}

// Status = keadaan basis data untuk `geo status` dan Studio.
type Status struct {
	Dir       string `json:"-"`
	CityReady bool   `json:"city_ready"`
	ASNReady  bool   `json:"asn_ready"`
	CityBuilt int64  `json:"city_built,omitempty"` // unix detik waktu pembuatan basis data
	ASNBuilt  int64  `json:"asn_built,omitempty"`
}

// Ready true bila minimal salah satu basis data termuat.
func (s Status) Ready() bool { return s.CityReady || s.ASNReady }

func (d *DB) Status() Status {
	d.mu.RLock()
	defer d.mu.RUnlock()
	s := Status{Dir: d.dir}
	if d.city != nil {
		s.CityReady, s.CityBuilt = true, int64(d.city.Metadata.BuildEpoch)
	}
	if d.asn != nil {
		s.ASNReady, s.ASNBuilt = true, int64(d.asn.Metadata.BuildEpoch)
	}
	return s
}

// Lookup mencari satu IP. IP tidak valid, privat/lokal, atau basis data kosong → Info kosong tanpa error.
func (d *DB) Lookup(ip string) Info {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil || !isPublic(addr) {
		return Info{}
	}
	addr = addr.Unmap()
	d.refreshIfChanged()
	d.mu.RLock()
	defer d.mu.RUnlock()
	var out Info
	if d.city != nil {
		var rec cityRecord
		if res := d.city.Lookup(addr); res.Found() && res.Decode(&rec) == nil {
			out.Country, out.City = strings.ToUpper(rec.Country.ISO), rec.City.Names.En
		}
	}
	if d.asn != nil {
		var rec asnRecord
		if res := d.asn.Lookup(addr); res.Found() && res.Decode(&rec) == nil && rec.Number != 0 {
			out.ASN, out.Org = rec.Number, rec.Org
			out.NetType = Classify(rec.Number, rec.Org)
		}
	}
	return out
}

func isPublic(a netip.Addr) bool {
	return a.IsValid() && !a.IsPrivate() && !a.IsLoopback() && !a.IsLinkLocalUnicast() &&
		!a.IsMulticast() && !a.IsUnspecified() && !a.IsLinkLocalMulticast()
}
