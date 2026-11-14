package security

import (
	"strings"
	"testing"
)

// Kasus nyata dari diff QA gate: nama berkas, identifier, dan pemanggilan fungsi bukan rahasia.
func TestSanitizeKeepsCodeAndFileNames(t *testing.T) {
	s := NewSanitizer()
	keep := []string{
		"migrations/041_payments_status_needs_review.sql",
		"+ 041_payments_status_needs_review.sql dibuat",
		"const token = useAdminToken()",
		"const token = useAdminToken();",
		"apiKey := os.Getenv(\"API_KEY\")",
		"const secret = process.env.JWT_SECRET",
		"password = req.body.password",
		"token: string;",
		"AUTH_HEADER_NAME_FOR_INTERNAL_SERVICES",
		"func handleAdminPaymentReconciliationRequest()",
		"TOKEN=${ADMIN_TOKEN_FROM_ENVIRONMENT}",
	}
	for _, in := range keep {
		out, _ := s.Sanitize(in)
		if out != in {
			t.Errorf("bukan rahasia ikut disamarkan:\n in: %s\nout: %s", in, out)
		}
	}
}

func TestSanitizeStillMasksLiterals(t *testing.T) {
	s := NewSanitizer()
	mask := map[string]string{
		"const token = 'abcd1234efgh5678'":            "abcd1234efgh5678",
		"ADMIN_TOKEN=plainsecretvalue":                "plainsecretvalue",
		"DB_PASSWORD: \"hunter2hunter2\"":             "hunter2hunter2",
		"key " + "f3a9c1d2e4b5a6978877665544332211aa": "f3a9c1d2e4b5a6978877665544332211aa",
		"id " + "9fK2xQ7mLp4Rt8Vb1Nc6Hs3Jd5Wg0Ya2":    "9fK2xQ7mLp4Rt8Vb1Nc6Hs3Jd5Wg0Ya2",
	}
	for in, secret := range mask {
		out, _ := s.Sanitize(in)
		if strings.Contains(out, secret) {
			t.Errorf("rahasia lolos: %s -> %s", in, out)
		}
	}
}
