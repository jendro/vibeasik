package api

import "testing"

// Todo #3252: setiap rute mutasi (inti + modul registry) wajib menolak permintaan tanpa token lokal.
func TestAllMutationRoutesRequireToken(t *testing.T) {
	s := newGuardTestServer(t)
	routes := s.MutationRoutes()
	if len(routes) < 100 {
		t.Fatalf("hanya %d rute mutasi terdeteksi; audit tidak menjangkau semua route", len(routes))
	}
	for _, want := range []string{"POST /api/projects", "PUT /api/settings/llm", "DELETE /api/todos/:id", "POST /mcp", "POST /api/setup/account"} {
		found := false
		for _, r := range routes {
			if r == want {
				found = true
			}
		}
		if !found {
			t.Errorf("rute %q tidak ditemukan dalam audit", want)
		}
	}
	for _, v := range s.UnguardedMutations() {
		t.Errorf("rute mutasi tanpa penjagaan token: %s", v)
	}
}
