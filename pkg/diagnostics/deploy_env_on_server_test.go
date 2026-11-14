package diagnostics

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// deploy.sh yang menyatakan env-on-server dan tidak menyalin .env apa pun lolos aturan env; yang menyatakan
// penanda tetapi tetap menyalin .env masih dinilai.
func TestDeployScriptEnvOnServer(t *testing.T) {
	cases := map[string]int{
		"#!/usr/bin/env bash\n# vibeasik:env-on-server\nexec ./backend/scripts/deploy-prod.sh \"$@\"\n": 0,
		"#!/usr/bin/env bash\n# vibeasik:env-on-server\ncp backend/.env.server backend/.env\n":          1,
		"#!/usr/bin/env bash\nexec ./backend/scripts/deploy-prod.sh \"$@\"\n":                           2,
	}
	for body, want := range cases {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "deploy.sh"), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		if got := len(inspectDeployScript(&models.Project{RootPath: root})); got != want {
			t.Errorf("deploy.sh %q: %d temuan, mau %d", body, got, want)
		}
	}
}
