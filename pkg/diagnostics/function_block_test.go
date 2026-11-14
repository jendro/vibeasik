package diagnostics

import (
	"strings"
	"testing"
)

func TestFunctionBlockGo(t *testing.T) {
	src := `package user

func helper() int { return 1 }

func SaveUser(db *DB, u User) error {
	if u.Email == "" {
		return errEmpty
	}
	for _, r := range u.Roles {
		db.Exec("INSERT INTO roles VALUES (?)", r)
	}
	return nil
}

var tail = 3`
	b, ok := FunctionBlock("pkg/user/save.go", src, 10)
	if !ok || b.Name != "SaveUser" || b.Start != 5 || b.End != 13 || b.Line != 10 || b.Cut {
		t.Fatalf("blok fungsi Go: %+v", b)
	}
	if !strings.Contains(b.Code, "▶ 10 │ \t\tdb.Exec(") || !strings.Contains(b.Code, " 5 │ func SaveUser") || !strings.Contains(b.Code, "13 │ }") || strings.Contains(b.Code, "helper") {
		t.Errorf("kode blok harus fungsi utuh bernomor dengan penanda:\n%s", b.Code)
	}
	if got := b.Label(); got != "fungsi `SaveUser` (baris 5–13)" {
		t.Errorf("label = %q", got)
	}
	// Baris di luar fungsi bernama → potongan sekitar baris.
	top, ok := FunctionBlock("pkg/user/save.go", src, 15)
	if !ok || top.Name != "" || top.Start != 7 || top.End != 15 || !strings.Contains(top.Code, "▶ 15 │ var tail") {
		t.Fatalf("potongan di luar fungsi: %+v", top)
	}
	if _, ok := FunctionBlock("pkg/user/save.go", src, 99); ok {
		t.Error("baris di luar berkas tidak punya blok")
	}
}

func TestFunctionBlockVueScriptAndLongFunction(t *testing.T) {
	vue := `<template>
  <p class="text-[10px]">{{ total }}</p>
</template>

<script setup lang="ts">
const items = ref([])
async function load() {
  const res = await $fetch('/api/items')
  items.value = res
}
</script>`
	b, ok := FunctionBlock("app/pages/items.vue", vue, 8)
	if !ok || b.Name != "load" || b.Start != 7 || b.End != 10 {
		t.Fatalf("fungsi di <script> Vue: %+v", b)
	}
	tpl, ok := FunctionBlock("app/pages/items.vue", vue, 2)
	if !ok || tpl.Name != "" || !strings.Contains(tpl.Code, "▶  2 │   <p class") {
		t.Fatalf("baris template memakai potongan: %+v\n%s", tpl, tpl.Code)
	}

	var long strings.Builder
	long.WriteString("function big() {\n")
	for i := 0; i < 150; i++ {
		long.WriteString("  step()\n")
	}
	long.WriteString("}\n")
	lb, ok := FunctionBlock("src/big.js", long.String(), 100)
	if !ok || lb.Name != "big" || lb.Start != 1 || lb.End != 152 || !lb.Cut {
		t.Fatalf("fungsi panjang dipotong di sekitar baris: %+v", lb)
	}
	if n := strings.Count(lb.Code, "\n") + 1; n > funcBlockMaxLines+3 || !strings.Contains(lb.Code, "⋯") || !strings.Contains(lb.Code, "▶ 100 │") {
		t.Errorf("potongan fungsi panjang (%d baris):\n%s", n, lb.Code)
	}
}

func TestFunctionBlockPython(t *testing.T) {
	src := "import os\n\ndef load(path):\n    with open(path) as f:\n        data = f.read()\n    return data\n\nVALUE = 1\n"
	b, ok := FunctionBlock("tools/load.py", src, 5)
	if !ok || b.Name != "load" || b.Start != 3 || b.End != 6 {
		t.Fatalf("fungsi Python: %+v", b)
	}
}
