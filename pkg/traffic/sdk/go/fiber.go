package vibeasiktraffic

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Fiber = middleware Go Fiber v2: app.Use(vibeasiktraffic.Fiber()) — pasang setelah recover, sebelum route.
// Panic dilaporkan sebagai event error lalu diteruskan ke middleware recover; error yang dikembalikan handler (5xx) juga dilaporkan.
func Fiber() fiber.Handler {
	if Enabled() {
		ensureStarted()
	}
	return func(c *fiber.Ctx) error {
		if !Enabled() {
			return c.Next()
		}
		start := time.Now()
		defer func() {
			if p := recover(); p != nil {
				RecordPanic(p, string([]byte(c.Method())), string([]byte(c.Route().Path)), string([]byte(c.Path())), fiber.StatusInternalServerError)
				recordFiber(c, start, fiber.StatusInternalServerError)
				panic(p) // diteruskan ke recover.New()
			}
		}()
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			status = errStatus(err)
			if status >= 500 {
				RecordError(err, string([]byte(c.Method())), string([]byte(c.Route().Path)), string([]byte(c.Path())), status)
			}
		}
		recordFiber(c, start, status)
		return err
	}
}

func errStatus(err error) int {
	if fe, ok := err.(*fiber.Error); ok {
		return fe.Code
	}
	return fiber.StatusInternalServerError
}

func recordFiber(c *fiber.Ctx, start time.Time, status int) {
	h := http.Header{}
	for _, k := range []string{"CF-Connecting-IP", "True-Client-IP", "X-Real-IP", "X-Forwarded-For", "CF-IPCountry", "X-Vercel-IP-Country", "CloudFront-Viewer-Country", "X-Country-Code"} {
		if v := c.Get(k); v != "" {
			h.Set(k, v)
		}
	}
	ip := ClientIP(h, c.IP())
	// Salin string: buffer fasthttp dipakai ulang setelah handler selesai.
	Record(Event{
		TS: start.UnixMilli(), Method: c.Method(), Route: string([]byte(c.Route().Path)), Path: string([]byte(c.Path())), Status: status,
		DurMS:   float64(time.Since(start).Microseconds()) / 1000,
		BytesIn: int64(len(c.Request().Body())), BytesOut: int64(len(c.Response().Body())),
		IP: string([]byte(ip)), Country: Country(h), UA: string(c.Request().Header.UserAgent()), Referer: RefererHost(string(c.Request().Header.Referer())),
	})
}
