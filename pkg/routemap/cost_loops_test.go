package routemap

import "testing"

// Bentuk loop dari backend nyata yang dulu dilaporkan N+1 padahal jumlah query-nya tidak tumbuh per baris
// (loop per chunk, keyset, hapus per batch, Limit kecil, return di dalam loop), bersama dua yang memang N+1.
const loopShapesFixture = `package main

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

type Job struct{ ID, ParticipantID uint }
type P struct{ ID, UserID uint }
type User struct {
	ID       uint
	Password string
}

const chunk = 200
const pageSize = 500

func main() {
	app := fiber.New()
	app.Post("/retry", retry)
	app.Get("/export", export)
	app.Post("/dlq", clearDLQ)
	app.Post("/login", login)
	app.Get("/notif", notif)
	app.Post("/move", move)
	app.Delete("/participants", removeMany)
}

func retry(c *fiber.Ctx) error {
	var jobs []Job
	db.Find(&jobs)
	for start := 0; start < len(jobs); start += chunk {
		end := min(start+chunk, len(jobs))
		ids := []uint{}
		for _, j := range jobs[start:end] {
			ids = append(ids, j.ParticipantID)
		}
		var ps []P
		db.Where("id IN ?", ids).Find(&ps)
	}
	return nil
}

func fetchBatch(ctx context.Context, after uint) []P {
	var ps []P
	db.WithContext(ctx).Where("id > ?", after).Limit(pageSize).Find(&ps)
	return ps
}

func export(c *fiber.Ctx) error {
	ctx := c.Context()
	var last uint
	for ctx.Err() == nil {
		batch := fetchBatch(ctx, last)
		if len(batch) == 0 {
			break
		}
		last = batch[len(batch)-1].ID
		if len(batch) < pageSize {
			break
		}
	}
	return nil
}

func deleteBatch() int64 {
	return db.Exec("DELETE FROM dlq LIMIT 500").RowsAffected
}

func clearDLQ(c *fiber.Ctx) error {
	for {
		n := deleteBatch()
		if n == 0 {
			break
		}
	}
	return nil
}

func respond(c *fiber.Ctx, u User) error {
	db.Model(&u).Update("last_login", 1)
	return c.JSON(u)
}

func login(c *fiber.Ctx) error {
	var users []User
	db.Where("email = ?", c.FormValue("email")).Limit(5).Find(&users)
	for _, u := range users {
		if u.Password == c.FormValue("password") {
			return respond(c, u)
		}
	}
	return nil
}

func syncOne(p P) {
	db.Create(&p)
	db.Create(&p)
}

func notif(c *fiber.Ctx) error {
	var ps []P
	db.Where("user_id = ?", 1).Limit(20).Find(&ps)
	for _, p := range ps {
		syncOne(p)
	}
	return nil
}

func maxNumber() int {
	var n int
	db.Raw("SELECT MAX(no) FROM p").Scan(&n)
	return n
}

func move(c *fiber.Ctx) error {
	no := maxNumber()
	var ps []P
	db.Find(&ps)
	for _, p := range ps {
		no++
		db.Model(&p).Update("no", no)
	}
	return nil
}

func logAction(id uint) {
	db.Create(&Job{ID: id})
}

func removeMany(c *fiber.Ctx) error {
	var ids []uint
	c.BodyParser(&ids)
	db.Where("id IN ?", ids).Delete(&P{})
	for _, id := range ids {
		logAction(id)
	}
	return nil
}
`

func TestLoopShapesFollowRealQueryCount(t *testing.T) {
	root := writeTree(t, map[string]string{"go.mod": "module demo\n\ngo 1.22\n", "main.go": loopShapesFixture})
	graphs := Analyze(root)
	rep := Costs(graphs)
	for _, path := range []string{"/retry", "/export", "/dlq", "/login", "/notif"} {
		if got := routeCost(t, rep, path); got.NPlusOne || got.Unbounded {
			t.Errorf("%s: Q(n)=%q n+1=%v tanpa-batas=%v — jumlah query tidak tumbuh per baris", path, got.QueryFormula, got.NPlusOne, got.Unbounded)
		}
	}
	if got := routeCost(t, rep, "/retry"); got.BigO != "O(n)" {
		t.Errorf("/retry: loop chunk berisi loop atas potongannya tetap O(n), bukan O(n²): %s (%s)", got.BigO, got.Formula)
	}

	hot := map[string]Hotspot{}
	for _, h := range Hotspots(graphs) {
		if h.Kind == "nplus1" {
			hot[h.Func.Name] = h
		}
	}
	for _, name := range []string{"main.retry", "main.export", "main.clearDLQ", "main.login", "main.notif"} {
		if h, ok := hot[name]; ok {
			t.Errorf("%s tidak boleh menjadi sumber N+1: %+v", name, h)
		}
	}
	// Update per baris di badan move sendiri: N+1 nyata, tetapi maxNumber (dipanggil sekali sebelum loop) bukan pembawanya.
	if mv, ok := hot["main.move"]; !ok || mv.Via != "" {
		t.Errorf("move: N+1 dari update per baris di badannya sendiri, Via harus kosong: %+v (ada=%v)", mv, ok)
	}
	// Log per ID di dalam loop: N+1 nyata lewat logAction.
	if rm, ok := hot["main.removeMany"]; !ok || rm.Via != "main.logAction" {
		t.Errorf("removeMany: N+1 lewat logAction harus tetap dilaporkan: %+v (ada=%v)", rm, ok)
	}
}
