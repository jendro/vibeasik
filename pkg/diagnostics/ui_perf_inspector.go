package diagnostics

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/util"
)

// Ambang batas ukuran berkas UI (baris kode)
const (
	uiComponentWarnLines     = 600
	uiComponentCriticalLines = 1000
	scriptModuleWarnLines    = 800
	androidLayoutWarnLines   = 300
	androidLayoutMaxDepth    = 10
)

// Ambang batas ukuran aset statis (byte)
const (
	webImageWarnBytes     = 500 * 1024
	webImageCriticalBytes = 1536 * 1024
	androidImageWarnBytes = 200 * 1024
	svgWarnBytes          = 200 * 1024
	videoWarnBytes        = 5 * 1024 * 1024
)

// Ekstensi aset media yang diperiksa ukurannya
var auditAssetExtensions = map[string]string{
	".png":  "image",
	".jpg":  "image",
	".jpeg": "image",
	".gif":  "image",
	".bmp":  "image",
	".webp": "image",
	".avif": "image",
	".svg":  "svg",
	".mp4":  "video",
	".webm": "video",
	".mov":  "video",
}

var (
	// Frontend (Vue / React / Svelte)
	feImgTagRegex         = regexp.MustCompile(`<img\b`)
	feImgWidthClassRegex  = regexp.MustCompile(`\b(w-|size-|aspect-)`)
	feImgHeightClassRegex = regexp.MustCompile(`\b(h-|size-|aspect-)`)
	feClickableDivRegex   = regexp.MustCompile(`<(div|span|li|td)\b[^>]*(@click|v-on:click|onClick=)`)
	feSmallFontTwRegex    = regexp.MustCompile(`text-\[(\d+(?:\.\d+)?)px\]`)
	feSmallFontCssRegex   = regexp.MustCompile(`font-size\s*:\s*(\d+(?:\.\d+)?)px`)
	feFixedWidthCssRegex  = regexp.MustCompile(`(?:^|[^a-z-])(?:min-)?width\s*:\s*(\d{3,4})px`)
	feFixedWidthTwRegex   = regexp.MustCompile(`(?:^|[\s"'])(?:min-)?w-\[(\d{3,4})px\]`)
	feStaticStyleRegex    = regexp.MustCompile(`(?:^|\s)style="`)
	feHexColorRegex       = regexp.MustCompile(`(?:[:\s"'(\[]|^)#(?:[0-9a-fA-F]{6}|[0-9a-fA-F]{3})(?:[0-9a-fA-F]{2})?\b`)
	feHeavyImportRegex    = regexp.MustCompile(`(?:import\s+[A-Za-z_$][\w$]*\s+from\s+['"](lodash|moment)['"]|require\(\s*['"](lodash|moment)['"]\s*\)|import\s+\*\s+as\s+\w+\s+from\s+['"](lodash|lucide-react|@mui/icons-material|@mui/material|date-fns|echarts|rxjs|antd)['"])`)
	feBase64AssetRegex    = regexp.MustCompile(`data:(?:image|font|application)/[a-zA-Z0-9.+-]+;base64,[A-Za-z0-9+/=]+`)
	feConsoleLogRegex     = regexp.MustCompile(`\bconsole\.(log|debug)\s*\(`)
	feZIndexRegex         = regexp.MustCompile(`(?:z-index\s*:\s*|z-\[)(\d{4,})`)
	feImportantRegex      = regexp.MustCompile(`!important`)
	feRouterStaticImport  = regexp.MustCompile(`(?m)^\s*import\s+\w+\s+from\s+['"][^'"]*(pages|views|screens)/[^'"]+['"]`)
	feRouterLazyRegex     = regexp.MustCompile(`(\(\)\s*=>\s*import\(|\blazy\(|defineAsyncComponent|loadable\()`)

	// Android (Kotlin / Java)
	andMainThreadBlockRegex = regexp.MustCompile(`(\brunBlocking\s*[{(]|\bThread\.sleep\s*\(|\.allowMainThreadQueries\(\))`)
	andGlobalScopeRegex     = regexp.MustCompile(`\bGlobalScope\.(launch|async)\b`)
	andNotifyAllRegex       = regexp.MustCompile(`\.notifyDataSetChanged\(\)`)
	andStaticContextJava    = regexp.MustCompile(`\bstatic\s+(?:final\s+)?(?:Context|Activity|View|Fragment|AppCompatActivity)\s+\w+`)
	andContextVarKotlin     = regexp.MustCompile(`\bvar\s+\w+\s*:\s*(?:Context|Activity|View|Fragment|AppCompatActivity)\??`)
	andMutableStateRegex    = regexp.MustCompile(`\bmutableState(?:Of|ListOf|MapOf)\s*\(`)
	andLazyItemsRegex       = regexp.MustCompile(`\bitems(?:Indexed)?\s*\(`)
	andBitmapDecodeRegex    = regexp.MustCompile(`\bBitmapFactory\.decode(File|Resource|Stream|ByteArray)\s*\(`)

	// Android XML layout
	xmlOpenTagRegex        = regexp.MustCompile(`<([A-Za-z][\w.]*)\b`)
	xmlTextSizeDpRegex     = regexp.MustCompile(`android:textSize="\d+(?:\.\d+)?(dp|px|dip)"`)
	xmlHardcodedTextRegex  = regexp.MustCompile(`android:(text|hint|contentDescription)="[^@?"][^"]*"`)
	xmlHardcodedColorRegex = regexp.MustCompile(`android:(textColor|background|tint|backgroundTint|src)="#[0-9a-fA-F]{3,8}"`)
	xmlDimensionDpRegex    = regexp.MustCompile(`android:layout_(width|height)="(\d+)dp"`)
	xmlLegacyListRegex     = regexp.MustCompile(`<(ListView|GridView)\b`)

	// Flutter (Dart)
	dartListNonBuilderRegex = regexp.MustCompile(`\b(ListView|GridView)\(`)
	dartShrinkWrapRegex     = regexp.MustCompile(`shrinkWrap\s*:\s*true`)
	dartImageNetworkRegex   = regexp.MustCompile(`\bImage\.network\s*\(`)
	dartOpacityRegex        = regexp.MustCompile(`(?:^|[^\w.])Opacity\s*\(`)
)

// uiIssue mengumpulkan beberapa lokasi pelanggaran aturan yang sama dalam satu berkas,
// agar laporan menampilkan satu temuan ringkas per berkas (bukan ratusan temuan per baris).
type uiIssue struct {
	firstLine int
	snippet   string
	count     int
}

type uiIssueSet struct {
	lines  []string
	issues map[string]*uiIssue
}

func newUIIssueSet(lines []string) *uiIssueSet {
	return &uiIssueSet{lines: lines, issues: make(map[string]*uiIssue)}
}

func (s *uiIssueSet) hit(ruleID string, lineIdx int) {
	if it, ok := s.issues[ruleID]; ok {
		it.count++
		return
	}
	snippet := ""
	if lineIdx >= 0 && lineIdx < len(s.lines) {
		snippet = strings.TrimSpace(s.lines[lineIdx])
		if len(snippet) > 220 {
			snippet = snippet[:220] + "…"
		}
	}
	s.issues[ruleID] = &uiIssue{firstLine: lineIdx + 1, snippet: snippet, count: 1}
}

func (s *uiIssueSet) get(ruleID string) *uiIssue {
	return s.issues[ruleID]
}

// uiRule mendeskripsikan temuan statis UI/performa
type uiRule struct {
	Category     string
	Severity     string
	Title        string
	Description  string // boleh memuat %s (nama berkas) diikuti %d (jumlah lokasi) dan %d (baris pertama)
	Suggestion   string
	TheoryBasis  string
	SuggestedFix string
}

var uiRules = map[string]uiRule{
	"FE-VFOR-MISSING-KEY": {
		Category:     "frontend",
		Severity:     "WARNING",
		Title:        "Rendering List Tanpa :key (v-for Missing Key)",
		Description:  "Berkas '%s' memiliki %d elemen v-for tanpa atribut :key (pertama pada baris %d). Vue tidak dapat melacak identitas item sehingga DOM dirender ulang penuh dan state input/animasi bisa tertukar.",
		Suggestion:   "Tambahkan :key unik dan stabil (ID entitas, bukan index array) pada setiap elemen v-for.",
		TheoryBasis:  "Vue.js Official Style Guide — Priority A (Essential): Keyed v-for & Virtual DOM Reconciliation.",
		SuggestedFix: "<li v-for=\"item in items\" :key=\"item.id\">{{ item.name }}</li>",
	},
	"FE-VIF-WITH-VFOR": {
		Category:     "frontend",
		Severity:     "WARNING",
		Title:        "v-if dan v-for pada Elemen yang Sama",
		Description:  "Berkas '%s' memiliki %d elemen yang memakai v-if bersamaan dengan v-for (pertama pada baris %d). Kondisi dievaluasi untuk setiap item sehingga render lebih lambat dan prioritas direktif membingungkan.",
		Suggestion:   "Saring data lewat computed property terlebih dahulu, atau pindahkan v-if ke elemen <template> pembungkus.",
		TheoryBasis:  "Vue.js Official Style Guide — Priority A (Essential): Avoid v-if with v-for.",
		SuggestedFix: "const activeUsers = computed(() => users.value.filter(u => u.isActive))\n// <li v-for=\"user in activeUsers\" :key=\"user.id\">",
	},
	"FE-IMG-LAYOUT-SHIFT": {
		Category:     "frontend",
		Severity:     "WARNING",
		Title:        "Gambar Tanpa Dimensi Eksplisit (Cumulative Layout Shift)",
		Description:  "Berkas '%s' memiliki %d tag <img> tanpa width/height atau kelas ukuran/aspect-ratio (pertama pada baris %d). Saat gambar selesai dimuat, konten di bawahnya melompat (CLS buruk).",
		Suggestion:   "Berikan atribut width & height, atau kelas ukuran tetap / aspect-ratio (misal: class=\"w-16 h-16\" atau \"aspect-video w-full\").",
		TheoryBasis:  "Google Core Web Vitals — Cumulative Layout Shift (CLS) target < 0.1.",
		SuggestedFix: "<img :src=\"avatar\" width=\"64\" height=\"64\" alt=\"Foto profil\" loading=\"lazy\" />",
	},
	"FE-IMG-MISSING-ALT": {
		Category:     "frontend",
		Severity:     "WARNING",
		Title:        "Gambar Tanpa Teks Alternatif (alt)",
		Description:  "Berkas '%s' memiliki %d tag <img> tanpa atribut alt (pertama pada baris %d). Pembaca layar tidak dapat mendeskripsikan gambar dan SEO terdampak.",
		Suggestion:   "Tambahkan alt deskriptif; gunakan alt=\"\" untuk gambar dekoratif murni.",
		TheoryBasis:  "WCAG 2.2 Success Criterion 1.1.1 (Non-text Content) — Level A.",
		SuggestedFix: "<img src=\"/logo.svg\" alt=\"Logo Vibeasik\" width=\"120\" height=\"32\" />",
	},
	"FE-IMG-NO-LAZY": {
		Category:     "frontend",
		Severity:     "INFO",
		Title:        "Gambar Tanpa Lazy Loading",
		Description:  "Berkas '%s' memiliki %d tag <img> tanpa loading=\"lazy\" (pertama pada baris %d). Gambar di luar viewport ikut diunduh saat halaman pertama dibuka sehingga memperlambat LCP.",
		Suggestion:   "Tambahkan loading=\"lazy\" untuk gambar di bawah lipatan layar, dan decoding=\"async\" untuk gambar besar.",
		TheoryBasis:  "Google Web Fundamentals — Browser-level Image Lazy Loading & Largest Contentful Paint (LCP).",
		SuggestedFix: "<img :src=\"thumb\" loading=\"lazy\" decoding=\"async\" width=\"320\" height=\"180\" alt=\"...\" />",
	},
	"FE-CLICKABLE-NON-INTERACTIVE": {
		Category:     "frontend",
		Severity:     "WARNING",
		Title:        "Elemen Non-Interaktif Diberi Event Klik (Aksesibilitas Keyboard)",
		Description:  "Berkas '%s' memiliki %d elemen <div>/<span> dengan event klik tanpa role & tabindex (pertama pada baris %d). Elemen ini tidak dapat dijangkau keyboard maupun pembaca layar.",
		Suggestion:   "Gunakan elemen <button> asli, atau tambahkan role=\"button\", tabindex=\"0\", dan handler @keydown.enter.",
		TheoryBasis:  "WCAG 2.2 SC 2.1.1 (Keyboard) & SC 4.1.2 (Name, Role, Value) — Level A.",
		SuggestedFix: "<button type=\"button\" class=\"...\" @click=\"open\">Buka</button>",
	},
	"FE-SMALL-FONT": {
		Category:     "frontend",
		Severity:     "INFO",
		Title:        "Ukuran Teks Terlalu Kecil (< 12px)",
		Description:  "Berkas '%s' memakai ukuran font di bawah 12px sebanyak %d kali (pertama pada baris %d). Teks sekecil ini sulit dibaca, terutama pada layar mobile dan pengguna dengan gangguan penglihatan.",
		Suggestion:   "Gunakan skala tipografi minimum 12px (text-xs) untuk label dan 14–16px untuk isi utama; konsistenkan via design token.",
		TheoryBasis:  "Material Design Type Scale (min 12sp caption) & Apple HIG Typography; WCAG 2.2 SC 1.4.4 (Resize Text).",
		SuggestedFix: "<!-- SEBELUM: class=\"text-[9px]\" -->\n<!-- SESUDAH: --> <span class=\"text-xs\">Label</span>",
	},
	"FE-FIXED-WIDTH-LAYOUT": {
		Category:     "frontend",
		Severity:     "WARNING",
		Title:        "Lebar Layout Tetap (Fixed Width ≥ 768px) — Tidak Responsif",
		Description:  "Berkas '%s' menetapkan lebar/min-width tetap ≥ 768px sebanyak %d kali (pertama pada baris %d). Layout akan terpotong atau memunculkan scroll horizontal di tablet/HP.",
		Suggestion:   "Ganti dengan max-width + w-full, grid/flex fluid, atau terapkan hanya pada breakpoint besar (md:/lg:).",
		TheoryBasis:  "Ethan Marcotte — Responsive Web Design (Fluid Grids) & WCAG 2.2 SC 1.4.10 (Reflow, 320 CSS px).",
		SuggestedFix: "<!-- SEBELUM: class=\"w-[1200px]\" -->\n<div class=\"w-full max-w-7xl mx-auto\">",
	},
	"FE-INLINE-STYLE-OVERUSE": {
		Category:     "frontend",
		Severity:     "INFO",
		Title:        "Penggunaan Inline Style Berlebihan",
		Description:  "Berkas '%s' memiliki %d atribut style=\"...\" statis (pertama pada baris %d). Inline style mem-bypass design system, sulit dirawat, dan tidak mendukung dark mode/breakpoint.",
		Suggestion:   "Pindahkan gaya ke kelas utility (Tailwind) atau <style scoped> dengan design token.",
		TheoryBasis:  "Design System Consistency & CSS Maintainability (Atomic/Utility-first CSS).",
		SuggestedFix: "<!-- SEBELUM: style=\"margin-top: 12px; color: #333\" -->\n<div class=\"mt-3 text-neutral-700 dark:text-neutral-300\">",
	},
	"FE-HARDCODED-COLORS": {
		Category:     "frontend",
		Severity:     "INFO",
		Title:        "Warna Hex Hardcoded Berulang (Tanpa Design Token)",
		Description:  "Berkas '%s' memuat %d kode warna hex langsung (pertama pada baris %d). Palet warna menjadi tidak konsisten antar halaman dan dark mode sulit diterapkan.",
		Suggestion:   "Definisikan warna di tema (tailwind.config / app.config / CSS variables) lalu gunakan kelas atau token semantik.",
		TheoryBasis:  "Design Tokens (W3C Design Tokens Community Group) & Atomic Design — Brad Frost.",
		SuggestedFix: ":root { --color-brand: #7c3aed; }\n.title { color: var(--color-brand); }",
	},
	"FE-HEAVY-LIB-IMPORT": {
		Category:     "frontend",
		Severity:     "WARNING",
		Title:        "Import Library Berat Secara Utuh (Bundle Bloat)",
		Description:  "Berkas '%s' mengimpor seluruh isi library berat %d kali (pertama pada baris %d). Tree-shaking gagal sehingga ukuran bundle JS membengkak dan waktu muat halaman melambat.",
		Suggestion:   "Impor fungsi spesifik (import debounce from 'lodash-es/debounce'), ganti moment dengan dayjs/date-fns, dan hindari import * as.",
		TheoryBasis:  "Webpack/Vite Tree Shaking (ES Module Static Analysis) & web.dev JavaScript Budget (< 170KB gzip).",
		SuggestedFix: "// SEBELUM: import _ from 'lodash'\nimport { debounce } from 'lodash-es'",
	},
	"FE-LISTENER-LEAK": {
		Category:     "frontend",
		Severity:     "WARNING",
		Title:        "Event Listener / Timer Tanpa Pembersihan (SPA Memory Leak)",
		Description:  "Berkas '%s' mendaftarkan %d listener/timer global tanpa pasangan removeEventListener/clearInterval (pertama pada baris %d). Setiap navigasi SPA menumpuk listener baru sehingga memori dan CPU terus naik.",
		Suggestion:   "Bersihkan di onBeforeUnmount/onUnmounted (Vue) atau fungsi cleanup useEffect (React), atau gunakan useEventListener dari VueUse.",
		TheoryBasis:  "Vue Composition API Lifecycle & Chrome DevTools Memory Profiling (Detached Listener Leak).",
		SuggestedFix: "onMounted(() => window.addEventListener('resize', onResize))\nonBeforeUnmount(() => window.removeEventListener('resize', onResize))",
	},
	"FE-INLINE-BASE64-ASSET": {
		Category:     "frontend",
		Severity:     "WARNING",
		Title:        "Aset Base64 Besar Tertanam di Kode Sumber",
		Description:  "Berkas '%s' memuat %d data URI base64 berukuran besar (pertama pada baris %d). Base64 menambah ~33%% ukuran, tidak bisa di-cache terpisah, dan memperbesar bundle JS/CSS.",
		Suggestion:   "Simpan aset sebagai file terpisah di public/ atau assets/ dan rujuk melalui URL.",
		TheoryBasis:  "HTTP Caching (RFC 9111) & web.dev — Avoid Large Inlined Assets.",
		SuggestedFix: "<img src=\"/images/hero.webp\" width=\"1200\" height=\"600\" alt=\"...\" />",
	},
	"FE-CONSOLE-LOG-LEFTOVER": {
		Category:     "frontend",
		Severity:     "INFO",
		Title:        "Sisa console.log Debugging di Kode Produksi",
		Description:  "Berkas '%s' memuat %d pemanggilan console.log/debug (pertama pada baris %d). Log berlebih memperlambat render loop, membocorkan data internal ke DevTools, dan mengotori konsol.",
		Suggestion:   "Hapus log debugging atau gunakan logger dengan level yang dimatikan pada build produksi.",
		TheoryBasis:  "Clean Code — Boy Scout Rule & OWASP Logging Cheat Sheet (Sensitive Data Exposure).",
		SuggestedFix: "if (import.meta.dev) console.debug('[debug]', payload)",
	},
	"FE-ZINDEX-ESCALATION": {
		Category:     "frontend",
		Severity:     "INFO",
		Title:        "Eskalasi z-index Ekstrem (≥ 1000)",
		Description:  "Berkas '%s' memakai z-index ≥ 1000 sebanyak %d kali (pertama pada baris %d). Ini tanda perang tumpukan layer yang membuat modal, dropdown, dan toast saling menutupi.",
		Suggestion:   "Definisikan skala z-index terpusat (dropdown 10, sticky 20, modal 50, toast 60) dan gunakan token tersebut.",
		TheoryBasis:  "CSS Stacking Context (CSS2.1 Appendix E) & Design System Layering Scale.",
		SuggestedFix: ":root { --z-dropdown: 10; --z-modal: 50; --z-toast: 60; }",
	},
	"FE-IMPORTANT-OVERUSE": {
		Category:     "frontend",
		Severity:     "INFO",
		Title:        "Penggunaan !important Berlebihan",
		Description:  "Berkas '%s' memakai !important sebanyak %d kali (pertama pada baris %d). Spesifisitas CSS menjadi kacau dan tema/override sulit dilakukan.",
		Suggestion:   "Perbaiki urutan/spesifisitas selector atau gunakan utility class; batasi !important hanya untuk utility override.",
		TheoryBasis:  "CSS Cascade & Specificity (W3C CSS Cascading Level 4) — ITCSS Architecture.",
		SuggestedFix: "/* Hindari: .btn { color: red !important } */\n.card .btn-danger { color: var(--color-danger); }",
	},

	// ===== Android (Kotlin / Java) =====
	"AND-MAIN-THREAD-BLOCKING": {
		Category:     "mobile",
		Severity:     "CRITICAL",
		Title:        "Operasi Blocking di Main/UI Thread (Risiko ANR)",
		Description:  "Berkas '%s' memuat %d pemanggilan blocking (runBlocking / Thread.sleep / allowMainThreadQueries) (pertama pada baris %d). UI membeku dan Android menampilkan dialog ANR jika thread utama tertahan > 5 detik.",
		Suggestion:   "Gunakan coroutine dengan Dispatchers.IO di viewModelScope/lifecycleScope, dan hapus allowMainThreadQueries() dari Room builder.",
		TheoryBasis:  "Android Developers — Keep Your App Responsive (ANR) & Kotlin Coroutines Main-Safety.",
		SuggestedFix: "viewModelScope.launch {\n    val data = withContext(Dispatchers.IO) { repository.load() }\n    _state.value = data\n}",
	},
	"AND-GLOBALSCOPE-LEAK": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "GlobalScope Coroutine (Lifecycle Leak)",
		Description:  "Berkas '%s' meluncurkan %d coroutine di GlobalScope (pertama pada baris %d). Coroutine tetap berjalan setelah layar ditutup, menahan referensi Activity dan menguras baterai.",
		Suggestion:   "Gunakan viewModelScope, lifecycleScope, atau CoroutineScope yang dibatalkan sesuai lifecycle.",
		TheoryBasis:  "Kotlin Structured Concurrency (Roman Elizarov) & Android Lifecycle-aware Components.",
		SuggestedFix: "lifecycleScope.launch { /* ... */ }",
	},
	"AND-NOTIFY-DATASET-CHANGED": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "RecyclerView notifyDataSetChanged() (Re-render Seluruh List)",
		Description:  "Berkas '%s' memanggil notifyDataSetChanged() sebanyak %d kali (pertama pada baris %d). Seluruh item di-bind ulang tanpa animasi sehingga scroll patah (jank) pada list panjang.",
		Suggestion:   "Gunakan ListAdapter + DiffUtil.ItemCallback atau notifyItemChanged/Inserted/Removed yang spesifik.",
		TheoryBasis:  "Android Developers — RecyclerView Performance & DiffUtil (Eugene Myers Diff Algorithm).",
		SuggestedFix: "class UserAdapter : ListAdapter<User, VH>(DIFF) { ... }\nadapter.submitList(newUsers)",
	},
	"AND-STATIC-CONTEXT-LEAK": {
		Category:     "mobile",
		Severity:     "CRITICAL",
		Title:        "Context/Activity/View Disimpan di Field Statis (Memory Leak)",
		Description:  "Berkas '%s' menyimpan referensi Context/Activity/View pada field statis/companion object sebanyak %d kali (pertama pada baris %d). Activity tidak pernah di-garbage-collect setelah rotasi layar atau navigasi.",
		Suggestion:   "Simpan applicationContext saja bila perlu, atau gunakan WeakReference / dependency injection (Hilt).",
		TheoryBasis:  "Android Developers — Avoiding Memory Leaks & LeakCanary Detection Patterns.",
		SuggestedFix: "// SEBELUM: companion object { lateinit var activity: Activity }\nclass Repo @Inject constructor(@ApplicationContext private val ctx: Context)",
	},
	"AND-COMPOSE-STATE-NO-REMEMBER": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "Jetpack Compose: mutableStateOf Tanpa remember",
		Description:  "Berkas '%s' membuat state Compose tanpa remember sebanyak %d kali (pertama pada baris %d). State di-reset setiap recomposition sehingga input hilang dan UI berkedip.",
		Suggestion:   "Bungkus dengan remember { mutableStateOf(...) } atau rememberSaveable, atau angkat state ke ViewModel.",
		TheoryBasis:  "Android Developers — State and Jetpack Compose (Recomposition & remember).",
		SuggestedFix: "var query by rememberSaveable { mutableStateOf(\"\") }",
	},
	"AND-COMPOSE-LAZY-NO-KEY": {
		Category:     "mobile",
		Severity:     "INFO",
		Title:        "Jetpack Compose: LazyColumn items() Tanpa key",
		Description:  "Berkas '%s' memanggil items() pada Lazy list tanpa parameter key sebanyak %d kali (pertama pada baris %d). Saat urutan berubah, Compose me-recompose item yang salah dan state per item bisa tertukar.",
		Suggestion:   "Berikan key stabil: items(users, key = { it.id }) { ... }.",
		TheoryBasis:  "Android Developers — Lists and Grids in Compose (Item Keys) & Compose Performance Best Practices.",
		SuggestedFix: "LazyColumn { items(users, key = { it.id }) { user -> UserRow(user) } }",
	},
	"AND-BITMAP-DECODE-UI": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "Decode Bitmap Manual di Layer UI (OOM & Jank)",
		Description:  "Berkas '%s' memanggil BitmapFactory.decode* sebanyak %d kali (pertama pada baris %d) di Activity/Fragment/Composable. Decode tanpa downsampling di main thread memicu OutOfMemoryError dan frame drop.",
		Suggestion:   "Gunakan Coil/Glide yang otomatis melakukan downsampling, caching, dan decode di background thread.",
		TheoryBasis:  "Android Developers — Loading Large Bitmaps Efficiently (inSampleSize).",
		SuggestedFix: "AsyncImage(model = url, contentDescription = null, modifier = Modifier.size(64.dp))",
	},

	// ===== Android XML Layout =====
	"AND-TEXTSIZE-NOT-SP": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "Ukuran Teks Memakai dp/px, Bukan sp",
		Description:  "Layout '%s' menetapkan textSize dalam dp/px sebanyak %d kali (pertama pada baris %d). Teks tidak ikut membesar saat pengguna mengatur ukuran font sistem (aksesibilitas).",
		Suggestion:   "Gunakan satuan sp untuk textSize, atau TextAppearance dari tema.",
		TheoryBasis:  "Android Developers — Support Different Pixel Densities (sp vs dp) & WCAG 2.2 SC 1.4.4.",
		SuggestedFix: "android:textAppearance=\"?attr/textAppearanceBodyMedium\"  <!-- atau android:textSize=\"14sp\" -->",
	},
	"AND-HARDCODED-TEXT": {
		Category:     "mobile",
		Severity:     "INFO",
		Title:        "Teks Hardcoded di Layout (Tidak Memakai @string)",
		Description:  "Layout '%s' memiliki %d teks literal pada android:text/hint (pertama pada baris %d). Aplikasi tidak dapat dilokalisasi dan teks sulit dikelola.",
		Suggestion:   "Pindahkan teks ke res/values/strings.xml dan rujuk via @string/nama.",
		TheoryBasis:  "Android Lint HardcodedText & Android Localization Guidelines.",
		SuggestedFix: "android:text=\"@string/login_button\"",
	},
	"AND-IMAGE-NO-CONTENTDESC": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "ImageView/ImageButton Tanpa contentDescription",
		Description:  "Layout '%s' memiliki %d gambar/ikon tanpa contentDescription (pertama pada baris %d). TalkBack tidak dapat membacakan fungsi tombol ikon.",
		Suggestion:   "Tambahkan android:contentDescription=\"@string/...\", atau importantForAccessibility=\"no\" untuk gambar dekoratif.",
		TheoryBasis:  "Android Accessibility — Content Labels & Lint ContentDescription; WCAG 2.2 SC 1.1.1.",
		SuggestedFix: "android:contentDescription=\"@string/cd_close\"",
	},
	"AND-SMALL-TOUCH-TARGET": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "Area Sentuh Tombol Terlalu Kecil (< 48dp)",
		Description:  "Layout '%s' memiliki %d tombol/kontrol interaktif dengan lebar/tinggi < 48dp (pertama pada baris %d). Tombol sulit ditekan dan sering salah sentuh.",
		Suggestion:   "Pastikan area sentuh minimal 48×48dp (gunakan minWidth/minHeight atau padding, bukan memperkecil ikon).",
		TheoryBasis:  "Material Design Accessibility — Touch Targets 48×48dp & WCAG 2.2 SC 2.5.8 (Target Size).",
		SuggestedFix: "android:minWidth=\"48dp\"\nandroid:minHeight=\"48dp\"",
	},
	"AND-LEGACY-LISTVIEW": {
		Category:     "mobile",
		Severity:     "INFO",
		Title:        "Memakai ListView/GridView Lama (Bukan RecyclerView)",
		Description:  "Layout '%s' memakai ListView/GridView sebanyak %d kali (pertama pada baris %d). Komponen lama tidak mendaur ulang view secara efisien dan tidak mendukung DiffUtil.",
		Suggestion:   "Migrasi ke RecyclerView + ListAdapter, atau LazyColumn bila memakai Jetpack Compose.",
		TheoryBasis:  "Android Developers — Create Dynamic Lists with RecyclerView.",
		SuggestedFix: "<androidx.recyclerview.widget.RecyclerView android:id=\"@+id/list\" ... />",
	},
	"AND-HARDCODED-COLOR-XML": {
		Category:     "mobile",
		Severity:     "INFO",
		Title:        "Warna Hex Hardcoded di Layout (Tanpa @color/Tema)",
		Description:  "Layout '%s' memuat %d warna hex langsung (pertama pada baris %d). Dark theme dan konsistensi brand sulit dijaga.",
		Suggestion:   "Definisikan di res/values/colors.xml atau atribut tema (?attr/colorPrimary) lalu rujuk dari layout.",
		TheoryBasis:  "Material Theming & Android Night Mode (DayNight Theme).",
		SuggestedFix: "android:textColor=\"?attr/colorOnSurface\"",
	},

	// ===== Flutter (Dart) =====
	"MOB-LISTVIEW-NON-BUILDER": {
		Category:     "mobile",
		Severity:     "INFO",
		Title:        "Flutter ListView/GridView Tanpa .builder",
		Description:  "Berkas '%s' membuat ListView/GridView dengan children statis sebanyak %d kali (pertama pada baris %d). Seluruh item dibangun sekaligus meskipun tidak terlihat, berat untuk daftar panjang/dinamis.",
		Suggestion:   "Gunakan ListView.builder / GridView.builder / ListView.separated untuk data dinamis.",
		TheoryBasis:  "Flutter Performance Best Practices — Build Lazily (ListView.builder).",
		SuggestedFix: "ListView.builder(itemCount: items.length, itemBuilder: (ctx, i) => ItemTile(items[i]))",
	},
	"MOB-SHRINKWRAP-LIST": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "Flutter shrinkWrap: true pada Scrollable",
		Description:  "Berkas '%s' memakai shrinkWrap: true sebanyak %d kali (pertama pada baris %d). Semua anak diukur di awal sehingga lazy loading hilang dan scroll menjadi berat.",
		Suggestion:   "Gunakan CustomScrollView + Sliver (SliverList) alih-alih menumpuk ListView ber-shrinkWrap di dalam Column/ScrollView.",
		TheoryBasis:  "Flutter API Docs — ScrollView.shrinkWrap (\"significantly more expensive\") & Slivers.",
		SuggestedFix: "CustomScrollView(slivers: [SliverToBoxAdapter(child: Header()), SliverList.builder(...)])",
	},
	"MOB-IMAGE-NETWORK-NO-CACHE": {
		Category:     "mobile",
		Severity:     "INFO",
		Title:        "Flutter Image.network Tanpa Cache/Resize",
		Description:  "Berkas '%s' memakai Image.network sebanyak %d kali (pertama pada baris %d) tanpa cacheWidth/cacheHeight atau cached_network_image. Gambar resolusi penuh di-decode ke memori dan diunduh ulang.",
		Suggestion:   "Gunakan CachedNetworkImage atau set cacheWidth/cacheHeight sesuai ukuran tampil.",
		TheoryBasis:  "Flutter Performance — Image Decoding Memory (ResizeImage) & HTTP Caching.",
		SuggestedFix: "CachedNetworkImage(imageUrl: url, memCacheWidth: 300)",
	},
	"MOB-OPACITY-WIDGET": {
		Category:     "mobile",
		Severity:     "INFO",
		Title:        "Flutter Widget Opacity (saveLayer Mahal)",
		Description:  "Berkas '%s' memakai widget Opacity sebanyak %d kali (pertama pada baris %d). Opacity memicu saveLayer offscreen yang mahal, terutama bila dianimasikan.",
		Suggestion:   "Gunakan AnimatedOpacity/FadeTransition untuk animasi, atau warna dengan alpha (withOpacity) untuk elemen statis.",
		TheoryBasis:  "Flutter Performance Best Practices — Use Opacity Sparingly (saveLayer).",
		SuggestedFix: "FadeTransition(opacity: animation, child: child)",
	},
}

// inspectUIPerformance menjalankan aturan tampilan & performa untuk FE, Android, dan Flutter.
func inspectUIPerformance(content string, relPath string, lang string, lines []string) []models.Finding {
	findings := make([]models.Finding, 0)
	isAndroid := (lang == "kotlin" || lang == "java") && isAndroidSourceFile(relPath, content)
	findings = append(findings, inspectFileSize(relPath, lang, lines, isAndroid)...)

	set := newUIIssueSet(lines)
	switch lang {
	case "vue", "typescript", "javascript", "svelte":
		collectFrontendUIIssues(set, content, relPath, lang, lines)
	case "kotlin", "java":
		// Kotlin/Java server (Spring, Ktor) tidak memakai aturan Android
		if isAndroid {
			collectAndroidCodeIssues(set, content, relPath, lines)
		}
	case "xml":
		collectAndroidLayoutIssues(set, relPath, lines)
		if depth, line := xmlMaxNestingDepth(lines); depth > androidLayoutMaxDepth {
			findings = append(findings, models.Finding{
				Category:     "mobile",
				Severity:     "WARNING",
				RuleID:       "AND-DEEP-LAYOUT",
				Title:        fmt.Sprintf("Hierarki Layout Terlalu Dalam (%d Level)", depth),
				Description:  fmt.Sprintf("Layout '%s' memiliki kedalaman view %d level (batas Android Lint: %d). Setiap level menambah pass measure/layout sehingga render frame melewati 16ms (jank).", filepath.Base(relPath), depth, androidLayoutMaxDepth),
				Suggestion:   "Ratakan hierarki dengan ConstraintLayout, <merge>, atau pecah ke <include>/ViewStub untuk bagian yang jarang tampil.",
				TargetFile:   relPath,
				Line:         line,
				TheoryBasis:  "Android Lint TooDeepLayout & Android Developers — Optimizing Layout Hierarchies.",
				SuggestedFix: "<androidx.constraintlayout.widget.ConstraintLayout ...>\n    <!-- susun view sejajar dengan constraint, bukan nested LinearLayout -->",
			})
		}
	case "dart":
		collectFlutterIssues(set, content, lines)
	}

	return append(findings, renderIssueSet(set, uiRules, relPath)...)
}

// renderIssueSet mengubah kumpulan pelanggaran per aturan menjadi satu temuan ringkas per aturan.
func renderIssueSet(set *uiIssueSet, rules map[string]uiRule, relPath string) []models.Finding {
	findings := make([]models.Finding, 0, len(set.issues))
	ruleIDs := make([]string, 0, len(set.issues))
	for id := range set.issues {
		ruleIDs = append(ruleIDs, id)
	}
	// Urutan stabil agar laporan konsisten antar pemindaian
	sort.Strings(ruleIDs)

	for _, id := range ruleIDs {
		rule, ok := rules[id]
		if !ok {
			continue
		}
		it := set.get(id)
		findings = append(findings, models.Finding{
			Category:     rule.Category,
			Severity:     rule.Severity,
			RuleID:       id,
			Title:        rule.Title,
			Description:  fmt.Sprintf(rule.Description, filepath.Base(relPath), it.count, it.firstLine),
			Suggestion:   rule.Suggestion,
			TargetFile:   relPath,
			Line:         it.firstLine,
			Snippet:      it.snippet,
			TheoryBasis:  rule.TheoryBasis,
			SuggestedFix: rule.SuggestedFix,
		})
	}
	return findings
}

func isAndroidScreenFile(base string, isAndroid bool) bool {
	return isAndroid && util.ContainsAny(base, "Activity", "Fragment", "Screen", "Adapter", "ViewModel")
}

func isFlutterWidgetFile(lang, lowerPath, base string) bool {
	if lang != "dart" {
		return false
	}
	return util.ContainsAny(lowerPath, "/screens/", "/pages/", "/widgets/", "/views/") ||
		strings.HasSuffix(base, "_screen.dart") || strings.HasSuffix(base, "_page.dart") || strings.HasSuffix(base, "_widget.dart")
}

// uiSizeKind menjelaskan jenis berkas UI besar (aturan, kategori, saran) untuk satu platform.
type uiSizeKind struct {
	ruleID, category, kind, suggestion, fix string
}

var (
	webComponentSize = uiSizeKind{
		"FE-LARGE-COMPONENT", "frontend", "Halaman/Komponen UI",
		"Pecah menjadi sub-komponen per bagian layar (header, filter, list, modal), pindahkan data statis ke file terpisah, dan pindahkan logika ke composable/hook.",
		"// Struktur yang disarankan:\n// pages/report.vue            -> hanya layout & orkestrasi (< 300 baris)\n// components/report/Filter.vue\n// components/report/Table.vue\n// composables/useReport.ts    -> state & pemanggilan API",
	}
	androidScreenSize = uiSizeKind{
		"AND-GOD-ACTIVITY", "mobile", "Activity/Fragment/Screen Android",
		"Pindahkan logika ke ViewModel + UseCase/Repository, pecah UI menjadi Composable/Fragment kecil atau custom View yang dapat dipakai ulang.",
		"// Activity hanya merender state:\nclass OrderActivity : ComponentActivity() {\n    private val vm: OrderViewModel by viewModels()\n    override fun onCreate(b: Bundle?) { super.onCreate(b); setContent { OrderScreen(vm) } }\n}",
	}
	flutterWidgetSize = uiSizeKind{
		"MOB-LARGE-WIDGET-FILE", "mobile", "Screen/Widget Flutter",
		"Ekstrak sub-tree widget menjadi StatelessWidget terpisah (bukan method _buildX), dan pindahkan state ke Bloc/Provider/Riverpod.",
		"class OrderHeader extends StatelessWidget {\n  const OrderHeader({super.key});\n  @override\n  Widget build(BuildContext context) => ...;\n}",
	}
)

func uiComponentSizeFinding(k uiSizeKind, relPath, base string, lines []string) []models.Finding {
	lineCount := len(lines)
	if lineCount <= uiComponentWarnLines {
		return nil
	}
	severity := "WARNING"
	if lineCount > uiComponentCriticalLines {
		severity = "CRITICAL"
	}
	return []models.Finding{{
		Category:     k.category,
		Severity:     severity,
		RuleID:       k.ruleID,
		Title:        fmt.Sprintf("%s Terlalu Besar (%d Baris)", k.kind, lineCount),
		Description:  fmt.Sprintf("Berkas '%s' memiliki %d baris (batas wajar %d, kritis > %d). Berkas UI raksasa memperlambat hot-reload/kompilasi, memperbesar chunk yang dimuat, memicu re-render luas, dan sulit direview.", base, lineCount, uiComponentWarnLines, uiComponentCriticalLines),
		Suggestion:   k.suggestion,
		TargetFile:   relPath,
		Line:         1,
		Snippet:      firstNonEmptyLine(lines),
		TheoryBasis:  "Single Responsibility Principle (SOLID) & Atomic Design (Brad Frost) — Komponen kecil, fokus, dan dapat diuji.",
		SuggestedFix: k.fix,
	}}
}

func xmlLayoutSizeFinding(relPath, base string, lineCount int) []models.Finding {
	if lineCount <= androidLayoutWarnLines {
		return nil
	}
	return []models.Finding{{
		Category:     "mobile",
		Severity:     "WARNING",
		RuleID:       "AND-LARGE-LAYOUT",
		Title:        fmt.Sprintf("Layout XML Terlalu Besar (%d Baris)", lineCount),
		Description:  fmt.Sprintf("Layout '%s' memiliki %d baris (batas wajar %d). Inflate layout besar memperlambat pembukaan layar dan menyulitkan pemeliharaan.", base, lineCount, androidLayoutWarnLines),
		Suggestion:   "Pecah menjadi beberapa layout dengan <include>/<merge>, gunakan ViewStub untuk bagian opsional, atau migrasi ke Jetpack Compose.",
		TargetFile:   relPath,
		Line:         1,
		TheoryBasis:  "Android Developers — Re-using Layouts with <include> & Delayed Loading with ViewStub.",
		SuggestedFix: "<include layout=\"@layout/section_header\" />\n<ViewStub android:id=\"@+id/stub_details\" android:layout=\"@layout/details\" ... />",
	}}
}

func scriptModuleSizeFinding(relPath, base string, lines []string) []models.Finding {
	lineCount := len(lines)
	return []models.Finding{{
		Category:     "frontend",
		Severity:     "WARNING",
		RuleID:       "FE-LARGE-MODULE",
		Title:        fmt.Sprintf("Modul Frontend Terlalu Besar (%d Baris)", lineCount),
		Description:  fmt.Sprintf("Modul '%s' memiliki %d baris (batas wajar %d). Modul gemuk tidak dapat di-code-split dan ikut dimuat di semua halaman yang mengimpornya.", base, lineCount, scriptModuleWarnLines),
		Suggestion:   "Pecah per domain (useProjects.ts, useDiagnostics.ts, ...) agar tree-shaking dan lazy loading efektif.",
		TargetFile:   relPath,
		Line:         1,
		Snippet:      firstNonEmptyLine(lines),
		TheoryBasis:  "Modular Design (David Parnas — Information Hiding) & Code Splitting (Vite/Webpack Dynamic Import).",
		SuggestedFix: "// composables/api/projects.ts\n// composables/api/diagnostics.ts\n// composables/useApi.ts -> re-export saja",
	}}
}

// inspectFileSize menandai halaman/komponen/screen yang terlalu besar
func inspectFileSize(relPath string, lang string, lines []string, isAndroid bool) []models.Finding {
	lineCount := len(lines)
	base := filepath.Base(relPath)
	lowerPath := strings.ToLower(filepath.ToSlash(relPath))

	isUIComponent := lang == "vue" || lang == "svelte" ||
		strings.HasSuffix(base, ".tsx") || strings.HasSuffix(base, ".jsx")
	isAndroidScreen := isAndroidScreenFile(base, isAndroid)
	isFlutterWidget := isFlutterWidgetFile(lang, lowerPath, base)

	switch {
	case isUIComponent || isAndroidScreen || isFlutterWidget:
		kind := webComponentSize
		if isAndroidScreen {
			kind = androidScreenSize
		} else if isFlutterWidget {
			kind = flutterWidgetSize
		}
		return uiComponentSizeFinding(kind, relPath, base, lines)
	case lang == "xml":
		return xmlLayoutSizeFinding(relPath, base, lineCount)
	case isJSLang(lang) && lineCount > scriptModuleWarnLines && isFrontendSourcePath(lowerPath):
		return scriptModuleSizeFinding(relPath, base, lines)
	}
	return nil
}

func isFrontendSourcePath(lowerPath string) bool {
	for _, seg := range []string{"/components/", "/pages/", "/views/", "/composables/", "/stores/", "/hooks/", "/layouts/", "/screens/"} {
		if strings.Contains("/"+lowerPath, seg) {
			return !strings.Contains(lowerPath, "server/") && !strings.Contains(lowerPath, "api/")
		}
	}
	return false
}

// tagSpan mengembalikan teks tag HTML/XML yang dimulai pada baris idx kolom start hingga '>' penutup
// di luar tanda kutip (maks 25 baris), sehingga atribut berisi '=>' atau '>' tidak memotong tag.
func tagSpan(lines []string, idx int, start int) string {
	var b strings.Builder
	var quote byte
	for j := idx; j < len(lines) && j <= idx+25; j++ {
		seg := lines[j]
		if j == idx {
			seg = seg[start:]
		} else {
			b.WriteByte(' ')
		}
		for k := 0; k < len(seg); k++ {
			c := seg[k]
			b.WriteByte(c)
			switch {
			case quote != 0:
				if c == quote {
					quote = 0
				}
			case c == '"' || c == '\'':
				quote = c
			case c == '>':
				return b.String()
			}
		}
	}
	return b.String()
}

// vueSections menandai bagian setiap baris berkas .vue: 't' template, 's' script, 'c' style
func vueSections(lines []string) []byte {
	sec := make([]byte, len(lines))
	cur := byte(0)
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "<template"):
			cur = 't'
		case strings.HasPrefix(line, "<script"):
			cur = 's'
		case strings.HasPrefix(line, "<style"):
			cur = 'c'
		}
		sec[i] = cur
		if strings.HasPrefix(line, "</template>") || strings.HasPrefix(line, "</script>") || strings.HasPrefix(line, "</style>") {
			cur = 0
		}
	}
	return sec
}

// feCounter menghitung pelanggaran ambang per berkas beserta baris pertamanya.
type feCounter struct {
	count int
	first int
}

func newFECounter() feCounter { return feCounter{first: -1} }

func (c *feCounter) add(n, idx int) {
	c.count += n
	if c.first < 0 {
		c.first = idx
	}
}

// feScan menyimpan status pemindaian satu berkas frontend.
type feScan struct {
	set                                *uiIssueSet
	lines                              []string
	lang                               string
	isMarkup, isAppSource              bool
	sections                           []byte
	listenerUnpaired, intervalUnpaired bool
	staticStyles, hexColors            feCounter
	importants, consoleLogs            feCounter
}

func newFEScan(set *uiIssueSet, content, relPath, lang string, lines []string) *feScan {
	isMarkup := lang == "vue" || lang == "svelte" || strings.HasSuffix(relPath, ".tsx") || strings.HasSuffix(relPath, ".jsx")
	s := &feScan{
		set:         set,
		lines:       lines,
		lang:        lang,
		isMarkup:    isMarkup,
		isAppSource: isMarkup || isFrontendSourcePath(strings.ToLower(filepath.ToSlash(relPath))),
		// Listener / timer leak: dihitung per berkas
		listenerUnpaired: strings.Contains(content, "addEventListener(") && !strings.Contains(content, "removeEventListener("),
		intervalUnpaired: strings.Contains(content, "setInterval(") && !strings.Contains(content, "clearInterval("),
		staticStyles:     newFECounter(),
		hexColors:        newFECounter(),
		importants:       newFECounter(),
		consoleLogs:      newFECounter(),
	}
	if lang == "vue" {
		s.sections = vueSections(lines)
	}
	return s
}

func isFELineSkippable(trimmed string) bool {
	return trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "<!--")
}

// scanScriptRules menjalankan aturan yang berlaku pada semua berkas frontend (bukan hanya markup).
func (s *feScan) scanScriptRules(idx int, line string) {
	if (s.listenerUnpaired && strings.Contains(line, "addEventListener(") && !strings.Contains(line, "once: true")) ||
		(s.intervalUnpaired && strings.Contains(line, "setInterval(")) {
		s.set.hit("FE-LISTENER-LEAK", idx)
	}
	if feHeavyImportRegex.MatchString(line) {
		s.set.hit("FE-HEAVY-LIB-IMPORT", idx)
	}
	if m := feBase64AssetRegex.FindString(line); len(m) > 4000 {
		s.set.hit("FE-INLINE-BASE64-ASSET", idx)
	}
	if s.isAppSource && feConsoleLogRegex.MatchString(line) {
		s.consoleLogs.add(1, idx)
	}
}

// sectionFlags menentukan apakah baris ada di <template> / <style>; ok=false bila baris di <script> .vue.
func (s *feScan) sectionFlags(idx int) (inTemplate, inStyle, ok bool) {
	if s.sections == nil {
		return true, true, true
	}
	inTemplate, inStyle = s.sections[idx] == 't', s.sections[idx] == 'c'
	return inTemplate, inStyle, inTemplate || inStyle
}

// vForTag mengembalikan teks tag yang memuat v-for pada posisi pos di baris idx.
func (s *feScan) vForTag(idx, pos int) string {
	line := s.lines[idx]
	tagStart := strings.LastIndex(line[:pos], "<")
	if tagStart < 0 {
		tagStart = 0
	}
	tag := tagSpan(s.lines, idx, tagStart)
	// cari awal tag jika atribut v-for berada di baris lanjutan
	if !strings.Contains(line[:pos], "<") {
		for k := idx - 1; k >= 0 && k >= idx-8; k-- {
			if lt := strings.LastIndex(s.lines[k], "<"); lt >= 0 {
				return tagSpan(s.lines, k, lt)
			}
		}
	}
	return tag
}

// checkVFor: v-for tanpa key & v-if bersamaan.
func (s *feScan) checkVFor(idx int, line string) {
	pos := strings.Index(line, "v-for=")
	if s.lang != "vue" || pos < 0 {
		return
	}
	tag := s.vForTag(idx, pos)
	if !util.ContainsAny(tag, ":key", "v-bind:key", " key=") {
		s.set.hit("FE-VFOR-MISSING-KEY", idx)
	}
	if strings.Contains(tag, "v-if=") {
		s.set.hit("FE-VIF-WITH-VFOR", idx)
	}
}

// checkImages: <img> — dimensi, alt, lazy.
func (s *feScan) checkImages(idx int, line string) {
	for _, loc := range feImgTagRegex.FindAllStringIndex(line, -1) {
		tag := tagSpan(s.lines, idx, loc[0])
		hasWidth := strings.Contains(tag, "width=") || feImgWidthClassRegex.MatchString(tag)
		hasHeight := strings.Contains(tag, "height=") || feImgHeightClassRegex.MatchString(tag)
		if !hasWidth || !hasHeight {
			s.set.hit("FE-IMG-LAYOUT-SHIFT", idx)
		}
		if !strings.Contains(tag, "alt=") && !strings.Contains(tag, ":alt=") {
			s.set.hit("FE-IMG-MISSING-ALT", idx)
		}
		if !strings.Contains(tag, "loading=") {
			s.set.hit("FE-IMG-NO-LAZY", idx)
		}
	}
}

func (s *feScan) checkClickable(idx int, line string) {
	m := feClickableDivRegex.FindStringIndex(line)
	if m == nil {
		return
	}
	tag := tagSpan(s.lines, idx, m[0])
	if !strings.Contains(tag, "role=") || !util.ContainsAny(tag, "tabindex", "tabIndex") {
		s.set.hit("FE-CLICKABLE-NON-INTERACTIVE", idx)
	}
}

// hitNumeric mencatat ruleID untuk tiap kecocokan grup-1 regex yang angkanya memenuhi pred.
func (s *feScan) hitNumeric(re *regexp.Regexp, line string, idx int, ruleID string, pred func(float64) bool) {
	for _, m := range re.FindAllStringSubmatch(line, -1) {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil && pred(v) {
			s.set.hit(ruleID, idx)
		}
	}
}

func (s *feScan) checkSizing(idx int, line string) {
	small := func(v float64) bool { return v < 11 }
	wide := func(v float64) bool { return v >= 768 }
	s.hitNumeric(feSmallFontTwRegex, line, idx, "FE-SMALL-FONT", small)
	s.hitNumeric(feSmallFontCssRegex, line, idx, "FE-SMALL-FONT", small)
	s.hitNumeric(feFixedWidthCssRegex, line, idx, "FE-FIXED-WIDTH-LAYOUT", wide)
	s.hitNumeric(feFixedWidthTwRegex, line, idx, "FE-FIXED-WIDTH-LAYOUT", wide)
}

func (s *feScan) checkStyleCounters(idx int, line string) {
	if feStaticStyleRegex.MatchString(line) {
		s.staticStyles.add(1, idx)
	}
	if n := len(feHexColorRegex.FindAllStringIndex(line, -1)); n > 0 && !strings.Contains(line, "var(--") {
		s.hexColors.add(n, idx)
	}
	if n := len(feImportantRegex.FindAllStringIndex(line, -1)); n > 0 {
		s.importants.add(n, idx)
	}
	s.hitNumeric(feZIndexRegex, line, idx, "FE-ZINDEX-ESCALATION", func(v float64) bool { return v >= 1000 })
}

// scanMarkupRules menjalankan aturan markup/gaya (Vue/Svelte/JSX) pada satu baris.
func (s *feScan) scanMarkupRules(idx int, line string) {
	// Pada .vue: aturan markup hanya di <template>, aturan gaya di <template>/<style>; abaikan <script>
	inTemplate, _, ok := s.sectionFlags(idx)
	if !ok {
		return
	}
	if inTemplate {
		s.checkVFor(idx, line)
		s.checkImages(idx, line)
		s.checkClickable(idx, line)
	}
	s.checkSizing(idx, line)
	s.checkStyleCounters(idx, line)
}

// flush mencatat aturan berbasis ambang jumlah per berkas.
func (s *feScan) flush() {
	hitN := func(ruleID string, c feCounter, threshold int) {
		if c.count < threshold || c.first < 0 {
			return
		}
		s.set.hit(ruleID, c.first)
		s.set.issues[ruleID].count = c.count
	}
	hitN("FE-INLINE-STYLE-OVERUSE", s.staticStyles, 8)
	hitN("FE-HARDCODED-COLORS", s.hexColors, 12)
	hitN("FE-IMPORTANT-OVERUSE", s.importants, 5)
	hitN("FE-CONSOLE-LOG-LEFTOVER", s.consoleLogs, 3)
}

func collectFrontendUIIssues(set *uiIssueSet, content string, relPath string, lang string, lines []string) {
	s := newFEScan(set, content, relPath, lang, lines)
	for idx, line := range lines {
		if isFELineSkippable(strings.TrimSpace(line)) {
			continue
		}
		s.scanScriptRules(idx, line)
		if s.isMarkup {
			s.scanMarkupRules(idx, line)
		}
	}
	s.flush()
}

// androidCodeScan menyimpan konteks berkas Kotlin/Java Android dan status blok companion.
type androidCodeScan struct {
	set                                  *uiIssueSet
	isUILayer, isComposable, isViewModel bool
	hasLazyList                          bool
	inCompanion                          bool
	companionDepth                       int
}

func newAndroidCodeScan(set *uiIssueSet, content, relPath string) *androidCodeScan {
	base := filepath.Base(relPath)
	return &androidCodeScan{
		set:          set,
		isUILayer:    util.ContainsAny(base, "Activity", "Fragment", "Screen") || strings.Contains(content, "@Composable"),
		isComposable: strings.Contains(content, "@Composable"),
		isViewModel:  strings.Contains(base, "ViewModel"),
		hasLazyList:  util.ContainsAny(content, "LazyColumn", "LazyRow", "LazyVerticalGrid"),
	}
}

// scanBasic: aturan berbasis satu regex (thread utama, GlobalScope, notifyDataSetChanged, static context).
func (a *androidCodeScan) scanBasic(idx int, trimmed string) {
	if andMainThreadBlockRegex.MatchString(trimmed) {
		// runBlocking/Thread.sleep di luar layer UI masih mungkin wajar (CLI/test); allowMainThreadQueries selalu ditandai
		if a.isUILayer || strings.Contains(trimmed, "allowMainThreadQueries") {
			a.set.hit("AND-MAIN-THREAD-BLOCKING", idx)
		}
	}
	if andGlobalScopeRegex.MatchString(trimmed) {
		a.set.hit("AND-GLOBALSCOPE-LEAK", idx)
	}
	if andNotifyAllRegex.MatchString(trimmed) {
		a.set.hit("AND-NOTIFY-DATASET-CHANGED", idx)
	}
	if andStaticContextJava.MatchString(trimmed) {
		a.set.hit("AND-STATIC-CONTEXT-LEAK", idx)
	}
}

// scanCompanion melacak blok companion object / object deklarasi tingkat atas.
func (a *androidCodeScan) scanCompanion(idx int, trimmed string) {
	if strings.HasPrefix(trimmed, "companion object") || strings.HasPrefix(trimmed, "object ") {
		a.inCompanion = true
		a.companionDepth = 0
	}
	if !a.inCompanion {
		return
	}
	a.companionDepth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
	if andContextVarKotlin.MatchString(trimmed) {
		a.set.hit("AND-STATIC-CONTEXT-LEAK", idx)
	}
	if a.companionDepth <= 0 && strings.Contains(trimmed, "}") {
		a.inCompanion = false
	}
}

// scanCompose: aturan Jetpack Compose dan decode bitmap di UI.
func (a *androidCodeScan) scanCompose(idx int, trimmed string) {
	if a.isComposable && !a.isViewModel && andMutableStateRegex.MatchString(trimmed) &&
		!strings.Contains(trimmed, "remember") && !strings.HasPrefix(trimmed, "private val _") {
		a.set.hit("AND-COMPOSE-STATE-NO-REMEMBER", idx)
	}
	if a.hasLazyList && andLazyItemsRegex.MatchString(trimmed) && !util.ContainsAny(trimmed, "key =", "key=") {
		a.set.hit("AND-COMPOSE-LAZY-NO-KEY", idx)
	}
	if a.isUILayer && andBitmapDecodeRegex.MatchString(trimmed) {
		a.set.hit("AND-BITMAP-DECODE-UI", idx)
	}
}

func collectAndroidCodeIssues(set *uiIssueSet, content string, relPath string, lines []string) {
	a := newAndroidCodeScan(set, content, relPath)
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		a.scanBasic(idx, trimmed)
		a.scanCompanion(idx, trimmed)
		a.scanCompose(idx, trimmed)
	}
}

var imageViewTags = map[string]bool{"ImageView": true, "ImageButton": true, "AppCompatImageView": true, "ShapeableImageView": true}

var touchableTags = map[string]bool{
	"Button": true, "ImageButton": true, "MaterialButton": true, "CheckBox": true,
	"RadioButton": true, "Switch": true, "SwitchMaterial": true,
}

// xmlShortTagName mengambil nama tag tanpa paket (com.foo.Bar -> Bar).
func xmlShortTagName(tagName string) string {
	if i := strings.LastIndex(tagName, "."); i >= 0 {
		return tagName[i+1:]
	}
	return tagName
}

func isTouchableTag(short, tag string) bool {
	if touchableTags[short] {
		return true
	}
	return (short == "ImageView" || short == "TextView") && (strings.Contains(tag, "onClick") || strings.Contains(tag, `clickable="true"`))
}

func hasSmallDimension(tag string) bool {
	for _, dm := range xmlDimensionDpRegex.FindAllStringSubmatch(tag, -1) {
		if v, _ := strconv.Atoi(dm[2]); v > 0 && v < 48 {
			return true
		}
	}
	return false
}

// checkXMLOpenTag memeriksa aksesibilitas (contentDescription, target sentuh) pada tag yang dibuka di baris idx.
func checkXMLOpenTag(set *uiIssueSet, lines []string, idx int, trimmed string, m []int) {
	short := xmlShortTagName(trimmed[m[2]:m[3]])
	tag := tagSpan(lines, idx, strings.Index(lines[idx], "<"))

	if imageViewTags[short] && !strings.Contains(tag, "contentDescription") && !strings.Contains(tag, `importantForAccessibility="no"`) {
		set.hit("AND-IMAGE-NO-CONTENTDESC", idx)
	}
	if isTouchableTag(short, tag) && !strings.Contains(tag, "minHeight") && !strings.Contains(tag, "minWidth") && hasSmallDimension(tag) {
		set.hit("AND-SMALL-TOUCH-TARGET", idx)
	}
}

func collectAndroidLayoutIssues(set *uiIssueSet, relPath string, lines []string) {
	hardcodedColors, firstColor := 0, -1

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		if xmlTextSizeDpRegex.MatchString(trimmed) {
			set.hit("AND-TEXTSIZE-NOT-SP", idx)
		}
		if m := xmlHardcodedTextRegex.FindStringSubmatch(trimmed); m != nil && m[1] != "contentDescription" {
			set.hit("AND-HARDCODED-TEXT", idx)
		}
		if xmlLegacyListRegex.MatchString(trimmed) {
			set.hit("AND-LEGACY-LISTVIEW", idx)
		}
		if xmlHardcodedColorRegex.MatchString(trimmed) {
			hardcodedColors++
			if firstColor < 0 {
				firstColor = idx
			}
		}
		if m := xmlOpenTagRegex.FindStringSubmatchIndex(trimmed); m != nil {
			checkXMLOpenTag(set, lines, idx, trimmed, m)
		}
	}

	if hardcodedColors >= 5 && firstColor >= 0 {
		set.hit("AND-HARDCODED-COLOR-XML", firstColor)
		set.issues["AND-HARDCODED-COLOR-XML"].count = hardcodedColors
	}
}

// isSelfClosingAt: ">" di posisi gt didahului "/" (tag self-closing).
func isSelfClosingAt(s string, gt int) bool {
	return gt > 0 && s[gt-1] == '/'
}

// multilineTagSelfCloses mencari ">" penutup tag multi-baris mulai baris from dan melaporkan apakah self-closing.
func multilineTagSelfCloses(lines []string, from int) bool {
	for j := from; j < len(lines); j++ {
		if g := strings.Index(lines[j], ">"); g >= 0 {
			return isSelfClosingAt(lines[j], g)
		}
	}
	return false
}

// xmlMaxNestingDepth menghitung kedalaman elemen XML terdalam dan baris tempat kedalaman itu dicapai
func xmlMaxNestingDepth(lines []string) (int, int) {
	depth, maxDepth, maxLine := 0, 0, 1
	inComment := false
	for idx, line := range lines {
		s := line
		for len(s) > 0 {
			if inComment {
				end := strings.Index(s, "-->")
				if end < 0 {
					break
				}
				inComment = false
				s = s[end+3:]
				continue
			}
			lt := strings.Index(s, "<")
			if lt < 0 {
				break
			}
			s = s[lt:]
			switch {
			case strings.HasPrefix(s, "<!--"):
				inComment = true
				s = s[4:]
			case strings.HasPrefix(s, "<?"), strings.HasPrefix(s, "<!"):
				s = s[2:]
			case strings.HasPrefix(s, "</"):
				depth--
				s = s[2:]
			default:
				depth++
				if depth > maxDepth {
					maxDepth = depth
					maxLine = idx + 1
				}
				gt := strings.Index(s, ">")
				if gt < 0 {
					// tag multi-baris: cari penutupnya di baris berikutnya
					if multilineTagSelfCloses(lines, idx+1) {
						depth--
					}
					s = ""
					break
				}
				if isSelfClosingAt(s, gt) {
					depth--
				}
				s = s[gt+1:]
			}
		}
	}
	return maxDepth, maxLine
}

func collectFlutterIssues(set *uiIssueSet, content string, lines []string) {
	usesCachedImage := strings.Contains(content, "CachedNetworkImage") || strings.Contains(content, "cacheWidth")
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if dartListNonBuilderRegex.MatchString(trimmed) {
			set.hit("MOB-LISTVIEW-NON-BUILDER", idx)
		}
		if dartShrinkWrapRegex.MatchString(trimmed) {
			set.hit("MOB-SHRINKWRAP-LIST", idx)
		}
		if !usesCachedImage && dartImageNetworkRegex.MatchString(trimmed) {
			set.hit("MOB-IMAGE-NETWORK-NO-CACHE", idx)
		}
		if dartOpacityRegex.MatchString(trimmed) {
			set.hit("MOB-OPACITY-WIDGET", idx)
		}
	}
}

// InspectAssetFile menandai aset gambar/video yang terlalu berat untuk dimuat aplikasi
func InspectAssetFile(relPath string, size int64) *models.Finding {
	ext := strings.ToLower(filepath.Ext(relPath))
	kind, ok := auditAssetExtensions[ext]
	if !ok {
		return nil
	}
	lowerPath := strings.ToLower(filepath.ToSlash(relPath))
	isAndroidRes := strings.Contains(lowerPath, "/res/drawable") || strings.Contains(lowerPath, "/res/mipmap")

	warnAt, critAt := int64(webImageWarnBytes), int64(webImageCriticalBytes)
	switch {
	case kind == "video":
		warnAt, critAt = videoWarnBytes, videoWarnBytes*4
	case kind == "svg":
		warnAt, critAt = svgWarnBytes, svgWarnBytes*5
	case isAndroidRes:
		warnAt, critAt = androidImageWarnBytes, webImageWarnBytes*2
	}
	if size < warnAt {
		return nil
	}

	severity := "WARNING"
	if size >= critAt {
		severity = "CRITICAL"
	}
	category := "frontend"
	if isAndroidRes || strings.Contains(lowerPath, "android/") || strings.Contains(lowerPath, "/flutter_assets/") {
		category = "mobile"
	}

	suggestion := "Kompres dan konversi ke WebP/AVIF, sesuaikan resolusi dengan ukuran tampil maksimum, dan sediakan srcset untuk layar berbeda."
	fix := "# Contoh kompresi:\ncwebp -q 75 -resize 1600 0 hero.png -o hero.webp"
	switch {
	case kind == "video":
		suggestion = "Transcode ke H.264/VP9 bitrate lebih rendah, gunakan poster image, preload=\"none\", atau streaming dari CDN."
		fix = "ffmpeg -i intro.mp4 -vcodec libx264 -crf 28 -preset slow -vf scale=1280:-2 intro-720p.mp4"
	case kind == "svg":
		suggestion = "Optimasi dengan SVGO, hapus metadata/raster tertanam, atau ganti dengan PNG/WebP bila berisi gambar raster."
		fix = "npx svgo illustration.svg -o illustration.min.svg"
	case ext == ".gif":
		suggestion = "Ganti GIF animasi dengan video MP4/WebM (autoplay muted loop) atau animated WebP — ukurannya bisa 5–10× lebih kecil."
		fix = "ffmpeg -i anim.gif -movflags faststart -pix_fmt yuv420p anim.mp4"
	case isAndroidRes:
		suggestion = "Konversi ke WebP (Android Studio: Convert to WebP), sediakan varian per density (mdpi–xxxhdpi), atau gunakan VectorDrawable untuk ikon."
		fix = "# Android Studio: klik kanan drawable -> Convert to WebP (lossy 75%)"
	}

	return &models.Finding{
		Category:     category,
		Severity:     severity,
		RuleID:       "ASSET-OVERSIZED",
		Title:        fmt.Sprintf("Aset Media Terlalu Berat (%s)", humanBytes(size)),
		Description:  fmt.Sprintf("Berkas '%s' berukuran %s (ambang %s). Aset berat memperlambat waktu muat halaman/layar (LCP), memboroskan kuota data pengguna mobile, dan memperbesar ukuran build/APK.", filepath.Base(relPath), humanBytes(size), humanBytes(warnAt)),
		Suggestion:   suggestion,
		TargetFile:   relPath,
		Line:         1,
		TheoryBasis:  "Google Core Web Vitals (LCP < 2.5s) & web.dev Image Optimization; Android App Bundle Size Guidelines.",
		SuggestedFix: fix,
	}
}

func humanBytes(n int64) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.0f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%d B", n)
}

func firstNonEmptyLine(lines []string) string {
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			return t
		}
	}
	return ""
}

// isAndroidLayoutXML membatasi pemeriksaan XML hanya pada layout Android (bukan values/manifest/konfigurasi lain)
func isAndroidLayoutXML(relPath string) bool {
	p := filepath.ToSlash(strings.ToLower(relPath))
	return strings.Contains(p, "/res/layout") || strings.HasPrefix(p, "res/layout")
}
