package routemap

import (
	"go/ast"
	"go/types"
	"sort"
)

// Pemeriksaan tipe parsial dengan go/types (type checker resmi Go, mesin yang sama dengan gopls): paket lokal
// dicek dari AST, paket eksternal diganti stub kosong sehingga tidak perlu mengunduh dependensi. Error tipe
// (anggota paket eksternal yang tidak dikenal) diabaikan; yang dibutuhkan hanya resolusi pemanggilan fungsi &
// method lokal (Uses/Selections) dan implementasi interface lokal untuk dispatch ala CHA.

type typeIndex struct {
	info  *types.Info
	pkgs  map[string]*types.Package
	named []*types.TypeName // tipe bernama lokal non-interface (kandidat implementasi interface)
	impls map[*types.Func][]string
}

type localImporter struct {
	m       *module
	byPkg   map[string][]*ast.File
	ti      *typeIndex
	loading map[string]bool
}

func (im *localImporter) Import(path string) (*types.Package, error) {
	return im.ImportFrom(path, "", 0)
}

func (im *localImporter) ImportFrom(path, _ string, _ types.ImportMode) (*types.Package, error) {
	if p, ok := im.ti.pkgs[path]; ok {
		return p, nil
	}
	files, local := im.byPkg[path]
	if !local || im.loading[path] {
		name := defaultPkgName(path)
		if n, ok := im.m.pkgNames[path]; ok {
			name = n
		}
		p := types.NewPackage(path, name)
		p.MarkComplete()
		if !local {
			im.ti.pkgs[path] = p
		}
		return p, nil
	}
	im.loading[path] = true
	conf := types.Config{Importer: im, Error: func(error) {}, FakeImportC: true}
	p, _ := conf.Check(path, im.m.fset, files, im.ti.info)
	delete(im.loading, path)
	if p != nil {
		im.ti.pkgs[path] = p
	}
	return p, nil
}

// typeCheck menjalankan go/types untuk semua paket lokal modul.
func (m *module) typeCheck() {
	byPkg := map[string][]*ast.File{}
	for _, fi := range m.files {
		byPkg[fi.Pkg] = append(byPkg[fi.Pkg], fi.AST)
	}
	ti := &typeIndex{
		info: &types.Info{Uses: map[*ast.Ident]types.Object{}, Defs: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}},
		pkgs: map[string]*types.Package{}, impls: map[*types.Func][]string{},
	}
	im := &localImporter{m: m, byPkg: byPkg, ti: ti, loading: map[string]bool{}}
	paths := make([]string, 0, len(byPkg))
	for p := range byPkg {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		_, _ = im.Import(p)
	}
	for _, p := range paths {
		pkg := ti.pkgs[p]
		if pkg == nil {
			continue
		}
		for _, n := range pkg.Scope().Names() {
			if tn, ok := pkg.Scope().Lookup(n).(*types.TypeName); ok && !types.IsInterface(tn.Type()) {
				ti.named = append(ti.named, tn)
			}
		}
	}
	m.types = ti
}

// funcKey memetakan *types.Func lokal ke key funcInfo ("<pkg>.Func" / "<pkg>.Type.Method").
func funcKey(f *types.Func) string {
	f = f.Origin()
	if f.Pkg() == nil {
		return ""
	}
	sig, _ := f.Type().(*types.Signature)
	if sig != nil && sig.Recv() != nil {
		t := sig.Recv().Type()
		if p, ok := t.(*types.Pointer); ok {
			t = p.Elem()
		}
		if n, ok := t.(*types.Named); ok {
			return f.Pkg().Path() + "." + n.Origin().Obj().Name() + "." + f.Name()
		}
		return ""
	}
	return f.Pkg().Path() + "." + f.Name()
}

// typedFunc: objek fungsi yang dirujuk ekspresi menurut go/types (nil bila tidak diketahui).
func (m *module) typedFunc(e ast.Expr) *types.Func {
	if m.types == nil {
		return nil
	}
	var id *ast.Ident
	switch x := e.(type) {
	case *ast.Ident:
		id = x
	case *ast.SelectorExpr:
		if sel := m.types.info.Selections[x]; sel != nil {
			f, _ := sel.Obj().(*types.Func)
			return f
		}
		id = x.Sel
	default:
		return nil
	}
	f, _ := m.types.info.Uses[id].(*types.Func)
	return f
}

// implementations: key method konkret lokal yang mengimplementasikan method interface f (CHA).
func (m *module) implementations(f *types.Func) []string {
	if out, ok := m.types.impls[f]; ok {
		return out
	}
	var out []string
	sig, _ := f.Type().(*types.Signature)
	if sig != nil && sig.Recv() != nil {
		if iface, ok := sig.Recv().Type().Underlying().(*types.Interface); ok {
			for _, tn := range m.types.named {
				t := tn.Type()
				if !types.Implements(t, iface) && !types.Implements(types.NewPointer(t), iface) {
					continue
				}
				if k := tn.Pkg().Path() + "." + tn.Name() + "." + f.Name(); m.funcs[k] != nil {
					out = append(out, k)
				}
			}
		}
	}
	sort.Strings(out)
	m.types.impls[f] = out
	return out
}

func isInterfaceMethod(f *types.Func) bool {
	sig, _ := f.Type().(*types.Signature)
	return sig != nil && sig.Recv() != nil && types.IsInterface(sig.Recv().Type())
}
