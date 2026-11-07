package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestValidateHelpers(t *testing.T) {
	if needText("x", "  ", 5) == "" || needText("x", "abcdef", 5) == "" || needText("x", "abc", 5) != "" {
		t.Error("needText")
	}
	if maxLen("x", strings.Repeat("é", 6), 5) == "" || maxLen("x", strings.Repeat("é", 5), 5) != "" {
		t.Error("maxLen menghitung karakter, bukan byte")
	}
	if oneOf("m", "", "a", "b") != "" || oneOf("m", "a", "a", "b") != "" || oneOf("m", "z", "a", "b") == "" {
		t.Error("oneOf")
	}
	if needOneOf("m", "", "a") == "" || needID("id", 0) == "" || needID("id", 3) != "" {
		t.Error("needOneOf/needID")
	}
	if absPath("p", "") != "" || absPath("p", "/tmp/x") != "" || absPath("p", "rel/x") == "" || absPath("p", "/a\x00b") == "" {
		t.Error("absPath")
	}
	if firstProblem("", "b", "c") != "b" || firstProblem("", "") != "" {
		t.Error("firstProblem")
	}
}

func TestBindJSONAndParamID(t *testing.T) {
	app := fiber.New()
	type in struct {
		N int `json:"n"`
	}
	app.Post("/opt/:id", func(c *fiber.Ctx) error {
		if _, p := paramID(c, "id", "demo"); p != "" {
			return badPayload(c, p)
		}
		var v in
		if p := bindJSON(c, &v); p != "" {
			return badPayload(c, p)
		}
		return c.SendString("ok")
	})
	app.Post("/req", func(c *fiber.Ctx) error {
		var v in
		if p := bindJSONRequired(c, &v); p != "" {
			return badPayload(c, p)
		}
		return c.SendString("ok")
	})
	do := func(path, body string) int {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode
	}
	cases := []struct {
		path, body string
		want       int
	}{
		{"/opt/1", "", 200}, {"/opt/1", `{"n":2}`, 200}, {"/opt/1", `{"n":"x"}`, 400}, {"/opt/1", `{rusak`, 400},
		{"/opt/0", "", 400}, {"/opt/-3", "", 400}, {"/opt/abc", "", 400},
		{"/req", "", 400}, {"/req", `{"n":1}`, 200}, {"/req", `[1`, 400},
	}
	for _, c := range cases {
		if got := do(c.path, c.body); got != c.want {
			t.Errorf("POST %s %q → %d, want %d", c.path, c.body, got, c.want)
		}
	}
}
