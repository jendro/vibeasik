package routemap

import "testing"

const costFixture = `package main

import (
	"sort"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Order struct{ ID uint }
type User struct {
	ID     uint
	Orders []Order
}

var db *gorm.DB

func main() {
	app := fiber.New()
	app.Get("/one", one)
	app.Get("/list", list)
	app.Get("/pairs", pairs)
	app.Get("/nplus", nplus)
	app.Get("/sorted", sorted)
	app.Get("/fixed", fixed)
	app.Get("/tree", tree)
	app.Get("/one", one)
}

func one(c *fiber.Ctx) error {
	var u User
	db.First(&u, 1)
	return c.JSON(u)
}

func list(c *fiber.Ctx) error {
	var us []User
	db.Find(&us)
	total := 0
	for _, u := range us {
		total += int(u.ID)
	}
	return c.JSON(total)
}

func pairs(c *fiber.Ctx) error {
	var us []User
	db.Find(&us)
	hits := 0
	for i := range us {
		for j := range us {
			if us[i].ID == us[j].ID {
				hits++
			}
		}
	}
	return c.JSON(hits)
}

func nplus(c *fiber.Ctx) error {
	var us []User
	db.Find(&us)
	for i := range us {
		db.Where("user_id = ?", us[i].ID).Find(&us[i].Orders)
	}
	return c.JSON(us)
}

func sorted(c *fiber.Ctx) error {
	var us []User
	db.Find(&us)
	sort.Slice(us, func(i, j int) bool { return us[i].ID < us[j].ID })
	return c.JSON(us)
}

func fixed(c *fiber.Ctx) error {
	sum := 0
	for i := 0; i < 3; i++ {
		sum += helper(i)
	}
	return c.JSON(sum)
}

func helper(i int) int { return i * 2 }

func tree(c *fiber.Ctx) error {
	return c.JSON(walk(3))
}

func walk(d int) int {
	if d == 0 {
		return 1
	}
	return walk(d-1) + 1
}
`

func routeCost(t *testing.T, rep *CostReport, path string) RouteCost {
	t.Helper()
	for _, r := range rep.Routes {
		if r.Path == path {
			return r
		}
	}
	t.Fatalf("biaya rute %s tidak ditemukan", path)
	return RouteCost{}
}

func TestRouteCostFormulas(t *testing.T) {
	root := writeTree(t, map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": costFixture})
	rep := Costs(Analyze(root))

	cases := []struct {
		path, formula, bigO, queries string
		ticks1                       int64
		nPlusOne                     bool
	}{
		{"/one", "3", "O(1)", "1", 3, false},
		{"/list", "2n + 5", "O(n)", "1", 7, false},
		{"/pairs", "3n² + 2n + 5", "O(n²)", "1", 10, false},
		{"/nplus", "2n + 4", "O(n)", "n + 1", 6, true},
		{"/sorted", "2n log n + 4", "O(n log n)", "1", 6, false},
		{"/fixed", "16", "O(1)", "0", 16, false},
	}
	for _, tc := range cases {
		got := routeCost(t, rep, tc.path)
		if got.Formula != tc.formula || got.BigO != tc.bigO || got.QueryFormula != tc.queries || got.Ticks1 != tc.ticks1 || got.NPlusOne != tc.nPlusOne {
			t.Errorf("%s: T(n)=%q %s Q(n)=%q T(1)=%d n+1=%v; mau T(n)=%q %s Q(n)=%q T(1)=%d n+1=%v",
				tc.path, got.Formula, got.BigO, got.QueryFormula, got.Ticks1, got.NPlusOne,
				tc.formula, tc.bigO, tc.queries, tc.ticks1, tc.nPlusOne)
		}
	}
	if !routeCost(t, rep, "/tree").Recursive {
		t.Error("/tree: rekursi tidak ditandai")
	}
	if len(rep.Routes) != 7 {
		t.Errorf("rute ganda harus tampil sekali: %d rute, mau 7", len(rep.Routes))
	}
	// Urutan: yang paling mahal saat data tumbuh berada di atas.
	if rep.Routes[0].Path != "/pairs" {
		t.Errorf("rute termahal = %s, mau /pairs", rep.Routes[0].Path)
	}
	// Pertumbuhan /pairs pada n = 100: 3·100² + 2·100 + 5.
	if g := routeCost(t, rep, "/pairs").Growth[2]; g.N != 100 || g.Ticks != 30205 {
		t.Errorf("/pairs T(100) = %v, mau 30205", g.Ticks)
	}
}

func TestLoopBounds(t *testing.T) {
	root := writeTree(t, map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": `package main

import (
	"sync"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()
	app.Get("/once", func(c *fiber.Ctx) error {
		var o sync.Once
		items := []int{}
		o.Do(func() {
			for range items {
			}
		})
		return nil
	})
	app.Get("/spin", func(c *fiber.Ctx) error {
		for {
			if ready() {
				break
			}
		}
		return nil
	})
}

func ready() bool { return true }

`})
	// Callback Do (sync.Once, singleflight) berjalan sekali: satu loop di dalamnya tetap O(n), bukan O(n²).
	if once := routeCost(t, Costs(Analyze(root)), "/once"); once.BigO != "O(n)" {
		t.Errorf("/once: %s (%s), mau O(n)", once.BigO, once.Formula)
	}
	got := routeCost(t, Costs(Analyze(root)), "/spin")
	if !got.Unbounded || got.BigO != "O(n)" {
		t.Errorf("/spin: unbounded=%v %s, mau unbounded O(n)", got.Unbounded, got.BigO)
	}
}

func TestHotspotsPointAtSourceFunction(t *testing.T) {
	root := writeTree(t, map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": `package main

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

type Item struct{ ID uint }

func main() {
	app := fiber.New()
	app.Get("/a", listA)
	app.Get("/b", listB)
	app.Get("/c", plain)
}

func listA(c *fiber.Ctx) error { return c.JSON(enrich()) }
func listB(c *fiber.Ctx) error { return c.JSON(enrich()) }
func plain(c *fiber.Ctx) error { return c.JSON(loadOne(1)) }

// enrich: loop atas data yang memanggil fungsi ber-query → sumber N+1 walau query-nya di fungsi lain.
func enrich() []Item {
	var items []Item
	db.Find(&items)
	for i := range items {
		items[i] = loadOne(items[i].ID)
		for j := range items {
			_ = j
		}
	}
	return items
}

func loadOne(id uint) Item {
	var it Item
	db.First(&it, id)
	return it
}
`})
	hs := Hotspots(Analyze(root))
	kinds := map[string]Hotspot{}
	for _, h := range hs {
		if h.Func.Name != "main.enrich" {
			t.Errorf("sumber harus main.enrich, bukan %s (%s)", h.Func.Name, h.Kind)
		}
		kinds[h.Kind] = h
	}
	n1, ok := kinds["nplus1"]
	if !ok || n1.Via != "main.loadOne" || n1.RouteCount != 2 || len(n1.Routes) != 2 || n1.Routes[0] != "GET /a" {
		t.Errorf("hotspot N+1: %+v", n1)
	}
	if g, ok := kinds["growth"]; !ok || g.Func.BigO != "O(n²)" || g.Func.End <= g.Func.Line {
		t.Errorf("hotspot pertumbuhan: %+v", g)
	}
	if len(hs) != 2 {
		t.Errorf("handler pemanggil tidak boleh ikut dilaporkan: %d hotspot", len(hs))
	}
}
