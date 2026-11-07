package diagnostics

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Default proyek vibeasik (Aturan 45): dev.sh menjalankan semua komponen sekaligus, deploy.sh memasang
// .env.<tujuan> lalu selalu mengembalikan .env.local, FE/mobile menunjuk port BE yang benar saat dev,
// dan tidak ada duplicate import (warning Nuxt/unimport maupun import ganda dalam satu berkas).

// effectivePort = port yang benar-benar dipakai saat dev: run_port vibeasik, atau port deklarasi kode.
func effectivePort(c models.Component) int {
	if c.RunPort > 0 {
		return c.RunPort
	}
	return c.Port
}

func isMobileComponent(c models.Component) bool {
	fw := strings.ToLower(c.Framework)
	return c.Role == models.RoleFrontendMobile || c.Language == "dart" ||
		strings.Contains(fw, "flutter") || strings.Contains(fw, "react-native") || strings.Contains(fw, "expo")
}

// inspectProjectTooling memeriksa skrip wajib di root proyek: dev.sh & deploy.sh.
func inspectProjectTooling(project *models.Project) []models.Finding {
	root := project.RootPath
	if root == "" || root == "-" {
		return nil
	}
	var findings []models.Finding

	devPath := filepath.Join(root, "dev.sh")
	if info, err := os.Stat(devPath); err != nil {
		findings = append(findings, models.Finding{
			Category:     "deployment",
			Severity:     "WARNING",
			RuleID:       "DEV-SCRIPT-MISSING",
			Title:        "dev.sh Belum Ada di Root Proyek",
			Description:  "Proyek wajib punya dev.sh di root yang menyalakan semua komponen (BE, FE, mobile) dengan satu perintah, memakai port dari blok proyek di PORTS.md dan env FE→BE yang menunjuk port BE.",
			Suggestion:   "Buat dev.sh: set PORT tiap komponen dari blok PORTS.md, ekspor URL API BE ke FE/mobile, jalankan semua komponen paralel, dan hentikan semuanya dengan trap saat Ctrl+C.",
			TargetFile:   "dev.sh",
			TheoryBasis:  "ARCH-002 Unified Local Orchestration (Dev Runner).",
			SuggestedFix: devScriptTemplate,
		})
	} else {
		content := readSmallFile(devPath)
		if info.Mode()&0o111 == 0 {
			findings = append(findings, models.Finding{
				Category:    "deployment",
				Severity:    "INFO",
				RuleID:      "DEV-SCRIPT-NOT-EXECUTABLE",
				Title:       "dev.sh Belum Executable",
				Description: "dev.sh ada tetapi tidak punya izin eksekusi, sehingga ./dev.sh gagal.",
				Suggestion:  "Jalankan chmod +x dev.sh lalu commit (git menyimpan bit eksekusi).",
				TargetFile:  "dev.sh",
			})
		}
		var missing []string
		for _, c := range project.Components {
			rel := strings.Trim(filepath.ToSlash(c.RelativePath), "/.")
			if rel == "" {
				continue
			}
			if !strings.Contains(content, rel) && !strings.Contains(content, filepath.Base(rel)) {
				missing = append(missing, fmt.Sprintf("%s (%s)", c.Name, rel))
			}
		}
		if len(missing) > 0 {
			findings = append(findings, models.Finding{
				Category:    "deployment",
				Severity:    "WARNING",
				RuleID:      "DEV-SCRIPT-INCOMPLETE",
				Title:       "dev.sh Tidak Menjalankan Semua Komponen",
				Description: fmt.Sprintf("dev.sh tidak menyebut folder komponen: %s. Semua komponen proyek wajib berjalan dari dev.sh.", strings.Join(missing, ", ")),
				Suggestion:  "Tambahkan perintah dev komponen tersebut ke dev.sh (cd ke foldernya, PORT dari blok PORTS.md).",
				TargetFile:  "dev.sh",
			})
		}
	}

	findings = append(findings, inspectMobileDevScript(project)...)

	deployPath := filepath.Join(root, "deploy.sh")
	if !fileExists(deployPath) {
		findings = append(findings, models.Finding{
			Category:     "deployment",
			Severity:     "WARNING",
			RuleID:       "DEPLOY-SCRIPT-MISSING",
			Title:        "deploy.sh Belum Ada di Root Proyek",
			Description:  "Proyek wajib punya deploy.sh: pengguna cukup mengisi .env.server (atau .env.<nama> pilihannya), skrip memasangnya sebagai env aktif, build & upload ke server, lalu mengembalikan .env.local walau deploy gagal.",
			Suggestion:   "Buat deploy.sh dengan argumen nama env (default server), salin .env.<nama> ke berkas env aktif, trap EXIT untuk memulihkan .env.local, dan pastikan .gitignore memuat .env / .env.* / !.env.example.",
			TargetFile:   "deploy.sh",
			TheoryBasis:  "SEC-002 & DEV-007: hanya .env.example yang di-commit; env server dipasang saat deploy saja.",
			SuggestedFix: deployScriptTemplate,
		})
		return findings
	}
	content := readSmallFile(deployPath)
	if !strings.Contains(content, ".env.local") || !strings.Contains(content, "trap") {
		findings = append(findings, models.Finding{
			Category:    "deployment",
			Severity:    "WARNING",
			RuleID:      "DEPLOY-ENV-NOT-RESTORED",
			Title:       "deploy.sh Tidak Mengembalikan .env.local",
			Description: "deploy.sh wajib mengembalikan env aktif ke .env.local setelah deploy, termasuk saat deploy gagal (trap EXIT). Tanpa itu laptop dev tertinggal memakai env server.",
			Suggestion:  "Tambahkan trap 'cp .env.local .env' EXIT (per folder komponen) sebelum menyalin .env.<tujuan>.",
			TargetFile:  "deploy.sh",
		})
	}
	if !deployEnvSelectRe.MatchString(content) {
		findings = append(findings, models.Finding{
			Category:    "deployment",
			Severity:    "INFO",
			RuleID:      "DEPLOY-ENV-NOT-SELECTABLE",
			Title:       "deploy.sh Tidak Memakai .env.<tujuan>",
			Description: "deploy.sh tidak terlihat memasang .env.server / .env.<nama> pilihan pengguna, sehingga konfigurasi server tidak cukup diatur lewat satu berkas env.",
			Suggestion:  "Terima argumen nama env (./deploy.sh server-baru → .env.server-baru, default .env.server) lalu salin sebagai env aktif selama build & upload.",
			TargetFile:  "deploy.sh",
		})
	}
	return findings
}

// isFlutterComponent: komponen mobile Flutter, yang dijalankan ke HP lewat dev-mobile.sh (flutter-win).
func isFlutterComponent(c models.Component) bool {
	return isMobileComponent(c) && (c.Language == "dart" || strings.Contains(strings.ToLower(c.Framework), "flutter"))
}

// inspectMobileDevScript (Aturan 49): proyek dengan komponen Flutter wajib punya dev-mobile.sh di root yang
// menjalankan aplikasi ke HP/emulator di PC Windows (flutter-win, default) atau laptop, dengan API_BASE_URL ke
// IP LAN laptop. Patokan: nemudijalan/dev-mobile.sh.
func inspectMobileDevScript(project *models.Project) []models.Finding {
	var mobile *models.Component
	for i := range project.Components {
		if isFlutterComponent(project.Components[i]) {
			mobile = &project.Components[i]
			break
		}
	}
	if mobile == nil {
		return nil
	}
	path := filepath.Join(project.RootPath, "dev-mobile.sh")
	info, err := os.Stat(path)
	if err != nil {
		return []models.Finding{{
			Category:     "deployment",
			Severity:     "WARNING",
			RuleID:       "MOBILE-DEV-SCRIPT-MISSING",
			Title:        "dev-mobile.sh Belum Ada di Root Proyek",
			Description:  "Proyek dengan aplikasi mobile Flutter wajib punya dev-mobile.sh di root agar aplikasi bisa langsung dijalankan ke HP: default di PC Windows lewat flutter-win (sync kode via SSH lalu flutter run), atau './dev-mobile.sh laptop', dengan API_BASE_URL menunjuk IP LAN laptop dan port BE dari PORTS.md.",
			Suggestion:   "Salin contoh di bawah ke dev-mobile.sh (chmod +x), sesuaikan folder mobile, port BE, path API & berkas env dev Flutter bila berbeda, lalu commit.",
			TargetFile:   "dev-mobile.sh",
			TheoryBasis:  "ARCH-002 Unified Local Orchestration (Dev Runner), patokan nemudijalan/dev-mobile.sh.",
			SuggestedFix: mobileDevScript(project, mobile),
		}}
	}
	var findings []models.Finding
	if info.Mode()&0o111 == 0 {
		findings = append(findings, models.Finding{
			Category:    "deployment",
			Severity:    "INFO",
			RuleID:      "MOBILE-DEV-SCRIPT-NOT-EXECUTABLE",
			Title:       "dev-mobile.sh Belum Executable",
			Description: "dev-mobile.sh ada tetapi tidak punya izin eksekusi, sehingga ./dev-mobile.sh gagal.",
			Suggestion:  "Jalankan chmod +x dev-mobile.sh lalu commit (git menyimpan bit eksekusi).",
			TargetFile:  "dev-mobile.sh",
		})
	}
	content := readSmallFile(path)
	rel := strings.Trim(filepath.ToSlash(mobile.RelativePath), "/.")
	var missing []string
	for _, want := range []string{rel, "flutter-win", "API_BASE_URL"} {
		if want != "" && !strings.Contains(content, want) {
			missing = append(missing, want)
		}
	}
	if len(missing) > 0 {
		findings = append(findings, models.Finding{
			Category:     "deployment",
			Severity:     "WARNING",
			RuleID:       "MOBILE-DEV-SCRIPT-INCOMPLETE",
			Title:        "dev-mobile.sh Tidak Sesuai Standar",
			Description:  fmt.Sprintf("dev-mobile.sh tidak menyebut: %s. Skrip wajib menjalankan folder mobile lewat flutter-win (PC Windows, default) atau laptop, dengan API_BASE_URL ke IP LAN laptop.", strings.Join(missing, ", ")),
			Suggestion:   "Samakan dengan contoh standar (patokan nemudijalan/dev-mobile.sh).",
			TargetFile:   "dev-mobile.sh",
			SuggestedFix: mobileDevScript(project, mobile),
		})
	}
	return findings
}

//go:embed dev_mobile.sh.tmpl
var mobileDevScriptTemplate string

var nonSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

// mobileDevScript mengisi template dev-mobile.sh: folder mobile, port BE proyek, & folder unik di PC Windows.
func mobileDevScript(project *models.Project, mobile *models.Component) string {
	rel := strings.Trim(filepath.ToSlash(mobile.RelativePath), "/.")
	if rel == "" {
		rel = "mobile"
	}
	bePort := 4105
	for _, c := range project.Components {
		if c.Role == models.RoleBackend && effectivePort(c) > 0 {
			bePort = effectivePort(c)
			break
		}
	}
	remote := strings.Trim(nonSlugRe.ReplaceAllString(strings.ToLower(project.Name), "_"), "_")
	if remote == "" {
		remote = "app"
	}
	return strings.NewReplacer("__MOBILE_DIR__", rel, "__BE_PORT__", strconv.Itoa(bePort),
		"__REMOTE_NAME__", remote+"_"+filepath.Base(rel)).Replace(mobileDevScriptTemplate)
}

var deployEnvSelectRe = regexp.MustCompile(`\.env\.(server|production|prod|\$|"\$)`)

func readSmallFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil || len(b) > 256*1024 {
		return ""
	}
	return string(b)
}

// Port lintas proyek: setiap proyek memakai blok PORTS.md sendiri, jadi port sama = bentrok saat dijalankan serentak.
func crossProjectPortFindings(project *models.Project, allProjects []models.Project) []models.Finding {
	var findings []models.Finding
	reported := map[int]bool{}
	for _, comp := range project.Components {
		port := effectivePort(comp)
		if port <= 0 || reported[port] {
			continue
		}
		var shared []string
		for _, other := range allProjects {
			if other.ID == project.ID {
				continue
			}
			for _, oc := range other.Components {
				if effectivePort(oc) == port {
					shared = append(shared, fmt.Sprintf("'%s' (%s)", other.Name, oc.Name))
				}
			}
		}
		if len(shared) == 0 {
			continue
		}
		reported[port] = true
		findings = append(findings, models.Finding{
			ID:          fmt.Sprintf("shared_port_%d", port),
			RuleID:      fmt.Sprintf("PORT-SHARED-%d", port),
			Category:    "port",
			Severity:    "WARNING",
			Title:       fmt.Sprintf("Port Bentrok Antar Proyek: :%d", port),
			Description: fmt.Sprintf("Komponen '%s' memakai port %d yang juga dipakai %s. Proyek tidak bisa dijalankan serentak (address already in use).", comp.Name, port, strings.Join(shared, ", ")),
			Suggestion:  "Pindahkan ke port blok proyek di PORTS.md (Aturan 35) lewat vibeasik_assign_port, perbarui env PORT, dev.sh, dan URL API FE/mobile dalam satu commit.",
			TargetFile:  comp.Name,
			Line:        port,
			TheoryBasis: "NET-001 Alokasi Port per Blok Proyek.",
		})
	}
	return findings
}

var (
	envURLLineRe = regexp.MustCompile(`^(?:export\s+)?([A-Za-z_][A-Za-z0-9_.]*)\s*=\s*["']?(https?://[^\s"']+)`)
	localURLRe   = regexp.MustCompile(`https?://(localhost|127\.0\.0\.1|0\.0\.0\.0)(?::(\d+))?`)
	// kunci URL BE milik proyek sendiri (NUXT_PUBLIC_API_BASE, VITE_API_URL, API_BASE_URL, BACKEND_URL, ...);
	// URL layanan luar (OLLAMA_BASE_URL, SYSTEM_ONE_URL) sengaja tidak cocok
	apiKeyHint = regexp.MustCompile(`(?i)^(?:[A-Z]+_PUBLIC_|VITE_|REACT_APP_|PUBLIC_)?(?:API|BACKEND|BE|SERVER)(?:_BASE)?(?:_URL|_HOST|_ORIGIN)?$`)
)

// inspectFrontendBackendLink memastikan FE & mobile memuat data BE yang benar saat dev: URL API di env
// menunjuk port BE proyek ini, dan aplikasi mobile tidak memakai localhost (emulator/HP tidak bisa menjangkaunya).
// only != 0 membatasi pemeriksaan ke satu komponen (AnalyzeComponent).
func inspectFrontendBackendLink(project *models.Project, only uint) []models.Finding {
	bePorts := map[int]bool{}
	var bePortList []string
	for _, c := range project.Components {
		if c.Role == models.RoleBackend && effectivePort(c) > 0 {
			bePorts[effectivePort(c)] = true
			bePortList = append(bePortList, strconv.Itoa(effectivePort(c)))
		}
	}
	var findings []models.Finding
	for _, c := range project.Components {
		if (only != 0 && c.ID != only) || c.Role == models.RoleBackend || (!strings.HasPrefix(c.Role, "frontend") && !isMobileComponent(c)) {
			continue
		}
		mobile := isMobileComponent(c)
		for _, envName := range []string{".env.local", ".env", ".env.development"} {
			path := filepath.Join(c.AbsolutePath, envName)
			rel := filepath.ToSlash(filepath.Join(c.RelativePath, envName))
			for _, e := range envURLs(path) {
				lineNo, key, url := e.line, e.key, e.url
				if !apiKeyHint.MatchString(key) {
					continue
				}
				m := localURLRe.FindStringSubmatch(url)
				if m == nil {
					continue
				}
				if mobile {
					findings = append(findings, mobileLocalhostFinding(c, rel, lineNo, key+"="+url))
				}
				if port, _ := strconv.Atoi(m[2]); port > 0 && len(bePorts) > 0 && !bePorts[port] {
					findings = append(findings, models.Finding{
						Category:    "architecture",
						Severity:    "WARNING",
						RuleID:      "FE-API-PORT-MISMATCH",
						Title:       fmt.Sprintf("URL API '%s' Tidak Menunjuk Port Backend", c.Name),
						Description: fmt.Sprintf("%s di %s menunjuk port %d, sedangkan backend proyek berjalan di port %s. Saat dev, data BE gagal dimuat.", key, rel, port, strings.Join(bePortList, "/")),
						Suggestion:  "Samakan URL API dengan port BE dari blok PORTS.md, dan biarkan dev.sh mengekspor URL ini agar selalu sinkron dengan PORT backend.",
						TargetFile:  rel,
						Line:        lineNo,
						Snippet:     key + "=" + url,
					})
				}
			}
		}
		if mobile {
			findings = append(findings, mobileSourceLocalhost(c)...)
		}
	}
	return findings
}

type envURL struct {
	line     int
	key, url string
}

// envURLs mengembalikan setiap baris env bernilai URL.
func envURLs(path string) []envURL {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []envURL
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		if m := envURLLineRe.FindStringSubmatch(strings.TrimSpace(sc.Text())); m != nil {
			out = append(out, envURL{n, m[1], m[2]})
		}
	}
	return out
}

func mobileLocalhostFinding(c models.Component, rel string, line int, snippet string) models.Finding {
	return models.Finding{
		Category:     "mobile",
		Severity:     "WARNING",
		RuleID:       "MOBILE-API-LOCALHOST",
		Title:        fmt.Sprintf("Aplikasi Mobile '%s' Memanggil BE lewat localhost", c.Name),
		Description:  "Di emulator Android localhost adalah emulator itu sendiri, dan di HP fisik adalah HP itu, sehingga data BE tidak pernah termuat saat dev.",
		Suggestion:   "Ambil URL API dari env/--dart-define yang diisi dev.sh dengan IP LAN laptop (atau 10.0.2.2 khusus emulator Android); pastikan BE listen di 0.0.0.0 (NET-002) dan izinkan cleartext HTTP hanya untuk build debug.",
		TargetFile:   rel,
		Line:         line,
		Snippet:      snippet,
		SuggestedFix: "const apiBase = String.fromEnvironment('API_BASE_URL', defaultValue: 'http://10.0.2.2:4105');\n// dev.sh: flutter run --dart-define=API_BASE_URL=http://$LAN_IP:$BE_PORT",
	}
}

// mobileSourceLocalhost mencari URL localhost tertanam di kode mobile (lib/, src/, app/), maksimal satu temuan per berkas.
func mobileSourceLocalhost(c models.Component) []models.Finding {
	var findings []models.Finding
	scanned := 0
	for _, sub := range []string{"lib", "src", "app"} {
		dir := filepath.Join(c.AbsolutePath, sub)
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if ignoredDirNames[d.Name()] || strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".dart" && ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
				return nil
			}
			if scanned++; scanned > 600 || len(findings) >= 5 {
				return filepath.SkipAll
			}
			for i, line := range strings.Split(readSmallFile(path), "\n") {
				trim := strings.TrimSpace(line)
				if strings.HasPrefix(trim, "//") || !localURLRe.MatchString(trim) {
					continue
				}
				rel, _ := filepath.Rel(c.AbsolutePath, path)
				findings = append(findings, mobileLocalhostFinding(c, filepath.ToSlash(filepath.Join(c.RelativePath, rel)), i+1, trim))
				break
			}
			return nil
		})
	}
	return findings
}

var (
	// import ... from 'x' / import 'x' (baris tunggal), dipisah antara import type dan import nilai
	importFromRe = regexp.MustCompile(`^\s*import\s+(type\s+)?(?:[^'";]*?\s+from\s+)?['"]([^'"]+)['"]`)
	// ekspor bernama tingkat atas yang dibaca unimport (auto-import Nuxt) dari composables/ & utils/
	exportNameRe = regexp.MustCompile(`^export\s+(?:async\s+)?(?:const|let|var|function\*?|class)\s+([A-Za-z_$][\w$]*)`)
)

// inspectDuplicateImports: modul yang sama diimpor lebih dari sekali dalam satu berkas JS/TS/Vue.
func inspectDuplicateImports(relPath string, lines []string) []models.Finding {
	seen := map[string]int{}
	var findings []models.Finding
	for i, line := range lines {
		m := importFromRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key := m[1] + m[2]
		first, dup := seen[key]
		if !dup {
			seen[key] = i + 1
			continue
		}
		findings = append(findings, models.Finding{
			Category:     "frontend",
			Severity:     "WARNING",
			RuleID:       "FE-DUPLICATE-IMPORT",
			Title:        fmt.Sprintf("Duplicate Import: '%s'", m[2]),
			Description:  fmt.Sprintf("'%s' diimpor lagi di baris %d padahal sudah diimpor di baris %d.", m[2], i+1, first),
			Suggestion:   "Gabungkan menjadi satu pernyataan import untuk modul tersebut.",
			TargetFile:   relPath,
			Line:         i + 1,
			Snippet:      strings.TrimSpace(line),
			SuggestedFix: "import { a, b } from 'modul'",
		})
	}
	return findings
}

// inspectNuxtDuplicateAutoImports meniru warning unimport "Duplicated imports" pada Nuxt: nama yang sama
// diekspor dari lebih dari satu berkas di composables/ atau utils/, sehingga salah satunya diabaikan diam-diam.
func inspectNuxtDuplicateAutoImports(c models.Component) []models.Finding {
	isNuxt := strings.Contains(strings.ToLower(c.Framework), "nuxt") ||
		fileExists(filepath.Join(c.AbsolutePath, "nuxt.config.ts")) || fileExists(filepath.Join(c.AbsolutePath, "nuxt.config.js"))
	if !isNuxt {
		return nil
	}
	owners := map[string][]string{}
	for _, dirRel := range []string{"composables", "utils", "app/composables", "app/utils", "shared/utils"} {
		dir := filepath.Join(c.AbsolutePath, dirRel)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			ext := filepath.Ext(e.Name())
			if e.IsDir() || (ext != ".ts" && ext != ".js") || strings.Contains(e.Name(), ".test.") || strings.Contains(e.Name(), ".spec.") {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(c.RelativePath, dirRel, e.Name()))
			names := map[string]bool{}
			for _, line := range strings.Split(readSmallFile(filepath.Join(dir, e.Name())), "\n") {
				if m := exportNameRe.FindStringSubmatch(line); m != nil {
					names[m[1]] = true
				} else if strings.HasPrefix(line, "export default") {
					names[camelFromFile(e.Name())] = true
				}
			}
			for n := range names {
				owners[n] = append(owners[n], rel)
			}
		}
	}
	for n, files := range owners {
		if builtinAutoImports[n] {
			owners[n] = append(files, "#imports (bawaan Nuxt/Vue)")
		}
	}
	var names []string
	for n, files := range owners {
		if len(files) > 1 {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	var findings []models.Finding
	for _, n := range names {
		files := owners[n]
		sort.Strings(files)
		findings = append(findings, models.Finding{
			Category:    "frontend",
			Severity:    "WARNING",
			RuleID:      "FE-DUPLICATE-IMPORT",
			Title:       fmt.Sprintf("Duplicated Auto-Import Nuxt: '%s'", n),
			Description: fmt.Sprintf("'%s' diekspor dari %s. Nuxt (unimport) menampilkan WARN Duplicated imports dan hanya memakai salah satunya, jadi kode bisa memanggil fungsi yang salah.", n, strings.Join(files, " dan ")),
			Suggestion:  "Sisakan satu definisi (atau ganti nama salah satunya) lalu perbarui pemanggilnya; warning dev server harus bersih.",
			TargetFile:  userFile(files),
		})
	}
	return findings
}

// builtinAutoImports: auto-import bawaan Nuxt/Vue/Nuxt UI yang sering tertimpa composable buatan sendiri.
var builtinAutoImports = map[string]bool{
	"ref": true, "computed": true, "reactive": true, "watch": true, "watchEffect": true, "nextTick": true,
	"onMounted": true, "onUnmounted": true, "onBeforeUnmount": true, "toRef": true, "toRefs": true,
	"useFetch": true, "useLazyFetch": true, "useAsyncData": true, "useLazyAsyncData": true, "useState": true,
	"useRoute": true, "useRouter": true, "navigateTo": true, "useHead": true, "useSeoMeta": true,
	"useCookie": true, "useRuntimeConfig": true, "useAppConfig": true, "useNuxtApp": true, "useError": true,
	"createError": true, "showError": true, "clearError": true, "useRequestHeaders": true, "definePageMeta": true,
	"defineNuxtPlugin": true, "defineNuxtRouteMiddleware": true, "useToast": true, "useColorMode": true,
}

func userFile(files []string) string {
	for _, f := range files {
		if !strings.HasPrefix(f, "#") {
			return f
		}
	}
	return files[0]
}

func camelFromFile(name string) string {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	parts := strings.FieldsFunc(stem, func(r rune) bool { return r == '-' || r == '_' || r == '.' })
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

const devScriptTemplate = `#!/usr/bin/env bash
# Jalankan semua komponen proyek sekaligus. Port dari blok PORTS.md (Aturan 35).
set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BE_PORT="${BE_PORT:-4105}"; FE_PORT="${FE_PORT:-4100}"
LAN_IP="$(hostname -I 2>/dev/null | awk '{print $1}')"; LAN_IP="${LAN_IP:-127.0.0.1}"
trap 'kill 0' EXIT INT TERM
(cd "$ROOT/backend" && PORT=$BE_PORT go run .) &
(cd "$ROOT/frontend" && PORT=$FE_PORT NUXT_PUBLIC_API_BASE=http://localhost:$BE_PORT bun run dev) &
# (cd "$ROOT/mobile" && flutter run --dart-define=API_BASE_URL=http://$LAN_IP:$BE_PORT) &
wait`

const deployScriptTemplate = `#!/usr/bin/env bash
# ./deploy.sh [nama-env]  → pakai .env.<nama-env> (default: server), lalu kembali ke .env.local.
set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="${1:-server}"
DIRS=(backend frontend)
for d in "${DIRS[@]}"; do
  [ -f "$ROOT/$d/.env.$TARGET" ] || { echo "Isi dulu $d/.env.$TARGET (salin dari .env.example)"; exit 1; }
done
# deploy pertama: simpan env dev sekarang sebagai .env.local agar tidak pernah hilang
for d in "${DIRS[@]}"; do [ -f "$ROOT/$d/.env.local" ] || [ ! -f "$ROOT/$d/.env" ] || cp "$ROOT/$d/.env" "$ROOT/$d/.env.local"; done
restore() { for d in "${DIRS[@]}"; do [ ! -f "$ROOT/$d/.env.local" ] || cp "$ROOT/$d/.env.local" "$ROOT/$d/.env"; done; }
trap restore EXIT
for d in "${DIRS[@]}"; do cp "$ROOT/$d/.env.$TARGET" "$ROOT/$d/.env"; done
# build + upload; host & path server dibaca dari .env.$TARGET (DEPLOY_HOST, DEPLOY_PATH)
set -a; . "$ROOT/backend/.env.$TARGET"; set +a
(cd "$ROOT/backend" && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o app .)
(cd "$ROOT/frontend" && bun run build)
rsync -az --delete "$ROOT/backend/app" "$ROOT/frontend/.output" "$DEPLOY_HOST:$DEPLOY_PATH/"`
