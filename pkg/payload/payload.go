// Package payload: validasi bentuk payload HTTP bersama (todo #3279) untuk server inti, modul Pro, dan cloud.
// Satu-satunya tempat aturan bentuk input; aturan bisnis tetap di service.
package payload

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
)

// Validasi payload bersama (todo #3279) — satu-satunya tempat aturan bentuk input HTTP.
// Pola di handler:
//
//	var in struct{ Name string `json:"name"` }
//	if p := FirstProblem(BindJSONRequired(c, &in), NeedText("name", in.Name, MaxName)); p != "" {
//		return Bad(c, p)
//	}
//
// Tiap helper mengembalikan pesan masalah ("" = lolos). Respons selalu 400 {"error": pesan}.
// Aturan bisnis tetap di service; di sini hanya bentuk: JSON valid, wajib, panjang, enum, id.
const (
	MaxName  = 200   // nama, judul, slug, kunci pendek
	MaxLabel = 500   // judul panjang, deskripsi singkat
	MaxText  = 20000 // isi bebas (catatan, konten)
	MaxPath  = 4096  // path berkas/direktori
	MaxURL   = 2048
)

func Bad(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": msg})
}

// BindJSON mem-parsing body ke dst. Body kosong dianggap lolos (semua field opsional);
// body yang tidak bisa dibaca sebagai JSON bertipe benar ditolak.
func BindJSON(c *fiber.Ctx, dst any) string {
	if len(c.Body()) == 0 {
		return ""
	}
	if err := c.BodyParser(dst); err != nil {
		return "body JSON tidak valid: " + err.Error()
	}
	return ""
}

// BindJSONRequired seperti BindJSON, tetapi body kosong juga ditolak.
func BindJSONRequired(c *fiber.Ctx, dst any) string {
	if len(c.Body()) == 0 {
		return "body JSON wajib diisi"
	}
	return BindJSON(c, dst)
}

// FirstProblem mengembalikan pesan masalah pertama yang tidak kosong.
func FirstProblem(msgs ...string) string {
	for _, m := range msgs {
		if m != "" {
			return m
		}
	}
	return ""
}

// NeedText: wajib (bukan kosong/spasi) dan maksimal max karakter.
func NeedText(field, v string, max int) string {
	if strings.TrimSpace(v) == "" {
		return fmt.Sprintf("field %s wajib diisi", field)
	}
	return MaxLen(field, v, max)
}

// MaxLen: batas panjang (karakter) untuk field opsional.
func MaxLen(field, v string, max int) string {
	if utf8.RuneCountInString(v) > max {
		return fmt.Sprintf("field %s maksimal %d karakter", field, max)
	}
	return ""
}

// OneOf: nilai kosong lolos (opsional); selain itu harus salah satu dari allowed.
func OneOf(field, v string, allowed ...string) string {
	if v == "" {
		return ""
	}
	for _, a := range allowed {
		if v == a {
			return ""
		}
	}
	return fmt.Sprintf("field %s harus salah satu dari: %s", field, strings.Join(allowed, ", "))
}

// NeedOneOf: wajib dan salah satu dari allowed.
func NeedOneOf(field, v string, allowed ...string) string {
	if strings.TrimSpace(v) == "" {
		return fmt.Sprintf("field %s wajib diisi", field)
	}
	return OneOf(field, v, allowed...)
}

// NeedID: id numerik wajib > 0.
func NeedID(field string, v uint) string {
	if v == 0 {
		return fmt.Sprintf("field %s wajib diisi (id > 0)", field)
	}
	return ""
}

// AbsPath: bila diisi harus berupa path absolut tanpa karakter NUL, maksimal MaxPath.
func AbsPath(field, v string) string {
	if v == "" {
		return ""
	}
	if strings.ContainsRune(v, 0) || !strings.HasPrefix(v, "/") && !(len(v) > 2 && v[1] == ':') {
		return fmt.Sprintf("field %s harus path absolut yang valid", field)
	}
	return MaxLen(field, v, MaxPath)
}

// ParamID membaca parameter rute integer > 0; label dipakai di pesan ("project", "todo", ...).
func ParamID(c *fiber.Ctx, name, label string) (uint, string) {
	id, err := c.ParamsInt(name)
	if err != nil || id <= 0 {
		return 0, "invalid " + label + " id"
	}
	return uint(id), ""
}
