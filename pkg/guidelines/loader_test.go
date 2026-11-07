package guidelines

import "testing"

const testPortsMD = "* Proyek baru mengambil **slot bebas terkecil berikutnya** — saat ini slot `35`.\n\n" +
	"| Slot | Blok | Proyek (id) | Komponen → port rencana _(run_port sekarang)_ |\n" +
	"| :--- | :--- | :--- | :--- |\n" +
	"| 2 | 4120–4129 | silatsolo.com (#12) | SilatSoloFE (#131, frontend_client) → **4120** _(sekarang 3001)_<br>SilatsoloBE (#132, backend) → **4125** _(sekarang 4125)_ |\n" +
	"| — | 87xx | Vibeasik Platform (#44) | Cloud Server (#151) → Aturan 8 (8760/8765/8770) |\n"

func TestParsePortRegistry(t *testing.T) {
	reg := ParsePortRegistry("PORTS.md", testPortsMD)
	if reg.NextSlot != 35 {
		t.Errorf("next slot = %d, want 35", reg.NextSlot)
	}
	p := reg.Projects[12]
	if p.Slot != 2 || p.Components[131] != 4120 || p.Components[132] != 4125 {
		t.Errorf("slot 2 parsed wrong: %+v", p)
	}
	if reg.Projects[44].Slot != -1 {
		t.Errorf("vibeasik row should be slot -1, got %+v", reg.Projects[44])
	}
}
