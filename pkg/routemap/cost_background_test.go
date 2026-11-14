package routemap

import "testing"

// Fungsi di luar rute (worker antrean, goroutine dari main, callback yang diteruskan sebagai nilai) ikut menjadi
// sumber biaya; kode mati dan skrip sekali jalan tidak.
const backgroundFixture = `package main

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

type Job struct{ ID uint }

type handlerFn func(id uint) error

var registry = map[string]handlerFn{}

func register(name string, fn handlerFn) { registry[name] = fn }

func main() {
	app := fiber.New()
	app.Get("/ok", ok)
	register("email", sendEmails)
	go nightly()
}

func ok(c *fiber.Ctx) error { return c.SendString("ok") }

// sendEmails: worker antrean yang didaftarkan sebagai nilai — N+1.
func sendEmails(id uint) error {
	var jobs []Job
	db.Find(&jobs)
	for _, j := range jobs {
		db.First(&j, j.ID)
	}
	return nil
}

// nightly: goroutine dari main — N+1.
func nightly() {
	var jobs []Job
	db.Find(&jobs)
	for _, j := range jobs {
		db.Delete(&j)
	}
}

// unused: kode mati — tidak dilaporkan.
func unused() {
	var jobs []Job
	db.Find(&jobs)
	for _, j := range jobs {
		db.Delete(&j)
	}
}
`

const benchFixture = `package main

import "gorm.io/gorm"

var db *gorm.DB

type Row struct{ ID uint }

func main() {
	var rows []Row
	db.Find(&rows)
	for _, r := range rows {
		db.First(&r, r.ID)
	}
}
`

func TestBackgroundFunctionsAreCosted(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod": "module demo\n\ngo 1.22\n", "main.go": backgroundFixture,
		"cmd/bench_rows/main.go": benchFixture,
	})
	got := map[string]Hotspot{}
	for _, h := range Hotspots(Analyze(root)) {
		if h.Kind == "nplus1" {
			got[h.Func.Name] = h
		}
	}
	for _, name := range []string{"main.sendEmails", "main.nightly"} {
		h, ok := got[name]
		if !ok || !h.Background || h.RouteCount != 0 {
			t.Errorf("%s: fungsi di luar rute harus dilaporkan sebagai background: %+v (ada=%v)", name, h, ok)
		}
	}
	if _, ok := got["main.unused"]; ok {
		t.Error("kode mati tidak boleh dilaporkan")
	}
	for name, h := range got {
		if h.Func.File == "cmd/bench_rows/main.go" {
			t.Errorf("skrip benchmark tidak boleh dilaporkan: %s", name)
		}
	}
}

// Loop peristiwa (range channel, for { select }) dihitung per pesan, bukan loop atas data; main alat bantu di cmd/
// yang tidak membuat router bukan titik masuk aplikasi.
const eventLoopFixture = `package main

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

type Task struct{ ID uint }

type Manager struct {
	tasks chan *Task
	quit  chan struct{}
}

var mgr = &Manager{tasks: make(chan *Task, 10), quit: make(chan struct{})}

func main() {
	app := fiber.New()
	app.Get("/ok", func(c *fiber.Ctx) error { return nil })
	go mgr.consume()
	go mgr.poll()
}

func (m *Manager) consume() {
	for t := range m.tasks {
		db.First(t, t.ID)
	}
}

func (m *Manager) poll() {
	for {
		select {
		case t := <-m.tasks:
			db.First(t, t.ID)
		case <-m.quit:
			return
		}
	}
}
`

const toolMainFixture = `package main

import "gorm.io/gorm"

var db *gorm.DB

type Row struct{ ID uint }

func main() {
	var rows []Row
	db.Find(&rows)
	for _, r := range rows {
		db.Delete(&r)
	}
}
`

func TestEventLoopsAndToolMains(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod": "module demo\n\ngo 1.22\n", "main.go": eventLoopFixture,
		"cmd/compare-data/main.go": toolMainFixture,
	})
	for _, h := range Hotspots(Analyze(root)) {
		t.Errorf("tidak boleh ada sumber biaya (loop peristiwa / alat bantu): %s %s Q=%s", h.Kind, h.Func.Name, h.Func.QueryFormula)
	}
}
