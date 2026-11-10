// Aplikasi contoh yang sengaja error (Go Fiber + SDK traffic). Dijalankan oleh uji ujung-ke-ujung di pkg/api.
package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	vt "github.com/vibeasik/vibeasik/pkg/traffic/sdk/go"
)

type DBError struct{ ID int }

func (e *DBError) Error() string { return fmt.Sprintf("db down id=%d", e.ID) }

func loadUser(id int, token string) {
	_ = token
	panic(&DBError{ID: id})
}

func main() {
	var n atomic.Int32
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(recover.New())
	app.Use(vt.Fiber())
	app.Get("/boom/:id", func(c *fiber.Ctx) error {
		loadUser(int(n.Add(1)), c.Get("Authorization"))
		return nil
	})
	app.Get("/ok", func(c *fiber.Ctx) error { return c.SendString("ok") })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	go func() { _ = app.Listener(ln) }()

	base := "http://" + ln.Addr().String()
	for _, p := range []string{"/boom/11", "/boom/22", "/ok", "/boom/33", "/ok"} {
		req, _ := http.NewRequest(http.MethodGet, base+p+"?token=SECRETQ", strings.NewReader(`{"password":"SECRETB"}`))
		req.Header.Set("Authorization", "Bearer SECRETH")
		req.Header.Set("Cookie", "sid=SECRETC")
		if resp, err := http.DefaultClient.Do(req); err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}
	vt.Flush()
	fmt.Println("selesai")
	os.Exit(0)
}
