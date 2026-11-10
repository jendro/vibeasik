package geo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupFixture(t *testing.T) {
	db := New(writeFixtureDB(t))
	defer db.Release()
	cases := []struct {
		ip   string
		want Info
	}{
		{"203.0.113.9", Info{Country: "ID", City: "Jakarta", ASN: 7713, Org: "PT Telekomunikasi Indonesia", NetType: NetResidential}},
		{"198.51.100.200", Info{Country: "ID", City: "Surabaya", ASN: 23693, Org: "PT Telekomunikasi Selular", NetType: NetMobile}},
		{"192.0.2.1", Info{Country: "US", City: "Ashburn", ASN: 16509, Org: "Amazon.com, Inc.", NetType: NetHosting}},
		{"100.64.1.1", Info{Country: "SG", ASN: 64500, Org: "Example VPN Services", NetType: NetVPN}},
		{"::ffff:203.0.113.9", Info{Country: "ID", City: "Jakarta", ASN: 7713, Org: "PT Telekomunikasi Indonesia", NetType: NetResidential}},
	}
	for _, c := range cases {
		if got := db.Lookup(c.ip); got != c.want {
			t.Errorf("Lookup(%s) = %+v, want %+v", c.ip, got, c.want)
		}
	}
}

func TestLookupEmptyCases(t *testing.T) {
	db := New(writeFixtureDB(t))
	defer db.Release()
	for _, ip := range []string{"", "bukan-ip", "10.1.2.3", "127.0.0.1", "192.168.0.5", "::1", "8.8.8.8", "203.0.113.0/24", "203.0.113.x"} {
		if got := db.Lookup(ip); !got.Empty() {
			t.Errorf("Lookup(%q) = %+v, want kosong", ip, got)
		}
	}
}

func TestMissingFilesAreEmptyNotError(t *testing.T) {
	db := New(filepath.Join(t.TempDir(), "tidak-ada"))
	if got := db.Lookup("203.0.113.9"); !got.Empty() {
		t.Errorf("tanpa berkas harus kosong, dapat %+v", got)
	}
	if db.Status().Ready() {
		t.Error("Status tidak boleh Ready tanpa berkas")
	}
	// Berkas rusak diperlakukan sama.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ASNFile), []byte("bukan mmdb"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := New(dir).Lookup("203.0.113.9"); !got.Empty() {
		t.Errorf("berkas rusak harus kosong, dapat %+v", got)
	}
}

func TestReloadPicksUpNewFiles(t *testing.T) {
	src := writeFixtureDB(t)
	dir := t.TempDir()
	db := New(dir)
	defer db.Release()
	if !db.Lookup("203.0.113.9").Empty() {
		t.Fatal("seharusnya kosong sebelum berkas ada")
	}
	for _, name := range []string{CityFile, ASNFile} {
		b, _ := os.ReadFile(filepath.Join(src, name))
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	db.Reload()
	st := db.Status()
	if !st.CityReady || !st.ASNReady || st.CityBuilt != 1767225600 {
		t.Errorf("status setelah Reload salah: %+v", st)
	}
	if got := db.Lookup("203.0.113.9"); got.City != "Jakarta" {
		t.Errorf("setelah Reload: %+v", got)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		asn  uint32
		org  string
		want string
	}{
		{23693, "apa saja", NetMobile},
		{7713, "", NetResidential},
		{13335, "", NetHosting},
		{9009, "", NetVPN},
		{64500, "Acme Cloud Hosting Ltd", NetHosting},
		{64500, "FastVPN Inc", NetVPN},
		{64500, "Indosat Ooredoo Hutchison", NetMobile},
		{64500, "Biznet Networks", NetResidential},
		{64500, "Universitas Antah Berantah", NetUnknown},
		{64500, "State Farm Automobile Insurance", NetUnknown},
		{64500, "T-Mobile USA, Inc.", NetMobile},
		{0, "", NetUnknown},
	}
	for _, c := range cases {
		if got := Classify(c.asn, c.org); got != c.want {
			t.Errorf("Classify(%d,%q) = %s, want %s", c.asn, c.org, got, c.want)
		}
	}
	if ClassifyASN(23693) != NetMobile || ClassifyASN(1) != "" {
		t.Error("ClassifyASN salah")
	}
}

func TestLookupNoticesReplacedFiles(t *testing.T) {
	src := writeFixtureDB(t)
	dir := t.TempDir()
	db := New(dir) // belum ada berkas
	defer db.Release()
	for _, name := range []string{CityFile, ASNFile} {
		b, _ := os.ReadFile(filepath.Join(src, name))
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	db.lastCheck.Store(0) // lewati jeda pemeriksaan
	if got := db.Lookup("203.0.113.9"); got.City != "Jakarta" {
		t.Errorf("berkas baru harus terbaca otomatis: %+v", got)
	}
}
