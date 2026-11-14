package models

import (
	"path/filepath"
	"testing"
)

// Dua komponen dengan path relatif yang sama: kunci komponen diturunkan dari folder di bawah akar walau
// RelativePath kosong, path temuan mendahulukan komponen asalnya, dan ID yang bentrok antarkomponen dipisah.
func TestFindingComponentResolution(t *testing.T) {
	root := "/srv/app"
	p := Project{RootPath: root, Components: []Component{
		{Name: "Web", AbsolutePath: filepath.Join(root, "frontend")},
		{Name: "Admin", AbsolutePath: filepath.Join(root, "admin")},
		{Name: "Luar", AbsolutePath: "/opt/luar"},
		{Name: "Tercatat", RelativePath: "./svc/api/", AbsolutePath: filepath.Join(root, "svc/api")},
	}}
	keys := []string{p.ComponentKey(p.Components[0]), p.ComponentKey(p.Components[1]), p.ComponentKey(p.Components[2]), p.ComponentKey(p.Components[3])}
	if keys[0] != "frontend" || keys[1] != "admin" || keys[2] != "Luar" || keys[3] != "svc/api" {
		t.Fatalf("kunci komponen = %v", keys)
	}
	paths := p.FindingPaths("admin", "app/layouts/default.vue")
	if paths[0] != filepath.Join(root, "admin/app/layouts/default.vue") {
		t.Fatalf("komponen asal harus didahulukan: %v", paths)
	}
	if legacy := p.FindingPaths("", "app/layouts/default.vue"); legacy[0] != filepath.Join(root, "app/layouts/default.vue") {
		t.Fatalf("tanpa komponen urutan lama tetap: %v", legacy)
	}

	fs := DeduplicateFindings([]Finding{
		{RuleID: "FE-SMALL-FONT", TargetFile: "app/layouts/default.vue", Line: 185, Component: "frontend"},
		{RuleID: "FE-SMALL-FONT", TargetFile: "app/layouts/default.vue", Line: 185, Component: "admin"},
		{RuleID: "FE-SMALL-FONT", TargetFile: "app/layouts/default.vue", Line: 185, Component: "admin"},
	})
	if len(fs) != 2 || fs[0].ID != "FE-SMALL-FONT_app/layouts/default.vue_185" || fs[1].ID != "FE-SMALL-FONT_app/layouts/default.vue_185@admin" {
		t.Fatalf("temuan dua komponen harus terpisah dengan ID pertama tetap: %+v", fs)
	}
}
