package api

import (
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestAliasJSONForKeepsOnlyListedModels(t *testing.T) {
	if got := aliasJSONFor(nil, []string{"a"}); got != "" {
		t.Fatalf("nil aliases must mean unchanged, got %q", got)
	}
	got := aliasJSONFor(map[string]string{"vendor/a": " Model A ", "vendor/gone": "Gone", "vendor/b": "vendor/b"}, []string{"vendor/a", "vendor/b"})
	cfg := models.LLMConfig{AliasJSON: got}
	if cfg.ModelAlias("vendor/a") != "Model A" || cfg.ModelAlias("vendor/gone") != "" || cfg.ModelAlias("vendor/b") != "" {
		t.Fatalf("unexpected aliases %s", got)
	}
	if empty := aliasJSONFor(map[string]string{}, []string{"x"}); empty != "{}" {
		t.Fatalf("empty map must clear aliases, got %q", empty)
	}
}
