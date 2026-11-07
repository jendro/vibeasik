package update

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/vibeasik/vibeasik/pkg/localauth"
)

type settings struct {
	Auto *bool `json:"auto,omitempty"`
}

func settingsPath() string { return filepath.Join(localauth.Dir(), "update.json") }

// Auto true bila pembaruan boleh diunduh dan dipasang otomatis di latar (bawaan: ya).
func Auto() bool {
	var s settings
	if raw, err := os.ReadFile(settingsPath()); err == nil {
		_ = json.Unmarshal(raw, &s)
	}
	return s.Auto == nil || *s.Auto
}

// SetAuto menyimpan pilihan pembaruan otomatis.
func SetAuto(v bool) error {
	if err := os.MkdirAll(localauth.Dir(), 0o700); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(settings{Auto: &v}, "", "  ")
	return os.WriteFile(settingsPath(), raw, 0o600)
}
