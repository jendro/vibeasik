package payload

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestValidateHelpers(t *testing.T) {
	if NeedText("x", "  ", 5) == "" || NeedText("x", "abcdef", 5) == "" || NeedText("x", "abc", 5) != "" {
		t.Error("NeedText")
	}
	if MaxLen("x", strings.Repeat("é", 6), 5) == "" || MaxLen("x", strings.Repeat("é", 5), 5) != "" {
		t.Error("MaxLen menghitung karakter, bukan byte")
	}
	if OneOf("m", "", "a", "b") != "" || OneOf("m", "a", "a", "b") != "" || OneOf("m", "z", "a", "b") == "" {
		t.Error("OneOf")
	}
	if NeedOneOf("m", "", "a") == "" || NeedID("id", 0) == "" || NeedID("id", 3) != "" {
		t.Error("NeedOneOf/NeedID")
	}
	if AbsPath("p", "") != "" || AbsPath("p", "/tmp/x") != "" || AbsPath("p", "rel/x") == "" || AbsPath("p", "/a\x00b") == "" {
		t.Error("AbsPath")
	}
	if FirstProblem("", "b", "c") != "b" || FirstProblem("", "") != "" {
		t.Error("FirstProblem")
	}
}

func TestBindJSONAndParamID(t *testing.T) {
	app := fiber.New()
	type in struct {
		N int `json:"n"`
	}
	app.Post("/opt/:id", func(c *fiber.Ctx) error {
		if _, p := ParamID(c, "id", "demo"); p != "" {
			return Bad(c, p)
		}
		var v in
		if p := BindJSON(c, &v); p != "" {
			return Bad(c, p)
		}
		return c.SendString("ok")
	})
	app.Post("/req", func(c *fiber.Ctx) error {
		var v in
		if p := BindJSONRequired(c, &v); p != "" {
			return Bad(c, p)
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
