package vibeasiktraffic

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Fiber = middleware Go Fiber v2: app.Use(vibeasiktraffic.Fiber()) — pasang setelah recover, sebelum route.
func Fiber() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !Enabled() {
			return c.Next()
		}
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				status = fe.Code
			} else {
				status = fiber.StatusInternalServerError
			}
		}
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
		return err
	}
}
