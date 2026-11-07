package registry

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type fakeModule struct{ name, feature string }

func (f fakeModule) Name() string    { return f.name }
func (f fakeModule) Feature() string { return f.feature }
func (f fakeModule) Models() []any   { return []any{struct{}{}} }

func TestRegisterAndEntitlement(t *testing.T) {
	Register(fakeModule{"b-pro", "staff"})
	Register(fakeModule{"a-core", ""})
	all := All()
	if len(all) < 2 || all[0].Name() != "a-core" {
		t.Fatalf("modules must be sorted by name, got %v", all)
	}
	if len(Models()) < 2 {
		t.Fatalf("models from ModelProvider must be collected")
	}
	if !Enabled(all[0]) {
		t.Fatalf("community module must always be enabled")
	}
	if Has("staff") {
		t.Fatalf("pro feature must be locked by default")
	}
	SetEntitlement(func(f string) bool { return f == "staff" })
	defer SetEntitlement(func(f string) bool { return f == "" })
	if !Has("staff") {
		t.Fatalf("entitlement must unlock staff")
	}
}

func TestGateReturns402(t *testing.T) {
	app := fiber.New()
	app.Get("/x", Gate("orchestrator"), func(c *fiber.Ctx) error { return c.SendString("ok") })
	resp, err := app.Test(httptest.NewRequest("GET", "/x", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusPaymentRequired {
		t.Fatalf("want 402, got %d", resp.StatusCode)
	}
	if !IsUpgradeRequired(UpgradeRequired("x")) {
		t.Fatalf("IsUpgradeRequired must detect its own error")
	}
}
