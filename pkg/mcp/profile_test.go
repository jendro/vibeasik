package mcp

import "testing"

func TestTodoProfileToolsExist(t *testing.T) {
	s := &Server{}
	all := s.allTools()
	have := map[string]bool{}
	for _, x := range filterProfile(all, "todo") {
		have[x.Name] = true
	}
	for _, n := range todoCoreTools {
		if !have[n] {
			t.Errorf("tool profil todo %q tidak ada di build Community", n)
		}
	}
	if len(filterProfile(all, "")) != len(all) || len(filterProfile(all, "full")) != len(all) {
		t.Fatal("tanpa profil semua tool harus tampil")
	}
}
