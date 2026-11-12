// Command winres membuat berkas .syso (objek COFF berisi section .rsrc) untuk biner Windows: ikon
// aplikasi (RT_GROUP_ICON ID 1) dan manifest (RT_MANIFEST ID 1). Go linker menautkan .syso di folder
// paket main secara otomatis. Hanya pustaka standar, agar build tidak bergantung alat dari luar.
//
//	go run ./tools/winres -ico app.ico -manifest app.manifest -out rsrc_windows_amd64.syso
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
)

const (
	rtIcon      = 3
	rtGroupIcon = 14
	rtManifest  = 24
	langEnUS    = 0x0409

	machineAMD64    = 0x8664
	relAMD64Addr32N = 3 // IMAGE_REL_AMD64_ADDR32NB
)

type resource struct {
	typ, id uint32
	data    []byte
}

func main() {
	ico := flag.String("ico", "", "berkas .ico (boleh berisi banyak ukuran)")
	manifest := flag.String("manifest", "", "berkas manifest XML")
	out := flag.String("out", "rsrc_windows_amd64.syso", "berkas .syso keluaran")
	flag.Parse()
	var res []resource
	if *ico != "" {
		b, err := os.ReadFile(*ico)
		if err != nil {
			fail(err)
		}
		r, err := iconResources(b)
		if err != nil {
			fail(err)
		}
		res = append(res, r...)
	}
	if *manifest != "" {
		b, err := os.ReadFile(*manifest)
		if err != nil {
			fail(err)
		}
		res = append(res, resource{rtManifest, 1, b})
	}
	if len(res) == 0 {
		fail(errors.New("tidak ada resource: isi -ico dan/atau -manifest"))
	}
	if err := os.WriteFile(*out, coff(res), 0o644); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "winres:", err)
	os.Exit(1)
}

// iconResources memecah .ico menjadi RT_ICON per ukuran (ID 1..n) dan satu RT_GROUP_ICON ID 1.
func iconResources(b []byte) ([]resource, error) {
	if len(b) < 6 || binary.LittleEndian.Uint16(b[2:]) != 1 {
		return nil, errors.New("bukan berkas .ico")
	}
	n := int(binary.LittleEndian.Uint16(b[4:]))
	var res []resource
	group := new(bytes.Buffer)
	binary.Write(group, binary.LittleEndian, [3]uint16{0, 1, uint16(n)})
	for i := 0; i < n; i++ {
		e := b[6+16*i : 6+16*i+16]
		size := binary.LittleEndian.Uint32(e[8:])
		off := binary.LittleEndian.Uint32(e[12:])
		if int(off+size) > len(b) {
			return nil, errors.New(".ico rusak")
		}
		id := uint32(i + 1)
		res = append(res, resource{rtIcon, id, b[off : off+size]})
		group.Write(e[:12])                                  // lebar, tinggi, warna, cadangan, planes, bitcount, ukuran
		binary.Write(group, binary.LittleEndian, uint16(id)) // GRPICONDIRENTRY: ID resource, bukan offset
	}
	return append(res, resource{rtGroupIcon, 1, group.Bytes()}), nil
}

// coff menyusun objek COFF dengan satu section .rsrc: pohon direktori tipe → ID → bahasa → data.
func coff(res []resource) []byte {
	types := map[uint32][]resource{}
	for _, r := range res {
		types[r.typ] = append(types[r.typ], r)
	}
	var typeIDs []uint32
	for t := range types {
		typeIDs = append(typeIDs, t)
		sort.Slice(types[t], func(i, j int) bool { return types[t][i].id < types[t][j].id })
	}
	sort.Slice(typeIDs, func(i, j int) bool { return typeIDs[i] < typeIDs[j] })

	dirSize := func(entries int) int { return 16 + 8*entries }
	// Tata letak: direktori akar, direktori per tipe, direktori bahasa per resource, entri data, lalu isi.
	off := dirSize(len(typeIDs))
	typeDirOff := map[uint32]int{}
	for _, t := range typeIDs {
		typeDirOff[t] = off
		off += dirSize(len(types[t]))
	}
	langDirOff := map[[2]uint32]int{}
	for _, t := range typeIDs {
		for _, r := range types[t] {
			langDirOff[[2]uint32{t, r.id}] = off
			off += dirSize(1)
		}
	}
	dataEntryOff := map[[2]uint32]int{}
	for _, t := range typeIDs {
		for _, r := range types[t] {
			dataEntryOff[[2]uint32{t, r.id}] = off
			off += 16
		}
	}
	dataOff := map[[2]uint32]int{}
	for _, t := range typeIDs {
		for _, r := range types[t] {
			off = (off + 7) &^ 7
			dataOff[[2]uint32{t, r.id}] = off
			off += len(r.data)
		}
	}
	sec := make([]byte, (off+7)&^7)
	le := binary.LittleEndian
	dir := func(at, entries int) { le.PutUint16(sec[at+14:], uint16(entries)) } // hanya entri ID
	entry := func(at int, id uint32, target int, isDir bool) {
		le.PutUint32(sec[at:], id)
		v := uint32(target)
		if isDir {
			v |= 0x80000000
		}
		le.PutUint32(sec[at+4:], v)
	}
	dir(0, len(typeIDs))
	var relocs []uint32
	for i, t := range typeIDs {
		entry(16+8*i, t, typeDirOff[t], true)
		dir(typeDirOff[t], len(types[t]))
		for j, r := range types[t] {
			k := [2]uint32{t, r.id}
			entry(typeDirOff[t]+16+8*j, r.id, langDirOff[k], true)
			dir(langDirOff[k], 1)
			entry(langDirOff[k]+16, langEnUS, dataEntryOff[k], false)
			de := dataEntryOff[k]
			le.PutUint32(sec[de:], uint32(dataOff[k])) // RVA, dibetulkan linker lewat relokasi
			le.PutUint32(sec[de+4:], uint32(len(r.data)))
			relocs = append(relocs, uint32(de))
			copy(sec[dataOff[k]:], r.data)
		}
	}

	const fileHdr, secHdr, relSize, symSize = 20, 40, 10, 18
	rawPtr := fileHdr + secHdr
	relPtr := rawPtr + len(sec)
	symPtr := relPtr + relSize*len(relocs)
	out := new(bytes.Buffer)
	w := func(v any) { binary.Write(out, le, v) }
	w(uint16(machineAMD64))
	w(uint16(1)) // jumlah section
	w(uint32(0)) // timestamp: 0 agar keluaran deterministik
	w(uint32(symPtr))
	w(uint32(1)) // satu simbol: .rsrc
	w(uint16(0))
	w(uint16(0))
	out.WriteString(".rsrc\x00\x00\x00")
	w(uint32(0))
	w(uint32(0))
	w(uint32(len(sec)))
	w(uint32(rawPtr))
	w(uint32(relPtr))
	w(uint32(0))
	w(uint16(len(relocs)))
	w(uint16(0))
	w(uint32(0x40000040)) // INITIALIZED_DATA | MEM_READ
	out.Write(sec)
	for _, r := range relocs {
		w(r)
		w(uint32(0)) // indeks simbol .rsrc
		w(uint16(relAMD64Addr32N))
	}
	out.WriteString(".rsrc\x00\x00\x00")
	w(uint32(0)) // nilai
	w(int16(1))  // nomor section
	w(uint16(0))
	w(uint8(3)) // IMAGE_SYM_CLASS_STATIC
	w(uint8(0))
	w(uint32(4)) // tabel string kosong
	return out.Bytes()
}
