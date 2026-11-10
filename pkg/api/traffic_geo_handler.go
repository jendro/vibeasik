package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/vibeasik/vibeasik/pkg/traffic/geo"
)

// getTrafficGeoStatus melaporkan apakah basis data lokasi & jaringan IP terpasang (untuk empty state Studio).
func (s *Server) getTrafficGeoStatus(c *fiber.Ctx) error {
	st := geo.Default().Status()
	return c.JSON(fiber.Map{
		"city_ready": st.CityReady, "asn_ready": st.ASNReady,
		"city_built": st.CityBuilt, "asn_built": st.ASNBuilt,
		"auto_update": geo.AutoEnabled(),
		"update_cmd":  "vibeasik geo update",
	})
}
