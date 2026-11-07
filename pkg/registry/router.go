package registry

import "github.com/gofiber/fiber/v2"

// Router membungkus fiber.Router sehingga setiap route modul otomatis melewati Gate(feature).
// Modul tidak bisa lupa memasang gate, dan gate tidak bocor ke route modul lain.
type Router struct {
	r       fiber.Router
	feature string
}

// NewRouter membuat Router untuk modul dengan kunci fitur feature ("" = tanpa gate).
func NewRouter(r fiber.Router, feature string) Router { return Router{r: r, feature: feature} }

func (g Router) wrap(h []fiber.Handler) []fiber.Handler {
	if g.feature == "" {
		return h
	}
	return append([]fiber.Handler{Gate(g.feature)}, h...)
}

// Get mendaftarkan route GET.
func (g Router) Get(path string, h ...fiber.Handler) { g.r.Get(path, g.wrap(h)...) }

// Post mendaftarkan route POST.
func (g Router) Post(path string, h ...fiber.Handler) { g.r.Post(path, g.wrap(h)...) }

// Put mendaftarkan route PUT.
func (g Router) Put(path string, h ...fiber.Handler) { g.r.Put(path, g.wrap(h)...) }

// Patch mendaftarkan route PATCH.
func (g Router) Patch(path string, h ...fiber.Handler) { g.r.Patch(path, g.wrap(h)...) }

// Delete mendaftarkan route DELETE.
func (g Router) Delete(path string, h ...fiber.Handler) { g.r.Delete(path, g.wrap(h)...) }
