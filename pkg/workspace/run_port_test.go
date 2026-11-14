package workspace

import (
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func TestRunPortPrefersAllocatedBlock(t *testing.T) {
	if got := runPort(models.Component{Port: 8091, RunPort: 4425}); got != 4425 {
		t.Fatalf("port blok proyek harus menang: %d", got)
	}
	if got := runPort(models.Component{Port: 3020}); got != 3020 {
		t.Fatalf("tanpa alokasi pakai port terdeteksi: %d", got)
	}
}
