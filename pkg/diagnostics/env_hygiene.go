package diagnostics

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Konvensi multi env: .env.local (laptop dev) dan satu berkas per server tujuan (.env.server,
// .env.server.<nama>, .env.production, ...) tidak pernah masuk git; yang di-commit hanya template.
// Nama template selain .env.example diterima agar konvensi framework (Symfony .env.dist) tidak jadi temuan palsu.
var envTemplateNames = map[string]bool{".env.example": true, ".env.sample": true, ".env.template": true, ".env.dist": true}

// envIgnoreProbes: nama env yang wajib ter-ignore walau belum ada di disk. ".env.server.server-baru" mewakili
// server berikutnya, jadi pola harus wildcard (.env.*), bukan daftar nama server yang sudah ada.
var envIgnoreProbes = []string{".env", ".env.local", ".env.server", ".env.server.server-baru"}

var envKeyRegex = regexp.MustCompile(`^(?:export\s+)?([A-Za-z_][A-Za-z0-9_.]*)\s*=`)

const envGitignoreBlock = "# env per mesin: hanya template yang di-commit\n.env\n.env.*\n!.env.example"

func isEnvFileName(name string) bool {
	return name == ".env" || strings.HasPrefix(name, ".env.")
}

// envDir memuat hasil pemindaian satu folder: berkas env asli, template, dan label lokasi.
type envDir struct {
	dir, relBase string
	files        []string // berkas env asli (bukan template)
	template     string
	where        string
}

func (e *envDir) rel(name string) string { return filepath.ToSlash(filepath.Join(e.relBase, name)) }

// scanEnvDir membaca isi folder; ok=false bila folder tidak terbaca.
func scanEnvDir(dir, relBase string) (*envDir, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}
	e := &envDir{dir: dir, relBase: relBase}
	for _, ent := range entries {
		name := ent.Name()
		if ent.IsDir() || !isEnvFileName(name) {
			continue
		}
		if envTemplateNames[name] {
			if e.template == "" || name == ".env.example" {
				e.template = name
			}
			continue
		}
		e.files = append(e.files, name)
	}
	e.where = filepath.ToSlash(relBase)
	if e.where == "" || e.where == "." {
		e.where = "root proyek"
	}
	return e, true
}

func (e *envDir) trackedFinding(p string) models.Finding {
	return models.Finding{
		Category:     "security",
		Severity:     "CRITICAL",
		RuleID:       "ENV-FILE-TRACKED",
		Title:        fmt.Sprintf("Berkas env %s ter-commit ke git", p),
		Description:  fmt.Sprintf("'%s' tercatat di git, jadi ikut terunggah ke remote & server lain. Menambah pola di .gitignore saja tidak mengeluarkan berkas yang sudah ter-commit, dan isinya tetap ada di riwayat git.", p),
		Suggestion:   "Keluarkan dari index (git rm --cached), pastikan .gitignore menutup .env & .env.*, lalu ganti (rotate) semua secret di dalamnya karena sudah ada di riwayat. Hanya .env.example yang di-commit.",
		TargetFile:   e.rel(p),
		TheoryBasis:  "The Twelve-Factor App (Factor III: Config) & OWASP Secrets Management Cheat Sheet.",
		SuggestedFix: fmt.Sprintf("git rm --cached %s\n# .gitignore\n%s", p, envGitignoreBlock),
	}
}

func (e *envDir) notIgnoredFinding(n string) models.Finding {
	return models.Finding{
		Category:     "security",
		Severity:     "CRITICAL",
		RuleID:       "ENV-NOT-IGNORED",
		Title:        fmt.Sprintf("Berkas env %s tidak ter-ignore", e.rel(n)),
		Description:  fmt.Sprintf("'%s' ada di disk tetapi tidak cocok dengan pola .gitignore mana pun, jadi commit berikutnya (git add .) akan mengunggah secret-nya.", e.rel(n)),
		Suggestion:   "Tutup semua varian env sekaligus dengan pola wildcard di .gitignore, dan kecualikan hanya .env.example.",
		TargetFile:   e.rel(n),
		TheoryBasis:  "The Twelve-Factor App (Factor III: Config) & OWASP Secrets Management Cheat Sheet.",
		SuggestedFix: envGitignoreBlock,
	}
}

func (e *envDir) ignoreIncompleteFinding(missingProbes []string) models.Finding {
	return models.Finding{
		Category:     "security",
		Severity:     "WARNING",
		RuleID:       "ENV-IGNORE-INCOMPLETE",
		Title:        fmt.Sprintf("Pola .gitignore belum menutup semua varian env di %s", e.where),
		Description:  fmt.Sprintf("Nama env berikut belum ada tetapi akan lolos ke git bila dibuat: %s. Tiap server tujuan butuh berkas env sendiri (.env.server.<nama>), jadi pola harus wildcard, bukan daftar nama server.", strings.Join(missingProbes, ", ")),
		Suggestion:   "Ganti pola env per nama dengan blok wildcard berikut di .gitignore.",
		TargetFile:   e.rel(".gitignore"),
		TheoryBasis:  "The Twelve-Factor App (Factor III: Config).",
		SuggestedFix: envGitignoreBlock,
	}
}

func (e *envDir) exampleIgnoredFinding() models.Finding {
	return models.Finding{
		Category:     "config",
		Severity:     "WARNING",
		RuleID:       "ENV-EXAMPLE-IGNORED",
		Title:        fmt.Sprintf("%s ikut ter-ignore sehingga tidak pernah terunggah", e.rel(e.template)),
		Description:  fmt.Sprintf("Pola .gitignore (biasanya .env* atau .env.*) juga menutup '%s', jadi server & developer baru tidak mendapat daftar variabel yang dibutuhkan.", e.template),
		Suggestion:   fmt.Sprintf("Tambahkan pengecualian !%s setelah pola env di .gitignore, lalu commit template-nya.", e.template),
		TargetFile:   e.rel(".gitignore"),
		SuggestedFix: envGitignoreBlock,
	}
}

// ignoreProbes: berkas env di disk + nama probe yang belum ada + template, untuk dicek ke git check-ignore.
func (e *envDir) ignoreProbes() (probes []string, onDisk map[string]bool) {
	probes = append([]string{}, e.files...)
	onDisk = map[string]bool{}
	for _, n := range e.files {
		onDisk[n] = true
	}
	for _, n := range envIgnoreProbes {
		if !onDisk[n] {
			probes = append(probes, n)
		}
	}
	if e.template != "" {
		probes = append(probes, e.template)
	}
	return probes, onDisk
}

// gitFindings: temuan yang bergantung pada status git (tracked, ignore, template ter-ignore).
func (e *envDir) gitFindings() []models.Finding {
	var findings []models.Finding
	tracked := trackedEnvFiles(e.dir)
	isTracked := map[string]bool{}
	for _, p := range tracked {
		findings = append(findings, e.trackedFinding(p))
		isTracked[p] = true
	}

	probes, onDisk := e.ignoreProbes()
	ignored := gitIgnored(e.dir, probes)

	for _, n := range e.files {
		if !ignored[n] && !isTracked[n] {
			findings = append(findings, e.notIgnoredFinding(n))
		}
	}
	// Probe hanya bermakna bila folder ini memang memakai env (ada berkas env atau template)
	var missingProbes []string
	if len(e.files) > 0 || e.template != "" {
		for _, n := range envIgnoreProbes {
			if !onDisk[n] && !ignored[n] {
				missingProbes = append(missingProbes, n)
			}
		}
	}
	if len(missingProbes) > 0 {
		findings = append(findings, e.ignoreIncompleteFinding(missingProbes))
	}
	if e.template != "" && ignored[e.template] && !isTracked[e.template] {
		findings = append(findings, e.exampleIgnoredFinding())
	}
	return findings
}

func (e *envDir) exampleMissingFinding() models.Finding {
	return models.Finding{
		Category:     "config",
		Severity:     "WARNING",
		RuleID:       "ENV-EXAMPLE-MISSING",
		Title:        fmt.Sprintf("Template .env.example belum ada di %s", e.where),
		Description:  fmt.Sprintf("Ada berkas env (%s) tetapi tidak ada .env.example yang di-commit. Saat aplikasi diunggah ke server baru, tidak ada daftar variabel yang harus diisi.", strings.Join(e.files, ", ")),
		Suggestion:   "Buat .env.example berisi semua kunci dengan nilai dummy/kosong (tanpa secret asli) dan commit. Setiap server cukup menyalinnya menjadi .env.server.<nama>.",
		TargetFile:   e.rel(".env.example"),
		TheoryBasis:  "The Twelve-Factor App (Factor III: Config).",
		SuggestedFix: "# .env.example\nPORT=8080\nDATABASE_URL=\nSTORAGE_DIR=/var/lib/nama-app/storage",
	}
}

// driftFinding: kunci yang dipakai di env asli tetapi tidak didokumentasikan di template: server baru yang
// disiapkan dari template akan kekurangan variabel. Yang dilaporkan hanya nama kunci, tidak pernah nilainya.
func (e *envDir) driftFinding() *models.Finding {
	documented := envKeys(filepath.Join(e.dir, e.template))
	missing := map[string][]string{}
	for _, n := range e.files {
		for k := range envKeys(filepath.Join(e.dir, n)) {
			if !documented[k] {
				missing[k] = append(missing[k], n)
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	keys := make([]string, 0, len(missing))
	for k := range missing {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s (%s)", k, strings.Join(missing[k], ", ")))
	}
	return &models.Finding{
		Category:     "config",
		Severity:     "WARNING",
		RuleID:       "ENV-EXAMPLE-DRIFT",
		Title:        fmt.Sprintf("%s tidak memuat %d kunci yang dipakai berkas env", e.rel(e.template), len(keys)),
		Description:  fmt.Sprintf("Kunci berikut ada di berkas env tetapi tidak ada di %s: %s. Server baru yang disiapkan dari template akan kekurangan konfigurasi.", e.template, strings.Join(parts, "; ")),
		Suggestion:   fmt.Sprintf("Tambahkan kunci tersebut ke %s dengan nilai dummy/kosong (jangan salin nilai asli).", e.template),
		TargetFile:   e.rel(e.template),
		TheoryBasis:  "The Twelve-Factor App (Factor III: Config).",
		SuggestedFix: strings.Join(keys, "=\n") + "=",
	}
}

// inspectEnvHygiene memeriksa berkas env di satu folder (root proyek / komponen). Status ignore & tracked
// dibaca dari git sendiri (check-ignore / ls-files), jadi wildcard, negasi (!) dan .gitignore bertingkat
// dihitung persis seperti git. Folder di luar repo git hanya diperiksa template & kelengkapan kuncinya.
func inspectEnvHygiene(dir, relBase string) []models.Finding {
	e, ok := scanEnvDir(dir, relBase)
	if !ok {
		return nil
	}
	var findings []models.Finding
	if isGitWorkTree(dir) {
		findings = e.gitFindings()
	}
	if len(e.files) == 0 {
		return findings
	}
	if e.template == "" {
		return append(findings, e.exampleMissingFinding())
	}
	if f := e.driftFinding(); f != nil {
		findings = append(findings, *f)
	}
	return findings
}

// envKeys membaca nama kunci (tanpa nilai) dari berkas env.
func envKeys(path string) map[string]bool {
	keys := map[string]bool{}
	f, err := os.Open(path)
	if err != nil {
		return keys
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := envKeyRegex.FindStringSubmatch(strings.TrimSpace(sc.Text())); m != nil {
			keys[m[1]] = true
		}
	}
	return keys
}

func isGitWorkTree(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// trackedEnvFiles: berkas env (bukan template) di bawah dir yang tercatat di index git, relatif ke dir.
func trackedEnvFiles(dir string) []string {
	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, p := range strings.Split(string(out), "\x00") {
		base := filepath.Base(p)
		if p != "" && isEnvFileName(base) && !envTemplateNames[base] {
			files = append(files, p)
		}
	}
	return files
}

// gitIgnored mengembalikan nama (relatif ke dir) yang cocok dengan pola ignore; --no-index agar status
// tracked tidak memengaruhi hasil, sehingga yang dinilai murni pola .gitignore.
func gitIgnored(dir string, names []string) map[string]bool {
	res := map[string]bool{}
	if len(names) == 0 {
		return res
	}
	cmd := exec.Command("git", append([]string{"check-ignore", "--no-index", "--"}, names...)...)
	cmd.Dir = dir
	out, _ := cmd.Output() // exit 1 = tidak ada yang ter-ignore
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			res[line] = true
		}
	}
	return res
}
