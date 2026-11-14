package diagnostics

import "testing"

// N+1 per baris tidak dilaporkan untuk loop yang dipotong konstanta, query ke tabel yang berbeda tiap putaran,
// dan alat CLI pemeliharaan di cmd/; loop atas data biasa tetap dilaporkan.
func TestNPlusOneLineRuleRealShapes(t *testing.T) {
	capped := `package main

func show(rows []Group) {
	for i, dp := range rows {
		if i >= 3 {
			break
		}
		db.Where("user_id = ?", dp.UserID).Find(&out)
	}
	for _, dp := range rows {
		db.Where("user_id = ?", dp.UserID).Find(&out)
	}
}
`
	wantLines(t, ruleLines(t, "internal/report/show.go", "go", capped), "QUERY-NPLUS1", 11)

	sameLine := "package main\n\nfunc f(rows []R) {\n\tfor i, r := range rows {\n\t\tif i > 10 { break }\n\t\tdb.First(&r, r.ID)\n\t}\n}\n"
	wantLines(t, ruleLines(t, "internal/x/f.go", "go", sameLine), "QUERY-NPLUS1")

	tables := `package handlers

func list(results []Row) {
	for _, r := range results {
		var n int64
		_ = config.DB.Table(r.TableName).Count(&n).Error
	}
	for _, r := range results {
		var n int64
		config.DB.Table("users").Where("id = ?", r.ID).Count(&n)
	}
}
`
	wantLines(t, ruleLines(t, "handlers/legacy.go", "go", tables), "QUERY-NPLUS1", 10)

	tool := "package main\n\nfunc main() {\n\tfor _, u := range users {\n\t\tconfig.DB.Where(\"user_id = ?\", u.ID).Find(&ps)\n\t}\n}\n"
	wantLines(t, ruleLines(t, "cmd/datacheck/main.go", "go", tool), "QUERY-NPLUS1")
	wantLines(t, ruleLines(t, "cmd/server/main.go", "go", tool), "QUERY-NPLUS1", 5)
}

// Latar modal dengan @click.self bukan kontrol; elemen dengan handler klik biasa tetap dilaporkan.
func TestClickableSkipsSelfOnlyBackdrop(t *testing.T) {
	vue := `<template>
  <div v-if="open" class="fixed inset-0 bg-black/60" data-test="modal" @click.self="close">
    <div class="rounded-xl bg-white">isi</div>
  </div>
</template>
`
	got := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "app/components/Modal.vue", vue), "app/components/Modal.vue", "vue"))
	if f, ok := got["FE-CLICKABLE-NON-INTERACTIVE"]; ok {
		t.Errorf("latar modal @click.self tidak boleh dilaporkan: baris %d", f.Line)
	}
	mixed := `<template>
  <div class="backdrop" @click.self="close" @click.stop="pick">x</div>
</template>
`
	got = ruleSet(InspectFile(writeFixture(t, t.TempDir(), "app/components/Mixed.vue", mixed), "app/components/Mixed.vue", "vue"))
	if _, ok := got["FE-CLICKABLE-NON-INTERACTIVE"]; !ok {
		t.Error("tag yang juga punya handler klik selain .self tetap dilaporkan")
	}
	if onlySelfClicks(`<div @click.prevent.self="x">`) != true || onlySelfClicks(`<div @click="x">`) {
		t.Error("onlySelfClicks salah membaca modifier")
	}
}
