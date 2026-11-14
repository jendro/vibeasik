package routemap

import "testing"

// Loop bersarang dilaporkan sebagai sumber pertumbuhan hanya bila loop dalamnya pencarian/pencocokan yang bisa
// diganti map; pemrosesan data bersarang (baris × kolom, menyusun keluaran) dianggap wajar. Query di dalam loop
// tetap N+1.
const avoidableFixture = `package main

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

type User struct{ ID uint; Name string }
type Order struct{ ID, UserID uint; Tags []string }

func main() {
	app := fiber.New()
	app.Get("/join", join)
	app.Get("/csv", csv)
	app.Get("/find", findVia)
}

// join: untuk tiap order, cari user-nya dengan loop → bisa diganti map[UserID]User.
func join(c *fiber.Ctx) error {
	var orders []Order
	var users []User
	db.Find(&orders)
	db.Find(&users)
	out := []string{}
	for _, o := range orders {
		for _, u := range users {
			if u.ID == o.UserID {
				out = append(out, u.Name)
				break
			}
		}
	}
	return c.JSON(out)
}

// csv: tiap order × tiap tag-nya ditulis — pemrosesan yang memang harus menyentuh semua sel.
func csv(c *fiber.Ctx) error {
	var orders []Order
	db.Find(&orders)
	var b strings.Builder
	for _, o := range orders {
		for _, t := range o.Tags {
			if t == "" {
				continue
			}
			b.WriteString(t)
		}
	}
	return c.SendString(b.String())
}

func userOf(users []User, id uint) User {
	for _, u := range users {
		if u.ID == id {
			return u
		}
	}
	return User{}
}

// findVia: pencarian linear lewat fungsi lain di dalam loop data → bisa dihindari.
func findVia(c *fiber.Ctx) error {
	var orders []Order
	var users []User
	db.Find(&orders)
	db.Find(&users)
	names := []string{}
	for _, o := range orders {
		names = append(names, userOf(users, o.UserID).Name)
	}
	return c.JSON(names)
}
`

func TestGrowthOnlyWhenAvoidable(t *testing.T) {
	root := writeTree(t, map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": avoidableFixture})
	graphs := Analyze(root)
	growth := map[string]Hotspot{}
	for _, h := range Hotspots(graphs) {
		if h.Kind == "growth" {
			growth[h.Func.Name] = h
		}
	}
	if _, ok := growth["main.join"]; !ok {
		t.Errorf("join (cari user dengan loop di dalam loop order) harus dilaporkan: %+v", growth)
	}
	if _, ok := growth["main.findVia"]; !ok {
		t.Errorf("findVia (pencarian linear lewat userOf di dalam loop) harus dilaporkan: %+v", growth)
	}
	if h, ok := growth["main.csv"]; ok {
		t.Errorf("csv (order × tag, pemrosesan wajar) tidak boleh dilaporkan: %+v", h)
	}
	rep := Costs(graphs)
	if got := routeCost(t, rep, "/csv"); got.BigO != "O(n²)" || got.Avoidable {
		t.Errorf("/csv tetap O(n²) di Biaya Proses tetapi tidak ditandai bisa dihindari: %s avoidable=%v", got.BigO, got.Avoidable)
	}
}
