package routemap

import "testing"

// Loop atas koleksi berukuran tetap (daftar literal tingkat paket, potongan teks) dulu dihitung sebagai loop atas
// data, sehingga fungsi seperti AssetURL atau pencarian template membuat pemanggilnya tampak O(n²)/O(n³).
const fixedRangeFixture = `package main

import (
	"os"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

type Row struct{ ID uint; URL string }

var templates = []string{"a", "b", "c"}

var tableMeta = map[string]string{"users": "u", "webinars": "w"}

var ruleRe = regexp.MustCompile("([^{]+){([^}]*)}")

func main() {
	app := fiber.New()
	app.Get("/rows", listRows)
	app.Get("/tables", tables)
	app.Get("/css", css)
}

func assetURL(v string) string {
	for _, h := range strings.Split(os.Getenv("MIRRORS"), ",") {
		if strings.Contains(v, h) {
			return h
		}
	}
	for _, t := range templates {
		if t == v {
			return t
		}
	}
	return v
}

func listRows(c *fiber.Ctx) error {
	var rows []Row
	db.Find(&rows)
	out := []string{}
	for _, r := range rows {
		out = append(out, assetURL(r.URL))
	}
	return c.JSON(out)
}

func tables(c *fiber.Ctx) error {
	total := int64(0)
	for name := range tableMeta {
		var n int64
		db.Table(name).Count(&n)
		total += n
	}
	return c.JSON(total)
}

func css(c *fiber.Ctx) error {
	props := map[string]string{}
	for _, m := range ruleRe.FindAllStringSubmatch(string(c.Body()), -1) {
		for _, d := range strings.Split(m[2], ";") {
			k, v, _ := strings.Cut(d, ":")
			props[k] = v
		}
	}
	return c.JSON(props)
}
`

func TestFixedCollectionsAreNotDataLoops(t *testing.T) {
	root := writeTree(t, map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": fixedRangeFixture})
	rep := Costs(Analyze(root))
	if got := routeCost(t, rep, "/rows"); got.BigO != "O(n)" {
		t.Errorf("/rows: loop atas baris yang memanggil helper teks/daftar tetap harus O(n): %s (%s)", got.BigO, got.Formula)
	}
	if got := routeCost(t, rep, "/tables"); got.NPlusOne {
		t.Errorf("/tables: COUNT per tabel dari peta literal 2 entri bukan N+1: Q(n)=%q", got.QueryFormula)
	}
	if got := routeCost(t, rep, "/css"); got.BigO == "O(n²)" {
		t.Errorf("/css: aturan × deklarasi satu teks CSS bukan O(n²): %s", got.Formula)
	}
}
