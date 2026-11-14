package routemap

import (
	"fmt"
	"testing"
)

const probeSrc = `package main

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

type X struct{ ID uint; New bool; Items []X }

const maxN = 20

func main() {
	app := fiber.New()
	app.Get("/p1", p1); app.Get("/p2", p2); app.Get("/p3", p3); app.Get("/p4", p4); app.Get("/p5", p5)
	app.Get("/p6", p6); app.Get("/p7", p7); app.Get("/p8", p8); app.Get("/p9", p9); app.Get("/p10", p10)
	app.Get("/p11", p11); app.Get("/p12", p12); app.Get("/p13", p13); app.Get("/p14", p14); app.Get("/p15", p15)
}

func load() []X { var xs []X; db.Find(&xs); return xs }

// p1 pencarian dengan return: O(n), Q=1
func p1(c *fiber.Ctx) error { for _, x := range load() { if x.ID == 7 { return c.JSON(x) } }; return nil }
// p2 break tanpa syarat di akhir: loop jalan sekali → Q = 2
func p2(c *fiber.Ctx) error { for _, x := range load() { db.First(&x, x.ID); break }; return nil }
// p3 query hanya saat i == 0: sekali → Q = 2
func p3(c *fiber.Ctx) error { for i, x := range load() { if i == 0 { db.First(&x, x.ID) } }; return nil }
// p4 query bersyarat per item: Q = n + 1 (kasus terburuk)
func p4(c *fiber.Ctx) error { for _, x := range load() { if x.New { db.Create(&x) } }; return nil }
// p5 loop berhenti dengan flag: O(n)
func p5(c *fiber.Ctx) error { xs := load(); found := false; for i := 0; i < len(xs) && !found; i++ { if xs[i].ID == 3 { found = true } }; return c.JSON(found) }
// p6 konstanta 10: Q = 11, bukan N+1
func p6(c *fiber.Ctx) error { for i := 0; i < 10; i++ { var x X; db.First(&x, i) }; return nil }
// p7 antrean BFS: O(n)
func p7(c *fiber.Ctx) error { q := load(); n := 0; for len(q) > 0 { q = q[1:]; n++ }; return c.JSON(n) }
// p8 berhenti setelah batas konstanta: if i >= maxN { break } → Q = 21
func p8(c *fiber.Ctx) error { for i, x := range load() { if i >= maxN { break }; db.First(&x, x.ID) }; return nil }
// p9 continue sebelum query: tetap n + 1
func p9(c *fiber.Ctx) error { for _, x := range load() { if !x.New { continue }; db.Create(&x) }; return nil }
// p10 return query di loop: sekali → Q = 2
func p10(c *fiber.Ctx) error { for _, x := range load() { if x.New { return db.Create(&x).Error } }; return nil }
// p11 switch dengan query di satu case: n + 1
func p11(c *fiber.Ctx) error { for _, x := range load() { switch { case x.New: db.Create(&x) } }; return nil }
// p12 break di dalam switch tidak keluar loop: n + 1
func p12(c *fiber.Ctx) error { for _, x := range load() { switch { case x.New: break }; db.Create(&x) }; return nil }
// p13 break berlabel keluar dari loop luar setelah query pertama: Q = 2
func p13(c *fiber.Ctx) error {
outer:
	for _, x := range load() { for _, y := range x.Items { db.Create(&y); break outer } }
	return nil
}
// p14 if-else: cabang query vs tanpa query → n + 1
func p14(c *fiber.Ctx) error { for _, x := range load() { if x.New { db.Create(&x) } else { _ = x } }; return nil }
// p15 query sebelum loop, loop memproses di memori: O(n), Q = 1
func p15(c *fiber.Ctx) error { xs := load(); s := 0; for _, x := range xs { s += int(x.ID) }; return c.JSON(s) }
`

// TestLoopStopConditionsAndBranches: syarat berhenti (return, break, break berlabel, batas indeks konstanta, flag)
// dan kondisi (if sekali-jalan i == 0, if/else, switch, continue) menentukan berapa kali isi loop berjalan.
func TestLoopStopConditionsAndBranches(t *testing.T) {
	root := writeTree(t, map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": probeSrc})
	rep := Costs(Analyze(root))
	want := map[string]struct{ big, q string }{
		"/p1": {"O(n)", "1"}, "/p2": {"O(1)", "2"}, "/p3": {"O(n)", "2"}, "/p4": {"O(n)", "n + 1"},
		"/p5": {"O(n)", "1"}, "/p6": {"O(1)", "10"}, "/p7": {"O(n)", "1"}, "/p8": {"O(1)", "21"},
		"/p9": {"O(n)", "n + 1"}, "/p10": {"O(n)", "2"}, "/p11": {"O(n)", "n + 1"}, "/p12": {"O(n)", "n + 1"},
		"/p13": {"O(1)", "2"}, "/p14": {"O(n)", "n + 1"}, "/p15": {"O(n)", "1"},
	}
	for i := 1; i <= len(want); i++ {
		path := fmt.Sprintf("/p%d", i)
		r := routeCost(t, rep, path)
		if w := want[path]; r.BigO != w.big || r.QueryFormula != w.q {
			t.Errorf("%s: %s Q=%s, mau %s Q=%s", path, r.BigO, r.QueryFormula, w.big, w.q)
		}
	}
}
