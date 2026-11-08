package knowledgesync

import (
	"path/filepath"
	"regexp"
	"strings"

	"gorm.io/gorm"

	"github.com/vibeasik/vibeasik/pkg/models"
)

// Pemeriksaan privasi sebelum sebuah node boleh dikirim ke server. Sikapnya sengaja konservatif: lebih baik
// melewatkan node yang berguna daripada mengirim data yang tidak seharusnya keluar dari komputer pengguna.
// Node yang ditolak tidak disunting; ia tidak dikirim sama sekali.

var privacyPatterns = []struct {
	reason string
	re     *regexp.Regexp
}{
	{"memuat path berkas", regexp.MustCompile(`(?i)(^|[\s"'(\[=:])(/(users|home|var|etc|opt|usr|root|srv|mnt|tmp|private|volumes)/[^\s"')\]]+|~/[^\s"')\]]+|[a-z]:\\[^\s"')\]]+)`)},
	{"memuat alamat IP", regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)},
	{"memuat host internal", regexp.MustCompile(`(?i)\b(localhost|[a-z0-9-]+\.(local|internal|lan|corp|intranet|home))\b`)},
	{"memuat URL dengan kredensial", regexp.MustCompile(`(?i)://[^/\s:@]+:[^/\s@]+@`)},
	{"memuat URL dengan token", regexp.MustCompile(`(?i)https?://\S*[?&](token|key|secret|password|passwd|sig|signature|auth|apikey|api_key)=`)},
	{"memuat string acak panjang", regexp.MustCompile(`\b[0-9a-fA-F]{24,}\b|\b[A-Za-z0-9+/_-]{40,}={0,2}`)},
	{"memuat UUID", regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)},
	{"memuat email", regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)},
	{"memuat nomor telepon", regexp.MustCompile(`(\+62|\b08)[0-9][0-9 -]{7,13}[0-9]`)},
	{"memuat nomor kartu", regexp.MustCompile(`\b(?:\d[ -]?){13,19}\b`)},
	{"bertanda rahasia/kepemilikan", regexp.MustCompile(`(?i)\b(confidential|proprietary|internal only|do not (share|distribute)|rahasia perusahaan|jangan disebar|hak milik)\b`)},
	{"memuat kunci privat", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)},
}

// maxCodeRun batas baris kode berturut-turut; pengetahuan harus berupa prinsip/pola, bukan salinan kode proyek.
const maxCodeRun = 15

// genericNames kata umum yang sering menjadi nama komponen; tidak dianggap identitas proyek.
var genericNames = map[string]bool{
	"backend": true, "frontend": true, "server": true, "client": true, "admin": true, "mobile": true, "web": true,
	"api": true, "app": true, "apps": true, "core": true, "main": true, "docs": true, "test": true, "tests": true,
	"utils": true, "common": true, "shared": true, "service": true, "services": true, "gateway": true, "worker": true,
	"public": true, "private": true, "default": true, "project": true, "module": true, "package": true, "monorepo": true,
	"nuxt": true, "next": true, "react": true, "vue": true, "flutter": true, "fiber": true, "laravel": true,
	"golang": true, "node": true, "bundle": true, "other": true, "website": true, "dashboard": true,
}

// PrivacyChecker memeriksa teks terhadap pola umum dan terhadap identitas proyek di database lokal pengguna.
type PrivacyChecker struct {
	names   []*regexp.Regexp
	custom  []*regexp.Regexp
	literal []string
}

// NewPrivacyChecker memuat nama proyek/komponen/repo dan kata sensor milik pengguna dari database lokal.
func NewPrivacyChecker(db *gorm.DB) *PrivacyChecker {
	p := &PrivacyChecker{}
	seen := map[string]bool{}
	addName := func(s string) {
		s = strings.ToLower(strings.TrimSpace(s))
		if len(s) < 4 || genericNames[s] || seen[s] {
			return
		}
		seen[s] = true
		p.names = append(p.names, regexp.MustCompile(`(?i)(^|[^\p{L}\p{N}])`+regexp.QuoteMeta(s)+`($|[^\p{L}\p{N}])`))
	}
	if db != nil {
		var projects []models.Project
		db.Select("name", "slug", "root_path", "git_remote").Find(&projects)
		for _, pr := range projects {
			addName(pr.Name)
			addName(pr.Slug)
			addName(filepath.Base(pr.RootPath))
			if r := pr.GitRemote; r != "" {
				addName(strings.TrimSuffix(filepath.Base(strings.TrimRight(r, "/")), ".git"))
			}
		}
		var comps []models.Component
		db.Select("name", "relative_path", "absolute_path").Find(&comps)
		for _, c := range comps {
			addName(c.Name)
			addName(filepath.Base(c.RelativePath))
			addName(filepath.Base(c.AbsolutePath))
		}
		var kws []models.CensoredKeyword
		db.Where("is_active = ?", true).Find(&kws)
		for _, k := range kws {
			if strings.TrimSpace(k.Keyword) == "" {
				continue
			}
			if k.IsRegex {
				if re, err := regexp.Compile(k.Keyword); err == nil {
					p.custom = append(p.custom, re)
				}
			} else {
				p.literal = append(p.literal, strings.ToLower(k.Keyword))
			}
		}
	}
	return p
}

// Check mengembalikan alasan penolakan, atau "" bila teks aman dikirim.
func (p *PrivacyChecker) Check(text string) string {
	if r := staticCheck(text); r != "" {
		return r
	}
	for _, re := range p.names {
		if re.MatchString(text) {
			return "memuat nama proyek atau komponen Anda"
		}
	}
	lower := strings.ToLower(text)
	for _, l := range p.literal {
		if strings.Contains(lower, l) {
			return "memuat kata sensor Anda"
		}
	}
	for _, re := range p.custom {
		if re.MatchString(text) {
			return "memuat pola sensor Anda"
		}
	}
	return ""
}

func staticCheck(text string) string {
	for _, pp := range privacyPatterns {
		if pp.re.MatchString(text) {
			return pp.reason
		}
	}
	if longCodeRun(text) {
		return "memuat potongan kode panjang"
	}
	return ""
}

// longCodeRun: lebih dari maxCodeRun baris berturut-turut yang tampak seperti kode, atau blok ``` yang panjang.
func longCodeRun(text string) bool {
	run, inFence, fenceLines := 0, false, 0
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			if inFence && fenceLines > maxCodeRun {
				return true
			}
			inFence, fenceLines = !inFence, 0
			run = 0
			continue
		}
		if inFence {
			fenceLines++
		}
		if trim == "" {
			run = 0
			continue
		}
		looksCode := strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") ||
			strings.HasSuffix(trim, "{") || strings.HasSuffix(trim, "}") || strings.HasSuffix(trim, ";") ||
			strings.HasSuffix(trim, ")") && strings.Contains(trim, "(") && !strings.Contains(trim, " ")
		if looksCode {
			run++
			if run > maxCodeRun {
				return true
			}
		} else {
			run = 0
		}
	}
	return inFence && fenceLines > maxCodeRun
}

// LocalOnly: node Galeri Desain & Pelajaran Desain berisi data milik pengguna (referensi situs, tangkapan layar,
// gambar sebelum/sesudah, narasi). Ia hanya cermin lokal; yang boleh keluar hanya pengetahuan hasil abstraksi
// yang dikirim pengguna lewat jalur sumber lain (promoted/qa_review/gap_research). Tidak pernah dikirim ke server.
func LocalOnly(source, key string) bool {
	switch source {
	case models.KnowledgeSrcDesign, models.KnowledgeSrcLesson, models.KnowledgeSrcTheory:
		return true
	}
	k := strings.ToLower(key)
	return strings.HasPrefix(k, "design:") || strings.HasPrefix(k, "theory:")
}
