package guidelines

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePortRegistryRoundTrip(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "PORTS.md"))
	if err != nil {
		t.Skip("PORTS.md repo tidak ada")
	}
	path := filepath.Join(t.TempDir(), "PORTS.md")
	os.WriteFile(path, src, 0o644)
	before := ParsePortRegistry(path, string(src))

	rows := []PortRow{
		{Slot: 2, ProjectID: 12, ProjectName: "silatsolo.com", Comps: []PortRowComp{
			{ID: 131, Name: "SilatSoloFE", Role: "frontend_client", Port: 4120, RunPort: 4120},
			{ID: 132, Name: "SilatsoloBE", Role: "backend", Port: 4125, RunPort: 8082},
		}},
		{Slot: 35, ProjectID: 99, ProjectName: "Baru", Comps: []PortRowComp{{ID: 900, Name: "BaruFE", Role: "frontend_client", Port: 4450, RunPort: 4450}}},
	}
	if err := WritePortRegistry(path, rows, 36); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	out := string(raw)
	after := ParsePortRegistry(path, out)

	if after.NextSlot != 36 {
		t.Errorf("next slot = %d", after.NextSlot)
	}
	if len(after.Projects) != len(before.Projects)+1 {
		t.Errorf("rows lost or duplicated: before %d after %d", len(before.Projects), len(after.Projects))
	}
	if after.Projects[99].Components[900] != 4450 || after.Projects[12].Components[132] != 4125 {
		t.Errorf("new/updated rows not parsed back: %+v %+v", after.Projects[99], after.Projects[12])
	}
	if after.Projects[44].Slot != -1 || after.Projects[10].Components[22] != 4100 {
		t.Error("untouched rows (vibeasik —, other projects) must be kept")
	}
	if !strings.Contains(out, "SilatSoloFE (#131, frontend_client) → **4120**<br>") || !strings.Contains(out, "**4125** _(sekarang 8082)_") {
		t.Error("run_port equal to plan must drop the 'sekarang' note, different must keep it")
	}
	if strings.Index(out, "| 35 | 4450–4459 | Baru (#99)") < strings.Index(out, "| 33 |") || strings.Index(out, "| — |") < strings.Index(out, "| 35 |") {
		t.Error("rows must be sorted by slot with — rows last")
	}
	if !strings.Contains(out, "dikelola otomatis oleh engine") {
		t.Error("status note not replaced")
	}
}
