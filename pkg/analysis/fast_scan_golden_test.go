package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/vibeasik/vibeasik/pkg/models"
)

var updateGolden = flag.Bool("update-golden", false, "tulis ulang testdata/fast_scan_golden.json")

type goldenScenario struct {
	name  string
	files map[string]string
	comp  models.Component
}

func goldenScenarios() []goldenScenario {
	var sc []goldenScenario
	add := func(name string, files map[string]string) {
		sc = append(sc, goldenScenario{name: name, files: files})
	}
	add("empty", map[string]string{})

	// --- Go: setiap token go.mod sendiri-sendiri, lalu semuanya.
	goTokens := []string{
		"github.com/gofiber/fiber/v2", "github.com/gofiber/fiber/v3", "github.com/gin-gonic/gin",
		"github.com/labstack/echo/v4", "github.com/labstack/echo", "github.com/go-chi/chi/v5", "github.com/go-chi/chi",
		"github.com/gorilla/mux", "github.com/zeromicro/go-zero", "github.com/kataras/iris", "github.com/beego/beego",
		"github.com/gofiber/contrib/jwt", "golang-jwt/jwt", "dgrijalva/jwt-go", "firebase.google.com/go/v4",
		"gorm.io/gorm", "gorm.io/driver/mysql", "go-sql-driver/mysql", "gorm.io/driver/postgres", "jackc/pgx",
		"lib/pq", "gorm.io/driver/sqlite", "mattn/go-sqlite3", "github.com/jmoiron/sqlx", "go.mongodb.org/mongo-driver",
		"github.com/redis/go-redis", "go-redis/redis", "golang.org/x/crypto", "github.com/google/uuid", "gofrs/uuid",
		"github.com/joho/godotenv", "github.com/spf13/viper", "github.com/go-playground/validator",
		"github.com/klauspost/compress", "fasthttp", "github.com/gorilla/websocket", "go.uber.org/zap",
		"github.com/sirupsen/logrus", "github.com/rs/zerolog", "github.com/swaggo/swag",
		"github.com/prometheus/client_golang", "sqlite",
	}
	for _, t := range goTokens {
		add("gomod/"+t, map[string]string{"go.mod": "module x\nrequire " + t + " v1.0.0\n"})
	}
	add("gomod/all", map[string]string{"go.mod": "module x\n" + strings.Join(goTokens, "\n")})
	add("gomod/zap+logrus", map[string]string{"go.mod": "go.uber.org/zap github.com/sirupsen/logrus"})
	add("gomod/logrus+zerolog", map[string]string{"go.mod": "github.com/sirupsen/logrus github.com/rs/zerolog"})
	add("gomod/bad-perm", map[string]string{"go.mod/": ""})
	for _, m := range []string{"cors.New", "middleware.CORS", "logger.New", "recover.New", "nothing"} {
		add("main/"+m, map[string]string{"go.mod": "module x", "main.go": "package main\nfunc main(){ " + m + "() }"})
	}
	add("main/all", map[string]string{"go.mod": "module x", "main.go": "cors.New logger.New recover.New"})

	dirsets := [][]string{
		{"domain", "usecase", "repository"}, {"domain", "usecases", "repositories"}, {"domain", "usecase"},
		{"controllers", "models"}, {"controllers"}, {"handlers", "services"}, {"cmd", "internal"}, {"cmd"},
		{"domain", "usecase", "repository", "controllers", "models"}, {"controllers", "models", "handlers", "services"},
		{"handlers", "services", "cmd", "internal"},
	}
	for _, ds := range dirsets {
		f := map[string]string{"go.mod": "module x"}
		for _, d := range ds {
			f[d+"/.keep"] = ""
		}
		add("godirs/"+strings.Join(ds, "+"), f)
		f2 := map[string]string{"go.mod": "module x"}
		for k, v := range f {
			f2[k] = v
		}
		f2["MEKANISME_ENKRIPSI.md"] = "x"
		add("godirs-e2ee/"+strings.Join(ds, "+"), f2)
	}
	add("goe2ee/KEAMANAN", map[string]string{"go.mod": "m", "controllers/a.go": "", "models/a.go": "", "KEAMANAN_DAN_PEMULIHAN_AKUN.md": ""})
	for _, c := range []string{"encrypted_", "family_key", "AES", "plain"} {
		add("goe2ee/key-"+c, map[string]string{"go.mod": "m", "models/a.go": "", "controllers/key_ctl.go": "x " + c})
		add("goe2ee/asset-"+c, map[string]string{"go.mod": "m", "models/a.go": "", "controllers/asset.go": "x " + c})
		add("goe2ee/other-"+c, map[string]string{"go.mod": "m", "models/a.go": "", "controllers/user.go": "x " + c})
	}

	// --- DB dari source go
	dbFiles := []string{"database/database.go", "database/db.go", "db/database.go", "db/db.go", "database.go", "db.go", "internal/database/db.go", "internal/database/database.go"}
	for _, f := range dbFiles {
		for _, c := range []string{"driver/mysql gorm", "mysql", "postgres gorm", "lib/pq", "pgx gorm", "sqlite gorm", "sqlite", "none gorm",
			"dbName := os.Getenv(\"DB_NAME\")\n\tif dbName == \"\" { dbName = \"shop_db\"",
			"mysql\ndbName := os.Getenv(\"DB_NAME\")\n if dbName == \"\" { dbName = \"zz\""} {
			add("dbsrc/"+f+"/"+c, map[string]string{f: "package db\n" + c})
		}
	}
	add("dbsrc/two-files", map[string]string{"database/database.go": "mysql", "db.go": "postgres"})
	add("dbsrc+gomod", map[string]string{"database/db.go": "postgres", "go.mod": "gorm.io/gorm gorm.io/driver/mysql"})
	add("dbsrc+gomod-sqlite", map[string]string{"go.mod": "gorm.io/gorm sqlite lib/pq go-sql-driver/mysql"})

	// --- Env
	envNames := []string{".env", ".env.example", ".env.server", ".env.local", ".env.development"}
	envLines := []string{
		"DATABASE.DEFAULT.DATABASE=ci_db", "DB_DATABASE=laravel", "DB_NAME=n", "DATABASE_NAME=dn",
		"DATABASE.DEFAULT.DBDRIVER=MySQLi", "DB_CONNECTION=mysql", "DB_DRIVER=postgres", "DB_CONNECTION=postgre", "DB_CONNECTION=sqlite",
		"DB_CONNECTION=\"Postgres\"", "DATABASE.DEFAULT.HOSTNAME=localhost", "DB_HOST=127.0.0.1", "DATABASE_HOST=h",
		"DATABASE.DEFAULT.PORT=3306", "DB_PORT=5432", "app.baseURL='http://x'", "APP_URL=http://y", "CI_ENVIRONMENT=development",
		"APP_ENV=prod", "PORT=8080", "SERVER_PORT=9", "JWT_SECRET=abc", "REDIS_URL=redis://x", "OPENAI_API_KEY=sk-123",
		"AI_PROVIDER=x", "FIREBASE_KEY=zzz", "SOME_TOKEN=tok", "MY_PASSWORD=pw", "PLAIN=1", "# comment", "", "NOEQUALS", "A=b=c",
		"APP.BASEURL=z", "JWT_SECRET=1\nJWT_EXP=2", "REDIS_HOST=a\nREDIS_PORT=1",
	}
	for i, l := range envLines {
		add("env/line/"+string(rune('a'+i%26))+string(rune('a'+i/26))+"/"+l, map[string]string{".env": l + "\n"})
	}
	for _, n := range envNames {
		add("env/file/"+n, map[string]string{n: "DB_DATABASE=x\nDB_CONNECTION=mysql\n"})
	}
	add("env/priority", map[string]string{".env": "DB_NAME=a\n", ".env.example": "DB_NAME=b\n"})
	add("env/all", map[string]string{".env": strings.Join(envLines, "\n")})
	add("env/all+go", map[string]string{".env": strings.Join(envLines, "\n"), "go.mod": "gorm.io/gorm gorm.io/driver/postgres"})

	// --- Manifest prisma / auth
	for _, p := range []string{`provider = "postgresql"`, `provider = "mysql"`, `provider = "sqlite"`, `provider = "mongodb"`} {
		add("prisma/"+p, map[string]string{"prisma/schema.prisma": p})
	}
	add("prisma/with-gomod", map[string]string{"prisma/schema.prisma": `provider = "mysql"`, "go.mod": "x"})
	add("auth/nuxt-ts", map[string]string{"server/api/auth.ts": "x"})
	add("auth/nuxt-dir", map[string]string{"server/api/auth/[...].ts": "x"})
	add("auth/nuxt+env-jwt", map[string]string{"server/api/auth.ts": "x", ".env": "JWT_SECRET=1"})

	// --- Node / frontend
	nodeDeps := []string{"nuxt", "next", "@sveltejs/kit", "svelte", "vue", "react", "@angular/core", "@nuxt/ui", "tailwindcss",
		"@nuxtjs/tailwindcss", "unocss", "primevue", "vuetify", "radix-vue", "shadcn-vue", "@radix-ui/react-dialog", "element-plus",
		"ant-design-vue", "antd", "daisyui", "@headlessui/vue", "@headlessui/react", "@iconify-json/lucide", "lucide-vue-next",
		"lucide-react", "@iconify-json/simple-icons", "@heroicons/vue", "@heroicons/react", "pinia", "@pinia/nuxt", "vuex",
		"@reduxjs/toolkit", "redux", "zustand", "ofetch", "axios", "@tanstack/vue-query", "@tanstack/react-query", "@trpc/client",
		"socket.io-client", "zod", "vee-validate", "@formkit/vue", "chart.js", "vue-chartjs", "echarts", "vue-echarts",
		"apexcharts", "vue3-apexcharts", "@vueuse/core", "date-fns", "dayjs", "typescript", "vue-tsc", "eslint", "@nuxt/eslint",
		"vitest", "@playwright/test"}
	pkgJSON := func(dep, section, ver string) string {
		return `{"name":"x","` + section + `":{"` + dep + `":"` + ver + `"}}`
	}
	for _, d := range nodeDeps {
		add("node/dep/"+d, map[string]string{"package.json": pkgJSON(d, "dependencies", "^3.1.0")})
		add("node/devdep/"+d, map[string]string{"package.json": pkgJSON(d, "devDependencies", "^3.1.0")})
	}
	add("node/nuxt4-dep", map[string]string{"package.json": pkgJSON("nuxt", "dependencies", "^4.0.1")})
	add("node/nuxt4-dev", map[string]string{"package.json": pkgJSON("nuxt", "devDependencies", "4.1.0")})
	add("node/nuxt4-dev-only-ver3", map[string]string{"package.json": pkgJSON("nuxt", "devDependencies", "3.9.0")})
	add("node/all", map[string]string{"package.json": `{"packageManager":"bun@1.1","dependencies":{"` + strings.Join(nodeDeps, `":"1","`) + `":"1"}}`})
	add("node/bad-json", map[string]string{"package.json": "{not json"})
	add("node/empty-obj", map[string]string{"package.json": "{}"})
	add("node/pm", map[string]string{"package.json": `{"packageManager":"pnpm@9","dependencies":{"vue":"3"}}`})
	for _, cfg := range []string{"nuxt.config.ts", "nuxt.config.js", "next.config.js", "next.config.mjs", "next.config.ts"} {
		add("node/cfg/"+cfg, map[string]string{"package.json": "{}", cfg: ""})
		add("node/cfgonly/"+cfg, map[string]string{cfg: ""})
	}
	for _, lock := range []string{"pnpm-lock.yaml", "yarn.lock", "package-lock.json", "bun.lockb"} {
		add("node/lock/"+lock, map[string]string{"package.json": "{}", lock: ""})
		add("node/locknuxt/"+lock, map[string]string{"package.json": "{}", lock: "", "nuxt.config.ts": ""})
	}
	nodeDirs := []string{"pages", "app/pages", "composables", "app/composables", "components", "app/components", "server/api", "server/routes", "middleware", "app/middleware"}
	for i := 0; i < 1<<4; i++ {
		var combo []string
		for j := 0; j < 4; j++ {
			if i&(1<<j) != 0 {
				combo = append(combo, []string{"pages", "composables", "components", "server/api"}[j])
			}
		}
		for _, fw := range []string{"nuxt", "next", "vue"} {
			f := map[string]string{"package.json": pkgJSON(fw, "dependencies", "3")}
			for _, d := range combo {
				f[d+"/.keep"] = ""
			}
			add("nodedirs/"+fw+"/"+strings.Join(combo, "+"), f)
		}
	}
	for _, d := range nodeDirs {
		add("nodedir1/"+d, map[string]string{"package.json": pkgJSON("nuxt", "dependencies", "3"), d + "/.keep": ""})
		add("nodedir1b/"+d, map[string]string{"package.json": pkgJSON("vue", "dependencies", "3"), d + "/.keep": ""})
	}
	add("nodedirs/nuxt-pages-comp-mw", map[string]string{"package.json": pkgJSON("nuxt", "dependencies", "3"), "app/pages/a": "", "app/composables/a": "", "app/middleware/a": "", "server/routes/a": ""})

	// --- Flutter
	for _, t := range []string{"flutter_bloc", "bloc:", "flutter_riverpod", "riverpod:", "provider:", "get:", "other", "dio:", "go_router:", "hive:", "isar:", "shared_preferences:",
		"flutter_bloc dio: go_router: hive: shared_preferences:", "provider: dio:"} {
		add("flutter/"+t, map[string]string{"pubspec.yaml": "name: x\n" + t})
	}
	add("flutter/main", map[string]string{"pubspec.yaml": "x", "lib/main.dart": "void main(){}"})

	// --- PHP
	add("php/spark", map[string]string{"spark": "x"})
	add("php/ci4routes", map[string]string{"app/Config/Routes.php": "x"})
	add("php/composer-ci4", map[string]string{"composer.json": `{"require":{"codeigniter4/framework":"*"}}`})
	add("php/composer-codeigniter", map[string]string{"composer.json": `{"require":{"codeigniter/x":"*"}}`})
	add("php/ci3", map[string]string{"application/config/routes.php": "x"})
	add("php/ci3core", map[string]string{"system/core/CodeIgniter.php": "x"})
	add("php/artisan", map[string]string{"artisan": "x"})
	add("php/composer-laravel", map[string]string{"composer.json": `{"require":{"laravel/framework":"*"}}`})
	add("php/composer-other", map[string]string{"composer.json": `{"require":{"x/y":"*"}}`})
	add("php/ci4+laravel", map[string]string{"spark": "", "artisan": ""})
	add("php/ci3+laravel", map[string]string{"application/config/routes.php": "", "artisan": ""})
	add("php/laravel-routes", map[string]string{"artisan": "x", "routes/api.php": "Route::get", "routes/web.php": "x"})

	// --- Key files
	allKeys := []string{".env.example", ".env", ".env.server", ".env.local", ".env.development", "docker-compose.yml", "docker-compose.yaml",
		"Dockerfile", "package.json", "go.mod", "pubspec.yaml", "Makefile", "database/database.go", "database/db.go", "db/database.go",
		"db/db.go", "database.go", "db.go", "internal/database/db.go", "internal/database/database.go", "internal/config/config.go",
		"config/config.go", "utils/config.go", "prisma/schema.prisma", "drizzle.config.ts", "nuxt.config.ts", "nuxt.config.js",
		"next.config.js", "next.config.mjs", "next.config.ts", "vite.config.ts", "vite.config.js", "tailwind.config.ts", "tailwind.config.js",
		"tsconfig.json", "spark", "app/Config/Routes.php", "app/Config/Database.php", "app/Config/App.php", "app/Config/Filters.php",
		"application/config/routes.php", "application/config/database.php", "composer.json", "routes/api.php", "routes/web.php", "main.go",
		"app.vue", "app/app.vue", "app/composables/useApi.ts", "composables/useApi.ts", "lib/main.dart", "README.md"}
	long := strings.Repeat("line\n", 500)
	for _, k := range allKeys {
		add("keys/"+k, map[string]string{k: long})
	}
	ctrl := []string{"controllers", "routes", "handlers", "api", "composables", "app/composables", "server/api", "app/Controllers", "app/Controllers/Api", "application/controllers"}
	for _, c := range ctrl {
		f := map[string]string{}
		for _, n := range []string{"a.go", "b_test.go", "c.ts", "d.test.ts", "e.js", "f.php", "g.txt", "h.go", "i.go", "j.go", "k.go", "l.go", "m.go", "n.go", "o.go", "p.go", "q.go"} {
			f[c+"/"+n] = strings.Repeat("x\n", 200)
		}
		f[c+"/sub/z.go"] = "z"
		add("ctrl/"+c, f)
	}
	add("ctrl/all", map[string]string{"controllers/a.go": "a", "routes/b.go": "b", "api/c.ts": "c", "app/Controllers/d.php": "d"})
	add("keys/empty-file", map[string]string{"go.mod": ""})

	// --- Component fallback variants
	for _, c := range []models.Component{
		{Language: "go", PackageManager: "go", DevCommand: "dev", BuildCommand: "bld", Port: 1234, Name: "n", Role: "BE"},
		{Port: 0}, {DevCommand: "x"}, {BuildCommand: "y", Framework: "fw"},
	} {
		for _, base := range []string{"go.mod", "pubspec.yaml", "package.json", "spark", "artisan", "composer.json-ci", "composer.json-lv", "nuxt", "none"} {
			f := map[string]string{}
			switch base {
			case "composer.json-ci":
				f["composer.json"] = "codeigniter"
			case "composer.json-lv":
				f["composer.json"] = "laravel"
			case "nuxt":
				f["package.json"] = "{}"
				f["nuxt.config.ts"] = ""
			case "none":
			default:
				f[base] = ""
			}
			sc = append(sc, goldenScenario{name: "comp/" + base + "/" + c.DevCommand + c.BuildCommand + c.Framework + string(rune('0'+c.Port%10)), files: f, comp: c})
		}
	}
	return sc
}

func runGoldenScenario(t *testing.T, s goldenScenario) string {
	dir := t.TempDir()
	for p, c := range s.files {
		if strings.HasSuffix(p, "/") {
			_ = os.MkdirAll(filepath.Join(dir, p), 0o755)
			continue
		}
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	comp := s.comp
	var res models.FastScanResult
	rawAI := ""
	if strings.HasPrefix(s.name, "comp/") {
		rawAI = "free text"
	}
	fallbackHeuristic(&res, dir, &comp, rawAI)
	out := map[string]interface{}{
		"arch":     detectArchitecture(dir),
		"env":      extractEnvHeuristic(dir),
		"keys0":    extractKeyFiles(dir, false),
		"keys1":    extractKeyFiles(dir, true),
		"fallback": res,
		"comp":     comp,
	}
	b, _ := json.Marshal(out)
	str := strings.ReplaceAll(string(b), dir, "<DIR>")
	sum := sha256.Sum256([]byte(str))
	return hex.EncodeToString(sum[:8])
}

func TestFastScanHeuristicsGolden(t *testing.T) {
	scs := goldenScenarios()
	got := make(map[string]string, len(scs))
	for _, s := range scs {
		if _, dup := got[s.name]; dup {
			t.Fatalf("nama skenario ganda: %s", s.name)
		}
		got[s.name] = runGoldenScenario(t, s)
	}
	path := filepath.Join("testdata", "fast_scan_golden.json")
	if *updateGolden {
		_ = os.MkdirAll("testdata", 0o755)
		b, _ := json.MarshalIndent(got, "", " ")
		if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]string
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	var bad []string
	for k, v := range got {
		if want[k] != v {
			bad = append(bad, k)
		}
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			bad = append(bad, k+" (hilang)")
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		t.Fatalf("%d skenario berubah, contoh: %v", len(bad), bad[:min(len(bad), 10)])
	}
}
