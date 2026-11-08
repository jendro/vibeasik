package diagnostics

import (
	"fmt"
	"strings"
	"testing"
)

func repeatLines(line string, n int) string {
	return strings.Repeat(line+"\n", n)
}

// Test karakterisasi aturan UI/perf: mengunci keluaran inspectUIPerformance, inspectFileSize,
// InspectAssetFile, dan xmlMaxNestingDepth.
func TestUIPerfGolden(t *testing.T) {
	vue := `<template>
  <div @click="go" class="x">a</div>
  <li v-for="i in items" v-if="i.ok">{{ i }}</li>
  <li
    v-for="j in more"
    :key="j.id"
  >{{ j }}</li>
  <img src="a.png">
  <img src="b.png" class="w-4 h-4" alt="b" loading="lazy">
  <p class="text-[9px]" style="font-size: 10px; color: #fff">x</p>
  <p style="width: 1024px">y</p>
  <p class="w-[800px] z-[9999]">z</p>
  <span @click="a" role="button" tabindex="0">k</span>
  <!-- komentar -->
</template>
<script setup>
import _ from 'lodash'
window.addEventListener('resize', f)
setInterval(f, 10)
console.log('a')
console.log('b')
console.log('c')
const img = "data:image/png;base64,` + strings.Repeat("A", 4100) + `"
</script>
<style>
.a { z-index: 99999; color: #aabbcc !important; font-size: 8px; }
` + repeatLines(`.b { color: #123456 !important; }`, 12) + repeatLines(`<i style="a">`, 8) + `</style>
`
	tsx := `import moment from 'moment'
export const A = () => (
  <div onClick={go}>
    <img src="a.png" />
  </div>
)
console.debug('x')
console.log('y')
console.log('z')
document.addEventListener('x', f, { once: true })
`
	kotlinAndroid := `import android.os.Bundle
class MainActivity : AppCompatActivity() {
    companion object {
        var ctx: Context? = null
    }
    fun f() {
        runBlocking { }
        GlobalScope.launch { }
        adapter.notifyDataSetChanged()
        val b = BitmapFactory.decodeFile(p)
    }
}
@Composable
fun Screen() {
    val s = mutableStateOf(1)
    val t = remember { mutableStateOf(2) }
    LazyColumn {
        items(list) { }
        items(list, key = { it.id }) { }
    }
}
`
	javaAndroid := `import android.content.Context;
public class A extends Activity {
  static Context c;
  void f() { Thread.sleep(10); }
}
`
	kotlinServer := `package srv
fun f() { runBlocking { } }
`
	xmlLayout := `<?xml version="1.0"?>
<!-- c -->
<LinearLayout xmlns:android="x"
    android:layout_width="match_parent">
  <ImageView android:layout_width="24dp" android:src="#fff" />
  <ImageView android:contentDescription="@string/a" android:importantForAccessibility="no" />
  <Button android:layout_width="30dp" android:layout_height="30dp" android:text="Click" android:textSize="12sp"/>
  <TextView android:onClick="go" android:layout_width="40dp" android:minHeight="48dp" android:textSize="12dp"/>
  <ListView android:id="@+id/l"/>
  <View android:background="#111"/>
  <View android:background="#222"/>
  <View android:background="#333"/>
  <TextView android:textColor="#111" android:background="#222" android:tint="#333" android:backgroundTint="#444" android:hint="x"/>
  <com.foo.AppCompatImageView
      android:layout_width="10dp"
      android:layout_height="10dp"/>
</LinearLayout>
`
	deep := ""
	for i := 0; i < 12; i++ {
		deep += fmt.Sprintf("<L%d>\n", i)
	}
	for i := 11; i >= 0; i-- {
		deep += fmt.Sprintf("</L%d>\n", i)
	}
	dart := `import 'package:flutter/material.dart';
Widget build() {
  // ListView(
  return ListView(children: [
    Image.network(u),
    Opacity(opacity: 0.5, child: x),
    ListView(shrinkWrap: true),
  ]);
}
`
	dartCached := dart + "final c = CachedNetworkImage();\n"

	out := map[string]any{}
	out["vue"] = inspectUIPerformance(vue, "components/A.vue", "vue", strings.Split(vue, "\n"))
	out["tsx"] = inspectUIPerformance(tsx, "src/components/A.tsx", "typescript", strings.Split(tsx, "\n"))
	out["ts_nonui"] = inspectUIPerformance(tsx, "server/a.ts", "typescript", strings.Split(tsx, "\n"))
	out["kotlin"] = inspectUIPerformance(kotlinAndroid, "app/src/main/java/MainActivity.kt", "kotlin", strings.Split(kotlinAndroid, "\n"))
	out["java"] = inspectUIPerformance(javaAndroid, "app/src/main/java/A.java", "java", strings.Split(javaAndroid, "\n"))
	out["kotlin_server"] = inspectUIPerformance(kotlinServer, "srv/a.kt", "kotlin", strings.Split(kotlinServer, "\n"))
	out["xml"] = inspectUIPerformance(xmlLayout, "app/src/main/res/layout/a.xml", "xml", strings.Split(xmlLayout, "\n"))
	out["xml_deep"] = inspectUIPerformance(deep, "res/layout/deep.xml", "xml", strings.Split(deep, "\n"))
	out["dart"] = inspectUIPerformance(dart, "lib/screens/a_screen.dart", "dart", strings.Split(dart, "\n"))
	out["dart_cached"] = inspectUIPerformance(dartCached, "lib/a.dart", "dart", strings.Split(dartCached, "\n"))

	// ukuran berkas
	sizes := map[string]any{}
	mk := func(n int) []string { return strings.Split(repeatLines("x", n), "\n") }
	for _, c := range []struct {
		name, rel, lang string
		n               int
		android         bool
	}{
		{"vue_small", "pages/a.vue", "vue", 500, false},
		{"vue_warn", "pages/a.vue", "vue", 700, false},
		{"vue_crit", "pages/a.vue", "vue", 1100, false},
		{"android_screen", "app/MainActivity.kt", "kotlin", 700, true},
		{"android_other", "app/Util.kt", "kotlin", 700, true},
		{"flutter_widget", "lib/screens/x.dart", "dart", 700, false},
		{"flutter_suffix", "lib/x_page.dart", "dart", 1100, false},
		{"dart_plain", "lib/x.dart", "dart", 1100, false},
		{"xml_big", "res/layout/a.xml", "xml", 400, false},
		{"xml_small", "res/layout/a.xml", "xml", 200, false},
		{"ts_module", "src/composables/useA.ts", "typescript", 900, false},
		{"ts_module_server", "src/server/useA.ts", "typescript", 900, false},
		{"go_big", "a.go", "go", 2000, false},
	} {
		sizes[c.name] = inspectFileSize(c.rel, c.lang, mk(c.n), c.android)
	}
	out["sizes"] = sizes

	// aset
	assets := map[string]any{}
	for _, c := range []struct {
		rel  string
		size int64
	}{
		{"a.txt", 1 << 30},
		{"img/a.png", 100},
		{"img/a.png", 600 * 1024},
		{"img/a.png", 2 * 1024 * 1024},
		{"img/a.gif", 600 * 1024},
		{"v/a.mp4", 6 * 1024 * 1024},
		{"v/a.mp4", 30 * 1024 * 1024},
		{"v/a.mp4", 1024},
		{"i/a.svg", 300 * 1024},
		{"i/a.svg", 2 * 1024 * 1024},
		{"app/src/main/res/drawable/a.png", 250 * 1024},
		{"app/src/main/res/mipmap-xhdpi/a.png", 4 * 1024 * 1024},
		{"android/assets/a.webp", 700 * 1024},
		{"build/flutter_assets/a.jpg", 700 * 1024},
	} {
		assets[fmt.Sprintf("%s@%d", c.rel, c.size)] = InspectAssetFile(c.rel, c.size)
	}
	out["assets"] = assets

	depth := map[string]any{}
	for name, body := range map[string]string{
		"deep":      deep,
		"selfclose": "<A>\n<B/>\n<C\n a=\"1\"\n/>\n</A>\n",
		"comment":   "<!-- <X> <Y> -->\n<A>\n<!-- multi\n <Z>\n-->\n<B></B></A>\n<?xml v?>\n<!DOCTYPE x>\n",
		"empty":     "",
	} {
		d, l := xmlMaxNestingDepth(strings.Split(body, "\n"))
		depth[name] = []int{d, l}
	}
	out["depth"] = depth
	assertGolden(t, "ui_perf", out)
}
