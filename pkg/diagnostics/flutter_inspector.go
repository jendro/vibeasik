package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Audit best practice khusus Flutter. Seluruh aturan di berkas ini hanya dijalankan pada kode Dart
// milik aplikasi Flutter (lihat isFlutterSource); Dart murni (server shelf/dart_frog, CLI, paket
// non-UI) tidak diperiksa dengan aturan widget/lifecycle Flutter.

var (
	flutterPubspecRegex = regexp.MustCompile(`(?m)^\s*sdk:\s*flutter\b|^flutter:\s*$`)
	flutterLintsRegex   = regexp.MustCompile(`(?m)^\s*include:\s*package:(?:flutter_lints|lints|very_good_analysis|lint|solid_lints|pedantic_mono|rigorous_lints)/`)

	// Deklarasi build(BuildContext) pada StatelessWidget/State
	dartBuildDeclRegex = regexp.MustCompile(`^\s*(?:@override\s+)?Widget\s+build\s*\(\s*BuildContext\s+\w+\s*\)`)

	dartSetStateRegex     = regexp.MustCompile(`\bsetState\s*\(`)
	dartHeavySyncRegex    = regexp.MustCompile(`\b(?:jsonDecode|json\.decode|readAsStringSync|readAsBytesSync|readAsLinesSync|writeAsStringSync|sleep)\s*\(`)
	dartInlineFutureRegex = regexp.MustCompile(`\b(future|stream)\s*:\s*([A-Za-z_][\w.]*(?:\(\))?\.)?([A-Za-z_]\w*)\s*\(`)
	dartAwaitRegex        = regexp.MustCompile(`\bawait\b`)
	dartClassOpenRegex    = regexp.MustCompile(`\b(?:class|mixin|extension)\b`)
	dartDisposeDeclRegex  = regexp.MustCompile(`^\s*(?:@override\s+)?(?:Future<void>|void)\s+(?:dispose|close|cancel)\s*\(\s*\)`)
	dartContextUseRegex   = regexp.MustCompile(`\bcontext\b`)
	dartMountedRegex      = regexp.MustCompile(`\bmounted\b`)
	dartAsyncBodyRegex    = regexp.MustCompile(`\basync\*?\s*(?:\{|=>)`)
	dartPrintRegex        = regexp.MustCompile(`(?:^|[^\w.])print\s*\(`)
	dartPrefsSetRegex     = regexp.MustCompile(`\.set(?:String|StringList)\s*\(`)
	dartSensitiveKeyRegex = regexp.MustCompile(`(?i)\.set(?:String|StringList)\s*\(\s*(?:['"][^'"]*(?:token|jwt|passw(?:or)?d|secret|credential|api_?key|session_?(?:id|token|key)|\bpin\b|_pin\b|\bpin_|pincode|\botp\b)[^'"]*['"]|[\w.]*(?:token|jwt|passw(?:or)?d|secret|credential|api_?key)\w*)\s*,`)

	// Objek ber-lifecycle yang wajib dilepas oleh pemiliknya
	dartResourceTypes      = `TextEditingController|ScrollController|PageController|TabController|AnimationController|FocusNode|StreamController|StreamSubscription|Timer|ValueNotifier`
	dartResourceCtorRegex  = regexp.MustCompile(`\b(?:TextEditingController|ScrollController|PageController|TabController|AnimationController|FocusNode|StreamController|ValueNotifier)(?:<[^>]*>)?\s*(?:\.\w+)?\(`)
	dartTypedFieldRegex    = regexp.MustCompile(`^\s*(?:late\s+)?(?:final\s+)?(` + dartResourceTypes + `)(?:<[^>]*>)?\??\s+(_?\w+)\s*(=|;)`)
	dartInferredFieldRegex = regexp.MustCompile(`^\s*(?:late\s+)?(?:final|var)\s+(_?\w+)\s*=\s*(` + dartResourceTypes + `)(?:<[^>]*>)?\s*[.(]`)
	dartListenFieldRegex   = regexp.MustCompile(`^\s*(?:late\s+)?(?:final\s+|var\s+)?(?:StreamSubscription(?:<[^>]*>)?\??\s+)?(_?\w+)\s*=\s*[\w.()]+\.listen\(`)
)

// flutterRules memakai format uiRule yang sama dengan aturan tampilan (Description: berkas, jumlah, baris)
var flutterRules = map[string]uiRule{
	"MOB-SETSTATE-IN-BUILD": {
		Category:     "mobile",
		Severity:     "CRITICAL",
		Title:        "Flutter setState() Dipanggil Langsung di build()",
		Description:  "Berkas '%s' memanggil setState() langsung di tubuh build() sebanyak %d kali (pertama pada baris %d). Flutter melempar \"setState() called during build\" atau memicu rebuild berulang tanpa henti.",
		Suggestion:   "Mutasi state hanya di event handler (onPressed, onChanged) atau lifecycle (initState/didChangeDependencies); untuk efek setelah frame gunakan WidgetsBinding.instance.addPostFrameCallback.",
		TheoryBasis:  "Flutter API Docs — State.setState (\"must not be called during build\") & Widget Lifecycle.",
		SuggestedFix: "@override\nvoid initState() {\n  super.initState();\n  WidgetsBinding.instance.addPostFrameCallback((_) => setState(() => _ready = true));\n}",
	},
	"MOB-UI-THREAD-FREEZE": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "Flutter Komputasi/IO Sinkron di build()",
		Description:  "Berkas '%s' menjalankan jsonDecode / IO sinkron / sleep di dalam build() sebanyak %d kali (pertama pada baris %d). build() dipanggil setiap frame/rebuild di UI isolate sehingga frame melewati 16ms (jank).",
		Suggestion:   "Pindahkan parsing & IO ke state/repository (initState, FutureProvider, Bloc) dan gunakan compute()/Isolate.run untuk JSON besar; build() hanya memetakan state ke widget.",
		TheoryBasis:  "Flutter Performance Best Practices — Control build() cost & Concurrency and isolates (compute / Isolate.run).",
		SuggestedFix: "late final Future<Report> _report = compute(parseReport, rawJson);",
	},
	"MOB-FUTURE-IN-BUILD": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "Flutter FutureBuilder/StreamBuilder Membuat Future di build()",
		Description:  "Berkas '%s' membuat future/stream langsung di argumen FutureBuilder/StreamBuilder sebanyak %d kali (pertama pada baris %d). Setiap rebuild memicu request baru, loading berkedip, dan beban jaringan berlipat.",
		Suggestion:   "Simpan Future/Stream di field State (initState / late final) atau provider, lalu oper field tersebut ke future:/stream:.",
		TheoryBasis:  "Flutter API Docs — FutureBuilder (\"The future must have been obtained earlier, e.g. during State.initState\").",
		SuggestedFix: "late final Future<List<Item>> _items = repo.fetchItems();\n...\nFutureBuilder(future: _items, builder: ...)",
	},
	"MOB-CONTROLLER-IN-BUILD": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "Flutter Controller/FocusNode Dibuat di build()",
		Description:  "Berkas '%s' membuat controller/FocusNode di dalam build() sebanyak %d kali (pertama pada baris %d). Objek dibuat ulang setiap rebuild: teks/posisi scroll hilang dan instance lama tidak pernah di-dispose.",
		Suggestion:   "Jadikan field State: buat di initState (atau late final) dan lepaskan di dispose().",
		TheoryBasis:  "Flutter Cookbook — Handle changes to a text field (controller dibuat di State, di-dispose di dispose()).",
		SuggestedFix: "class _FormState extends State<Form> {\n  final _name = TextEditingController();\n  @override\n  void dispose() { _name.dispose(); super.dispose(); }\n}",
	},
	"MOB-ASYNC-GAP-CONTEXT": {
		Category:     "mobile",
		Severity:     "WARNING",
		Title:        "Flutter BuildContext/setState Dipakai Setelah await Tanpa Cek mounted",
		Description:  "Berkas '%s' memakai context atau setState setelah await tanpa memeriksa mounted sebanyak %d kali (pertama pada baris %d). Bila layar sudah ditutup saat future selesai, aplikasi melempar \"setState() called after dispose()\" atau memakai context yang tidak valid.",
		Suggestion:   "Tambahkan if (!mounted) return; (State) atau if (!context.mounted) return; setelah await, atau ambil Navigator/ScaffoldMessenger sebelum await.",
		TheoryBasis:  "Dart lint use_build_context_synchronously (flutter_lints) & Flutter API Docs — State.mounted.",
		SuggestedFix: "await repo.save(item);\nif (!context.mounted) return;\nNavigator.of(context).pop();",
	},
	"MOB-INSECURE-LOCAL-STORAGE": {
		Category:     "security",
		Severity:     "CRITICAL",
		Title:        "Flutter Token/Kredensial Disimpan di SharedPreferences",
		Description:  "Berkas '%s' menyimpan nilai sensitif (token/password/secret) ke SharedPreferences sebanyak %d kali (pertama pada baris %d). SharedPreferences adalah berkas teks polos (XML/plist) yang mudah diekstrak dari backup atau perangkat root/jailbreak.",
		Suggestion:   "Gunakan flutter_secure_storage (Android Keystore / iOS Keychain) untuk token & kredensial; SharedPreferences hanya untuk preferensi non-sensitif.",
		TheoryBasis:  "OWASP MASVS-STORAGE-1 & Flutter Docs — Security (store secrets in Keychain/Keystore).",
		SuggestedFix: "const storage = FlutterSecureStorage();\nawait storage.write(key: 'access_token', value: token);",
	},
	"MOB-PRINT-STATEMENT": {
		Category:     "code_smell",
		Severity:     "INFO",
		Title:        "Flutter print() di Kode Aplikasi",
		Description:  "Berkas '%s' memakai print() sebanyak %d kali (pertama pada baris %d). print ikut ke build rilis, dapat membocorkan data ke log perangkat (adb logcat), dan pesan panjang bisa terpotong/di-throttle.",
		Suggestion:   "Gunakan debugPrint/log dari dart:developer atau package logger, dan bungkus log diagnostik dengan if (kDebugMode).",
		TheoryBasis:  "Dart lint avoid_print (flutter_lints) & Flutter Docs — Debugging (debugPrint, dart:developer log).",
		SuggestedFix: "import 'dart:developer' as dev;\ndev.log('sync selesai', name: 'SyncService');",
	},
}

var flutterPubspecCache sync.Map // path pubspec.yaml -> bool (Flutter atau bukan)

// isFlutterSource menandai berkas Dart milik aplikasi/paket Flutter: mengimpor package:flutter,
// memakai BuildContext, atau berada di paket yang pubspec.yaml-nya bergantung pada SDK Flutter.
func isFlutterSource(fullPath string, content string) bool {
	if strings.Contains(content, "package:flutter/") || strings.Contains(content, "BuildContext") {
		return true
	}
	dir := filepath.Dir(fullPath)
	for i := 0; i < 10; i++ {
		pub := filepath.Join(dir, "pubspec.yaml")
		if cached, ok := flutterPubspecCache.Load(pub); ok {
			return cached.(bool)
		}
		if data, err := os.ReadFile(pub); err == nil {
			isFlutter := flutterPubspecRegex.Match(data)
			flutterPubspecCache.Store(pub, isFlutter)
			return isFlutter
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return false
}

// dartBlockEnd mengembalikan indeks baris terakhir dari deklarasi yang dimulai di baris start:
// badan { ... } diikuti hingga kurawal tertutup, badan => diikuti hingga ';' di kedalaman awal.
// dartBlockScan menyimpan status pemindaian karakter demi karakter untuk dartBlockEnd.
type dartBlockScan struct {
	brace, paren int
	opened       bool
	arrow        bool
}

// step memproses karakter s[k]; true bila deklarasi berakhir pada karakter ini.
func (d *dartBlockScan) step(s string, k int) bool {
	switch s[k] {
	case '{':
		d.brace++
		if !d.arrow {
			d.opened = true
		}
	case '}':
		d.brace--
		return d.opened && !d.arrow && d.brace <= 0
	case '(', '[':
		d.paren++
	case ')', ']':
		d.paren--
		return d.arrow && d.paren < 0 // badan => sebagai argumen berakhir saat kurung pemanggil ditutup
	case ',':
		return d.arrow && d.paren <= 0 && d.brace <= 0
	case '=':
		if k+1 < len(s) && s[k+1] == '>' && !d.opened && d.paren == 0 && d.brace == 0 {
			d.arrow = true
		}
	case ';':
		if d.arrow && d.paren <= 0 && d.brace <= 0 {
			return true
		}
		return !d.opened && !d.arrow && d.paren <= 0 // deklarasi tanpa badan
	}
	return false
}

// dartBlockEnd mengembalikan indeks baris terakhir dari deklarasi yang dimulai di baris start:
// badan { ... } diikuti hingga kurawal tertutup, badan => diikuti hingga ';' di kedalaman awal.
func dartBlockEnd(sl []string, start int) int {
	var d dartBlockScan
	for j := start; j < len(sl); j++ {
		s := sl[j]
		for k := 0; k < len(s); k++ {
			if d.step(s, k) {
				return j
			}
		}
	}
	return len(sl) - 1
}

// dartClosureMask menandai, untuk setiap karakter pada baris start..end, apakah posisi itu berada
// di dalam closure (fungsi anonim `() { }`, `() async { }`, atau `=> expr`). Kode di dalam closure
// tidak dieksekusi saat build() berjalan (mis. onPressed), sehingga aturan build() mengabaikannya.
type dartFrame struct {
	ch      byte
	closure bool
}

// dartMasker menyimpan stack kurung dan status badan fungsi untuk dartClosureMask.
type dartMasker struct {
	stack            []dartFrame
	bodyStarted      bool // kurawal/=> pertama di kedalaman 0 adalah badan fungsi itu sendiri, bukan closure
	lastParenControl bool // kurung terakhir yang ditutup adalah kondisi if/for/while/switch/catch
}

func (w *dartMasker) inClosure() bool {
	for _, f := range w.stack {
		if f.closure {
			return true
		}
	}
	return false
}

func (w *dartMasker) popArrows() {
	for len(w.stack) > 0 && w.stack[len(w.stack)-1].ch == '>' {
		w.stack = w.stack[:len(w.stack)-1]
	}
}

func (w *dartMasker) pop() {
	if len(w.stack) > 0 {
		w.stack = w.stack[:len(w.stack)-1]
	}
}

var dartControlKeywords = []string{"if", "for", "while", "switch", "catch"}

// endsWithControlKeyword: teks sebelum '(' diakhiri kata kunci kontrol (bukan bagian identifier lain).
func endsWithControlKeyword(before string) bool {
	for _, kw := range dartControlKeywords {
		if strings.HasSuffix(before, kw) && (len(before) == len(kw) || !isIdentByte(before[len(before)-len(kw)-1])) {
			return true
		}
	}
	return false
}

func (w *dartMasker) openParen(s string, k int) {
	f := dartFrame{ch: '('}
	if endsWithControlKeyword(strings.TrimRight(s[:k], " \t")) {
		f.ch = 'c'
	}
	w.stack = append(w.stack, f)
}

func (w *dartMasker) closeParen() {
	w.popArrows()
	if len(w.stack) > 0 {
		w.lastParenControl = w.stack[len(w.stack)-1].ch == 'c'
		w.stack = w.stack[:len(w.stack)-1]
	}
}

func (w *dartMasker) openBrace(sl []string, j, k, start int) {
	s := sl[j]
	before := strings.TrimRight(s[:k], " \t")
	if before == "" && j > start {
		before = strings.TrimRight(sl[j-1], " \t")
	}
	closure := (strings.HasSuffix(before, ")") && !w.lastParenControl) ||
		strings.HasSuffix(before, "async") || strings.HasSuffix(before, "async*")
	if !w.bodyStarted && len(w.stack) == 0 {
		closure, w.bodyStarted = false, true
	}
	w.stack = append(w.stack, dartFrame{ch: '{', closure: closure})
}

// arrow menangani "=>" di s[k]; mengembalikan indeks karakter terakhir yang dikonsumsi dan true bila ditangani.
func (w *dartMasker) arrow(s string, k int, m []bool) (int, bool) {
	if k+1 >= len(s) || s[k+1] != '>' {
		return k, false
	}
	if !w.bodyStarted && len(w.stack) == 0 {
		w.bodyStarted = true // build(...) => expr
		return k + 1, true
	}
	w.stack = append(w.stack, dartFrame{ch: '>', closure: true})
	m[k] = true
	m[k+1] = true
	return k + 1, true
}

// handle memperbarui stack untuk karakter non-"=>" pada s[k].
func (w *dartMasker) handle(sl []string, j, k, start int) {
	switch sl[j][k] {
	case '(':
		w.openParen(sl[j], k)
	case '[':
		w.stack = append(w.stack, dartFrame{ch: '['})
	case ')', ']':
		w.closeParen()
	case '{':
		w.openBrace(sl, j, k, start)
	case '}':
		w.popArrows()
		w.pop()
	case ',', ';':
		w.popArrows()
	}
}

// dartClosureMask menandai, untuk setiap karakter pada baris start..end, apakah posisi itu berada
// di dalam closure (fungsi anonim `() { }`, `() async { }`, atau `=> expr`). Kode di dalam closure
// tidak dieksekusi saat build() berjalan (mis. onPressed), sehingga aturan build() mengabaikannya.
func dartClosureMask(sl []string, start int, end int) [][]bool {
	w := &dartMasker{}
	mask := make([][]bool, end-start+1)
	for j := start; j <= end && j < len(sl); j++ {
		s := sl[j]
		m := make([]bool, len(s))
		for k := 0; k < len(s); k++ {
			if s[k] == '=' {
				if nk, ok := w.arrow(s, k, m); ok {
					k = nk
					continue
				}
			}
			w.handle(sl, j, k, start)
			m[k] = w.inClosure()
		}
		mask[j-start] = m
	}
	return mask
}

func isIdentByte(b byte) bool {
	return b == '_' || b == '$' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// inspectFlutterBestPractices menjalankan aturan best practice Flutter (lifecycle, build(), async gap,
// keamanan penyimpanan, logging) pada satu berkas Dart Flutter.
// collectFlutterBuildRules: aturan di dalam build(): hanya kode yang benar-benar dieksekusi saat build (di luar closure).
func collectFlutterBuildRules(set *uiIssueSet, content string, sl []string) {
	hasBuilders := strings.Contains(content, "FutureBuilder") || strings.Contains(content, "StreamBuilder")
	for i := range sl {
		if !dartBuildDeclRegex.MatchString(sl[i]) {
			continue
		}
		end := dartBlockEnd(sl, i)
		mask := dartClosureMask(sl, i, end)
		for j := i + 1; j <= end; j++ {
			s, m := sl[j], mask[j-i]
			direct := func(re *regexp.Regexp) bool {
				for _, loc := range re.FindAllStringIndex(s, -1) {
					if loc[0] < len(m) && !m[loc[0]] {
						return true
					}
				}
				return false
			}
			if direct(dartSetStateRegex) {
				set.hit("MOB-SETSTATE-IN-BUILD", j)
			}
			if direct(dartHeavySyncRegex) {
				set.hit("MOB-UI-THREAD-FREEZE", j)
			}
			if direct(dartResourceCtorRegex) {
				set.hit("MOB-CONTROLLER-IN-BUILD", j)
			}
			// future:/stream: dibangun ulang setiap build, juga di dalam builder closure
			if mm := dartInlineFutureRegex.FindStringSubmatch(s); mm != nil && mm[3] != "value" && mm[3] != "error" && hasBuilders {
				set.hit("MOB-FUTURE-IN-BUILD", j)
			}
		}
	}
}

// asyncGapScan melacak titik jeda await dalam satu badan async.
type asyncGapScan struct {
	pending   bool // sudah melewati titik jeda await (widget bisa saja sudah di-unmount)
	exprDepth int  // kurung yang masih terbuka pada ekspresi await (argumennya dievaluasi sebelum jeda)
}

// step memproses satu baris dan mengembalikan true bila baris itu memakai context/setState setelah jeda.
func (a *asyncGapScan) step(s string) bool {
	if dartMountedRegex.MatchString(s) {
		a.pending, a.exprDepth = false, 0
		return false
	}
	if a.exprDepth > 0 {
		_, dp, _ := bracketDelta(s)
		if a.exprDepth += dp; a.exprDepth <= 0 {
			a.exprDepth, a.pending = 0, true
		}
		return false
	}
	flag := a.pending && (dartContextUseRegex.MatchString(s) || dartSetStateRegex.MatchString(s))
	if loc := dartAwaitRegex.FindStringIndex(s); loc != nil {
		if _, dp, _ := bracketDelta(s[loc[0]:]); dp > 0 {
			a.exprDepth = dp // await foo(\n  context, ...) — tunggu hingga kurung ditutup
		} else {
			a.pending = true
		}
	}
	return flag
}

// collectFlutterAsyncGap: context/setState setelah await tanpa cek mounted, per badan async (termasuk closure).
func collectFlutterAsyncGap(set *uiIssueSet, sl []string) {
	flaggedGap := make(map[int]bool)
	for i := range sl {
		if !dartAsyncBodyRegex.MatchString(sl[i]) {
			continue
		}
		end := dartBlockEnd(sl, i)
		var scan asyncGapScan
		for j := i; j <= end; j++ {
			s := sl[j]
			if j > i && dartAsyncBodyRegex.MatchString(s) {
				// closure async bersarang dianalisis tersendiri; lompati badannya
				j = dartBlockEnd(sl, j)
				continue
			}
			if scan.step(s) && !flaggedGap[j] {
				flaggedGap[j] = true
				set.hit("MOB-ASYNC-GAP-CONTEXT", j)
			}
		}
	}
}

func isDartAppCode(relPath string) bool {
	lowerPath := strings.ToLower(filepath.ToSlash(relPath))
	return !strings.HasPrefix(lowerPath, "test/") && !strings.Contains(lowerPath, "/test/") &&
		!strings.HasPrefix(lowerPath, "tool/") && !strings.Contains(lowerPath, "/tool/") &&
		!strings.Contains(lowerPath, "integration_test/")
}

// collectFlutterLogAndStorage: print() di kode aplikasi dan SharedPreferences untuk kunci sensitif.
func collectFlutterLogAndStorage(set *uiIssueSet, content, relPath string, sl, lines []string) {
	isAppCode := isDartAppCode(relPath)
	usesPrefs := strings.Contains(content, "SharedPreferences")
	for i, s := range sl {
		if isAppCode && dartPrintRegex.MatchString(s) && !strings.Contains(s, "kDebugMode") {
			set.hit("MOB-PRINT-STATEMENT", i)
		}
		// Kunci penyimpanan ada di dalam string, jadi baris asli yang dicocokkan
		if usesPrefs && dartPrefsSetRegex.MatchString(s) && dartSensitiveKeyRegex.MatchString(lines[i]) {
			set.hit("MOB-INSECURE-LOCAL-STORAGE", i)
		}
	}
}

// inspectFlutterBestPractices menjalankan aturan best practice Flutter (lifecycle, build(), async gap,
// keamanan penyimpanan, logging) pada satu berkas Dart Flutter.
func inspectFlutterBestPractices(content string, relPath string, lines []string) []models.Finding {
	sl := stripCode(lines, "dart")
	set := newUIIssueSet(lines)

	collectFlutterBuildRules(set, content, sl)
	collectFlutterAsyncGap(set, sl)
	collectFlutterLogAndStorage(set, content, relPath, sl, lines)

	findings := renderIssueSet(set, flutterRules, relPath)
	if f := inspectFlutterResourceLeaks(sl, relPath, lines); f != nil {
		findings = append(findings, *f)
	}
	return findings
}

// inspectFlutterResourceLeaks mencari field controller/FocusNode/StreamController/StreamSubscription/
// Timer yang dibuat oleh kelas itu sendiri namun tidak pernah di-dispose/close/cancel.
// Field yang diterima dari konstruktor (milik widget induk) tidak dihitung.
type dartResourceField struct {
	name, kind string
	line       int
}

// collectDartResourceFields mengumpulkan field resource tingkat kelas dan rentang baris dispose()/close()/cancel().
func collectDartResourceFields(sl []string) ([]dartResourceField, [][2]int) {
	var fields []dartResourceField
	var disposeRanges [][2]int
	depth := 0
	inClass := false // kurawal tingkat atas yang sedang terbuka milik class/mixin/extension (bukan fungsi)
	for i, s := range sl {
		if depth == 0 {
			inClass = dartClassOpenRegex.MatchString(s)
		}
		if depth == 1 && dartDisposeDeclRegex.MatchString(s) {
			disposeRanges = append(disposeRanges, [2]int{i, dartBlockEnd(sl, i)})
		}
		if depth == 1 && inClass { // field kelas
			if f, ok := matchDartResourceField(s, i); ok {
				fields = append(fields, f)
			}
		}
		db, _, _ := bracketDelta(s)
		depth += db
		if depth < 0 {
			depth = 0
		}
	}
	return fields, disposeRanges
}

func matchDartResourceField(s string, line int) (dartResourceField, bool) {
	if m := dartTypedFieldRegex.FindStringSubmatch(s); m != nil {
		return dartResourceField{name: m[2], kind: m[1], line: line}, true
	}
	if m := dartInferredFieldRegex.FindStringSubmatch(s); m != nil {
		return dartResourceField{name: m[1], kind: m[2], line: line}, true
	}
	if m := dartListenFieldRegex.FindStringSubmatch(s); m != nil {
		return dartResourceField{name: m[1], kind: "StreamSubscription", line: line}, true
	}
	return dartResourceField{}, false
}

// isDartFieldLeaked: field dibuat oleh kelas itu sendiri namun tidak pernah dilepas.
func isDartFieldLeaked(f dartResourceField, code string, sl []string, disposeRanges [][2]int) bool {
	q := regexp.QuoteMeta(f.name)
	owned := regexp.MustCompile(`\b`+q+`\s*(?:=|\?\?=)\s*(?:`+dartResourceTypes+`)\b|\b`+q+`\s*=\s*[\w.()]+\.listen\(`).MatchString(code) ||
		strings.Contains(sl[f.line], "=")
	if !owned {
		return false
	}
	// Dilepas bila dipanggil langsung (_c.dispose()) atau disebut di dalam dispose()/close(),
	// mis. for (final c in [_a, _b]) c.dispose();
	nameRe := regexp.MustCompile(`\b` + q + `\b`)
	released := regexp.MustCompile(`\b` + q + `\b[^\n]*\b(?:dispose|close|cancel)\b`).MatchString(code)
	for _, r := range disposeRanges {
		for k := r[0]; k <= r[1] && !released; k++ {
			released = nameRe.MatchString(sl[k])
		}
	}
	return !released
}

// inspectFlutterResourceLeaks mencari field controller/FocusNode/StreamController/StreamSubscription/
// Timer yang dibuat oleh kelas itu sendiri namun tidak pernah di-dispose/close/cancel.
// Field yang diterima dari konstruktor (milik widget induk) tidak dihitung.
func inspectFlutterResourceLeaks(sl []string, relPath string, lines []string) *models.Finding {
	fields, disposeRanges := collectDartResourceFields(sl)
	if len(fields) == 0 {
		return nil
	}
	code := strings.Join(sl, "\n")
	var leaked []string
	first := -1
	for _, f := range fields {
		if !isDartFieldLeaked(f, code, sl, disposeRanges) {
			continue
		}
		leaked = append(leaked, f.name)
		if first < 0 || f.line < first {
			first = f.line
		}
	}
	if len(leaked) == 0 {
		return nil
	}
	sort.Strings(leaked)
	return &models.Finding{
		Category:     "mobile",
		Severity:     "WARNING",
		RuleID:       "MOB-CONTROLLER-LEAK",
		Title:        fmt.Sprintf("Flutter Resource Tidak Dilepas: %s", strings.Join(leaked, ", ")),
		Description:  fmt.Sprintf("Berkas '%s' membuat %d controller/FocusNode/StreamController/StreamSubscription/Timer (%s) tanpa dispose()/close()/cancel(). Listener & ticker tetap hidup setelah layar ditutup sehingga memori bocor dan callback berjalan pada widget yang sudah di-unmount.", filepath.Base(relPath), len(leaked), strings.Join(leaked, ", ")),
		Suggestion:   "Lepaskan di @override void dispose(): controller/FocusNode → .dispose(), StreamController → .close(), StreamSubscription/Timer → .cancel(); panggil super.dispose() terakhir.",
		TargetFile:   relPath,
		Line:         first + 1,
		Snippet:      strings.TrimSpace(lines[first]),
		TheoryBasis:  "Flutter API Docs — State.dispose, ChangeNotifier.dispose & StreamSubscription.cancel (Widget Lifecycle).",
		SuggestedFix: "@override\nvoid dispose() {\n  _controller.dispose();\n  _subscription?.cancel();\n  super.dispose();\n}",
	}
}

// inspectFlutterProject memeriksa konfigurasi paket Flutter (pubspec.yaml & analysis_options.yaml).
func inspectFlutterProject(pubspecPath string, relPubspec string) []models.Finding {
	data, err := os.ReadFile(pubspecPath)
	if err != nil || !flutterPubspecRegex.Match(data) {
		return nil
	}
	dir := filepath.Dir(pubspecPath)
	relDir := filepath.Dir(relPubspec)
	opts, err := os.ReadFile(filepath.Join(dir, "analysis_options.yaml"))
	if err == nil && flutterLintsRegex.Match(opts) {
		return nil
	}
	desc := "Paket Flutter ini tidak memiliki analysis_options.yaml"
	target := filepath.Join(relDir, "pubspec.yaml")
	if err == nil {
		desc = "analysis_options.yaml tidak meng-include set lint resmi (flutter_lints/lints/very_good_analysis)"
		target = filepath.Join(relDir, "analysis_options.yaml")
	}
	return []models.Finding{{
		Category:     "mobile",
		Severity:     "WARNING",
		RuleID:       "MOB-NO-FLUTTER-LINTS",
		Title:        "Flutter Tanpa Set Lint Resmi (flutter_lints)",
		Description:  desc + ". Tanpa lint, `flutter analyze` tidak memperingatkan use_build_context_synchronously, avoid_print, prefer_const_constructors, dan aturan Effective Dart lainnya.",
		Suggestion:   "Tambahkan dev_dependency flutter_lints dan buat analysis_options.yaml berisi include: package:flutter_lints/flutter.yaml, lalu jalankan flutter analyze di CI.",
		TargetFile:   target,
		Line:         1,
		TheoryBasis:  "Flutter Docs — Linting (flutter_lints, bawaan flutter create) & Effective Dart.",
		SuggestedFix: "# analysis_options.yaml\ninclude: package:flutter_lints/flutter.yaml\n\nlinter:\n  rules:\n    - use_build_context_synchronously\n    - prefer_const_constructors",
	}}
}
