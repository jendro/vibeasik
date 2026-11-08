package paging

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// Batas paginasi opsional untuk endpoint koleksi yang bentuk responsnya (array/objek) tidak boleh berubah.
const (
	// ListDefaultLimit: jumlah baris bila klien tidak mengirim `limit` — longgar agar klien lama tetap menerima
	// koleksi normal utuh, tetapi tetap membatasi tabel yang tumbuh liar.
	ListDefaultLimit = 1000
	// ListMaxLimit: batas atas `limit` yang diminta klien.
	ListMaxLimit = 1000
	// HeaderTotal membawa jumlah seluruh baris sebelum dipotong.
	HeaderTotal = "X-Total-Count"
)

// Window menghitung irisan [start,end) dari total baris. limit<=0 → def; limit>max → max; offset<0 → 0.
func Window(total, limit, offset, def, max int) (start, end int) {
	if def <= 0 {
		def = ListDefaultLimit
	}
	if max <= 0 {
		max = ListMaxLimit
	}
	if limit <= 0 {
		limit = def
	}
	if limit > max {
		limit = max
	}
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end = offset + limit
	if end > total {
		end = total
	}
	return offset, end
}

// Query membaca `limit` & `offset` opsional dari query string (nilai tak valid dianggap kosong).
func Query(c *fiber.Ctx) (limit, offset int) {
	limit, _ = strconv.Atoi(c.Query("limit"))
	offset, _ = strconv.Atoi(c.Query("offset"))
	return limit, offset
}

// SetTotal mengirim total baris lewat header X-Total-Count.
func SetTotal(c *fiber.Ctx, total int) {
	c.Set(HeaderTotal, strconv.Itoa(total))
	c.Append(fiber.HeaderAccessControlExposeHeaders, HeaderTotal)
}

// Apply memotong items sesuai ?limit=&offset= (default/maks ListDefaultLimit/ListMaxLimit) dan mengisi
// X-Total-Count. Hasil selalu slice non-nil sehingga JSON tetap `[]`, bukan `null`.
func Apply[T any](c *fiber.Ctx, items []T) []T {
	limit, offset := Query(c)
	SetTotal(c, len(items))
	s, e := Window(len(items), limit, offset, ListDefaultLimit, ListMaxLimit)
	if s == 0 && e == len(items) && items != nil {
		return items
	}
	out := make([]T, e-s)
	copy(out, items[s:e])
	return out
}
