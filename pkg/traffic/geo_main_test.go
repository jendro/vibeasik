package traffic

import (
	"os"
	"testing"
)

// Uji tidak boleh bergantung pada basis data geo yang kebetulan terpasang di mesin pengembang.
func TestMain(m *testing.M) {
	SetGeoResolver(nil)
	os.Exit(m.Run())
}
