package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test karakterisasi InspectFile: mengunci temuan (urutan, baris, teks) untuk berbagai bahasa/peran berkas.
func TestInspectFileGolden(t *testing.T) {
	secret := "jwt_secret = \"" + "abcdefghijkl1234" + "\""
	bigHandler := strings.Repeat("// filler\n", 410)
	cases := []struct {
		name, rel, lang, body string
	}{
		{"go_handler", "api/user_handler.go", "go", `package api
import "net/http"
` + bigHandler + `
func h() {
	for _, u := range users {
		db.Find(&orders, "user_id = ?", u.ID)
	}
	db.Raw(fmt.Sprintf("SELECT * FROM users WHERE id = %s", id))
	q := "SELECT * FROM orders"
	_ = q
	db.Where("name LIKE '%abc%'")
	_, err := doThing()
	_ = err
	c := &http.Client{}
	` + secret + `
	exec.Command("sh", "-c", "ping " + host)
	os.ReadFile(base + req.Query("x"))
	b, _ := io.ReadAll(r.Body)
	rows, err := db.Query("x")
	tx := db.Begin()
	app.Post("/users", handler)
	app.Post("/login", handler)
	app.Post("/items", authMiddleware, handler)
	app.Get("/getUser/x", handler)
	api := app.Group("/api", AuthRequired)
	api.Post("/safe", handler)
	app.Use(cors.New(cors.Config{AllowOrigins: "*"}))
	// TODO: implement this
	panic("not implemented")
}
`},
		{"ts_handler", "src/controller/x.ts", "typescript", `import fs from 'fs'
const a = fs.readFileSync('x')
try { x() } catch (e) {}
items.forEach(i => {
  prisma.user.find(i)
})
/* TODO: implement later */
app.delete('/things', h)
`},
		{"vue_ui", "components/Foo.vue", "vue", `<template><div v-html="raw"></div><div v-html="DOMPurify.sanitize(x)"></div></template>
<script setup>
alert('x')
window.confirm('y')
el.innerHTML = z
try { a() } catch (e) {}
</script>
`},
		{"vue_tab", "components/detail/InfoTab.vue", "vue", `<script setup>
onMounted(() => { loadData() })
getProjectBaseKnowledge()
</script>
`},
		{"py", "app/views.py", "python", `def f():
    pass
    # TODO: implement
    try:
        x()
    except Exception: pass
    subprocess.run("a" + b, shell=True)
    cursor.execute(f"SELECT * FROM t WHERE a={x}")
`},
		{"kotlin_android", "app/src/main/java/A.kt", "kotlin", `import android.os.Bundle
class A {
  fun f() {
    try { x() } catch (e: Exception) {}
    db.rawQuery("SELECT * FROM t", null)
    Runtime.getRuntime().exec("a" + b)
  }
}
`},
		{"dart_flutter", "lib/main.dart", "dart", `import 'package:flutter/material.dart';
class W extends StatefulWidget {
  final c = TextEditingController();
  Widget build(BuildContext context) {
    try { x(); } catch (e) {}
    print('x');
    return Container();
  }
}
`},
		{"php", "index.php", "php", `<?php
# comment
$db->query("SELECT * FROM t WHERE x" . $y);
shell_exec("a" . $b);
`},
		{"xml", "res/layout/a.xml", "xml", `<LinearLayout xmlns:android="x"><TextView android:layout_width="wrap_content"/></LinearLayout>
`},
	}
	out := map[string]any{}
	dir := t.TempDir()
	for _, c := range cases {
		p := filepath.Join(dir, strings.ReplaceAll(c.rel, "/", "_"))
		if err := os.WriteFile(p, []byte(c.body), 0o644); err != nil {
			t.Fatal(err)
		}
		out[c.name] = InspectFile(p, c.rel, c.lang)
	}
	// berkas tak ada
	out["missing"] = InspectFile(filepath.Join(dir, "none.go"), "none.go", "go")
	assertGolden(t, "inspect_file", out)
}
