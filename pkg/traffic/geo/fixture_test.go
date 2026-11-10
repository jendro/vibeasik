package geo

import (
	"bytes"
	"encoding/binary"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
)

// Penulis MMDB minimal untuk uji: pohon IPv4 24-bit, nilai berupa peta string/uint. Berkas dibuat saat uji,
// tidak ada biner yang disimpan di repo.

type mmdbEntry struct {
	prefix string
	value  map[string]any // string, uint32, atau peta bertingkat map[string]any
}

func encCtrl(typ, size int) []byte {
	var ext []byte
	if size >= 29 { // ukuran 29..284 disandikan di satu byte tambahan
		ext, size = []byte{byte(size - 29)}, 29
	}
	if typ < 8 {
		return append([]byte{byte(typ<<5 | size)}, ext...)
	}
	return append([]byte{byte(size), byte(typ - 7)}, ext...) // tipe ekstensi
}

func encValue(v any) []byte {
	switch x := v.(type) {
	case string:
		return append(encCtrl(2, len(x)), x...)
	case uint32:
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, x)
		for len(b) > 0 && b[0] == 0 {
			b = b[1:]
		}
		return append(encCtrl(6, len(b)), b...)
	case uint16:
		b := []byte{byte(x >> 8), byte(x)}
		return append(encCtrl(5, 2), b...)
	case map[string]any:
		out := encCtrl(7, len(x))
		for _, k := range sortedKeys(x) {
			out = append(out, encValue(k)...)
			out = append(out, encValue(x[k])...)
		}
		return out
	}
	panic("tipe tidak didukung")
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := range keys {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// buildMMDB menyusun berkas MMDB IPv4 (record 24 bit) dari daftar prefix.
func buildMMDB(t testing.TB, dbType string, entries []mmdbEntry) []byte {
	t.Helper()
	type node [2]int // >=0: node berikutnya; <0: -(1+indeks data); sentinel kosong = -1<<30
	const empty = -1 << 30
	nodes := []node{{empty, empty}}
	var data bytes.Buffer
	for _, e := range entries {
		p := netip.MustParsePrefix(e.prefix)
		off := data.Len()
		data.Write(encValue(e.value))
		cur := 0
		bits := p.Bits()
		a4 := p.Addr().As4()
		for i := 0; i < bits; i++ {
			bit := int(a4[i/8]>>(7-uint(i%8))) & 1
			if i == bits-1 {
				nodes[cur][bit] = -(1 + off)
				break
			}
			if nodes[cur][bit] <= 0 {
				nodes = append(nodes, node{empty, empty})
				nodes[cur][bit] = len(nodes) - 1
			}
			cur = nodes[cur][bit]
		}
	}
	count := len(nodes)
	rec := func(v int) int {
		switch {
		case v == empty:
			return count
		case v < 0:
			return count + 16 + (-v - 1)
		}
		return v
	}
	var out bytes.Buffer
	for _, n := range nodes {
		l, r := rec(n[0]), rec(n[1])
		out.Write([]byte{byte(l >> 16), byte(l >> 8), byte(l), byte(r >> 16), byte(r >> 8), byte(r)})
	}
	out.Write(make([]byte, 16))
	out.Write(data.Bytes())
	out.WriteString("\xAB\xCD\xEFMaxMind.com")
	out.Write(encValue(map[string]any{
		"node_count": uint32(count), "record_size": uint16(24), "ip_version": uint16(4), "database_type": dbType,
		"binary_format_major_version": uint16(2), "binary_format_minor_version": uint16(0), "build_epoch": uint32(1767225600),
	}))
	return out.Bytes()
}

func cityEntry(prefix, country, city string) mmdbEntry {
	v := map[string]any{"country": map[string]any{"iso_code": country}}
	if city != "" {
		v["city"] = map[string]any{"names": map[string]any{"en": city}}
	}
	return mmdbEntry{prefix, v}
}

func asnEntry(prefix string, asn uint32, org string) mmdbEntry {
	return mmdbEntry{prefix, map[string]any{"autonomous_system_number": asn, "autonomous_system_organization": org}}
}

// writeFixtureDB menulis pasangan basis data kecil ke folder sementara dan mengembalikan foldernya.
func writeFixtureDB(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	city := buildMMDB(t, "DBIP-City-Lite", []mmdbEntry{
		cityEntry("203.0.113.0/24", "ID", "Jakarta"),
		cityEntry("198.51.100.0/24", "ID", "Surabaya"),
		cityEntry("192.0.2.0/24", "US", "Ashburn"),
		cityEntry("100.64.0.0/10", "SG", ""),
	})
	asn := buildMMDB(t, "DBIP-ASN-Lite", []mmdbEntry{
		asnEntry("203.0.113.0/24", 7713, "PT Telekomunikasi Indonesia"),
		asnEntry("198.51.100.0/24", 23693, "PT Telekomunikasi Selular"),
		asnEntry("192.0.2.0/24", 16509, "Amazon.com, Inc."),
		asnEntry("100.64.0.0/10", 64500, "Example VPN Services"),
	})
	for name, b := range map[string][]byte{CityFile: city, ASNFile: asn} {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
