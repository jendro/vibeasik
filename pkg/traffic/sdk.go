package traffic

import (
	"bufio"
	"embed"
	"os"
	"path/filepath"
	"strings"
)

// SDK collector untuk BE live. Berkasnya tetap (tanpa templating); konfigurasi sepenuhnya lewat env
// sehingga kunci tidak pernah tertulis di kode proyek.
//
//go:embed sdk/go/traffic.go sdk/go/errors.go sdk/go/fiber.go sdk/browser/vibeasik-errors.js sdk/node/vibeasik-traffic.cjs sdk/node/vibeasik-traffic.nitro.ts sdk/python/vibeasik_traffic.py sdk/php/VibeasikTraffic.php
var sdkFS embed.FS

type SDKFile struct {
	Path    string `json:"path"` // lokasi saran relatif terhadap folder komponen
	Content string `json:"content"`
}

type SDKKit struct {
	ID       string    `json:"id"`
	Label    string    `json:"label"`
	Language string    `json:"language"`
	Files    []SDKFile `json:"files"`
	Usage    string    `json:"usage"` // potongan kode yang ditempel di entrypoint aplikasi
	Notes    string    `json:"notes"`
}

type kitSpec struct {
	id, label, lang string
	files           [][2]string // {berkas embed, path tujuan}
	usage, notes    string
}

var kitSpecs = []kitSpec{
	{"go-fiber", "Go · Fiber", "go", [][2]string{{"sdk/go/traffic.go", "internal/vibeasiktraffic/traffic.go"}, {"sdk/go/errors.go", "internal/vibeasiktraffic/errors.go"}, {"sdk/go/fiber.go", "internal/vibeasiktraffic/fiber.go"}},
		"import \"{{module}}/internal/vibeasiktraffic\"\n\napp := fiber.New()\napp.Use(recover.New())\napp.Use(vibeasiktraffic.Fiber()) // sebelum route\n\n// saat graceful shutdown:\nvibeasiktraffic.Flush()",
		"Pola rute Fiber (/users/:id) ikut terkirim, jadi statistik per endpoint akurat. Panic dilaporkan sebagai error lalu diteruskan ke recover; heartbeat tiap 30 detik. Opsional: VIBEASIK_RELEASE."},
	{"go-http", "Go · net/http, chi, gin, echo", "go", [][2]string{{"sdk/go/traffic.go", "internal/vibeasiktraffic/traffic.go"}, {"sdk/go/errors.go", "internal/vibeasiktraffic/errors.go"}},
		"import \"{{module}}/internal/vibeasiktraffic\"\n\nhandler := vibeasiktraffic.Middleware(router) // router = http.Handler (mux/chi/gin engine)\nhttp.ListenAndServe(addr, handler)\n\n// saat graceful shutdown:\nvibeasiktraffic.Flush()",
		"Tanpa pola rute framework; vibeasik menormalkan path (/users/42 → /users/:id). Panic dilaporkan sebagai error lalu diteruskan seperti semula; heartbeat tiap 30 detik. Opsional: VIBEASIK_RELEASE."},
	{"node-express", "Node.js · Express / Nest (express)", "javascript", [][2]string{{"sdk/node/vibeasik-traffic.cjs", "vibeasik-traffic.cjs"}},
		"const { vibeasikTraffic, vibeasikErrors } = require('./vibeasik-traffic.cjs')\n// ESM: import { vibeasikTraffic, vibeasikErrors } from './vibeasik-traffic.cjs'\n\napp.use(vibeasikTraffic()) // paling awal, sebelum router\n// ... rute ...\napp.use(vibeasikErrors()) // middleware error: paling akhir, setelah semua rute",
		"Butuh Node 18+ (fetch bawaan). Pola rute Express (req.route.path) ikut terkirim. Error rute, uncaughtException & unhandledRejection dilaporkan (tanpa mengubah perilaku crash bawaan); heartbeat tiap 30 detik. Opsional: VIBEASIK_RELEASE."},
	{"nitro", "Nuxt server / Nitro", "typescript", [][2]string{{"sdk/node/vibeasik-traffic.nitro.ts", "server/plugins/vibeasik-traffic.ts"}},
		"// Tidak perlu kode tambahan: plugin di server/plugins/ dimuat otomatis oleh Nitro.",
		"Aset /_nuxt tidak dicatat (ubah lewat VIBEASIK_TRAFFIC_SKIP). Error server (hook error Nitro) dan heartbeat tiap 30 detik ikut terkirim. Opsional: VIBEASIK_RELEASE."},
	{"python", "Python · FastAPI / Starlette / Flask / Django", "python", [][2]string{{"sdk/python/vibeasik_traffic.py", "vibeasik_traffic.py"}},
		"from vibeasik_traffic import VibeasikTrafficASGI, VibeasikTrafficWSGI\n\n# FastAPI / Starlette\napp.add_middleware(VibeasikTrafficASGI)\n\n# Flask\napp.wsgi_app = VibeasikTrafficWSGI(app.wsgi_app)\n\n# Flask/Django menangkap exception sendiri: laporkan lewat capture_exception(exc)",
		"Tanpa dependensi tambahan (urllib + thread latar). Pola rute FastAPI ikut terkirim. Exception dilaporkan lalu di-raise ulang; heartbeat tiap 30 detik. Flask/Django menangkap exception sendiri: panggil capture_exception(exc) dari sinyal/handler error-nya. Opsional: VIBEASIK_RELEASE."},
	{"laravel", "PHP · Laravel", "php", [][2]string{{"sdk/php/VibeasikTraffic.php", "app/Http/Middleware/VibeasikTraffic.php"}},
		"// Laravel 11+: bootstrap/app.php\n->withMiddleware(function ($middleware) {\n    $middleware->append(\\App\\Http\\Middleware\\VibeasikTraffic::class);\n})\n\n// Laravel ≤10: app/Http/Kernel.php → $middleware\n\\App\\Http\\Middleware\\VibeasikTraffic::class,\n\n// Laporkan exception (Laravel 11+: bootstrap/app.php)\n->withExceptions(fn ($e) => $e->report(fn (\\Throwable $t) => \\App\\Http\\Middleware\\VibeasikTraffic::captureError($t)))",
		"PHP-FPM: event diantre ke berkas temp lalu dikirim per batch setelah respons terkirim (terminate). Error dilaporkan lewat report() Laravel; heartbeat dikirim oportunistik (maks sekali per 30 detik saat ada request). Opsional: VIBEASIK_RELEASE."},
	{"browser", "Browser · JavaScript (Nuxt/Vue/React/lainnya)", "javascript", [][2]string{{"sdk/browser/vibeasik-errors.js", "public/vibeasik-errors.js"}},
		"<!-- kunci ingest sumber ini kelihatan di browser: kunci hanya bisa mengirim error browser -->\n<script src=\"/vibeasik-errors.js\" data-url=\"https://URL-INGEST\" data-key=\"KUNCI-INGEST\" data-release=\"v1.0.0\" defer></script>",
		"Menangkap window error & unhandledrejection; dedup dan maksimum per halaman; hanya path (tanpa query/hash) yang dikirim. Tidak dipilih otomatis oleh SuggestKit."},
}

// Kits mengembalikan seluruh SDK; module = path modul Go (untuk contoh import), boleh kosong.
func Kits(module string) []SDKKit {
	if module == "" {
		module = "github.com/kamu/app"
	}
	out := make([]SDKKit, 0, len(kitSpecs))
	for _, k := range kitSpecs {
		kit := SDKKit{ID: k.id, Label: k.label, Language: k.lang, Usage: strings.ReplaceAll(k.usage, "{{module}}", module), Notes: k.notes}
		for _, f := range k.files {
			b, _ := sdkFS.ReadFile(f[0])
			kit.Files = append(kit.Files, SDKFile{Path: f[1], Content: string(b)})
		}
		out = append(out, kit)
	}
	return out
}

func KitByID(id, module string) *SDKKit {
	for _, k := range Kits(module) {
		if k.ID == id {
			return &k
		}
	}
	return nil
}

// kitRules diperiksa berurutan; kecocokan pertama menang (l = bahasa, f = framework, keduanya huruf kecil).
var kitRules = []struct {
	kit   string
	match func(l, f string) bool
}{
	{"go-fiber", func(l, f string) bool { return l == "go" && strings.Contains(f, "fiber") }},
	{"go-http", func(l, f string) bool { return l == "go" }},
	{"nitro", func(l, f string) bool { return strings.Contains(f, "nuxt") || strings.Contains(f, "nitro") }},
	{"python", func(l, f string) bool {
		return l == "python" || strings.Contains(f, "fastapi") || strings.Contains(f, "flask") || strings.Contains(f, "django")
	}},
	{"laravel", func(l, f string) bool { return l == "php" || strings.Contains(f, "laravel") }},
	{"node-express", func(l, f string) bool {
		return l == "typescript" || l == "javascript" || strings.Contains(f, "express") || strings.Contains(f, "nest")
	}},
}

// SuggestKit memilih SDK dari bahasa/framework komponen hasil scan.
func SuggestKit(language, framework string) string {
	l, f := strings.ToLower(language), strings.ToLower(framework)
	for _, r := range kitRules {
		if r.match(l, f) {
			return r.kit
		}
	}
	return "go-fiber"
}

// GoModule membaca path modul dari go.mod komponen (untuk contoh import yang bisa langsung ditempel).
func GoModule(dir string) string {
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// InstallKit menulis berkas SDK ke folder komponen. Berkas yang sudah ada tidak ditimpa kecuali overwrite.
func InstallKit(kit *SDKKit, dir string, overwrite bool) (written, skipped []string, err error) {
	for _, f := range kit.Files {
		target := filepath.Join(dir, filepath.FromSlash(f.Path))
		if _, statErr := os.Stat(target); statErr == nil && !overwrite {
			skipped = append(skipped, f.Path)
			continue
		}
		if err = os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return
		}
		if err = os.WriteFile(target, []byte(f.Content), 0o644); err != nil {
			return
		}
		written = append(written, f.Path)
	}
	return
}
