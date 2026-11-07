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

// inspectEnvHygiene memeriksa berkas env di satu folder (root proyek / komponen). Status ignore & tracked
// dibaca dari git sendiri (check-ignore / ls-files), jadi wildcard, negasi (!) dan .gitignore bertingkat
// dihitung persis seperti git. Folder di luar repo git hanya diperiksa template & kelengkapan kuncinya.
func inspectEnvHygiene(dir, relBase string) []models.Finding {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var envFiles []string // berkas env asli (bukan template) di folder ini
	template := ""
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !isEnvFileName(name) {
			continue
		}
		if envTemplateNames[name] {
			if template == "" || name == ".env.example" {
				template = name
			}
			continue
		}
		envFiles = append(envFiles, name)
	}
	rel := func(name string) string { return filepath.ToSlash(filepath.Join(relBase, name)) }
	where := filepath.ToSlash(relBase)
	if where == "" || where == "." {
		where = "root proyek"
	}
	var findings []models.Finding

	if isGitWorkTree(dir) {
		tracked := trackedEnvFiles(dir)
		for _, p := range tracked {
			findings = append(findings, models.Finding{
				Category:     "security",
				Severity:     "CRITICAL",
				RuleID:       "ENV-FILE-TRACKED",
				Title:        fmt.Sprintf("Berkas env %s ter-commit ke git", p),
				Description:  fmt.Sprintf("'%s' tercatat di git, jadi ikut terunggah ke remote & server lain. Menambah pola di .gitignore saja tidak mengeluarkan berkas yang sudah ter-commit, dan isinya tetap ada di riwayat git.", p),
				Suggestion:   "Keluarkan dari index (git rm --cached), pastikan .gitignore menutup .env & .env.*, lalu ganti (rotate) semua secret di dalamnya karena sudah ada di riwayat. Hanya .env.example yang di-commit.",
				TargetFile:   rel(p),
				TheoryBasis:  "The Twelve-Factor App (Factor III: Config) & OWASP Secrets Management Cheat Sheet.",
				SuggestedFix: fmt.Sprintf("git rm --cached %s\n# .gitignore\n%s", p, envGitignoreBlock),
			})
		}
		isTracked := map[string]bool{}
		for _, p := range tracked {
			isTracked[p] = true
		}

		probes := append([]string{}, envFiles...)
		onDisk := map[string]bool{}
		for _, n := range envFiles {
			onDisk[n] = true
		}
		for _, n := range envIgnoreProbes {
			if !onDisk[n] {
				probes = append(probes, n)
			}
		}
		if template != "" {
			probes = append(probes, template)
		}
		ignored := gitIgnored(dir, probes)

		var missingProbes []string
		for _, n := range envFiles {
			if !ignored[n] && !isTracked[n] {
				findings = append(findings, models.Finding{
					Category:     "security",
					Severity:     "CRITICAL",
					RuleID:       "ENV-NOT-IGNORED",
					Title:        fmt.Sprintf("Berkas env %s tidak ter-ignore", rel(n)),
					Description:  fmt.Sprintf("'%s' ada di disk tetapi tidak cocok dengan pola .gitignore mana pun, jadi commit berikutnya (git add .) akan mengunggah secret-nya.", rel(n)),
					Suggestion:   "Tutup semua varian env sekaligus dengan pola wildcard di .gitignore, dan kecualikan hanya .env.example.",
					TargetFile:   rel(n),
					TheoryBasis:  "The Twelve-Factor App (Factor III: Config) & OWASP Secrets Management Cheat Sheet.",
					SuggestedFix: envGitignoreBlock,
				})
			}
		}
		// Probe hanya bermakna bila folder ini memang memakai env (ada berkas env atau template)
		if len(envFiles) > 0 || template != "" {
			for _, n := range envIgnoreProbes {
				if !onDisk[n] && !ignored[n] {
					missingProbes = append(missingProbes, n)
				}
			}
		}
		if len(missingProbes) > 0 {
			findings = append(findings, models.Finding{
				Category:     "security",
				Severity:     "WARNING",
				RuleID:       "ENV-IGNORE-INCOMPLETE",
				Title:        fmt.Sprintf("Pola .gitignore belum menutup semua varian env di %s", where),
				Description:  fmt.Sprintf("Nama env berikut belum ada tetapi akan lolos ke git bila dibuat: %s. Tiap server tujuan butuh berkas env sendiri (.env.server.<nama>), jadi pola harus wildcard, bukan daftar nama server.", strings.Join(missingProbes, ", ")),
				Suggestion:   "Ganti pola env per nama dengan blok wildcard berikut di .gitignore.",
				TargetFile:   rel(".gitignore"),
				TheoryBasis:  "The Twelve-Factor App (Factor III: Config).",
				SuggestedFix: envGitignoreBlock,
			})
		}

		if template != "" && ignored[template] && !isTracked[template] {
			findings = append(findings, models.Finding{
				Category:     "config",
				Severity:     "WARNING",
				RuleID:       "ENV-EXAMPLE-IGNORED",
				Title:        fmt.Sprintf("%s ikut ter-ignore sehingga tidak pernah terunggah", rel(template)),
				Description:  fmt.Sprintf("Pola .gitignore (biasanya .env* atau .env.*) juga menutup '%s', jadi server & developer baru tidak mendapat daftar variabel yang dibutuhkan.", template),
				Suggestion:   fmt.Sprintf("Tambahkan pengecualian !%s setelah pola env di .gitignore, lalu commit template-nya.", template),
				TargetFile:   rel(".gitignore"),
				SuggestedFix: envGitignoreBlock,
			})
		}
	}

	if len(envFiles) == 0 {
		return findings
	}
	if template == "" {
		return append(findings, models.Finding{
			Category:     "config",
			Severity:     "WARNING",
			RuleID:       "ENV-EXAMPLE-MISSING",
			Title:        fmt.Sprintf("Template .env.example belum ada di %s", where),
			Description:  fmt.Sprintf("Ada berkas env (%s) tetapi tidak ada .env.example yang di-commit. Saat aplikasi diunggah ke server baru, tidak ada daftar variabel yang harus diisi.", strings.Join(envFiles, ", ")),
			Suggestion:   "Buat .env.example berisi semua kunci dengan nilai dummy/kosong (tanpa secret asli) dan commit. Setiap server cukup menyalinnya menjadi .env.server.<nama>.",
			TargetFile:   rel(".env.example"),
			TheoryBasis:  "The Twelve-Factor App (Factor III: Config).",
			SuggestedFix: "# .env.example\nPORT=8080\nDATABASE_URL=\nSTORAGE_DIR=/var/lib/nama-app/storage",
		})
	}

	// Kunci yang dipakai di env asli tetapi tidak didokumentasikan di template: server baru yang disiapkan
	// dari template akan kekurangan variabel. Yang dilaporkan hanya nama kunci, tidak pernah nilainya.
	documented := envKeys(filepath.Join(dir, template))
	missing := map[string][]string{}
	for _, n := range envFiles {
		for k := range envKeys(filepath.Join(dir, n)) {
			if !documented[k] {
				missing[k] = append(missing[k], n)
			}
		}
	}
	if len(missing) == 0 {
		return findings
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
	return append(findings, models.Finding{
		Category:     "config",
		Severity:     "WARNING",
		RuleID:       "ENV-EXAMPLE-DRIFT",
		Title:        fmt.Sprintf("%s tidak memuat %d kunci yang dipakai berkas env", rel(template), len(keys)),
		Description:  fmt.Sprintf("Kunci berikut ada di berkas env tetapi tidak ada di %s: %s. Server baru yang disiapkan dari template akan kekurangan konfigurasi.", template, strings.Join(parts, "; ")),
		Suggestion:   fmt.Sprintf("Tambahkan kunci tersebut ke %s dengan nilai dummy/kosong (jangan salin nilai asli).", template),
		TargetFile:   rel(template),
		TheoryBasis:  "The Twelve-Factor App (Factor III: Config).",
		SuggestedFix: strings.Join(keys, "=\n") + "=",
	})
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
