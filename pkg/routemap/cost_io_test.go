package routemap

import (
	"strings"
	"testing"
)

// I/O per item di loop data (HTTP keluar, berkas, email, antrean/pubsub, goroutine per item) dilaporkan seperti
// N+1 query beserta jenisnya; perhitungan di memori dan I/O sekali di luar loop tidak.
const ioFixture = `package main

import (
	"net/http"
	"net/smtp"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var db *gorm.DB
var rdb *redis.Client
var client = &http.Client{}

type Row struct{ ID uint; URL, Path, Email string }

func main() {
	app := fiber.New()
	app.Get("/http", perHTTP)
	app.Get("/file", perFile)
	app.Get("/mail", perMail)
	app.Get("/pub", perPublish)
	app.Get("/go", perGoroutine)
	app.Get("/mem", inMemory)
	app.Get("/once", fileOnce)
}

func load() []Row {
	var rows []Row
	db.Find(&rows)
	return rows
}

func perHTTP(c *fiber.Ctx) error {
	for _, r := range load() {
		req, _ := http.NewRequest("GET", r.URL, nil)
		client.Do(req)
	}
	return nil
}

func exists(p string) bool {
	_, err := os.ReadFile(p)
	return err == nil
}

func perFile(c *fiber.Ctx) error {
	n := 0
	for _, r := range load() {
		if exists(r.Path) {
			n++
		}
	}
	return c.JSON(n)
}

func perMail(c *fiber.Ctx) error {
	for _, r := range load() {
		smtp.SendMail("smtp:25", nil, "a@b", []string{r.Email}, nil)
	}
	return nil
}

func perPublish(c *fiber.Ctx) error {
	for _, r := range load() {
		rdb.Publish(c.Context(), "rows", r.ID)
	}
	return nil
}

func perGoroutine(c *fiber.Ctx) error {
	for _, r := range load() {
		go strings.ToUpper(r.Email)
	}
	return nil
}

func inMemory(c *fiber.Ctx) error {
	rows := load()
	out := []string{}
	for _, a := range rows {
		for _, b := range rows {
			out = append(out, a.Email+b.Email)
		}
	}
	return c.JSON(out)
}

func fileOnce(c *fiber.Ctx) error {
	rows := load()
	data, _ := os.ReadFile("tpl.html")
	for _, r := range rows {
		_ = strings.ReplaceAll(string(data), "{id}", r.Email)
	}
	return nil
}
`

func TestIOPerItemIsReported(t *testing.T) {
	root := writeTree(t, map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": ioFixture})
	graphs := Analyze(root)
	rep := Costs(graphs)
	want := map[string]string{"/http": "http", "/file": "file", "/mail": "mail", "/pub": "queue", "/go": "goroutine"}
	for path, kind := range want {
		got := routeCost(t, rep, path)
		if !got.NPlusOne || !strings.Contains(strings.Join(got.IOKinds, ","), kind) {
			t.Errorf("%s: I/O per item %q harus terdeteksi: n+1=%v kinds=%v Q=%q", path, kind, got.NPlusOne, got.IOKinds, got.QueryFormula)
		}
	}
	for _, path := range []string{"/mem", "/once"} {
		if got := routeCost(t, rep, path); got.NPlusOne {
			t.Errorf("%s: tanpa I/O per item, tidak boleh N+1: kinds=%v Q=%q", path, got.IOKinds, got.QueryFormula)
		}
	}
	hot := map[string]Hotspot{}
	for _, h := range Hotspots(graphs) {
		hot[h.Kind+" "+h.Func.Name] = h
	}
	if h, ok := hot["nplus1 main.perFile"]; !ok || h.Via != "main.exists" {
		t.Errorf("perFile: sumber I/O lewat exists harus dilaporkan: %+v (ada=%v)", h, ok)
	}
	if _, ok := hot["growth main.inMemory"]; ok {
		t.Errorf("inMemory: loop bersarang pemrosesan data wajar, tidak dilaporkan")
	}
}
