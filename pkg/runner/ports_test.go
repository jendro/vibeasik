package runner

import (
	"fmt"
	"net"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestAllocate_ExistingValidRunPort_NoReassign(t *testing.T) {
	db := setupDB(t)
	slot := 0
	db.Create(&models.Project{ID: 1, Name: "Proj1", Slug: "proj1", PortSlot: &slot})
	comp := models.Component{
		ID:        101,
		ProjectID: 1,
		Name:      "api",
		Role:      models.RoleBackend,
		Port:      8080,
		RunPort:   4500,
	}
	db.Create(&comp)

	// When project already has a slot and reassign=false, keep RunPort 4500 even if it's not the slot planned port
	got, err := Allocate(db, &comp, false)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if got != 4500 {
		t.Fatalf("got port %d, want 4500", got)
	}
	if comp.RunPort != 4500 {
		t.Fatalf("comp.RunPort = %d, want 4500", comp.RunPort)
	}
}

func TestAllocate_ExistingRunPort_WithReassign(t *testing.T) {
	db := setupDB(t)
	slot := 0
	db.Create(&models.Project{ID: 1, Name: "Proj1", Slug: "proj1", PortSlot: &slot})
	// Backend in slot 0 planned port is slotBase(0)+5 = 4105
	comp := models.Component{
		ID:        101,
		ProjectID: 1,
		Name:      "api",
		Role:      models.RoleBackend,
		Port:      8080,
		RunPort:   4105,
	}
	db.Create(&comp)

	// With reassign=true, RunPort 4105 must be abandoned because comp.RunPort is 4105.
	// Since 4105 is comp.RunPort, usable(4105) will be false (p != comp.RunPort).
	// It should pick another free port in slot 0, e.g. 4100, 4101, etc.
	got, err := Allocate(db, &comp, true)
	if err != nil {
		t.Fatalf("Allocate with reassign failed: %v", err)
	}
	if got == 4105 {
		t.Fatalf("reassign must not choose original RunPort 4105, got %d", got)
	}
	if got < slotBase(0) || got >= slotBase(0)+SlotSize {
		t.Fatalf("reassigned port %d should be within slot 0 [%d, %d)", got, slotBase(0), slotBase(0)+SlotSize)
	}
	if comp.RunPort != got {
		t.Fatalf("comp.RunPort %d != returned port %d", comp.RunPort, got)
	}

	var updated models.Component
	db.First(&updated, comp.ID)
	if updated.RunPort != got {
		t.Fatalf("DB RunPort %d != returned port %d", updated.RunPort, got)
	}
}

func TestAllocate_NoProject_UsesOverflow(t *testing.T) {
	db := setupDB(t)
	comp := models.Component{
		ID:        201,
		ProjectID: 0,
		Name:      "orphan",
		Role:      models.RoleFrontendClient,
		Port:      3000,
		RunPort:   0,
	}
	db.Create(&comp)

	got, err := Allocate(db, &comp, false)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if got < OverflowStart || got > OverflowEnd {
		t.Fatalf("got port %d, want within overflow range [%d, %d]", got, OverflowStart, OverflowEnd)
	}
	if comp.RunPort != got {
		t.Fatalf("comp.RunPort = %d, want %d", comp.RunPort, got)
	}
}

func TestAllocate_SlotPortBusy_FallsBackToSlotOffset(t *testing.T) {
	db := setupDB(t)
	slot := freeSlot(t)
	db.Create(&models.Project{ID: 1, Name: "Proj1", Slug: "proj1", PortSlot: &slot})

	plannedPort := slotBase(slot) // Frontend client takes +0
	busyLn, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", plannedPort))
	if err != nil {
		t.Skipf("cannot bind planned port %d for test: %v", plannedPort, err)
	}
	defer busyLn.Close()

	comp := models.Component{
		ID:        301,
		ProjectID: 1,
		Name:      "web",
		Role:      models.RoleFrontendClient,
		Port:      3000,
		RunPort:   0,
	}
	db.Create(&comp)

	got, err := Allocate(db, &comp, false)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if got == plannedPort {
		t.Fatalf("got busy port %d, should have bypassed it", got)
	}
	if got < slotBase(slot) || got >= slotBase(slot)+SlotSize {
		t.Fatalf("got port %d outside slot %d [%d, %d)", got, slot, slotBase(slot), slotBase(slot)+SlotSize)
	}
}

func TestAllocate_SlotFull_FallsBackToOverflow(t *testing.T) {
	db := setupDB(t)
	slot := 0
	db.Create(&models.Project{ID: 1, Name: "Proj1", Slug: "proj1", PortSlot: &slot})

	// Claim all 10 ports of slot 0 with other components
	base := slotBase(slot)
	for i := 0; i < SlotSize; i++ {
		other := models.Component{
			ID:        uint(1000 + i),
			ProjectID: 1,
			Name:      fmt.Sprintf("other-%d", i),
			RunPort:   base + i,
		}
		db.Create(&other)
	}

	comp := models.Component{
		ID:        401,
		ProjectID: 1,
		Name:      "new-comp",
		Role:      models.RoleBackend,
		Port:      8080,
		RunPort:   0,
	}
	db.Create(&comp)

	got, err := Allocate(db, &comp, false)
	if err != nil {
		t.Fatalf("Allocate should fallback to overflow, failed: %v", err)
	}
	if got < OverflowStart || got > OverflowEnd {
		t.Fatalf("got port %d, want in overflow range [%d, %d]", got, OverflowStart, OverflowEnd)
	}
}

func TestAllocate_ReassignWhenRunPortClaimedByOther(t *testing.T) {
	db := setupDB(t)
	slot := 1
	db.Create(&models.Project{ID: 1, Name: "Proj1", Slug: "proj1", PortSlot: &slot})

	conflictPort := slotBase(slot)
	other := models.Component{
		ID:        500,
		ProjectID: 1,
		Name:      "existing",
		RunPort:   conflictPort,
	}
	db.Create(&other)

	// comp has RunPort = conflictPort, but reassign is false.
	// Since other claimed conflictPort, Allocate cannot keep conflictPort!
	comp := models.Component{
		ID:        501,
		ProjectID: 1,
		Name:      "conflicted",
		Role:      models.RoleFrontendClient,
		RunPort:   conflictPort,
	}
	db.Create(&comp)

	got, err := Allocate(db, &comp, false)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if got == conflictPort {
		t.Fatalf("should not allocate conflictPort %d owned by other", conflictPort)
	}
}
