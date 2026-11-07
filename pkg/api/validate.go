package api

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
//	if p := firstProblem(bindJSONRequired(c, &in), needText("name", in.Name, maxName)); p != "" {
//		return badPayload(c, p)
//	}
//
// Tiap helper mengembalikan pesan masalah ("" = lolos). Respons selalu 400 {"error": pesan}.
// Aturan bisnis tetap di service; di sini hanya bentuk: JSON valid, wajib, panjang, enum, id.
const (
	maxName  = 200   // nama, judul, slug, kunci pendek
	maxLabel = 500   // judul panjang, deskripsi singkat
	maxText  = 20000 // isi bebas (catatan, konten)
	maxPath  = 4096  // path berkas/direktori
	maxURL   = 2048
)

func badPayload(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": msg})
}

// bindJSON mem-parsing body ke dst. Body kosong dianggap lolos (semua field opsional);
// body yang tidak bisa dibaca sebagai JSON bertipe benar ditolak.
func bindJSON(c *fiber.Ctx, dst any) string {
	if len(c.Body()) == 0 {
		return ""
	}
	if err := c.BodyParser(dst); err != nil {
		return "body JSON tidak valid: " + err.Error()
	}
	return ""
}

// bindJSONRequired seperti bindJSON, tetapi body kosong juga ditolak.
func bindJSONRequired(c *fiber.Ctx, dst any) string {
	if len(c.Body()) == 0 {
		return "body JSON wajib diisi"
	}
	return bindJSON(c, dst)
}

// firstProblem mengembalikan pesan masalah pertama yang tidak kosong.
func firstProblem(msgs ...string) string {
	for _, m := range msgs {
		if m != "" {
			return m
		}
	}
	return ""
}

// needText: wajib (bukan kosong/spasi) dan maksimal max karakter.
func needText(field, v string, max int) string {
	if strings.TrimSpace(v) == "" {
		return fmt.Sprintf("field %s wajib diisi", field)
	}
	return maxLen(field, v, max)
}

// maxLen: batas panjang (karakter) untuk field opsional.
func maxLen(field, v string, max int) string {
	if utf8.RuneCountInString(v) > max {
		return fmt.Sprintf("field %s maksimal %d karakter", field, max)
	}
	return ""
}

// oneOf: nilai kosong lolos (opsional); selain itu harus salah satu dari allowed.
func oneOf(field, v string, allowed ...string) string {
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

// needOneOf: wajib dan salah satu dari allowed.
func needOneOf(field, v string, allowed ...string) string {
	if strings.TrimSpace(v) == "" {
		return fmt.Sprintf("field %s wajib diisi", field)
	}
	return oneOf(field, v, allowed...)
}

// needID: id numerik wajib > 0.
func needID(field string, v uint) string {
	if v == 0 {
		return fmt.Sprintf("field %s wajib diisi (id > 0)", field)
	}
	return ""
}

// absPath: bila diisi harus berupa path absolut tanpa karakter NUL, maksimal maxPath.
func absPath(field, v string) string {
	if v == "" {
		return ""
	}
	if strings.ContainsRune(v, 0) || !strings.HasPrefix(v, "/") && !(len(v) > 2 && v[1] == ':') {
		return fmt.Sprintf("field %s harus path absolut yang valid", field)
	}
	return maxLen(field, v, maxPath)
}

// paramID membaca parameter rute integer > 0; label dipakai di pesan ("project", "todo", ...).
func paramID(c *fiber.Ctx, name, label string) (uint, string) {
	id, err := c.ParamsInt(name)
	if err != nil || id <= 0 {
		return 0, "invalid " + label + " id"
	}
	return uint(id), ""
}
