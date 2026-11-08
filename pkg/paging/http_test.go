package paging

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestWindow(t *testing.T) {
	cases := []struct{ total, limit, offset, def, max, s, e int }{
		{10, 0, 0, 1000, 1000, 0, 10},
		{10, 3, 2, 1000, 1000, 2, 5},
		{10, 5, 8, 1000, 1000, 8, 10},
		{10, 5, 99, 1000, 1000, 10, 10},
		{10, 5, -4, 1000, 1000, 0, 5},
		{2000, 0, 0, 1000, 1000, 0, 1000},
		{2000, 5000, 0, 1000, 1000, 0, 1000},
		{0, 0, 0, 0, 0, 0, 0},
	}
	for i, c := range cases {
		s, e := Window(c.total, c.limit, c.offset, c.def, c.max)
		if s != c.s || e != c.e {
			t.Errorf("case %d: got %d,%d want %d,%d", i, s, e, c.s, c.e)
		}
	}
}

func TestApplyHeaderAndShape(t *testing.T) {
	app := fiber.New()
	app.Get("/n", func(c *fiber.Ctx) error {
		items := make([]int, 7)
		for i := range items {
			items[i] = i
		}
		return c.JSON(Apply(c, items))
	})
	app.Get("/nil", func(c *fiber.Ctx) error { return c.JSON(Apply[int](c, nil)) })
	get := func(u string) (string, string) {
		resp, err := app.Test(httptest.NewRequest("GET", u, nil))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		return string(b), resp.Header.Get(HeaderTotal)
	}
	if b, tot := get("/n"); b != "[0,1,2,3,4,5,6]" || tot != "7" {
		t.Errorf("tanpa limit: %s %s", b, tot)
	}
	b, tot := get("/n?limit=2&offset=3")
	var got []int
	_ = json.Unmarshal([]byte(b), &got)
	if len(got) != 2 || got[0] != 3 || tot != "7" {
		t.Errorf("paged: %s %s", b, tot)
	}
	if b, _ := get("/n?limit=abc&offset=x"); b != "[0,1,2,3,4,5,6]" {
		t.Errorf("invalid: %s", b)
	}
	if b, tot := get("/nil"); b != "[]" || tot != "0" {
		t.Errorf("nil: %s %s", b, tot)
	}
}
