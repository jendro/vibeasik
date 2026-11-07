package models

import "encoding/json"

// JSONValue mengubah nilai kolom ber-serializer JSON (map/slice) menjadi teks JSON untuk Updates(map[string]interface{}).
// gorm tidak menerapkan serializer pada update berbentuk map: map/slice mentah membuat SELURUH update gagal diam-diam
// (kolom lain di map yang sama ikut tidak tersimpan).
func JSONValue(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}
