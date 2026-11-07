package database

import (
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestClassifyLegacyTodoCategory(t *testing.T) {
	cases := []struct{ title, ref, want string }{
		{"apa saja", "SEC-001", models.CategorySecurity},
		{"Perbaiki ARSITEKTUR modul", "", models.CategoryArchitecture},
		{"Jalankan Runner", "", models.CategoryArchitecture},
		{"x", "arch-2", models.CategoryArchitecture},
		{"Tambah Composable baru", "", models.CategoryUIUX},
		{"cek HEALTH", "", models.CategoryBackend},
		{"Atur Proxy", "", models.CategoryPerformance},
		{"Neural net", "", models.CategoryFeature},
		{"x", "PLAN-3", models.CategoryFeature},
		{"Setup CI", "", models.CategoryDevOps},
		{"xyz", "", models.CategoryGeneral},
		// prioritas: SEC menang atas ARCH; ARCH menang atas FE
		{"runner", "SEC-9", models.CategorySecurity},
		{"runner UI", "", models.CategoryArchitecture},
	}
	for _, c := range cases {
		if got := classifyLegacyTodoCategory(c.title, c.ref); got != c.want {
			t.Errorf("%q/%q = %q, want %q", c.title, c.ref, got, c.want)
		}
	}
}
