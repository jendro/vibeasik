package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

func ruleSet(findings []models.Finding) map[string]models.Finding {
	m := make(map[string]models.Finding)
	for _, f := range findings {
		m[f.RuleID] = f
	}
	return m
}

func writeFixture(t *testing.T, dir, rel, content string) string {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return full
}

func TestInspectUIPerformanceVue(t *testing.T) {
	vue := `<template>
  <div>
    <li v-for="item in items" v-if="item.ok">{{ item.name }}</li>
    <li
      v-for="row in rows"
      :key="row.id"
      @click="() => select(row)"
    >ok</li>
    <img :src="avatar">
    <img src="/a.png" width="10" height="10" alt="a" loading="lazy">
    <div class="w-[1200px] text-[9px]" @click="open">x</div>
    <button class="text-xs" @click="open">ok</button>
  </div>
</template>

<script setup lang="ts">
const help = 'contoh <img onerror=alert(1)> di string'
onMounted(() => window.addEventListener('resize', onResize))
</script>
`
	got := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "app/pages/demo.vue", vue), "app/pages/demo.vue", "vue"))

	for _, id := range []string{"FE-VFOR-MISSING-KEY", "FE-VIF-WITH-VFOR", "FE-IMG-LAYOUT-SHIFT", "FE-IMG-MISSING-ALT", "FE-IMG-NO-LAZY", "FE-FIXED-WIDTH-LAYOUT", "FE-SMALL-FONT", "FE-CLICKABLE-NON-INTERACTIVE", "FE-LISTENER-LEAK"} {
		if _, ok := got[id]; !ok {
			t.Errorf("expected %s to be reported", id)
		}
	}
	if f := got["FE-VFOR-MISSING-KEY"]; f.Line != 3 || !strings.Contains(f.Description, "1 elemen") {
		t.Errorf("v-for multi-line with :key must not be flagged, got line %d: %s", f.Line, f.Description)
	}
	if f := got["FE-IMG-MISSING-ALT"]; f.Line != 9 {
		t.Errorf("<img> inside <script> string must be ignored, got line %d", f.Line)
	}
}

func TestInspectFileSizeThresholds(t *testing.T) {
	big := "<template>\n" + strings.Repeat("<div>row</div>\n", uiComponentCriticalLines+5) + "</template>\n"
	got := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "components/Huge.vue", big), "components/Huge.vue", "vue"))
	if f, ok := got["FE-LARGE-COMPONENT"]; !ok || f.Severity != "CRITICAL" {
		t.Fatalf("expected CRITICAL FE-LARGE-COMPONENT, got %+v", f)
	}

	small := "<template>\n<div>ok</div>\n</template>\n"
	if _, ok := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "components/Small.vue", small), "components/Small.vue", "vue"))["FE-LARGE-COMPONENT"]; ok {
		t.Errorf("small component must not be flagged")
	}
}

func TestInspectAndroidKotlin(t *testing.T) {
	kt := `package demo

class MainActivity : AppCompatActivity() {
    companion object {
        lateinit var instance: Activity
        var current: Context? = null
    }

    override fun onCreate(b: Bundle?) {
        runBlocking { repo.load() }
        GlobalScope.launch { sync() }
        adapter.notifyDataSetChanged()
        val bmp = BitmapFactory.decodeFile(path)
        try { x() } catch (e: Exception) {}
    }
}

@Composable
fun Screen(users: List<User>) {
    var query = mutableStateOf("")
    val ok by remember { mutableStateOf(1) }
    LazyColumn { items(users) { UserRow(it) } }
}
`
	got := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "app/src/main/java/MainActivity.kt", kt), "app/src/main/java/MainActivity.kt", "kotlin"))
	for _, id := range []string{"AND-MAIN-THREAD-BLOCKING", "AND-GLOBALSCOPE-LEAK", "AND-NOTIFY-DATASET-CHANGED", "AND-STATIC-CONTEXT-LEAK", "AND-BITMAP-DECODE-UI", "AND-COMPOSE-STATE-NO-REMEMBER", "AND-COMPOSE-LAZY-NO-KEY", "SMELL-SWALLOWED-ERROR"} {
		if _, ok := got[id]; !ok {
			t.Errorf("expected %s to be reported", id)
		}
	}
	if f := got["AND-COMPOSE-STATE-NO-REMEMBER"]; !strings.Contains(f.Description, "1 kali") {
		t.Errorf("remembered state must not be counted: %s", f.Description)
	}
}

func TestInspectAndroidLayoutXML(t *testing.T) {
	var nested strings.Builder
	nested.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	for i := 0; i < 12; i++ {
		nested.WriteString(`<LinearLayout android:layout_width="match_parent" android:layout_height="wrap_content">` + "\n")
	}
	nested.WriteString(`<TextView android:text="Halo" android:textSize="14dp" android:textColor="#FF0000"/>` + "\n")
	nested.WriteString(`<ImageButton
    android:layout_width="32dp"
    android:layout_height="32dp"
    android:src="@drawable/ic_close"/>` + "\n")
	nested.WriteString(`<ListView android:layout_width="match_parent" android:layout_height="match_parent"/>` + "\n")
	for i := 0; i < 12; i++ {
		nested.WriteString("</LinearLayout>\n")
	}

	dir := t.TempDir()
	rel := "app/src/main/res/layout/activity_main.xml"
	got := ruleSet(InspectFile(writeFixture(t, dir, rel, nested.String()), rel, "xml"))
	for _, id := range []string{"AND-DEEP-LAYOUT", "AND-TEXTSIZE-NOT-SP", "AND-HARDCODED-TEXT", "AND-IMAGE-NO-CONTENTDESC", "AND-SMALL-TOUCH-TARGET", "AND-LEGACY-LISTVIEW"} {
		if _, ok := got[id]; !ok {
			t.Errorf("expected %s to be reported", id)
		}
	}

	// XML non-layout (values) tidak boleh ikut dipindai lewat InspectDirectory
	writeFixture(t, dir, "app/src/main/res/values/strings.xml", `<resources><string name="a">Halo</string></resources>`)
	for _, f := range InspectDirectory(dir, "") {
		if strings.Contains(f.TargetFile, "values") {
			t.Errorf("values XML must be skipped, got %s", f.RuleID)
		}
	}
}

func TestInspectFlutterWidgets(t *testing.T) {
	dart := `Widget build(BuildContext context) {
  return ListView(
    shrinkWrap: true,
    children: [Opacity(opacity: 0.5, child: Image.network(url))],
  );
}
`
	got := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "lib/screens/home_screen.dart", dart), "lib/screens/home_screen.dart", "dart"))
	for _, id := range []string{"MOB-LISTVIEW-NON-BUILDER", "MOB-SHRINKWRAP-LIST", "MOB-IMAGE-NETWORK-NO-CACHE", "MOB-OPACITY-WIDGET"} {
		if _, ok := got[id]; !ok {
			t.Errorf("expected %s to be reported", id)
		}
	}
}

func TestInspectAssetFile(t *testing.T) {
	if f := InspectAssetFile("public/images/hero.png", 2*1024*1024); f == nil || f.Severity != "CRITICAL" || f.Category != "frontend" {
		t.Errorf("expected CRITICAL frontend asset finding, got %+v", f)
	}
	if f := InspectAssetFile("app/src/main/res/drawable/bg.png", 300*1024); f == nil || f.Category != "mobile" {
		t.Errorf("expected mobile drawable finding, got %+v", f)
	}
	if f := InspectAssetFile("public/icon.png", 20*1024); f != nil {
		t.Errorf("small asset must not be flagged")
	}
}

// Gambar di dalam overlay kondisional (UModal, layer v-if fixed inset-0, komponen *Lightbox) baru dirender
// saat dibuka dan langsung terlihat, jadi FE-IMG-NO-LAZY tidak boleh muncul; gambar biasa tetap ditandai.
func TestInspectLazyImageSkipsOverlays(t *testing.T) {
	overlay := `<template>
  <div>
    <img :src="cover" alt="c" width="10" height="10">
    <UModal :open="open">
      <template #content>
        <img :src="full" alt="f" class="max-w-full h-auto">
      </template>
    </UModal>
    <div
      v-if="preview"
      class="fixed inset-0 z-50"
    >
      <img :src="preview.src" :alt="preview.name" class="w-full">
    </div>
  </div>
</template>
`
	got := InspectFile(writeFixture(t, t.TempDir(), "app/pages/gal.vue", overlay), "app/pages/gal.vue", "vue")
	f, ok := ruleSet(got)["FE-IMG-NO-LAZY"]
	if !ok || f.Line != 3 || !strings.Contains(f.Description, "1 tag") {
		t.Fatalf("only the inline image (line 3) must be flagged, got %+v", f)
	}

	box := "<template>\n  <img :src=\"u\" alt=\"x\" width=\"1\" height=\"1\">\n</template>\n"
	if _, ok := ruleSet(InspectFile(writeFixture(t, t.TempDir(), "components/ShotLightbox.vue", box), "components/ShotLightbox.vue", "vue"))["FE-IMG-NO-LAZY"]; ok {
		t.Fatal("images in a *Lightbox component must not be flagged")
	}
}
