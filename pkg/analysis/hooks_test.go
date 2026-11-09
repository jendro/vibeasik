package analysis

import (
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestQueryActiveLLM_PreFlightSanitization(t *testing.T) {
	origLLMQuery := LLMQuery
	defer func() { LLMQuery = origLLMQuery }()

	var capturedSys, capturedUser string
	LLMQuery = func(db *gorm.DB, sys, user string) (string, error) {
		capturedSys = sys
		capturedUser = user
		return "mock response", nil
	}

	rawSys := "System prompt with ADMIN_PASSWORD=SuperSecretPass123! and API_KEY: sk-proj-12345678901234567890"
	rawUser := "User prompt containing DATABASE_URL=postgres://root:SuperSecret123@localhost:5432/db"

	resp, err := queryActiveLLM(nil, rawSys, rawUser)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "mock response" {
		t.Errorf("expected 'mock response', got %q", resp)
	}

	// Verifikasi rahasia tidak pernah lolos ke LLMQuery
	if strings.Contains(capturedSys, "SuperSecretPass123!") {
		t.Errorf("system prompt leaked password to LLM: %s", capturedSys)
	}
	if strings.Contains(capturedSys, "sk-proj-12345678901234567890") {
		t.Errorf("system prompt leaked api key to LLM: %s", capturedSys)
	}
	if strings.Contains(capturedUser, "SuperSecret123") {
		t.Errorf("user prompt leaked db password to LLM: %s", capturedUser)
	}

	// Verifikasi alias deterministik terpasang
	if !strings.Contains(capturedSys, "[ALIASED_") {
		t.Errorf("expected aliased token in system prompt, got: %s", capturedSys)
	}
	if !strings.Contains(capturedUser, "[ALIASED_") {
		t.Errorf("expected aliased token in user prompt, got: %s", capturedUser)
	}
}
