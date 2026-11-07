package registry

import (
	"errors"
	"fmt"
	"sync"

	"github.com/gofiber/fiber/v2"
)

var (
	entMu sync.RWMutex
	// hasFeature diganti klien akun (login & entitlement bertanda tangan). Bawaan: hanya fitur Community.
	hasFeature = func(feature string) bool { return feature == "" }
)

// SetEntitlement memasang pemeriksa fitur, dipanggil oleh klien akun setelah entitlement diverifikasi.
func SetEntitlement(fn func(feature string) bool) {
	entMu.Lock()
	defer entMu.Unlock()
	hasFeature = fn
}

// Has melaporkan apakah fitur aktif untuk pengguna ini. Fitur "" selalu aktif.
func Has(feature string) bool {
	if feature == "" {
		return true
	}
	entMu.RLock()
	defer entMu.RUnlock()
	return hasFeature(feature)
}

// Enabled melaporkan apakah modul boleh dipakai sekarang.
func Enabled(m Module) bool { return Has(m.Feature()) }

// UpgradeURL halaman paket yang ditampilkan saat fitur terkunci.
var UpgradeURL = "https://vibeasik.com/pricing"

// ErrUpgradeRequired dikembalikan saat fitur butuh paket berbayar atau tidak ada di build ini.
type ErrUpgradeRequired struct{ Feature string }

func (e *ErrUpgradeRequired) Error() string {
	return fmt.Sprintf("upgrade_required: fitur %q butuh paket Pro (%s)", e.Feature, UpgradeURL)
}

// UpgradeRequired membuat ErrUpgradeRequired untuk feature.
func UpgradeRequired(feature string) error { return &ErrUpgradeRequired{Feature: feature} }

// IsUpgradeRequired melaporkan apakah err berasal dari fitur terkunci.
func IsUpgradeRequired(err error) bool {
	var e *ErrUpgradeRequired
	return errors.As(err, &e)
}

// Gate adalah middleware Fiber yang menjawab HTTP 402 bila fitur belum aktif.
func Gate(feature string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if Has(feature) {
			return c.Next()
		}
		return RespondUpgrade(c, feature)
	}
}

// RespondUpgrade menulis jawaban 402 standar untuk fitur terkunci.
func RespondUpgrade(c *fiber.Ctx, feature string) error {
	return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
		"error":       "upgrade_required",
		"feature":     feature,
		"upgrade_url": UpgradeURL,
	})
}
