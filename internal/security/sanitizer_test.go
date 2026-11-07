package security

import (
	"strings"
	"testing"
)

// fakeGitHubToken disusun dari potongan agar pemindai rahasia tidak menganggapnya token asli.
var fakeGitHubToken = "ghp" + "_1234567890abcdefghijklmnopqrstuvwxyz"

func TestSanitizeSecrets(t *testing.T) {
	input := `
	Config details:
	OPENAI_KEY: sk-proj-1234567890abcdef1234567890abcdef1234
	GOOGLE_KEY: AIzaSyD9876543210abcdefghijklmnopqrs
	DATABASE_URL: postgres://admin:SuperSecretPass123!@localhost:5432/myapp
	GITHUB: ` + fakeGitHubToken + `
	Repeated OpenAI: sk-proj-1234567890abcdef1234567890abcdef1234
	`

	cleaned, audit := GlobalSanitizer.Sanitize(input)

	if audit.RedactedCount < 4 {
		t.Errorf("Expected at least 4 redacted records, got %d", audit.RedactedCount)
	}

	if strings.Contains(cleaned, "sk-proj-1234567890") {
		t.Errorf("OpenAI key was not redacted in text!")
	}
	if strings.Contains(cleaned, "AIzaSyD9876543210") {
		t.Errorf("Google key was not redacted in text!")
	}
	if strings.Contains(cleaned, "SuperSecretPass123!") {
		t.Errorf("Database password was not redacted in text!")
	}
	if strings.Contains(cleaned, "ghp_1234567890") {
		t.Errorf("GitHub token was not redacted in text!")
	}

	// Verify alias is present
	if !strings.Contains(cleaned, "[ALIASED_") {
		t.Errorf("Expected cleaned text to contain [ALIASED_..., got:\n%s", cleaned)
	}
}

func TestCustomKeywordCensorship(t *testing.T) {
	rules := []CustomKeywordRule{
		{
			ID:          1,
			Keyword:     "CompanySecret123",
			Category:    "company_secret",
			Description: "Internal password",
			IsRegex:     false,
			IsActive:    true,
		},
		{
			ID:          2,
			Keyword:     `auth-internal-[a-z0-9]{8}`,
			Category:    "internal_token",
			Description: "Internal service token regex",
			IsRegex:     true,
			IsActive:    true,
		},
	}

	GlobalSanitizer.SetCustomKeywords(rules)

	sample := "Please connect using CompanySecret123 and token auth-internal-abc12345 in our private backend."
	cleaned, audit := GlobalSanitizer.Sanitize(sample)

	if audit.CustomMatches != 2 {
		t.Errorf("Expected 2 custom keyword matches, got %d", audit.CustomMatches)
	}

	if strings.Contains(cleaned, "CompanySecret123") {
		t.Errorf("Custom keyword was not censored: %s", cleaned)
	}

	if strings.Contains(cleaned, "auth-internal-abc12345") {
		t.Errorf("Custom regex token was not censored: %s", cleaned)
	}

	if !strings.Contains(cleaned, "[ALIASED_COMPANY_SECRET_") {
		t.Errorf("Expected alias [ALIASED_COMPANY_SECRET_..., got: %s", cleaned)
	}

	if !strings.Contains(cleaned, "[ALIASED_INTERNAL_TOKEN_") {
		t.Errorf("Expected alias [ALIASED_INTERNAL_TOKEN_..., got: %s", cleaned)
	}
}
