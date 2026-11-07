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
//go:embed sdk/go/traffic.go sdk/go/fiber.go sdk/node/vibeasik-traffic.cjs sdk/node/vibeasik-traffic.nitro.ts sdk/python/vibeasik_traffic.py sdk/php/VibeasikTraffic.php
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
	{"go-fiber", "Go · Fiber", "go", [][2]string{{"sdk/go/traffic.go", "internal/vibeasiktraffic/traffic.go"}, {"sdk/go/fiber.go", "internal/vibeasiktraffic/fiber.go"}},
		"import \"{{module}}/internal/vibeasiktraffic\"\n\napp := fiber.New()\napp.Use(recover.New())\napp.Use(vibeasiktraffic.Fiber()) // sebelum route\n\n// saat graceful shutdown:\nvibeasiktraffic.Flush()",
		"Pola rute Fiber (/users/:id) ikut terkirim, jadi statistik per endpoint akurat."},
	{"go-http", "Go · net/http, chi, gin, echo", "go", [][2]string{{"sdk/go/traffic.go", "internal/vibeasiktraffic/traffic.go"}},
		"import \"{{module}}/internal/vibeasiktraffic\"\n\nhandler := vibeasiktraffic.Middleware(router) // router = http.Handler (mux/chi/gin engine)\nhttp.ListenAndServe(addr, handler)\n\n// saat graceful shutdown:\nvibeasiktraffic.Flush()",
		"Tanpa pola rute framework; vibeasik menormalkan path (/users/42 → /users/:id)."},
	{"node-express", "Node.js · Express / Nest (express)", "javascript", [][2]string{{"sdk/node/vibeasik-traffic.cjs", "vibeasik-traffic.cjs"}},
		"const { vibeasikTraffic } = require('./vibeasik-traffic.cjs')\n// ESM: import { vibeasikTraffic } from './vibeasik-traffic.cjs'\n\napp.use(vibeasikTraffic()) // paling awal, sebelum router",
		"Butuh Node 18+ (fetch bawaan). Pola rute Express (req.route.path) ikut terkirim."},
	{"nitro", "Nuxt server / Nitro", "typescript", [][2]string{{"sdk/node/vibeasik-traffic.nitro.ts", "server/plugins/vibeasik-traffic.ts"}},
		"// Tidak perlu kode tambahan: plugin di server/plugins/ dimuat otomatis oleh Nitro.",
		"Aset /_nuxt tidak dicatat (ubah lewat VIBEASIK_TRAFFIC_SKIP)."},
	{"python", "Python · FastAPI / Starlette / Flask / Django", "python", [][2]string{{"sdk/python/vibeasik_traffic.py", "vibeasik_traffic.py"}},
		"from vibeasik_traffic import VibeasikTrafficASGI, VibeasikTrafficWSGI\n\n# FastAPI / Starlette\napp.add_middleware(VibeasikTrafficASGI)\n\n# Flask\napp.wsgi_app = VibeasikTrafficWSGI(app.wsgi_app)",
		"Tanpa dependensi tambahan (urllib + thread latar). Pola rute FastAPI ikut terkirim."},
	{"laravel", "PHP · Laravel", "php", [][2]string{{"sdk/php/VibeasikTraffic.php", "app/Http/Middleware/VibeasikTraffic.php"}},
		"// Laravel 11+: bootstrap/app.php\n->withMiddleware(function ($middleware) {\n    $middleware->append(\\App\\Http\\Middleware\\VibeasikTraffic::class);\n})\n\n// Laravel ≤10: app/Http/Kernel.php → $middleware\n\\App\\Http\\Middleware\\VibeasikTraffic::class,",
		"PHP-FPM: event diantre ke berkas temp lalu dikirim per batch setelah respons terkirim (terminate)."},
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

// SuggestKit memilih SDK dari bahasa/framework komponen hasil scan.
func SuggestKit(language, framework string) string {
	l, f := strings.ToLower(language), strings.ToLower(framework)
	switch {
	case l == "go" && strings.Contains(f, "fiber"):
		return "go-fiber"
	case l == "go":
		return "go-http"
	case strings.Contains(f, "nuxt") || strings.Contains(f, "nitro"):
		return "nitro"
	case l == "python" || strings.Contains(f, "fastapi") || strings.Contains(f, "flask") || strings.Contains(f, "django"):
		return "python"
	case l == "php" || strings.Contains(f, "laravel"):
		return "laravel"
	case l == "typescript" || l == "javascript" || strings.Contains(f, "express") || strings.Contains(f, "nest"):
		return "node-express"
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
