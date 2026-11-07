package diagnostics

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Ambang kompleksitas siklomatis (McCabe) per fungsi, mengikuti pita risiko SEI:
// 1–10 sederhana, 11–20 sedang, 21–50 berisiko tinggi, >50 tidak dapat diuji.
// Batas 15 setara gocyclo/Sonar; di atas 30 fungsi praktis mustahil diuji tuntas.
const (
	complexityInfoThreshold = 15
	complexityWarnThreshold = 30
)

// complexityProfile mendefinisikan awal fungsi dan token percabangan untuk satu bahasa.
// Semua pola dicocokkan pada baris yang sudah dibersihkan stripCode (tanpa string & komentar).
type complexityProfile struct {
	label     string
	funcStart []*regexp.Regexp // grup terakhir yang terisi = nama fungsi
	branch    *regexp.Regexp   // setiap kecocokan = +1 titik keputusan
	indent    bool             // badan fungsi ditentukan indentasi (Python), bukan kurung kurawal
	exprBody  *regexp.Regexp   // penanda badan ekspresi tanpa kurawal (=> / = expr)
}

var (
	arrowBodyRegex   = regexp.MustCompile(`=>`)
	kotlinExprRegex  = regexp.MustCompile(`\)\s*(?::[^=]*)?=[^=>]|=>`)
	cLikeBranchRegex = regexp.MustCompile(`\b(?:if|for|while|case|catch)\b|&&|\|\||\s\?\s`)
	// ?? (null-coalescing) sengaja tidak dihitung: nilai default seperti `x ?? this.x` pada copyWith
	// bukan jalur logika yang perlu diuji terpisah, dan menghitungnya membanjiri laporan dengan model data.
	tsBranchRegex = regexp.MustCompile(`\b(?:if|for|while|case|catch)\b|&&|\|\||\s\?\s`)

	tsFuncDeclRegex  = regexp.MustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s*\*?\s*([A-Za-z0-9_$]+)\s*[<(]`)
	tsArrowRegex     = regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let)\s+([A-Za-z0-9_$]+)\s*(?::[^=]+)?=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z0-9_$]+)\s*(?::[^=]+)?=>`)
	tsMethodRegex    = regexp.MustCompile(`^\s*(?:(?:public|private|protected|static|async|override|readonly|get|set)\s+)*\*?([A-Za-z_$][A-Za-z0-9_$]*)\s*(?:<[^>]*>)?\s*\([^)]*\)\s*(?::\s*[^{=]+)?\{\s*$`)
	dartFuncRegex    = regexp.MustCompile(`^\s*(?:@\w+\s+)?(?:(?:static|external)\s+)*([A-Za-z_]\w*(?:<[\w<>,? ]*>)?\??)\s+(?:get\s+)?([a-z_]\w*)\s*(?:<[^>]*>)?\(`)
	dartNotTypeWords = map[string]bool{"return": true, "await": true, "throw": true, "else": true, "case": true, "yield": true, "new": true, "final": true, "var": true, "const": true, "late": true, "if": true, "for": true, "while": true, "switch": true, "import": true, "export": true, "part": true}

	complexityProfiles = map[string]complexityProfile{
		"go": {
			label:     "Go",
			funcStart: []*regexp.Regexp{regexp.MustCompile(`^func\s+(?:\([^)]*\)\s*)?([A-Za-z0-9_]+)\s*[\[(]`)},
			branch:    regexp.MustCompile(`\b(?:if|for|case)\b|&&|\|\|`),
		},
		"typescript": {
			label:     "TypeScript",
			funcStart: []*regexp.Regexp{tsFuncDeclRegex, tsArrowRegex, tsMethodRegex},
			branch:    tsBranchRegex,
			exprBody:  arrowBodyRegex,
		},
		"kotlin": {
			label:     "Kotlin",
			funcStart: []*regexp.Regexp{regexp.MustCompile(`^\s*(?:(?:private|public|internal|protected|override|suspend|inline|open|operator|tailrec)\s+)*fun\s+(?:<[^>]+>\s*)?(?:[A-Za-z0-9_.]+\.)?([A-Za-z0-9_]+)\s*\(`)},
			branch:    regexp.MustCompile(`\b(?:if|for|while|catch)\b|^\s*[^/\n]*->\s*[^{]*$|&&|\|\||\?:`),
			exprBody:  kotlinExprRegex,
		},
		"java": {
			label:     "Java",
			funcStart: []*regexp.Regexp{regexp.MustCompile(`^\s*(?:(?:public|private|protected|static|final|synchronized|abstract)\s+)+[A-Za-z0-9_<>\[\],.? ]+\s+([A-Za-z0-9_]+)\s*\([^;]*$`)},
			branch:    cLikeBranchRegex,
		},
		"dart": {
			label:     "Dart",
			funcStart: []*regexp.Regexp{dartFuncRegex},
			branch:    tsBranchRegex,
			exprBody:  arrowBodyRegex,
		},
		"php": {
			label:     "PHP",
			funcStart: []*regexp.Regexp{regexp.MustCompile(`^\s*(?:(?:public|private|protected|static|final|abstract)\s+)*function\s+&?([A-Za-z0-9_]+)\s*\(`)},
			branch:    regexp.MustCompile(`\b(?:if|elseif|for|foreach|while|case|catch)\b|&&|\|\||\s\?\s`),
		},
		"python": {
			label:     "Python",
			funcStart: []*regexp.Regexp{regexp.MustCompile(`^\s*(?:async\s+)?def\s+([A-Za-z0-9_]+)\s*\(`)},
			branch:    regexp.MustCompile(`\b(?:if|elif|for|while|except|case)\b|\band\b|\bor\b`),
			indent:    true,
		},
	}
)

func init() {
	// JavaScript memakai pola yang sama dengan TypeScript. Vue/Svelte tidak punya profil: kode UI
	// frontend tidak diukur CC-nya (lihat scanFunctionsInDirectory), cukup ukuran komponennya.
	js := complexityProfiles["typescript"]
	js.label = "JavaScript"
	complexityProfiles["javascript"] = js
}

// cxFrame adalah fungsi yang sedang dihitung; fungsi bersarang punya frame sendiri
type cxFrame struct {
	name       string
	line       int
	count      int // 1 + titik keputusan
	braceBase  int // kedalaman kurawal sebelum deklarasi
	parenBase  int
	indent     int  // Python: indentasi baris def
	bodyOpened bool // kurawal badan fungsi sudah dibuka
	exprBody   bool // badan berupa ekspresi (=> / = expr), berakhir saat baris tidak berlanjut
}

// matchFuncStart mengembalikan nama fungsi bila baris (bersih) adalah deklarasi fungsi
func matchFuncStart(profile complexityProfile, lang string, stripped string) string {
	for _, re := range profile.funcStart {
		m := re.FindStringSubmatch(stripped)
		if m == nil {
			continue
		}
		if lang == "dart" {
			// Grup 1 = tipe kembalian; tolak kata kunci (return foo(...), await bar(...)) & baris pemanggilan
			if dartNotTypeWords[m[1]] || strings.HasSuffix(strings.TrimSpace(stripped), ";") || strings.Contains(stripped[:strings.Index(stripped, "(")], "=") {
				continue
			}
		}
		name := ""
		for _, g := range m[1:] {
			if g != "" {
				name = g
			}
		}
		if name != "" && !isControlKeyword(name) {
			return name
		}
	}
	return ""
}

// isContinuationLine menandai baris yang ekspresinya berlanjut ke baris berikutnya
func isContinuationLine(trimmed string) bool {
	for _, suf := range []string{"=>", "=", "&&", "||", "??", "?", ":", ",", "(", "[", "+", "-", "."} {
		if strings.HasSuffix(trimmed, suf) {
			return true
		}
	}
	return false
}

// checkFunctionComplexity menghitung kompleksitas siklomatis McCabe (1 + titik keputusan) setiap
// fungsi bernama dan menandai yang melewati ambang. Batas fungsi dilacak lewat kedalaman kurawal
// (atau indentasi pada Python), sehingga kode setelah fungsi berakhir — kode tingkat atas, method
// kelas lain — tidak ikut terhitung. String & komentar diabaikan.
func checkFunctionComplexity(content string, targetFile string, lang string, findings *[]models.Finding) {
	profile, ok := complexityProfiles[lang]
	if !ok {
		return
	}
	lines := strings.Split(content, "\n")
	for _, f := range measureFunctions(lines, lang, profile) {
		if f.count <= complexityInfoThreshold {
			continue
		}
		severity := "INFO"
		risk := "sedang–tinggi"
		if f.count > complexityWarnThreshold {
			severity, risk = "WARNING", "sangat tinggi (sulit diuji tuntas)"
		}
		*findings = append(*findings, models.Finding{
			RuleID:       "DEEP-FUNC-HIGH-COMPLEXITY",
			Category:     "code_smell",
			Severity:     severity,
			Title:        fmt.Sprintf("[DEEP] Kompleksitas Siklomatis Tinggi pada Fungsi '%s' (CC=%d)", f.name, f.count),
			Description:  fmt.Sprintf("Fungsi %s '%s' pada baris %d memiliki kompleksitas siklomatis %d (batas %d; risiko %s). Artinya minimal %d jalur independen yang harus diuji agar setiap cabang (if/loop/case/catch/operator logika) tercakup.", profile.label, f.name, f.line, f.count, complexityInfoThreshold, risk, f.count),
			Suggestion:   "Pecah fungsi menjadi sub-fungsi bernama yang masing-masing satu tanggung jawab, gunakan guard clause / early return untuk meratakan if bersarang, dan ganti switch/if berantai berbasis tipe dengan map lookup atau polymorphism.",
			TargetFile:   targetFile,
			Line:         f.line,
			Snippet:      strings.TrimSpace(lines[f.line-1]),
			TheoryBasis:  "McCabe (1976) Cyclomatic Complexity & SEI risk bands (11–20 sedang, 21–50 tinggi, >50 tidak teruji); ambang setara gocyclo -over 15 / ESLint complexity.",
			SuggestedFix: "// Ekstrak blok bersyarat ke helper bernama & gunakan early return:\n// if (!valid) return err\n// return handleByType[kind](input)",
			IsDeep:       true,
			Status:       models.FindingStatusOpen,
		})
	}
}

// FuncComplexity adalah CC satu fungsi bernama (dipakai gerbang kompleksitas orkestrator).
type FuncComplexity struct {
	Name string
	Line int
	CC   int
}

// ComplexityLimit: batas CC per fungsi, sama dengan ambang temuan DEEP-FUNC-HIGH-COMPLEXITY.
const ComplexityLimit = complexityInfoThreshold

// MeasureComplexity mengukur CC tiap fungsi bernama di satu berkas dengan cakupan yang sama dengan audit:
// bahasa tanpa profil, kode UI frontend, berkas test & hasil generator → nil.
func MeasureComplexity(relPath, content string) []FuncComplexity {
	lang := auditFileExtensions[strings.ToLower(filepath.Ext(relPath))]
	profile, ok := complexityProfiles[lang]
	if !ok || isUISourceFile(relPath, lang) || isTestOrGeneratedFile(filepath.Base(relPath)) || isGeneratedContent(content) {
		return nil
	}
	var out []FuncComplexity
	for _, f := range measureFunctions(strings.Split(content, "\n"), lang, profile) {
		out = append(out, FuncComplexity{Name: f.name, Line: f.line, CC: f.count})
	}
	return out
}

// measureFunctions menghitung CC tiap fungsi bernama; urutan hasil = urutan fungsi berakhir (fungsi bersarang dulu).
func measureFunctions(lines []string, lang string, profile complexityProfile) []cxFrame {
	stripped := stripCode(lines, lang)

	var stack []*cxFrame
	var done []cxFrame
	brace, paren := 0, 0

	pop := func() {
		done = append(done, *stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	for idx, s := range stripped {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			continue
		}

		if profile.indent {
			ind := len(s) - len(strings.TrimLeft(s, " \t"))
			for paren == 0 && len(stack) > 0 && ind <= stack[len(stack)-1].indent {
				pop()
			}
			if name := matchFuncStart(profile, lang, s); name != "" {
				stack = append(stack, &cxFrame{name: name, line: idx + 1, count: 1, indent: ind})
			}
			if len(stack) > 0 {
				stack[len(stack)-1].count += len(profile.branch.FindAllStringIndex(s, -1))
			}
			_, dp, _ := bracketDelta(s)
			paren += dp
			if paren < 0 {
				paren = 0
			}
			continue
		}

		pushed := false
		if name := matchFuncStart(profile, lang, s); name != "" {
			// Deklarasi tanpa badan (abstract/interface) tidak dihitung
			if !strings.HasSuffix(trimmed, ";") {
				// Frame sebelumnya yang belum punya badan ternyata hanya deklarasi → buang
				if n := len(stack); n > 0 && !stack[n-1].bodyOpened && !stack[n-1].exprBody {
					stack = stack[:n-1]
				}
				stack = append(stack, &cxFrame{name: name, line: idx + 1, count: 1, braceBase: brace, parenBase: paren})
				pushed = true
			}
		}

		if len(stack) > 0 {
			top := stack[len(stack)-1]
			// Baris deklarasi sendiri tidak dihitung kecuali badan ekspresinya (sesudah =>)
			counted := s
			if pushed {
				if loc := strings.Index(s, "=>"); loc >= 0 {
					counted = s[loc:]
				} else if !strings.Contains(s, "{") {
					counted = ""
				} else {
					counted = s[strings.Index(s, "{"):]
				}
			}
			top.count += len(profile.branch.FindAllStringIndex(counted, -1))
			if !top.bodyOpened && profile.exprBody != nil && profile.exprBody.MatchString(s) {
				top.exprBody = true
			}
		}

		db, dp, maxB := bracketDelta(s)
		for _, f := range stack {
			if !f.bodyOpened && brace+maxB > f.braceBase && !f.exprBody {
				f.bodyOpened = true
			}
		}
		brace += db
		paren += dp
		if brace < 0 {
			brace = 0
		}
		if paren < 0 {
			paren = 0
		}

		for len(stack) > 0 {
			top := stack[len(stack)-1]
			ended := false
			switch {
			case top.bodyOpened:
				ended = brace <= top.braceBase
			case top.exprBody:
				ended = paren <= top.parenBase && brace <= top.braceBase && !isContinuationLine(trimmed)
			}
			if !ended {
				break
			}
			pop()
		}
	}
	for len(stack) > 0 {
		pop()
	}
	return done
}

func isControlKeyword(name string) bool {
	switch name {
	case "if", "for", "while", "switch", "catch", "return", "when", "else", "new", "function", "constructor", "super", "foreach", "elseif":
		return true
	}
	return false
}

// isTestOrGeneratedFile menandai berkas test dan keluaran generator kode berdasarkan nama.
func isTestOrGeneratedFile(base string) bool {
	for _, suf := range []string{
		"_test.go", "_test.dart", "_test.py", ".pb.go", "_gen.go", "_generated.go",
		".g.dart", ".freezed.dart", ".gr.dart", ".mocks.dart", ".pb.dart", ".pbenum.dart", ".pbjson.dart", ".config.dart",
		".min.js", ".d.ts",
	} {
		if strings.HasSuffix(base, suf) {
			return true
		}
	}
	return strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") ||
		strings.HasPrefix(base, "mock_") || strings.HasPrefix(base, "test_") ||
		base == "generated_plugin_registrant.dart" || strings.HasPrefix(base, "GeneratedPluginRegistrant.")
}

// isGeneratedContent mengenali penanda berkas hasil generator (konvensi Go "Code generated ... DO NOT EDIT"
// dan header build_runner/protoc lain) pada beberapa baris pertama.
func isGeneratedContent(content string) bool {
	head := content
	if len(head) > 600 {
		head = head[:600]
	}
	lower := strings.ToLower(head)
	return strings.Contains(lower, "do not edit") || strings.Contains(head, "GENERATED CODE - DO NOT MODIFY") ||
		strings.Contains(head, "@generated")
}
