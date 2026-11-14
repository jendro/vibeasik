package diagnostics

import (
	"fmt"
	"strings"
	"testing"
)

// goHandlerFile: berkas handler Go dengan `handlers` handler fiber dan `helpers` fungsi bantu (± 12 baris kode tiap fungsi).
func goHandlerFile(handlers, helpers int) string {
	var b strings.Builder
	b.WriteString("package handlers\n\nimport \"github.com/gofiber/fiber/v2\"\n\n")
	for i := 0; i < handlers; i++ {
		fmt.Fprintf(&b, "func Handle%d(c *fiber.Ctx) error {\n", i)
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "\tx%d := %d\n\t_ = x%d\n", j, j, j)
		}
		b.WriteString("\treturn nil\n}\n\n")
	}
	for i := 0; i < helpers; i++ {
		fmt.Fprintf(&b, "func helper%d(v int) int {\n", i)
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "\tv += %d\n", j)
		}
		b.WriteString("\treturn v\n}\n\n")
	}
	return b.String()
}

func godHit(t *testing.T, rel, src string) bool {
	t.Helper()
	path := writeFixture(t, t.TempDir(), rel, src)
	for _, f := range InspectFile(path, rel, "go") {
		if f.RuleID == "SMELL-GOD-CONTROLLER" {
			return true
		}
	}
	return false
}

// Berkas handler yang sudah dipecah per fitur (sedikit handler, banyak fungsi bantu) bukan god controller;
// banyak handler atau ukuran sangat besar tetap dilaporkan; tabel rute dilewati.
func TestGodControllerCountsUseCases(t *testing.T) {
	if godHit(t, "handlers/admin_participant_list.go", goHandlerFile(1, 40)) {
		t.Error("1 handler + fungsi bantu (satu fitur) tidak boleh dilaporkan")
	}
	if !godHit(t, "handlers/admin.go", goHandlerFile(9, 25)) {
		t.Error("9 handler dalam satu berkas besar harus dilaporkan")
	}
	if !godHit(t, "handlers/huge.go", goHandlerFile(2, 110)) {
		t.Error("berkas handler > 1200 baris kode tetap dilaporkan")
	}
	if godHit(t, "routes/routes.go", goHandlerFile(9, 25)) {
		t.Error("tabel rute tidak dilaporkan sebagai god controller")
	}
}
